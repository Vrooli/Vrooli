// Transcript scheduler tests cover lifecycle ownership and stable session-id
// extraction without touching the operator's real harness transcript roots.
package orchestration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
)

func TestTranscriptImportSchedulerStartStopIsIdempotent(t *testing.T) {
	scheduler := NewTranscriptImportScheduler(nil, time.Millisecond)
	scheduler.Start(context.Background())
	scheduler.Start(context.Background())
	scheduler.Stop()
	scheduler.Stop()
}

// The first tick is a whole interval away and every restart restarts that
// interval, so without a startup sweep a frequently-restarted deployment would
// never import at all. The long interval here means only a startup sweep can
// satisfy this test.
func TestTranscriptImportSchedulerSweepsOnStart(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	scheduler := NewTranscriptImportScheduler(&Orchestrator{}, time.Hour)
	scheduler.Start(context.Background())
	defer scheduler.Stop()

	deadline := time.Now().Add(2 * time.Second)
	for scheduler.Sweeps() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("scheduler did not sweep at startup")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// A cancelled context must not start work the caller has already given up on.
func TestTranscriptImportSchedulerSkipsSweepWhenContextIsDone(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	scheduler := NewTranscriptImportScheduler(&Orchestrator{}, time.Hour)
	scheduler.Start(ctx)
	scheduler.Stop()
	if got := scheduler.Sweeps(); got != 0 {
		t.Fatalf("sweeps = %d, want 0 for an already-cancelled context", got)
	}
}

func TestTranscriptSessionIDFromPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, []byte(`{"type":"session_meta","id":"session-123"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := transcriptSessionIDFromPath(path); got != "session-123" {
		t.Fatalf("session id=%q", got)
	}
}

// Codex launched by web-console writes rollouts under a per-session CODEX_HOME,
// outside the default ~/.codex root. Discovery must find those homes or every
// web-console effort stays absent from the evidence corpus.
func TestDiscoverWebConsoleCodexRootsFindsIsolatedHomes(t *testing.T) {
	home := t.TempDir()
	base := webConsoleCodexSessionsBase(home)
	const sessionID = "40e51337-isolated-home"
	sessionsDir := filepath.Join(base, sessionID, "sessions", "2026", "10", "21")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A stray non-directory entry beside the session homes must not become a root.
	if err := os.WriteFile(filepath.Join(base, "README"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	roots := (&TranscriptImportScheduler{}).discoverWebConsoleCodexRoots(home)
	if len(roots) != 1 {
		t.Fatalf("roots = %d, want exactly one isolated codex home", len(roots))
	}
	got := roots[0]
	if got.webConsoleSessionID != sessionID {
		t.Fatalf("session id = %q, want %q", got.webConsoleSessionID, sessionID)
	}
	if got.harness != "codex" || got.runner != domain.RunnerTypeCodex {
		t.Fatalf("harness/runner = %q/%q", got.harness, got.runner)
	}
	if want := filepath.Join(base, sessionID, "sessions"); got.path != want {
		t.Fatalf("path = %q, want %q", got.path, want)
	}
}

// A missing base means no web-console activity, which must be a quiet no-op and
// never an error that aborts the sweep.
func TestDiscoverWebConsoleCodexRootsIsEmptyWithoutBase(t *testing.T) {
	if roots := (&TranscriptImportScheduler{}).discoverWebConsoleCodexRoots(t.TempDir()); len(roots) != 0 {
		t.Fatalf("roots = %d, want 0 when no web-console base exists", len(roots))
	}
}

type webConsoleRunsStub struct {
	repository.RunRepository
	runs []*domain.Run
	err  error
}

func (s webConsoleRunsStub) ListByWebConsoleSessionIDs(context.Context, []string) ([]*domain.Run, error) {
	return s.runs, s.err
}

// A live/managed run owns the admission-path capture, so its session must be
// marked managed (skip import) to avoid a duplicate. A prior sweep's imported
// run must NOT be marked managed, or later rollout segments of the same session
// are lost. A lookup error marks every candidate managed so the conservative
// outcome is a retried skip, never a duplicate. The lookup is batched: one call
// resolves the whole candidate set.
func TestManagedWebConsoleSessions(t *testing.T) {
	stub := webConsoleRunsStub{runs: []*domain.Run{
		{ExecutionMode: domain.ExecutionModeAttached, Status: domain.RunStatusRunning, WebConsoleSessionID: "live"},
		{ExecutionMode: domain.ExecutionModeAttached, Status: domain.RunStatusNeedsReview, WebConsoleSessionID: "dead-stub"},
		{ExecutionMode: domain.ExecutionModeImported, Status: domain.RunStatusComplete, WebConsoleSessionID: "imported-only"},
	}}
	s := &TranscriptImportScheduler{orchestrator: &Orchestrator{runs: stub}}
	got := s.managedWebConsoleSessions(context.Background(), []string{"live", "dead-stub", "imported-only", "never-seen"})
	if !got["live"] {
		t.Fatalf("a live (running) attached session must be managed: %#v", got)
	}
	if got["dead-stub"] {
		t.Fatal("a terminal attach stub must NOT block importing the real transcript")
	}
	if got["imported-only"] {
		t.Fatal("a session with only an imported run must not be managed (segments must still import)")
	}
	if got["never-seen"] {
		t.Fatal("a session with no run must not be managed")
	}
}

// A lookup failure must mark every candidate managed so the conservative outcome
// is a skipped import the next sweep retries, never a duplicated run.
func TestManagedWebConsoleSessionsTreatsLookupErrorAsManaged(t *testing.T) {
	s := &TranscriptImportScheduler{orchestrator: &Orchestrator{runs: webConsoleRunsStub{err: context.DeadlineExceeded}}}
	got := s.managedWebConsoleSessions(context.Background(), []string{"a", "b"})
	if !got["a"] || !got["b"] {
		t.Fatalf("lookup error must mark all candidates managed: %#v", got)
	}
}

// The import budget is env-overridable; an unset env uses the bounded default so
// the first post-discovery sweep never tries to ingest the whole backlog.
func TestWebConsoleImportBudget(t *testing.T) {
	t.Setenv("AGENT_MANAGER_SWEEP_MAX_WEBCONSOLE_IMPORTS", "")
	if got := webConsoleImportBudget(); got != defaultWebConsoleImportBudget {
		t.Fatalf("default budget = %d, want %d", got, defaultWebConsoleImportBudget)
	}
	t.Setenv("AGENT_MANAGER_SWEEP_MAX_WEBCONSOLE_IMPORTS", "7")
	if got := webConsoleImportBudget(); got != 7 {
		t.Fatalf("env budget = %d, want 7", got)
	}
	t.Setenv("AGENT_MANAGER_SWEEP_MAX_WEBCONSOLE_IMPORTS", "0")
	if got := webConsoleImportBudget(); got != 0 {
		t.Fatalf("budget = %d, want 0 (unbounded)", got)
	}
}
