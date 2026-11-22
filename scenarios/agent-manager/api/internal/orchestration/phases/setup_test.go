package phases

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil/fixtures"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

type fakeWorkspaceSandboxEnsurer struct {
	calls int32
	fn    func(context.Context) error
}

func (f *fakeWorkspaceSandboxEnsurer) EnsureAvailable(ctx context.Context) error {
	atomic.AddInt32(&f.calls, 1)
	if f.fn != nil {
		return f.fn(ctx)
	}
	return nil
}

func (f *fakeWorkspaceSandboxEnsurer) CallCount() int {
	return int(atomic.LoadInt32(&f.calls))
}

func setupWorkspaceFixture(t *testing.T) (*domain.Run, *domain.Task, *domain.AgentProfile) {
	t.Helper()
	task := fixtures.NewTask(t)
	task.ProjectRoot = t.TempDir()
	task.ScopePath = "."
	run := fixtures.NewRun(t, task.ID, uuid.Nil)
	run.RunMode = domain.RunModeSandboxed
	run.SandboxConfig = fixtures.NewSandboxConfig(nil)
	profile := fixtures.NewAgentProfile(t)
	return run, task, profile
}

func TestCreateSandboxWorkspace_EnsuresWorkspaceSandboxWhenUnavailable(t *testing.T) {
	run, task, profile := setupWorkspaceFixture(t)
	provider := mocks.NewFakeSandboxProvider()
	available := atomic.Bool{}
	provider.IsAvailableFunc = func(context.Context) (bool, string) {
		if available.Load() {
			return true, "ok"
		}
		return false, "connection refused"
	}
	ensurer := &fakeWorkspaceSandboxEnsurer{fn: func(context.Context) error {
		available.Store(true)
		return nil
	}}

	out, err := CreateSandboxWorkspace(context.Background(), SetupWorkspaceInput{
		Deps:    Deps{Levers: config.DefaultLevers(), WorkspaceSandbox: ensurer},
		Run:     run,
		Task:    task,
		Profile: profile,
		Sandbox: provider,
	})
	if err != nil {
		t.Fatalf("CreateSandboxWorkspace returned error: %v", err)
	}
	if out.SandboxID == nil || out.WorkDir == "" {
		t.Fatalf("expected sandbox output, got %+v", out)
	}
	if ensurer.CallCount() != 1 {
		t.Fatalf("expected one ensure call, got %d", ensurer.CallCount())
	}
}

func TestCreateSandboxWorkspace_RetriesCreateWithSameIdempotencyKey(t *testing.T) {
	run, task, profile := setupWorkspaceFixture(t)
	provider := mocks.NewFakeSandboxProvider()
	var createCalls int32
	provider.CreateFunc = func(ctx context.Context, req sandbox.CreateRequest) (*sandbox.Sandbox, error) {
		if atomic.AddInt32(&createCalls, 1) == 1 {
			return nil, &domain.SandboxError{
				Operation:   "create",
				Cause:       errors.New("dial tcp 127.0.0.1:15120: connect: connection refused"),
				IsTransient: true,
				CanRetry:    true,
			}
		}
		return mocks.NewFakeSandboxProvider().Create(ctx, req)
	}
	levers := config.DefaultLevers()
	levers.Sandbox.OperationMaxAttempts = 2
	levers.Sandbox.OperationInitialBackoff = time.Millisecond
	levers.Sandbox.OperationMaxBackoff = time.Millisecond

	_, err := CreateSandboxWorkspace(context.Background(), SetupWorkspaceInput{
		Deps:    Deps{Levers: levers, WorkspaceSandbox: &fakeWorkspaceSandboxEnsurer{}},
		Run:     run,
		Task:    task,
		Profile: profile,
		Sandbox: provider,
	})
	if err != nil {
		t.Fatalf("CreateSandboxWorkspace returned error: %v", err)
	}

	reqs := provider.CreateRequests()
	if len(reqs) != 2 {
		t.Fatalf("expected two create attempts, got %d", len(reqs))
	}
	if reqs[0].IdempotencyKey == "" || reqs[0].IdempotencyKey != reqs[1].IdempotencyKey {
		t.Fatalf("expected stable idempotency key across retries, got %q and %q", reqs[0].IdempotencyKey, reqs[1].IdempotencyKey)
	}
}

