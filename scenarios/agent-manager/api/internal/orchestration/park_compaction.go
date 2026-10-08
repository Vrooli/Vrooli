// This file compacts the native session of a run that stays parked, so its
// wake does not re-send the whole conversation after the prompt cache expired.
package orchestration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
)

// A wake re-sends the parked conversation. Provider prompt caches expire
// (about 30 minutes for the models measured), so a run woken later pays for its
// whole context again; on a BAS orchestrator those wakes cost about 40% of its
// tokens. Compacting a long-parked session while the cache is still warm makes
// the compaction itself cheap and every later wake start from a summary. How a
// session is compacted belongs to its runner ([runner.SessionCompactor]).
const (
	// DefaultParkCompactionDelay is how long a run stays parked before its
	// session is compacted. Short parks (a child that ends within minutes)
	// keep their full context.
	DefaultParkCompactionDelay = 20 * time.Minute
	// DefaultParkCompactionMinContextTokens skips sessions too small for a
	// summary to save anything.
	DefaultParkCompactionMinContextTokens = 40_000
	// parkCompactionTimeout bounds one compaction; a wake waits at most this
	// long for it.
	parkCompactionTimeout = 5 * time.Minute
)

// WithParkCompaction overrides when parked sessions are compacted. A zero or
// negative delay disables compaction.
func WithParkCompaction(delay time.Duration, minContextTokens int64) Option {
	return func(o *Orchestrator) {
		o.parkCompaction = &parkCompactionSettings{delay: delay, minContextTokens: minContextTokens}
	}
}

type parkCompactionSettings struct {
	delay            time.Duration
	minContextTokens int64
}

func (o *Orchestrator) parkCompactionSettings() parkCompactionSettings {
	if o.parkCompaction != nil {
		return *o.parkCompaction
	}
	return parkCompactionSettings{delay: DefaultParkCompactionDelay, minContextTokens: DefaultParkCompactionMinContextTokens}
}

// parkCompactionEligible reports whether a parked run's session may be
// compacted out of band: a codec-pipe run with a session. An interactive
// session stays owned by its live terminal and native goal loop (resuming a
// Codex interactive thread elsewhere clears its goal).
func parkCompactionEligible(run *domain.Run) bool {
	return run != nil && run.ResolvedConfig != nil &&
		run.ExecutionMode.Normalized() == domain.ExecutionModeCodecPipe &&
		run.SessionID != "" && run.AwaitHandle != nil
}

// sessionCompactor returns the run's runner when it declares session
// compaction.
func (o *Orchestrator) sessionCompactor(run *domain.Run) (runner.SessionCompactor, bool) {
	if o.runners == nil || run == nil || run.ResolvedConfig == nil {
		return nil, false
	}
	r, err := o.runners.Get(run.ResolvedConfig.RunnerType)
	if err != nil || r == nil || !r.Capabilities().SupportsSessionCompaction {
		return nil, false
	}
	compactor, ok := r.(runner.SessionCompactor)
	return compactor, ok
}

// scheduleParkCompaction arranges for a just-parked run's session to be
// compacted if the run is still parked on the same handle after the delay.
func (o *Orchestrator) scheduleParkCompaction(run *domain.Run) {
	settings := o.parkCompactionSettings()
	if settings.delay <= 0 || !parkCompactionEligible(run) {
		return
	}
	if _, ok := o.sessionCompactor(run); !ok {
		return
	}
	due := run.AwaitHandle.RegisteredAt.Add(settings.delay)
	if run.AwaitHandle.Deadline != nil && !run.AwaitHandle.Deadline.After(due) {
		// The timer wake comes first; compaction would not run before it.
		return
	}
	runID, registeredAt := run.ID, run.AwaitHandle.RegisteredAt
	wait := due.Sub(o.now())
	if wait < 0 {
		wait = 0
	}
	time.AfterFunc(wait, func() {
		defer obs.RecoverToFailure("park compaction", nil)
		o.compactParkedRun(runID, registeredAt)
	})
}

