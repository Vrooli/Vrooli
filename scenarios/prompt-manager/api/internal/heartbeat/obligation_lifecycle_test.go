package heartbeat

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestObligationLifecycleTransitions(t *testing.T) {
	for _, tc := range []struct {
		name string
		from string
		to   string
		ok   bool
	}{
		{"new dispatch intent", ObligationRunning, ObligationDispatchUncertain, true},
		{"cannot cancel without a retained intent", ObligationRunning, ObligationDispatchCanceled, false},
		{"uncertain dispatch canceled before send", ObligationDispatchUncertain, ObligationDispatchCanceled, true},
		{"owner observation resolves uncertainty", ObligationDispatchUncertain, ObligationPaused, true},
		{"approval retry returns to uncertain", ObligationApprovalPending, ObligationDispatchUncertain, true},
		{"deferred retry becomes live", ObligationAdmissionDeferred, ObligationRunning, true},
		{"paused run resumes", ObligationPaused, ObligationRunning, true},
		{"watcher recovers paused owner", ObligationWatcherUnavailable, ObligationPaused, true},
		{"interrupted preparation is replaced", ObligationPreparationInterrupted, ObligationRunning, true},
		{"same-state checkpoint", ObligationDispatchUncertain, ObligationDispatchUncertain, true},
		{"tombstone cannot be reopened", ObligationDispatchCanceled, ObligationRunning, false},
		{"unknown persisted state rejected", "corrupt", ObligationRunning, false},
		{"unknown target rejected", ObligationRunning, "corrupt", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := transitionObligationState(tc.from, tc.to)
			if (err == nil) != tc.ok {
				t.Fatalf("transition %q -> %q error=%v, want allowed=%v", tc.from, tc.to, err, tc.ok)
			}
		})
	}
}

func TestRunningEntryTransitionDoesNotMutateOnInvalidEdge(t *testing.T) {
	entry := runningEntry{State: ObligationDispatchCanceled}
	if err := entry.transition(ObligationRunning); err == nil {
		t.Fatal("expected canceled dispatch tombstone to reject reopening")
	}
	if entry.State != ObligationDispatchCanceled {
		t.Fatalf("invalid transition changed state to %q", entry.State)
	}
}

func TestRecoverRejectsUnknownPersistedObligationWithoutOverwritingFile(t *testing.T) {
	ctx := newTeamExecutionContext("team", &captureExecutor{}, t.TempDir(), nil)
	original := []byte(`{"teamId":"team","running":[{"AgentID":"agent","State":"future_state"}]}`)
	if err := os.MkdirAll(filepath.Dir(ctx.queueFilePath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ctx.queueFilePath(), original, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ctx.Recover(context.Background()); len(got) != 0 {
		t.Fatalf("recovered owners = %v, want none for unknown state", got)
	}
	retained, err := os.ReadFile(ctx.queueFilePath())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(retained, original) {
		t.Fatalf("invalid persisted state was overwritten: %s", retained)
	}
}

func TestRunRegistryRecoveryPreservesUnknownPersistedState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "heartbeat-active-runs.json")
	original := []byte(`[{"teamId":"team","agentId":"agent","runId":"run","state":"future_state"}]`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	registry := NewRunRegistry(dir)
	registry.Recover(context.Background(), nil, map[runOwnerIdentity]struct{}{})
	retained, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(retained, original) {
		t.Fatalf("invalid registry state was overwritten: %s", retained)
	}
}
