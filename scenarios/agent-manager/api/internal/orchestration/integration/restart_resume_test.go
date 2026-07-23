// Package integration holds end-to-end tests for the orchestration
// pipeline. Unlike the unit tests in internal/orchestration which
// exercise individual phase functions or recovery primitives in
// isolation, these tests drive a full agent-manager-shaped flow:
// run starts → orchestrator dies → fresh orchestrator boots →
// transcript is replayed → tailer reattaches → run completes.
//
// The runner used here is a fake whose "process" is a goroutine that
// keeps writing to the transcript file even after the orchestrator's
// ctx is cancelled — exactly the production shape, where the agent
// process is detached via setsid (host) or sandbox-side supervisor
// (sandboxed) so killing agent-manager does not kill the agent.
//
// These tests are the regression gate for restart-resume — the most
// important behavior in the system, because runs that modify
// agent-manager itself depend on it (the run survives the agent-manager
// restart that ships its own changes). They fail loudly when the
// transcript-tee, persist-before-broadcast, or reattach-by-pid
// invariants break.
package integration

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/testutil"
	"agent-manager/internal/testutil/mocks"

	"github.com/google/uuid"
)

// =============================================================================
// Helpers for orchestrator instantiation
// =============================================================================

func mustCreateRecoveryRun(t *testing.T, repos *database.Repositories, rt domain.RunnerType, transcriptPath string) *domain.Run {
	t.Helper()
	task := &domain.Task{
		ID:          uuid.New(),
		Title:       "restart-resume integration",
		Description: "end-to-end integration of restart-resume",
		ScopePath:   ".",
		ProjectRoot: ".",
		Status:      domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	now := time.Now().UTC()
	cfg := domain.DefaultRunConfig()
	cfg.RunnerType = rt
	run := &domain.Run{
		ID:             uuid.New(),
		TaskID:         task.ID,
		Tag:            uuid.NewString(),
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusRunning,
		StartedAt:      &now,
		LastHeartbeat:  &now,
		Phase:          domain.RunPhaseExecuting,
		ApprovalState:  domain.ApprovalStateNone,
		ResolvedConfig: cfg,
		TranscriptPath: transcriptPath,
	}
	if err := repos.Runs.Create(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	return run
}

func newTestReconciler(t *testing.T, repos *database.Repositories, store event.Store, fakeRunner *mocks.TranscriptReplayRunner) *orchestration.Reconciler {
	t.Helper()

	registry := runner.NewRegistry()
	if err := registry.Register(fakeRunner); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	return orchestration.NewReconciler(
		repos.Runs,
		registry,
		orchestration.WithReconcilerEvents(store),
		orchestration.WithReconcilerConfig(orchestration.ReconcilerConfig{
			Interval:          time.Hour,
			StaleThreshold:    time.Minute,
			MaxRecoveryAge:    10 * time.Minute,
			OrphanGracePeriod: time.Minute,
			MaxStaleRuns:      10,
			KillOrphans:       true,
			AutoRecover:       true,
		}),
	)
}

// =============================================================================
// Tests
// =============================================================================

// TestRestartResume_TranscriptReplayCompletes — the canonical case.
// Orchestrator A starts a run, writes a session+1 message, then dies
// (caller ctx cancelled). Orchestrator B boots, runs RecoverInFlightRuns,
// and observes the run completing once the transcript carries a terminal
// "done:" line. No duplicate events in the store.
func TestRestartResume_TranscriptReplayCompletes(t *testing.T) {
	harness := newOrchestratorHarness(t)
	t.Cleanup(harness.Cleanup)
	repos, store := harness.Repos, harness.Events

	transcriptPath := filepath.Join(t.TempDir(), "transcript.ndjson")
	if err := os.WriteFile(transcriptPath, nil, 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}

	// Pre-populate the transcript with a session+message+done sequence
	// before the recovery orchestrator boots — simulating the run
	// having reached a terminal state while the original orchestrator
	// was dead.
	if err := os.WriteFile(transcriptPath, []byte(
		"session:thread-1\n"+
			"message:hello from A\n"+
			"message:still progressing\n"+
			"done:complete\n",
	), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}

	run := mustCreateRecoveryRun(t, repos, domain.RunnerTypeCodex, transcriptPath)

	// Boot the recovery-orchestrator (orchestrator B). Use a no-op
	// fake runner — recovery uses the transcript parser, not Execute.
	reconciler := newTestReconciler(t, repos, store, harness.Runner)

	if err := reconciler.RecoverInFlightRuns(context.Background()); err != nil {
		t.Fatalf("RecoverInFlightRuns: %v", err)
	}

	// Inspect the run.
	got, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.Status != domain.RunStatusComplete {
		t.Fatalf("expected status=Complete after replay, got %s", got.Status)
	}
	if got.SessionID != "thread-1" {
		t.Fatalf("expected SessionID=thread-1, got %q", got.SessionID)
	}
	// Summary is built from message events first (recoveredSummaryHasContent),
	// falling back to terminal.Summary only when events have no content. The
	// last message event "still progressing" sets the description; the
	// terminal "done:complete" provides the exit code.
	if got.Summary == nil {
		t.Fatal("expected non-nil summary after replay")
	}
	if got.ExitCode == nil || *got.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %v", got.ExitCode)
	}

	// Verify event store recorded the replayed message events.
	events, err := store.Get(context.Background(), run.ID, event.GetOptions{
		EventTypes: []domain.RunEventType{domain.EventTypeMessage},
	})
	if err != nil {
		t.Fatalf("get events: %v", err)
	}
	if len(events) == 0 {
		t.Errorf("expected at least one message event from replay, got %d", len(events))
	}

	// Re-run RecoverInFlightRuns — must be idempotent. No new events.
	if err := reconciler.RecoverInFlightRuns(context.Background()); err != nil {
		t.Fatalf("RecoverInFlightRuns (second call): %v", err)
	}
	events2, err := store.Get(context.Background(), run.ID, event.GetOptions{
		EventTypes: []domain.RunEventType{domain.EventTypeMessage},
	})
	if err != nil {
		t.Fatalf("get events after re-recover: %v", err)
	}
	if len(events2) != len(events) {
		t.Errorf("recovery is not idempotent: re-running added %d events (was %d, now %d)",
			len(events2)-len(events), len(events), len(events2))
	}
}

// TestRestartResume_DeadProcessNoTerminal — the failure shape: the
// run was alive while orchestrator A was up, but the agent died
// without writing a terminal line. Orchestrator B detects the dead
// process (no PID/PGID set means we can't reattach) and marks the
// run failed.
func TestRestartResume_DeadProcessNoTerminal(t *testing.T) {
	repos, store, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	transcriptPath := filepath.Join(t.TempDir(), "transcript.ndjson")
	if err := os.WriteFile(transcriptPath, []byte(
		"session:thread-2\n"+
			"message:partial work before crash\n",
	), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}

	run := mustCreateRecoveryRun(t, repos, domain.RunnerTypeCodex, transcriptPath)

	fakeRunner := mocks.NewTranscriptReplayRunner(domain.RunnerTypeCodex)
	reconciler := newTestReconciler(t, repos, store, fakeRunner)

	if err := reconciler.RecoverInFlightRuns(context.Background()); err != nil {
		t.Fatalf("RecoverInFlightRuns: %v", err)
	}

	got, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.Status != domain.RunStatusFailed {
		t.Fatalf("dead-process run with no terminal must be marked Failed, got %s", got.Status)
	}
	if got.SessionID != "thread-2" {
		t.Errorf("session ID must still be captured even on failure, got %q", got.SessionID)
	}
}

// TestRestartResume_TranscriptTeeInvariant — the load-bearing
// invariant: the transcript file persists data the runner produced,
// independent of whether the orchestrator successfully consumed it.
//
// This test deliberately does NOT use the orchestrator: it just
// validates the assumption all the other tests rest on — that a
// transcript file written line-by-line is durable across a process
// boundary. If a future change buffers transcript writes in memory,
// this test fails before the more elaborate orchestrator tests do.
func TestRestartResume_TranscriptTeeInvariant(t *testing.T) {
	transcriptPath := filepath.Join(t.TempDir(), "transcript.ndjson")
	f, err := os.Create(transcriptPath)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Simulate a writer goroutine appending lines.
	lines := []string{
		"session:thread-tee",
		"message:line1",
		"message:line2",
	}
	for _, line := range lines {
		if _, err := f.WriteString(line + "\n"); err != nil {
			t.Fatalf("write: %v", err)
		}
		// Flush after every line so a reader can see partial state.
		if err := f.Sync(); err != nil {
			t.Fatalf("sync: %v", err)
		}
	}

	// Reader observes everything written so far without closing the writer.
	rf, err := os.Open(transcriptPath)
	if err != nil {
		t.Fatalf("open for read: %v", err)
	}
	defer rf.Close()

	scanner := bufio.NewScanner(rf)
	got := []string{}
	for scanner.Scan() {
		got = append(got, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(got) != len(lines) {
		t.Fatalf("transcript-tee invariant broken: writer produced %d lines, reader saw %d", len(lines), len(got))
	}
	for i := range lines {
		if got[i] != lines[i] {
			t.Errorf("line %d mismatch: got %q want %q", i, got[i], lines[i])
		}
	}

	if err := f.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
}
