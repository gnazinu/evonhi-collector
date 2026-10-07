package model

import "time"

const SchemaVersion = "1.1"

type ClusterStatePayload struct {
	SchemaVersion       string               `json:"schema_version"`
	ClusterID           string               `json:"cluster_id"`
	CollectedAt         time.Time            `json:"collected_at"`
	Workloads           []Workload           `json:"workloads"`
	ServiceAccounts     []ServiceAccount     `json:"service_accounts"`
	Roles               []Role               `json:"roles"`
	ClusterRoles        []ClusterRole        `json:"cluster_roles"`
	RoleBindings        []RoleBinding        `json:"role_bindings"`
	ClusterRoleBindings []ClusterRoleBinding `json:"cluster_role_bindings"`
	Secrets             []SecretRef          `json:"secrets"`
	Permissions         []Permission         `json:"permissions"`
	// Usage is the provenance of runtime usage data. Nil until an audit-log source is wired:
	// the control plane then reports telemetry confidence 0 instead of guessing.
	Usage *Usage `json:"usage"`
}

// Usage describes where observed request counts come from (e.g. k8s-audit) and over how long.
type Usage struct {
	Source     string  `json:"source"`
	WindowDays float64 `json:"window_days"`
}

type Permission struct {
	Namespace      string `json:"namespace"`
	ServiceAccount string `json:"service_account"`
	APIGroup       string `json:"api_group"`
	Resource       string `json:"resource"`
	Verb           string `json:"verb"`
	RecentRequests int    `json:"recent_requests"`
	Source         string `json:"source"`
}
