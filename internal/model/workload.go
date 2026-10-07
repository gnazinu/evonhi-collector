package model

type Workload struct {
	Namespace      string            `json:"namespace"`
	Name           string            `json:"name"`
	Kind           string            `json:"kind"`
	Replicas       int32             `json:"replicas"`
	ServiceAccount string            `json:"service_account"`
	Criticality    string            `json:"criticality"`
	TrafficRPS     float64           `json:"traffic_rps"`
	Labels         map[string]string `json:"labels,omitempty"`

	// Since schema 1.1.
	// AutomountToken is the pod-level automountServiceAccountToken (nil = inherit from the SA).
	AutomountToken *bool `json:"automount_token"`
	// MountedSecrets are readable from inside the containers (volumes, projected volumes,
	// env, envFrom). imagePullSecrets are excluded: the kubelet uses them, the container cannot.
	MountedSecrets []string `json:"mounted_secrets"`
	// Public is true when a LoadBalancer/NodePort Service, an external IP or an Ingress routes
	// traffic from outside the cluster to this workload's pods.
	Public    bool     `json:"public"`
	ExposedBy []string `json:"exposed_by"`

	// PodLabels are the pod template labels used to match Service selectors. Never serialized.
	PodLabels map[string]string `json:"-"`
}
