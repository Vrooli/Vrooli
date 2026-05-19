package requirements

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"test-genie/internal/orchestrator/phases"
)

func TestBuildPhaseStatusPayload(t *testing.T) {
	defs := []phases.Definition{
		{Name: phases.Structure},
		{Name: phases.Unit, Optional: true},
	}
	results := []phases.ExecutionResult{
		{Name: "Structure", Status: "PASSED"},
		{Name: "unit", Status: "failed"},
	}

	payload := buildPhaseStatusPayload(defs, results)
	if len(payload) != 2 {
		t.Fatalf("expected two payload entries, got %d", len(payload))
	}
	if payload[0].Phase != "structure" || payload[0].Status != "passed" || !payload[0].Recorded {
		t.Fatalf("unexpected first payload entry: %#v", payload[0])
	}
	if payload[1].Phase != "unit" || payload[1].Status != "failed" || !payload[1].Recorded || !payload[1].Optional {
		t.Fatalf("unexpected second payload entry: %#v", payload[1])
	}
}

func TestNewNodeSyncerReturnsNilWithoutScript(t *testing.T) {
	if syncer := NewNodeSyncer(t.TempDir()); syncer != nil {
		t.Fatalf("expected nil syncer when script missing")
	}
}

func TestNodeSyncerSyncSkipsWithoutScenarioOrRequirements(t *testing.T) {
	syncer := &nodeRequirementsSyncer{projectRoot: t.TempDir()}
	if err := syncer.Sync(context.Background(), SyncInput{}); err != nil {
		t.Fatalf("expected nil error when scenario missing, got %v", err)
	}
	input := SyncInput{
		ScenarioName: "demo",
		ScenarioDir:  filepath.Join(t.TempDir(), "scenarios", "demo"),
	}
	if err := syncer.Sync(context.Background(), input); err != nil {
		t.Fatalf("expected nil error when requirements dir missing, got %v", err)
	}
}

func TestNodeSyncerSyncRunsCommand(t *testing.T) {
	syncer, root, scenarioDir := setupNodeSyncer(t)
	restoreLookup := phases.OverrideCommandLookup(func(name string) (string, error) {
		if name == "node" {
			return "/usr/bin/node", nil
		}
		return "", fmt.Errorf("unexpected command %s", name)
	})
	defer restoreLookup()

	var capturedArgs []string
	var capturedCmd *exec.Cmd
	restoreExec := overrideExecCommand(func(ctx context.Context, name string, args ...string) *exec.Cmd {
		capturedArgs = append([]string{name}, args...)
		cmd := exec.CommandContext(ctx, "bash", "-c", "exit 0")
		capturedCmd = cmd
		return cmd
	})
	defer restoreExec()

	input := SyncInput{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		PhaseDefinitions: []phases.Definition{
			{Name: phases.Structure},
		},
		PhaseResults: []phases.ExecutionResult{
			{Name: "Structure", Status: "PASSED"},
		},
		CommandHistory: []string{"suite demo"},
	}

	if err := syncer.Sync(context.Background(), input); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	expectedArgs := []string{"node", filepath.Join(root, "scripts", "requirements", "report.js"), "--scenario", "demo", "--mode", "sync"}
	if strings.Join(capturedArgs, " ") != strings.Join(expectedArgs, " ") {
		t.Fatalf("unexpected args: %v", capturedArgs)
	}
	if capturedCmd == nil || capturedCmd.Dir != root {
		t.Fatalf("expected command dir %s, got %#v", root, capturedCmd)
	}
	env := strings.Join(capturedCmd.Env, "\n")
	if !strings.Contains(env, "REQUIREMENTS_SYNC_PHASE_STATUS=") {
		t.Fatalf("phase status env missing: %s", env)
	}
	if !strings.Contains(env, "REQUIREMENTS_SYNC_TEST_COMMANDS=") {
		t.Fatalf("command history env missing: %s", env)
	}
}

func TestNodeSyncerSyncErrorsWhenCommandUnavailable(t *testing.T) {
	syncer, _, scenarioDir := setupNodeSyncer(t)
	restoreLookup := phases.OverrideCommandLookup(func(string) (string, error) {
		return "", fmt.Errorf("not found")
	})
	defer restoreLookup()

	input := SyncInput{
		ScenarioName:     "demo",
		ScenarioDir:      scenarioDir,
		PhaseDefinitions: []phases.Definition{{Name: phases.Unit}},
		PhaseResults:     []phases.ExecutionResult{{Name: "unit", Status: "passed"}},
	}
	if err := syncer.Sync(context.Background(), input); err == nil || !strings.Contains(err.Error(), "node command not available") {
		t.Fatalf("expected command availability error, got %v", err)
	}
}

func TestNodeSyncerSyncRequiresPhaseMetadata(t *testing.T) {
	syncer, _, scenarioDir := setupNodeSyncer(t)
	restoreLookup := phases.OverrideCommandLookup(func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	})
	defer restoreLookup()

	input := SyncInput{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
	}
	if err := syncer.Sync(context.Background(), input); err == nil || !strings.Contains(err.Error(), "phase execution metadata missing") {
		t.Fatalf("expected metadata error, got %v", err)
	}
}

func TestNodeSyncerSyncPropagatesCommandErrors(t *testing.T) {
	syncer, _, scenarioDir := setupNodeSyncer(t)
	restoreLookup := phases.OverrideCommandLookup(func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	})
	defer restoreLookup()

	restoreExec := overrideExecCommand(func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "bash", "-c", "echo failure >&2; exit 1")
	})
	defer restoreExec()

	input := SyncInput{
		ScenarioName:     "demo",
		ScenarioDir:      scenarioDir,
		PhaseDefinitions: []phases.Definition{{Name: phases.Unit}},
		PhaseResults:     []phases.ExecutionResult{{Name: "unit", Status: "failed"}},
		CommandHistory:   []string{"suite demo"},
	}
	if err := syncer.Sync(context.Background(), input); err == nil || !strings.Contains(err.Error(), "requirements sync command failed") {
		t.Fatalf("expected command failure error, got %v", err)
	}
}

func setupNodeSyncer(t *testing.T) (*nodeRequirementsSyncer, string, string) {
	t.Helper()
	root := t.TempDir()
	scriptsDir := filepath.Join(root, "scripts", "requirements")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("create scripts dir: %v", err)
	}
	scriptPath := filepath.Join(scriptsDir, "report.js")
	if err := os.WriteFile(scriptPath, []byte("module.exports = {}"), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}
	scenarioDir := filepath.Join(root, "scenarios", "demo")
	if err := os.MkdirAll(filepath.Join(scenarioDir, "requirements"), 0o755); err != nil {
		t.Fatalf("create requirements dir: %v", err)
	}
	syncer := NewNodeSyncer(root)
	nodeSyncer, ok := syncer.(*nodeRequirementsSyncer)
	if !ok || nodeSyncer == nil {
		t.Fatalf("expected nodeRequirementsSyncer instance, got %#v", syncer)
	}
	return nodeSyncer, root, scenarioDir
}

func overrideExecCommand(fn func(context.Context, string, ...string) *exec.Cmd) func() {
	prev := execCommandContext
	execCommandContext = fn
	return func() { execCommandContext = prev }
}
