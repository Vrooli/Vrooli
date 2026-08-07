package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
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

// terminalRunStatuses are the test-genie run statuses that mean "no longer
// executing" — the diff verdict can be computed once a run reaches one.
func isTerminalRunStatus(status string) bool {
	switch status {
	case "passed", "failed", "aborted":
		return true
	default:
		return false
	}
}

// asRunBusy extracts a typed RunBusyError from a test-genie StartRun rejection:
// the one-run-per-scenario guard returns FailedPrecondition with a RunBusyInfo
// detail when a divergent run is already in flight. Returns nil for any other
// error so the caller propagates it unchanged.
func asRunBusy(err error) *baseline.RunBusyError {
	var ce *connect.Error
	if !errors.As(err, &ce) || ce.Code() != connect.CodeFailedPrecondition {
		return nil
	}
	for _, d := range ce.Details() {
		msg, derr := d.Value()
		if derr != nil {
			continue
		}
		if bi, ok := msg.(*runspb.RunBusyInfo); ok {
			return &baseline.RunBusyError{Scenario: bi.GetScenario(), RunID: bi.GetRunId(), Preset: bi.GetPreset()}
		}
	}
	return nil
}

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
// blocks on WaitRun and consumes the canonical terminal RunInfo carried by that
// response. The
// comprehensive preset is catalog-derived (drift-proof) in test-genie, so a
// baseline always covers every phase.
type baselineExecutor struct {
	runs baselineRunsClient
}

const baselineAdmissionCaller = "git-control-tower:baseline"

var (
	baselineAdmissionRetryDelays = []time.Duration{250 * time.Millisecond, 500 * time.Millisecond, time.Second, 2 * time.Second}
	baselineWaitRetryDelays      = []time.Duration{250 * time.Millisecond, 500 * time.Millisecond, time.Second, 2 * time.Second}
)

func (e baselineExecutor) StartRun(ctx context.Context, scenario string) (baseline.RunHandle, error) {
	cl, err := e.runs.client(ctx)
	if err != nil {
		return baseline.RunHandle{}, err
	}
	request := baselineStartRequest(scenario)
	resp, err := startBaselineRunWithRetry(ctx, func() (*connect.Response[runspb.StartRunResponse], error) {
		return cl.StartRun(ctx, request)
	})
	if err != nil {
		if busy := asRunBusy(err); busy != nil {
			return baseline.RunHandle{}, busy
		}
		return baseline.RunHandle{}, err
	}
	return baseline.RunHandle{
		RunID:                 resp.Msg.GetRunId(),
		EstimatedTotalSeconds: int(resp.Msg.GetEstimatedTotalSeconds()),
		EtaKnown:              resp.Msg.GetEtaKnown(),
		Coalesced:             resp.Msg.GetCoalesced(),
	}, nil
}

// startBaselineRunWithRetry keeps short-lived preview contention from becoming
// a terminal baseline failure. The parent capture context remains authoritative
// so a degraded Test Genie cannot make Git Control Tower wait indefinitely.
func startBaselineRunWithRetry(ctx context.Context, start func() (*connect.Response[runspb.StartRunResponse], error)) (*connect.Response[runspb.StartRunResponse], error) {
	for attempt := 0; ; attempt++ {
		response, err := start()
		if err == nil || !isPreviewSaturated(err) || attempt == len(baselineAdmissionRetryDelays) {
			return response, err
		}
		timer := time.NewTimer(baselineAdmissionRetryDelays[attempt])
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("wait for test-genie admission: %w", ctx.Err())
		case <-timer.C:
		}
	}
}

func isPreviewSaturated(err error) bool {
	var connectErr *connect.Error
	return errors.As(err, &connectErr) && connectErr.Code() == connect.CodeResourceExhausted
}

func baselineStartRequest(scenario string) *connect.Request[runspb.StartRunRequest] {
	request := connect.NewRequest(&runspb.StartRunRequest{
		Scenario:       scenario,
		Preset:         "comprehensive",
		CaptureProfile: "baseline",
		// Ordinary baseline capture deliberately uses shared-scoped provenance.
		// Strict linked-worktree evidence remains available to callers that ask
		// for it, but is never a prerequisite for retaining before behavior.
	})
	// Baselines are a trusted gateway workload. Without attribution they share
	// Test Genie's anonymous preview bucket with unrelated clients and can be
	// rejected despite available global capacity.
	request.Header().Set("X-Vrooli-Caller", baselineAdmissionCaller)
	return request
}

