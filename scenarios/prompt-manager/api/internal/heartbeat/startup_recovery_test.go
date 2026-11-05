package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStartupRecoveryUsesQueueOwnerReadAndRetainsRegistryOnlyRun(t *testing.T) {
	dir := t.TempDir()
	started := time.Date(2026, 10, 20, 8, 0, 0, 0, time.UTC)
	seed := NewRunRegistry(dir)
	seed.RegisterRecovered("team", "queued-agent", "queue-run", started, ObligationRunning, nil)
	seed.RegisterRecovered("team", "legacy-agent", "legacy-run", started.Add(-time.Hour), ObligationRunning, nil)

	registry := NewRunRegistry(dir)
	observer := &runRegistryObserverRecorder{}
	registry.SetObserver(observer)
	registry.LoadPersisted()
	client := newMockAgentClient().
		WithGetRunResponse("queue-run", &Run{ID: "queue-run", Status: "RUN_STATUS_PARKED", StartedAt: started.Format(time.RFC3339)}).
		WithGetRunResponse("legacy-run", &Run{ID: "legacy-run", Status: "RUN_STATUS_RUNNING", StartedAt: started.Add(-time.Hour).Format(time.RFC3339)})
	executor := newTestExecutor(t, nil, nil, client, "", registry, nil)
	store := NewTeamExecutionStore(nil, executor, dir, client)
	executor.SetTeamExecStore(store)
	writeQueueFile(t, dir, "team", persistedTeamQueue{
		TeamID:  "team",
		Running: []queuedExecution{{AgentID: "queued-agent", ProfileKey: "key", RunID: "queue-run", State: ObligationRunning}},
	})

	covered := store.Recover(context.Background())
	if _, ok := covered[runOwnerIdentity{TeamID: "team", AgentID: "queued-agent", RunID: "queue-run"}]; !ok {
		t.Fatal("queue-owned identity was not returned as covered")
	}
	registry.Recover(context.Background(), client, covered)

	client.mu.Lock()
	gotReads := append([]string(nil), client.getRunCalls...)
	created := len(client.createRunCalls)
	client.mu.Unlock()
	if strings.Join(gotReads, ",") != "queue-run,legacy-run" {
		t.Fatalf("owner reads = %v, want one queue read followed by one legacy fallback", gotReads)
	}
	if created != 0 {
		t.Fatalf("startup recovery dispatched %d new runs", created)
	}
	if observer.started != 0 {
		t.Fatalf("startup hydration emitted %d synthetic RunStarted events", observer.started)
	}
	queueOwner, ok := registry.GetActiveRun("team", "queued-agent")
	if !ok || queueOwner.RunID != "queue-run" || queueOwner.State != ObligationPaused || !queueOwner.StartedAt.Equal(started) {
		t.Fatalf("queue owner projection = %+v, ok=%v", queueOwner, ok)
	}
	legacyOwner, ok := registry.GetActiveRun("team", "legacy-agent")
	if !ok || legacyOwner.RunID != "legacy-run" || legacyOwner.State != ObligationRunning {
		t.Fatalf("registry-only legacy owner projection = %+v, ok=%v", legacyOwner, ok)
	}
}

