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
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// delayedRecoveryRunStore reuses the coordinator suite's Update fake and adds
// only the durable read needed by the timer callback.
type delayedRecoveryRunStore struct {
	*fakeRunStore
	mu  sync.Mutex
	run *domain.Run
	err error
}

func (s *delayedRecoveryRunStore) Get(_ context.Context, id uuid.UUID) (*domain.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return nil, s.err
	}
	if s.run == nil || s.run.ID != id {
		return nil, errors.New("run not found")
	}
	copy := *s.run
	if s.run.ResolvedConfig != nil {
		config := *s.run.ResolvedConfig
		copy.ResolvedConfig = &config
	}
	return &copy, nil
}

// delayedRecoverySessions embeds the existing SessionController fake and
// overrides only the values that must change while a timer is waiting.
type delayedRecoverySessions struct {
	*fakeSessions
	mu                   sync.Mutex
	activeSessionID      string
	screen               string
	screens              []string
	screenCalls          int
	screenObserved       chan struct{}
	sessionErr           error
	screenErr            error
	screenEntered        chan struct{}
	screenRelease        chan struct{}
	commandUnverified    bool
	clearScreenOnCommand bool
	sentCommands         []string
	sentPrompts          []string
}

func (s *delayedRecoverySessions) GetSession(ctx context.Context, id string) (webconsole.SessionInfo, error) {
	s.mu.Lock()
	activeID, err := s.activeSessionID, s.sessionErr
	s.mu.Unlock()
	if err != nil {
		return webconsole.SessionInfo{}, err
	}
	if id != activeID {
		return webconsole.SessionInfo{}, errors.New("wrong session")
	}
	return s.fakeSessions.GetSession(ctx, id)
}

func (s *delayedRecoverySessions) Screen(ctx context.Context, sessionID string, includeScrollback bool) (string, error) {
	if s.screenEntered != nil {
		select {
		case s.screenEntered <- struct{}{}:
		default:
		}
	}
	if s.screenRelease != nil {
		<-s.screenRelease
	}
	s.mu.Lock()
	activeID, err := s.activeSessionID, s.screenErr
	s.screenCalls++
	if len(s.screens) > 0 {
		s.screen = s.screens[0]
		s.screens = s.screens[1:]
	}
	screen := s.screen
	observed := s.screenObserved
	s.mu.Unlock()
	if observed != nil {
		select {
		case observed <- struct{}{}:
		default:
		}
	}
	if sessionID != activeID {
		return "", errors.New("wrong session")
	}
	if err != nil {
		return "", err
	}
	return screen, nil
}

func (s *delayedRecoverySessions) SendCommand(ctx context.Context, sessionID, command, source string) (webconsole.PromptSubmission, error) {
	s.mu.Lock()
	if sessionID != s.activeSessionID {
		s.mu.Unlock()
		return webconsole.PromptSubmission{}, errors.New("wrong session")
	}
	s.sentCommands = append(s.sentCommands, command)
	verified := !s.commandUnverified
	if s.clearScreenOnCommand {
		s.screen = "\u203a Ask Codex to do anything"
		s.screens = nil
	}
	s.mu.Unlock()
	if _, err := s.fakeSessions.SendCommand(ctx, sessionID, command, source); err != nil {
		return webconsole.PromptSubmission{}, err
	}
	return webconsole.PromptSubmission{Verified: verified, EnterPresses: 1}, nil
}

func (s *delayedRecoverySessions) SendPrompt(ctx context.Context, sessionID, prompt, source string) (webconsole.PromptSubmission, error) {
	s.mu.Lock()
	if sessionID != s.activeSessionID {
		s.mu.Unlock()
		return webconsole.PromptSubmission{}, errors.New("wrong session")
	}
	s.sentPrompts = append(s.sentPrompts, prompt)
	s.mu.Unlock()
	return s.fakeSessions.SendPrompt(ctx, sessionID, prompt, source)
}

