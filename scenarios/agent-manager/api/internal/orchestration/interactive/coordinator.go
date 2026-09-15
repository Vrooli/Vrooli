// This file coordinates interactive sessions and their run-control callbacks.
package interactive

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

const (
	// defaultDebounceWindow is the idle window after a success (turn-boundary)
	// terminal marker before the run is judged complete (design R2). Interactive
	// CLIs stay alive between turns, so a turn boundary is only a run boundary
	// once the transcript stops growing for this long.
	defaultDebounceWindow = 5 * time.Second
	// defaultActivityPoll is how often the debounce loop checks for transcript
	// growth (a new turn beginning).
	defaultActivityPoll = 250 * time.Millisecond
	// defaultSessionPoll is how often the mid-tail watcher confirms the
	// web-console session still exists.
	defaultSessionPoll = 5 * time.Second
	// defaultInterruptionRecoveryDelay gives provider capacity and transient
	// availability failures time to clear before a single automatic retry.
	defaultInterruptionRecoveryDelay = 30 * time.Minute
	// defaultSessionReattachWindow is how long a missing session is tolerated
	// (with re-resolution) before the run is declared session_lost. A
	// web-console restart can briefly hide a session that the persistent backend
	// then recovers.
	defaultSessionReattachWindow = 3 * time.Minute
	// defaultCoordinatorHeartbeat is how often the live path refreshes
	// Run.LastHeartbeat so the reconciler does not treat a live interactive run
	// as stale.
	defaultCoordinatorHeartbeat = 30 * time.Second
	// terminalReasonStructuredResult marks a goal-mode run that ended through its
	// validated structured result rather than a runner-native goal marker.
	terminalReasonStructuredResult = "structured_result"
	// terminalReasonMaxTurns marks a run that reached its configured assistant
	// turn ceiling. Phase 3 types this as an interruption, not a failure.
	terminalReasonMaxTurns = "max_turns"
	// defaultTimeoutDrainGrace is how long the tail keeps reading after the run's
	// timeout, once the harness has been asked to stop its turn, so the
	// harness's own closing record reaches the event store.
	defaultTimeoutDrainGrace = 30 * time.Second
	// defaultNativeGoalFallbackTurns is how many success turn boundaries a run
	// may produce with no goal marker ever observed before the coordinator stops
	// waiting for runner-native goal support and accepts a validated structured
	// terminal. It only applies when the harness was not verified native.
	defaultNativeGoalFallbackTurns = 3
)

func transcriptModel(run *domain.Run) string {
	if run == nil {
		return ""
	}
	if run.ResolvedConfig != nil && run.ResolvedConfig.Model != "" {
		return run.ResolvedConfig.Model
	}
	if run.ActualModel != "" {
		return run.ActualModel
	}
	return run.RequestedModel
}

// ErrSessionGone is returned by [Coordinator.TailToCompletion] when the
// web-console session hosting the interactive CLI disappeared mid-tail. It is a
// distinct sentinel (not context.Canceled) so [Coordinator.Finalize] can
// distinguish a vanished session (finalize the run) from a graceful shutdown
// (leave the run for restart recovery).
var ErrSessionGone = errors.New("web-console session no longer exists")

// ErrResumableInterruption keeps a run active while its server-owned recovery
// timer waits. It is intentionally not a terminal failure.
var ErrResumableInterruption = errors.New("interactive provider interruption is resumable")

// RunStore is the minimal persistence seam the coordinator needs.
// repository.RunRepository satisfies it.
type RunStore interface {
	Update(ctx context.Context, run *domain.Run) error
}

// StatusBroadcaster broadcasts a run status change to any live subscribers.
// The orchestration EventBroadcaster satisfies it. Optional (nil is safe).
type StatusBroadcaster interface {
	BroadcastRunStatus(run *domain.Run)
}

// SummaryBuilder optionally reconstructs a run summary from persisted events at
// finalize time. The recovery path supplies the event-store-backed builder; the
// live path relies on the terminal marker's own summary. Nil is safe.
type SummaryBuilder func(ctx context.Context, runID uuid.UUID) *domain.RunSummary

// ResultBuilder reconstructs the canonical terminal result and its legacy
// summary projection from persisted events.
type ResultBuilder func(ctx context.Context, runID uuid.UUID, success bool, exitCode int, terminalReason string) (*domain.RunResult, *domain.RunSummary)

