// Sandbox-cwd contract integration test.
//
// This is the regression gate for the "silent sandbox bypass" — the
// production bug fixed in Phase 1 of the agent-manager reliability pass,
// where SandboxConfig.Mode said "Protected" but the codec received the
// canonical project root as its WorkingDir (RunMode silently downgraded
// to in-place via a Go zero-value bool on the profile).
//
// The lower-layer regression tests pin individual links of the chain
// (DeriveRunMode, ApplyProfile, LauncherSelector, sandbox path
// translation). This test composes the full pipeline:
//
//	profile{Mode=Protected} ─► CreateRun (no RunMode override)
//	    ─► DeriveRunMode ─► RunModeSandboxed
//	    ─► SetupWorkspace ─► sandbox.Provider.Create + GetWorkspacePath
//	    ─► RunExecutor ─► runner.Execute(req)
//	    ─► assert: req.WorkingDir == provider's workspace path
//	            ⋀ req.SandboxID != nil
//
// And the negative control:
//
//	profile{Mode=Off} ─► RunModeInPlace ─► UseInPlaceWorkspace
//	    ─► assert: req.WorkingDir == task.ProjectRoot
//	            ⋀ req.SandboxID == nil
//
// If a future change re-introduces a bypass — by duplicating the run-mode
// decision, by adding a new "downgrade if X" branch, or by routing the
// codec around the workspace path — this test fails loudly with the exact
// observed WorkingDir, naming the bug class on the first read.
package integration

import (
	"context"
	"path/filepath"
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

// =============================================================================
// Capturing runner: records what each invocation actually saw
// =============================================================================

type capturedRequest struct {
	WorkingDir string
	SandboxID  *uuid.UUID
}

type capturingRunner struct {
	mu       sync.Mutex
	captured []capturedRequest
}

func (c *capturingRunner) record(req runner.ExecuteRequest) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var sbxID *uuid.UUID
	if req.SandboxID != nil {
		v := *req.SandboxID
		sbxID = &v
	}
	c.captured = append(c.captured, capturedRequest{
		WorkingDir: req.WorkingDir,
		SandboxID:  sbxID,
	})
}

func (c *capturingRunner) snapshot() []capturedRequest {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]capturedRequest, len(c.captured))
	copy(out, c.captured)
	return out
}

// newCapturingMockRunner wires the capture into a MockRunner whose
// Execute returns success without doing anything else, so the run can
// reach a terminal state quickly and the assertions run against a clean
// snapshot.
func newCapturingMockRunner(rt domain.RunnerType, sink *capturingRunner) *runner.MockRunner {
	mock := runner.NewMockRunner(rt)
	mock.SetAvailable(true, "ready")
	mock.ExecuteFunc = func(_ context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		sink.record(req)
		return &runner.ExecuteResult{
			Success:  true,
			ExitCode: 0,
			Summary:  &domain.RunSummary{Description: "ok"},
		}, nil
	}
	return mock
}

// =============================================================================
// Fake sandbox provider: returns a known workspace path so the assertion
// can pin the exact value the codec must see.
// =============================================================================

type fakeSandboxProvider struct {
	mu sync.Mutex

	sandboxID     uuid.UUID // ID returned by Create
	workDir       string    // Sandbox.WorkDir surfaced on Create
	workspacePath string    // value returned by GetWorkspacePath

	// Recorded for assertions.
	createCalls  int
	getPathCalls int
	deleteCalls  int
}

func (f *fakeSandboxProvider) Create(_ context.Context, req sandbox.CreateRequest) (*sandbox.Sandbox, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCalls++
	return &sandbox.Sandbox{
		ID:               f.sandboxID,
		ScopePath:        req.ScopePath,
		ProjectRoot:      req.ProjectRoot,
		Status:           sandbox.SandboxStatusActive,
		WorkDir:          f.workDir,
		HomeOverlayState: sandbox.HomeOverlayPresent,
		CreatedAt:        time.Now(),
	}, nil
}

func (f *fakeSandboxProvider) Get(_ context.Context, id uuid.UUID) (*sandbox.Sandbox, error) {
	return &sandbox.Sandbox{
		ID:               id,
		Status:           sandbox.SandboxStatusActive,
		WorkDir:          f.workDir,
		HomeOverlayState: sandbox.HomeOverlayPresent,
	}, nil
}

func (f *fakeSandboxProvider) Delete(_ context.Context, _ uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteCalls++
	return nil
}

func (f *fakeSandboxProvider) GetWorkspacePath(_ context.Context, _ uuid.UUID) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getPathCalls++
	return f.workspacePath, nil
}

