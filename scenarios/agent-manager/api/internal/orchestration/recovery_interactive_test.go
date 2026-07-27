package orchestration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/mocks"

	"github.com/google/uuid"
)

// fakeInteractiveSessions is an in-memory webconsole.SessionController for
// recovery tests: it reports whether a session is alive and records deletes.
type fakeInteractiveSessions struct {
	mu    sync.Mutex
	alive map[string]bool
}

func newFakeInteractiveSessions(aliveIDs ...string) *fakeInteractiveSessions {
	f := &fakeInteractiveSessions{alive: map[string]bool{}}
	for _, id := range aliveIDs {
		f.alive[id] = true
	}
	return f
}

func (f *fakeInteractiveSessions) setGone(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.alive[id] = false
}

func (f *fakeInteractiveSessions) CreateSession(context.Context, webconsole.CreateSessionParams) (string, error) {
	return "", nil
}

func (f *fakeInteractiveSessions) GetSession(_ context.Context, id string) (webconsole.SessionInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.alive[id] {
		return webconsole.SessionInfo{ID: id, Owner: webconsole.OwnerAgentManager}, nil
	}
	return webconsole.SessionInfo{}, webconsole.ErrSessionNotFound
}

func (f *fakeInteractiveSessions) DeleteSession(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.alive[id] = false
	return nil
}

func (f *fakeInteractiveSessions) SendText(context.Context, string, string, string) error { return nil }
func (f *fakeInteractiveSessions) SendPrompt(context.Context, string, string, string) error {
	return nil
}
func (f *fakeInteractiveSessions) Interrupt(context.Context, string, string) error { return nil }
func (f *fakeInteractiveSessions) Screen(context.Context, string, bool) (string, error) {
	return "", nil
}

func newInteractiveRecoveryReconciler(t *testing.T, sessions webconsole.SessionController) (*Reconciler, *database.Repositories) {
	t.Helper()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	registry := runner.NewRegistry()
	if err := registry.Register(mocks.NewTranscriptReplayRunner(domain.RunnerTypeClaudeCode)); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	reconciler := NewReconciler(
		repos.Runs,
		registry,
		WithReconcilerEvents(eventStore),
		WithReconcilerInteractive(sessions),
		WithReconcilerRunStateRoot(t.TempDir()),
		WithReconcilerConfig(ReconcilerConfig{
			Interval:          time.Hour,
			StaleThreshold:    time.Minute,
			MaxRecoveryAge:    10 * time.Minute,
			OrphanGracePeriod: time.Minute,
			MaxStaleRuns:      10,
			KillOrphans:       true,
			AutoRecover:       true,
		}),
	)
	reconciler.interactiveDebounce = 40 * time.Millisecond
	reconciler.interactiveSessionPoll = 20 * time.Millisecond
	return reconciler, repos
}

