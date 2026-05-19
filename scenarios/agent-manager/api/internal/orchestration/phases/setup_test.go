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
	"agent-manager/internal/testutil/fixtures"
	"agent-manager/internal/testutil/mocks"

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