// CoordinatorDeps bundles the collaborators an interactive Coordinator drives.
type CoordinatorDeps struct {
	// Substrate creates/tears down the web-console session and resolves the
	// agent-owned transcript. Required for the live Execute path; may be nil for
	// a recovery-only coordinator (reattach never launches).
	Substrate *Substrate
	// Tailer follows the agent-owned transcript and surfaces terminal markers.
	Tailer *Tailer
	// Sessions verifies session liveness (GetSession) for mid-tail and recovery
	// session-gone detection. When nil, session-gone is never detected.
	Sessions webconsole.SessionController
	// Runs persists run mutations (status, cursor, session id).
	Runs RunStore
	// Broadcaster fans out status changes (optional).
	Broadcaster StatusBroadcaster
	// NewSink builds the per-run event sink the tailer emits into. When nil, tail
	// events are parsed (advancing the cursor + terminal detection) but not
	// stored.
	NewSink func(runID uuid.UUID) runner.EventSink
	// Summary optionally builds the completion summary from stored events.
	Summary SummaryBuilder
	Result  ResultBuilder
	// Clock overrides time.Now for run timestamps (tests). Nil uses time.Now.
	Clock func() time.Time
	// Debounce overrides the turn-boundary idle window (0 uses the default).
	Debounce time.Duration
	// ActivityPoll overrides the debounce growth-check cadence (0 uses default).
	ActivityPoll time.Duration
	// SessionPoll overrides the mid-tail session-liveness cadence (0 default).
	SessionPoll time.Duration
	// SessionReattachWindow overrides how long a missing session is tolerated
	// before failing the run (0 uses the default 3 minutes).
	SessionReattachWindow time.Duration
	// InterruptionRecoveryDelay controls the delayed retry after a resumable
	// provider interruption. Zero uses the 30-minute production default.
	InterruptionRecoveryDelay time.Duration
	// Heartbeat overrides the live-path heartbeat cadence (0 uses the default;
	// negative disables the heartbeat goroutine — used by the recovery path,
	// which does not own the live run).
	Heartbeat time.Duration
	// TimeoutDrainGrace overrides how long the tail keeps reading after the
	// run's timeout interrupt (0 uses the default; negative disables the
	// interrupt and ends the tail at the deadline).
	TimeoutDrainGrace time.Duration
}

// Coordinator runs the interactive execution strategy: it launches the real
// interactive agent CLI in a web-console session, tails the agent-owned
// transcript to completion (terminal marker + turn-boundary idle-debounce,
// design R2), and finalizes the run — the parallel execution path decided in the
// design doc §1. The same TailToCompletion + Finalize seams drive restart
// recovery (the reconciler reattaches a Coordinator to an already-launched run).
type Coordinator struct {
	deps CoordinatorDeps

	clock        func() time.Time
	debounce     time.Duration
	activityPoll time.Duration
	sessionPoll  time.Duration
	heartbeat    time.Duration
	timeoutDrain time.Duration

	sessionReattachWindow   time.Duration
	recoveryDelay           time.Duration
	recoverableInterruption atomic.Bool
	recoveryScheduled       atomic.Bool
	recoveryReady           chan struct{}
	recoverySucceeded       atomic.Bool
}

// NewCoordinator builds a Coordinator, applying defaults for any unset cadence.
func NewCoordinator(deps CoordinatorDeps) *Coordinator {
	c := &Coordinator{
		deps:                  deps,
		clock:                 deps.Clock,
		debounce:              deps.Debounce,
		activityPoll:          deps.ActivityPoll,
		sessionPoll:           deps.SessionPoll,
		heartbeat:             deps.Heartbeat,
		timeoutDrain:          deps.TimeoutDrainGrace,
		sessionReattachWindow: deps.SessionReattachWindow,
		recoveryDelay:         deps.InterruptionRecoveryDelay,
		recoveryReady:         make(chan struct{}),
	}
	if c.timeoutDrain == 0 {
		c.timeoutDrain = defaultTimeoutDrainGrace
	}
	if c.clock == nil {
		c.clock = time.Now
	}
	if c.recoveryDelay == 0 {
		c.recoveryDelay = defaultInterruptionRecoveryDelay
	}
	if c.debounce <= 0 {
		c.debounce = defaultDebounceWindow
	}
	if c.activityPoll <= 0 {
		c.activityPoll = defaultActivityPoll
	}
	if c.sessionPoll <= 0 {
		c.sessionPoll = defaultSessionPoll
	}
	if c.heartbeat == 0 {
		c.heartbeat = defaultCoordinatorHeartbeat
	}
	if c.sessionReattachWindow <= 0 {
		c.sessionReattachWindow = defaultSessionReattachWindow
	}
	return c
}

