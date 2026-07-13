package interactive

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// fakeRunStore records how many times a run was persisted; the run pointer is
// mutated in place by the coordinator, so tests read fields off the run directly.
type fakeRunStore struct {
	mu      sync.Mutex
	updates int
	err     error
}

func (s *fakeRunStore) Update(_ context.Context, _ *domain.Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates++
	return s.err
}

// fakeBroadcaster records the sequence of broadcast run statuses.
type fakeBroadcaster struct {
	mu       sync.Mutex
	statuses []domain.RunStatus
}

func (b *fakeBroadcaster) BroadcastRunStatus(run *domain.Run) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.statuses = append(b.statuses, run.Status)
}

func newInteractiveRun(rt domain.RunnerType, transcriptPath string) *domain.Run {
	cfg := domain.DefaultRunConfig()
	cfg.RunnerType = rt
	return &domain.Run{
		ID:             uuid.New(),
		Tag:            uuid.NewString(),
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusRunning,
		Phase:          domain.RunPhaseExecuting,
		ExecutionMode:  domain.ExecutionModeInteractive,
		TranscriptPath: transcriptPath,
		ResolvedConfig: cfg,
	}
}

// newTestCoordinator builds a recovery-flavoured coordinator (no substrate,
// heartbeat disabled) wired to the real codec parsers, with fast cadences.
func newTestCoordinator(t *testing.T, sessions *fakeSessions, sink runner.EventSink) (*Coordinator, *fakeRunStore, *fakeBroadcaster) {
	t.Helper()
	store := &fakeRunStore{}
	bc := &fakeBroadcaster{}
	coord := NewCoordinator(CoordinatorDeps{
		Tailer:       NewTailer(codecParserResolver, WithTailPollInterval(10*time.Millisecond)),
		Sessions:     sessions,
		Runs:         store,
		Broadcaster:  bc,
		NewSink:      func(uuid.UUID) runner.EventSink { return sink },
		Debounce:     80 * time.Millisecond,
		ActivityPoll: 10 * time.Millisecond,
		SessionPoll:  15 * time.Millisecond,
		Heartbeat:    -1,
	})
	return coord, store, bc
}

func writeFixtureFile(t *testing.T, dir, fixture string) string {
	t.Helper()
	lines := readCodecFixture(t, fixture)
	path := filepath.Join(dir, "transcript.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
	return path
}

// TestCoordinatorCompletesOnTerminalPerAgent proves the coordinator finalizes a
// run Complete once each agent's on-disk terminal marker is tailed, including
// the claude idle-debounce (its end_turn is only a run boundary after the
// transcript stops growing).
func TestCoordinatorCompletesOnTerminalPerAgent(t *testing.T) {
	cases := []struct {
		name       string
		runnerType domain.RunnerType
		fixture    string
	}{
		{"claude", domain.RunnerTypeClaudeCode, "claude_ondisk_trace.jsonl"},
		{"codex", domain.RunnerTypeCodex, "codex_rollout_trace.jsonl"},
		{"grok", domain.RunnerTypeGrok, "grok_updates_trace.jsonl"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			path := writeFixtureFile(t, t.TempDir(), tc.fixture)
			run := newInteractiveRun(tc.runnerType, path)
			sess := newFakeSessions("sess-1")
			run.WebConsoleSessionID = "sess-1"
			sink := &collectSink{}
			coord, store, bc := newTestCoordinator(t, sess, sink)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			start := time.Now()
			terminal, err := coord.TailToCompletion(ctx, run)
			if err != nil {
				t.Fatalf("TailToCompletion: %v", err)
			}
			if terminal == nil || !terminal.Success {
				t.Fatalf("expected success terminal, got %+v", terminal)
			}
			// The debounce must have been observed (completion is not instant).
			if elapsed := time.Since(start); elapsed < 60*time.Millisecond {
				t.Errorf("completed in %v, expected to observe the idle-debounce window", elapsed)
			}
			if err := coord.Finalize(ctx, run, terminal, err); err != nil {
				t.Fatalf("Finalize: %v", err)
			}

			if run.Status != domain.RunStatusComplete {
				t.Fatalf("status = %s, want complete", run.Status)
			}
			if run.EndedAt == nil {
				t.Error("EndedAt not set")
			}
			if run.ExitCode == nil || *run.ExitCode != 0 {
				t.Errorf("exit code = %+v, want 0", run.ExitCode)
			}
			if run.TranscriptCursor <= 0 {
				t.Errorf("transcript cursor = %d, want > 0 (cursor should be persisted)", run.TranscriptCursor)
			}
			if sink.count(domain.EventTypeMessage) == 0 {
				t.Error("no message events emitted")
			}
			if store.updates == 0 {
				t.Error("run was never persisted")
			}
			if len(bc.statuses) == 0 || bc.statuses[len(bc.statuses)-1] != domain.RunStatusComplete {
				t.Errorf("expected a terminal complete broadcast, got %v", bc.statuses)
			}
		})
	}
}