func TestSetupWorkspace_ExistingSandboxDoesNotEnsureDependency(t *testing.T) {
	run, task, profile := setupWorkspaceFixture(t)
	provider := mocks.NewFakeSandboxProvider()
	provider.IsAvailableFunc = func(context.Context) (bool, string) {
		return false, "should not be checked"
	}
	ensurer := &fakeWorkspaceSandboxEnsurer{}
	sandboxID := uuid.New()

	out, err := SetupWorkspace(context.Background(), SetupWorkspaceInput{
		Deps:              Deps{Levers: config.DefaultLevers(), WorkspaceSandbox: ensurer},
		Run:               run,
		Task:              task,
		Profile:           profile,
		Sandbox:           provider,
		ExistingSandboxID: &sandboxID,
		ExistingWorkDir:   "/tmp/existing",
	})
	if err != nil {
		t.Fatalf("SetupWorkspace returned error: %v", err)
	}
	if out.WorkDir != "/tmp/existing" {
		t.Fatalf("expected existing workdir, got %q", out.WorkDir)
	}
	if ensurer.CallCount() != 0 {
		t.Fatalf("expected no ensure call for existing sandbox, got %d", ensurer.CallCount())
	}
}

func TestSetupWorkspace_ExistingSandboxRestoresWorkspaceAfterProviderRecovery(t *testing.T) {
	run, task, profile := setupWorkspaceFixture(t)
	provider := mocks.NewFakeSandboxProvider()
	available := atomic.Bool{}
	provider.IsAvailableFunc = func(context.Context) (bool, string) {
		if available.Load() {
			return true, "ok"
		}
		return false, "connection refused"
	}
	var resolvedID uuid.UUID
	provider.GetWorkspacePathFn = func(_ context.Context, id uuid.UUID) (string, error) {
		resolvedID = id
		return "/workspace/recovered", nil
	}
	ensurer := &fakeWorkspaceSandboxEnsurer{fn: func(context.Context) error {
		available.Store(true)
		return nil
	}}
	sandboxID := uuid.New()

	out, err := SetupWorkspace(context.Background(), SetupWorkspaceInput{
		Deps:              Deps{Levers: config.DefaultLevers(), WorkspaceSandbox: ensurer},
		Run:               run,
		Task:              task,
		Profile:           profile,
		Sandbox:           provider,
		ExistingSandboxID: &sandboxID,
	})
	if err != nil {
		t.Fatalf("SetupWorkspace returned error: %v", err)
	}
	if out.SandboxID == nil || *out.SandboxID != sandboxID || resolvedID != sandboxID {
		t.Fatalf("setup changed existing sandbox identity: output=%+v resolved=%s want=%s", out, resolvedID, sandboxID)
	}
	if out.WorkDir != "/workspace/recovered" {
		t.Fatalf("workdir = %q, want recovered path", out.WorkDir)
	}
	if ensurer.CallCount() != 1 {
		t.Fatalf("lifecycle ensure calls = %d, want 1", ensurer.CallCount())
	}
	if len(provider.CreateRequests()) != 0 {
		t.Fatalf("recovery allocated a replacement sandbox: %+v", provider.CreateRequests())
	}
}

