package main

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/discovery"

	"git-control-tower/internal/baseline"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	"github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// ---------------------------------------------------------------------------
// Concrete baseline seam implementations. These adapt GCT's existing clients
// (test-genie, scenario-auditor, visual capture, git) to the interfaces the
// baseline package declares. The baseline package never imports main, so all
// the live-dependency wiring lives here.
// ---------------------------------------------------------------------------

// baselineExecutor drives ONE comprehensive, durable test-genie run with the
// baseline capture profile (full diagnostics + all-pages visuals + video) via
// the durable RunsService: StartRun returns the run handle immediately (so a
// snapshot can return fast and pin server-side on completion), and AwaitResult
// blocks on WaitRun then reads the terminal phase set via GetRun. The
// comprehensive preset is catalog-derived (drift-proof) in test-genie, so a
// baseline always covers every phase.
type baselineExecutor struct {
	runs baselineRunsClient
}

func (e baselineExecutor) StartRun(ctx context.Context, scenario string) (baseline.RunHandle, error) {
	cl, err := e.runs.client(ctx)
	if err != nil {
		return baseline.RunHandle{}, err
	}
	resp, err := cl.StartRun(ctx, connect.NewRequest(&runspb.StartRunRequest{
		Scenario:       scenario,
		Preset:         "comprehensive",
		CaptureProfile: "baseline",
	}))
	if err != nil {
		return baseline.RunHandle{}, err
	}
	return baseline.RunHandle{
		RunID:                 resp.Msg.GetRunId(),
		EstimatedTotalSeconds: int(resp.Msg.GetEstimatedTotalSeconds()),
		EtaKnown:              resp.Msg.GetEtaKnown(),
	}, nil
}

func (e baselineExecutor) AwaitResult(ctx context.Context, scenario, runID string) (baseline.ExecResult, error) {
	cl, err := e.runs.client(ctx)
	if err != nil {
		return baseline.ExecResult{}, err
	}
	if _, err := cl.WaitRun(ctx, connect.NewRequest(&runspb.WaitRunRequest{
		Scenario: scenario, RunId: runID,
	})); err != nil {
		return baseline.ExecResult{}, err
	}
	got, err := cl.GetRun(ctx, connect.NewRequest(&runspb.GetRunRequest{
		Scenario: scenario, RunId: runID,
	}))
	if err != nil {
		return baseline.ExecResult{}, err
	}
	info := got.Msg.GetRun()
	out := baseline.ExecResult{RunID: info.GetRunId(), Success: info.GetStatus() == "passed"}
	for _, p := range info.GetPhases() {
		out.Phases = append(out.Phases, baseline.PhaseStatus{Name: p.GetName(), Status: p.GetStatus()})
	}
	return out, nil
}

// baselineRunsClient wraps test-genie's RunsService Connect-RPC for pin/unpin/
// compare. The URL is resolved through service discovery on every call so the
// client survives test-genie restarts.
type baselineRunsClient struct {
	httpClient *http.Client
	resolveURL func(ctx context.Context) (string, error)
}

func newBaselineRunsClient(timeout time.Duration) baselineRunsClient {
	return baselineRunsClient{
		httpClient: &http.Client{Timeout: timeout},
		resolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "test-genie")
		},
	}
}

func (c baselineRunsClient) client(ctx context.Context) (runs_v1connect.RunsServiceClient, error) {
	baseURL, err := c.resolveURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve test-genie url: %w", err)
	}
	return runs_v1connect.NewRunsServiceClient(c.httpClient, baseURL), nil
}

func (c baselineRunsClient) PinRun(ctx context.Context, scenario, runID, pinnedBy, reason string) error {
	cl, err := c.client(ctx)
	if err != nil {
		return err
	}
	_, err = cl.PinRun(ctx, connect.NewRequest(&runspb.PinRunRequest{
		Scenario: scenario, RunId: runID, PinnedBy: pinnedBy, Reason: reason,
	}))
	return err
}

func (c baselineRunsClient) UnpinRun(ctx context.Context, scenario, runID, pinnedBy string) error {
	cl, err := c.client(ctx)
	if err != nil {
		return err
	}
	_, err = cl.UnpinRun(ctx, connect.NewRequest(&runspb.UnpinRunRequest{
		Scenario: scenario, RunId: runID, PinnedBy: pinnedBy,
	}))
	return err
}

