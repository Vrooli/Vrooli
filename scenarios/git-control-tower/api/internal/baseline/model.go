// Package baseline implements git-control-tower's durable regression baseline.
// A baseline owns no test artifacts: it pins exactly one comprehensive Test
// Genie run and records the immutable identities needed to compare that run
// with a later comprehensive run.
package baseline

import (
	"fmt"
	"strings"
	"time"

	"git-control-tower/internal/git"
)

// SchemaVersion is the canonical on-disk manifest schema version.
const SchemaVersion = 2

// CaptureProfile is the Test Genie run shape used for every GCT baseline.
const CaptureProfile = "baseline"

// Verdict is Test Genie's comparison classification. Changed is retained for
// advisory visual evidence; it never raises the process exit code.
type Verdict string

const (
	VerdictClean         Verdict = "clean"
	VerdictChanged       Verdict = "changed"
	VerdictRegression    Verdict = "regression"
	VerdictNewFailure    Verdict = "new-failure"
	VerdictPreexisting   Verdict = "preexisting"
	VerdictNotComparable Verdict = "not-comparable"
)

// RunAnchor is the single immutable Test Genie run referenced by a baseline.
// DescriptorSnapshotRef is deliberately opaque to GCT: it identifies the
// snapshot inside the owning run and is never a filesystem path.
type RunAnchor struct {
	RunID                           string    `json:"run_id"`
	CapturedAt                      time.Time `json:"captured_at"`
	CaptureProfile                  string    `json:"capture_profile"`
	TreeDigest                      string    `json:"tree_digest,omitempty"`
	PhaseSetDigest                  string    `json:"phase_set_digest,omitempty"`
	DescriptorSnapshotRef           string    `json:"descriptor_snapshot_ref,omitempty"`
	DescriptorSnapshotDigest        string    `json:"descriptor_snapshot_digest,omitempty"`
	DescriptorSnapshotSchemaVersion int       `json:"descriptor_snapshot_schema_version,omitempty"`
	EvidenceTier                    string    `json:"evidence_tier,omitempty"`
	SourceScope                     string    `json:"source_scope,omitempty"`
	SourceStable                    bool      `json:"source_stable"`
}

// MigrationInfo preserves honest diagnostics for a V1 manifest whose run can
// be anchored but whose newer descriptor/digest identities are unknowable.
type MigrationInfo struct {
	FromSchemaVersion int       `json:"from_schema_version"`
	MigratedAt        time.Time `json:"migrated_at"`
	// PinReconciledAt is written only after Test Genie has accepted the
	// baseline's idempotent retention owner. Keeping this checkpoint beside the
	// migrated manifest makes a crash between rewrite and pin safe to resume.
	PinReconciledAt time.Time `json:"pin_reconciled_at,omitempty"`
	DegradedReasons []string  `json:"degraded_reasons,omitempty"`
}

// BaselineManifest is stored under
// data/<repoID>/baselines/<scenario>/<branch>/<name>.json.
type BaselineManifest struct {
	Name          string         `json:"name"`
	Scenario      string         `json:"scenario"`
	Branch        string         `json:"branch"`
	CreatedAt     time.Time      `json:"created_at"`
	CreatedBy     string         `json:"created_by,omitempty"`
	Git           git.State      `json:"git"`
	Run           RunAnchor      `json:"run"`
	Migration     *MigrationInfo `json:"migration,omitempty"`
	SchemaVersion int            `json:"schema_version"`
}

// RunID returns the baseline's sole Test Genie run identity.
func (m BaselineManifest) RunID() string { return m.Run.RunID }

// Validate enforces the single-run invariant before persistence. The scoped
// fields are additive: historical V2 manifests remain readable.
func (m BaselineManifest) Validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("baseline name is required")
	}
	if strings.TrimSpace(m.Scenario) == "" {
		return fmt.Errorf("scenario is required")
	}
	if strings.TrimSpace(m.Branch) == "" {
		return fmt.Errorf("branch is required")
	}
	if m.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported baseline schema version %d", m.SchemaVersion)
	}
	if strings.TrimSpace(m.Run.RunID) == "" {
		return fmt.Errorf("baseline run id is required")
	}
	if m.Run.CaptureProfile != CaptureProfile {
		return fmt.Errorf("baseline capture profile must be %q", CaptureProfile)
	}
	if m.Migration == nil {
		if m.Run.TreeDigest == "" || m.Run.PhaseSetDigest == "" {
			return fmt.Errorf("new baseline requires tree and phase-set digests")
		}
		if m.Run.DescriptorSnapshotRef == "" || m.Run.DescriptorSnapshotDigest == "" || m.Run.DescriptorSnapshotSchemaVersion == 0 {
			return fmt.Errorf("new baseline requires descriptor snapshot identity")
		}
	}
	return nil
}