func TestSetupWorkspace_ExistingSandboxRecoveryFailureRefusesFallback(t *testing.T) {
	run, task, profile := setupWorkspaceFixture(t)
	provider := mocks.NewFakeSandboxProvider()
	provider.IsAvailableFunc = func(context.Context) (bool, string) {
		return false, "connection refused"
	}
	pathCalls := 0
	provider.GetWorkspacePathFn = func(context.Context, uuid.UUID) (string, error) {
		pathCalls++
		return "", errors.New("must not resolve while unavailable")
	}
	recoveryErr := errors.New("lifecycle ensure refused")
	ensurer := &fakeWorkspaceSandboxEnsurer{fn: func(context.Context) error { return recoveryErr }}
	sandboxID := uuid.New()

	out, err := SetupWorkspace(context.Background(), SetupWorkspaceInput{
		Deps:              Deps{Levers: config.DefaultLevers(), WorkspaceSandbox: ensurer},
		Run:               run,
		Task:              task,
		Profile:           profile,
		Sandbox:           provider,
		ExistingSandboxID: &sandboxID,
	})
	var sandboxErr *domain.SandboxError
	if !errors.As(err, &sandboxErr) || !errors.Is(err, recoveryErr) {
		t.Fatalf("expected retryable sandbox recovery error, got output=%+v err=%v", out, err)
	}
	if sandboxErr.SandboxID == nil || *sandboxErr.SandboxID != sandboxID || !sandboxErr.Retryable() {
		t.Fatalf("error lost existing sandbox identity or retryability: %+v", sandboxErr)
	}
	if out.SandboxID != nil || out.WorkDir != "" || run.SandboxID != nil {
		t.Fatalf("failed recovery admitted workspace or mutated run: output=%+v run=%+v", out, run)
	}
	if ensurer.CallCount() != 1 || pathCalls != 0 || len(provider.CreateRequests()) != 0 {
		t.Fatalf("recovery failure attempted effects: ensure=%d path=%d create=%d", ensurer.CallCount(), pathCalls, len(provider.CreateRequests()))
	}
}

func TestSetupWorkspace_ExistingSandboxRemainsUnavailableAfterEnsure(t *testing.T) {
	run, task, profile := setupWorkspaceFixture(t)
	provider := mocks.NewFakeSandboxProvider()
	provider.IsAvailableFunc = func(context.Context) (bool, string) {
		return false, "still unavailable"
	}
	pathCalls := 0
	provider.GetWorkspacePathFn = func(context.Context, uuid.UUID) (string, error) {
		pathCalls++
		return "/workspace/must-not-open", nil
	}
	ensurer := &fakeWorkspaceSandboxEnsurer{}
	sandboxID := uuid.New()

	out, err := SetupWorkspace(context.Background(), SetupWorkspaceInput{
		Deps:              Deps{Levers: config.DefaultLevers(), WorkspaceSandbox: ensurer},
		Run:               run,
		Task:              task,
		Profile:           profile,
		Sandbox:           provider,
		ExistingSandboxID: &sandboxID,
	})
	var sandboxErr *domain.SandboxError
	if !errors.As(err, &sandboxErr) || sandboxErr.SandboxID == nil || *sandboxErr.SandboxID != sandboxID {
		t.Fatalf("expected existing sandbox unavailable error, got output=%+v err=%v", out, err)
	}
	if ensurer.CallCount() != 1 || pathCalls != 0 || len(provider.CreateRequests()) != 0 {
		t.Fatalf("still-unavailable provider crossed workspace boundary: ensure=%d path=%d create=%d", ensurer.CallCount(), pathCalls, len(provider.CreateRequests()))
	}
	if out.SandboxID != nil || out.WorkDir != "" {
		t.Fatalf("still-unavailable provider returned workspace: %+v", out)
	}
}

func TestSetupWorkspace_ExistingSandboxWorkspaceLookupFailureIsRetryable(t *testing.T) {
	run, task, profile := setupWorkspaceFixture(t)
	provider := mocks.NewFakeSandboxProvider()
	lookupErr := errors.New("workspace path endpoint temporarily unavailable")
	provider.GetWorkspacePathFn = func(_ context.Context, id uuid.UUID) (string, error) {
		return "", lookupErr
	}
	sandboxID := uuid.New()

	out, err := SetupWorkspace(context.Background(), SetupWorkspaceInput{
		Deps:              Deps{Levers: config.DefaultLevers()},
		Run:               run,
		Task:              task,
		Profile:           profile,
		Sandbox:           provider,
		ExistingSandboxID: &sandboxID,
	})
	var sandboxErr *domain.SandboxError
	if !errors.As(err, &sandboxErr) || !errors.Is(err, lookupErr) {
		t.Fatalf("expected typed workspace lookup failure, got output=%+v err=%v", out, err)
	}
	if sandboxErr.SandboxID == nil || *sandboxErr.SandboxID != sandboxID || !sandboxErr.Retryable() {
		t.Fatalf("lookup error lost existing sandbox identity or retryability: %+v", sandboxErr)
	}
	if out.SandboxID != nil || out.WorkDir != "" || len(provider.CreateRequests()) != 0 {
		t.Fatalf("lookup failure changed workspace or allocated replacement: output=%+v creates=%d", out, len(provider.CreateRequests()))
	}
}