// tailContext carries the per-run facts the tailer needs to re-discover a
// rotated codex rollout. The live path fills it from LaunchParams; the recovery
// path leaves it zero (run.TranscriptPath is the pinned seed, which still tails
// correctly — codex rotation across a restart is not followed, matching the
// design's decision not to persist the run dir).
type tailContext struct {
	RunDir     string
	WorkingDir string
	LaunchedAt time.Time
}

// Execute runs the full live interactive lifecycle for a run: launch the session
// + discover the transcript, persist the durable facts, flip the run to Running,
// then tail to completion and finalize. onRunning (nil-safe) fires once the run
// reaches Running so the spawn dispatcher can release the startup slot.
//
// Continue/Stop (Phase 5) plug in on top of this: a Continue types a follow-up
// into the session (producing transcript growth the debounce treats as a new
// turn), and a Stop calls Substrate.Stop. The tail loop already treats every
// success terminal as a turn boundary, so multi-turn reuse is a no-op here.
func (c *Coordinator) Execute(ctx context.Context, run *domain.Run, p LaunchParams, onRunning func()) error {
	if c.deps.Substrate == nil {
		return fmt.Errorf("interactive coordinator: substrate is not configured")
	}

	res, err := c.deps.Substrate.Launch(ctx, p)
	// Persist whatever the launch produced (session id at minimum) before acting
	// on an error, so a discovery failure still leaves the session id durable.
	ApplyToRun(run, res)
	if err != nil {
		if res.SessionID != "" {
			// The session exists but is unusable (no transcript) — tear it down.
			_ = c.deps.Substrate.Stop(context.Background(), res.SessionID, interactiveSource(run))
		}
		// Seeding may have already copied credentials into the run's relocated
		// home before the launch failed; the CLI is dead, so remove them.
		_ = c.deps.Substrate.CleanupCredentials(p.RunnerType, p.RunDir)
		return c.finalizeFailed(ctx, run, fmt.Sprintf("interactive launch failed: %v", err))
	}

	var mu sync.Mutex
	now := c.clock()
	mu.Lock()
	run.Status = domain.RunStatusRunning
	run.Phase = domain.RunPhaseExecuting
	if run.StartedAt == nil {
		run.StartedAt = &now
	}
	run.InteractiveInvocationStartedAt = run.StartedAt
	run.LastHeartbeat = &now
	run.UpdatedAt = now
	mu.Unlock()
	if uerr := c.update(ctx, &mu, run); uerr != nil {
		return uerr
	}
	c.broadcast(run)
	if onRunning != nil {
		onRunning()
	}

	stopHeartbeat := c.startHeartbeat(ctx, &mu, run)
	defer stopHeartbeat()

	tc := tailContext{RunDir: p.RunDir, WorkingDir: p.WorkingDir}
	terminal, tailErr := c.tailToCompletion(ctx, run, tc, &mu)
	return c.Finalize(ctx, run, terminal, tailErr)
}

// TailToCompletion drains the run's agent-owned transcript from its persisted
// cursor and follows it to a run boundary. It is the reusable core of both the
// live path and restart recovery: every success terminal is a turn boundary,
// and the run is complete only once the transcript stops growing for the
// debounce window (design R2). It returns the last terminal marker seen and a
// non-nil error only for a vanished session (ErrSessionGone) or a cancelled
// context.
func (c *Coordinator) TailToCompletion(ctx context.Context, run *domain.Run) (*runner.TranscriptTerminal, error) {
	var mu sync.Mutex
	return c.tailToCompletion(ctx, run, tailContext{}, &mu)
}

// tailToCompletion follows the transcript to a run boundary. When the run's
// timeout expires it asks the harness to stop its turn and keeps reading for a
// bounded grace, so the harness's closing record (codex writes turn_aborted,
// which its parser turns into the turn's usage receipt) is stored before the run
// is finalized. The result still reports the timeout.
func (c *Coordinator) tailToCompletion(ctx context.Context, run *domain.Run, tc tailContext, mu *sync.Mutex) (*runner.TranscriptTerminal, error) {
	loopCtx, timedOut, release := c.timeoutInterruptContext(ctx, run)
	defer release()
	terminal, err := c.tailLoop(loopCtx, run, tc, mu)
	if timedOut() && !errors.Is(err, ErrSessionGone) {
		if terminal != nil && !terminal.Success {
			// The harness's abort answers our own timeout interrupt; it is not
			// the run's outcome, which stays a timeout.
			terminal = nil
		}
		return terminal, context.DeadlineExceeded
	}
	return terminal, err
}

