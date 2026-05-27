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

// baselineExecutor triggers a test-genie suite run via the REST execute API and
// returns the runID-keyed result.
type baselineExecutor struct {
	client *TestGenieClient
}

func (e baselineExecutor) Execute(ctx context.Context, scenario string, phases []string, diagnosticsPreset string) (baseline.ExecResult, error) {
	res, err := e.client.ExecuteSuite(ctx, TestExecutionRequest{
		ScenarioName:      scenario,
		Phases:            phases,
		DiagnosticsPreset: diagnosticsPreset,
	})
	if err != nil {
		return baseline.ExecResult{}, err
	}
	out := baseline.ExecResult{RunID: res.RunID, Success: res.Success}
	for _, p := range res.Phases {
		out.Phases = append(out.Phases, baseline.PhaseStatus{Name: p.Name, Status: p.Status})
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

// baselineVisualClient wraps GCT's visual snapshot capture + storage.
type baselineVisualClient struct {
	bas     *BrowserAutomationClient
	storage *VisualCaptureStorage
	fs      FileIO
}

func (v baselineVisualClient) Capture(ctx context.Context, repoID int64, repoDir, scenario string, pinned bool) (baseline.VisualSnapshot, error) {
	// Pinned captures use baseline mode (role=baseline, additive) so routine
	// loose captures never delete them and multiple baselines coexist. Transient
	// captures (the "current" side of a diff) use capture mode — they replace the
	// single loose snapshot and don't accumulate.
	mode := CaptureModeCapture
	if pinned {
		mode = CaptureModeBaseline
	}
	meta, err := CaptureScenario(ctx, VisualCaptureDeps{
		BAS:     v.bas,
		Storage: v.storage,
		FS:      v.fs,
		RepoDir: repoDir,
		RepoID:  repoID,
	}, VisualCaptureRequest{ScenarioSlug: scenario, Mode: mode, TriggerType: "manual"})
	if err != nil {
		return baseline.VisualSnapshot{}, err
	}
	return baseline.VisualSnapshot{
		SnapshotID:      meta.ID,
		ScreenshotCount: meta.ScreenshotCount,
		Pages:           meta.Pages,
	}, nil
}

// Delete removes a pinned visual snapshot when its owning baseline is deleted.
func (v baselineVisualClient) Delete(_ context.Context, repoID int64, scenario, snapshotID string) error {
	if snapshotID == "" {
		return nil
	}
	return v.storage.DeleteSnapshotSet(repoID, scenario, snapshotID)
}

func (v baselineVisualClient) Get(ctx context.Context, repoID int64, scenario, snapshotID string) (baseline.VisualSnapshot, bool, error) {
	detail, err := v.storage.GetSnapshotSet(repoID, scenario, snapshotID)
	if err != nil {
		return baseline.VisualSnapshot{}, false, nil
	}
	return baseline.VisualSnapshot{
		SnapshotID:      detail.ID,
		ScreenshotCount: detail.ScreenshotCount,
		Pages:           detail.Pages,
	}, true, nil
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

// newBaselineService assembles the baseline orchestration service with all
// surface adapters wired to GCT's live clients.
func (s *Server) newBaselineService() *baseline.Service {
	exec := baselineExecutor{client: s.testGenieClient}
	// 15-minute timeout: diff triggers a fresh test-genie run before comparing.
	runs := newBaselineRunsClient(15 * time.Minute)
	visual := baselineVisualClient{bas: s.basClient, storage: s.visualCaptureStorage, fs: OSFileIO{}}

	// structure → test-genie "structure" phase; rules → "standards" phase. Both
	// run through test-genie (which itself invokes scenario-auditor for
	// standards), so GCT pins runs rather than calling scenario-auditor directly
	// (Decision 3) — mirroring workflows↔playbooks.
	adapters := map[string]baseline.SurfaceAdapter{
		baseline.SurfaceWorkflows: baseline.NewWorkflowsAdapter(exec, runs),
		baseline.SurfaceTests:     baseline.NewTestsAdapter(exec, runs),
		baseline.SurfaceStructure: baseline.NewStructureAdapter(exec, runs),
		baseline.SurfaceRules:     baseline.NewRulesAdapter(exec, runs),
		baseline.SurfaceVisuals:   baseline.NewVisualsAdapter(visual),
	}
	return baseline.NewService(baseline.Deps{
		Storage:  baseline.NewStorage(s.storageResolver),
		Adapters: adapters,
		Probe:    baselineStalenessProbe{},
		Runs:     runs,
	})
}