// RunStatus returns a non-blocking lifecycle snapshot via GetRunStatus.
func (e baselineExecutor) RunStatus(ctx context.Context, scenario, runID string) (baseline.RunStatusInfo, error) {
	cl, err := e.runs.client(ctx)
	if err != nil {
		return baseline.RunStatusInfo{}, err
	}
	resp, err := cl.GetRunStatus(ctx, connect.NewRequest(&runspb.GetRunStatusRequest{
		Scenario: scenario, RunId: runID,
	}))
	if err != nil {
		var connectErr *connect.Error
		if errors.As(err, &connectErr) && connectErr.Code() == connect.CodeNotFound {
			return baseline.RunStatusInfo{Missing: true}, nil
		}
		return baseline.RunStatusInfo{}, err
	}
	st := resp.Msg
	return baseline.RunStatusInfo{
		Status:                      st.GetStatus(),
		Terminal:                    isTerminalRunStatus(st.GetStatus()),
		Success:                     st.GetSuccess(),
		RecommendedNextCheckSeconds: int(st.GetRecommendedNextCheckSeconds()),
		Standing:                    st.GetStanding(),
	}, nil
}

// FindReusableRun lets Test Genie compute the current declared scope and
// validation-contract fingerprints beside the run index. This avoids using the
// whole Git worktree (or a commit SHA) as a cache key in shared workspaces.
func (e baselineExecutor) FindReusableRun(ctx context.Context, scenario string) (baseline.ReusableRun, bool, error) {
	cl, err := e.runs.client(ctx)
	if err != nil {
		return baseline.ReusableRun{}, false, err
	}
	resp, err := cl.FindRun(ctx, connect.NewRequest(&runspb.FindRunRequest{
		Scenario:           scenario,
		Preset:             "comprehensive",
		CaptureProfile:     "baseline",
		Status:             "passed",
		MatchCurrentSource: true,
	}))
	if err != nil {
		return baseline.ReusableRun{}, false, err
	}
	if !resp.Msg.GetFound() {
		return baseline.ReusableRun{}, false, nil
	}
	info := resp.Msg.GetRun()
	completedAt, _ := time.Parse(time.RFC3339, info.GetCompletedAt())
	return baseline.ReusableRun{RunID: info.GetRunId(), CompletedAt: completedAt}, true, nil
}

func (e baselineExecutor) AwaitResult(ctx context.Context, scenario, runID string) (baseline.ExecResult, error) {
	cl, err := e.runs.client(ctx)
	if err != nil {
		return baseline.ExecResult{}, err
	}
	waited, err := waitForBaselineTerminal(ctx, func() (*connect.Response[runspb.WaitRunResponse], error) {
		return cl.WaitRun(ctx, connect.NewRequest(&runspb.WaitRunRequest{
			Scenario: scenario, RunId: runID,
		}))
	})
	if err != nil {
		return baseline.ExecResult{}, err
	}
	if reasons := waited.Msg.GetDegradedReasons(); len(reasons) > 0 {
		return baseline.ExecResult{}, fmt.Errorf("run %s terminal evidence is degraded: %s", runID, strings.Join(reasons, "; "))
	}
	info := waited.Msg.GetTerminalRun()
	if info == nil || waited.Msg.GetTerminalSnapshotSchemaVersion() == 0 {
		return baseline.ExecResult{}, fmt.Errorf("run %s has no canonical terminal snapshot", runID)
	}
	switch info.GetStatus() {
	case "aborted", "timeout", "errored", "queued", "in_progress":
		return baseline.ExecResult{}, fmt.Errorf("run %s ended without comparable baseline artifacts (status=%s)", runID, info.GetStatus())
	}
	completedAt, _ := time.Parse(time.RFC3339, info.GetCompletedAt())
	out := baseline.ExecResult{
		RunID:                             info.GetRunId(),
		Success:                           info.GetStatus() == "passed",
		CompletedAt:                       completedAt,
		TreeDigest:                        info.GetTreeDigest(),
		PhaseSetDigest:                    info.GetPhaseSetDigest(),
		CaptureProfile:                    info.GetCaptureProfile(),
		DescriptorSnapshotDigest:          info.GetDescriptorSnapshotDigest(),
		DescriptorSnapshotSchemaVersion:   int(info.GetDescriptorSnapshotSchemaVersion()),
		GitSha:                            info.GetGitSha(),
		GitDirty:                          info.GetGitDirty(),
		ExecutionConfigurationFingerprint: info.GetExecutionConfigurationFingerprint(),
		GateQuality:                       info.GetGateQuality(),
		EvidenceTier:                      info.GetEvidenceTier(),
		SourceScope:                       info.GetSourceScope(),
		SourceStable:                      info.GetSourceStable(),
	}
	for _, p := range info.GetPhases() {
		out.Phases = append(out.Phases, baseline.PhaseStatus{Name: p.GetName(), Status: p.GetStatus()})
	}
	return out, nil
}