// timeoutInterruptContext returns the context the tail runs under. Without a
// deadline, a session, or a positive grace it is ctx itself. Otherwise it
// outlives ctx's deadline by the drain grace: at the deadline the harness is
// interrupted, and any other end of ctx (shutdown, stop) ends the tail at once.
func (c *Coordinator) timeoutInterruptContext(ctx context.Context, run *domain.Run) (context.Context, func() bool, func()) {
	deadline, ok := ctx.Deadline()
	if !ok || c.timeoutDrain <= 0 || c.deps.Sessions == nil || run.WebConsoleSessionID == "" {
		return ctx, func() bool { return false }, func() {}
	}
	extended, cancel := context.WithDeadline(context.WithoutCancel(ctx), deadline.Add(c.timeoutDrain))
	var timedOut atomic.Bool
	stop := context.AfterFunc(ctx, func() {
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			cancel()
			return
		}
		timedOut.Store(true)
		interruptCtx, cancelInterrupt := context.WithTimeout(context.Background(), c.timeoutDrain)
		defer cancelInterrupt()
		if err := c.deps.Sessions.Interrupt(interruptCtx, run.WebConsoleSessionID, interactiveSource(run)); err != nil {
			// Nothing will close the turn; stop waiting for it.
			cancel()
		}
	})
	return extended, timedOut.Load, func() {
		stop()
		cancel()
	}
}

