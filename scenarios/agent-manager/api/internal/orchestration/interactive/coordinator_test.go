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
	mu                  sync.Mutex
	updates             int
	err                 error
	saved               *domain.Run
	onGet               func(*fakeRunStore)
	onUpdate            func(*domain.Run)
	beforeBegin         func(run, saved *domain.Run)
	rejectStaleTerminal bool
}

func (s *fakeRunStore) Update(_ context.Context, run *domain.Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates++
	if s.rejectStaleTerminal && s.saved != nil && run.Status.IsTerminal() &&
		s.saved.InteractiveInvocationStartedAt != nil && run.InteractiveInvocationStartedAt != nil &&
		!s.saved.InteractiveInvocationStartedAt.Equal(*run.InteractiveInvocationStartedAt) {
		return domain.NewStateErrorWithID("Run", run.ID.String(), "invocation_changed", "update", "interactive invocation changed; stale observer refused")
	}
	if s.onUpdate != nil {
		s.onUpdate(run)
	}
	copy := *run
	s.saved = &copy
	return s.err
}

func (s *fakeRunStore) BeginInteractiveInvocation(_ context.Context, run *domain.Run, expectedStatus domain.RunStatus, expectedStartedAt *time.Time, at time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.beforeBegin != nil {
		s.beforeBegin(run, s.saved)
	}
	if s.saved != nil && (s.saved.Status != expectedStatus || (expectedStartedAt == nil) != (s.saved.InteractiveInvocationStartedAt == nil) ||
		(expectedStartedAt != nil && !s.saved.InteractiveInvocationStartedAt.Equal(*expectedStartedAt))) {
		return false, nil
	}
	stamp := at
	run.InteractiveInvocationStartedAt = &stamp
	run.LifecycleVersion++
	copy := *run
	s.saved = &copy
	return true, nil
}

func TestFinalizeYieldsWhenInvocationChangesAfterTerminalObservation(t *testing.T) {
	startedAt := time.Now().Add(-time.Minute).UTC().Truncate(time.Microsecond)
	newer := startedAt.Add(time.Second)
	run := newInteractiveRun(domain.RunnerTypeCodex, "retained-rollout.jsonl")
	run.TaskID = uuid.New()
	run.SessionID = "native-session"
	run.WebConsoleSessionID = "web-console-session"
	run.InteractiveInvocationStartedAt = &startedAt
	saved := *run
	saved.InteractiveInvocationStartedAt = &newer
	store := &fakeRunStore{saved: &saved, rejectStaleTerminal: true}
	coord := NewCoordinator(CoordinatorDeps{Runs: store})
	terminal := &runner.TranscriptTerminal{Success: true, ExitCode: 0, TerminalReason: "goal_complete"}
	err := coord.Finalize(context.Background(), run, terminal, nil)
	if !errors.Is(err, ErrRunOwnerChanged) {
		t.Fatalf("stale finalization error = %v, want owner-changed yield", err)
	}
	if run.Status != domain.RunStatusRunning || run.InteractiveInvocationStartedAt == nil || !run.InteractiveInvocationStartedAt.Equal(newer) {
		t.Fatalf("stale observer did not refresh current running owner: status=%s invocation=%v", run.Status, run.InteractiveInvocationStartedAt)
	}
	if store.saved.Status != domain.RunStatusRunning {
		t.Fatalf("terminal write escaped the invocation compare-and-set: stored status=%s", store.saved.Status)
	}
}

func (s *fakeRunStore) Get(_ context.Context, id uuid.UUID) (*domain.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.onGet != nil {
		s.onGet(s)
	}
	if s.saved == nil || s.saved.ID != id {
		return nil, nil
	}
	copy := *s.saved
	return &copy, nil
}

func (s *fakeRunStore) UpdateRunnerStreamState(ctx context.Context, run *domain.Run) (bool, error) {
	if err := s.Update(ctx, run); err != nil {
		return false, err
	}
	return true, nil
}

// ctxSensitiveRunStore fails any write made with an already-cancelled or
// expired context, modelling a real store. It proves terminal persistence on
// the timeout path uses a fresh context.
type ctxSensitiveRunStore struct {
	mu      sync.Mutex
	updates int
}

func (s *ctxSensitiveRunStore) Update(ctx context.Context, _ *domain.Run) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates++
	return nil
}

