// Closed-run diff integration test — the user-visible win of the
// diff-archive plan.
//
// Pre-archive, the failure mode was: an agent-manager run completes,
// the sandbox is auto-deleted by lifecycle policy, the user opens the
// run and sees an empty DiffViewer because GetRunDiff hits a sandbox
// whose overlay is unmounted. The fix is durable archives in
// workspace-sandbox: GET /sandboxes/{id}/diff transparently serves
// from the archive when the sandbox is in a terminal status, and the
// adapter passes the response through unchanged.
//
// This test pins the agent-manager half of that flow:
//
//   1. A run with a sandbox completes.
//   2. The sandbox is "torn down" (the fake provider continues to
//      respond, but signals the run is closed by returning an archive-
//      shaped DiffResult with ArchiveState set).
//   3. svc.GetRunDiff(ctx, runID) returns the archive content with
//      ArchiveState preserved end-to-end — Files, UnifiedDiff, Stats,
//      and ArchiveState all flow through GetRun → GetDiff → caller.
//
// Without the archive seam, GetRunDiff would return an empty diff and
// no consumer (UI, CLI, agent-manager) would have a way to tell the
// difference between "run had no changes" and "the diff was lost when
// the sandbox was deleted." That's the bug class this test pins.
//
// Negative variant: archived-but-not-captured (Error → Deleted). The
// adapter must surface ArchiveState=NotCaptured with empty Files so
// the UI renders an explicit "no diff captured" state instead of the
// generic "no changes" empty state. This is the second user-visible
// distinction the archive-state field exists to make.

package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/testutil"

	"github.com/google/uuid"
)

// archiveAwareSandboxProvider is a fake sandbox.Provider whose GetDiff
// returns a configurable DiffResult — including the new ArchiveState
// field. All other methods are minimum-viable so a run can flow
// through the orchestrator to completion. This is structurally similar
// to fakeSandboxProvider in sandbox_cwd_contract_test.go but
// specialized for diff-archive contract testing; keeping it separate
// avoids growing the cwd-contract fake with concerns it doesn't own.
type archiveAwareSandboxProvider struct {
	mu        sync.Mutex
	sandboxID uuid.UUID
	workDir   string
	workspace string

	// Configurable diff response — set by the test before the assertion
	// site so we can pin exactly what the archive endpoint returns.
	diffResp *sandbox.DiffResult
}

func (p *archiveAwareSandboxProvider) Create(_ context.Context, req sandbox.CreateRequest) (*sandbox.Sandbox, error) {
	return &sandbox.Sandbox{
		ID:               p.sandboxID,
		ScopePath:        req.ScopePath,
		ProjectRoot:      req.ProjectRoot,
		Status:           sandbox.SandboxStatusActive,
		WorkDir:          p.workDir,
		HomeOverlayState: sandbox.HomeOverlayPresent,
		CreatedAt:        time.Now(),
	}, nil
}

func (p *archiveAwareSandboxProvider) Get(_ context.Context, id uuid.UUID) (*sandbox.Sandbox, error) {
	return &sandbox.Sandbox{
		ID:               id,
		Status:           sandbox.SandboxStatusActive,
		WorkDir:          p.workDir,
		HomeOverlayState: sandbox.HomeOverlayPresent,
	}, nil
}

func (p *archiveAwareSandboxProvider) Delete(_ context.Context, _ uuid.UUID) error { return nil }

func (p *archiveAwareSandboxProvider) GetWorkspacePath(_ context.Context, _ uuid.UUID) (string, error) {
	return p.workspace, nil
}

func (p *archiveAwareSandboxProvider) GetDiff(_ context.Context, _ uuid.UUID) (*sandbox.DiffResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.diffResp == nil {
		return &sandbox.DiffResult{}, nil
	}
	// Return a copy so test mutation can't bleed across calls.
	c := *p.diffResp
	return &c, nil
}

func (p *archiveAwareSandboxProvider) Approve(_ context.Context, _ sandbox.ApproveRequest) (*sandbox.ApproveResult, error) {
	return &sandbox.ApproveResult{Success: true}, nil
}