func (c *Coordinator) tailLoop(ctx context.Context, run *domain.Run, tc tailContext, mu *sync.Mutex) (*runner.TranscriptTerminal, error) {
	var sink runner.EventSink
	if c.deps.NewSink != nil {
		sink = c.deps.NewSink(run.ID)
		if sink != nil {
			defer sink.Close()
		}
	}

	cursor := run.TranscriptCursor
	onAdvance := func(cur, lastSeq int64) error {
		mu.Lock()
		defer mu.Unlock()
		if cur > cursor {
			cursor = cur
		}
		if cur > run.TranscriptCursor {
			run.TranscriptCursor = cur
		}
		if lastSeq > run.TranscriptLastSeq {
			run.TranscriptLastSeq = lastSeq
		}
		return c.deps.Runs.Update(context.Background(), run)
	}
	onSessionID := func(sessionID string) error {
		mu.Lock()
		defer mu.Unlock()
		if sessionID == "" || run.SessionID == sessionID {
			return nil
		}
		run.SessionID = sessionID
		return c.deps.Runs.Update(context.Background(), run)
	}
	var lastGoal *runner.GoalMarker
	onGoalStatus := func(marker runner.GoalMarker) error {
		if lastGoal != nil && *lastGoal == marker {
			return nil
		}
		copy := marker
		lastGoal = &copy
		mu.Lock()
		if run.GoalDelivery != GoalDeliveryNativeVerified {
			run.GoalDelivery = GoalDeliveryNativeVerified
			if c.deps.Runs != nil {
				_ = c.deps.Runs.Update(context.Background(), run)
			}
		}
		mu.Unlock()
		if sink == nil {
			return nil
		}
		return sink.Emit(domain.NewGoalStatusChangedEvent(run.ID, marker.Objective, string(marker.Status), marker.Iteration, marker.LastReason))
	}

	var lastTerminal *runner.TranscriptTerminal
	for {
		if err := ctx.Err(); err != nil {
			return lastTerminal, err
		}

		tctx, cancel := context.WithCancel(ctx)
		gone := c.watchSession(tctx, cancel, run)

		mu.Lock()
		startCursor := cursor
		mu.Unlock()

		term, err := c.deps.Tailer.Tail(tctx, TailParams{
			RunID:          run.ID,
			RunnerType:     run.ResolvedConfig.RunnerType,
			Model:          transcriptModel(run),
			TranscriptPath: run.TranscriptPath,
			SessionID:      run.SessionID,
			Until:          run.ResolvedConfig.Until,
			MaxTurns:       run.ResolvedConfig.MaxTurns,
			Billing:        run.Billing,
			RunDir:         tc.RunDir,
			WorkingDir:     tc.WorkingDir,
			LaunchedAt:     tc.LaunchedAt,
			StartCursor:    startCursor,
			Sink:           sink,
			OnAdvance:      onAdvance,
			OnSessionID:    onSessionID,
			OnGoalStatus:   onGoalStatus,
			GoalMarkerObserved: func() bool {
				return lastGoal != nil
			},
			StructuredResultSatisfied: func() bool {
				return c.structuredResultSatisfied(ctx, run)
			},
		})
		cancel()

		if gone() {
			// The session vanished mid-tail: prefer the freshest terminal seen.
			return firstTerminal(term, lastTerminal), ErrSessionGone
		}
		if err != nil {
			if c.recoverableInterruption.Load() {
				// The watcher cancelled the tail so the coordinator can wait for
				// the delayed retry. Keep this coordinator attached; otherwise a
				// successful retry would have nobody reading the next turn.
				select {
				case <-c.recoveryReady:
					if c.recoverySucceeded.Load() {
						c.recoverableInterruption.Store(false)
						continue
					}
					return nil, ErrResumableInterruption
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			// The caller cancelled (shutdown); leave completion to recovery.
			return firstTerminal(term, lastTerminal), err
		}
		if term == nil {
			return lastTerminal, nil
		}
		if !term.Success {
			return term, nil
		}
		if term.TerminalReason != "" && strings.HasPrefix(term.TerminalReason, "goal_") {
			return term, nil
		}
		// A validated structured terminal is a real terminal in goal mode even
		// when the harness never emitted a runner-native goal marker. This is
		// the completion rule for a non-native harness and the fallback for a
		// native one that swallowed /goal.
		if term.TerminalReason == terminalReasonStructuredResult {
			return term, nil
		}

		// A success terminal is a turn boundary. Wait out the idle-debounce: if
		// the transcript grows, a new turn began and we keep tailing; otherwise
		// the run is complete.
		lastTerminal = term
		if lastTerminal.TerminalReason == "" {
			lastTerminal.TerminalReason = "turn_boundary_idle"
		}
		mu.Lock()
		boundary := cursor
		mu.Unlock()
		if !c.awaitNextTurn(ctx, run, tc, boundary) {
			return term, nil
		}
	}
}

// structuredResultSatisfied checks the canonical result projection after a
// provider success marker has been persisted. It is the deterministic
// structured-output completion rule for goal-driven interactive runs when the
// provider does not emit a native goal marker.
func (c *Coordinator) structuredResultSatisfied(ctx context.Context, run *domain.Run) bool {
	if c.deps.Result == nil || run == nil || run.ResolvedConfig == nil || run.ResolvedConfig.ResultSpec == nil {
		return false
	}
	// Only deterministic extraction may end the tail. A constrained extractor
	// can be unavailable or asynchronous and is not a transcript terminal rule.
	if run.ResolvedConfig.ResultSpec.ExtractionMode != "" && run.ResolvedConfig.ResultSpec.ExtractionMode != domain.StructuredExtractionDeterministic {
		return false
	}
	result, _ := c.deps.Result(ctx, run.ID, true, 0, "structured_result")
	return result != nil && result.Structured != nil && result.Structured.Status == domain.StructuredResultSuccess
}

// awaitNextTurn blocks until the transcript grows beyond the turn-boundary
// cursor (a new turn began → true) or the debounce window elapses with no growth
// (the run is idle → false). Growth is measured on the file, not the consumed
// cursor, because nothing is reading the file between turns. claude writes its
// interactive-only records (file-history-snapshot/ai-title/last-prompt) only at
// the start of a NEW turn, so file growth is a faithful new-turn signal.
func (c *Coordinator) awaitNextTurn(ctx context.Context, run *domain.Run, tc tailContext, boundary int64) bool {
	deadline := c.clock().Add(c.debounce)
	for {
		if c.transcriptGrew(run, tc, boundary) {
			return true
		}
		if !c.clock().Before(deadline) {
			return false
		}
		if !sleepCtx(ctx, c.activityPoll) {
			return false
		}
	}
}

// transcriptGrew reports whether new transcript bytes exist beyond boundary, or
// (for codex) a strictly newer rollout has rotated in.
func (c *Coordinator) transcriptGrew(run *domain.Run, tc tailContext, boundary int64) bool {
	path := run.TranscriptPath
	if run.ResolvedConfig != nil && run.ResolvedConfig.RunnerType == domain.RunnerTypeCodex {
		if newest, err := findTranscript(DiscoverParams{
			RunnerType: domain.RunnerTypeCodex,
			WorkingDir: tc.WorkingDir,
			RunDir:     tc.RunDir,
			LaunchedAt: tc.LaunchedAt,
		}); err == nil && newest != "" {
			if newest != path {
				return true
			}
			path = newest
		}
	}
	if info, err := os.Stat(path); err == nil {
		return info.Size() > boundary
	}
	return false
}

// watchSession polls GetSession while a tail is in flight; on NotFound it cancels
// the tail and latches "gone". Returns a reader for the latched flag. When the
// session controller is not configured (or the run has no session id yet), the
// watcher is inert and gone() is always false.
func (c *Coordinator) watchSession(ctx context.Context, cancel context.CancelFunc, run *domain.Run) func() bool {
	if c.deps.Sessions == nil || run.WebConsoleSessionID == "" {
		return func() bool { return false }
	}
	var mu sync.Mutex
	gone := false
	// A missing session is tolerated for sessionReattachWindow while the
	// persistent web-console backend re-resolves it. Only after the window
	// elapses is the run declared session_lost.
	var notFoundSince time.Time
	go func() {
		ticker := time.NewTicker(c.sessionPoll)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, err := c.deps.Sessions.GetSession(ctx, run.WebConsoleSessionID)
				if !errors.Is(err, webconsole.ErrSessionNotFound) {
					notFoundSince = time.Time{}
					if err == nil {
						if screen, screenErr := c.deps.Sessions.Screen(ctx, run.WebConsoleSessionID, true); screenErr == nil {
							if interruption, detected := ClassifyScreenInterruption(screen); detected {
								c.recoverableInterruption.Store(true)
								c.scheduleInterruptionRecovery(run, interruption)
								cancel()
							}
						}
					}
					continue
				}
				if notFoundSince.IsZero() {
					notFoundSince = c.clock()
				}
				if c.clock().Sub(notFoundSince) < c.sessionReattachWindow {
					continue
				}
				mu.Lock()
				gone = true
				mu.Unlock()
				cancel()
				return
			}
		}
	}()
	return func() bool {
		mu.Lock()
		defer mu.Unlock()
		return gone
	}
}

