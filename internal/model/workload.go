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
}
