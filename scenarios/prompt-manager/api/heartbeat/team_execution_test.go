package heartbeat

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEnqueueWhenIdle_StartsImmediately(t *testing.T) {
	exec := &captureExecutor{}
	ctx := newTeamExecutionContext("team-1", exec, t.TempDir())

	result, err := ctx.Enqueue(context.Background(), "agent-1", "profile-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "started" {
		t.Fatalf("expected status 'started', got %q", result.Status)
	}
	if result.Position != 0 {
		t.Fatalf("expected position 0, got %d", result.Position)
	}

	status := ctx.Status()
	if status.State != "active" {
		t.Fatalf("expected state 'active', got %q", status.State)
	}
	if status.Running == nil || *status.Running != "agent-1" {
		t.Fatalf("expected running agent-1, got %v", status.Running)
	}
}

func TestEnqueueWhenActive_QueuesSuccessfully(t *testing.T) {
	exec := &captureExecutor{}
	ctx := newTeamExecutionContext("team-1", exec, t.TempDir())

	// Start first agent
	if _, err := ctx.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue agent-1: %v", err)
	}

	// Queue second agent
	result, err := ctx.Enqueue(context.Background(), "agent-2", "profile-2")
	if err != nil {
		t.Fatalf("enqueue agent-2: %v", err)
	}
	if result.Status != "queued" {
		t.Fatalf("expected status 'queued', got %q", result.Status)
	}
	if result.Position != 1 {
		t.Fatalf("expected position 1, got %d", result.Position)
	}

	status := ctx.Status()
	if len(status.Queue) != 1 {
		t.Fatalf("expected 1 queued, got %d", len(status.Queue))
	}
	if status.Queue[0] != "agent-2" {
		t.Fatalf("expected agent-2 in queue, got %q", status.Queue[0])
	}
}

func TestEnqueueDuplicate_ReturnsError(t *testing.T) {
	exec := &captureExecutor{}
	ctx := newTeamExecutionContext("team-1", exec, t.TempDir())

	// Start first agent
	if _, err := ctx.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue agent-1: %v", err)
	}

	// Try to enqueue same agent again (already running)
	_, err := ctx.Enqueue(context.Background(), "agent-1", "profile-1")
	if err == nil {
		t.Fatal("expected error for duplicate enqueue")
	}
	if !IsMemberAlreadyQueued(err) {
		t.Fatalf("expected MemberAlreadyQueuedError, got %T: %v", err, err)
	}

	// Queue agent-2, then try to queue agent-2 again
	if _, err := ctx.Enqueue(context.Background(), "agent-2", "profile-2"); err != nil {
		t.Fatalf("enqueue agent-2: %v", err)
	}
	_, err = ctx.Enqueue(context.Background(), "agent-2", "profile-2")
	if err == nil {
		t.Fatal("expected error for duplicate queue entry")
	}
	if !IsMemberAlreadyQueued(err) {
		t.Fatalf("expected MemberAlreadyQueuedError, got %T: %v", err, err)
	}
}

func TestDequeueNext_StartsNextMember(t *testing.T) {
	exec := &captureExecutor{}
	ctx := newTeamExecutionContext("team-1", exec, t.TempDir())

	// Start agent-1, queue agent-2
	if _, err := ctx.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue agent-1: %v", err)
	}
	if _, err := ctx.Enqueue(context.Background(), "agent-2", "profile-2"); err != nil {
		t.Fatalf("enqueue agent-2: %v", err)
	}

	// Complete agent-1 -> agent-2 should start
	ctx.OnMemberComplete("agent-1")

	// Give goroutine a moment to start
	time.Sleep(50 * time.Millisecond)

	status := ctx.Status()
	if status.State != "active" {
		t.Fatalf("expected state 'active', got %q", status.State)
	}
	if status.Running == nil || *status.Running != "agent-2" {
		t.Fatalf("expected running agent-2, got %v", status.Running)
	}
	if len(status.Queue) != 0 {
		t.Fatalf("expected empty queue, got %d", len(status.Queue))
	}
}

func TestDequeueNext_BecomesIdleWhenEmpty(t *testing.T) {
	exec := &captureExecutor{}
	ctx := newTeamExecutionContext("team-1", exec, t.TempDir())

	// Start agent-1
	if _, err := ctx.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue agent-1: %v", err)
	}

	// Complete agent-1 with empty queue
	ctx.OnMemberComplete("agent-1")

	status := ctx.Status()
	if status.State != "idle" {
		t.Fatalf("expected state 'idle', got %q", status.State)
	}
	if status.Running != nil {
		t.Fatalf("expected running nil, got %v", status.Running)
	}
}

func TestQueueOrderIsFIFO(t *testing.T) {
	exec := &captureExecutor{}
	ctx := newTeamExecutionContext("team-1", exec, t.TempDir())

	// Start agent-1, queue agent-2, agent-3, agent-4
	if _, err := ctx.Enqueue(context.Background(), "agent-1", "p"); err != nil {
		t.Fatalf("enqueue agent-1: %v", err)
	}
	for _, id := range []string{"agent-2", "agent-3", "agent-4"} {
		if _, err := ctx.Enqueue(context.Background(), id, "p"); err != nil {
			t.Fatalf("enqueue %s: %v", id, err)
		}
	}

	status := ctx.Status()
	expected := []string{"agent-2", "agent-3", "agent-4"}
	if len(status.Queue) != len(expected) {
		t.Fatalf("expected queue length %d, got %d", len(expected), len(status.Queue))
	}
	for i, id := range expected {
		if status.Queue[i] != id {
			t.Fatalf("queue[%d]: expected %q, got %q", i, id, status.Queue[i])
		}
	}

	// Complete agent-1 -> agent-2 should be next
	ctx.OnMemberComplete("agent-1")
	time.Sleep(50 * time.Millisecond)

	status = ctx.Status()
	if status.Running == nil || *status.Running != "agent-2" {
		t.Fatalf("expected running agent-2, got %v", status.Running)
	}
	if len(status.Queue) != 2 {
		t.Fatalf("expected 2 remaining in queue, got %d", len(status.Queue))
	}
	if status.Queue[0] != "agent-3" || status.Queue[1] != "agent-4" {
		t.Fatalf("expected queue [agent-3, agent-4], got %v", status.Queue)
	}
}