func (f *fakeSandboxProvider) GetDiff(_ context.Context, _ uuid.UUID) (*sandbox.DiffResult, error) {
	return &sandbox.DiffResult{}, nil
}

func (f *fakeSandboxProvider) Approve(_ context.Context, _ sandbox.ApproveRequest) (*sandbox.ApproveResult, error) {
	return &sandbox.ApproveResult{Success: true}, nil
}

func (f *fakeSandboxProvider) Reject(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (f *fakeSandboxProvider) PartialApprove(_ context.Context, _ sandbox.PartialApproveRequest) (*sandbox.ApproveResult, error) {
	return &sandbox.ApproveResult{Success: true}, nil
}

func (f *fakeSandboxProvider) ApplyAtRunEnd(_ context.Context, _ sandbox.ApplyAtRunEndRequest) (*sandbox.ApplyAtRunEndResult, error) {
	return &sandbox.ApplyAtRunEndResult{Success: true, AppliedAt: time.Now()}, nil
}

func (f *fakeSandboxProvider) Stop(_ context.Context, _ uuid.UUID) error  { return nil }
func (f *fakeSandboxProvider) Start(_ context.Context, _ uuid.UUID) error { return nil }

func (f *fakeSandboxProvider) IsAvailable(_ context.Context) (bool, string) { return true, "" }

func (f *fakeSandboxProvider) ValidatePath(_ context.Context, path string, _ string) (*sandbox.PathValidationResult, error) {
	return &sandbox.PathValidationResult{Path: path, Valid: true, Exists: true, IsDirectory: true, WithinProjectRoot: true}, nil
}

func (f *fakeSandboxProvider) ExecProcess(_ context.Context, _ sandbox.ExecProcessRequest) (*sandbox.ExecProcessResult, error) {
	return &sandbox.ExecProcessResult{ExitCode: 0}, nil
}

// Compile-time interface check: if sandbox.Provider grows a method, this
// fails loudly here rather than at the test wiring site.
var _ sandbox.Provider = (*fakeSandboxProvider)(nil)

// =============================================================================
// TestSandboxCwdContract — the regression gate
// =============================================================================

// TestSandboxCwdContract_ProtectedRoutesThroughSandbox is the bypass
// regression gate. With Mode=Protected on the profile (and no req.RunMode
// override), the runner MUST see the sandbox-managed workspace path —
// not the canonical project root.
func TestSandboxCwdContract_ProtectedRoutesThroughSandbox(t *testing.T) {
	ctx := context.Background()

	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	projectRoot := t.TempDir()

	sandboxID := uuid.New()
	sandboxRoot := t.TempDir()
	wantWorkspacePath := filepath.Join(sandboxRoot, sandboxID.String(), "merged")
	provider := &fakeSandboxProvider{
		sandboxID:     sandboxID,
		workDir:       filepath.Join(sandboxRoot, sandboxID.String()),
		workspacePath: wantWorkspacePath,
	}

	capture := &capturingRunner{}
	mockRunner := newCapturingMockRunner(domain.RunnerTypeClaudeCode, capture)
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
		orchestration.WithSandbox(provider),
	)

	profile, err := svc.CreateProfile(ctx, &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "sandbox-cwd-contract-protected",
		RunnerType: domain.RunnerTypeClaudeCode,
		Model:      "mock-model",
		// The contract under test: Mode is the single source of truth
		// for whether this run goes through the sandbox provider.
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
	})
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	task, err := svc.CreateTask(ctx, &domain.Task{
		ID:          uuid.New(),
		Title:       "sandbox-cwd-contract",
		Description: "regression gate for the sandbox bypass",
		ScopePath:   projectRoot,
		ProjectRoot: projectRoot,
		Status:      domain.TaskStatusQueued,
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// NOTE: deliberately no req.RunMode — the test is that DeriveRunMode
	// picks RunModeSandboxed from the profile's Mode=Protected. If a
	// future regression downgrades it (the original bypass shape), the
	// codec WorkingDir assertion below catches it.
	run, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Prompt:         "sandbox-cwd contract test",
	})
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	finalRun, err := waitForTerminal(t, ctx, svc, run.ID, 15*time.Second)
	if err != nil {
		t.Fatalf("waitForTerminal: %v", err)
	}

	if finalRun.RunMode != domain.RunModeSandboxed {
		t.Fatalf("RunMode = %q, want %q (Mode=Protected must derive to sandboxed)",
			finalRun.RunMode, domain.RunModeSandboxed)
	}

	provider.mu.Lock()
	createCalls := provider.createCalls
	getPathCalls := provider.getPathCalls
	provider.mu.Unlock()
	if createCalls == 0 {
		t.Fatalf("sandbox.Provider.Create was never called — sandbox bypassed at SetupWorkspace")
	}
	if getPathCalls == 0 {
		t.Fatalf("sandbox.Provider.GetWorkspacePath was never called — workspace path not resolved")
	}

	requests := capture.snapshot()
	if len(requests) == 0 {
		t.Fatalf("runner.Execute was never invoked — orchestrator never reached the codec")
	}
	got := requests[0]

	// Primary assertion: the codec's WorkingDir is the sandbox-managed
	// path returned by the provider. This is the bypass-class regression
	// gate. If a future change re-introduces a path translation that
	// chdirs into the canonical repo, this assertion fails with the
	// observed WorkingDir naming the bug.
	if got.WorkingDir != wantWorkspacePath {
		t.Errorf("WorkingDir = %q, want %q (sandbox-managed workspace path)",
			got.WorkingDir, wantWorkspacePath)
	}
	if got.WorkingDir == projectRoot {
		t.Errorf("WorkingDir == ProjectRoot %q — silent sandbox bypass: codec is editing the canonical repo",
			projectRoot)
	}

	// Secondary assertion: the codec must also know which sandbox it is
	// running against. SandboxLauncher uses this to route exec calls
	// through workspace-sandbox; nil here means the launcher selector
	// would fall back to HostLauncher (the bypass shape, just at a
	// different layer).
	if got.SandboxID == nil {
		t.Fatalf("ExecuteRequest.SandboxID is nil — runner cannot route through SandboxLauncher")
	}
	if *got.SandboxID != sandboxID {
		t.Errorf("ExecuteRequest.SandboxID = %v, want %v (provider returned a different ID than the codec received)",
			*got.SandboxID, sandboxID)
	}
	if finalRun.SandboxID == nil || *finalRun.SandboxID != sandboxID {
		t.Errorf("Run.SandboxID = %v, want %v (sandbox UUID not persisted on the run)",
			finalRun.SandboxID, sandboxID)
	}
}