func createInteractiveRecoveryRun(t *testing.T, repos *database.Repositories, sessionID, transcriptPath string) *domain.Run {
	t.Helper()
	task := &domain.Task{
		ID:          uuid.New(),
		Title:       "Interactive recovery task",
		Description: "Validate interactive recovery",
		ScopePath:   ".",
		ProjectRoot: ".",
		Status:      domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	now := time.Now().UTC()
	stale := now.Add(-15 * time.Minute)
	cfg := domain.DefaultRunConfig()
	cfg.RunnerType = domain.RunnerTypeClaudeCode
	run := &domain.Run{
		ID:                  uuid.New(),
		TaskID:              task.ID,
		Tag:                 uuid.NewString(),
		RunMode:             domain.RunModeInPlace,
		Status:              domain.RunStatusRunning,
		StartedAt:           &now,
		LastHeartbeat:       &stale,
		Phase:               domain.RunPhaseExecuting,
		ApprovalState:       domain.ApprovalStateNone,
		ExecutionMode:       domain.ExecutionModeInteractive,
		WebConsoleSessionID: sessionID,
		TranscriptPath:      transcriptPath,
		ResolvedConfig:      cfg,
	}
	if err := repos.Runs.Create(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	return run
}

// TestRecoverInteractive_SessionGoneNoTerminal_FailsExplicitly proves an
// interactive run whose web-console session vanished (and has no terminal
// marker) is finalized Failed with an explicit reason naming the session —
// never left orphaned.
func TestRecoverInteractive_SessionGoneNoTerminal_FailsExplicitly(t *testing.T) {
	sessions := newFakeInteractiveSessions() // session id not alive => gone
	reconciler, repos := newInteractiveRecoveryReconciler(t, sessions)

	transcript := writeRecoveryTranscript(t, "message:partial interactive output\n")
	run := createInteractiveRecoveryRun(t, repos, "sess-gone", transcript)

	if _, err := reconciler.RecoverRun(context.Background(), run.ID); err != nil {
		t.Fatalf("RecoverRun: %v", err)
	}

	got, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.Status != domain.RunStatusFailed {
		t.Fatalf("status = %s, want failed", got.Status)
	}
	if !containsAll(got.ErrorMsg, "sess-gone", "no longer exists") {
		t.Errorf("error msg = %q, want it to name the vanished session", got.ErrorMsg)
	}
}

// TestRecoverInteractive_SessionGoneWithTerminal_Completes proves that when the
// transcript already carries a success terminal and the session is gone, the run
// completes (the last turn finished before the session was cleaned up).
func TestRecoverInteractive_SessionGoneWithTerminal_Completes(t *testing.T) {
	sessions := newFakeInteractiveSessions()
	reconciler, repos := newInteractiveRecoveryReconciler(t, sessions)

	transcript := writeRecoveryTranscript(t, "message:final interactive answer\ndone:final interactive answer\n")
	run := createInteractiveRecoveryRun(t, repos, "sess-gone", transcript)

	if _, err := reconciler.RecoverRun(context.Background(), run.ID); err != nil {
		t.Fatalf("RecoverRun: %v", err)
	}

	got, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.Status != domain.RunStatusComplete {
		t.Fatalf("status = %s, want complete", got.Status)
	}
}

// TestRecoverInteractive_SessionAliveReattachesUntilTerminal is the in-process
// restart drill: it drives the REAL recovery entry point (RecoverInFlightRuns),
// which discovers the in-flight interactive run from the persisted row, verifies
// the session is alive, and reattaches the transcript tailer. A terminal marker
// appended after reattach then completes the run — proving reattachment survives
// a simulated agent-manager restart.
func TestRecoverInteractive_SessionAliveReattachesUntilTerminal(t *testing.T) {
	sessions := newFakeInteractiveSessions("sess-live")
	reconciler, repos := newInteractiveRecoveryReconciler(t, sessions)

	// Seed the transcript with pre-restart output but no terminal yet.
	transcript := filepath.Join(t.TempDir(), "transcript.ndjson")
	if err := os.WriteFile(transcript, []byte("message:output before restart\n"), 0o644); err != nil {
		t.Fatalf("seed transcript: %v", err)
	}
	run := createInteractiveRecoveryRun(t, repos, "sess-live", transcript)

	// Real restart-recovery boot path: discover + reattach.
	if err := reconciler.RecoverInFlightRuns(context.Background()); err != nil {
		t.Fatalf("RecoverInFlightRuns: %v", err)
	}

	// The run must still be running (reattached, not finalized) right after boot.
	got, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.Status != domain.RunStatusRunning {
		t.Fatalf("status after reattach = %s, want still running", got.Status)
	}

	// The agent finishes its turn: append a final message + terminal marker to
	// the live session's transcript. The reattached tailer must observe it and
	// complete the run, without re-emitting the pre-restart output.
	appendRecoveryTranscript(t, transcript, "message:interactive completion\ndone:finished\n")

	waitForRecoveryStatus(t, repos, run.ID, domain.RunStatusComplete)
	final, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if final.Summary == nil || final.Summary.Description != "interactive completion" {
		t.Fatalf("summary = %+v, want interactive completion", final.Summary)
	}
}

// TestRecoverInteractive_SessionDiesMidReattach proves that if the session
// disappears while the reattached tailer is waiting for more output, the run is
// finalized Failed with an explicit reason rather than tailing forever.
func TestRecoverInteractive_SessionDiesMidReattach(t *testing.T) {
	sessions := newFakeInteractiveSessions("sess-flaky")
	reconciler, repos := newInteractiveRecoveryReconciler(t, sessions)
	// Speed up the mid-tail session watcher for this test.
	reconciler.interactiveDebounce = 40 * time.Millisecond

	transcript := filepath.Join(t.TempDir(), "transcript.ndjson")
	if err := os.WriteFile(transcript, []byte("message:waiting for input\n"), 0o644); err != nil {
		t.Fatalf("seed transcript: %v", err)
	}
	run := createInteractiveRecoveryRun(t, repos, "sess-flaky", transcript)

	if err := reconciler.RecoverInFlightRuns(context.Background()); err != nil {
		t.Fatalf("RecoverInFlightRuns: %v", err)
	}

	// The session dies while the tailer waits (no terminal ever arrives).
	sessions.setGone("sess-flaky")

	waitForRecoveryStatus(t, repos, run.ID, domain.RunStatusFailed)
	got, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if !containsAll(got.ErrorMsg, "sess-flaky", "no longer exists") {
		t.Errorf("error msg = %q, want it to name the vanished session", got.ErrorMsg)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
