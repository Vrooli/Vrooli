package agent_manager

import (
	"context"
	"log"
	"sync"
	"time"
)

const (
	profileReconcileRetryInterval = time.Minute
	profileReconcileTimeout       = 10 * time.Second
)

// ProfileStatus is the startup-owned readiness projection for the profile
// required by skill validation. Reconciliation failures do not take down the
// API, but they remain visible until a later retry succeeds.
type ProfileStatus struct {
	mu      sync.RWMutex
	ready   bool
	message string
}

func (s *ProfileStatus) Set(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil {
		s.ready = true
		s.message = ""
		return
	}
	s.ready = false
	s.message = err.Error()
}

// ReadyStatus reports whether the DTV profile is available to skill runs.
func (s *ProfileStatus) ReadyStatus() (bool, string) {
	if s == nil {
		return true, ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.ready {
		return true, ""
	}
	if s.message == "" {
		return false, "agent-manager profile reconciliation has not completed"
	}
	return false, "agent-manager profile reconciliation failed: " + s.message
}

// RetryProfileInitialization waits between bounded attempts and stops as soon
// as the owner confirms the required profile or its lifecycle context ends.
func RetryProfileInitialization(ctx context.Context, initialize func(context.Context) error, status *ProfileStatus, logger *log.Logger) {
	retryProfileInitialization(ctx, initialize, status, logger, profileReconcileRetryInterval, profileReconcileTimeout)
}

func retryProfileInitialization(ctx context.Context, initialize func(context.Context) error, status *ProfileStatus, logger *log.Logger, interval, timeout time.Duration) {
	if initialize == nil || status == nil {
		return
	}
	if logger == nil {
		logger = log.Default()
	}
	for {
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		err := initialize(attemptCtx)
		cancel()
		status.Set(err)
		if err == nil {
			logger.Printf("agent-manager profile reconciliation recovered")
			return
		}
		logger.Printf("agent-manager profile reconciliation retry failed: %v", err)
	}
}