// scheduleInterruptionRecovery keeps a capacity-affected run alive while a
// server-owned timer waits, then sends exactly one mode-aware continuation into
// the existing Web Console session. Cancellation of the parent run cancels the
// timer; repeated screen polls cannot duplicate the prompt.
func (c *Coordinator) scheduleInterruptionRecovery(run *domain.Run, interruption ResumableInterruption) {
	if run == nil || c.deps.Sessions == nil || !c.recoveryScheduled.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer close(c.recoveryReady)
		timer := time.NewTimer(c.recoveryDelay)
		defer timer.Stop()
		<-timer.C
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := c.deps.Sessions.GetSession(ctx, run.WebConsoleSessionID); err != nil {
			return
		}
		if c.deps.NewSink != nil {
			if events := c.deps.NewSink(run.ID); events != nil {
				_ = events.Emit(domain.NewLogEvent(run.ID, "warn", fmt.Sprintf(
					"resumable interactive interruption detected (%s: %s); attempting delayed recovery",
					interruption.Kind, interruption.Message)))
				_ = events.Close()
			}
		}
		if run.ResolvedConfig != nil && strings.TrimSpace(run.ResolvedConfig.Until) != "" {
			if err := c.deps.Sessions.SendText(ctx, run.WebConsoleSessionID, "/goal resume\n", interactiveSource(run)); err != nil {
				return
			}
		} else if err := c.deps.Sessions.SendPrompt(ctx, run.WebConsoleSessionID, "continue", interactiveSource(run)); err != nil {
			return
		}
		c.recoverySucceeded.Store(true)
	}()
}

// VerifySession reports whether the run's web-console session still exists. A
// transient RPC error is surfaced (recovery leaves the run for the next cycle);
// ErrSessionNotFound is reported as alive=false, err=nil.
func (c *Coordinator) VerifySession(ctx context.Context, run *domain.Run) (bool, error) {
	if c.deps.Sessions == nil || run.WebConsoleSessionID == "" {
		return false, fmt.Errorf("interactive coordinator: session controller not configured")
	}
	_, err := c.deps.Sessions.GetSession(ctx, run.WebConsoleSessionID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, webconsole.ErrSessionNotFound) {
		return false, nil
	}
	return false, err
}