// RecoverParkCompactions re-arms compaction for runs that were parked when
// agent-manager restarted.
func (o *Orchestrator) RecoverParkCompactions(ctx context.Context) error {
	runs, err := o.ListParkedRuns(ctx)
	if err != nil {
		return err
	}
	for _, run := range runs {
		o.scheduleParkCompaction(run)
	}
	return nil
}

// compactParkedRun compacts the run's session when it is still parked on the
// handle registered at registeredAt. A wake that arrives meanwhile waits for
// the compaction (awaitParkCompaction) instead of resuming beside it.
func (o *Orchestrator) compactParkedRun(runID uuid.UUID, registeredAt time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), parkCompactionTimeout)
	defer cancel()

	o.wakeMu.Lock()
	run, err := o.GetRun(ctx, runID)
	if err != nil || run == nil || run.Status != domain.RunStatusParked || run.AwaitHandle == nil ||
		!run.AwaitHandle.RegisteredAt.Equal(registeredAt) || !parkCompactionEligible(run) {
		o.wakeMu.Unlock()
		return
	}
	done := make(chan struct{})
	o.parkCompactions.Store(runID, done)
	o.wakeMu.Unlock()
	defer func() {
		o.parkCompactions.CompareAndDelete(runID, done)
		close(done)
	}()

	o.awaitParkedTurnEnd(ctx, runID)
	result, err := o.compactRunSession(ctx, run)
	o.recordParkCompaction(ctx, runID, result, err)
}

func (o *Orchestrator) compactRunSession(ctx context.Context, run *domain.Run) (*runner.CompactSessionResult, error) {
	compactor, ok := o.sessionCompactor(run)
	if !ok {
		return nil, runner.ErrCompactionNotSupported
	}
	root, err := o.resolveRunStateRoot(ctx)
	if err != nil {
		return nil, err
	}
	env, err := PrepareCodecSessionHome(root, run.ID, run.ResolvedConfig.RunnerType)
	if err != nil {
		return nil, err
	}
	workDir := ""
	if task, err := o.tasks.Get(ctx, run.TaskID); err == nil && task != nil {
		workDir = task.ProjectRoot
	}
	return compactor.CompactSession(ctx, runner.CompactSessionRequest{
		RunID:            run.ID,
		SessionID:        run.SessionID,
		Model:            run.ResolvedConfig.Model,
		WorkingDir:       workDir,
		Env:              env,
		MinContextTokens: o.parkCompactionSettings().minContextTokens,
	})
}

// recordParkCompaction leaves the outcome on the run's event log. A failed
// compaction is not fatal: the wake simply resumes the uncompacted session.
func (o *Orchestrator) recordParkCompaction(ctx context.Context, runID uuid.UUID, result *runner.CompactSessionResult, err error) {
	level, message := "info", ""
	switch {
	case errors.Is(err, runner.ErrCompactionNotSupported):
		return
	case err != nil:
		level, message = "warn", fmt.Sprintf("parked session compaction failed; the wake resumes the full session: %v", err)
	case result != nil && result.Compacted:
		message = fmt.Sprintf("compacted the parked session (context was %d tokens)", result.ContextTokens)
	case result != nil:
		message = "parked session not compacted: " + result.Reason
	default:
		return
	}
	obs.Component("park").Info("park compaction", obs.KeyRunID, runID.String(), "outcome", message)
	if o.events != nil {
		_ = o.events.Append(ctx, runID, domain.NewLogEvent(runID, level, message))
	}
}

// lockForWake takes wakeMu once no compaction of runID is in flight; the
// caller unlocks it.
func (o *Orchestrator) lockForWake(ctx context.Context, runID uuid.UUID) error {
	for {
		o.wakeMu.Lock()
		inFlight, busy := o.parkCompactions.Load(runID)
		if !busy {
			return nil
		}
		o.wakeMu.Unlock()
		select {
		case <-inFlight.(chan struct{}):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
