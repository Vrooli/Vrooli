package feedback

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// DefaultStuckMaxAge is the wall-clock age past which an agent_thinking
// round is auto-dismissed by the sweeper. Chosen well above any healthy
// agent run time so a slow run isn't interrupted, but short enough that a
// crashed run doesn't strand the user for hours.
const DefaultStuckMaxAge = 30 * time.Minute

// DefaultSweepInterval is how often the sweeper ticker runs once started.
const DefaultSweepInterval = 5 * time.Minute

// InitiativeLister is the minimal surface the sweeper needs to enumerate
// the initiatives whose feedback rounds it should walk. Mirrors the shape
// of initiatives.Store.LoadAll without importing the package directly so
// the sweeper stays unit-testable in isolation.
type InitiativeLister interface {
	ListNames() ([]string, error)
}

// Sweeper periodically scans every initiative's feedback rounds for ones
// stuck in agent_thinking and force-cancels any whose run is unreachable
// or whose updated_at is older than MaxAge. It is the safety net for the
// failure mode where the agent crashes without producing output and
// without anyone polling the round long enough to trip the failure
// counter in EnsurePolledTurn.
type Sweeper struct {
	Service     *Service
	Initiatives InitiativeLister
	MaxAge      time.Duration
	Interval    time.Duration
	Clock       func() time.Time
}

// NewSweeper constructs a Sweeper with sane defaults applied. Reads
// SWARM_MANAGER_FEEDBACK_STUCK_MAX_AGE and
// SWARM_MANAGER_FEEDBACK_SWEEP_INTERVAL so operators can tune without
// recompiling.
func NewSweeper(svc *Service, initiatives InitiativeLister) *Sweeper {
	return &Sweeper{
		Service:     svc,
		Initiatives: initiatives,
		MaxAge:      envDuration("SWARM_MANAGER_FEEDBACK_STUCK_MAX_AGE", DefaultStuckMaxAge),
		Interval:    envDuration("SWARM_MANAGER_FEEDBACK_SWEEP_INTERVAL", DefaultSweepInterval),
		Clock:       time.Now,
	}
}

// envDuration reads a time.Duration env var (e.g. "30m", "5m") and falls
// back to the default on parse error.
func envDuration(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		// Allow a bare integer to mean seconds for ergonomics.
		if n, err2 := strconv.Atoi(raw); err2 == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
		return fallback
	}
	return d
}

// RunOnce performs a single sweep pass. Safe to call repeatedly; safe to
// call from startup (synchronous) and from the ticker goroutine. Logs and
// continues on per-initiative errors so one broken initiative doesn't
// block recovery for the rest.
//
// Returns the number of rounds dismissed for observability. Errors only
// when the initiative listing itself fails — per-round errors are logged.
func (s *Sweeper) RunOnce(ctx context.Context) (int, error) {
	if s == nil || s.Service == nil || s.Initiatives == nil {
		return 0, nil
	}
	names, err := s.Initiatives.ListNames()
	if err != nil {
		return 0, fmt.Errorf("list initiatives: %w", err)
	}
	now := s.clock()
	maxAge := s.maxAge()
	dismissed := 0
	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			continue
		}
		rounds, err := s.Service.store.ListRounds(name)
		if err != nil {
			slog.Warn("feedback: sweeper: list rounds failed",
				"err", err, "initiative", name)
			continue
		}
		for _, r := range rounds {
			if r.Status != RoundStatusAgentThinking {
				continue
			}
			rationale := s.evaluate(name, r, now, maxAge)
			if rationale == "" {
				continue
			}
			if _, cErr := s.Service.Cancel(ctx, CancelRequest{
				InitiativeName: name,
				RoundNumber:    r.Number,
				Rationale:      rationale,
				DecidedBy:      "swarm-manager-sweeper",
			}); cErr != nil {
				slog.Warn("feedback: sweeper: cancel failed",
					"err", cErr, "initiative", name, "round", r.Number)
				continue
			}
			slog.Info("feedback: sweeper: dismissed stuck round",
				"initiative", name, "round", r.Number, "rationale", rationale)
			dismissed++
		}
	}
	if dismissed > 0 {
		slog.Info("feedback: sweeper: pass complete", "dismissed", dismissed)
	}
	return dismissed, nil
}

// evaluate decides whether the round should be force-dismissed and
// returns a rationale string. Returns "" when the round is healthy.
//
// Reasons to dismiss:
//  1. updated_at is older than MaxAge (the agent is wedged).
//  2. The initiative lock is gone (the holder lost its grip but the round
//     never advanced — typical when the API restarted while the run was
//     mid-flight; the lock sweep cleared the file but not the round).
func (s *Sweeper) evaluate(initiativeName string, r Round, now time.Time, maxAge time.Duration) string {
	updated, err := time.Parse(time.RFC3339, r.UpdatedAt)
	if err == nil && now.Sub(updated) > maxAge {
		return fmt.Sprintf("auto-dismissed: agent run timed out (round in agent_thinking >%s)", maxAge)
	}
	if s.Service.lock != nil {
		holder, _ := s.Service.lock.Inspect(initiativeName)
		if holder == nil {
			return "auto-dismissed: initiative lock no longer present (agent run gone)"
		}
	}
	return ""
}

// Start runs RunOnce on a ticker until ctx is cancelled. Should be called
// in its own goroutine; recovers from panics inside RunOnce so a transient
// disk error or store inconsistency doesn't kill the safety net.
func (s *Sweeper) Start(ctx context.Context) {
	if s == nil || s.Interval <= 0 {
		return
	}
	t := time.NewTicker(s.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.runWithRecover(ctx)
		}
	}
}

func (s *Sweeper) runWithRecover(ctx context.Context) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("feedback: sweeper: panic recovered", "panic", rec)
		}
	}()
	if _, err := s.RunOnce(ctx); err != nil {
		slog.Warn("feedback: sweeper: run-once failed", "err", err)
	}
}

func (s *Sweeper) clock() time.Time {
	if s.Clock != nil {
		return s.Clock().UTC()
	}
	return time.Now().UTC()
}

func (s *Sweeper) maxAge() time.Duration {
	if s.MaxAge > 0 {
		return s.MaxAge
	}
	return DefaultStuckMaxAge
}