type updateCountingRunRepo struct {
	repository.RunRepository
	calls int
}

func (r *updateCountingRunRepo) Update(context.Context, *domain.Run) error {
	r.calls++
	return nil
}

func TestCreateSandboxWorkspace_RejectsNonActiveCreateBeforeBindingRun(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    sandbox.SandboxStatus
		nilResult bool
	}{
		{name: "nil create result", nilResult: true},
		{name: "deleted idempotent tombstone", status: sandbox.SandboxStatusDeleted},
		{name: "checkpointed resource", status: sandbox.SandboxStatusCheckpointed},
		{name: "empty status"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run, task, profile := setupWorkspaceFixture(t)
			oldID := uuid.New()
			run.SandboxID = &oldID
			workspacePathCalls := 0
			provider := mocks.NewFakeSandboxProvider()
			provider.CreateFunc = func(context.Context, sandbox.CreateRequest) (*sandbox.Sandbox, error) {
				if tc.nilResult {
					return nil, nil
				}
				return &sandbox.Sandbox{ID: uuid.New(), Status: tc.status}, nil
			}
			provider.GetWorkspacePathFn = func(context.Context, uuid.UUID) (string, error) {
				workspacePathCalls++
				return "/tmp/unusable", nil
			}

			repo := &updateCountingRunRepo{}
			_, err := CreateSandboxWorkspace(context.Background(), SetupWorkspaceInput{
				Deps: Deps{Levers: config.DefaultLevers(), Runs: repo}, Run: run, Task: task, Profile: profile, Sandbox: provider,
			})
			if err == nil {
				t.Fatal("expected non-active sandbox creation result to fail")
			}
			if run.SandboxID == nil || *run.SandboxID != oldID {
				t.Fatalf("failed create changed Run.SandboxID to %v", run.SandboxID)
			}
			if repo.calls != 0 {
				t.Fatalf("run persistence called %d times for unusable sandbox", repo.calls)
			}
			if workspacePathCalls != 0 {
				t.Fatalf("workspace path was queried %d times for unusable sandbox", workspacePathCalls)
			}
		})
	}
}

func TestCreateSandboxWorkspace_RefusesUnavailableDependencyWithoutHostFallback(t *testing.T) {
	run, task, profile := setupWorkspaceFixture(t)
	provider := mocks.NewFakeSandboxProvider()
	provider.IsAvailableFunc = func(context.Context) (bool, string) { return false, "sandbox unavailable" }
	createCalls := 0
	provider.CreateFunc = func(context.Context, sandbox.CreateRequest) (*sandbox.Sandbox, error) {
		createCalls++
		return nil, errors.New("must not allocate after failed ensure")
	}
	failure := errors.New("dependency recovery refused")
	ensurer := &fakeWorkspaceSandboxEnsurer{fn: func(context.Context) error { return failure }}
	out, err := CreateSandboxWorkspace(context.Background(), SetupWorkspaceInput{
		Deps: Deps{Levers: config.DefaultLevers(), WorkspaceSandbox: ensurer}, Run: run, Task: task, Profile: profile, Sandbox: provider,
	})
	if err == nil || !errors.Is(err, failure) {
		t.Fatalf("failed dependency recovery did not refuse sandboxed setup: output=%+v error=%v", out, err)
	}
	if createCalls != 0 || run.SandboxID != nil || out.WorkDir != "" {
		t.Fatalf("failed recovery admitted execution or bound an unverified workspace: createCalls=%d run=%+v output=%+v", createCalls, run, out)
	}
	if ensurer.CallCount() != 1 {
		t.Fatalf("expected one bounded recovery attempt, got %d", ensurer.CallCount())
	}
}
