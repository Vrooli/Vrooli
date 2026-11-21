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
	"agent-manager/internal/repository"

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
	sentGoals            []string
	nativeOwner          string
	nativeThread         string
	nativeState          webconsole.NativeAgentState
	goalStatus           string
	goalUnknown          bool
	promptErr            error
	legacyPromptCalls    int
	nativePromptCalls    int
	beforeNativeRead     func()
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
	info, err := s.fakeSessions.GetSession(ctx, id)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.nativeOwner != "" {
		info.LaunchMode, info.ControlMode = "codex_app_server", "native_capable"
		info.NativeOwner, info.NativeThreadID = s.nativeOwner, s.nativeThread
	}
	return info, err
}

func (s *delayedRecoverySessions) ObserveNativeAgent(ctx context.Context, sessionID, threadID string) (webconsole.NativeAgentObservation, error) {
	if s.beforeNativeRead != nil {
		s.beforeNativeRead()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctx.Err() != nil {
		return webconsole.NativeAgentObservation{}, ctx.Err()
	}
	if sessionID != s.activeSessionID || threadID != s.nativeThread {
		return webconsole.NativeAgentObservation{}, errors.New("wrong native binding")
	}
	state := s.nativeState
	if state == "" {
		state = webconsole.NativeAgentIdle
	}
	status := s.goalStatus
	if status == "" {
		status = "paused"
	}
	return webconsole.NativeAgentObservation{State: state, Generation: s.nativeOwner, GoalAvailable: !s.goalUnknown, HasGoal: !s.goalUnknown, GoalStatus: status}, nil
}

func (s *delayedRecoverySessions) SetNativeGoalForSession(ctx context.Context, sessionID, threadID, owner, objective, status string, origin webconsole.NativeGoalOrigin, budget *int64) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if sessionID != s.activeSessionID || threadID != s.nativeThread || owner != s.nativeOwner || origin != webconsole.NativeGoalAutomatic || objective == "" || status != "active" {
		return "", errors.New("wrong native goal binding or provenance")
	}
	s.sentGoals = append(s.sentGoals, status)
	if s.commandUnverified {
		return "paused", nil
	}
	if s.clearScreenOnCommand {
		s.screen = "› Ask Codex to do anything"
		s.screens = nil
	}
	s.goalStatus = "active"
	return "active", nil
}

