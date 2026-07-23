package mocks

import (
	"context"
	"errors"
	"testing"

	"agent-manager/internal/adapters/sandbox"

	"github.com/google/uuid"
)

func TestFakeSandboxProviderDefaults(t *testing.T) {
	provider := NewFakeSandboxProvider()
	sandboxID := uuid.New()

	sbx, err := provider.Create(context.Background(), sandbox.CreateRequest{
		ScopePath:   "src",
		ProjectRoot: "/repo",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sbx.Status != sandbox.SandboxStatusActive {
		t.Fatalf("expected active sandbox, got %q", sbx.Status)
	}

	if _, err := provider.GetWorkspacePath(context.Background(), sandboxID); err != nil {
		t.Fatalf("GetWorkspacePath: %v", err)
	}
	if provider.GetWorkspacePathCallCount() != 1 {
		t.Fatalf("expected 1 workspace path request, got %d", provider.GetWorkspacePathCallCount())
	}
	ids := provider.GetWorkspacePathIDs()
	if len(ids) != 1 || ids[0] != sandboxID {
		t.Fatalf("workspace path IDs = %v, want [%v]", ids, sandboxID)
	}

	result, err := provider.ApplyAtRunEnd(context.Background(), sandbox.ApplyAtRunEndRequest{
		SandboxID: uuid.New(),
		RunID:     uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("ApplyAtRunEnd: %v", err)
	}
	if !result.Success {
		t.Fatal("expected default ApplyAtRunEnd success")
	}
	if provider.ApplyAtRunEndCallCount() != 1 {
		t.Fatalf("expected 1 apply request, got %d", provider.ApplyAtRunEndCallCount())
	}
}

func TestFakeSandboxProviderCoversDefaultLifecycleAndRecordedRequests(t *testing.T) {
	provider := NewFakeSandboxProvider()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	id := uuid.New()
	if err := provider.Delete(ctx, id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if provider.DeleteCallCount() != 1 || len(provider.DeleteContextErrs()) != 1 || !errors.Is(provider.DeleteContextErrs()[0], context.Canceled) {
		t.Fatalf("delete tracking: count=%d errors=%v", provider.DeleteCallCount(), provider.DeleteContextErrs())
	}
	if err := provider.Stop(context.Background(), id); err != nil || provider.StopCallCount() != 1 {
		t.Fatalf("Stop: err=%v calls=%d", err, provider.StopCallCount())
	}
	if err := provider.Start(context.Background(), id); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if resumed, err := provider.Resume(context.Background(), id); err != nil || resumed.ID != id || resumed.Status != sandbox.SandboxStatusActive {
		t.Fatalf("Resume: sandbox=%+v err=%v", resumed, err)
	}
	if diff, err := provider.GetDiff(context.Background(), id); err != nil || diff == nil {
		t.Fatalf("GetDiff: diff=%+v err=%v", diff, err)
	}
	if approved, err := provider.Approve(context.Background(), sandbox.ApproveRequest{}); err != nil || !approved.Success {
		t.Fatalf("Approve: result=%+v err=%v", approved, err)
	}
	if err := provider.Reject(context.Background(), id, "operator"); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if approved, err := provider.PartialApprove(context.Background(), sandbox.PartialApproveRequest{}); err != nil || !approved.Success {
		t.Fatalf("PartialApprove: result=%+v err=%v", approved, err)
	}
	if available, reason := provider.IsAvailable(context.Background()); !available || reason != "" {
		t.Fatalf("availability=%v %q", available, reason)
	}
	if validated, err := provider.ValidatePath(context.Background(), "src", "/repo"); err != nil || !validated.Valid || validated.Path != "src" {
		t.Fatalf("ValidatePath: result=%+v err=%v", validated, err)
	}
	if result, err := provider.ExecProcess(context.Background(), sandbox.ExecProcessRequest{}); err != nil || result.ExitCode != 0 {
		t.Fatalf("ExecProcess: result=%+v err=%v", result, err)
	}
	checkpoint, err := provider.TurnCheckpoint(context.Background(), sandbox.TurnCheckpointRequest{SandboxID: id})
	if err != nil || !checkpoint.Success || checkpoint.SandboxID != id || provider.TurnCheckpointCallCount() != 1 {
		t.Fatalf("TurnCheckpoint: result=%+v err=%v", checkpoint, err)
	}
}

func TestFakeSandboxProviderHonorsConfiguredFailuresAndCallbacks(t *testing.T) {
	provider := NewFakeSandboxProvider()
	injected := errors.New("injected")
	provider.DeleteErr, provider.StopErr, provider.ApplyAtRunEndErr, provider.TurnCheckpointErr = injected, injected, injected, injected
	if err := provider.Delete(context.Background(), uuid.New()); !errors.Is(err, injected) {
		t.Fatalf("Delete error=%v", err)
	}
	if err := provider.Stop(context.Background(), uuid.New()); !errors.Is(err, injected) {
		t.Fatalf("Stop error=%v", err)
	}
	if _, err := provider.ApplyAtRunEnd(context.Background(), sandbox.ApplyAtRunEndRequest{}); !errors.Is(err, injected) {
		t.Fatalf("ApplyAtRunEnd error=%v", err)
	}
	if _, err := provider.TurnCheckpoint(context.Background(), sandbox.TurnCheckpointRequest{}); !errors.Is(err, injected) {
		t.Fatalf("TurnCheckpoint error=%v", err)
	}
	called := false
	provider.CreateFunc = func(_ context.Context, req sandbox.CreateRequest) (*sandbox.Sandbox, error) {
		called = req.Name == "custom"
		return nil, injected
	}
	if _, err := provider.Create(context.Background(), sandbox.CreateRequest{Name: "custom"}); !errors.Is(err, injected) || !called {
		t.Fatalf("Create callback error=%v called=%v", err, called)
	}
}
