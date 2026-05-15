package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tu-usuario/evonhi-collector/internal/model"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	labelCriticality = "evonhi.io/criticality"
	defaultPageSize  = 200
)

// Collect builds a ClusterStatePayload by listing RBAC + workloads + secrets metadata.
// It never reads Secret.Data — only metadata is captured.
func Collect(ctx context.Context, cs *kubernetes.Clientset, clusterID string) (*model.ClusterStatePayload, error) {
	payload := &model.ClusterStatePayload{
		SchemaVersion: model.SchemaVersion,
		ClusterID:     clusterID,
		CollectedAt:   time.Now().UTC(),
	}

	sas, err := listServiceAccounts(ctx, cs)
	if err != nil {
		return nil, fmt.Errorf("service accounts: %w", err)
	}
	payload.ServiceAccounts = sas

	roles, err := listRoles(ctx, cs)
	if err != nil {
		return nil, fmt.Errorf("roles: %w", err)
	}
	payload.Roles = roles

	clusterRoles, err := listClusterRoles(ctx, cs)
	if err != nil {
		return nil, fmt.Errorf("cluster roles: %w", err)
	}
	payload.ClusterRoles = clusterRoles

	rbs, err := listRoleBindings(ctx, cs)
	if err != nil {
		return nil, fmt.Errorf("role bindings: %w", err)
	}
	payload.RoleBindings = rbs

	crbs, err := listClusterRoleBindings(ctx, cs)
	if err != nil {
		return nil, fmt.Errorf("cluster role bindings: %w", err)
	}
	payload.ClusterRoleBindings = crbs

	pods, err := listPods(ctx, cs)
	if err != nil {
		return nil, fmt.Errorf("pods: %w", err)
	}

	deployments, err := listDeployments(ctx, cs)
	if err != nil {
		return nil, fmt.Errorf("deployments: %w", err)
	}
	payload.Workloads = deployments

	secrets, err := listSecrets(ctx, cs, pods)
	if err != nil {
		return nil, fmt.Errorf("secrets: %w", err)
	}
	payload.Secrets = secrets

	roleIndex := indexRoles(roles)
	clusterRoleIndex := indexClusterRoles(clusterRoles)
	payload.Permissions = derivePermissions(rbs, crbs, roleIndex, clusterRoleIndex)

	return payload, nil
}

func listServiceAccounts(ctx context.Context, cs *kubernetes.Clientset) ([]model.ServiceAccount, error) {
	out := []model.ServiceAccount{}
	opts := metav1.ListOptions{Limit: defaultPageSize}
	for {
		list, err := cs.CoreV1().ServiceAccounts("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, sa := range list.Items {
			refs := make([]string, 0, len(sa.Secrets))
			for _, s := range sa.Secrets {
				refs = append(refs, s.Name)
			}
			out = append(out, model.ServiceAccount{
				Namespace:   sa.Namespace,
				Name:        sa.Name,
				Labels:      sa.Labels,
				Annotations: sa.Annotations,
				Secrets:     refs,
			})
		}
		if list.Continue == "" {
			break
		}
		opts.Continue = list.Continue
	}
	return out, nil
}

func listRoles(ctx context.Context, cs *kubernetes.Clientset) ([]model.Role, error) {
	out := []model.Role{}
	opts := metav1.ListOptions{Limit: defaultPageSize}
	for {
		list, err := cs.RbacV1().Roles("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, r := range list.Items {
			out = append(out, model.Role{
				Namespace: r.Namespace,
				Name:      r.Name,
				Rules:     convertRules(r.Rules),
			})
		}
		if list.Continue == "" {
			break
		}
		opts.Continue = list.Continue
	}
	return out, nil
}

func listClusterRoles(ctx context.Context, cs *kubernetes.Clientset) ([]model.ClusterRole, error) {
	out := []model.ClusterRole{}
	opts := metav1.ListOptions{Limit: defaultPageSize}
	for {
		list, err := cs.RbacV1().ClusterRoles().List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, r := range list.Items {
			out = append(out, model.ClusterRole{
				Name:  r.Name,
				Rules: convertRules(r.Rules),
			})
		}
		if list.Continue == "" {
			break
		}
		opts.Continue = list.Continue
	}
	return out, nil
}

