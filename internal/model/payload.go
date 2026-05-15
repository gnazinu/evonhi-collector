package model

import "time"

const SchemaVersion = "1.0"

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
}

type Permission struct {
	Namespace       string `json:"namespace"`
	ServiceAccount  string `json:"service_account"`
	APIGroup        string `json:"api_group"`
	Resource        string `json:"resource"`
	Verb            string `json:"verb"`
	RecentRequests  int    `json:"recent_requests"`
	Source          string `json:"source"`
}