func newDelayedRecoveryFixture(screen string, delay time.Duration) (*Coordinator, *domain.Run, *delayedRecoveryRunStore, *delayedRecoverySessions) {
	cfg := domain.DefaultRunConfig()
	cfg.RunnerType = domain.RunnerTypeCodex
	cfg.Until = "finish the assigned goal"
	run := &domain.Run{
		ID:                  uuid.New(),
		Status:              domain.RunStatusRunning,
		ExecutionMode:       domain.ExecutionModeInteractive,
		WebConsoleSessionID: "session-current",
		ResolvedConfig:      cfg,
	}
	persisted := *run
	persistedConfig := *cfg
	persisted.ResolvedConfig = &persistedConfig
	store := &delayedRecoveryRunStore{fakeRunStore: &fakeRunStore{}, run: &persisted}
	sessions := &delayedRecoverySessions{
		fakeSessions:    newFakeSessions(run.WebConsoleSessionID),
		activeSessionID: run.WebConsoleSessionID,
		screen:          screen,
	}
	coord := NewCoordinator(CoordinatorDeps{
		Runs:                      store,
		Sessions:                  sessions,
		SessionPoll:               time.Millisecond,
		InterruptionRecoveryDelay: delay,
	})
	return coord, run, store, sessions
}

const stalledGoalScreen = "› Ask Codex to do anything\n\n" +
	"GPT-6-Luna … Goal stalled (/goal resume) │ active-pane status\n" +
	"? for shortcuts                  ⚠ 2 · f2 │ right pane\n\n"

func awaitRecoveryFinished(t *testing.T, c *Coordinator) {
	t.Helper()
	select {
	case <-c.recoveryReady:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for delayed recovery to finish")
	}
}

func awaitRecoveryScheduled(t *testing.T, c *Coordinator) {
	t.Helper()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for !c.recoveryScheduled.Load() {
		select {
		case <-deadline.C:
			t.Fatal("timed out waiting for interruption recovery to be scheduled")
		case <-ticker.C:
		}
	}
}

func awaitScreenReads(t *testing.T, sessions *delayedRecoverySessions, count int) {
	t.Helper()
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	for {
		sessions.mu.Lock()
		observed := sessions.screenCalls >= count
		signal := sessions.screenObserved
		sessions.mu.Unlock()
		if observed {
			return
		}
		if signal == nil {
			t.Fatal("screen-read signal is not configured")
		}
		select {
		case <-signal:
		case <-deadline.C:
			t.Fatalf("timed out waiting for %d screen reads", count)
		}
	}
}