// Finalize writes the terminal outcome onto the run and broadcasts it. It is the
// single completion seam shared by the live and recovery paths.
//
//   - A graceful shutdown (ctx cancelled, session intact) leaves the run Running
//     so restart recovery re-adopts it — Finalize is a no-op.
//   - A success terminal completes the run (even if the session vanished
//     afterward: the last turn finished).
//   - A failure terminal fails the run with its reason.
//   - A vanished session with no success terminal fails the run with an explicit
//     reason so it is never left orphaned.
func (c *Coordinator) Finalize(ctx context.Context, run *domain.Run, terminal *runner.TranscriptTerminal, tailErr error) error {
	if errors.Is(tailErr, ErrResumableInterruption) {
		return nil
	}
	if tailErr != nil && errors.Is(tailErr, context.Canceled) && !errors.Is(tailErr, ErrSessionGone) {
		return nil
	}

	switch {
	case terminal != nil && terminal.Success:
		return c.finalizeComplete(ctx, run, terminal)
	case terminal != nil && !terminal.Success:
		msg := terminal.ErrorMessage
		if msg == "" {
			msg = "interactive run reported a failure terminal"
		}
		// Carry the codec's typed class/reason (for example a usage-limit
		// interruption) onto the run before finalizing.
		if terminal.TerminalClass != "" {
			run.TerminalClass = terminal.TerminalClass
		}
		if terminal.StopReason != "" {
			run.StopReason = terminal.StopReason
		}
		return c.finalizeFailed(ctx, run, msg)
	case errors.Is(tailErr, ErrSessionGone):
		run.TerminalClass = domain.RunTerminalClassInterruption
		run.StopReason = domain.RunStopReasonSessionLost
		return c.finalizeFailed(ctx, run, fmt.Sprintf(
			"web-console session %s no longer exists; interactive run cannot be recovered", run.WebConsoleSessionID))
	case errors.Is(tailErr, context.DeadlineExceeded):
		// The run hit its configured timeout ceiling. Phase 3 types this as the
		// timeout interruption (terminal_class=interruption, stop_reason=timeout)
		// with the last handoff attached; the terminal reason is recorded now.
		return c.finalizeTimeout(ctx, run)
	default:
		return c.finalizeFailed(ctx, run, "interactive run ended without a terminal marker")
	}
}

func (c *Coordinator) finalizeComplete(ctx context.Context, run *domain.Run, terminal *runner.TranscriptTerminal) error {
	now := c.clock()
	var mu sync.Mutex
	mu.Lock()
	run.Status = domain.RunStatusComplete
	run.Phase = domain.RunPhaseCompleted
	run.ErrorMsg = ""
	run.EndedAt = &now
	run.UpdatedAt = now
	run.TerminalClass = domain.RunTerminalClassVerdict
	run.StopReason = domain.RunStopReasonComplete
	if terminal.TerminalClass != "" {
		run.TerminalClass = terminal.TerminalClass
	}
	if terminal.StopReason != "" {
		run.StopReason = terminal.StopReason
	}
	exit := terminal.ExitCode
	run.ExitCode = &exit
	reason := terminal.TerminalReason
	if reason == "" {
		reason = "completed"
	}
	if c.deps.Result != nil {
		run.Result, run.Summary = c.deps.Result(ctx, run.ID, true, exit, reason)
	} else if summary := c.buildSummary(ctx, run, terminal); summary != nil {
		run.Summary = summary
	}
	run.LastHandoff = handoffFromTerminal(terminal, run.Result, run.Summary)
	mu.Unlock()
	if err := c.update(ctx, &mu, run); err != nil {
		return err
	}
	c.broadcast(run)
	return nil
}

func (c *Coordinator) finalizeFailed(ctx context.Context, run *domain.Run, reason string) error {
	now := c.clock()
	var mu sync.Mutex
	mu.Lock()
	run.Status = domain.RunStatusFailed
	run.Phase = domain.RunPhaseCompleted
	run.ErrorMsg = reason
	if run.TerminalClass == "" {
		run.TerminalClass = domain.RunTerminalClassInterruption
	}
	if run.StopReason == "" {
		run.StopReason = domain.RunStopReasonCrash
	}
	if c.deps.Result != nil {
		run.Result, run.Summary = c.deps.Result(ctx, run.ID, false, 1, "failed")
	}
	run.LastHandoff = handoffFromTerminal(nil, run.Result, run.Summary)
	run.EndedAt = &now
	run.UpdatedAt = now
	mu.Unlock()
	if err := c.update(ctx, &mu, run); err != nil {
		return err
	}
	c.broadcast(run)
	return nil
}

