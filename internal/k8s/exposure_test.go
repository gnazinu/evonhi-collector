package k8s

import (
	"bufio"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/tu-usuario/evonhi-collector/internal/model"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReadableSecretsExcludeImagePullSecrets(t *testing.T) {
	spec := corev1.PodSpec{
		ImagePullSecrets: []corev1.LocalObjectReference{{Name: "registry"}},
		Volumes: []corev1.Volume{
			{Name: "creds", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "db-creds"}}},
			{Name: "bundle", VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{Sources: []corev1.VolumeProjection{
				{Secret: &corev1.SecretProjection{LocalObjectReference: corev1.LocalObjectReference{Name: "tls"}}},
			}}}},
		},
		InitContainers: []corev1.Container{{EnvFrom: []corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "migrations"}}}}}},
		Containers: []corev1.Container{{Env: []corev1.EnvVar{{Name: "API_KEY", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "db-creds"}, Key: "key",
		}}}}}},
	}
	got := readableSecrets(spec)
	want := []string{"db-creds", "migrations", "tls"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("readableSecrets = %v, want %v", got, want)
	}

	idx := buildSecretMountIndex([]corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Namespace: "a", Name: "web-1"}, Spec: spec}})
	if via := idx["a/registry"][0].Via; via != "imagePullSecret" {
		t.Errorf("registry secret should be recorded as imagePullSecret, got %q", via)
	}
}

func TestMarkExposure(t *testing.T) {
	workloads := []model.Workload{
		{Namespace: "shop", Name: "web", PodLabels: map[string]string{"app": "web", "tier": "front"}},
		{Namespace: "shop", Name: "api", PodLabels: map[string]string{"app": "api"}},
		{Namespace: "shop", Name: "db", PodLabels: map[string]string{"app": "db"}},
		{Namespace: "other", Name: "web", PodLabels: map[string]string{"app": "web"}},
	}
	services := []corev1.Service{
		{ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: "web-lb"}, Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer, Selector: map[string]string{"app": "web"}}},
		{ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: "api"}, Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP, Selector: map[string]string{"app": "api"}}},
		{ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: "db"}, Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP, Selector: map[string]string{"app": "db"}}},
		{ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: "manual"}, Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeNodePort}},
	}
	ingresses := []networkingv1.Ingress{{
		ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: "public"},
		Spec: networkingv1.IngressSpec{Rules: []networkingv1.IngressRule{{IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{
			Paths: []networkingv1.HTTPIngressPath{{Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "api"}}}},
		}}}}},
	}}

	markExposure(workloads, services, ingresses)

	want := map[string][]string{
		"shop/web":  {"service/LoadBalancer/web-lb"},
		"shop/api":  {"ingress/public"},
		"shop/db":   nil,
		"other/web": nil,
	}
	for _, w := range workloads {
		key := w.Namespace + "/" + w.Name
		if w.Public != (want[key] != nil) || (want[key] != nil && !reflect.DeepEqual(w.ExposedBy, want[key])) {
			t.Errorf("%s: public=%v exposed_by=%v, want %v", key, w.Public, w.ExposedBy, want[key])
		}
	}
}

func TestAnnotationFiltering(t *testing.T) {
	in := map[string]string{
		lastAppliedAnnotation:                `{"data":{"password":"c2VjcmV0"}}`,
		"eks.amazonaws.com/role-arn":         "arn:aws:iam::1:role/app",
		"kubernetes.io/service-account.name": "app",
		"meta.helm.sh/release-name":          "app",
		"evonhi.io/public":                   "true",
	}
	sa := sanitizeAnnotations(in)
	if _, ok := sa[lastAppliedAnnotation]; ok {
		t.Error("last-applied-configuration must never be forwarded")
	}
	if sa["eks.amazonaws.com/role-arn"] == "" {
		t.Error("service account identity annotations are needed for the IAM bridge")
	}
	secret := secretAnnotations(in)
	want := map[string]string{"kubernetes.io/service-account.name": "app", "evonhi.io/public": "true"}
	if !reflect.DeepEqual(secret, want) {
		t.Errorf("secretAnnotations = %v, want %v", secret, want)
	}
}

// TestClusterRoleCoversEverythingTheAgentLists guards the regression where Jobs/CronJobs were
// collected without RBAC on the batch group, making every in-cluster run fail with 403.
func TestClusterRoleCoversEverythingTheAgentLists(t *testing.T) {
	required := map[string][]string{
		"":                          {"serviceaccounts", "pods", "services"},
		"apps":                      {"deployments", "statefulsets", "daemonsets", "replicasets"},
		"batch":                     {"jobs", "cronjobs"},
		"networking.k8s.io":         {"ingresses"},
		"rbac.authorization.k8s.io": {"roles", "clusterroles", "rolebindings", "clusterrolebindings"},
	}
	granted := parseClusterRole(t, "../../deploy/helm/evo-collector/templates/clusterrole.yaml")
	for group, resources := range required {
		for _, r := range resources {
			if !granted[group+"/"+r] {
				t.Errorf("ClusterRole does not grant list on %q in group %q", r, group)
			}
		}
	}
}

// parseClusterRole reads the (Helm-templated) ClusterRole and returns group/resource pairs that
// carry the list verb. Template directives are ignored, so opt-in rules count as granted.
func parseClusterRole(t *testing.T, path string) map[string]bool {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	granted := map[string]bool{}
	var groups, resources []string
	inResources := false
	flush := func(verbs string) {
		if strings.Contains(verbs, "list") {
			for _, g := range groups {
				for _, r := range resources {
					granted[g+"/"+r] = true
				}
			}
		}
		groups, resources, inResources = nil, nil, false
	}
	parseList := func(s string) []string {
		s = strings.TrimSpace(s)
		s = strings.TrimSuffix(strings.TrimPrefix(s, "["), "]")
		out := []string{}
		for _, item := range strings.Split(s, ",") {
			out = append(out, strings.Trim(strings.TrimSpace(item), `"`))
		}
		return out
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "- apiGroups:"):
			groups = parseList(strings.TrimPrefix(line, "- apiGroups:"))
		case strings.HasPrefix(line, "resources:"):
			inResources = true
		case strings.HasPrefix(line, "verbs:"):
			flush(line)
		case inResources && strings.HasPrefix(line, "- "):
			resources = append(resources, strings.TrimPrefix(line, "- "))
		}
	}
	return granted
}
