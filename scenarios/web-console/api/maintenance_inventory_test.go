package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/api-core/lifecycle"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"web-console/backends/codex"
	"web-console/internal/sessionstore"
)

func TestCodexMaintenanceClassifiesContinuation(t *testing.T) {
	activeGoal := &codex.NativeGoalState{Status: "active"}
	pausedGoal := &codex.NativeGoalState{Status: "paused"}
	tests := []struct {
		name  string
		state codex.NativeThreadState
		want  string
	}{
		{"idle", codex.NativeThreadState{Status: "idle", GoalAvailable: true}, "idle"},
		{"paused goal", codex.NativeThreadState{Status: "idle", GoalAvailable: true, Goal: pausedGoal}, "idle"},
		{"ordinary", codex.NativeThreadState{Status: "active", GoalAvailable: true}, "ordinary_turn_running"},
		{"durable goal", codex.NativeThreadState{Status: "active", ActiveTurnID: "goal-turn", GoalAvailable: true, Goal: activeGoal}, "native_goal_continuation"},
		{"ordinary input with goal", codex.NativeThreadState{Status: "active", ActiveTurnID: "ordinary-turn", ConsoleTurn: true, GoalAvailable: true, Goal: activeGoal}, "ordinary_turn_running"},
		{"ambiguous turn with goal", codex.NativeThreadState{Status: "active", GoalAvailable: true, Goal: activeGoal}, "native_state_unverified"},
		{"approval", codex.NativeThreadState{Status: "active", ActiveFlags: []string{"waitingOnApproval"}, GoalAvailable: true, Goal: activeGoal}, "waiting_for_native_input"},
		{"missing evidence", codex.NativeThreadState{Status: "idle"}, "native_state_unverified"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := codexMaintenanceStatus(tc.state); got != tc.want {
				t.Fatalf("maintenance state = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReadManagedThreadStatesUsesBoundedConcurrency(t *testing.T) {
	rows := make([]sessionstore.Metadata, 32)
	for i := range rows {
		rows[i] = sessionstore.Metadata{ID: fmt.Sprintf("session-%02d", i), NativeThreadID: fmt.Sprintf("thread-%02d", i)}
	}
	var active, peak atomic.Int32
	read := func(ctx context.Context, row sessionstore.Metadata, ids []string) ([]codex.NativeThreadState, error) {
		current := active.Add(1)
		defer active.Add(-1)
		for previous := peak.Load(); current > previous && !peak.CompareAndSwap(previous, current); previous = peak.Load() {
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
		if len(ids) != 1 || ids[0] != row.NativeThreadID {
			t.Errorf("thread IDs = %v, want [%s]", ids, row.NativeThreadID)
		}
		return []codex.NativeThreadState{{ThreadID: ids[0], Status: "idle", GoalAvailable: true}}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got := readManagedThreadStates(ctx, rows, read)
	if len(got) != len(rows) {
		t.Fatalf("observations = %d, want %d", len(got), len(rows))
	}
	if max := peak.Load(); max < 2 || max > maintenanceInventoryConcurrency {
		t.Fatalf("peak concurrent reads = %d, want 2..%d", max, maintenanceInventoryConcurrency)
	}
	for _, row := range rows {
		if got[row.ID].status != "idle" {
			t.Errorf("%s status = %q, want idle", row.ID, got[row.ID].status)
		}
	}
}

func TestRecoveryOnlyCodexThreadStaysInactiveDuringMaintenance(t *testing.T) {
	if got := codexMaintenanceReadMode(sessionstore.ControlModeRecoveryOnly); got != "recovery_only" {
		t.Fatalf("recovery-only read mode = %q, want recovery_only", got)
	}
	if got := codexMaintenanceReadMode(sessionstore.ControlModeNativeCapable); got != "read" {
		t.Fatalf("native-capable read mode = %q, want read", got)
	}
	if got := codexMaintenanceReadMode(""); got != "native_state_unverified" {
		t.Fatalf("unknown read mode = %q, want native_state_unverified", got)
	}

	inventory := hostWorkInventory(lifecycle.Inventory{Complete: true, Work: []*commonv1.LifecycleWorkItem{{
		Id: "inactive", Kind: "codex_app_server", Status: "recovery_only",
	}}})
	if !inventory.Complete || len(inventory.Blockers) != 0 || len(inventory.Work) != 0 {
		t.Fatalf("host recovery-only inventory = %+v, want complete with no blocker or active work", inventory)
	}
}
