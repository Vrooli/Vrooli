package transitionrunner

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/transitionrun"
)

const (
	defaultSweepInterval = 2 * time.Minute
	defaultApplyTimeout  = 45 * time.Second
	defaultRetryDelay    = 5 * time.Second
	defaultMaxRetryDelay = 2 * time.Minute
)

// Sweeper is the composition-owned recovery loop for every transition using
// the shared correlation journal. It does not know subject domains: Runner's
// registry dispatch remains the sole path from a completed workflow to a
// subject mutation.
type Sweeper struct {
	Runner        *Runner
	Interval      time.Duration
	ApplyTimeout  time.Duration
	RetryDelay    time.Duration
	MaxRetryDelay time.Duration
}

func NewSweeper(runner *Runner) *Sweeper {
	return &Sweeper{
		Runner:        runner,
		Interval:      envDuration("SWARM_MANAGER_TRANSITION_SWEEP_INTERVAL", defaultSweepInterval),
		ApplyTimeout:  defaultApplyTimeout,
		RetryDelay:    defaultRetryDelay,
		MaxRetryDelay: defaultMaxRetryDelay,
	}
}

// RunOnce attempts each incomplete correlation independently. A workflow
// that is still running or whose engine is temporarily unavailable remains a
// retry candidate. Other failures are durably recorded by Runner and skipped
// on later passes until an operator changes the correlation or input state.
func (s *Sweeper) RunOnce(ctx context.Context) (int, error) {
	if s == nil || s.Runner == nil {
		return 0, nil
	}
	candidates, err := s.Runner.ListUnapplied()
	if err != nil {
		return 0, err
	}
	applied := 0
	for _, candidate := range candidates {
		if candidate.LastApplyError != "" && !isRetryable(candidate.LastApplyError) {
			continue
		}
		if !s.retryDue(candidate, time.Now().UTC()) {
			continue
		}
		attemptCtx, cancel := context.WithTimeout(ctx, s.applyTimeout())
		_, err := s.Runner.Apply(attemptCtx, candidate.TransitionKey, candidate.ExecutionID)
		cancel()
		if err == nil {
			applied++
			continue
		}
		if isRetryableError(err) {
			continue
		}
		slog.Warn("transition runner sweeper: apply failed", "transition", candidate.TransitionKey, "execution_id", candidate.ExecutionID, "err", err)
	}
	if applied > 0 {
		slog.Info("transition runner sweeper: applied results", "count", applied)
	}
	return applied, nil
}

// retryDue applies capped exponential spacing only after a retryable failed
// apply. A negative RetryDelay is useful in deterministic tests; zero keeps
// the production default.
func (s *Sweeper) retryDue(candidate transitionrun.Correlation, now time.Time) bool {
	if candidate.LastApplyError == "" || candidate.LastApplyAttemptTime == "" || s.RetryDelay < 0 {
		return true
	}
	lastAttempt, err := time.Parse(time.RFC3339Nano, candidate.LastApplyAttemptTime)
	if err != nil {
		return true
	}
	delay := s.RetryDelay
	if delay <= 0 {
		delay = defaultRetryDelay
	}
	maximum := s.MaxRetryDelay
	if maximum <= 0 {
		maximum = defaultMaxRetryDelay
	}
	for attempts := candidate.ApplyAttemptCount; attempts > 1 && delay < maximum; attempts-- {
		delay *= 2
		if delay > maximum {
			delay = maximum
		}
	}
	return !now.Before(lastAttempt.Add(delay))
}

// Start runs recovery periodically until ctx is cancelled.
func (s *Sweeper) Start(ctx context.Context) {
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
			s.runOnceLogged(ctx)
		}
	}
}

func (s *Sweeper) RunOnceLogged(ctx context.Context) { s.runOnceLogged(ctx) }

func (s *Sweeper) runOnceLogged(ctx context.Context) {
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error("transition runner sweeper: panic recovered", "panic", recovered)
		}
	}()
	if _, err := s.RunOnce(ctx); err != nil {
		slog.Warn("transition runner sweeper: run-once failed", "err", err)
	}
}

func (s *Sweeper) applyTimeout() time.Duration {
	if s.ApplyTimeout > 0 {
		return s.ApplyTimeout
	}
	return defaultApplyTimeout
}

func isRetryableError(err error) bool {
	return errors.Is(err, agentmanager.ErrWorkflowNotReady) ||
		errors.Is(err, agentmanager.ErrNotAvailable) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled)
}

// isRetryable is intentionally conservative because persisted errors are
// strings. It is used only to decide whether a previously-recorded error is
// eligible for another automatic attempt; fresh errors are classified with
// errors.Is above.
func isRetryable(message string) bool {
	return strings.Contains(message, agentmanager.ErrWorkflowNotReady.Error()) ||
		strings.Contains(message, agentmanager.ErrNotAvailable.Error()) ||
		strings.Contains(message, context.DeadlineExceeded.Error()) ||
		strings.Contains(message, context.Canceled.Error())
}

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