func (c baselineRunsClient) CompareRuns(ctx context.Context, scenario, runIDA, runIDB, phase string) (baseline.CompareResult, error) {
	cl, err := c.client(ctx)
	if err != nil {
		return baseline.CompareResult{}, err
	}
	resp, err := cl.CompareRuns(ctx, connect.NewRequest(&runspb.CompareRunsRequest{
		Scenario: scenario, RunIdA: runIDA, RunIdB: runIDB, Phase: phase,
	}))
	if err != nil {
		return baseline.CompareResult{}, err
	}
	out := baseline.CompareResult{Verdict: resp.Msg.GetVerdict()}
	for _, p := range resp.Msg.GetPhases() {
		out.Phases = append(out.Phases, baseline.PhaseDiff{
			Phase:       p.GetPhase(),
			Verdict:     p.GetVerdict(),
			Regressions: p.GetRegressions(),
			NewFailures: p.GetNewFailures(),
			Preexisting: p.GetPreexistingFailures(),
			Cleared:     p.GetClearedFailures(),
		})
	}
	return out, nil
}

// ListRunVisuals enumerates a run's per-page visual artifacts (page set +
// screenshot count) — the metadata GCT diffs between two baselines' runs.
func (c baselineRunsClient) ListRunVisuals(ctx context.Context, scenario, runID string) ([]baseline.RunVisual, error) {
	cl, err := c.client(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := cl.ListRunVisuals(ctx, connect.NewRequest(&runspb.ListRunVisualsRequest{
		Scenario: scenario, RunId: runID,
	}))
	if err != nil {
		return nil, err
	}
	out := make([]baseline.RunVisual, 0, len(resp.Msg.GetVisuals()))
	for _, v := range resp.Msg.GetVisuals() {
		out = append(out, baseline.RunVisual{
			Page:                v.GetPage(),
			Label:               v.GetLabel(),
			ScreenshotRelPath:   v.GetScreenshotRelPath(),
			ScreenshotSizeBytes: v.GetScreenshotSizeBytes(),
		})
	}
	return out, nil
}

// CompareRunVisuals asks test-genie (the owner of the visual analyzer) to
// compare two runs' captures and returns the neutral per-page deltas GCT renders
// as the advisory "changed" tier.
func (c baselineRunsClient) CompareRunVisuals(ctx context.Context, scenario, baseRunID, curRunID string) ([]baseline.VisualDelta, error) {
	cl, err := c.client(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := cl.CompareRunVisuals(ctx, connect.NewRequest(&runspb.CompareRunVisualsRequest{
		Scenario: scenario, BaseRunId: baseRunID, CurrentRunId: curRunID,
	}))
	if err != nil {
		return nil, err
	}
	out := make([]baseline.VisualDelta, 0, len(resp.Msg.GetDeltas()))
	for _, d := range resp.Msg.GetDeltas() {
		out = append(out, baseline.VisualDelta{
			Page:            d.GetPage(),
			Label:           d.GetLabel(),
			Status:          d.GetStatus(),
			ChangedFraction: d.GetChangedFraction(),
		})
	}
	return out, nil
}

// baselineReachability is a fast, bounded liveness check of the test-genie
// backend: a short-timeout GET /health against the discovery-resolved URL. It is
// probed BEFORE a baseline commits to a multi-minute comprehensive run so an
// unreachable test-genie skips fast (clear reason) instead of blocking to the
// long execute/compare deadlines — the reported silent-hang class.
type baselineReachability struct {
	httpClient *http.Client
	resolveURL func(ctx context.Context) (string, error)
}

func newBaselineReachability(timeout time.Duration) baselineReachability {
	return baselineReachability{
		httpClient: &http.Client{Timeout: timeout},
		resolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "test-genie")
		},
	}
}

// Probe returns nil when test-genie answers GET /health within the probe's own
// short timeout. It bounds itself independently of ctx so the probe can never be
// the thing that hangs (the whole point of fail-fast).
func (r baselineReachability) Probe(ctx context.Context) error {
	probeCtx, cancel := context.WithTimeout(ctx, r.httpClient.Timeout)
	defer cancel()

	baseURL, err := r.resolveURL(probeCtx)
	if err != nil {
		return fmt.Errorf("resolve test-genie url: %w", err)
	}
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/health", nil)
	if err != nil {
		return fmt.Errorf("build health request: %w", err)
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health returned status %d", resp.StatusCode)
	}
	return nil
}

