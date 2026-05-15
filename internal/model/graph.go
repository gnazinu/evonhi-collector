package model

type ServiceAccount struct {
	Namespace   string            `json:"namespace"`
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Secrets     []string          `json:"secrets,omitempty"`
}

type PolicyRule struct {
	APIGroups       []string `json:"api_groups"`
	Resources       []string `json:"resources"`
	ResourceNames   []string `json:"resource_names,omitempty"`
	Verbs           []string `json:"verbs"`
	NonResourceURLs []string `json:"non_resource_urls,omitempty"`
}

type Role struct {
	Namespace string       `json:"namespace"`
	Name      string       `json:"name"`
	Rules     []PolicyRule `json:"rules"`
}

type ClusterRole struct {
	Name  string       `json:"name"`
	Rules []PolicyRule `json:"rules"`
}

type RoleRef struct {
	APIGroup string `json:"api_group"`
	Kind     string `json:"kind"`
	Name     string `json:"name"`
}

type Subject struct {
	Kind      string `json:"kind"`
	APIGroup  string `json:"api_group,omitempty"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type RoleBinding struct {
	Namespace string    `json:"namespace"`
	Name      string    `json:"name"`
	RoleRef   RoleRef   `json:"role_ref"`
	Subjects  []Subject `json:"subjects"`
}

type ClusterRoleBinding struct {
	Name     string    `json:"name"`
	RoleRef  RoleRef   `json:"role_ref"`
	Subjects []Subject `json:"subjects"`
}