func listRoleBindings(ctx context.Context, cs *kubernetes.Clientset) ([]model.RoleBinding, error) {
	out := []model.RoleBinding{}
	opts := metav1.ListOptions{Limit: defaultPageSize}
	for {
		list, err := cs.RbacV1().RoleBindings("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, b := range list.Items {
			out = append(out, model.RoleBinding{
				Namespace: b.Namespace,
				Name:      b.Name,
				RoleRef:   convertRoleRef(b.RoleRef),
				Subjects:  convertSubjects(b.Subjects),
			})
		}
		if list.Continue == "" {
			break
		}
		opts.Continue = list.Continue
	}
	return out, nil
}

func listClusterRoleBindings(ctx context.Context, cs *kubernetes.Clientset) ([]model.ClusterRoleBinding, error) {
	out := []model.ClusterRoleBinding{}
	opts := metav1.ListOptions{Limit: defaultPageSize}
	for {
		list, err := cs.RbacV1().ClusterRoleBindings().List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, b := range list.Items {
			out = append(out, model.ClusterRoleBinding{
				Name:     b.Name,
				RoleRef:  convertRoleRef(b.RoleRef),
				Subjects: convertSubjects(b.Subjects),
			})
		}
		if list.Continue == "" {
			break
		}
		opts.Continue = list.Continue
	}
	return out, nil
}