func TestStartupRecoveryReportsDifferentLegacyRunWithoutReplacingQueuePresentation(t *testing.T) {
	dir := t.TempDir()
	seed := NewRunRegistry(dir)
	seed.RegisterRecovered("team", "agent", "legacy-run", time.Time{}, ObligationRunning, nil)

	registry := NewRunRegistry(dir)
	registry.LoadPersisted()
	client := newMockAgentClient().
		WithGetRunResponse("queue-run", &Run{ID: "queue-run", Status: "RUN_STATUS_PARKED"}).
		WithGetRunResponse("legacy-run", &Run{ID: "legacy-run", Status: "RUN_STATUS_RUNNING"})
	executor := newTestExecutor(t, nil, nil, client, "", registry, nil)
	store := NewTeamExecutionStore(nil, executor, dir, client)
	executor.SetTeamExecStore(store)
	writeQueueFile(t, dir, "team", persistedTeamQueue{
		TeamID:  "team",
		Running: []queuedExecution{{AgentID: "agent", ProfileKey: "key", RunID: "queue-run", State: ObligationRunning}},
	})

	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previous) })
	covered := store.Recover(context.Background())
	// Queue hydration persists its canonical presentation before legacy
	// fallback classification. Both exact owner references must survive it.
	assertPersistedRunIDs(t, filepath.Join(dir, "heartbeat-active-runs.json"), "queue-run", "legacy-run")
	registry.Recover(context.Background(), client, covered)

	client.mu.Lock()
	gotReads := append([]string(nil), client.getRunCalls...)
	created := len(client.createRunCalls)
	client.mu.Unlock()
	if strings.Join(gotReads, ",") != "queue-run,legacy-run" {
		t.Fatalf("owner reads = %v, want queue and distinct legacy identity exactly once", gotReads)
	}
	if created != 0 {
		t.Fatalf("startup recovery dispatched %d new runs", created)
	}
	queueOwner, ok := registry.GetActiveRun("team", "agent")
	if !ok || queueOwner.RunID != "queue-run" {
		t.Fatalf("queue owner was replaced by legacy registry row: %+v, ok=%v", queueOwner, ok)
	}
	for _, required := range []string{"startup owner conflict", "queue_run_id=queue-run", "legacy_run_id=legacy-run", "legacy_owner_state=RUN_STATUS_RUNNING"} {
		if !strings.Contains(logs.String(), required) {
			t.Fatalf("conflict diagnostic missing %q: %s", required, logs.String())
		}
	}
	assertPersistedRunIDs(t, filepath.Join(dir, "heartbeat-active-runs.json"), "queue-run", "legacy-run")

	// The unresolved legacy identity survives another process start and is
	// independently checked again, while the queue run stays the sole primary.
	restarted := NewRunRegistry(dir)
	restarted.LoadPersisted()
	retryClient := newMockAgentClient().
		WithGetRunResponse("queue-run", &Run{ID: "queue-run", Status: "RUN_STATUS_PARKED"})
	retryExecutor := newTestExecutor(t, nil, nil, retryClient, "", restarted, nil)
	retryStore := NewTeamExecutionStore(nil, retryExecutor, dir, retryClient)
	retryExecutor.SetTeamExecStore(retryStore)
	retryCoverage := retryStore.Recover(context.Background())
	restarted.Recover(context.Background(), retryClient, retryCoverage)
	retryClient.mu.Lock()
	retryReads := append([]string(nil), retryClient.getRunCalls...)
	retryClient.mu.Unlock()
	if strings.Join(retryReads, ",") != "legacy-run" {
		t.Fatalf("restart owner reads = %v, want one exact read for the retained legacy identity", retryReads)
	}
	queueOwner, ok = restarted.GetActiveRun("team", "agent")
	if !ok || queueOwner.RunID != "queue-run" {
		t.Fatalf("restart changed queue primary presentation: %+v, ok=%v", queueOwner, ok)
	}
	assertPersistedRunIDs(t, filepath.Join(dir, "heartbeat-active-runs.json"), "queue-run", "legacy-run")

	// Exact terminal evidence retires only the supplemental legacy reference.
	terminalRegistry := NewRunRegistry(dir)
	terminalRegistry.LoadPersisted()
	terminalClient := newMockAgentClient().
		WithGetRunResponse("queue-run", &Run{ID: "queue-run", Status: "RUN_STATUS_PARKED"}).
		WithGetRunResponse("legacy-run", &Run{ID: "legacy-run", Status: "RUN_STATUS_COMPLETE"})
	terminalExecutor := newTestExecutor(t, nil, nil, terminalClient, "", terminalRegistry, nil)
	terminalStore := NewTeamExecutionStore(nil, terminalExecutor, dir, terminalClient)
	terminalExecutor.SetTeamExecStore(terminalStore)
	terminalCoverage := terminalStore.Recover(context.Background())
	terminalRegistry.Recover(context.Background(), terminalClient, terminalCoverage)
	terminalClient.mu.Lock()
	terminalReads := append([]string(nil), terminalClient.getRunCalls...)
	terminalClient.mu.Unlock()
	if strings.Join(terminalReads, ",") != "legacy-run" {
		t.Fatalf("terminal conflict reads = %v, want only the distinct legacy identity", terminalReads)
	}
	assertPersistedRunIDs(t, filepath.Join(dir, "heartbeat-active-runs.json"), "queue-run")
}

