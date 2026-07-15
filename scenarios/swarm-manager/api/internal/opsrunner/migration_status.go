package opsrunner

import (
	"encoding/json"
	"fmt"
	"os"
)

// Migration states for the Phase-8 persisted-state migration
// (docs/operations/migration/RUNBOOK.md). The status document is the small,
// durable projection the operator surface reads to render partial-migration
// state; the migration tooling is its writer.
const (
	MigrationNotStarted  = "not-started"
	MigrationStaged      = "staged"
	MigrationPromoted    = "promoted"
	MigrationQuarantined = "quarantined"
)

// MigrationStatus is the typed shape of the migration-status document
// ("<data root>/agentops/migration-status.json"). Phase 7 owns only the read
// contract; the Phase-8 migration tooling writes it at each runbook step
// (staged after §3, promoted after cutover, quarantined when a stop-condition
// fires).
type MigrationStatus struct {
	Kind          string `json:"kind"` // "agentops-migration-status"
	SchemaVersion string `json:"schema_version"`
	// State is one of the Migration* constants.
	State string `json:"state"`
	// Epoch is the migration epoch marker number (runbook §4).
	Epoch int `json:"epoch,omitempty"`
	// Counts of migrated objects per disposition.
	StagedCount      int `json:"staged_count,omitempty"`
	PromotedCount    int `json:"promoted_count,omitempty"`
	QuarantinedCount int `json:"quarantined_count,omitempty"`
	// StartedAt / UpdatedAt are RFC3339 timestamps stamped by the writer.
	StartedAt string `json:"started_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	// ReportPath points at the epoch report document, when written.
	ReportPath string `json:"report_path,omitempty"`
}

// validMigrationState reports whether s is a known migration state.
func validMigrationState(s string) bool {
	switch s {
	case MigrationNotStarted, MigrationStaged, MigrationPromoted, MigrationQuarantined:
		return true
	default:
		return false
	}
}

// LoadMigrationStatus reads the migration-status document at path. An absent
// file is the not-started state (found=false), never an error, so the operator
// surface can always render migration state. A present-but-invalid document is
// a fail-closed error: a half-written status must never masquerade as
// not-started.
func LoadMigrationStatus(path string) (MigrationStatus, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return MigrationStatus{State: MigrationNotStarted}, false, nil
		}
		return MigrationStatus{}, false, err
	}
	var st MigrationStatus
	if err := json.Unmarshal(raw, &st); err != nil {
		return MigrationStatus{}, false, fmt.Errorf("decode migration status %s: %w", path, err)
	}
	if !validMigrationState(st.State) {
		return MigrationStatus{}, false, fmt.Errorf("migration status %s has unknown state %q", path, st.State)
	}
	return st, true, nil
}
