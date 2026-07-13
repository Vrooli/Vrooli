package interactive

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
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
	// defaultCoordinatorHeartbeat is how often the live path refreshes
	// Run.LastHeartbeat so the reconciler does not treat a live interactive run
	// as stale.
	defaultCoordinatorHeartbeat = 30 * time.Second
)

// ErrSessionGone is returned by [Coordinator.TailToCompletion] when the
// web-console session hosting the interactive CLI disappeared mid-tail. It is a
// distinct sentinel (not context.Canceled) so [Coordinator.Finalize] can
// distinguish a vanished session (finalize the run) from a graceful shutdown
// (leave the run for restart recovery).
var ErrSessionGone = errors.New("web-console session no longer exists")

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
	// Clock overrides time.Now for run timestamps (tests). Nil uses time.Now.
	Clock func() time.Time
	// Debounce overrides the turn-boundary idle window (0 uses the default).
	Debounce time.Duration
	// ActivityPoll overrides the debounce growth-check cadence (0 uses default).
	ActivityPoll time.Duration
	// SessionPoll overrides the mid-tail session-liveness cadence (0 default).
	SessionPoll time.Duration
	// Heartbeat overrides the live-path heartbeat cadence (0 uses the default;
	// negative disables the heartbeat goroutine — used by the recovery path,
	// which does not own the live run).
	Heartbeat time.Duration
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
}

// NewCoordinator builds a Coordinator, applying defaults for any unset cadence.
func NewCoordinator(deps CoordinatorDeps) *Coordinator {
	c := &Coordinator{
		deps:         deps,
		clock:        deps.Clock,
		debounce:     deps.Debounce,
		activityPoll: deps.ActivityPoll,
		sessionPoll:  deps.SessionPoll,
		heartbeat:    deps.Heartbeat,
	}
	if c.clock == nil {
		c.clock = time.Now
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

func (c *Coordinator) tailToCompletion(ctx context.Context, run *domain.Run, tc tailContext, mu *sync.Mutex) (*runner.TranscriptTerminal, error) {
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
			TranscriptPath: run.TranscriptPath,
			RunDir:         tc.RunDir,
			WorkingDir:     tc.WorkingDir,
			LaunchedAt:     tc.LaunchedAt,
			StartCursor:    startCursor,
			Sink:           sink,
			OnAdvance:      onAdvance,
			OnSessionID:    onSessionID,
		})
		cancel()

		if gone() {
			// The session vanished mid-tail: prefer the freshest terminal seen.
			return firstTerminal(term, lastTerminal), ErrSessionGone
		}
		if err != nil {
			// The caller cancelled (shutdown); leave completion to recovery.
			return firstTerminal(term, lastTerminal), err
		}
		if term == nil {
			return lastTerminal, nil
		}
		if !term.Success {
			return term, nil
		}

		// A success terminal is a turn boundary. Wait out the idle-debounce: if
		// the transcript grows, a new turn began and we keep tailing; otherwise
		// the run is complete.
		lastTerminal = term
		mu.Lock()
		boundary := cursor
		mu.Unlock()
		if !c.awaitNextTurn(ctx, run, tc, boundary) {
			return term, nil
		}
	}
}

// awaitNextTurn blocks until the transcript grows beyond the turn-boundary
// cursor (a new turn began → true) or the debounce window elapses with no growth
// (the run is idle → false). Growth is measured on the file, not the consumed
// cursor, because nothing is reading the file between turns. claude writes its
// interactive-only records (file-history-snapshot/ai-title/last-prompt) only at
// the start of a NEW turn, so file growth is a faithful new-turn signal.
func (c *Coordinator) awaitNextTurn(ctx context.Context, run *domain.Run, tc tailContext, boundary int64) bool {
	deadline := time.Now().Add(c.debounce)
	for {
		if c.transcriptGrew(run, tc, boundary) {
			return true
		}
		if !time.Now().Before(deadline) {
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
	go func() {
		ticker := time.NewTicker(c.sessionPoll)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, err := c.deps.Sessions.GetSession(ctx, run.WebConsoleSessionID)
				if errors.Is(err, webconsole.ErrSessionNotFound) {
					mu.Lock()
					gone = true
					mu.Unlock()
					cancel()
					return
				}
			}
		}
	}()
	return func() bool {
		mu.Lock()
		defer mu.Unlock()
		return gone
	}
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
		return c.finalizeFailed(ctx, run, msg)
	case errors.Is(tailErr, ErrSessionGone):
		return c.finalizeFailed(ctx, run, fmt.Sprintf(
			"web-console session %s no longer exists; interactive run cannot be recovered", run.WebConsoleSessionID))
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
	exit := terminal.ExitCode
	run.ExitCode = &exit
	if summary := c.buildSummary(ctx, run, terminal); summary != nil {
		run.Summary = summary
	}
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
	run.EndedAt = &now
	run.UpdatedAt = now
	mu.Unlock()
	if err := c.update(ctx, &mu, run); err != nil {
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