func (p *archiveAwareSandboxProvider) Reject(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (p *archiveAwareSandboxProvider) PartialApprove(_ context.Context, _ sandbox.PartialApproveRequest) (*sandbox.ApproveResult, error) {
	return &sandbox.ApproveResult{Success: true}, nil
}

func (p *archiveAwareSandboxProvider) ApplyAtRunEnd(_ context.Context, _ sandbox.ApplyAtRunEndRequest) (*sandbox.ApplyAtRunEndResult, error) {
	return &sandbox.ApplyAtRunEndResult{Success: true, AppliedAt: time.Now()}, nil
}

func (p *archiveAwareSandboxProvider) TurnCheckpoint(_ context.Context, req sandbox.TurnCheckpointRequest) (*sandbox.TurnCheckpointResult, error) {
	return &sandbox.TurnCheckpointResult{SandboxID: req.SandboxID, Status: sandbox.SandboxStatusCheckpointed, Success: true, AppliedAt: time.Now()}, nil
}

func (p *archiveAwareSandboxProvider) Stop(_ context.Context, _ uuid.UUID) error  { return nil }
func (p *archiveAwareSandboxProvider) Start(_ context.Context, _ uuid.UUID) error { return nil }
func (p *archiveAwareSandboxProvider) Resume(_ context.Context, id uuid.UUID) (*sandbox.Sandbox, error) {
	return &sandbox.Sandbox{ID: id, Status: sandbox.SandboxStatusActive, WorkDir: p.workspace, CreatedAt: time.Now()}, nil
}

func (p *archiveAwareSandboxProvider) IsAvailable(_ context.Context) (bool, string) { return true, "" }

func (p *archiveAwareSandboxProvider) ValidatePath(_ context.Context, path string, _ string) (*sandbox.PathValidationResult, error) {
	return &sandbox.PathValidationResult{Path: path, Valid: true, Exists: true, IsDirectory: true, WithinProjectRoot: true}, nil
}

func (p *archiveAwareSandboxProvider) ExecProcess(_ context.Context, _ sandbox.ExecProcessRequest) (*sandbox.ExecProcessResult, error) {
	return &sandbox.ExecProcessResult{ExitCode: 0}, nil
}

var _ sandbox.Provider = (*archiveAwareSandboxProvider)(nil)

// runClosedDiffScenario sets up a sandboxed run that completes, then
// arms the provider with `archive` for the GetRunDiff assertion. Returns
// the orchestrator and runID so the caller can run targeted asserts.
func runClosedDiffScenario(t *testing.T, archive *sandbox.DiffResult) (*orchestration.Orchestrator, uuid.UUID, *archiveAwareSandboxProvider) {
	t.Helper()
	ctx := context.Background()

	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	projectRoot := t.TempDir()

	provider := &archiveAwareSandboxProvider{
		sandboxID: uuid.New(),
		workDir:   t.TempDir(),
		workspace: t.TempDir(),
	}

	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(_ context.Context, _ runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{
			Success:  true,
			ExitCode: 0,
			Summary:  &domain.RunSummary{Description: "ok"},
		}, nil
	}
	registry := runner.NewRegistry()
	if err := registry.Register(mockRunner); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          time.Minute,
			MaxConcurrentRuns:       4,
			RequireSandboxByDefault: true,
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency),
		newTestRolePolicyOption(t),
		orchestration.WithSandbox(provider),
	)

	profile, err := svc.CreateProfile(ctx, &domain.AgentProfile{
		ID:   uuid.New(),
		Name: "closed-run-diff",

		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected}, RoleRef: "code.default",
	})
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	task, err := svc.CreateTask(ctx, &domain.Task{
		ID:          uuid.New(),
		Title:       "closed-run-diff",
		ScopePath:   projectRoot,
		ProjectRoot: projectRoot,
		Status:      domain.TaskStatusQueued,
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	run, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Prompt:         "closed-run-diff scenario",
	})
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	if _, err := waitForTerminal(t, ctx, svc, run.ID, 15*time.Second); err != nil {
		t.Fatalf("waitForTerminal: %v", err)
	}

	// Arm the provider's diff response only after the run is terminal.
	// In production this corresponds to: the sandbox transitioned to
	// Approved/Rejected/Deleted, the workspace-sandbox archive seam
	// fired, and the next /diff request resolves to the archive.
	provider.mu.Lock()
	provider.diffResp = archive
	provider.mu.Unlock()

	return svc, run.ID, provider
}