func TestDelayedInterruptionRecoveryRevalidatesRunAndScreen(t *testing.T) {
	tests := []struct {
		name       string
		current    func(*delayedRecoveryRunStore, *delayedRecoverySessions)
		wantResume bool
	}{
		{name: "same active run and stalled screen resumes existing goal", wantResume: true},
		{
			name: "cleared interruption does not send stale resume",
			current: func(_ *delayedRecoveryRunStore, sessions *delayedRecoverySessions) {
				sessions.mu.Lock()
				sessions.screen = "› Ask Codex to do anything\nWorking (3s · esc to interrupt)"
				sessions.mu.Unlock()
			},
		},
		{
			name: "completed run does not resume",
			current: func(store *delayedRecoveryRunStore, _ *delayedRecoverySessions) {
				store.mu.Lock()
				store.run.Status = domain.RunStatusComplete
				store.mu.Unlock()
			},
		},
		{
			name: "cancelled run does not resume",
			current: func(store *delayedRecoveryRunStore, _ *delayedRecoverySessions) {
				store.mu.Lock()
				store.run.Status = domain.RunStatusCancelled
				store.mu.Unlock()
			},
		},
		{
			name: "changed durable session does not resume original session",
			current: func(store *delayedRecoveryRunStore, _ *delayedRecoverySessions) {
				store.mu.Lock()
				store.run.WebConsoleSessionID = "replacement-session"
				store.mu.Unlock()
			},
		},
		{
			name: "missing session does not resume",
			current: func(_ *delayedRecoveryRunStore, sessions *delayedRecoverySessions) {
				sessions.mu.Lock()
				sessions.sessionErr = webconsole.ErrSessionNotFound
				sessions.mu.Unlock()
			},
		},
		{
			name: "run lookup failure does not resume",
			current: func(store *delayedRecoveryRunStore, _ *delayedRecoverySessions) {
				store.mu.Lock()
				store.err = errors.New("temporary store failure")
				store.mu.Unlock()
			},
		},
		{
			name: "goal marker without configured native goal does not send a generic prompt",
			current: func(store *delayedRecoveryRunStore, _ *delayedRecoverySessions) {
				store.mu.Lock()
				store.run.ResolvedConfig.Until = ""
				store.mu.Unlock()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord, run, store, sessions := newDelayedRecoveryFixture(stalledGoalScreen, time.Millisecond)
			if tt.current != nil {
				tt.current(store, sessions)
			}
			interruption, ok := ClassifyScreenInterruption(stalledGoalScreen)
			if !ok {
				t.Fatal("stalled-goal fixture did not classify")
			}
			coord.scheduleInterruptionRecovery(context.Background(), run, interruption)
			awaitRecoveryFinished(t, coord)

			sessions.mu.Lock()
			defer sessions.mu.Unlock()
			if tt.wantResume {
				if len(sessions.sentCommands) != 1 || sessions.sentCommands[0] != "/goal resume" || len(sessions.sentPrompts) != 0 {
					t.Fatalf("recovery sends = commands %q, prompts %q; want one verified native resume command", sessions.sentCommands, sessions.sentPrompts)
				}
				if !coord.recoveryCanReattach.Load() {
					t.Fatal("verified recovery was not reported successful")
				}
				return
			}
			if len(sessions.sentCommands) != 0 || len(sessions.sentPrompts) != 0 {
				t.Fatalf("stale recovery sent commands %q or prompts %q", sessions.sentCommands, sessions.sentPrompts)
			}
		})
	}
}

func TestDelayedInterruptionRecoveryUsesCapturedRunAndSessionIDs(t *testing.T) {
	coord, run, _, sessions := newDelayedRecoveryFixture(stalledGoalScreen, 30*time.Millisecond)
	interruption, ok := ClassifyScreenInterruption(stalledGoalScreen)
	if !ok {
		t.Fatal("stalled-goal fixture did not classify")
	}
	coord.scheduleInterruptionRecovery(context.Background(), run, interruption)
	run.ID = uuid.New()
	run.WebConsoleSessionID = "replacement-session"
	awaitRecoveryFinished(t, coord)

	sessions.mu.Lock()
	defer sessions.mu.Unlock()
	if len(sessions.sentCommands) != 1 || sessions.sentCommands[0] != "/goal resume" {
		t.Fatalf("recovery sends = %q, want one command for captured run/session", sessions.sentCommands)
	}
}

func TestDelayedInterruptionRecoveryRechecksCancellationAfterScreenRead(t *testing.T) {
	coord, run, _, sessions := newDelayedRecoveryFixture(stalledGoalScreen, time.Millisecond)
	parentCtx, cancelParent := context.WithCancel(context.Background())
	defer cancelParent()
	sessions.mu.Lock()
	sessions.screenEntered = make(chan struct{}, 1)
	sessions.screenRelease = make(chan struct{})
	entered, release := sessions.screenEntered, sessions.screenRelease
	sessions.mu.Unlock()
	interruption, ok := ClassifyScreenInterruption(stalledGoalScreen)
	if !ok {
		t.Fatal("stalled-goal fixture did not classify")
	}
	coord.scheduleInterruptionRecovery(parentCtx, run, interruption)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("delayed recovery did not re-read the session screen")
	}
	cancelParent()
	close(release)
	awaitRecoveryFinished(t, coord)

	sessions.mu.Lock()
	defer sessions.mu.Unlock()
	if len(sessions.sentCommands) != 0 || len(sessions.sentPrompts) != 0 {
		t.Fatalf("cancelled run received input after screen read: commands %q prompts %q", sessions.sentCommands, sessions.sentPrompts)
	}
}

