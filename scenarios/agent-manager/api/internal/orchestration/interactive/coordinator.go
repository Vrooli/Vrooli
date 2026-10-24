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
	// then recovers. This must cover the full managed restart/recovery path, not
	// merely the normal API restart latency; declaring loss inside this window
	// can leave a live PTY orphaned from its Agent Manager run.
	defaultSessionReattachWindow = 15 * time.Minute
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
	Get(ctx context.Context, id uuid.UUID) (*domain.Run, error)
	Update(ctx context.Context, run *domain.Run) error
	BeginInteractiveInvocation(ctx context.Context, run *domain.Run, expectedStatus domain.RunStatus, expectedStartedAt *time.Time, at time.Time) (bool, error)
	UpdateRunnerStreamState(ctx context.Context, run *domain.Run) (bool, error)
}

var (
	ErrRunOwnerUnavailable = errors.New("interactive run owner could not be verified")
	ErrRunOwnerChanged     = errors.New("interactive run owner changed during observation")
)

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
	// CompletionOptions is owner-supplied for a particular recovered invocation.
	// Ordinary coordinator construction leaves it zero-valued.
	CompletionOptions TailCompletionOptions
	// MintIdentityToken creates the child bearer after a reserved session ID is
	// bound to the run. Nil disables child identity minting for isolated callers.
	MintIdentityToken func(ctx context.Context, run *domain.Run, sessionID string) (string, error)
	// Clock overrides time.Now for run timestamps (tests). Nil uses time.Now.
	Clock func() time.Time
	// Debounce overrides the turn-boundary idle window (0 uses the default).
	Debounce time.Duration
	// ActivityPoll overrides the debounce growth-check cadence (0 uses default).
	ActivityPoll time.Duration
	// SessionPoll overrides the mid-tail session-liveness cadence (0 default).
	SessionPoll time.Duration
	// SessionReattachWindow overrides how long a missing session is tolerated
	// before failing the run (0 uses the production default of 15 minutes).
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
	recoveryCanReattach     atomic.Bool
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

// TailContext carries durable per-run facts the tailer needs to re-discover a
// rotated rollout after a continuation or restart recovery.
type TailContext struct {
	RunDir     string
	WorkingDir string
	LaunchedAt time.Time
}

func (c *Coordinator) persistTranscriptPath(ctx context.Context, run *domain.Run, path string) error {
	if path == "" || path == run.TranscriptPath {
		return nil
	}
	if c.deps.Runs == nil {
		return fmt.Errorf("interactive transcript path change requires durable run owner")
	}
	previousPath, previousCursor := run.TranscriptPath, run.TranscriptCursor
	run.TranscriptPath = path
	run.TranscriptCursor = 0
	if err := c.deps.Runs.Update(ctx, run); err != nil {
		run.TranscriptPath = previousPath
		run.TranscriptCursor = previousCursor
		return err
	}
	return nil
}