// baselineStalenessProbe counts commits and changed files between a baseline
// sha and current HEAD. Read-only (feedback_no_git_mutations).
type baselineStalenessProbe struct{}

func (baselineStalenessProbe) Since(ctx context.Context, repoDir, sha string) (int, int, error) {
	if sha == "" {
		return 0, 0, nil
	}
	commitsOut, err := gitReadOnly(ctx, repoDir, "rev-list", "--count", sha+"..HEAD")
	if err != nil {
		return 0, 0, err
	}
	commits, _ := strconv.Atoi(strings.TrimSpace(commitsOut))

	// Distinct files changed since the baseline sha, INCLUDING uncommitted
	// edits. `git diff --name-only <sha>` (no second ref) compares the baseline
	// sha against the working tree, so it captures both committed-since changes
	// and uncommitted modifications — the common mid-implementation agent case
	// that the old `sha..HEAD` form reported as zero drift.
	changed := map[string]struct{}{}
	diffOut, err := gitReadOnly(ctx, repoDir, "diff", "--name-only", sha)
	if err != nil {
		return commits, 0, err
	}
	for _, line := range strings.Split(strings.TrimSpace(diffOut), "\n") {
		if s := strings.TrimSpace(line); s != "" {
			changed[s] = struct{}{}
		}
	}
	// Untracked files aren't in `git diff`; pull them from porcelain status.
	if statusOut, serr := gitReadOnly(ctx, repoDir, "status", "--porcelain", "--untracked-files=all"); serr == nil {
		for _, line := range strings.Split(statusOut, "\n") {
			if len(line) > 3 {
				if path := strings.TrimSpace(line[3:]); path != "" {
					changed[path] = struct{}{}
				}
			}
		}
	}
	return commits, len(changed), nil
}

func gitReadOnly(ctx context.Context, repoDir string, args ...string) (string, error) {
	full := append([]string{"--no-optional-locks", "-C", repoDir}, args...)
	out, err := exec.CommandContext(ctx, "git", full...).Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

// baselineRepoResolver maps an optional explicit repoID (0 = active) to a
// concrete (repoID, repoDir) pair for the baseline handler. It reuses
// RepoService's resolution logic (same package, so unexported helpers are
// reachable).
type baselineRepoResolver struct{ repos *RepoService }

func (r baselineRepoResolver) Resolve(ctx context.Context, repoID int64) (int64, string, error) {
	if r.repos == nil {
		return 0, "", fmt.Errorf("repo service unavailable")
	}
	if repoID > 0 {
		rec, err := r.repos.resolveByID(ctx, repoID)
		if err != nil {
			return 0, "", err
		}
		return rec.ID, rec.Path, nil
	}
	if resolved, tried, err := r.repos.resolveFromActive(ctx); tried {
		if err != nil {
			return 0, "", err
		}
		return resolved.ID, resolved.Path, nil
	}
	resolved, err := r.repos.resolveFromRoot(ctx)
	if err != nil {
		return 0, "", err
	}
	return resolved.ID, resolved.Path, nil
}

// newBaselineService assembles the baseline orchestration service. A baseline
// is ONE comprehensive, durable test-genie run pinned once; every surface
// (structure, rules, tests, workflows, visuals) is a phase-set / artifact view
// over that single run. GCT owns no run history — test-genie does (Decision 3).
func (s *Server) newBaselineService() *baseline.Service {
	// 30-minute timeout: AwaitResult's WaitRun blocks server-side until the
	// comprehensive run is terminal, so the executor's client must outlast a long
	// run (bounded the same as the detached snapshot tail).
	execRuns := newBaselineRunsClient(30 * time.Minute)
	exec := baselineExecutor{runs: execRuns}
	// 15-minute timeout: a diff triggers a fresh comprehensive run before
	// comparing.
	runs := newBaselineRunsClient(15 * time.Minute)

	return baseline.NewService(baseline.Deps{
		Storage: baseline.NewStorage(s.storageResolver),
		Exec:    exec,
		Runs:    runs,
		Probe:   baselineStalenessProbe{},
		// Fast-skip an unreachable test-genie in ~5s instead of blocking the
		// whole snapshot to the multi-minute execute/compare deadlines.
		Reachable: newBaselineReachability(5 * time.Second),
	})
}