func listPods(ctx context.Context, cs *kubernetes.Clientset) ([]corev1.Pod, error) {
	out := []corev1.Pod{}
	opts := metav1.ListOptions{Limit: defaultPageSize}
	for {
		list, err := cs.CoreV1().Pods("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		out = append(out, list.Items...)
		if list.Continue == "" {
			break
		}
		opts.Continue = list.Continue
	}
	return out, nil
}

func listDeployments(ctx context.Context, cs *kubernetes.Clientset) ([]model.Workload, error) {
	out := []model.Workload{}
	opts := metav1.ListOptions{Limit: defaultPageSize}
	for {
		list, err := cs.AppsV1().Deployments("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, d := range list.Items {
			replicas := int32(0)
			if d.Spec.Replicas != nil {
				replicas = *d.Spec.Replicas
			}
			sa := d.Spec.Template.Spec.ServiceAccountName
			if sa == "" {
				sa = "default"
			}
			crit := d.Labels[labelCriticality]
			if crit == "" {
				crit = "medium"
			}
			out = append(out, model.Workload{
				Namespace:      d.Namespace,
				Name:           d.Name,
				Kind:           "Deployment",
				Replicas:       replicas,
				ServiceAccount: sa,
				Criticality:    crit,
				TrafficRPS:     0,
				Labels:         d.Labels,
			})
		}
		if list.Continue == "" {
			break
		}
		opts.Continue = list.Continue
	}
	return out, nil
}

// listSecrets enumerates Secrets capturing ONLY metadata. Secret.Data is never read.
// mounted_by is derived from the pod list.
func listSecrets(ctx context.Context, cs *kubernetes.Clientset, pods []corev1.Pod) ([]model.SecretRef, error) {
	mountIdx := buildSecretMountIndex(pods)
	out := []model.SecretRef{}
	opts := metav1.ListOptions{Limit: defaultPageSize}
	for {
		list, err := cs.CoreV1().Secrets("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, s := range list.Items {
			// Defense in depth: never propagate Data even if API returns it.
			_ = s.Data
			key := s.Namespace + "/" + s.Name
			mounts := mountIdx[key]
			if mounts == nil {
				mounts = []model.SecretMount{}
			}
			out = append(out, model.SecretRef{
				Namespace:   s.Namespace,
				Name:        s.Name,
				Type:        string(s.Type),
				Labels:      s.Labels,
				Annotations: s.Annotations,
				MountedBy:   mounts,
			})
		}
		if list.Continue == "" {
			break
		}
		opts.Continue = list.Continue
	}
	return out, nil
}

func getWorkloadName(p corev1.Pod) string {
	if len(p.OwnerReferences) > 0 {
		owner := p.OwnerReferences[0]
		if owner.Kind == "ReplicaSet" {
			lastDash := strings.LastIndex(owner.Name, "-")
			if lastDash > 0 {
				return owner.Name[:lastDash]
			}
		}
		return owner.Name
	}
	return p.Name
}

func buildSecretMountIndex(pods []corev1.Pod) map[string][]model.SecretMount {
	idx := map[string][]model.SecretMount{}
	add := func(ns, secretName string, m model.SecretMount) {
		if secretName == "" {
			return
		}
		key := ns + "/" + secretName
		idx[key] = append(idx[key], m)
	}
	for _, p := range pods {
		sa := p.Spec.ServiceAccountName
		if sa == "" {
			sa = "default"
		}
		mount := model.SecretMount{Namespace: p.Namespace, Workload: getWorkloadName(p), ServiceAccount: sa}
		for _, v := range p.Spec.Volumes {
			if v.Secret != nil {
				add(p.Namespace, v.Secret.SecretName, mount)
			}
		}
		for _, ips := range p.Spec.ImagePullSecrets {
			add(p.Namespace, ips.Name, mount)
		}
		for _, c := range p.Spec.Containers {
			for _, ef := range c.EnvFrom {
				if ef.SecretRef != nil {
					add(p.Namespace, ef.SecretRef.Name, mount)
				}
			}
			for _, e := range c.Env {
				if e.ValueFrom != nil && e.ValueFrom.SecretKeyRef != nil {
					add(p.Namespace, e.ValueFrom.SecretKeyRef.Name, mount)
				}
			}
		}
	}
	return idx
}

func convertRules(in []rbacv1.PolicyRule) []model.PolicyRule {
	out := make([]model.PolicyRule, 0, len(in))
	for _, r := range in {
		out = append(out, model.PolicyRule{
			APIGroups:       r.APIGroups,
			Resources:       r.Resources,
			ResourceNames:   r.ResourceNames,
			Verbs:           r.Verbs,
			NonResourceURLs: r.NonResourceURLs,
		})
	}
	return out
}

func convertRoleRef(r rbacv1.RoleRef) model.RoleRef {
	return model.RoleRef{APIGroup: r.APIGroup, Kind: r.Kind, Name: r.Name}
}

func convertSubjects(in []rbacv1.Subject) []model.Subject {
	out := make([]model.Subject, 0, len(in))
	for _, s := range in {
		out = append(out, model.Subject{
			Kind:      s.Kind,
			APIGroup:  s.APIGroup,
			Name:      s.Name,
			Namespace: s.Namespace,
		})
	}
	return out
}

func indexRoles(roles []model.Role) map[string][]model.PolicyRule {
	idx := map[string][]model.PolicyRule{}
	for _, r := range roles {
		idx[r.Namespace+"/"+r.Name] = r.Rules
	}
	return idx
}

func indexClusterRoles(roles []model.ClusterRole) map[string][]model.PolicyRule {
	idx := map[string][]model.PolicyRule{}
	for _, r := range roles {
		idx[r.Name] = r.Rules
	}
	return idx
}

// derivePermissions expands every (ServiceAccount subject → bound role → policy rules)
// into a flat Permission list. Cartesian over apiGroups × resources × verbs.
// recent_requests is left at 0; the audit-log aggregator (PR4) populates it.
func derivePermissions(
	rbs []model.RoleBinding,
	crbs []model.ClusterRoleBinding,
	roleIdx map[string][]model.PolicyRule,
	clusterRoleIdx map[string][]model.PolicyRule,
) []model.Permission {
	out := []model.Permission{}

	expand := func(saNS, saName string, rules []model.PolicyRule, source string) {
		for _, rule := range rules {
			groups := rule.APIGroups
			if len(groups) == 0 {
				groups = []string{""}
			}
			resources := rule.Resources
			if len(resources) == 0 {
				resources = []string{"*"}
			}
			for _, g := range groups {
				for _, res := range resources {
					for _, v := range rule.Verbs {
						out = append(out, model.Permission{
							Namespace:      saNS,
							ServiceAccount: saName,
							APIGroup:       g,
							Resource:       res,
							Verb:           v,
							RecentRequests: 0,
							Source:         source,
						})
					}
				}
			}
		}
	}

	resolveRules := func(ref model.RoleRef, bindingNS string) []model.PolicyRule {
		switch ref.Kind {
		case "Role":
			return roleIdx[bindingNS+"/"+ref.Name]
		case "ClusterRole":
			return clusterRoleIdx[ref.Name]
		}
		return nil
	}

	for _, b := range rbs {
		rules := resolveRules(b.RoleRef, b.Namespace)
		if rules == nil {
			continue
		}
		for _, s := range b.Subjects {
			if s.Kind != "ServiceAccount" {
				continue
			}
			ns := s.Namespace
			if ns == "" {
				ns = b.Namespace
			}
			expand(ns, s.Name, rules, "RoleBinding/"+b.Namespace+"/"+b.Name)
		}
	}

	for _, b := range crbs {
		rules := resolveRules(b.RoleRef, "")
		if rules == nil {
			continue
		}
		for _, s := range b.Subjects {
			if s.Kind != "ServiceAccount" {
				continue
			}
			expand(s.Namespace, s.Name, rules, "ClusterRoleBinding/"+b.Name)
		}
	}

	return out
}