// TestCoordinatorClaudeIdleDebounceSpansTurns proves a claude end_turn is treated
// as a TURN boundary, not a run boundary: when the transcript grows (a second
// turn) within the debounce window, the coordinator keeps tailing and only
// completes after the final turn goes idle — consuming BOTH turns' events.
func TestCoordinatorClaudeIdleDebounceSpansTurns(t *testing.T) {
	turn := readCodecFixture(t, "claude_ondisk_trace.jsonl")
	perTurnMessages := 0 // filled after we know the codec's message yield

	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(turn, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("seed turn 1: %v", err)
	}

	run := newInteractiveRun(domain.RunnerTypeClaudeCode, path)
	sess := newFakeSessions("sess-1")
	run.WebConsoleSessionID = "sess-1"
	sink := &collectSink{}
	// A generous debounce so appending turn 2 lands inside the idle window.
	store := &fakeRunStore{}
	coord := NewCoordinator(CoordinatorDeps{
		Tailer:       NewTailer(codecParserResolver, WithTailPollInterval(10*time.Millisecond)),
		Sessions:     sess,
		Runs:         store,
		NewSink:      func(uuid.UUID) runner.EventSink { return sink },
		Debounce:     400 * time.Millisecond,
		ActivityPoll: 10 * time.Millisecond,
		SessionPoll:  20 * time.Millisecond,
		Heartbeat:    -1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	done := make(chan *runner.TranscriptTerminal, 1)
	go func() {
		term, err := coord.TailToCompletion(ctx, run)
		if err != nil {
			t.Errorf("TailToCompletion: %v", err)
		}
		done <- term
	}()

	// Wait until turn 1 has been consumed (the run is at its first turn boundary
	// and now in the debounce window), then append turn 2.
	waitFor(t, 3*time.Second, func() bool {
		return sink.count(domain.EventTypeMessage) > 0
	})
	perTurnMessages = sink.count(domain.EventTypeMessage)
	// Give the loop a beat to enter the debounce, then grow the transcript.
	time.Sleep(40 * time.Millisecond)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatalf("append turn 2: %v", err)
	}
	if _, err := f.WriteString(strings.Join(turn, "\n") + "\n"); err != nil {
		t.Fatalf("write turn 2: %v", err)
	}
	_ = f.Close()

	select {
	case term := <-done:
		if term == nil || !term.Success {
			t.Fatalf("expected success terminal after both turns, got %+v", term)
		}
	case <-ctx.Done():
		t.Fatalf("coordinator did not complete: %v", ctx.Err())
	}

	// Both turns must have been consumed — proof the run did not finalize at the
	// first end_turn.
	if got := sink.count(domain.EventTypeMessage); got <= perTurnMessages {
		t.Fatalf("message count = %d, want > %d (both turns consumed across the boundary)", got, perTurnMessages)
	}
}

// TestCoordinatorSessionGoneMidTailFails proves that when the web-console session
// disappears while tailing a run that has no terminal marker yet, the coordinator
// finalizes the run Failed with an explicit reason rather than tailing forever.
func TestCoordinatorSessionGoneMidTailFails(t *testing.T) {
	// A partial codex rollout with NO terminal marker.
	full := readCodecFixture(t, "codex_rollout_trace.jsonl")
	partial := full[:len(full)-2]
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(partial, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write partial: %v", err)
	}

	run := newInteractiveRun(domain.RunnerTypeCodex, path)
	sess := newFakeSessions("sess-gone")
	run.WebConsoleSessionID = "sess-gone"
	// Session is already gone: the watcher will observe NotFound and cancel.
	sess.deleted["sess-gone"] = true

	sink := &collectSink{}
	coord, _, _ := newTestCoordinator(t, sess, sink)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	terminal, err := coord.TailToCompletion(ctx, run)
	if !errors.Is(err, ErrSessionGone) {
		t.Fatalf("expected ErrSessionGone, got %v", err)
	}
	if err := coord.Finalize(ctx, run, terminal, err); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if run.Status != domain.RunStatusFailed {
		t.Fatalf("status = %s, want failed", run.Status)
	}
	if !strings.Contains(run.ErrorMsg, "sess-gone") || !strings.Contains(run.ErrorMsg, "no longer exists") {
		t.Errorf("error msg = %q, want it to name the vanished session", run.ErrorMsg)
	}
}

// TestCoordinatorFinalizeSemantics covers the finalize decision table directly:
// success/failure terminals, session-gone with and without a prior terminal, and
// graceful shutdown (which must leave the run Running for restart recovery).
func TestCoordinatorFinalizeSemantics(t *testing.T) {
	success := &runner.TranscriptTerminal{Success: true, ExitCode: 0}
	failure := &runner.TranscriptTerminal{Success: false, ExitCode: 1, ErrorMessage: "boom"}

	cases := []struct {
		name       string
		terminal   *runner.TranscriptTerminal
		tailErr    error
		wantStatus domain.RunStatus
		wantErrSub string
	}{
		{"success", success, nil, domain.RunStatusComplete, ""},
		{"success-then-session-gone", success, ErrSessionGone, domain.RunStatusComplete, ""},
		{"failure", failure, nil, domain.RunStatusFailed, "boom"},
		{"session-gone-no-terminal", nil, ErrSessionGone, domain.RunStatusFailed, "no longer exists"},
		{"graceful-shutdown", nil, context.Canceled, domain.RunStatusRunning, ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			run := newInteractiveRun(domain.RunnerTypeCodex, "")
			run.WebConsoleSessionID = "s1"
			coord, _, _ := newTestCoordinator(t, newFakeSessions("s1"), &collectSink{})
			if err := coord.Finalize(context.Background(), run, tc.terminal, tc.tailErr); err != nil {
				t.Fatalf("Finalize: %v", err)
			}
			if run.Status != tc.wantStatus {
				t.Fatalf("status = %s, want %s", run.Status, tc.wantStatus)
			}
			if tc.wantErrSub != "" && !strings.Contains(run.ErrorMsg, tc.wantErrSub) {
				t.Errorf("error msg = %q, want substring %q", run.ErrorMsg, tc.wantErrSub)
			}
			if tc.wantStatus == domain.RunStatusRunning && run.EndedAt != nil {
				t.Error("graceful shutdown must not set EndedAt")
			}
		})
	}
}