// refreshNewerInvocation reloads the exact retained run before an observed
// terminal can close it. A continuation may be accepted while this coordinator
// is still draining the preceding turn; that durable invocation fence makes the
// earlier terminal stale until the new turn is consumed.
func (c *Coordinator) refreshNewerInvocation(ctx context.Context, run *domain.Run) (bool, error) {
	if run == nil || run.InteractiveInvocationStartedAt == nil {
		return false, nil
	}
	current, err := c.deps.Runs.Get(ctx, run.ID)
	if err != nil {
		return false, err
	}
	if current == nil || current.ID != run.ID || current.TaskID != run.TaskID ||
		current.SessionID != run.SessionID || current.WebConsoleSessionID != run.WebConsoleSessionID {
		return false, ErrRunOwnerUnavailable
	}
	if !current.Status.IsActive() {
		*run = *current
		return false, ErrRunOwnerChanged
	}
	if current.InteractiveInvocationStartedAt == nil {
		return false, ErrRunOwnerUnavailable
	}
	if current.InteractiveInvocationStartedAt.Equal(*run.InteractiveInvocationStartedAt) {
		return false, nil
	}
	if !current.InteractiveInvocationStartedAt.After(*run.InteractiveInvocationStartedAt) {
		return false, ErrRunOwnerUnavailable
	}
	*run = *current
	return true, nil
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

	// Reserve the exact session relation durably before any Create submission.
	// Retention can then join the starting run even before a response arrives.
	expectedSessionID := uuid.NewSHA1(run.ID, []byte("agent-manager:interactive-session:v1")).String()
	if run.WebConsoleSessionID != "" {
		return fmt.Errorf("interactive launch already has a different session binding; recover that owner instead")
	}
	previous := run.WebConsoleSessionID
	run.WebConsoleSessionID = expectedSessionID
	if c.deps.Runs == nil {
		run.WebConsoleSessionID = previous
		return fmt.Errorf("interactive launch requires durable run owner")
	}
	if err := c.deps.Runs.Update(ctx, run); err != nil {
		run.WebConsoleSessionID = previous
		return fmt.Errorf("persist reserved interactive session: %w", err)
	}
	p.RequestedSessionID = expectedSessionID
	if c.deps.MintIdentityToken != nil {
		token, mintErr := c.deps.MintIdentityToken(ctx, run, expectedSessionID)
		if mintErr != nil {
			return c.finalizeFailed(ctx, run, fmt.Sprintf("mint interactive child identity: %v", mintErr))
		}
		if strings.TrimSpace(token) == "" {
			return c.finalizeFailed(ctx, run, "mint interactive child identity returned no token")
		}
		p.AgentRunIdentityToken = token
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
	priorInvocation := run.InteractiveInvocationStartedAt
	run.Status = domain.RunStatusRunning
	run.Phase = domain.RunPhaseExecuting
	if run.StartedAt == nil {
		run.StartedAt = &now
	}
	run.InteractiveInvocationStartedAt = run.StartedAt
	run.LastHeartbeat = &now
	run.UpdatedAt = now
	mu.Unlock()
	if run.InteractiveInvocationStartedAt != nil {
		updated, err := c.deps.Runs.BeginInteractiveInvocation(ctx, run, domain.RunStatusStarting, priorInvocation, *run.InteractiveInvocationStartedAt)
		if err != nil {
			return errors.Join(
				fmt.Errorf("persist initial interactive invocation fence: %w", err),
				c.cleanupUnadmittedInitialSession(run, priorInvocation, res.SessionID, p.RunnerType, p.RunDir),
			)
		}
		if !updated {
			return errors.Join(
				fmt.Errorf("initial interactive invocation owner changed before it reached Running"),
				c.cleanupUnadmittedInitialSession(run, priorInvocation, res.SessionID, p.RunnerType, p.RunDir),
			)
		}
	}
	if uerr := c.update(ctx, &mu, run); uerr != nil {
		return uerr
	}
	c.broadcast(run)
	if onRunning != nil {
		onRunning()
	}

	tc := TailContext{RunDir: p.RunDir, WorkingDir: p.WorkingDir, LaunchedAt: now}
	return c.TailAndFinalizeLive(ctx, run, tc, TailCompletionOptions{})
}

// cleanupUnadmittedInitialSession stops a launched session only while the
// durable row still names this exact session and its pre-launch invocation.
// A failed/ambiguous CAS may mean cancellation won, but a newer invocation or
// session must never be stopped by the stale launcher.
func (c *Coordinator) cleanupUnadmittedInitialSession(run *domain.Run, priorInvocation *time.Time, sessionID string, runnerType domain.RunnerType, runDir string) error {
	if run == nil || c.deps.Runs == nil || c.deps.Substrate == nil || sessionID == "" {
		return nil
	}
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	current, err := c.deps.Runs.Get(cleanupCtx, run.ID)
	if err != nil {
		return fmt.Errorf("verify unadmitted interactive session owner before cleanup: %w", err)
	}
	if current == nil || current.TaskID != run.TaskID || current.WebConsoleSessionID != sessionID ||
		!nullableTimesEqual(current.InteractiveInvocationStartedAt, priorInvocation) ||
		(current.Status != domain.RunStatusStarting && current.Status != domain.RunStatusCancelled && current.Status != domain.RunStatusFailed) {
		return nil
	}
	if err := c.deps.Substrate.Stop(cleanupCtx, sessionID, interactiveSource(current)); err != nil {
		return fmt.Errorf("stop unadmitted interactive session %s: %w", sessionID, err)
	}
	if err := c.deps.Substrate.CleanupCredentials(runnerType, runDir); err != nil {
		return fmt.Errorf("clean credentials after unadmitted interactive session: %w", err)
	}
	return nil
}

func nullableTimesEqual(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

// TailToCompletion drains the run's agent-owned transcript from its persisted
// cursor and follows it to a run boundary. It is the reusable tail core for
// live execution and restart recovery; the live wrapper additionally owns
// heartbeat refresh. Every success terminal is a turn boundary, and the run
// completes only once the transcript stops growing for the debounce window
// (design R2). It returns the last terminal marker seen and a non-nil error
// only for a vanished session (ErrSessionGone) or a cancelled context.
func (c *Coordinator) TailToCompletion(ctx context.Context, run *domain.Run) (*runner.TranscriptTerminal, error) {
	return c.TailToCompletionWithOptions(ctx, run, c.deps.CompletionOptions)
}

// TailCompletionOptions narrows completion behavior for a specific invocation
// without changing the normal live/recovery tail contract.
type TailCompletionOptions struct {
	// CompleteAfterFirstSuccessfulTurnWithoutGoalMarker is used only when a
	// terminal native goal (complete or blocked) is continued without
	// reinstalling that goal. The first successful turn in the new invocation
	// then closes the continued run only if no new goal marker is observed.
	CompleteAfterFirstSuccessfulTurnWithoutGoalMarker bool
}

// TailToCompletionWithOptions drains an interactive invocation with an
// invocation-scoped completion policy. Existing callers use TailToCompletion
// and keep the default behavior.
func (c *Coordinator) TailToCompletionWithOptions(ctx context.Context, run *domain.Run, options TailCompletionOptions) (*runner.TranscriptTerminal, error) {
	return c.TailToCompletionWithContext(ctx, run, TailContext{}, options)
}

// TailToCompletionWithContext drains an invocation with persisted per-run
// discovery context. It does not refresh LastHeartbeat; restart recovery uses
// this path because it observes an existing owner rather than owning its live
// execution.
func (c *Coordinator) TailToCompletionWithContext(ctx context.Context, run *domain.Run, tailCtx TailContext, options TailCompletionOptions) (*runner.TranscriptTerminal, error) {
	var mu sync.Mutex
	return c.tailToCompletion(ctx, run, tailCtx, &mu, options)
}

// TailLiveToCompletionWithOptions drives a live invocation with the same
// heartbeat lifecycle used by initial Execute and interactive continuation.
// The heartbeat stops and joins before this method returns.
func (c *Coordinator) TailLiveToCompletionWithOptions(ctx context.Context, run *domain.Run, tailCtx TailContext, options TailCompletionOptions) (*runner.TranscriptTerminal, error) {
	var mu sync.Mutex
	stopHeartbeat := c.startHeartbeat(ctx, &mu, run)
	defer stopHeartbeat()
	return c.tailToCompletion(ctx, run, tailCtx, &mu, options)
}

// tailToCompletion follows the transcript to a run boundary. When the run's
// timeout expires it asks the harness to stop its turn and keeps reading for a
// bounded grace, so the harness's closing record (codex writes turn_aborted,
// which its parser turns into the turn's usage receipt) is stored before the run
// is finalized. The result still reports the timeout.
func (c *Coordinator) tailToCompletion(ctx context.Context, run *domain.Run, tc TailContext, mu *sync.Mutex, options TailCompletionOptions) (*runner.TranscriptTerminal, error) {
	loopCtx, timedOut, release := c.timeoutInterruptContext(ctx, run)
	defer release()
	terminal, err := c.tailLoop(loopCtx, run, tc, mu, options)
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

// TailAndFinalizeLive keeps the active owner attached when an accepted
// continuation supersedes a terminal already buffered by this tailer.
func (c *Coordinator) TailAndFinalizeLive(ctx context.Context, run *domain.Run, tc TailContext, options TailCompletionOptions) error {
	return c.tailAndFinalize(ctx, run, tc, options, true)
}

// TailAndFinalize reattaches without a live heartbeat, as used by restart
// recovery. Both paths share the same invocation-fence and terminal commit.
func (c *Coordinator) TailAndFinalize(ctx context.Context, run *domain.Run, tc TailContext, options TailCompletionOptions) error {
	return c.tailAndFinalize(ctx, run, tc, options, false)
}

func (c *Coordinator) tailAndFinalize(ctx context.Context, run *domain.Run, tc TailContext, options TailCompletionOptions, live bool) error {
	for {
		var terminal *runner.TranscriptTerminal
		var tailErr error
		if live {
			terminal, tailErr = c.TailLiveToCompletionWithOptions(ctx, run, tc, options)
		} else {
			terminal, tailErr = c.TailToCompletionWithContext(ctx, run, tc, options)
		}
		finalErr := c.Finalize(ctx, run, terminal, tailErr)
		if !errors.Is(finalErr, ErrRunOwnerChanged) || !run.Status.IsActive() || ctx.Err() != nil {
			if errors.Is(finalErr, ErrRunOwnerChanged) && !run.Status.IsActive() {
				return nil
			}
			return finalErr
		}
	}
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

func (c *Coordinator) tailLoop(ctx context.Context, run *domain.Run, tc TailContext, mu *sync.Mutex, options TailCompletionOptions) (*runner.TranscriptTerminal, error) {
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
		// TranscriptCursor is an offset in TranscriptPath, not a global byte
		// counter. Tailer commits path changes before reading the new file, so
		// every advance here belongs to the currently selected file.
		cursor = cur
		run.TranscriptCursor = cur
		if lastSeq > run.TranscriptLastSeq {
			run.TranscriptLastSeq = lastSeq
		}
		updated, err := c.deps.Runs.UpdateRunnerStreamState(context.Background(), run)
		if err != nil {
			return err
		}
		if !updated {
			return ErrRunOwnerChanged
		}
		return nil
	}
	onTranscriptPath := func(path string) error {
		mu.Lock()
		defer mu.Unlock()
		if path == "" || path == run.TranscriptPath {
			return nil
		}
		previousLocalCursor := cursor
		if err := c.persistTranscriptPath(context.Background(), run, path); err != nil {
			cursor = previousLocalCursor
			return err
		}
		cursor = 0
		return nil
	}
	onSessionID := func(sessionID string) error {
		mu.Lock()
		defer mu.Unlock()
		if sessionID == "" || run.SessionID == sessionID {
			return nil
		}
		run.SessionID = sessionID
		updated, err := c.deps.Runs.UpdateRunnerStreamState(context.Background(), run)
		if err != nil {
			return err
		}
		if !updated {
			return ErrRunOwnerChanged
		}
		return nil
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
		gone := c.watchSession(tctx, ctx, cancel, run)

		mu.Lock()
		startCursor := cursor
		mu.Unlock()

		term, err := c.deps.Tailer.Tail(tctx, TailParams{
			RunID:            run.ID,
			RunnerType:       run.ResolvedConfig.RunnerType,
			Model:            transcriptModel(run),
			TranscriptPath:   run.TranscriptPath,
			SessionID:        run.SessionID,
			Until:            run.ResolvedConfig.Until,
			MaxTurns:         run.ResolvedConfig.MaxTurns,
			Billing:          run.Billing,
			RunDir:           tc.RunDir,
			WorkingDir:       tc.WorkingDir,
			LaunchedAt:       tc.LaunchedAt,
			StartCursor:      startCursor,
			GoalDrainTimeout: c.debounce,
			Sink:             sink,
			OnAdvance:        onAdvance,
			OnTranscriptPath: onTranscriptPath,
			OnSessionID:      onSessionID,
			OnGoalStatus:     onGoalStatus,
			GoalMarkerObserved: func() bool {
				return lastGoal != nil
			},
			StructuredResultSatisfied: func() bool {
				return c.structuredResultSatisfied(ctx, run)
			},
			CompleteAfterFirstSuccessfulTurnWithoutGoalMarker: options.CompleteAfterFirstSuccessfulTurnWithoutGoalMarker,
		})
		cancel()

		if gone() {
			// The session vanished mid-tail: prefer the freshest terminal seen.
			return firstTerminal(term, lastTerminal), ErrSessionGone
		}
		if err != nil {
			if errors.Is(err, ErrRunOwnerChanged) {
				newer, refreshErr := c.refreshNewerInvocation(ctx, run)
				if refreshErr == nil && newer {
					lastTerminal = nil
					continue
				}
				if refreshErr != nil {
					return nil, fmt.Errorf("%w: %v", ErrRunOwnerUnavailable, refreshErr)
				}
				return nil, ErrRunOwnerChanged
			}
			if c.recoverableInterruption.Load() {
				// The watcher cancelled the tail so the coordinator can wait for
				// the delayed retry. Keep this coordinator attached when a fresh
				// check proves it is safe to observe the same session again.
				select {
				case <-c.recoveryReady:
					if c.recoveryCanReattach.Load() {
						c.resetInterruptionRecoveryCycle()
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
		newer, refreshErr := c.refreshNewerInvocation(ctx, run)
		if refreshErr != nil {
			return nil, fmt.Errorf("%w: %v", ErrRunOwnerUnavailable, refreshErr)
		}
		if newer {
			lastTerminal = nil
			continue
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
			newer, refreshErr = c.refreshNewerInvocation(ctx, run)
			if refreshErr != nil {
				return nil, fmt.Errorf("%w: %v", ErrRunOwnerUnavailable, refreshErr)
			}
			if newer {
				lastTerminal = nil
				continue
			}
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
func (c *Coordinator) awaitNextTurn(ctx context.Context, run *domain.Run, tc TailContext, boundary int64) bool {
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
func (c *Coordinator) transcriptGrew(run *domain.Run, tc TailContext, boundary int64) bool {
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
func (c *Coordinator) watchSession(ctx, recoveryCtx context.Context, cancel context.CancelFunc, run *domain.Run) func() bool {
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
								c.scheduleInterruptionRecovery(recoveryCtx, run, interruption)
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

// resetInterruptionRecoveryCycle clears only the completed cycle after the
// tail loop observed its closed result channel. A later distinct interruption
// can then own a fresh timer without duplicating the completed cycle's input.
func (c *Coordinator) resetInterruptionRecoveryCycle() {
	c.recoverableInterruption.Store(false)
	c.recoveryCanReattach.Store(false)
	c.recoveryScheduled.Store(false)
	c.recoveryReady = make(chan struct{})
}

// scheduleInterruptionRecovery keeps a run alive while a server-owned timer
// waits, then sends exactly one mode-aware continuation into the existing Web
// Console session if the same run is still active and the same interruption is
// still on screen. Parent cancellation cancels the timer; repeated screen polls
// cannot duplicate the prompt.
func (c *Coordinator) scheduleInterruptionRecovery(parentCtx context.Context, run *domain.Run, interruption ResumableInterruption) {
	if run == nil || run.WebConsoleSessionID == "" || c.deps.Sessions == nil || !c.recoveryScheduled.CompareAndSwap(false, true) {
		return
	}
	runID := run.ID
	sessionID := run.WebConsoleSessionID
	go func() {
		defer close(c.recoveryReady)
		timer := time.NewTimer(c.recoveryDelay)
		defer timer.Stop()
		select {
		case <-parentCtx.Done():
			return
		case <-timer.C:
		}
		ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
		defer cancel()
		reader, ok := c.deps.Runs.(interface {
			Get(context.Context, uuid.UUID) (*domain.Run, error)
		})
		if !ok {
			return
		}
		current, err := reader.Get(ctx, runID)
		if err != nil || current == nil || current.ID != runID || !current.Status.IsActive() ||
			current.WebConsoleSessionID == "" || current.WebConsoleSessionID != sessionID {
			return
		}
		session, err := c.deps.Sessions.GetSession(ctx, current.WebConsoleSessionID)
		if err != nil || session.ID != current.WebConsoleSessionID {
			return
		}
		screen, err := c.deps.Sessions.Screen(ctx, current.WebConsoleSessionID, true)
		if err != nil || ctx.Err() != nil {
			return
		}
		currentInterruption, detected := ClassifyScreenInterruption(screen)
		if !detected || currentInterruption != interruption {
			c.recoveryCanReattach.Store(true)
			return
		}
		hasNativeGoal := current.ResolvedConfig != nil && strings.TrimSpace(current.ResolvedConfig.Until) != ""
		if interruption.Kind == "goal_stalled" && !hasNativeGoal {
			return
		}
		if ctx.Err() != nil {
			return
		}
		if c.deps.NewSink != nil {
			if events := c.deps.NewSink(runID); events != nil {
				_ = events.Emit(domain.NewLogEvent(runID, "warn", fmt.Sprintf(
					"resumable interactive interruption detected (%s: %s); attempting delayed recovery",
					interruption.Kind, interruption.Message)))
				_ = events.Close()
			}
		}
		if ctx.Err() != nil {
			return
		}
		if hasNativeGoal {
			submission, err := c.deps.Sessions.SendCommand(ctx, current.WebConsoleSessionID, "/goal resume", interactiveSource(current))
			if err != nil || !submission.Verified {
				return
			}
		} else {
			submission, err := c.deps.Sessions.SendPrompt(ctx, current.WebConsoleSessionID, "continue", interactiveSource(current))
			if err != nil || !submission.Verified {
				return
			}
		}
		c.recoveryCanReattach.Store(true)
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
//
// Finalize never deletes the web-console session: it stays live so `run
// continue` can type a follow-up into it. The Reconciler bounds that retention
// (releaseRetainedInteractiveSessions) and archives the session once the run has
// been ended for ReconcilerConfig.InteractiveSessionRetention; continuing after
// that requires a new run.
func (c *Coordinator) Finalize(ctx context.Context, run *domain.Run, terminal *runner.TranscriptTerminal, tailErr error) error {
	if errors.Is(tailErr, ErrRunOwnerChanged) || errors.Is(tailErr, ErrRunOwnerUnavailable) {
		return tailErr
	}
	if errors.Is(tailErr, ErrResumableInterruption) {
		return nil
	}
	if tailErr != nil && errors.Is(tailErr, context.Canceled) && !errors.Is(tailErr, ErrSessionGone) {
		return nil
	}

	var finalizeErr error
	switch {
	case terminal != nil && terminal.Success:
		finalizeErr = c.finalizeComplete(ctx, run, terminal)
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
		finalizeErr = c.finalizeFailed(ctx, run, msg)
	case errors.Is(tailErr, ErrSessionGone):
		run.TerminalClass = domain.RunTerminalClassInterruption
		run.StopReason = domain.RunStopReasonSessionLost
		finalizeErr = c.finalizeFailed(ctx, run, fmt.Sprintf(
			"web-console session %s no longer exists; interactive run cannot be recovered", run.WebConsoleSessionID))
	case errors.Is(tailErr, context.DeadlineExceeded):
		// The run hit its configured timeout ceiling. Phase 3 types this as the
		// timeout interruption (terminal_class=interruption, stop_reason=timeout)
		// with the last handoff attached; the terminal reason is recorded now.
		finalizeErr = c.finalizeTimeout(ctx, run)
	default:
		finalizeErr = c.finalizeFailed(ctx, run, "interactive run ended without a terminal marker")
	}
	if !isInvocationFenceConflict(finalizeErr) {
		return finalizeErr
	}
	current, err := c.deps.Runs.Get(ctx, run.ID)
	if err != nil {
		return fmt.Errorf("%w: read current run after terminal compare-and-set refused: %v", ErrRunOwnerUnavailable, err)
	}
	if current == nil || current.ID != run.ID || current.TaskID != run.TaskID ||
		current.SessionID != run.SessionID || current.WebConsoleSessionID != run.WebConsoleSessionID {
		return ErrRunOwnerUnavailable
	}
	if current.Status.IsActive() && current.InteractiveInvocationStartedAt != nil && run.InteractiveInvocationStartedAt != nil &&
		current.InteractiveInvocationStartedAt.Equal(*run.InteractiveInvocationStartedAt) {
		return ErrRunOwnerUnavailable
	}
	*run = *current
	if !current.Status.IsActive() {
		return nil
	}
	return ErrRunOwnerChanged
}

func isInvocationFenceConflict(err error) bool {
	var stateErr *domain.StateError
	return errors.As(err, &stateErr) && stateErr.Operation == "update" && stateErr.Reason == "interactive invocation changed; stale observer refused"
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
