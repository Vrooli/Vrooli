package orchestrator

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestSandboxAffectedScenariosReturnsSortedRegistryMatches(t *testing.T) {
	home := t.TempDir()
	mergedPath := filepath.Join(home, "merged")
	ctx := context.Background()
	now := time.Now().Add(-time.Minute).UTC()
	writeSandboxInstance(t, home, "beta", filepath.Join(mergedPath, "scenarios", "beta"), now, scenarioruntime.StatusRunning)
	writeSandboxInstance(t, home, "alpha", filepath.Join(mergedPath, "scenarios", "alpha"), now, scenarioruntime.StatusRunning)
	writeSandboxInstance(t, home, "gamma", filepath.Join(home, "repo", "scenarios", "gamma"), now, scenarioruntime.StatusRunning)

	svc := New(t.TempDir(), home, io.Discard, io.Discard)
	affected, err := svc.SandboxAffectedScenarios(ctx, mergedPath)
	if err != nil {
		t.Fatalf("SandboxAffectedScenarios: %v", err)
	}
	if got := strings.Join(affected, ","); got != "alpha,beta" {
		t.Fatalf("affected = %q, want alpha,beta", got)
	}
}

func TestSandboxAffectedScenariosIgnoresStoppedAndLegacyOnlyScenarios(t *testing.T) {
	home := t.TempDir()
	mergedPath := filepath.Join(home, "merged")
	ctx := context.Background()
	now := time.Now().Add(-time.Minute).UTC()
	writeSandboxInstance(t, home, "stopped-one", filepath.Join(mergedPath, "scenarios", "stopped-one"), now, scenarioruntime.StatusStopped)
	writeSandboxInstance(t, home, "running-one", filepath.Join(mergedPath, "scenarios", "running-one"), now, scenarioruntime.StatusRunning)

	svc := New(t.TempDir(), home, io.Discard, io.Discard)
	affected, err := svc.SandboxAffectedScenarios(ctx, mergedPath)
	if err != nil {
		t.Fatalf("SandboxAffectedScenarios: %v", err)
	}
	if got := strings.Join(affected, ","); got != "running-one" {
		t.Fatalf("affected = %q, want running-one (stopped scenarios ignored)", got)
	}
}

func writeSandboxInstance(t *testing.T, home, name, workingDir string, startedAt time.Time, status string) {
	t.Helper()
	ctx := context.Background()
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer store.Close()
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		InstanceID: "inst-" + name,
		Scenario:   name,
		Status:     scenarioruntime.StatusStarting,
		Phase:      "develop",
		WorkingDir: workingDir,
		StartedAt:  startedAt,
		OwnerKind:  scenarioruntime.OwnerKindLifecycle,
	}, scenarioruntime.DefaultHeartbeatTTL)
	if err != nil {
		t.Fatalf("CreateLease %s: %v", name, err)
	}
	if status != scenarioruntime.StatusStarting {
		if _, err := store.UpdateInstanceStatus(ctx, instance.InstanceID, instance.Generation, status, instance.Phase); err != nil {
			t.Fatalf("UpdateInstanceStatus %s: %v", name, err)
		}
	}
}