// TestCoordinatorReattachFromCursorNoDuplicateEvents proves that reattaching a
// tailer from the persisted cursor after a restart replays only NEW transcript
// content — a second turn appended after the first completed — without
// re-emitting the already-consumed first turn.
func TestCoordinatorReattachFromCursorNoDuplicateEvents(t *testing.T) {
	turn := readCodecFixture(t, "codex_rollout_trace.jsonl")
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(turn, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("seed turn 1: %v", err)
	}

	run := newInteractiveRun(domain.RunnerTypeCodex, path)
	sess := newFakeSessions("sess-1")
	run.WebConsoleSessionID = "sess-1"

	// First attach consumes turn 1 to its terminal and persists the cursor.
	sink1 := &collectSink{}
	coord1, _, _ := newTestCoordinator(t, sess, sink1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := coord1.TailToCompletion(ctx, run); err != nil {
		t.Fatalf("first TailToCompletion: %v", err)
	}
	turn1Messages := sink1.count(domain.EventTypeMessage)
	cursorAfterTurn1 := run.TranscriptCursor
	if turn1Messages == 0 || cursorAfterTurn1 <= 0 {
		t.Fatalf("turn 1 not consumed (messages=%d cursor=%d)", turn1Messages, cursorAfterTurn1)
	}

	// Simulate the restart: a fresh coordinator + fresh sink, same run row (its
	// cursor survived). Append turn 2 to the transcript.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatalf("append turn 2: %v", err)
	}
	if _, err := f.WriteString(strings.Join(turn, "\n") + "\n"); err != nil {
		t.Fatalf("write turn 2: %v", err)
	}
	_ = f.Close()

	sink2 := &collectSink{}
	coord2, _, _ := newTestCoordinator(t, sess, sink2)
	if _, err := coord2.TailToCompletion(ctx, run); err != nil {
		t.Fatalf("reattach TailToCompletion: %v", err)
	}

	// The reattach consumed only turn 2 — no re-emission of turn 1.
	if got := sink2.count(domain.EventTypeMessage); got != turn1Messages {
		t.Fatalf("reattach emitted %d messages, want %d (only the new turn, no duplicates)", got, turn1Messages)
	}
	if run.TranscriptCursor <= cursorAfterTurn1 {
		t.Errorf("cursor did not advance across the reattach: %d <= %d", run.TranscriptCursor, cursorAfterTurn1)
	}
}

// waitFor polls cond until it is true or the timeout elapses.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("condition not met within %v", timeout)
}
