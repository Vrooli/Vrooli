package goals

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// DefaultWorkflowSweepInterval is how often the goal workflow sweeper looks for
// terminal results to apply.
const DefaultWorkflowSweepInterval = 2 * time.Minute

// applyCollectTimeout bounds one apply attempt. CollectWorkflow performs a
// server-side blocking wait, so an unbounded sweep over many in-flight runs
// would serialise into a long stall.
const applyCollectTimeout = 45 * time.Second

// WorkflowSweeper drives the goal workflow apply hop.
//
// A goal transition (goal.plan, goal.discover, milestone.review) starts an agent
// workflow and leaves a correlation record behind. Applying the terminal result
// — decoding the verdict, stamping verified-delivered, recording the proposals —
// is a separate, explicitly-triggered step, because applying a result mutates
// operator-visible state and must not be a side effect of generic housekeeping
// (see archtest.TestExecutionHousekeepingDoesNotApplyWorkflowResults).
//
// That left goals with no trigger at all: every result ever produced sat
// unapplied on disk. This sweeper is the missing owner, and it is deliberately
// the goal domain's own component rather than a hook in execution housekeeping,
// so the apply hop stays where the domain rules live. It mirrors review.Sweeper,
// which plays the same role for backlog review rounds.
type WorkflowSweeper struct {
	Handler  *Handler
	Interval time.Duration
}

// NewWorkflowSweeper constructs a sweeper for the goal workflow apply hop.
func NewWorkflowSweeper(handler *Handler) *WorkflowSweeper {
	return &WorkflowSweeper{
		Handler:  handler,
		Interval: envDuration("SWARM_MANAGER_GOAL_WORKFLOW_SWEEP_INTERVAL", DefaultWorkflowSweepInterval),
	}
}

// RunOnce applies every terminal goal workflow result and returns how many
// landed. Per-record errors are logged and skipped so one stuck correlation
// cannot block the rest.
func (s *WorkflowSweeper) RunOnce(ctx context.Context) (int, error) {
	if s == nil || s.Handler == nil || s.Handler.workflow == nil || s.Handler.proposalRecorder == nil {
		return 0, nil
	}
	pending, err := s.Handler.ListPendingWorkflows()
	if err != nil {
		return 0, err
	}
	applied := 0
	for _, record := range pending {
		if record.Stale {
			// Apply refuses these by design. Retrying cannot help, and the
			// listing already reports the reason, so skip without noise.
			continue
		}
		if s.applyOne(ctx, record) {
			applied++
		}
	}
	if applied > 0 {
		slog.Info("[goals] workflow sweeper: applied results", "count", applied)
	}
	return applied, nil
}

// applyOne applies a single correlation and reports whether it landed.
func (s *WorkflowSweeper) applyOne(ctx context.Context, record PendingWorkflow) bool {
	attemptCtx, cancel := context.WithTimeout(ctx, applyCollectTimeout)
	defer cancel()

	result, err := s.Handler.applyWorkflow(attemptCtx, record.GoalName, record.ExecutionID)
	if err == nil {
		slog.Info("[goals] workflow sweeper: applied",
			"goal", record.GoalName, "transition", record.Transition,
			"execution_id", record.ExecutionID, "outcome", result.Outcome)
		return true
	}
	if isTransientApplyFailure(err) {
		// Still running, or the engine is unreachable. Either way the record is
		// healthy and will be retried; marking it would be a false accusation.
		return false
	}
	s.Handler.recordApplyFailure(record.GoalName, record.ExecutionID, err)
	// Log only when the failure is new or has changed. A permanently stuck
	// correlation would otherwise reprint the same line on every tick forever.
	if strings.TrimSpace(record.LastError) != strings.TrimSpace(err.Error()) {
		slog.Warn("[goals] workflow sweeper: apply failed",
			"goal", record.GoalName, "transition", record.Transition,
			"execution_id", record.ExecutionID, "attempts", record.Attempts+1, "err", err)
	}
	return false
}

// isTransientApplyFailure reports whether an apply error means "ask again
// later" rather than "this correlation is broken". Only the second kind is
// worth recording against the record or logging.
func isTransientApplyFailure(err error) bool {
	return errors.Is(err, ErrWorkflowNotReady) ||
		errors.Is(err, ErrWorkflowUnavailable) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled)
}

// Start runs RunOnce on a ticker until ctx is cancelled. Run it in its own
// goroutine. Panics are recovered so a transient disk fault cannot take the
// apply hop down for the life of the process.
func (s *WorkflowSweeper) Start(ctx context.Context) {
	if s == nil || s.Interval <= 0 {
		return
	}
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runWithRecover(ctx)
		}
	}
}

// RunOnceLogged is the boot-time entry point: one pass whose failure is logged
// rather than returned, since nothing at startup can act on the error.
func (s *WorkflowSweeper) RunOnceLogged(ctx context.Context) { s.runWithRecover(ctx) }

func (s *WorkflowSweeper) runWithRecover(ctx context.Context) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("[goals] workflow sweeper: panic recovered", "panic", rec)
		}
	}()
	if _, err := s.RunOnce(ctx); err != nil {
		slog.Warn("[goals] workflow sweeper: run-once failed", "err", err)
	}
}

// envDuration reads a duration override, accepting either a Go duration string
// ("90s") or a bare seconds count. An unparseable value falls back to the
// default rather than disabling the sweeper.
func envDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
		return parsed
	}
	if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}