func TestTailToCompletionReattachesAfterClearedInterruptionAndResetsForNextCycle(t *testing.T) {
	const sessionID = "fixture-thread"
	lines := readCodecFixture(t, "codex_goal_accepted.jsonl")
	if len(lines) != 4 {
		t.Fatalf("native goal fixture has %d lines, want 4", len(lines))
	}
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	initial := strings.Join(lines[:2], "\n") + "\n"
	if err := os.WriteFile(path, []byte(initial), 0600); err != nil {
		t.Fatal(err)
	}

	run := newInteractiveRun(domain.RunnerTypeCodex, path)
	run.WebConsoleSessionID = sessionID
	run.SessionID = sessionID
	now := time.Now().UTC()
	run.StartedAt = &now
	run.ResolvedConfig.Until = "complete the fixture"
	persisted := *run
	config := *run.ResolvedConfig
	persisted.ResolvedConfig = &config
	store := &delayedRecoveryRunStore{fakeRunStore: &fakeRunStore{}, run: &persisted}
	sessions := &delayedRecoverySessions{
		fakeSessions:         newFakeSessions(sessionID),
		activeSessionID:      sessionID,
		screen:               "› Ask Codex to do anything",
		screens:              []string{stalledGoalScreen, "› Ask Codex to do anything", "› Ask Codex to do anything"},
		screenObserved:       make(chan struct{}, 12),
		clearScreenOnCommand: true,
	}
	coord := NewCoordinator(CoordinatorDeps{
		Tailer:                    NewTailer(codecParserResolver, WithTailPollInterval(2*time.Millisecond)),
		Sessions:                  sessions,
		Runs:                      store,
		SessionPoll:               2 * time.Millisecond,
		InterruptionRecoveryDelay: 5 * time.Millisecond,
		Debounce:                  20 * time.Millisecond,
		Heartbeat:                 -1,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	type outcome struct {
		terminal *runner.TranscriptTerminal
		err      error
	}
	done := make(chan outcome, 1)
	go func() {
		terminal, err := coord.TailToCompletion(ctx, run)
		done <- outcome{terminal: terminal, err: err}
	}()

	// First detection cancels the initial tail. The delayed check sees the
	// cleared screen and sends no input, then the third read proves that tailing
	// resumed. A later interruption must get one fresh recovery cycle.
	awaitScreenReads(t, sessions, 3)
	sessions.mu.Lock()
	initialSends := append([]string(nil), sessions.sentCommands...)
	if len(initialSends) != 0 {
		sessions.mu.Unlock()
		t.Fatalf("cleared interruption received stale input: %q", initialSends)
	}
	sessions.screen = stalledGoalScreen
	sessions.mu.Unlock()
	awaitScreenReads(t, sessions, 5)
	awaitScreenReads(t, sessions, 6)
	sessions.mu.Lock()
	sends := append([]string(nil), sessions.sentCommands...)
	sessions.mu.Unlock()
	if len(sends) != 1 || sends[0] != "/goal resume" {
		t.Fatalf("second interruption recovery sends = %q, want one fresh resume command", sends)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	completion := strings.Join(lines[2:], "\n") + "\n"
	if _, err := f.WriteString(completion); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case result := <-done:
		if result.err != nil || result.terminal == nil || result.terminal.TerminalReason != "goal_complete" {
			t.Fatalf("reattached tail result = (%+v, %v), want goal_complete", result.terminal, result.err)
		}
	case <-ctx.Done():
		t.Fatal("reattached tail did not consume the later goal-complete event")
	}
}

func TestDelayedInterruptionRecoveryKeepsCapacityPromptMode(t *testing.T) {
	capacityScreen := "Selected model is at capacity. Please try again."
	coord, run, store, sessions := newDelayedRecoveryFixture(capacityScreen, time.Millisecond)
	store.mu.Lock()
	store.run.ResolvedConfig.Until = ""
	store.mu.Unlock()
	interruption, ok := ClassifyScreenInterruption(capacityScreen)
	if !ok {
		t.Fatal("capacity fixture did not classify")
	}
	coord.scheduleInterruptionRecovery(context.Background(), run, interruption)
	awaitRecoveryFinished(t, coord)

	sessions.mu.Lock()
	defer sessions.mu.Unlock()
	if len(sessions.sentPrompts) != 1 || sessions.sentPrompts[0] != "continue" || len(sessions.sentCommands) != 0 {
		t.Fatalf("capacity recovery sends commands=%q prompts=%q, want existing continue prompt", sessions.sentCommands, sessions.sentPrompts)
	}
	if !coord.recoveryCanReattach.Load() {
		t.Fatal("verified capacity prompt was not reported successful")
	}
}

func TestDelayedInterruptionRecoveryDoesNotClaimUnverifiedCommandSuccess(t *testing.T) {
	coord, run, _, sessions := newDelayedRecoveryFixture(stalledGoalScreen, time.Millisecond)
	sessions.mu.Lock()
	sessions.commandUnverified = true
	sessions.mu.Unlock()
	interruption, _ := ClassifyScreenInterruption(stalledGoalScreen)
	coord.scheduleInterruptionRecovery(context.Background(), run, interruption)
	awaitRecoveryFinished(t, coord)
	if coord.recoveryCanReattach.Load() {
		t.Fatal("unverified command submission was reported as successful recovery")
	}

	sessions.mu.Lock()
	defer sessions.mu.Unlock()
	if len(sessions.sentCommands) != 1 || sessions.sentCommands[0] != "/goal resume" {
		t.Fatalf("unverified command attempts = %q, want one non-replayed native command attempt", sessions.sentCommands)
	}
}

func TestDelayedInterruptionRecoveryUsesParentContextNotTailPollContext(t *testing.T) {
	coord, run, _, sessions := newDelayedRecoveryFixture(stalledGoalScreen, 10*time.Millisecond)
	parentCtx, cancelParent := context.WithCancel(context.Background())
	defer cancelParent()
	tailCtx, cancelTail := context.WithCancel(parentCtx)
	defer cancelTail()

	// The session watcher cancels this tail attempt after detecting an
	// interruption. The delayed recovery belongs to the owning run context.
	coord.watchSession(tailCtx, parentCtx, cancelTail, run)
	awaitRecoveryFinished(t, coord)
	if tailCtx.Err() == nil {
		t.Fatal("expected interruption detection to cancel the current tail poll")
	}
	if parentCtx.Err() != nil {
		t.Fatalf("watcher cancelled the parent run context: %v", parentCtx.Err())
	}

	sessions.mu.Lock()
	defer sessions.mu.Unlock()
	if len(sessions.sentCommands) != 1 || sessions.sentCommands[0] != "/goal resume" || len(sessions.sentPrompts) != 0 {
		t.Fatalf("delayed recovery sends commands=%q prompts=%q, want one verified native resume on the original session", sessions.sentCommands, sessions.sentPrompts)
	}
}

func TestDelayedInterruptionRecoveryStopsWhenParentRunIsCancelled(t *testing.T) {
	coord, run, _, sessions := newDelayedRecoveryFixture(stalledGoalScreen, time.Hour)
	parentCtx, cancelParent := context.WithCancel(context.Background())
	defer cancelParent()
	sessions.mu.Lock()
	sessions.screenEntered = make(chan struct{}, 1)
	sessions.screenRelease = make(chan struct{})
	entered, release := sessions.screenEntered, sessions.screenRelease
	sessions.mu.Unlock()
	tailCtx, cancelTail := context.WithCancel(parentCtx)
	defer cancelTail()
	coord.watchSession(tailCtx, parentCtx, cancelTail, run)

	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("watcher did not inspect the session screen")
	}
	close(release)
	awaitRecoveryScheduled(t, coord)
	cancelParent()
	awaitRecoveryFinished(t, coord)

	sessions.mu.Lock()
	defer sessions.mu.Unlock()
	if len(sessions.sentCommands) != 0 || len(sessions.sentPrompts) != 0 {
		t.Fatalf("cancelled parent run still received recovery input: commands %q prompts %q", sessions.sentCommands, sessions.sentPrompts)
	}
}
