package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
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

func TestRunRegistry_RegisterRecoveredPreservesOwnerProjection(t *testing.T) {
	dir := t.TempDir()
	reg := NewRunRegistry(dir)
	ownerStarted := time.Date(2026, 10, 14, 12, 30, 0, 0, time.UTC)
	reg.RegisterRecovered("team", "agent", "run", ownerStarted, ObligationNeedsReview, nil)

	active := reg.ListActive()
	if len(active) != 1 {
		t.Fatalf("active count = %d, want 1", len(active))
	}
	if active[0].StartedAt != ownerStarted || active[0].State != ObligationNeedsReview {
		t.Fatalf("owner projection not retained: %+v", active[0])
	}

	// An absent owner timestamp remains absent through persistence/recovery.
	reg.RegisterRecovered("team", "agent", "run", time.Time{}, ObligationNeedsReview, nil)
	reg2 := NewRunRegistry(dir)
	reg2.Recover(context.Background(), &mockAgentClient{})
	recovered, ok := reg2.GetActiveRun("team", "agent")
	if !ok {
		t.Fatal("run was not recovered")
	}
	if !recovered.StartedAt.IsZero() || recovered.State != ObligationNeedsReview {
		t.Fatalf("unknown start or owner state was changed during recovery: %+v", recovered)
	}
}

func TestRunRegistry_RegisterOwnerRunPreservesNeedsReviewState(t *testing.T) {
	reg := NewRunRegistry(t.TempDir())
	startedAt := time.Date(2026, 10, 17, 6, 0, 0, 0, time.UTC)
	reg.RegisterOwnerRun("team", "agent", "run", startedAt, nil, ObligationNeedsReview)
	active, ok := reg.GetActiveRun("team", "agent")
	if !ok || active.State != ObligationNeedsReview || active.StartedAt != startedAt {
		t.Fatalf("initial owner observation lost review state or original start: active=%+v ok=%v", active, ok)
	}
}

type runRegistryObserverRecorder struct {
	started int
}

func (o *runRegistryObserverRecorder) RunStarted(ActiveRun)                        { o.started++ }
func (o *runRegistryObserverRecorder) RunEnded(ActiveRun, time.Time, bool, string) {}

func TestRunRegistry_RegisterRecoveredDoesNotPublishNewStart(t *testing.T) {
	reg := NewRunRegistry(t.TempDir())
	observer := &runRegistryObserverRecorder{}
	reg.SetObserver(observer)
	reg.RegisterRecovered("team", "agent", "run", time.Time{}, ObligationNeedsReview, nil)
	if observer.started != 0 {
		t.Fatalf("recovery emitted %d new-start events", observer.started)
	}
	reg.Register("team", "agent-2", "run-2", time.Now(), nil)
	if observer.started != 1 {
		t.Fatalf("ordinary registration emitted %d start events, want 1", observer.started)
	}
}

func TestRunRegistry_RecoveryDoesNotTrustLegacyLocalStartAsOwnerStart(t *testing.T) {
	dir := t.TempDir()
	legacyLocalStart := time.Date(2026, 10, 17, 5, 59, 50, 0, time.UTC)
	legacy := NewRunRegistry(dir)
	legacy.Register("team", "agent", "run", legacyLocalStart, nil)

	recovered := NewRunRegistry(dir)
	client := newMockAgentClient()
	client.getRunErr = errors.New("owner unavailable")
	recovered.Recover(context.Background(), client)

	run, ok := recovered.GetActiveRun("team", "agent")
	if !ok || run.State != ObligationOwnerUnreachable {
		t.Fatalf("owner-unreachable obligation not retained: run=%+v found=%t", run, ok)
	}
	if !run.StartedAt.IsZero() {
		t.Fatalf("legacy local request timestamp was presented as the owner's start: %s", run.StartedAt)
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

func TestRunRegistry_RecoverRetainsOwnerUnreachable(t *testing.T) {
	dir := t.TempDir()
	reg := NewRunRegistry(dir)
	reg.Register("team-1", "agent-1", "run-1", time.Now(), func() {})

	// Owner unavailable during recovery: absence of evidence is not terminal.
	reg2 := NewRunRegistry(dir)
	client := newMockAgentClient()
	client.getRunErr = errors.New("connection refused")
	reg2.Recover(context.Background(), client)

	if reg2.Count() != 1 {
		t.Fatalf("expected owner-unreachable run retained, got %d", reg2.Count())
	}
	run, ok := reg2.GetActiveRun("team-1", "agent-1")
	if !ok {
		t.Fatal("expected retained run")
	}
	if run.State != ObligationOwnerUnreachable {
		t.Fatalf("expected owner_unreachable state, got %q", run.State)
	}
}

func TestRunRegistry_RecoverRetainsMissingRunAsUncertain(t *testing.T) {
	dir := t.TempDir()
	reg := NewRunRegistry(dir)
	reg.Register("team-1", "agent-1", "run-1", time.Now(), func() {})

	reg2 := NewRunRegistry(dir)
	client := newMockAgentClient().WithGetRunResponse("run-1", nil)
	reg2.Recover(context.Background(), client)

	run, ok := reg2.GetActiveRun("team-1", "agent-1")
	if !ok {
		t.Fatal("expected missing run retained as uncertain")
	}
	if run.State != ObligationDispatchUncertain {
		t.Fatalf("expected dispatch_uncertain state, got %q", run.State)
	}
}

func TestRunRegistry_RecoverClassifiesParkedAsPaused(t *testing.T) {
	dir := t.TempDir()
	reg := NewRunRegistry(dir)
	reg.Register("team-1", "agent-1", "run-1", time.Now(), func() {})

	reg2 := NewRunRegistry(dir)
	client := newMockAgentClient().
		WithGetRunResponse("run-1", &Run{ID: "run-1", Status: "RUN_STATUS_PARKED"})
	reg2.Recover(context.Background(), client)

	run, ok := reg2.GetActiveRun("team-1", "agent-1")
	if !ok {
		t.Fatal("expected parked run retained")
	}
	if run.State != ObligationPaused {
		t.Fatalf("expected paused state, got %q", run.State)
	}
}

func TestRunRegistry_RecoverClassifiesNeedsReview(t *testing.T) {
	dir := t.TempDir()
	reg := NewRunRegistry(dir)
	reg.Register("team-1", "agent-1", "run-1", time.Date(2026, 10, 17, 5, 59, 50, 0, time.UTC), func() {})

	reg2 := NewRunRegistry(dir)
	ownerStarted := time.Date(2026, 10, 14, 12, 30, 0, 0, time.UTC)
	client := newMockAgentClient().WithGetRunResponse("run-1", &Run{ID: "run-1", Status: "RUN_STATUS_NEEDS_REVIEW", StartedAt: ownerStarted.Format(time.RFC3339)})
	reg2.Recover(context.Background(), client)

	run, ok := reg2.GetActiveRun("team-1", "agent-1")
	if !ok {
		t.Fatal("expected needs-review run retained")
	}
	if run.State != ObligationNeedsReview {
		t.Fatalf("expected needs_review state, got %q", run.State)
	}
	if run.StartedAt != ownerStarted {
		t.Fatalf("recovery start=%s, want owner start=%s", run.StartedAt, ownerStarted)
	}
}
