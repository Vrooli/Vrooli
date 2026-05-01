package mocks

import (
	"context"
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
