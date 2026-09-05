package heartbeat

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunRegistry_RegisterAndListActive(t *testing.T) {
	reg := NewRunRegistry(t.TempDir())
	now := time.Now().UTC()

	reg.Register("team-1", "agent-1", "run-1", now, func() {})

	active := reg.ListActive()
	if len(active) != 1 {
		t.Fatalf("expected 1 active run, got %d", len(active))
	}
	if active[0].RunID != "run-1" {
		t.Errorf("expected run ID run-1, got %s", active[0].RunID)
	}
}

func TestRunRegistry_Unregister(t *testing.T) {
	reg := NewRunRegistry(t.TempDir())
	reg.Register("team-1", "agent-1", "run-1", time.Now(), func() {})

	reg.Unregister("team-1", "agent-1")

	if count := reg.Count(); count != 0 {
		t.Fatalf("expected 0 after unregister, got %d", count)
	}
}

func TestRunRegistry_UnregisterNonexistent(t *testing.T) {
	reg := NewRunRegistry(t.TempDir())
	// Should not panic
	reg.Unregister("team-1", "agent-1")
}

func TestRunRegistry_GetActiveRun(t *testing.T) {
	reg := NewRunRegistry(t.TempDir())
	reg.Register("team-1", "agent-1", "run-1", time.Now(), func() {})

	run, ok := reg.GetActiveRun("team-1", "agent-1")
	if !ok {
		t.Fatal("expected to find active run")
	}
	if run.RunID != "run-1" {
		t.Errorf("expected run-1, got %s", run.RunID)
	}

	_, ok = reg.GetActiveRun("team-1", "agent-2")
	if ok {
		t.Fatal("expected not to find run for different agent")
	}
}

func TestRunRegistry_Count(t *testing.T) {
	reg := NewRunRegistry(t.TempDir())
	if reg.Count() != 0 {
		t.Fatal("expected 0 initially")
	}

	reg.Register("team-1", "agent-1", "run-1", time.Now(), func() {})
	reg.Register("team-1", "agent-2", "run-2", time.Now(), func() {})

	if reg.Count() != 2 {
		t.Fatalf("expected 2, got %d", reg.Count())
	}
}

func TestRunRegistry_RegisterOverwritesSameKey(t *testing.T) {
	reg := NewRunRegistry(t.TempDir())
	reg.Register("team-1", "agent-1", "run-1", time.Now(), func() {})
	reg.Register("team-1", "agent-1", "run-2", time.Now(), func() {})

	if reg.Count() != 1 {
		t.Fatalf("expected 1 (overwritten), got %d", reg.Count())
	}
	run, _ := reg.GetActiveRun("team-1", "agent-1")
	if run.RunID != "run-2" {
		t.Errorf("expected overwritten run-2, got %s", run.RunID)
	}
}

func TestRunRegistry_PersistAndRecover(t *testing.T) {
	dir := t.TempDir()
	reg := NewRunRegistry(dir)
	now := time.Now().UTC()

	reg.Register("team-1", "agent-1", "run-1", now, func() {})
	reg.Register("team-2", "agent-2", "run-2", now, func() {})

	// Verify file was written
	filePath := filepath.Join(dir, "heartbeat-active-runs.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("expected persist file: %v", err)
	}

	var persisted []persistedRun
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("unmarshal persisted: %v", err)
	}
	if len(persisted) != 2 {
		t.Fatalf("expected 2 persisted runs, got %d", len(persisted))
	}

	// Recover into a fresh registry, with a mock client that says runs are still active
	reg2 := NewRunRegistry(dir)
	mockClient := newMockAgentClient().
		WithGetRunResponse("run-1", &Run{ID: "run-1", Status: "RUN_STATUS_RUNNING"}).
		WithGetRunResponse("run-2", &Run{ID: "run-2", Status: "RUN_STATUS_RUNNING"})

	reg2.Recover(context.Background(), mockClient)

	if reg2.Count() != 2 {
		t.Fatalf("expected 2 recovered, got %d", reg2.Count())
	}
}

func TestRunRegistry_RecoverFiltersTerminalRuns(t *testing.T) {
	dir := t.TempDir()
	reg := NewRunRegistry(dir)

	reg.Register("team-1", "agent-1", "run-1", time.Now(), func() {})
	reg.Register("team-1", "agent-2", "run-2", time.Now(), func() {})

	reg2 := NewRunRegistry(dir)
	mockClient := newMockAgentClient().
		WithGetRunResponse("run-1", &Run{ID: "run-1", Status: "RUN_STATUS_COMPLETE"}).
		WithGetRunResponse("run-2", &Run{ID: "run-2", Status: "RUN_STATUS_RUNNING"})

	reg2.Recover(context.Background(), mockClient)

	if reg2.Count() != 1 {
		t.Fatalf("expected 1 (terminal filtered), got %d", reg2.Count())
	}
	_, ok := reg2.GetActiveRun("team-1", "agent-2")
	if !ok {
		t.Fatal("expected agent-2 run to be recovered")
	}
}

func TestRunRegistry_RecoverNoFile(t *testing.T) {
	reg := NewRunRegistry(t.TempDir())
	mockClient := newMockAgentClient()

	// Should not panic or error
	reg.Recover(context.Background(), mockClient)
	if reg.Count() != 0 {
		t.Fatalf("expected 0 with no persist file, got %d", reg.Count())
	}
}

func TestRunRegistry_RecoverCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "heartbeat-active-runs.json")
	if err := os.WriteFile(filePath, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := NewRunRegistry(dir)
	mockClient := newMockAgentClient()

	// Should handle gracefully
	reg.Recover(context.Background(), mockClient)
	if reg.Count() != 0 {
		t.Fatalf("expected 0 with corrupted file, got %d", reg.Count())
	}
}

func TestRunRegistry_PersistAfterUnregister(t *testing.T) {
	dir := t.TempDir()
	reg := NewRunRegistry(dir)

	reg.Register("team-1", "agent-1", "run-1", time.Now(), func() {})
	reg.Unregister("team-1", "agent-1")

	// Recover into fresh registry
	reg2 := NewRunRegistry(dir)
	mockClient := newMockAgentClient()
	reg2.Recover(context.Background(), mockClient)

	if reg2.Count() != 0 {
		t.Fatalf("expected 0 after unregister + recover, got %d", reg2.Count())
	}
}