func (s *ctxSensitiveRunStore) BeginInteractiveInvocation(ctx context.Context, run *domain.Run, _ domain.RunStatus, _ *time.Time, at time.Time) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	stamp := at
	run.InteractiveInvocationStartedAt = &stamp
	return true, nil
}

func (s *ctxSensitiveRunStore) Get(context.Context, uuid.UUID) (*domain.Run, error) {
	return nil, nil
}

func (s *ctxSensitiveRunStore) UpdateRunnerStreamState(ctx context.Context, run *domain.Run) (bool, error) {
	if err := s.Update(ctx, run); err != nil {
		return false, err
	}
	return true, nil
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
		// Keep the session-reattach window short so a genuinely-gone session
		// fails the run promptly in tests.
		SessionReattachWindow: 60 * time.Millisecond,
		Heartbeat:             -1,
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

func TestLiveCoordinatorRefreshesHeartbeatDuringQuietTail(t *testing.T) {
	transcript := filepath.Join(t.TempDir(), "quiet.jsonl")
	if err := os.WriteFile(transcript, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	run := newInteractiveRun(domain.RunnerTypeClaudeCode, transcript)
	run.WebConsoleSessionID = "sess-quiet"
	initialHeartbeat := time.Now().Add(-time.Hour)
	run.LastHeartbeat = &initialHeartbeat
	store := &fakeRunStore{}
	heartbeats := make(chan time.Time, 8)
	store.onUpdate = func(updated *domain.Run) {
		if updated.LastHeartbeat != nil && updated.LastHeartbeat.After(initialHeartbeat) {
			select {
			case heartbeats <- *updated.LastHeartbeat:
			default:
			}
		}
	}
	sink := &collectSink{}
	coord := NewCoordinator(CoordinatorDeps{
		Tailer:                NewTailer(codecParserResolver, WithTailPollInterval(5*time.Millisecond)),
		Sessions:              newFakeSessions("sess-quiet"),
		Runs:                  store,
		NewSink:               func(uuid.UUID) runner.EventSink { return sink },
		Heartbeat:             10 * time.Millisecond,
		Debounce:              30 * time.Millisecond,
		ActivityPoll:          5 * time.Millisecond,
		SessionPoll:           10 * time.Millisecond,
		SessionReattachWindow: time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := coord.TailLiveToCompletionWithOptions(ctx, run, TailContext{}, TailCompletionOptions{})
		done <- err
	}()

	select {
	case <-heartbeats:
		// Heartbeat is an observation of the live coordinator, not evidence of
		// provider progress. The tail is still waiting for a terminal rollout.
	case err := <-done:
		t.Fatalf("live tail returned before heartbeat refresh: %v", err)
	case <-ctx.Done():
		t.Fatal("live tail did not refresh its heartbeat while waiting")
	}

	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("live tail error = %v, want cancellation", err)
	}

	store.mu.Lock()
	updatesAfterReturn := store.updates
	store.mu.Unlock()
	draining := true
	for draining {
		select {
		case <-heartbeats:
		default:
			draining = false
		}
	}
	select {
	case <-heartbeats:
		t.Fatal("heartbeat continued after the live tail returned")
	case <-time.After(3 * coord.heartbeat):
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.updates != updatesAfterReturn {
		t.Fatalf("run writes after live tail returned: before=%d after=%d", updatesAfterReturn, store.updates)
	}
}

func TestCoordinatorPersistsCodexRotationPathAndFileCursor(t *testing.T) {
	runDir := t.TempDir()
	sessionsDir := filepath.Join(runDir, "codex", "sessions", "2026", "07", "13")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	fixture := readCodecFixture(t, "codex_rollout_trace.jsonl")
	head := fixture[:len(fixture)-2]
	rolloutA := filepath.Join(sessionsDir, "rollout-2026-07-13T10-00-00-aaaa.jsonl")
	headBytes := []byte(strings.Join(head, "\n") + "\n")
	if err := os.WriteFile(rolloutA, headBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Minute)
	if err := os.Chtimes(rolloutA, old, old); err != nil {
		t.Fatal(err)
	}

	run := newInteractiveRun(domain.RunnerTypeCodex, rolloutA)
	run.WebConsoleSessionID = "sess-context"
	sink := &collectSink{}
	coord, store, _ := newTestCoordinator(t, newFakeSessions("sess-context"), sink)
	var persisted []struct {
		path   string
		cursor int64
	}
	store.onUpdate = func(updated *domain.Run) {
		persisted = append(persisted, struct {
			path   string
			cursor int64
		}{updated.TranscriptPath, updated.TranscriptCursor})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := coord.TailLiveToCompletionWithOptions(ctx, run, TailContext{RunDir: runDir}, TailCompletionOptions{})
		done <- err
	}()

	waitFor(t, time.Second, func() bool { return run.TranscriptCursor == int64(len(headBytes)) })
	rolloutB := filepath.Join(sessionsDir, "rollout-2026-07-13T11-00-00-bbbb.jsonl")
	fullBytes := []byte(strings.Join(fixture, "\n") + "\n")
	if err := os.WriteFile(rolloutB, fullBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("tail rotated Codex run: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("tail did not finish after rotation: %v (messages=%d path=%q cursor=%d)", ctx.Err(), sink.count(domain.EventTypeMessage), run.TranscriptPath, run.TranscriptCursor)
	}
	if run.TranscriptPath != rolloutB {
		t.Fatalf("persisted transcript path = %q, want rotated path %q", run.TranscriptPath, rolloutB)
	}
	if run.TranscriptCursor != int64(len(fullBytes)) {
		t.Fatalf("rotated file cursor = %d, want file-local offset %d", run.TranscriptCursor, len(fullBytes))
	}
	if got := sink.count(domain.EventTypeMessage); got != 2 {
		t.Fatalf("message events after one rotation = %d, want 2 (one from each distinct rollout)", got)
	}
	resetPersisted := false
	for _, row := range persisted {
		if row.path == rolloutB && row.cursor == 0 {
			resetPersisted = true
			break
		}
	}
	if !resetPersisted {
		t.Fatalf("path/cursor reset was not persisted atomically before reading rotated file: %+v", persisted)
	}

	// A later coordinator starts from the selected file's persisted offset, not
	// the original seed path or the seed's larger offset. Appending one turn
	// makes that restart boundary observable without relying on provider state.
	file, err := os.OpenFile(rolloutB, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(fullBytes); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	restartedSink := &collectSink{}
	restarted, _, _ := newTestCoordinator(t, newFakeSessions("sess-context"), restartedSink)
	restartCtx, stopRestart := context.WithTimeout(context.Background(), time.Second)
	defer stopRestart()
	if _, err := restarted.TailToCompletionWithContext(restartCtx, run, TailContext{RunDir: runDir}, TailCompletionOptions{}); err != nil {
		t.Fatalf("reattach after rotated rollout: %v", err)
	}
	if run.TranscriptPath != rolloutB || run.TranscriptCursor != int64(2*len(fullBytes)) {
		t.Fatalf("reattached path/cursor = %q/%d, want rotated file at %d", run.TranscriptPath, run.TranscriptCursor, 2*len(fullBytes))
	}
	if got := restartedSink.count(domain.EventTypeMessage); got != 2 {
		t.Fatalf("reattached message events = %d, want the appended turn's user and assistant messages", got)
	}
}

func TestLiveTailConsumesNewInvocationInsteadOfBufferedGoalTerminal(t *testing.T) {
	cases := []struct {
		name    string
		initial []string
	}{
		{name: "native goal terminal", initial: readCodecFixture(t, "codex_goal_accepted.jsonl")},
		{name: "failed turn terminal", initial: []string{
			`{"type":"turn_context","payload":{"turn_id":"old-failed-turn"}}`,
			`{"type":"event_msg","payload":{"type":"turn_aborted","turn_id":"old-failed-turn","reason":"old provider failure"}}`,
		}},
		{name: "idle turn boundary", initial: readCodecFixture(t, "codex_rollout_trace.jsonl")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runDir := t.TempDir()
			sessionDir := filepath.Join(runDir, "codex", "sessions", "2026", "07", "13")
			if err := os.MkdirAll(sessionDir, 0o755); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(sessionDir, "rollout-2026-07-13T10-00-00-live.jsonl")
			if err := os.WriteFile(path, []byte(strings.Join(tc.initial, "\n")+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			startedAt := time.Now().Add(-time.Minute).UTC().Truncate(time.Microsecond)
			run := newInteractiveRun(domain.RunnerTypeCodex, path)
			run.TaskID = uuid.New()
			run.SessionID = "fixture-thread"
			run.WebConsoleSessionID = "wc-live-invocation"
			run.InteractiveInvocationStartedAt = &startedAt
			saved := *run
			fenceVisible := make(chan struct{})
			var fenced bool
			store := &fakeRunStore{saved: &saved}
			store.onGet = func(store *fakeRunStore) {
				if fenced {
					return
				}
				fenced = true
				newer := startedAt.Add(time.Second)
				store.saved.InteractiveInvocationStartedAt = &newer
				close(fenceVisible)
			}
			sink := &collectSink{}
			coord := NewCoordinator(CoordinatorDeps{
				Tailer:                NewTailer(codecParserResolver, WithTailPollInterval(5*time.Millisecond)),
				Sessions:              newFakeSessions("wc-live-invocation"),
				Runs:                  store,
				NewSink:               func(uuid.UUID) runner.EventSink { return sink },
				Debounce:              40 * time.Millisecond,
				ActivityPoll:          5 * time.Millisecond,
				SessionPoll:           5 * time.Millisecond,
				SessionReattachWindow: 40 * time.Millisecond,
				Heartbeat:             -1,
			})
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() {
				done <- coord.TailAndFinalizeLive(ctx, run, TailContext{RunDir: runDir}, TailCompletionOptions{})
			}()
			select {
			case <-fenceVisible:
			case <-ctx.Done():
				t.Fatalf("old goal terminal was not inspected before the fence: %v", ctx.Err())
			}
			oldMessages := sink.count(domain.EventTypeMessage)
			newTurn := []string{
				`{"type":"turn_context","payload":{"turn_id":"continued-live-turn"}}`,
				`{"type":"event_msg","payload":{"type":"user_message","message":"accepted continuation"}}`,
				`{"type":"event_msg","payload":{"type":"agent_message","message":"continued response"}}`,
				`{"type":"event_msg","payload":{"type":"task_complete","turn_id":"continued-live-turn"}}`,
			}
			file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := file.WriteString(strings.Join(newTurn, "\n") + "\n"); err != nil {
				_ = file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("tail/finalize live continuation: %v", err)
				}
			case <-ctx.Done():
				t.Fatalf("live observer did not consume the response after its newer invocation fence: %v", ctx.Err())
			}
			if run.Status != domain.RunStatusComplete || run.InteractiveInvocationStartedAt == nil || !run.InteractiveInvocationStartedAt.After(startedAt) {
				t.Fatalf("run finalized without retaining the current invocation: status=%s invocation=%v", run.Status, run.InteractiveInvocationStartedAt)
			}
			if got := sink.count(domain.EventTypeMessage); got <= oldMessages {
				t.Fatalf("tail did not consume the later response: messages before=%d after=%d", oldMessages, got)
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

	terminal, err := coord.TailToCompletionWithContext(ctx, run, TailContext{RunDir: dir}, TailCompletionOptions{})
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

// TestCoordinatorSessionGoneWindowAllowsPersistentRecovery documents the
// failure mode found in production: a persistent PTY can remain alive while
// Web Console temporarily has no session row during a managed restart. The
// watcher must not declare loss during that re-resolution window.
func TestCoordinatorSessionGoneWindowAllowsPersistentRecovery(t *testing.T) {
	run := newInteractiveRun(domain.RunnerTypeCodex, "")
	run.WebConsoleSessionID = "sess-recovering"
	sess := newFakeSessions("sess-recovering")
	sess.deleted["sess-recovering"] = true

	coord := NewCoordinator(CoordinatorDeps{
		Sessions:              sess,
		SessionPoll:           5 * time.Millisecond,
		Heartbeat:             -1,
		SessionReattachWindow: 40 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	gone := coord.watchSession(ctx, ctx, cancel, run)
	time.Sleep(15 * time.Millisecond)
	sess.mu.Lock()
	sess.deleted["sess-recovering"] = false
	sess.mu.Unlock()
	time.Sleep(35 * time.Millisecond)
	if gone() {
		t.Fatal("session was declared gone after it reappeared within the reattach window")
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
		{"timeout", nil, context.DeadlineExceeded, domain.RunStatusFailed, "exceeded its configured timeout"},
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

// TestCoordinatorTimeoutPersistsOnExpiredContext proves the timeout terminal is
// written even though the tail context is already expired when it fires — the
// live Phase 2 smoke found the run stuck RUNNING because the write inherited the
// expired deadline.
func TestCoordinatorTimeoutPersistsOnExpiredContext(t *testing.T) {
	run := newInteractiveRun(domain.RunnerTypeClaudeCode, "")
	run.WebConsoleSessionID = "s1"
	store := &ctxSensitiveRunStore{}
	coord := NewCoordinator(CoordinatorDeps{
		Runs:      store,
		Heartbeat: -1,
	})
	expired, cancel := context.WithCancel(context.Background())
	cancel()
	if err := coord.Finalize(expired, run, nil, context.DeadlineExceeded); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if run.Status != domain.RunStatusFailed {
		t.Fatalf("status = %s, want failed", run.Status)
	}
	if run.TerminalClass != domain.RunTerminalClassInterruption || run.StopReason != domain.RunStopReasonTimeout {
		t.Fatalf("terminal class/reason = %q/%q, want interruption/timeout", run.TerminalClass, run.StopReason)
	}
	if store.updates == 0 {
		t.Fatal("timeout terminal was not persisted on a fresh context")
	}
}

// TestCoordinatorClassifiesSessionLost proves a vanished session with no
// retained terminal is a session_lost interruption, not a bare failure.
func TestCoordinatorClassifiesSessionLost(t *testing.T) {
	run := newInteractiveRun(domain.RunnerTypeClaudeCode, "")
	run.WebConsoleSessionID = "s1"
	coord, _, _ := newTestCoordinator(t, newFakeSessions("s1"), &collectSink{})
	if err := coord.Finalize(context.Background(), run, nil, ErrSessionGone); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if run.TerminalClass != domain.RunTerminalClassInterruption || run.StopReason != domain.RunStopReasonSessionLost {
		t.Fatalf("class/reason = %q/%q, want interruption/session_lost", run.TerminalClass, run.StopReason)
	}
}

// TestCoordinatorStructuredResultSatisfiedUsesOnlyDeterministicSuccess proves
// that the goal-session completion fallback trusts the canonical structured
// result projection, while refusing constrained extraction as a tail terminal.
func TestCoordinatorStructuredResultSatisfiedUsesOnlyDeterministicSuccess(t *testing.T) {
	run := newInteractiveRun(domain.RunnerTypeCodex, "")
	run.ResolvedConfig.ResultSpec = &domain.ResultSpec{
		Kind:           domain.ResultSpecKindJSONSchema,
		Schema:         []byte(`{"type":"object"}`),
		ExtractionMode: domain.StructuredExtractionDeterministic,
	}
	coord := NewCoordinator(CoordinatorDeps{
		Result: func(context.Context, uuid.UUID, bool, int, string) (*domain.RunResult, *domain.RunSummary) {
			return &domain.RunResult{Structured: &domain.StructuredResult{Status: domain.StructuredResultSuccess}}, nil
		},
	})
	if !coord.structuredResultSatisfied(context.Background(), run) {
		t.Fatal("deterministic structured success should satisfy the completion rule")
	}

	run.ResolvedConfig.ResultSpec.ExtractionMode = domain.StructuredExtractionConstrained
	if coord.structuredResultSatisfied(context.Background(), run) {
		t.Fatal("constrained extraction must not satisfy the completion rule")
	}
}

// TestCoordinatorTurnCapEndsRun proves an interactive run that reaches its
// configured assistant-turn ceiling returns with the max_turns terminal reason
// instead of tailing forever.
func TestCoordinatorTurnCapEndsRun(t *testing.T) {
	path := writeFixtureFile(t, t.TempDir(), "claude_ondisk_trace.jsonl")
	run := newInteractiveRun(domain.RunnerTypeClaudeCode, path)
	run.ResolvedConfig.Until = "stop when the plan is complete"
	run.ResolvedConfig.MaxTurns = 1
	sess := newFakeSessions("sess-1")
	run.WebConsoleSessionID = "sess-1"
	sink := &collectSink{}
	coord, _, _ := newTestCoordinator(t, sess, sink)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	terminal, err := coord.TailToCompletion(ctx, run)
	if err != nil {
		t.Fatalf("TailToCompletion: %v", err)
	}
	if terminal == nil {
		t.Fatal("expected a terminal at the turn ceiling")
	}
	if terminal.TerminalReason != terminalReasonMaxTurns {
		t.Fatalf("terminal reason = %q, want %q", terminal.TerminalReason, terminalReasonMaxTurns)
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

func TestCoordinatorRetainsOldTranscriptWhenRotationPersistenceFails(t *testing.T) {
	runDir := t.TempDir()
	sessionsDir := filepath.Join(runDir, "codex", "sessions", "2026", "07", "13")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	fixture := readCodecFixture(t, "codex_rollout_trace.jsonl")
	head := fixture[:len(fixture)-1]
	rolloutA := filepath.Join(sessionsDir, "rollout-2026-07-13T10-00-00-aaaa.jsonl")
	headBytes := []byte(strings.Join(head, "\n") + "\n")
	if err := os.WriteFile(rolloutA, headBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Minute)
	if err := os.Chtimes(rolloutA, old, old); err != nil {
		t.Fatal(err)
	}
	rolloutB := filepath.Join(sessionsDir, "rollout-2026-07-13T11-00-00-bbbb.jsonl")
	if err := os.WriteFile(rolloutB, []byte(strings.Join(fixture, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldProof := errors.New("rotated path write refused")
	store := &fakeRunStore{}
	store.onUpdate = func(updated *domain.Run) {
		if updated.TranscriptPath == rolloutB && updated.TranscriptCursor == 0 {
			store.err = oldProof
		}
	}
	sink := &collectSink{}
	coord := NewCoordinator(CoordinatorDeps{
		Tailer:                NewTailer(codecParserResolver, WithTailPollInterval(5*time.Millisecond)),
		Sessions:              newFakeSessions("sess-context"),
		Runs:                  store,
		NewSink:               func(uuid.UUID) runner.EventSink { return sink },
		Debounce:              30 * time.Millisecond,
		ActivityPoll:          5 * time.Millisecond,
		SessionPoll:           10 * time.Millisecond,
		SessionReattachWindow: time.Second,
	})
	run := newInteractiveRun(domain.RunnerTypeCodex, rolloutA)
	run.WebConsoleSessionID = "sess-context"
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := coord.TailToCompletionWithContext(ctx, run, TailContext{RunDir: runDir}, TailCompletionOptions{})
	if !errors.Is(err, oldProof) {
		t.Fatalf("tail error = %v, want path persistence refusal", err)
	}
	if run.TranscriptPath != rolloutA || run.TranscriptCursor != int64(len(headBytes)) {
		t.Fatalf("in-memory owner path/cursor = %q/%d, want prior proof %q/%d", run.TranscriptPath, run.TranscriptCursor, rolloutA, len(headBytes))
	}
	if got := sink.count(domain.EventTypeMessage); got != 1 {
		t.Fatalf("message events after failed path write = %d, want only seed rollout output", got)
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

func TestCoordinatorReservedBindingRefusesPreSubmitPersistenceAndRepeat(t *testing.T) {
	for _, tc := range []struct {
		name       string
		existing   string
		persistErr error
	}{
		{"write failure", "", errors.New("owner write refused")}, {"existing identity", uuid.NewString(), nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sessions := newFakeSessions("unrelated")
			sub := NewSubstrate(sessions, func(domain.RunnerType) (runner.AgentLaunchInfo, error) { return fakeLaunchInfo{}, nil })
			store := &fakeRunStore{err: tc.persistErr}
			c := NewCoordinator(CoordinatorDeps{Substrate: sub, Runs: store})
			run := newInteractiveRun(domain.RunnerTypeCodex, "")
			run.WebConsoleSessionID = tc.existing
			if err := c.Execute(t.Context(), run, LaunchParams{RunID: run.ID, RunnerType: domain.RunnerTypeCodex}, nil); err == nil {
				t.Fatal("launch admitted without owner persistence")
			}
			if len(sessions.calls) != 0 || run.WebConsoleSessionID != tc.existing {
				t.Fatalf("effects=%v binding=%s", sessions.calls, run.WebConsoleSessionID)
			}
		})
	}
}

func TestCoordinatorInitialInvocationCASFailureStopsOnlyItsStillOwnedSession(t *testing.T) {
	for _, tc := range []struct {
		name        string
		advance     func(run, saved *domain.Run)
		wantCleanup bool
	}{
		{
			name: "cancelled exact owner",
			advance: func(_ *domain.Run, saved *domain.Run) {
				saved.Status = domain.RunStatusCancelled
			},
			wantCleanup: true,
		},
		{
			name: "newer session and invocation",
			advance: func(_ *domain.Run, saved *domain.Run) {
				saved.WebConsoleSessionID = "newer-session"
				stamp := time.Now().Add(time.Second)
				saved.InteractiveInvocationStartedAt = &stamp
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runDir := t.TempDir()
			homeDir := t.TempDir()
			workingDir := t.TempDir()
			rollout := filepath.Join(homeDir, ".claude", "projects", SlugifyCwd(workingDir), "rollout-initial.jsonl")
			writeFile(t, rollout)

			sessions := newFakeSessions("created")
			substrate := NewSubstrate(sessions, fakeResolver(fakeLaunchInfo{
				rt: domain.RunnerTypeClaudeCode, tagKey: "CLAUDE_AGENT_TAG", binary: "/usr/bin/claude",
			}), WithHomeDir(homeDir), WithDiscoveryTimeout(time.Second), WithPollInterval(time.Millisecond), WithPromptBootDelay(0), WithStopGrace(0))
			store := &fakeRunStore{beforeBegin: tc.advance}
			coord := NewCoordinator(CoordinatorDeps{Substrate: substrate, Runs: store})
			started := time.Now().Add(-time.Second)
			run := newInteractiveRun(domain.RunnerTypeClaudeCode, "")
			run.TaskID = uuid.New()
			run.Status = domain.RunStatusStarting
			run.StartedAt = &started
			launchErr := coord.Execute(context.Background(), run, LaunchParams{
				RunID: run.ID, RunnerType: domain.RunnerTypeClaudeCode, Tag: run.Tag,
				WorkingDir: workingDir, RunDir: runDir, Prompt: "perform the task", Config: run.ResolvedConfig,
			}, nil)
			if launchErr == nil || !strings.Contains(launchErr.Error(), "initial interactive invocation owner changed") {
				t.Fatalf("Execute error = %v, want initial owner-change refusal", launchErr)
			}
			calls := sessions.calls
			interrupted, archived := false, false
			for _, call := range calls {
				interrupted = interrupted || call == "interrupt"
				archived = archived || call == "archive"
			}
			if interrupted != tc.wantCleanup || archived != tc.wantCleanup {
				t.Fatalf("session cleanup calls = %v, want cleanup=%v", calls, tc.wantCleanup)
			}
		})
	}
}

func TestCoordinatorDoesNotRemintForAlreadyBoundSessionRetry(t *testing.T) {
	run := newInteractiveRun(domain.RunnerTypeCodex, "")
	run.WebConsoleSessionID = "existing-session"
	store := &fakeRunStore{}
	mintCalls := 0
	coord := NewCoordinator(CoordinatorDeps{
		Substrate: &Substrate{}, Runs: store,
		MintIdentityToken: func(context.Context, *domain.Run, string) (string, error) {
			mintCalls++
			return "replacement-bearer", nil
		},
	})
	err := coord.Execute(context.Background(), run, LaunchParams{}, nil)
	if err == nil || !strings.Contains(err.Error(), "already has a different session binding") {
		t.Fatalf("Execute error = %v, want existing binding refusal", err)
	}
	if mintCalls != 0 || store.updates != 0 || run.WebConsoleSessionID != "existing-session" {
		t.Fatalf("retry changed identity state: mint=%d updates=%d session=%q", mintCalls, store.updates, run.WebConsoleSessionID)
	}
}

func TestCoordinatorPersistsSessionBindingBeforeIdentityMint(t *testing.T) {
	run := newInteractiveRun(domain.RunnerTypeCodex, "")
	var firstPersisted string
	store := &fakeRunStore{onUpdate: func(saved *domain.Run) {
		if firstPersisted == "" {
			firstPersisted = saved.WebConsoleSessionID
		}
	}}
	mintedSession := ""
	coord := NewCoordinator(CoordinatorDeps{
		Substrate: &Substrate{}, Runs: store,
		MintIdentityToken: func(_ context.Context, candidate *domain.Run, sessionID string) (string, error) {
			mintedSession = sessionID
			if candidate.WebConsoleSessionID != sessionID {
				t.Fatalf("mint saw run binding %q, want %q", candidate.WebConsoleSessionID, sessionID)
			}
			return "", errors.New("test stops before launch")
		},
	})
	if err := coord.Execute(context.Background(), run, LaunchParams{}, nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if firstPersisted == "" || firstPersisted != mintedSession || run.WebConsoleSessionID != mintedSession {
		t.Fatalf("reservation/mint ordering: persisted=%q minted=%q run=%q", firstPersisted, mintedSession, run.WebConsoleSessionID)
	}
}
