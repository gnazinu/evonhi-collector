package sender

import (
	"time"

	"github.com/tu-usuario/evonhi-collector/internal/model"
)

// Envelope conforms to evo_saas TelemetrySnapshotCreate
// (app/schemas.py:196 — source_kind, payload, summary, collected_at).
type Envelope struct {
	SourceKind  string                     `json:"source_kind"`
	Payload     *model.ClusterStatePayload `json:"payload"`
	Summary     map[string]any             `json:"summary"`
	CollectedAt time.Time                  `json:"collected_at"`
}

func BuildEnvelope(p *model.ClusterStatePayload) *Envelope {
	return &Envelope{
		SourceKind:  "connector",
		Payload:     p,
		Summary:     summarize(p),
		CollectedAt: p.CollectedAt,
	}
}

func summarize(p *model.ClusterStatePayload) map[string]any {
	criticals := 0
	for _, w := range p.Workloads {
		if w.Criticality == "critical" || w.Criticality == "high" {
			criticals++
		}
	}
	return map[string]any{
		"schema_version":        p.SchemaVersion,
		"cluster_id":            p.ClusterID,
		"workloads":             len(p.Workloads),
		"service_accounts":      len(p.ServiceAccounts),
		"roles":                 len(p.Roles),
		"cluster_roles":         len(p.ClusterRoles),
		"role_bindings":         len(p.RoleBindings),
		"cluster_role_bindings": len(p.ClusterRoleBindings),
		"secrets":               len(p.Secrets),
		"permissions":           len(p.Permissions),
		"critical_workloads":    criticals,
	}
}