func assertPersistedRunIDs(t *testing.T, path string, want ...string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read persisted registry: %v", err)
	}
	var runs []persistedRun
	if err := json.Unmarshal(data, &runs); err != nil {
		t.Fatalf("decode persisted registry: %v", err)
	}
	got := make([]string, 0, len(runs))
	for _, run := range runs {
		got = append(got, run.RunID)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("persisted run IDs = %v, want %v", got, want)
	}
}

func TestStartupRecoveryFallsBackToRegistryWhenQueueSnapshotMissingOrCorrupt(t *testing.T) {
	for _, test := range []struct {
		name    string
		corrupt bool
	}{
		{name: "missing"},
		{name: "corrupt", corrupt: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			seed := NewRunRegistry(dir)
			seed.RegisterRecovered("team", "agent", "legacy-run", time.Time{}, ObligationRunning, nil)
			if test.corrupt {
				writeQueueFile(t, dir, "team", persistedTeamQueue{TeamID: "team"})
				// Model an interrupted write of a real queue snapshot.
				if err := os.Truncate(filepath.Join(dir, "team-queue-team.json"), 1); err != nil {
					t.Fatal(err)
				}
			}

			registry := NewRunRegistry(dir)
			registry.LoadPersisted()
			client := newMockAgentClient().WithGetRunResponse("legacy-run", &Run{ID: "legacy-run", Status: "RUN_STATUS_RUNNING"})
			executor := newTestExecutor(t, nil, nil, client, "", registry, nil)
			store := NewTeamExecutionStore(nil, executor, dir, client)
			executor.SetTeamExecStore(store)
			covered := store.Recover(context.Background())
			registry.Recover(context.Background(), client, covered)

			client.mu.Lock()
			gotReads := append([]string(nil), client.getRunCalls...)
			created := len(client.createRunCalls)
			client.mu.Unlock()
			if strings.Join(gotReads, ",") != "legacy-run" {
				t.Fatalf("fallback owner reads = %v, want [legacy-run]", gotReads)
			}
			if created != 0 {
				t.Fatalf("startup recovery dispatched %d new runs", created)
			}
			if run, ok := registry.GetActiveRun("team", "agent"); !ok || run.RunID != "legacy-run" {
				t.Fatalf("registry fallback lost owner: %+v, ok=%v", run, ok)
			}
		})
	}
}

func TestQueueRecoveryDoesNotClassifyMismatchedOwnerResponseAsTerminal(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().WithGetRunResponse("queue-run", &Run{ID: "different-run", Status: "RUN_STATUS_COMPLETE"})
	registry := NewRunRegistry(dir)
	executor := newTestExecutor(t, nil, nil, client, "", registry, nil)
	store := NewTeamExecutionStore(nil, executor, dir, client)
	executor.SetTeamExecStore(store)
	writeQueueFile(t, dir, "team", persistedTeamQueue{
		TeamID:  "team",
		Running: []queuedExecution{{AgentID: "agent", ProfileKey: "key", RunID: "queue-run", State: ObligationRunning}},
	})

	owners := store.Recover(context.Background())
	if _, ok := owners[runOwnerIdentity{TeamID: "team", AgentID: "agent", RunID: "queue-run"}]; !ok {
		t.Fatal("mismatched response must retain exact queue identity as covered")
	}
	status := store.GetOrCreate("team").Status()
	if len(status.RunningAgentIDs) != 1 || len(status.UncertainAgentIDs) != 1 {
		t.Fatalf("mismatched owner response must retain uncertain queue slot: %+v", status)
	}
	active, ok := registry.GetActiveRun("team", "agent")
	if !ok || active.RunID != "queue-run" || active.State != ObligationDispatchUncertain {
		t.Fatalf("registry adopted mismatched owner response: %+v, ok=%v", active, ok)
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.getRunCalls) != 1 || client.getRunCalls[0] != "queue-run" {
		t.Fatalf("queue owner reads = %v, want one exact read for queue-run", client.getRunCalls)
	}
	if len(client.createRunCalls) != 0 {
		t.Fatalf("mismatched terminal response released capacity and dispatched %d runs", len(client.createRunCalls))
	}
}
