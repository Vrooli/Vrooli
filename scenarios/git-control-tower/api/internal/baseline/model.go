// Package baseline implements git-control-tower's cross-surface baseline
// substrate: a manifest of pointers (Decision 1 — baselines own pointers, not
// artifacts) into the surfaces that make up a scenario's review state
// (workflows, tests, structure, visuals, rules).
//
// A baseline is captured before an agent starts implementing a change and is
// the regression-diagnosis primitive that replaces `git stash`
// (feedback_no_git_stash): capture, implement, diff.
package baseline

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"git-control-tower/internal/git"
)

// SchemaVersion is the on-disk manifest schema version. Bump on
// backwards-incompatible field changes.
const SchemaVersion = 1

// Surface IDs. These are the stable identifiers an agent passes to --include
// and the keys of BaselineManifest.Surfaces.
const (
	SurfaceWorkflows = "workflows"
	SurfaceTests     = "tests"
	SurfaceStructure = "structure"
	SurfaceVisuals   = "visuals"
	SurfaceRules     = "rules"
)

// AllSurfaces is the canonical capture order (cheap → expensive). The CLI's
// default (empty --include) captures every available surface in this order.
var AllSurfaces = []string{SurfaceStructure, SurfaceRules, SurfaceTests, SurfaceVisuals, SurfaceWorkflows}

// SurfacePointer kinds — where the referenced artifact actually lives.
const (
	// KindTestGenieRun references a pinned test-genie run by runID.
	KindTestGenieRun = "test-genie-run"
	// KindGCTLocalSnapshot references a GCT-local JSON snapshot by ID.
	KindGCTLocalSnapshot = "gct-local-snapshot"
	// KindExternal references an artifact owned by another subsystem.
	KindExternal = "external"
)

// Verdict classifies a surface diff. Severity order (high→low) is
// regression > not-comparable > new-failure > preexisting > clean, matching
// test-genie's RunsService classifier so exit codes are consistent.
type Verdict string

const (
	VerdictClean         Verdict = "clean"
	VerdictRegression    Verdict = "regression"
	VerdictNewFailure    Verdict = "new-failure"
	VerdictPreexisting   Verdict = "preexisting"
	VerdictNotComparable Verdict = "not-comparable"
)

var verdictRank = map[Verdict]int{
	VerdictClean:         0,
	VerdictPreexisting:   1,
	VerdictNewFailure:    2,
	VerdictNotComparable: 3,
	VerdictRegression:    4,
}

// WorseVerdict returns the higher-severity of two verdicts. Used to roll
// per-surface verdicts up into the overall baseline-diff verdict.
func WorseVerdict(a, b Verdict) Verdict {
	if verdictRank[b] > verdictRank[a] {
		return b
	}
	return a
}

// SurfacePointer references a surface's captured artifact by stable ID. It
// carries a small summary so list/show can render without dereferencing the
// owning subsystem.
type SurfacePointer struct {
	SurfaceID  string          `json:"surface_id"`
	Kind       string          `json:"kind"`
	Ref        string          `json:"ref"` // runID, local snapshot ID, or external ref
	CapturedAt time.Time       `json:"captured_at"`
	Summary    json.RawMessage `json:"summary,omitempty"` // surface-specific short summary
}

// BaselineManifest is the JSON document stored under
// data/<repoID>/baselines/<scenario>/<branch>/<name>.json. It owns no
// artifacts — only pointers into the owning subsystems.
type BaselineManifest struct {
	Name      string                    `json:"name"`
	Scenario  string                    `json:"scenario"`
	Branch    string                    `json:"branch"`
	CreatedAt time.Time                 `json:"created_at"`
	CreatedBy string                    `json:"created_by,omitempty"` // "agent" | "ui:matt"
	Git       git.State                 `json:"git"`
	Surfaces  map[string]SurfacePointer `json:"surfaces"`
	// Skipped records surfaces that were requested at capture time but not
	// captured — keyed surfaceID → reason (adapter unavailable or capture
	// failed). Persisting it lets show/diff reveal a partial baseline instead
	// of letting it masquerade as complete: a baseline that never captured
	// `tests` must not read as "tests clean".
	Skipped       map[string]string `json:"skipped,omitempty"`
	SchemaVersion int               `json:"schema_version"`
}

// Validate checks the required fields are present. It does not validate
// surface availability — that is the adapter layer's job.
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
	return nil
}
