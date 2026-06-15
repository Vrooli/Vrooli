package baseline

import "context"

// Target identifies what a baseline operates on for a single request. repoID
// scopes GCT-local storage; repoDir is the absolute working tree; Scenario is
// the slug passed to test-genie.
type Target struct {
	RepoID   int64
	RepoDir  string
	Scenario string
}

// SurfaceDiff is the comparison of a captured surface against the surface's
// current state, bucketed locally from one comprehensive CompareRuns.
type SurfaceDiff struct {
	SurfaceID   string
	Verdict     Verdict
	Regressions []string // passing in baseline, failing now
	NewFailures []string // absent in baseline, failing now
	Preexisting []string // failing in both
	Cleared     []string // failing in baseline, passing now
	Summary     string   // one-line human summary
}

// Staleness reports how far the working tree has drifted from the baseline's
// captured commit.
type Staleness struct {
	CommitsSince int
	FilesChanged int
	LikelyStale  bool // heuristic: many commits or many files changed since capture
}

// ComputeStaleness reports how far repoDir's HEAD has drifted from the baseline
// commit. LikelyStale is a heuristic: more than 10 commits or more than 20
// files changed since capture.
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
// Seam interfaces — injected so the service is unit-testable with fakes and the
// baseline package never imports the flat main package (no import cycle).
// ---------------------------------------------------------------------------

// PhaseStatus is a single phase's terminal status.
type PhaseStatus struct {
	Name   string
	Status string // passed | failed | skipped
}

// ExecResult is the outcome of triggering one comprehensive test-genie run.
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

// Executor triggers ONE comprehensive, durable test-genie run (all phases) with
// the baseline capture profile (full diagnostics + all-pages visuals + video)
// and returns the runID-keyed result. Diffs reuse the same comprehensive run so
// every surface is a view over a single execution.
type Executor interface {
	Execute(ctx context.Context, scenario string) (ExecResult, error)
}

// RunVisual is one page's UI-smoke visual artifact captured by a run under the
// baseline profile (mirrors test-genie's RunsService.RunVisual). GCT diffs the
// page set + screenshot count at the metadata level between two runs.
type RunVisual struct {
	Page                string
	Label               string
	ScreenshotRelPath   string
	ScreenshotSizeBytes int64
}

// RunsClient wraps test-genie's RunsService: pin/unpin a comprehensive run,
// compare two runs (empty phase = every phase), and enumerate a run's visual
// artifacts. GCT owns no run history — test-genie does.
type RunsClient interface {
	PinRun(ctx context.Context, scenario, runID, pinnedBy, reason string) error
	UnpinRun(ctx context.Context, scenario, runID, pinnedBy string) error
	// CompareRuns with an empty phase returns every phase's delta; GCT buckets
	// the PhaseDiff[] into surfaces locally (option-c).
	CompareRuns(ctx context.Context, scenario, runIDA, runIDB, phase string) (CompareResult, error)
	// ListRunVisuals enumerates the per-page visual artifacts a run captured.
	ListRunVisuals(ctx context.Context, scenario, runID string) ([]RunVisual, error)
}

// StalenessProbe computes commits and files changed since a sha. Read-only.
type StalenessProbe interface {
	Since(ctx context.Context, repoDir, sha string) (commits int, files int, err error)
}

// Reachability is a fast, bounded liveness check of the test-genie backend that
// owns the comprehensive run. It is probed BEFORE committing to a multi-minute
// run so an unreachable subsystem skips fast (clear reason) instead of blocking
// to the long execute/compare client deadlines — the reported silent-hang
// class. A nil error means reachable; a non-nil error's message is surfaced to
// the operator as the skip reason.
type Reachability interface {
	// Probe returns nil when test-genie is reachable. Implementations MUST apply
	// their own short timeout (independent of ctx's long deadline) so the probe
	// itself can never be the thing that hangs.
	Probe(ctx context.Context) error
}
