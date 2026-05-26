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

// baselineAuditor wraps scenario-auditor's standards check (start + poll) and
// normalizes violations into baseline.Issue values for diffing.
type baselineAuditor struct {
	client      *AuditorClient
	pollTimeout time.Duration
}

func (a baselineAuditor) Scan(ctx context.Context, scenario, scanType string) ([]baseline.Issue, error) {
	job, err := a.client.StartCheck(ctx, scenario, scanType)
	if err != nil {
		return nil, fmt.Errorf("start auditor check: %w", err)
	}
	deadline := time.Now().Add(a.pollTimeout)
	status := job.Status
	for {
		switch strings.ToLower(status.Status) {
		case "completed", "complete", "done", "finished":
			return violationsToIssues(status.Result), nil
		case "failed", "error", "cancelled", "canceled":
			return nil, fmt.Errorf("auditor check %s: %s", status.Status, status.Error)
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("auditor check timed out after %s", a.pollTimeout)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
		s, perr := a.client.GetJobStatus(ctx, job.JobID)
		if perr != nil {
			return nil, fmt.Errorf("poll auditor job: %w", perr)
		}
		status = *s
	}
}

func violationsToIssues(result *AuditorCheckResult) []baseline.Issue {
	if result == nil {
		return nil
	}
	issues := make([]baseline.Issue, 0, len(result.Violations))
	for _, v := range result.Violations {
		// Key excludes the volatile line number so a shifted-but-identical
		// violation is not misread as cleared+regressed.
		key := strings.Join([]string{v.Standard, v.Type, v.FilePath, v.Title}, "|")
		issues = append(issues, baseline.Issue{
			Key:      key,
			Severity: v.Severity,
			Title:    v.Title,
			FilePath: v.FilePath,
		})
	}
	return issues
}

// baselineVisualClient wraps GCT's visual snapshot capture + storage.
type baselineVisualClient struct {
	bas     *BrowserAutomationClient
	storage *VisualCaptureStorage
	fs      FileIO
}

func (v baselineVisualClient) Capture(ctx context.Context, repoID int64, repoDir, scenario string) (baseline.VisualSnapshot, error) {
	meta, err := CaptureScenario(ctx, VisualCaptureDeps{
		BAS:     v.bas,
		Storage: v.storage,
		FS:      v.fs,
		RepoDir: repoDir,
		RepoID:  repoID,
	}, VisualCaptureRequest{ScenarioSlug: scenario, Mode: "capture", TriggerType: "manual"})
	if err != nil {
		return baseline.VisualSnapshot{}, err
	}
	return baseline.VisualSnapshot{
		SnapshotID:      meta.ID,
		ScreenshotCount: meta.ScreenshotCount,
		Pages:           meta.Pages,
	}, nil
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

	filesOut, err := gitReadOnly(ctx, repoDir, "diff", "--name-only", sha, "HEAD")
	if err != nil {
		return commits, 0, err
	}
	files := 0
	for _, line := range strings.Split(strings.TrimSpace(filesOut), "\n") {
		if strings.TrimSpace(line) != "" {
			files++
		}
	}
	return commits, files, nil
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
	auditor := baselineAuditor{client: s.auditorClient, pollTimeout: 10 * time.Minute}
	visual := baselineVisualClient{bas: s.basClient, storage: s.visualCaptureStorage, fs: OSFileIO{}}
	snaps := baseline.NewSnapshotStore(s.storageResolver)

	adapters := map[string]baseline.SurfaceAdapter{
		baseline.SurfaceWorkflows: baseline.NewWorkflowsAdapter(exec, runs),
		baseline.SurfaceTests:     baseline.NewTestsAdapter(exec, runs),
		baseline.SurfaceStructure: baseline.NewStructureAdapter(auditor, snaps),
		baseline.SurfaceRules:     baseline.NewRulesAdapter(auditor, snaps),
		baseline.SurfaceVisuals:   baseline.NewVisualsAdapter(visual),
	}
	return baseline.NewService(baseline.Deps{
		Storage:  baseline.NewStorage(s.storageResolver),
		Snaps:    snaps,
		Adapters: adapters,
		Probe:    baselineStalenessProbe{},
		Runs:     runs,
	})
}