func TestPersistAndRecover(t *testing.T) {
	dir := t.TempDir()
	exec := &captureExecutor{}
	ctx := newTeamExecutionContext("team-1", exec, dir)

	// Start agent-1, queue agent-2, agent-3
	if _, err := ctx.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue agent-1: %v", err)
	}
	if _, err := ctx.Enqueue(context.Background(), "agent-2", "profile-2"); err != nil {
		t.Fatalf("enqueue agent-2: %v", err)
	}
	if _, err := ctx.Enqueue(context.Background(), "agent-3", "profile-3"); err != nil {
		t.Fatalf("enqueue agent-3: %v", err)
	}

	// Verify queue file exists
	queueFile := filepath.Join(dir, "team-queue-team-1.json")
	if _, err := os.Stat(queueFile); os.IsNotExist(err) {
		t.Fatal("expected queue file to exist after enqueue")
	}

	// Create new context and recover
	ctx2 := newTeamExecutionContext("team-1", exec, dir)
	ctx2.Recover()

	status := ctx2.Status()
	if status.Running == nil || *status.Running != "agent-1" {
		t.Fatalf("expected recovered running agent-1, got %v", status.Running)
	}
	if len(status.Queue) != 2 {
		t.Fatalf("expected 2 in recovered queue, got %d", len(status.Queue))
	}
	if status.Queue[0] != "agent-2" || status.Queue[1] != "agent-3" {
		t.Fatalf("expected recovered queue [agent-2, agent-3], got %v", status.Queue)
	}
}

func TestStatus_ReflectsCurrentState(t *testing.T) {
	exec := &captureExecutor{}
	ctx := newTeamExecutionContext("team-1", exec, t.TempDir())

	// Initially idle
	status := ctx.Status()
	if status.State != "idle" {
		t.Fatalf("expected initial state 'idle', got %q", status.State)
	}
	if status.Running != nil {
		t.Fatalf("expected initial running nil, got %v", status.Running)
	}
	if len(status.Queue) != 0 {
		t.Fatalf("expected initial empty queue, got %d", len(status.Queue))
	}
	if status.TeamID != "team-1" {
		t.Fatalf("expected teamId 'team-1', got %q", status.TeamID)
	}

	// After enqueue -> active
	if _, err := ctx.Enqueue(context.Background(), "agent-1", "p"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	status = ctx.Status()
	if status.State != "active" {
		t.Fatalf("expected state 'active' after enqueue, got %q", status.State)
	}

	// After completion with empty queue -> idle
	ctx.OnMemberComplete("agent-1")
	status = ctx.Status()
	if status.State != "idle" {
		t.Fatalf("expected state 'idle' after completion, got %q", status.State)
	}
}

// TestTeamExecutionStore_Recover tests store-level recovery across teams.
func TestTeamExecutionStore_Recover(t *testing.T) {
	dir := t.TempDir()
	exec := &captureExecutor{}

	// Create some queue state manually
	store1 := NewTeamExecutionStore(exec, dir)
	if _, err := store1.Enqueue(context.Background(), "team-a", "agent-1", "p1"); err != nil {
		t.Fatalf("enqueue team-a/agent-1: %v", err)
	}
	if _, err := store1.Enqueue(context.Background(), "team-b", "agent-2", "p2"); err != nil {
		t.Fatalf("enqueue team-b/agent-2: %v", err)
	}

	// Create new store and recover
	store2 := NewTeamExecutionStore(exec, dir)
	store2.Recover(context.Background())

	statusA := store2.Status("team-a")
	if statusA.State != "active" {
		t.Fatalf("expected team-a state 'active', got %q", statusA.State)
	}

	statusB := store2.Status("team-b")
	if statusB.State != "active" {
		t.Fatalf("expected team-b state 'active', got %q", statusB.State)
	}
}

// TestTeamExecutionStore_OnComplete routes completion to correct team.
func TestTeamExecutionStore_OnComplete(t *testing.T) {
	exec := &captureExecutor{}
	store := NewTeamExecutionStore(exec, t.TempDir())

	// Enqueue agent on team-1
	if _, err := store.Enqueue(context.Background(), "team-1", "agent-1", "p"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	// Queue agent-2
	if _, err := store.Enqueue(context.Background(), "team-1", "agent-2", "p"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// Complete agent-1
	store.OnComplete("team-1", "agent-1")
	time.Sleep(50 * time.Millisecond)

	status := store.Status("team-1")
	if status.Running == nil || *status.Running != "agent-2" {
		t.Fatalf("expected running agent-2 after OnComplete, got %v", status.Running)
	}
}

// TestTeamExecutionStore_StatusUnknownTeam returns idle for unknown teams.
func TestTeamExecutionStore_StatusUnknownTeam(t *testing.T) {
	exec := &captureExecutor{}
	store := NewTeamExecutionStore(exec, t.TempDir())

	status := store.Status("nonexistent")
	if status.State != "idle" {
		t.Fatalf("expected idle for unknown team, got %q", status.State)
	}
	if status.TeamID != "nonexistent" {
		t.Fatalf("expected teamId 'nonexistent', got %q", status.TeamID)
	}
}
