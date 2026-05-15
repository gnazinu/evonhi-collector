package model

type SecretMount struct {
	Namespace      string `json:"namespace"`
	Workload       string `json:"workload"`
	ServiceAccount string `json:"service_account,omitempty"`
}

type SecretRef struct {
	Namespace   string            `json:"namespace"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	MountedBy   []SecretMount     `json:"mounted_by"`
}