// waitForBaselineTerminal reattaches when a durable WaitRun attachment returns
// a live snapshot. WaitRun normally blocks until terminal, but its contract
// also permits a non-terminal snapshot when the attachment is interrupted.
// Treating that response as missing terminal evidence permanently fails a
// baseline even though the server-owned run is still progressing.
func waitForBaselineTerminal(ctx context.Context, wait func() (*connect.Response[runspb.WaitRunResponse], error)) (*connect.Response[runspb.WaitRunResponse], error) {
	for attempt := 0; ; attempt++ {
		response, err := wait()
		if err != nil {
			if isTransientBaselineWaitError(err) && attempt < len(baselineWaitRetryDelays) {
				timer := time.NewTimer(baselineWaitRetryDelays[attempt])
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil, ctx.Err()
				case <-timer.C:
					continue
				}
			}
			return nil, err
		}
		status := ""
		if response != nil && response.Msg != nil && response.Msg.GetStatus() != nil {
			status = response.Msg.GetStatus().GetStatus()
		}
		if isTerminalRunStatus(status) || (response != nil && response.Msg != nil && response.Msg.GetTerminalRun() != nil) {
			return response, nil
		}
		if status != "" && status != "queued" && status != "in_progress" {
			return response, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
}

func isTransientBaselineWaitError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || strings.Contains(message, "unexpected eof")
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
	out := baseline.CompareResult{Comparison: resp.Msg, Verdict: resp.Msg.GetVerdict(), Phases: append([]*runspb.PhaseDiff(nil), resp.Msg.GetPhases()...)}
	return out, nil
}

// ListRunArtifacts consumes Test Genie's typed, path-free evidence catalog.
func (c baselineRunsClient) ListRunArtifacts(ctx context.Context, scenario, runID string) (baseline.ArtifactCatalog, error) {
	cl, err := c.client(ctx)
	if err != nil {
		return baseline.ArtifactCatalog{}, err
	}
	resp, err := cl.ListRunArtifacts(ctx, connect.NewRequest(&runspb.ListRunArtifactsRequest{
		Scenario: scenario, RunId: runID,
	}))
	if err != nil {
		return baseline.ArtifactCatalog{}, err
	}
	return baseline.ArtifactCatalog{
		RunID:            runID,
		SchemaVersion:    int(resp.Msg.GetSchemaVersion()),
		Digest:           resp.Msg.GetDigest(),
		Artifacts:        resp.Msg.GetArtifacts(),
		LegacyDiscovered: resp.Msg.GetLegacyDiscovered(),
		DegradedReasons:  resp.Msg.GetDegradedReasons(),
	}, nil
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
		ReuseTTL:  diffRunReuseTTL(),
	})
}

// defaultDiffRunReuseTTL bounds clean-tree run reuse: a completed run at the
// current sha is reused only when it finished within this window, so a diff
// never serves a verdict from a run old enough that the environment (deps,
// external state) may have drifted even though the tree sha matches.
const defaultDiffRunReuseTTL = 15 * time.Minute

// diffRunReuseTTL resolves the reuse window. Lever GCT_DIFF_RUN_REUSE_TTL
// accepts a Go duration ("15m", "1h"); "0" disables reuse entirely.
func diffRunReuseTTL() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("GCT_DIFF_RUN_REUSE_TTL")); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d >= 0 {
			return d
		}
	}
	return defaultDiffRunReuseTTL
}
