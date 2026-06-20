package k8s

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func ptr32(v int32) *int32 { return &v }

func podSpec(sa string) corev1.PodSpec { return corev1.PodSpec{ServiceAccountName: sa} }

func TestBuildWorkloadDefaults(t *testing.T) {
	w := buildWorkload("ns", "name", "Deployment", 3, corev1.PodSpec{}, nil)
	if w.ServiceAccount != "default" {
		t.Errorf("expected SA fallback 'default', got %q", w.ServiceAccount)
	}
	if w.Criticality != "medium" {
		t.Errorf("expected criticality fallback 'medium', got %q", w.Criticality)
	}

	labeled := buildWorkload("ns", "n", "Deployment", 1, podSpec("app-sa"), map[string]string{labelCriticality: "high"})
	if labeled.ServiceAccount != "app-sa" || labeled.Criticality != "high" {
		t.Errorf("expected explicit SA/criticality, got %q/%q", labeled.ServiceAccount, labeled.Criticality)
	}
}

func TestListWorkloadsCoversAllKindsAndDedupesOwned(t *testing.T) {
	owned := metav1.OwnerReference{Kind: "Deployment", Controller: ptr32Bool(true)}

	cs := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Namespace: "a", Name: "web"},
			Spec:       appsv1.DeploymentSpec{Replicas: ptr32(2), Template: corev1.PodTemplateSpec{Spec: podSpec("web-sa")}},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Namespace: "a", Name: "db"},
			Spec:       appsv1.StatefulSetSpec{Replicas: ptr32(1), Template: corev1.PodTemplateSpec{Spec: podSpec("db-sa")}},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Namespace: "a", Name: "agent"},
			Spec:       appsv1.DaemonSetSpec{Template: corev1.PodTemplateSpec{Spec: podSpec("agent-sa")}},
		},
		// Standalone ReplicaSet — must be included.
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{Namespace: "a", Name: "standalone-rs"},
			Spec:       appsv1.ReplicaSetSpec{Replicas: ptr32(1), Template: corev1.PodTemplateSpec{Spec: podSpec("rs-sa")}},
		},
		// Deployment-owned ReplicaSet — must be skipped.
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{Namespace: "a", Name: "web-rs", OwnerReferences: []metav1.OwnerReference{owned}},
			Spec:       appsv1.ReplicaSetSpec{Replicas: ptr32(2), Template: corev1.PodTemplateSpec{Spec: podSpec("web-sa")}},
		},
		// Standalone Job — included.
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Namespace: "a", Name: "migrate"},
			Spec:       batchv1.JobSpec{Template: corev1.PodTemplateSpec{Spec: podSpec("job-sa")}},
		},
		// CronJob-owned Job — skipped.
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Namespace: "a", Name: "cron-job-1", OwnerReferences: []metav1.OwnerReference{{Kind: "CronJob", Controller: ptr32Bool(true)}}},
			Spec:       batchv1.JobSpec{Template: corev1.PodTemplateSpec{Spec: podSpec("cj-sa")}},
		},
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{Namespace: "a", Name: "nightly"},
			Spec:       batchv1.CronJobSpec{JobTemplate: batchv1.JobTemplateSpec{Spec: batchv1.JobSpec{Template: corev1.PodTemplateSpec{Spec: podSpec("nightly-sa")}}}},
		},
	)

	workloads, err := listWorkloads(context.Background(), cs)
	if err != nil {
		t.Fatalf("listWorkloads: %v", err)
	}

	gotKinds := map[string]string{} // name -> kind
	for _, w := range workloads {
		gotKinds[w.Name] = w.Kind
	}

	want := map[string]string{
		"web":           "Deployment",
		"db":            "StatefulSet",
		"agent":         "DaemonSet",
		"standalone-rs": "ReplicaSet",
		"migrate":       "Job",
		"nightly":       "CronJob",
	}
	for name, kind := range want {
		if gotKinds[name] != kind {
			t.Errorf("expected %s as %s, got %q", name, kind, gotKinds[name])
		}
	}
	if _, ok := gotKinds["web-rs"]; ok {
		t.Error("Deployment-owned ReplicaSet should be skipped")
	}
	if _, ok := gotKinds["cron-job-1"]; ok {
		t.Error("CronJob-owned Job should be skipped")
	}
	if len(workloads) != len(want) {
		t.Errorf("expected %d workloads, got %d", len(want), len(workloads))
	}
}

func ptr32Bool(b bool) *bool { return &b }
