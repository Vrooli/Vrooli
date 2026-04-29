package sandbox

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/types"
)

// HealConfig controls automatic remounting of sandboxes with stale mounts.
type HealConfig struct {
	// IdleGracePeriod is the minimum time since LastUsedAt before
	// a sandbox is eligible for auto-remount. Default: 30s.
	IdleGracePeriod time.Duration

	// MaxConsecutiveFailures is the cap on remount attempts before
	// marking a sandbox as Error. Default: 5.
	MaxConsecutiveFailures int

	// BaseBackoff is the initial backoff duration after a failed
	// remount attempt. Doubled on each failure, capped at 1h. Default: 30s.
	BaseBackoff time.Duration
}

// DefaultHealConfig returns a HealConfig with sensible defaults.
func DefaultHealConfig() HealConfig {
	return HealConfig{
		IdleGracePeriod:        30 * time.Second,
		MaxConsecutiveFailures: 5,
		BaseBackoff:            30 * time.Second,
	}
}

const maxBackoff = 1 * time.Hour

// healState tracks per-sandbox remount failure history.
type healState struct {
	consecutiveFailures int
	lastAttempt         time.Time
	lastError           string
}

// healTracker maintains in-memory failure state for auto-heal.
// Resets on API restart, which is fine since restart triggers fresh reconciliation.
type healTracker struct {
	mu       sync.Mutex
	failures map[uuid.UUID]*healState
}

func newHealTracker() *healTracker {
	return &healTracker{
		failures: make(map[uuid.UUID]*healState),
	}
}

func (t *healTracker) get(id uuid.UUID) *healState {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.failures[id]
}

func (t *healTracker) recordFailure(id uuid.UUID, now time.Time, errMsg string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	s, ok := t.failures[id]
	if !ok {
		s = &healState{}
		t.failures[id] = s
	}
	s.consecutiveFailures++
	s.lastAttempt = now
	s.lastError = errMsg
	return s.consecutiveFailures
}

func (t *healTracker) reset(id uuid.UUID) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.failures, id)
}

// ReconcileActiveMounts checks all Active sandboxes for stale mounts
// and attempts automatic Stop->Start healing for eligible ones.
func (s *Service) ReconcileActiveMounts(ctx context.Context, tracker *healTracker, cfg HealConfig) {
	if s == nil || s.repo == nil || s.driver == nil {
		return
	}

	result, err := s.repo.List(ctx, &types.ListFilter{
		Status: []types.Status{types.StatusActive},
		Limit:  10000,
	})
	if err != nil || result == nil {
		return
	}

	now := time.Now()
	for _, sandbox := range result.Sandboxes {
		if sandbox == nil || sandbox.Status != types.StatusActive {
			continue
		}

		// Check mount health. Drivers without a real mount (CopyDriver)
		// don't implement MountVerifier; VerifyIfSupported returns nil for
		// them, so the heal loop never tries to auto-heal them.
		if err := driver.VerifyIfSupported(ctx, s.driver, sandbox); err == nil {
			// Mount is healthy; clear any prior failure state.
			tracker.reset(sandbox.ID)
			continue
		}

		state := tracker.get(sandbox.ID)
		eligible, reason := isEligibleForHeal(sandbox, state, cfg, now)
		if !eligible {
			s.logAuditEvent(ctx, sandbox, "sandbox.auto-heal-skipped", "system", "system", map[string]interface{}{
				"reason": reason,
			})
			continue
		}

		// Attempt heal
		if err := s.healSandbox(ctx, sandbox.ID); err != nil {
			count := tracker.recordFailure(sandbox.ID, now, err.Error())
			s.logAuditEvent(ctx, sandbox, "sandbox.auto-heal-failed", "system", "system", map[string]interface{}{
				"error":              err.Error(),
				"consecutiveFailure": count,
			})

			// If max failures exceeded, transition to Error status.
			if count >= cfg.MaxConsecutiveFailures {
				s.markAutoHealExhausted(ctx, sandbox, count)
				tracker.reset(sandbox.ID)
			}
			continue
		}

		// Success
		tracker.reset(sandbox.ID)
		s.logAuditEvent(ctx, sandbox, "sandbox.auto-healed", "system", "system", nil)
	}
}

// healSandbox performs a Stop->Start cycle to remount a stale sandbox.
// Leverages idempotency guarantees of Stop/Start.
func (s *Service) healSandbox(ctx context.Context, id uuid.UUID) error {
	_, err := s.Stop(ctx, id)
	if err != nil {
		// If sandbox was deleted between listing and healing, skip it.
		var notFound *types.NotFoundError
		if errors.As(err, &notFound) {
			return nil
		}
		return fmt.Errorf("heal stop: %w", err)
	}

	_, err = s.Start(ctx, id)
	if err != nil {
		var notFound *types.NotFoundError
		if errors.As(err, &notFound) {
			return nil
		}
		return fmt.Errorf("heal start: %w", err)
	}

	return nil
}

// markAutoHealExhausted transitions a sandbox to Error after max failures.
func (s *Service) markAutoHealExhausted(ctx context.Context, sandbox *types.Sandbox, failures int) {
	// Re-fetch to get latest state (avoid stale version conflicts).
	current, err := s.Get(ctx, sandbox.ID)
	if err != nil {
		return
	}
	// Only escalate if still Active or Stopped (a failed heal leaves it Stopped).
	if current.Status != types.StatusActive && current.Status != types.StatusStopped {
		return
	}

	current.Status = types.StatusError
	current.ErrorMsg = fmt.Sprintf("auto-heal failed after %d consecutive attempts: %s",
		failures, "mount could not be restored automatically")

	if err := s.repo.Update(ctx, current); err != nil {
		s.logAuditEvent(ctx, sandbox, "sandbox.warning", "system", "system", map[string]interface{}{
			"message": "failed to mark sandbox as error after exhausted heal attempts: " + err.Error(),
		})
		return
	}

	s.logAuditEvent(ctx, current, "sandbox.auto-heal-exhausted", "system", "system", map[string]interface{}{
		"consecutiveFailures": failures,
	})
}

// isEligibleForHeal checks whether a sandbox is safe and due for auto-heal.
func isEligibleForHeal(sandbox *types.Sandbox, state *healState, cfg HealConfig, now time.Time) (bool, string) {
	// Safety guard: skip recently-used sandboxes.
	if cfg.IdleGracePeriod > 0 && now.Sub(sandbox.LastUsedAt) < cfg.IdleGracePeriod {
		return false, "recently used"
	}

	if state == nil {
		return true, ""
	}

	// Max failures already exceeded (should have been escalated).
	if state.consecutiveFailures >= cfg.MaxConsecutiveFailures {
		return false, "max failures exceeded"
	}

	// Respect exponential backoff.
	required := backoffDuration(state.consecutiveFailures, cfg.BaseBackoff)
	if now.Sub(state.lastAttempt) < required {
		return false, "backing off"
	}

	return true, ""
}

// backoffDuration returns base * 2^(failures-1), capped at 1 hour.
// Returns 0 for failures <= 0.
func backoffDuration(failures int, base time.Duration) time.Duration {
	if failures <= 0 {
		return 0
	}
	d := base
	for i := 1; i < failures; i++ {
		d *= 2
		if d > maxBackoff {
			return maxBackoff
		}
	}
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}
