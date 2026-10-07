package k8s

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var update = flag.Bool("update", false, "rewrite testdata/snapshot.golden.json")

const goldenPath = "testdata/snapshot.golden.json"

// TestCollectProducesContractPayload runs the whole collection against a fake cluster and freezes
// the JSON the control plane receives. The same golden file is validated against
// evonhi-core's contracts/cluster-snapshot.v1.schema.json in CI, so a field rename on either side
// breaks a build instead of a customer's analysis.
func TestCollectProducesContractPayload(t *testing.T) {
	automountOff := false
	labels := map[string]string{"app": "web"}
	cs := fake.NewSimpleClientset(
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: "web-sa", Annotations: map[string]string{
			"eks.amazonaws.com/role-arn": "arn:aws:iam::111:role/web",
			lastAppliedAnnotation:        "{}",
		}}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: "worker-sa"}, AutomountServiceAccountToken: &automountOff},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: "web"},
			Spec: appsv1.DeploymentSpec{Replicas: ptr32(2), Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec:       corev1.PodSpec{ServiceAccountName: "web-sa", ImagePullSecrets: []corev1.LocalObjectReference{{Name: "registry"}}},
			}},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: "worker"},
			Spec: appsv1.DeploymentSpec{Replicas: ptr32(1), Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "worker"}},
				Spec: corev1.PodSpec{ServiceAccountName: "worker-sa", Volumes: []corev1.Volume{
					{Name: "db", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "db-creds"}}},
				}},
			}},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: "worker-7d9f-abcde", OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "worker-7d9f"}}},
			Spec: corev1.PodSpec{ServiceAccountName: "worker-sa", Volumes: []corev1.Volume{
				{Name: "db", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "db-creds"}}},
			}},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: "web"},
			Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer, Selector: labels},
		},
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: "deployer"},
			Rules:      []rbacv1.PolicyRule{{APIGroups: []string{"apps"}, Resources: []string{"deployments"}, Verbs: []string{"patch"}}},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: "secret-reader"},
			Rules:      []rbacv1.PolicyRule{{APIGroups: []string{""}, Resources: []string{"secrets"}, Verbs: []string{"get"}}},
		},
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: "web-deploys"},
			RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "Role", Name: "deployer"},
			Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: "web-sa", Namespace: "shop"}},
		},
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: "worker-reads"},
			RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "secret-reader"},
			Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: "worker-sa"}},
		},
	)

	payload, err := Collect(context.Background(), cs, "fake-cluster", false)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	payload.CollectedAt = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC) // deterministic golden

	got, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got = append(got, '\n')
	if *update {
		if err := os.WriteFile(goldenPath, got, 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (run go test ./internal/k8s -update): %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("payload drifted from %s; if intended, run go test ./internal/k8s -update and re-validate against the contract\n%s", goldenPath, got)
	}
}
