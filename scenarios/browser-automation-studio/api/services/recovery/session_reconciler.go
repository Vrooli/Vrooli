package recovery

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/database"
)

// DefaultSessionReconcileInterval limits recovery load while ensuring a
// terminal execution cannot hold driver capacity until its idle TTL expires.
const DefaultSessionReconcileInterval = time.Minute

// DefaultSessionTerminalGrace gives normal close/release cleanup a short
// opportunity to finish before recovery force-closes the orphan.
const DefaultSessionTerminalGrace = 15 * time.Second

type sessionInventory interface {
	ListObservedSessions(context.Context) ([]driver.ObservedSession, error)
	ForceCloseSession(context.Context, string) error
}

type executionLookup interface {
	GetExecution(context.Context, uuid.UUID) (*database.ExecutionIndex, error)
}

// SessionReconciler force-closes only sessions whose owner has reached a
// terminal state and has remained terminal past the grace period.
type SessionReconciler struct {
	driver   sessionInventory
	repo     executionLookup
	log      *logrus.Logger
	grace    time.Duration
	interval time.Duration
	now      func() time.Time
}

type SessionReconcileResult struct {
	Observed int
	Closed   int
	Skipped  int
}

type SessionReconcilerOption func(*SessionReconciler)

func WithSessionTerminalGrace(grace time.Duration) SessionReconcilerOption {
	return func(r *SessionReconciler) {
		if grace >= 0 {
			r.grace = grace
		}
	}
}

func WithSessionReconcileInterval(interval time.Duration) SessionReconcilerOption {
	return func(r *SessionReconciler) {
		if interval > 0 {
			r.interval = interval
		}
	}
}

func newSessionReconciler(driverClient sessionInventory, repo executionLookup, log *logrus.Logger, opts ...SessionReconcilerOption) *SessionReconciler {
	r := &SessionReconciler{driver: driverClient, repo: repo, log: log, grace: DefaultSessionTerminalGrace, interval: DefaultSessionReconcileInterval, now: time.Now}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// NewSessionReconciler builds the recovery worker used by the API lifecycle.
func NewSessionReconciler(driverClient *driver.Client, repo database.Repository, log *logrus.Logger, opts ...SessionReconcilerOption) *SessionReconciler {
	return newSessionReconciler(driverClient, repo, log, opts...)
}

// ReconcileOnce performs one bounded, idempotent recovery pass.
func (r *SessionReconciler) ReconcileOnce(ctx context.Context) (SessionReconcileResult, error) {
	sessions, err := r.driver.ListObservedSessions(ctx)
	if err != nil {
		return SessionReconcileResult{}, fmt.Errorf("list driver sessions: %w", err)
	}
	result := SessionReconcileResult{Observed: len(sessions)}
	for _, session := range sessions {
		ownerID, err := uuid.Parse(session.OwnerExecutionID)
		if err != nil {
			result.Skipped++
			continue
		}
		execution, err := r.repo.GetExecution(ctx, ownerID)
		if err != nil || execution == nil || !database.IsTerminalStatus(execution.Status) {
			result.Skipped++
			continue
		}
		terminalAt := execution.UpdatedAt
		if execution.CompletedAt != nil {
			terminalAt = *execution.CompletedAt
		}
		if r.now().Before(terminalAt.Add(r.grace)) {
			result.Skipped++
			continue
		}
		if err := r.driver.ForceCloseSession(ctx, session.ID); err != nil {
			if r.log != nil {
				r.log.WithError(err).WithField("session_id", session.ID).Warn("terminal session reconciliation failed")
			}
			continue
		}
		result.Closed++
	}
	return result, nil
}

// Run owns the periodic loop; callers cancel its context during API shutdown.
func (r *SessionReconciler) Run(ctx context.Context) {
	if _, err := r.ReconcileOnce(ctx); err != nil && r.log != nil {
		r.log.WithError(err).Warn("initial terminal session reconciliation failed")
	}
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := r.ReconcileOnce(ctx); err != nil && r.log != nil {
				r.log.WithError(err).Warn("terminal session reconciliation failed")
			}
		}
	}
}
