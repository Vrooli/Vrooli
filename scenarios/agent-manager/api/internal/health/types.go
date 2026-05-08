// Package health owns the persisted audit log of model and runner health
// transitions. Replaces the pre-Phase-2 in-memory modelregistry.HealthStore.
//
// Design:
//   - Two append-only SQLite tables (model_health_audit, runner_health_audit)
//     are the source of truth.
//   - "Current health" is derived by MAX(timestamp) GROUP BY (runner, model).
//   - Eviction is a separate background concern (eviction.go) — the store
//     itself never silently drops rows.
//   - Health transitions emit typed eventlog events at the call site
//     (model.health.transition, runner.health.transition); this package
//     writes the audit row and exposes Snapshot/Audit query APIs.
//
// Contract:
//   - Status values are stable strings — they appear in persisted rows
//     and the JSON snapshot endpoint.
//   - Reasons (when populated on failed status) are fallback.Reason values
//     stored as their stable string form.
//
// DOC: scenarios/agent-manager/docs/internal/EVENT_TAXONOMY.md
// DOC: scenarios/agent-manager/docs/internal/SEAMS.md
package health

import "time"

// Status labels a single (runner, model) entry's most recent observation.
type Status string

const (
	// StatusUnknown means no observation has been recorded yet (or
	// observations have been evicted). Distinct from "ok" so callers can
	// distinguish "not yet probed" from "probed and healthy."
	StatusUnknown Status = "unknown"

	// StatusOK means the most recent observation succeeded.
	StatusOK Status = "ok"

	// StatusFailed means the most recent observation classified the
	// target as unavailable.
	StatusFailed Status = "failed"
)

// ModelEntry is a current-health snapshot row for a single (runner, model)
// pair. The JSON shape matches the pre-Phase-2 modelregistry.ModelHealth
// so the existing /api/v1/runner-models/health endpoint stays compatible.
type ModelEntry struct {
	Status      Status    `json:"status"`
	LastChecked time.Time `json:"lastChecked"`
	Message     string    `json:"message,omitempty"`
	Reason      string    `json:"reason,omitempty"`
}

// RunnerEntry is the analogous current-snapshot row for a runner.
type RunnerEntry struct {
	Status      Status    `json:"status"`
	LastChecked time.Time `json:"lastChecked"`
	Message     string    `json:"message,omitempty"`
	Reason      string    `json:"reason,omitempty"`
}

// Snapshot is the JSON-shaped current-health payload returned by the
// snapshot endpoint. Top-level shape preserved from pre-Phase-2 for
// API compatibility; "models" key holds the legacy nested model map and
// "runners" gains a parallel runner-level map.
type Snapshot struct {
	// Models is the legacy shape: runner_type → model_id → ModelEntry.
	// Runners that were registered but have no entries appear with an
	// empty inner map so consumers always see the same top-level shape.
	Models map[string]map[string]ModelEntry `json:"models"`

	// Runners is the runner-level current-snapshot: runner_type → RunnerEntry.
	Runners map[string]RunnerEntry `json:"runners"`
}

// AuditRow is one observation in the model_health_audit or
// runner_health_audit table. ModelID is empty for runner-level rows.
type AuditRow struct {
	ID          int64     `db:"id" json:"id"`
	Timestamp   time.Time `db:"timestamp" json:"timestamp"`
	RunnerType  string    `db:"runner_type" json:"runnerType"`
	ModelID     string    `db:"model_id" json:"modelId,omitempty"`
	Status      Status    `db:"status" json:"status"`
	Reason      string    `db:"reason" json:"reason,omitempty"`
	Message     string    `db:"message" json:"message,omitempty"`
	TriggeredBy string    `db:"triggered_by" json:"triggeredBy"`
}

// AuditQuery filters historical audit rows. Empty fields are wildcards.
// Limit defaults to 100 when zero.
type AuditQuery struct {
	RunnerType string
	ModelID    string
	Since      time.Time
	Until      time.Time
	Status     Status
	Limit      int
}