// TestSandboxCwdContract_OffRunsInPlace is the negative control. With
// Mode=Off explicitly set, the run MUST execute in-place at the
// project root and never touch the sandbox provider.
func TestSandboxCwdContract_OffRunsInPlace(t *testing.T) {
	ctx := context.Background()

	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	projectRoot := t.TempDir()

	provider := &fakeSandboxProvider{
		sandboxID:     uuid.New(),
		workDir:       "/should/never/be/used",
		workspacePath: "/should/never/be/used/merged",
	}

	capture := &capturingRunner{}
	mockRunner := newCapturingMockRunner(domain.RunnerTypeClaudeCode, capture)
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
			RequireSandboxByDefault: false,
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency),
		orchestration.WithSandbox(provider),
	)

	profile, err := svc.CreateProfile(ctx, &domain.AgentProfile{
		ID:            uuid.New(),
		Name:          "sandbox-cwd-contract-off",
		RunnerType:    domain.RunnerTypeClaudeCode,
		Model:         "mock-model",
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff},
	})
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	task, err := svc.CreateTask(ctx, &domain.Task{
		ID:          uuid.New(),
		Title:       "sandbox-cwd-contract-off",
		Description: "negative control for the sandbox bypass gate",
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
		Prompt:         "off-mode contract test",
	})
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	finalRun, err := waitForTerminal(t, ctx, svc, run.ID, 15*time.Second)
	if err != nil {
		t.Fatalf("waitForTerminal: %v", err)
	}

	if finalRun.RunMode != domain.RunModeInPlace {
		t.Fatalf("RunMode = %q, want %q (Mode=Off must derive to in-place)",
			finalRun.RunMode, domain.RunModeInPlace)
	}

	provider.mu.Lock()
	createCalls := provider.createCalls
	provider.mu.Unlock()
	if createCalls != 0 {
		t.Errorf("sandbox.Provider.Create was called %d times for an in-place run; expected 0", createCalls)
	}

	requests := capture.snapshot()
	if len(requests) == 0 {
		t.Fatalf("runner.Execute was never invoked")
	}
	got := requests[0]

	if got.WorkingDir != projectRoot {
		t.Errorf("WorkingDir = %q, want %q (in-place run must execute at project root)",
			got.WorkingDir, projectRoot)
	}
	if got.SandboxID != nil {
		t.Errorf("ExecuteRequest.SandboxID = %v, want nil (in-place runs have no sandbox)", *got.SandboxID)
	}
	if finalRun.SandboxID != nil {
		t.Errorf("Run.SandboxID = %v, want nil (in-place runs persist no sandbox UUID)", *finalRun.SandboxID)
	}
}
