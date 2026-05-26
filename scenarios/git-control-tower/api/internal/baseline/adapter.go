package baseline

import "context"

// Target identifies what a surface adapter operates on for a single request.
// repoID scopes GCT-local storage; repoDir is the absolute working tree the
// surface scans; Scenario is the slug passed to owning subsystems.
type Target struct {
	RepoID   int64
	RepoDir  string
	Scenario string
}

// CaptureOptions tune a capture (Plan A §3.3 fast vs full).
type CaptureOptions struct {
	// Fast is the agent default: skip heavy diagnostics (video/HAR/trace).
	Fast bool
	// PinnedBy is the pin owner recorded on any test-genie run a surface pins
	// (e.g. "gct:baseline:plan-7c3"). Deleting the baseline unpins it.
	PinnedBy string
}

// SurfaceDiff is the comparison of a captured surface pointer against the
// surface's current state.
type SurfaceDiff struct {
	SurfaceID   string
	Verdict     Verdict
	Regressions []string // passing in baseline, failing now
	NewFailures []string // absent in baseline, failing now
	Preexisting []string // failing in both
	Cleared     []string // failing in baseline, passing now
	Summary     string   // one-line human summary
}

// Staleness reports how far the working tree has drifted from a pointer's
// captured commit.
type Staleness struct {
	CommitsSince int
	FilesChanged int
	LikelyStale  bool // heuristic: many commits or many files changed since capture
}

// SurfaceAdapter is the contract every review surface implements. Adapters own
// the surface-specific capture + diff logic; the baseline service orchestrates
// them (Decision 3 — ownership boundaries).
//
// Staleness is intentionally NOT on this interface: all surfaces in one
// baseline are captured at a single git commit, so drift is uniform and
// computed once by the service via ComputeStaleness, not per-surface.
type SurfaceAdapter interface {
	ID() string
	// Available reports whether the surface can be captured for the target
	// right now (owning subsystem healthy, applicable to the scenario).
	Available(ctx context.Context, t Target) bool
	// Capture snapshots the surface and returns a pointer to the artifact.
	Capture(ctx context.Context, t Target, opts CaptureOptions) (SurfacePointer, error)
	// Diff compares a captured pointer against the surface's current state.
	Diff(ctx context.Context, t Target, ptr SurfacePointer) (SurfaceDiff, error)
}

// ComputeStaleness reports how far repoDir's HEAD has drifted from the
// baseline commit. LikelyStale is a heuristic: more than 10 commits or more
// than 20 files changed since capture.
func ComputeStaleness(ctx context.Context, probe StalenessProbe, repoDir, baselineSha string) (Staleness, error) {
	if probe == nil || baselineSha == "" {
		return Staleness{}, nil
	}
	commits, files, err := probe.Since(ctx, repoDir, baselineSha)
	if err != nil {
		return Staleness{}, err
	}
	return Staleness{
		CommitsSince: commits,
		FilesChanged: files,
		LikelyStale:  commits > 10 || files > 20,
	}, nil
}

// ---------------------------------------------------------------------------
// Seam interfaces — injected so adapters are unit-testable with fakes and the
// baseline package never imports the flat main package (no import cycle).
// ---------------------------------------------------------------------------

// PhaseStatus is a single phase's terminal status.
type PhaseStatus struct {
	Name   string
	Status string // passed | failed | skipped
}

// ExecResult is the outcome of triggering a test-genie run.
type ExecResult struct {
	RunID   string
	Success bool
	Phases  []PhaseStatus
}

// PhaseDiff mirrors test-genie's RunsService per-phase classification.
type PhaseDiff struct {
	Phase       string
	Verdict     string
	Regressions []string
	NewFailures []string
	Preexisting []string
	Cleared     []string
}

// CompareResult mirrors test-genie's CompareRunsResponse.
type CompareResult struct {
	Verdict string
	Phases  []PhaseDiff
}

// Executor triggers a test-genie suite run scoped to a phase set + diagnostics
// preset and returns the runID-keyed result. Wraps test-genie's execute API.
type Executor interface {
	Execute(ctx context.Context, scenario string, phases []string, diagnosticsPreset string) (ExecResult, error)
}

// RunsClient wraps test-genie's RunsService (pin/unpin/compare) for the
// workflows + tests adapters.
type RunsClient interface {
	PinRun(ctx context.Context, scenario, runID, pinnedBy, reason string) error
	UnpinRun(ctx context.Context, scenario, runID, pinnedBy string) error
	CompareRuns(ctx context.Context, scenario, runIDA, runIDB, phase string) (CompareResult, error)
}

// VisualSnapshot is the metadata of a GCT-local visual snapshot set.
type VisualSnapshot struct {
	SnapshotID      string
	ScreenshotCount int
	Pages           []string
}

// VisualClient wraps GCT's local visual snapshot capture/listing.
type VisualClient interface {
	Capture(ctx context.Context, repoID int64, repoDir, scenario string) (VisualSnapshot, error)
	Get(ctx context.Context, repoID int64, scenario, snapshotID string) (VisualSnapshot, bool, error)
}

// StalenessProbe computes commits and files changed since a sha. Read-only.
type StalenessProbe interface {
	Since(ctx context.Context, repoDir, sha string) (commits int, files int, err error)
}
