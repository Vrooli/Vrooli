package mocks

import (
	"context"
	"testing"

	adapterrunner "agent-manager/internal/adapters/runner"

	"github.com/google/uuid"
)

func TestFakeLauncherRecordsCalls(t *testing.T) {
	launcher := NewFakeLauncher("host")
	req := adapterrunner.LaunchRequest{Command: "agent-cli"}

	_, err := launcher.Launch(context.Background(), req)
	if err != nil {
		t.Fatalf("Launch returned error: %v", err)
	}

	calls := launcher.LaunchCalls()
	if len(calls) != 1 || calls[0].Command != "agent-cli" {
		t.Fatalf("expected recorded launch request, got %#v", calls)
	}
}

func TestFakeSandboxLauncherFactoryRecordsIDs(t *testing.T) {
	launcher := NewFakeLauncher("sandbox")
	factory := NewFakeSandboxLauncherFactory(launcher)
	id := uuid.New()

	got := factory.LauncherFor(id)
	if got != launcher {
		t.Fatal("expected configured launcher")
	}
	ids := factory.CalledIDs()
	if len(ids) != 1 || ids[0] != id {
		t.Fatalf("expected recorded sandbox ID %s, got %v", id, ids)
	}
}
