// Package runindex is the shared read contract for test-genie's append-only
// run index (coverage/runs.index.json): the record types stored in the index
// and a read-only loader. test-genie owns the write/locking side (its
// internal/shared/runs Index aliases these types); cached status readers such
// as scenario-completeness-scoring consume this package to enumerate runs
// without depending on the test-genie service.
//
// The write side replaces the index file atomically (write-tmp + rename), so
// the lock-free read here can never observe a torn file — at worst it reads
// the previous complete generation.
package runindex

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Run status values recorded in the index.
const (
	// StatusQueued is a run that has been admitted but is waiting for a global
	// concurrency slot before it starts executing. It is non-terminal: a waiter
	// blocks through it until the run is promoted to in_progress and then to a
	// terminal state. Queued records carry only the shape stamped at admission
	// (run id, scenario, preset); the executor fills the rest on promotion.
	StatusQueued     = "queued"
	StatusInProgress = "in_progress"
	StatusPassed     = "passed"
	StatusFailed     = "failed"
	StatusAborted    = "aborted"
)

// DiagnosticsConfig is the serialized diagnostics profile a run was executed
// with. It is immutable once the run completes.
type DiagnosticsConfig struct {
	Video   bool `json:"video"`
	Console bool `json:"console"`
	Network bool `json:"network"`
	HAR     bool `json:"har"`
	Trace   bool `json:"trace"`
	DOM     bool `json:"dom"`
}

// PinRecord protects a run from retention GC while an external consumer (e.g.
// a git-control-tower baseline) references it.
type PinRecord struct {
	PinnedBy string    `json:"pinned_by"`
	PinnedAt time.Time `json:"pinned_at"`
	Reason   string    `json:"reason,omitempty"`
}

// PhaseRecord is a compact per-phase summary stored in the index for fast
// enumeration without reading per-run phase-results files.
type PhaseRecord struct {
	Name            string `json:"name"`
	Status          string `json:"status"`
	DurationSeconds int    `json:"duration_seconds"`
	Comparable      bool   `json:"comparable,omitempty"`
	Advisory        bool   `json:"advisory,omitempty"`
	ArtifactBacked  bool   `json:"artifact_backed,omitempty"`
	NonComparable   bool   `json:"non_comparable,omitempty"`
}

// RunRecord is the index entry for a single test-genie execution.
type RunRecord struct {
	RunID           string        `json:"run_id"`
	Scenario        string        `json:"scenario"`
	StartedAt       time.Time     `json:"started_at"`
	CompletedAt     time.Time     `json:"completed_at,omitempty"`
	Status          string        `json:"status"`
	Phases          []PhaseRecord `json:"phases,omitempty"`
	GitSha          string        `json:"git_sha,omitempty"`
	GitBranch       string        `json:"git_branch,omitempty"`
	GitDirty        bool          `json:"git_dirty,omitempty"`
	GitDirtySummary string        `json:"git_dirty_summary,omitempty"`
	// TreeDigest is the scenario working-tree content digest (see the sibling
	// treedigest package) captured at run START, so it identifies the
	// byte-state the phases actually executed against. Empty on runs that
	// predate digest stamping ("unknown" freshness).
	TreeDigest string `json:"tree_digest,omitempty"`
	// Preset is the requested suite preset ("quick"|"comprehensive"|...) and
	// CaptureProfile the capture-depth dial (""|"baseline"). Both are stamped at
	// run start and identify WHAT suite shape executed, so a consumer can reuse a
	// completed run only when it matches the shape it needs (git-control-tower
	// reuses a clean-tree comprehensive+baseline run instead of re-running it).
	// Empty on runs that predate shape stamping.
	Preset         string            `json:"preset,omitempty"`
	CaptureProfile string            `json:"capture_profile,omitempty"`
	PlannedPhases  []string          `json:"planned_phases,omitempty"`
	PhaseSetDigest string            `json:"phase_set_digest,omitempty"`
	Diagnostics    DiagnosticsConfig `json:"diagnostics"`
	Pins           []PinRecord       `json:"pins,omitempty"`
}

// IsPinned reports whether the run is protected from retention GC.
func (r RunRecord) IsPinned() bool { return len(r.Pins) > 0 }

// PhaseStatus returns the run's phases as a name -> status map.
func (r RunRecord) PhaseStatus() map[string]string {
	m := make(map[string]string, len(r.Phases))
	for _, p := range r.Phases {
		m[p.Name] = p.Status
	}
	return m
}

// IndexPath returns the absolute path to a scenario's run index file.
func IndexPath(scenarioDir string) string {
	return filepath.Join(scenarioDir, "coverage", "runs.index.json")
}

// Load reads a scenario's run index and returns the records sorted
// newest-first. A missing or empty index yields (nil, nil) — no runs is a
// normal state, not an error.
func Load(scenarioDir string) ([]RunRecord, error) {
	data, err := os.ReadFile(IndexPath(scenarioDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read run index: %w", err)
	}
	if len(data) == 0 {
		return nil, nil
	}
	var records []RunRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("parse run index: %w", err)
	}
	SortNewestFirst(records)
	return records, nil
}

// SortNewestFirst orders records newest-first (by StartedAt, then RunID),
// matching the order the test-genie write side serves.
func SortNewestFirst(records []RunRecord) {
	sort.SliceStable(records, func(a, b int) bool {
		if records[a].StartedAt.Equal(records[b].StartedAt) {
			return records[a].RunID > records[b].RunID
		}
		return records[a].StartedAt.After(records[b].StartedAt)
	})
}