func (s *delayedRecoverySessions) SendPromptForNativeOwner(ctx context.Context, sessionID, threadID, owner, prompt, source string) (webconsole.PromptSubmission, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctx.Err() != nil {
		return webconsole.PromptSubmission{}, ctx.Err()
	}
	if sessionID != s.activeSessionID || threadID != s.nativeThread || owner != s.nativeOwner {
		return webconsole.PromptSubmission{}, errors.New("native generation changed")
	}
	s.sentPrompts = append(s.sentPrompts, prompt)
	s.nativePromptCalls++
	return webconsole.PromptSubmission{Verified: !s.commandUnverified}, s.promptErr
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
	s.legacyPromptCalls++
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
		SessionID:           "native-current",
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
		nativeOwner:     "native-owner",
		nativeThread:    run.SessionID,
	}
	coord := NewCoordinator(CoordinatorDeps{
		RecoveryReceipts:          &recoveryReceiptFake{},
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
				if len(sessions.sentGoals) != 1 || sessions.sentGoals[0] != "active" || len(sessions.sentPrompts) != 0 {
					t.Fatalf("recovery sends = commands %q, prompts %q; want one verified native resume command", sessions.sentGoals, sessions.sentPrompts)
				}
				if !coord.recoveryCanReattach.Load() {
					t.Fatal("verified recovery was not reported successful")
				}
				return
			}
			if len(sessions.sentGoals) != 0 || len(sessions.sentPrompts) != 0 {
				t.Fatalf("stale recovery sent commands %q or prompts %q", sessions.sentGoals, sessions.sentPrompts)
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
	if len(sessions.sentGoals) != 1 || sessions.sentGoals[0] != "active" {
		t.Fatalf("recovery sends = %q, want one command for captured run/session", sessions.sentGoals)
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
	if len(sessions.sentGoals) != 0 || len(sessions.sentPrompts) != 0 {
		t.Fatalf("cancelled run received input after screen read: commands %q prompts %q", sessions.sentGoals, sessions.sentPrompts)
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
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
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
		nativeOwner:          "native-owner",
		nativeThread:         sessionID,
		activeSessionID:      sessionID,
		screen:               "› Ask Codex to do anything",
		screens:              []string{stalledGoalScreen, "› Ask Codex to do anything", "› Ask Codex to do anything"},
		screenObserved:       make(chan struct{}, 12),
		clearScreenOnCommand: true,
	}
	coord := NewCoordinator(CoordinatorDeps{
		RecoveryReceipts:          &recoveryReceiptFake{},
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
		terminal, err := coord.TailToCompletionWithContext(ctx, run, TailContext{RunDir: filepath.Dir(path)}, TailCompletionOptions{})
		done <- outcome{terminal: terminal, err: err}
	}()

	// First detection cancels the initial tail. The delayed check sees the
	// cleared screen and sends no input, then the third read proves that tailing
	// resumed. A later interruption must get one fresh recovery cycle.
	awaitScreenReads(t, sessions, 3)
	sessions.mu.Lock()
	initialSends := append([]string(nil), sessions.sentGoals...)
	if len(initialSends) != 0 {
		sessions.mu.Unlock()
		t.Fatalf("cleared interruption received stale input: %q", initialSends)
	}
	sessions.screen = stalledGoalScreen
	sessions.mu.Unlock()
	awaitScreenReads(t, sessions, 5)
	awaitScreenReads(t, sessions, 6)
	sessions.mu.Lock()
	sends := append([]string(nil), sessions.sentGoals...)
	sessions.mu.Unlock()
	if len(sends) != 1 || sends[0] != "active" {
		t.Fatalf("second interruption recovery sends = %q, want one fresh resume command", sends)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
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
	if len(sessions.sentPrompts) != 1 || sessions.sentPrompts[0] != "continue" || len(sessions.sentGoals) != 0 {
		t.Fatalf("capacity recovery sends commands=%q prompts=%q, want existing continue prompt", sessions.sentGoals, sessions.sentPrompts)
	}
	if !coord.recoveryCanReattach.Load() {
		t.Fatal("verified capacity prompt was not reported successful")
	}
}

func TestDelayedInterruptionRecoveryDoesNotClaimUnacceptedNativeGoalSuccess(t *testing.T) {
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
	if len(sessions.sentGoals) != 1 || sessions.sentGoals[0] != "active" {
		t.Fatalf("unverified command attempts = %q, want one non-replayed native command attempt", sessions.sentGoals)
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
	if len(sessions.sentGoals) != 1 || sessions.sentGoals[0] != "active" || len(sessions.sentPrompts) != 0 {
		t.Fatalf("delayed recovery sends commands=%q prompts=%q, want one verified native resume on the original session", sessions.sentGoals, sessions.sentPrompts)
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
	if len(sessions.sentGoals) != 0 || len(sessions.sentPrompts) != 0 {
		t.Fatalf("cancelled parent run still received recovery input: commands %q prompts %q", sessions.sentGoals, sessions.sentPrompts)
	}
}

func TestDelayedNativeRecoveryRejectsChangedBindingAndPreservesBusyOwner(t *testing.T) {
	for _, tc := range []struct {
		name         string
		change       func(*delayedRecoveryRunStore, *delayedRecoverySessions)
		wantReattach bool
	}{
		{name: "new thread", change: func(store *delayedRecoveryRunStore, _ *delayedRecoverySessions) {
			store.mu.Lock()
			defer store.mu.Unlock()
			store.run.SessionID = "replacement-thread"
		}},
		{name: "new invocation", change: func(store *delayedRecoveryRunStore, _ *delayedRecoverySessions) {
			store.mu.Lock()
			defer store.mu.Unlock()
			store.run.LifecycleVersion++
		}},
		{name: "parked", change: func(store *delayedRecoveryRunStore, _ *delayedRecoverySessions) {
			store.mu.Lock()
			defer store.mu.Unlock()
			store.run.Status = domain.RunStatusParked
		}},
		{name: "replacement owner", change: func(_ *delayedRecoveryRunStore, sessions *delayedRecoverySessions) {
			sessions.mu.Lock()
			defer sessions.mu.Unlock()
			sessions.nativeOwner = "replacement-owner"
		}},
		{name: "unknown native", change: func(_ *delayedRecoveryRunStore, sessions *delayedRecoverySessions) {
			sessions.mu.Lock()
			defer sessions.mu.Unlock()
			sessions.nativeState = webconsole.NativeAgentUnknown
		}},
		{name: "cleared screen with unknown native", change: func(_ *delayedRecoveryRunStore, sessions *delayedRecoverySessions) {
			sessions.mu.Lock()
			defer sessions.mu.Unlock()
			sessions.nativeState = webconsole.NativeAgentUnknown
			sessions.screen = "shell ready"
		}},
		{name: "busy native keeps observing without duplicate input", wantReattach: true, change: func(_ *delayedRecoveryRunStore, sessions *delayedRecoverySessions) {
			sessions.mu.Lock()
			defer sessions.mu.Unlock()
			sessions.nativeState = webconsole.NativeAgentBusy
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			coord, run, store, sessions := newDelayedRecoveryFixture(stalledGoalScreen, 20*time.Millisecond)
			interruption, _ := ClassifyScreenInterruption(stalledGoalScreen)
			coord.scheduleInterruptionRecovery(context.Background(), run, interruption)
			tc.change(store, sessions)
			awaitRecoveryFinished(t, coord)
			if coord.recoveryCanReattach.Load() != tc.wantReattach {
				t.Fatalf("reattach=%t, want %t", coord.recoveryCanReattach.Load(), tc.wantReattach)
			}
			sessions.mu.Lock()
			defer sessions.mu.Unlock()
			if len(sessions.sentGoals)+len(sessions.sentPrompts)+len(sessions.sentCommands) != 0 {
				t.Fatalf("changed/unknown/busy owner received effects: goals=%v prompts=%v commands=%v", sessions.sentGoals, sessions.sentPrompts, sessions.sentCommands)
			}
		})
	}
}

func TestDelayedRecoveryPreservesLegacyProviderPromptWithoutNativeFallback(t *testing.T) {
	coord, run, store, sessions := newDelayedRecoveryFixture("Selected model is at capacity", time.Millisecond)
	run.ResolvedConfig.RunnerType = domain.RunnerTypeClaudeCode
	store.mu.Lock()
	store.run.ResolvedConfig.RunnerType = domain.RunnerTypeClaudeCode
	store.mu.Unlock()
	sessions.mu.Lock()
	sessions.nativeOwner = ""
	sessions.mu.Unlock()
	interruption, _ := ClassifyScreenInterruption("Selected model is at capacity")
	coord.scheduleInterruptionRecovery(context.Background(), run, interruption)
	awaitRecoveryFinished(t, coord)
	sessions.mu.Lock()
	defer sessions.mu.Unlock()
	if !coord.recoveryCanReattach.Load() || sessions.legacyPromptCalls != 1 || sessions.nativePromptCalls != 0 || len(sessions.sentCommands) != 0 {
		t.Fatalf("legacy recovery changed routes: reattach=%t legacy=%d native=%d", coord.recoveryCanReattach.Load(), sessions.legacyPromptCalls, sessions.nativePromptCalls)
	}
}

func TestDelayedNativeRecoveryRechecksCancellationAfterNativeRead(t *testing.T) {
	for _, cancelContext := range []bool{false, true} {
		name := "durable cancellation"
		if cancelContext {
			name = "parent cancellation"
		}
		t.Run(name, func(t *testing.T) {
			coord, run, store, sessions := newDelayedRecoveryFixture(stalledGoalScreen, time.Millisecond)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			sessions.beforeNativeRead = func() {
				if cancelContext {
					cancel()
					return
				}
				store.mu.Lock()
				store.run.Status = domain.RunStatusCancelled
				store.mu.Unlock()
			}
			interruption, _ := ClassifyScreenInterruption(stalledGoalScreen)
			coord.scheduleInterruptionRecovery(ctx, run, interruption)
			awaitRecoveryFinished(t, coord)
			sessions.mu.Lock()
			defer sessions.mu.Unlock()
			if coord.recoveryCanReattach.Load() || len(sessions.sentGoals)+len(sessions.sentPrompts)+len(sessions.sentCommands) != 0 {
				t.Fatal("cancellation during native read allowed recovery")
			}
		})
	}
}

// The receipt fake is deliberately persistent across Coordinator reconstruction.
// No database, provider, subprocess or host discovery is used by these cases.
type recoveryReceiptFake struct {
	repository.IdempotencyRepository
	mu     sync.Mutex
	record *domain.IdempotencyRecord
}

func (r *recoveryReceiptFake) CheckContinuation(ctx context.Context, key string) (*domain.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.record == nil {
		return nil, nil
	}
	copy := *r.record
	return &copy, nil
}
func (r *recoveryReceiptFake) ReserveContinuation(ctx context.Context, key string, id uuid.UUID, hash []byte, ttl time.Duration) (*domain.IdempotencyRecord, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.record != nil && r.record.Status != domain.IdempotencyStatusFailed {
		copy := *r.record
		return &copy, false, nil
	}
	r.record = &domain.IdempotencyRecord{Key: key, EntityID: &id, EntityType: "Continuation", Response: append([]byte(nil), hash...), Status: domain.IdempotencyStatusPending}
	copy := *r.record
	return &copy, true, nil
}
func (r *recoveryReceiptFake) CompleteContinuation(ctx context.Context, key string, id uuid.UUID, hash []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.record.Status = domain.IdempotencyStatusComplete
	return nil
}
func (r *recoveryReceiptFake) FailContinuation(ctx context.Context, key string, id uuid.UUID, hash []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.record.Status = domain.IdempotencyStatusFailed
	return nil
}

func TestDelayedNativeRecoveryUsesAuthoritativeGoalState(t *testing.T) {
	for _, status := range []string{"active", "complete", "blocked", "unknown"} {
		t.Run(status, func(t *testing.T) {
			c, run, _, sessions := newDelayedRecoveryFixture(stalledGoalScreen, time.Millisecond)
			sessions.goalStatus = status
			initial, _ := sessions.GetSession(context.Background(), run.WebConsoleSessionID)
			interruption, _ := ClassifyScreenInterruption(stalledGoalScreen)
			got := c.recoverInterruptedSession(context.Background(), run, initial, interruption)
			if got != (status != "unknown") || len(sessions.sentGoals) != 0 || sessions.nativePromptCalls != 0 {
				t.Fatalf("status=%s recovered=%v mutations=%v prompts=%d", status, got, sessions.sentGoals, sessions.nativePromptCalls)
			}
		})
	}
}
func TestDelayedNativePromptAmbiguitySurvivesCoordinatorReconstruction(t *testing.T) {
	for _, outcome := range []string{"lost_ack", "verified", "pre_effect_refused"} {
		t.Run(outcome, func(t *testing.T) {
			c, run, store, sessions := newDelayedRecoveryFixture("Selected model is at capacity", time.Millisecond)
			run.ResolvedConfig.Until = ""
			store.run.ResolvedConfig.Until = ""
			if outcome == "lost_ack" {
				sessions.promptErr = errors.New("ack lost")
			}
			if outcome == "pre_effect_refused" {
				sessions.promptErr = webconsole.ErrNativePromptRefused
			}
			initial, _ := sessions.GetSession(context.Background(), run.WebConsoleSessionID)
			interruption, _ := ClassifyScreenInterruption("Selected model is at capacity")
			_ = c.recoverInterruptedSession(context.Background(), run, initial, interruption)
			sessions.promptErr = nil
			rebuilt := NewCoordinator(c.deps)
			got := rebuilt.recoverInterruptedSession(context.Background(), run, initial, interruption)
			wantCalls := 1
			if outcome == "pre_effect_refused" {
				wantCalls = 2
			}
			if sessions.nativePromptCalls != wantCalls || got != (outcome != "lost_ack") {
				t.Fatalf("outcome=%s resumed=%v prompts=%d", outcome, got, sessions.nativePromptCalls)
			}
		})
	}
}