// handoffFromTerminal prefers the terminal's own summary, then the persisted
// result's final output, then the run summary. It is the text a resume prompt
// carries forward.
func handoffFromTerminal(terminal *runner.TranscriptTerminal, result *domain.RunResult, summary *domain.RunSummary) string {
	if terminal != nil && terminal.Summary != nil && strings.TrimSpace(terminal.Summary.Description) != "" {
		return terminal.Summary.Description
	}
	if result != nil && strings.TrimSpace(result.FinalOutput) != "" {
		return result.FinalOutput
	}
	if summary != nil {
		return summary.Description
	}
	return ""
}

// finalizeTimeout ends a run that reached its configured timeout ceiling. The
// terminal reason is "timeout" so Swarm can classify the interruption; Phase 3
// adds the typed terminal_class/stop_reason pair and the retained handoff.
//
// The tail context is already expired when this is called, so the terminal
// state is persisted on a fresh bounded context — using the expired one would
// fail the write and leave the run stuck in RUNNING (observed live).
func (c *Coordinator) finalizeTimeout(ctx context.Context, run *domain.Run) error {
	persistCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	now := c.clock()
	var mu sync.Mutex
	mu.Lock()
	run.Status = domain.RunStatusFailed
	run.Phase = domain.RunPhaseCompleted
	run.ErrorMsg = "interactive run exceeded its configured timeout"
	run.TerminalClass = domain.RunTerminalClassInterruption
	run.StopReason = domain.RunStopReasonTimeout
	if c.deps.Result != nil {
		run.Result, run.Summary = c.deps.Result(persistCtx, run.ID, false, 1, "timeout")
	}
	run.LastHandoff = handoffFromTerminal(nil, run.Result, run.Summary)
	run.EndedAt = &now
	run.UpdatedAt = now
	mu.Unlock()
	if err := c.update(persistCtx, &mu, run); err != nil {
		return err
	}
	c.broadcast(run)
	return nil
}

// buildSummary prefers the event-store-backed summary (recovery path) and falls
// back to the terminal marker's own summary (live path).
func (c *Coordinator) buildSummary(ctx context.Context, run *domain.Run, terminal *runner.TranscriptTerminal) *domain.RunSummary {
	if c.deps.Summary != nil {
		if s := c.deps.Summary(ctx, run.ID); s != nil {
			return s
		}
	}
	return terminal.Summary
}

// startHeartbeat refreshes Run.LastHeartbeat on the live path so the reconciler
// does not treat an in-flight interactive run as stale. Returns a stop function.
// A non-positive heartbeat interval disables it (recovery path).
func (c *Coordinator) startHeartbeat(ctx context.Context, mu *sync.Mutex, run *domain.Run) func() {
	if c.heartbeat <= 0 {
		return func() {}
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(c.heartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := c.clock()
				mu.Lock()
				run.LastHeartbeat = &now
				mu.Unlock()
				_ = c.update(ctx, mu, run)
			}
		}
	}()
	return func() {
		close(stop)
		<-done
	}
}

func (c *Coordinator) update(ctx context.Context, mu *sync.Mutex, run *domain.Run) error {
	mu.Lock()
	defer mu.Unlock()
	return c.deps.Runs.Update(ctx, run)
}

func (c *Coordinator) broadcast(run *domain.Run) {
	if c.deps.Broadcaster != nil {
		c.deps.Broadcaster.BroadcastRunStatus(run)
	}
}

// interactiveSource is the diagnostic SendInput attribution for a run's session
// traffic (locked decision 4 — attribution only, no lease).
func interactiveSource(run *domain.Run) string {
	return sessionSource(run.ID)
}

// sessionSource builds the diagnostic SendInput attribution for a run id.
func sessionSource(id uuid.UUID) string {
	return "agent-manager:run-" + id.String()
}

// firstTerminal returns a if non-nil, else b.
func firstTerminal(a, b *runner.TranscriptTerminal) *runner.TranscriptTerminal {
	if a != nil {
		return a
	}
	return b
}