// TestGetRunDiff_ServesArchivedComplete is the headline integration
// test for the diff-archive feature. It pins:
//
//   - Files, UnifiedDiff, and Stats survive transit through the adapter
//     and the orchestration layer unchanged.
//   - ArchiveState=Complete propagates end-to-end.
//
// Pre-archive, GetRunDiff returned an empty DiffResult (the live
// overlay was gone). With the archive seam, this exact scenario serves
// the captured snapshot.
func TestGetRunDiff_ServesArchivedComplete(t *testing.T) {
	ctx := context.Background()

	archived := &sandbox.DiffResult{
		Files: []sandbox.FileChange{
			{
				ID:           uuid.New(),
				FilePath:     "src/main.go",
				ChangeType:   sandbox.FileChangeModified,
				FileSize:     1024,
				LinesAdded:   12,
				LinesRemoved: 4,
			},
		},
		UnifiedDiff: "diff --git a/src/main.go b/src/main.go\n@@ -1,3 +1,4 @@\n+// added\n",
		Stats: sandbox.DiffStats{
			FilesChanged:  1,
			FilesModified: 1,
			LinesAdded:    12,
			LinesRemoved:  4,
			TotalBytes:    1024,
		},
		ArchiveState: sandbox.ArchiveStateComplete,
	}

	svc, runID, _ := runClosedDiffScenario(t, archived)

	got, err := svc.GetRunDiff(ctx, runID)
	if err != nil {
		t.Fatalf("GetRunDiff: %v", err)
	}
	if got == nil {
		t.Fatal("GetRunDiff returned nil — closed-run diff broken")
	}

	if got.ArchiveState != sandbox.ArchiveStateComplete {
		t.Errorf("ArchiveState = %q, want %q (archive_state must propagate end-to-end)",
			got.ArchiveState, sandbox.ArchiveStateComplete)
	}
	if len(got.Files) != 1 {
		t.Fatalf("Files len = %d, want 1 (archive content must survive the GetRunDiff seam)", len(got.Files))
	}
	if got.Files[0].FilePath != "src/main.go" {
		t.Errorf("Files[0].FilePath = %q, want src/main.go", got.Files[0].FilePath)
	}
	if got.Files[0].LinesAdded != 12 || got.Files[0].LinesRemoved != 4 {
		t.Errorf("Files[0] line counts = (+%d/-%d), want (+12/-4)",
			got.Files[0].LinesAdded, got.Files[0].LinesRemoved)
	}
	if got.Stats.FilesModified != 1 {
		t.Errorf("Stats.FilesModified = %d, want 1", got.Stats.FilesModified)
	}
	if got.Stats.LinesAdded != 12 {
		t.Errorf("Stats.LinesAdded = %d, want 12", got.Stats.LinesAdded)
	}
	if got.UnifiedDiff == "" {
		t.Error("UnifiedDiff is empty — archive's unified diff text was dropped")
	}
}

// TestGetRunDiff_ArchiveNotCapturedDistinguishedFromEmpty pins the
// taxonomy distinction the UI depends on: an archive that was
// deliberately skipped (Error → Deleted) must surface
// ArchiveState=NotCaptured with empty Files, NOT a generic empty diff.
//
// The UI renders these two cases differently — "no diff captured" vs.
// "no changes detected" — so collapsing them silently regresses the
// audit trail.
func TestGetRunDiff_ArchiveNotCapturedDistinguishedFromEmpty(t *testing.T) {
	ctx := context.Background()

	notCaptured := &sandbox.DiffResult{
		Files:        nil,
		UnifiedDiff:  "",
		Stats:        sandbox.DiffStats{},
		ArchiveState: sandbox.ArchiveStateNotCaptured,
	}

	svc, runID, _ := runClosedDiffScenario(t, notCaptured)

	got, err := svc.GetRunDiff(ctx, runID)
	if err != nil {
		t.Fatalf("GetRunDiff: %v", err)
	}
	if got == nil {
		t.Fatal("GetRunDiff returned nil for not_captured archive — should be an empty-content marker, not nil")
	}
	if got.ArchiveState != sandbox.ArchiveStateNotCaptured {
		t.Errorf("ArchiveState = %q, want %q (UI relies on this taxonomy to distinguish 'no capture' from 'no changes')",
			got.ArchiveState, sandbox.ArchiveStateNotCaptured)
	}
	if len(got.Files) != 0 {
		t.Errorf("Files len = %d, want 0 (not_captured archives carry no files)", len(got.Files))
	}
}
