package supervisor

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// HealthChecker is a function that checks if the sidecar is healthy.
// Returns nil if healthy, error otherwise.
type HealthChecker func(ctx context.Context) error

// ProcessSupervisor manages the lifecycle of a sidecar process.
// It handles automatic restarts with exponential backoff.
type ProcessSupervisor struct {
	config      Config
	process     Process
	healthCheck HealthChecker
	log         *logrus.Logger

	state        State
	restartTimes []time.Time // Timestamps of recent restarts (within RestartWindow)

	subscribers map[chan StateEvent]struct{}
	subsMu      sync.RWMutex

	stopCh  chan struct{} // Signals the monitor goroutine to stop
	stopped bool          // True if Stop() has been called
	mu      sync.RWMutex
}

// NewProcessSupervisor creates a new ProcessSupervisor.
//
// Parameters:
//   - config: configuration for restart behavior
//   - process: the process to manage
//   - healthCheck: function to verify sidecar is healthy after start
//   - log: logger for supervisor events
func NewProcessSupervisor(config Config, process Process, healthCheck HealthChecker, log *logrus.Logger) *ProcessSupervisor {
	return &ProcessSupervisor{
		config:       config,
		process:      process,
		healthCheck:  healthCheck,
		log:          log,
		state:        StateStopped,
		restartTimes: make([]time.Time, 0, config.MaxRestarts+1),
		subscribers:  make(map[chan StateEvent]struct{}),
	}
}

// Start begins the sidecar process and waits for it to become healthy.
func (s *ProcessSupervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.state == StateRunning || s.state == StateStarting {
		s.mu.Unlock()
		return fmt.Errorf("supervisor already running")
	}
	if s.stopped {
		s.mu.Unlock()
		return fmt.Errorf("supervisor has been stopped")
	}

	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	// setState acquires its own lock, so we must release ours first
	s.setState(StateStarting, nil)

	// Start the process
	if err := s.process.Start(); err != nil {
		s.setState(StateStopped, err)
		return fmt.Errorf("failed to start process: %w", err)
	}

	// Wait for health check to pass
	if err := s.waitForHealthy(ctx); err != nil {
		// Stop the unhealthy process
		_ = s.process.Stop(s.config.GracefulStop)
		s.setState(StateStopped, err)
		return fmt.Errorf("process failed health check: %w", err)
	}

	s.setState(StateRunning, nil)

	// Start the monitor goroutine
	go s.monitorLoop()

	return nil
}

// waitForHealthy polls the health check until it passes or timeout.
func (s *ProcessSupervisor) waitForHealthy(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.config.StartupTimeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("startup timeout: %w", ctx.Err())
		case <-ticker.C:
			if err := s.healthCheck(ctx); err == nil {
				return nil
			}
		}
	}
}

// monitorLoop watches for process exit and triggers restarts.
func (s *ProcessSupervisor) monitorLoop() {
	for {
		select {
		case <-s.stopCh:
			return
		case <-s.process.ExitChan():
			s.handleProcessExit()
		}
	}
}

// handleProcessExit is called when the process exits unexpectedly.
func (s *ProcessSupervisor) handleProcessExit() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}

	// Check if we've exceeded max restarts
	now := time.Now()
	s.pruneRestartTimes(now)
	s.restartTimes = append(s.restartTimes, now)

	exceededMaxRestarts := len(s.restartTimes) > s.config.MaxRestarts
	restartCount := len(s.restartTimes)
	s.mu.Unlock()

	// setState acquires its own lock, so we must release ours first
	if exceededMaxRestarts {
		s.log.WithFields(logrus.Fields{
			"restart_count": restartCount,
			"max_restarts":  s.config.MaxRestarts,
			"window":        s.config.RestartWindow,
		}).Error("Max restarts exceeded, entering unrecoverable state")
		s.setState(StateUnrecoverable, fmt.Errorf("max restarts (%d) exceeded within %v", s.config.MaxRestarts, s.config.RestartWindow))
		return
	}

	s.setState(StateRestarting, nil)

	// Calculate backoff delay
	delay := s.calculateBackoff(restartCount)
	s.log.WithFields(logrus.Fields{
		"restart_count": restartCount,
		"backoff_delay": delay,
	}).Info("Scheduling restart after backoff")

	// Wait for backoff
	select {
	case <-s.stopCh:
		return
	case <-time.After(delay):
	}

	// Attempt restart
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	s.log.Info("Attempting to restart playwright-driver")

	ctx, cancel := context.WithTimeout(context.Background(), s.config.StartupTimeout)
	defer cancel()

	if err := s.process.Start(); err != nil {
		s.log.WithError(err).Error("Failed to restart process")
		// The monitor loop will pick up the exit and try again
		return
	}

	if err := s.waitForHealthy(ctx); err != nil {
		s.log.WithError(err).Error("Restarted process failed health check")
		_ = s.process.Stop(s.config.GracefulStop)
		return
	}

	s.setState(StateRunning, nil)
	s.log.Info("Successfully restarted playwright-driver")
}

// pruneRestartTimes removes restart times outside the restart window.
func (s *ProcessSupervisor) pruneRestartTimes(now time.Time) {
	cutoff := now.Add(-s.config.RestartWindow)
	valid := make([]time.Time, 0, len(s.restartTimes))
	for _, t := range s.restartTimes {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	s.restartTimes = valid
}

// calculateBackoff calculates the backoff delay for a restart.
func (s *ProcessSupervisor) calculateBackoff(restartCount int) time.Duration {
	// delay = initial * (multiplier ^ (restartCount - 1))
	exponent := float64(restartCount - 1)
	if exponent < 0 {
		exponent = 0
	}
	multiplier := math.Pow(s.config.BackoffMultiplier, exponent)
	delay := time.Duration(float64(s.config.InitialBackoff) * multiplier)

	// Clamp to max
	if delay > s.config.MaxBackoff {
		delay = s.config.MaxBackoff
	}

	return delay
}

// Stop gracefully stops the sidecar process.
func (s *ProcessSupervisor) Stop(ctx context.Context) error {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return nil
	}
	s.stopped = true

	if s.stopCh != nil {
		close(s.stopCh)
	}
	s.mu.Unlock()

	// setState acquires its own lock, so we must release ours first
	s.setState(StateStopping, nil)

	// Stop the process
	if err := s.process.Stop(s.config.GracefulStop); err != nil {
		s.log.WithError(err).Warn("Error stopping process")
	}

	s.setState(StateStopped, nil)

	// Close all subscriber channels
	s.subsMu.Lock()
	for ch := range s.subscribers {
		close(ch)
		delete(s.subscribers, ch)
	}
	s.subsMu.Unlock()

	return nil
}

// Restart stops and then starts the sidecar process.
func (s *ProcessSupervisor) Restart(ctx context.Context) error {
	s.mu.Lock()
	if s.state == StateUnrecoverable {
		// Reset the restart counter for manual restarts
		s.restartTimes = nil
	}
	s.mu.Unlock()

	// Stop the current process
	s.setState(StateRestarting, nil)
	if err := s.process.Stop(s.config.GracefulStop); err != nil {
		s.log.WithError(err).Warn("Error stopping process during restart")
	}

	// Start fresh
	if err := s.process.Start(); err != nil {
		s.setState(StateStopped, err)
		return fmt.Errorf("failed to start process: %w", err)
	}

	// Wait for health
	if err := s.waitForHealthy(ctx); err != nil {
		_ = s.process.Stop(s.config.GracefulStop)
		s.setState(StateStopped, err)
		return fmt.Errorf("process failed health check: %w", err)
	}

	s.setState(StateRunning, nil)
	return nil
}

// State returns the current supervisor state.
func (s *ProcessSupervisor) State() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// RestartCount returns the number of restarts within the current window.
func (s *ProcessSupervisor) RestartCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	cutoff := now.Add(-s.config.RestartWindow)
	count := 0
	for _, t := range s.restartTimes {
		if t.After(cutoff) {
			count++
		}
	}
	return count
}

// Subscribe returns a channel that receives state change events.
func (s *ProcessSupervisor) Subscribe() <-chan StateEvent {
	ch := make(chan StateEvent, 16)

	s.subsMu.Lock()
	s.subscribers[ch] = struct{}{}
	s.subsMu.Unlock()

	return ch
}

// Unsubscribe removes a subscriber channel.
func (s *ProcessSupervisor) Unsubscribe(ch <-chan StateEvent) {
	s.subsMu.Lock()
	defer s.subsMu.Unlock()

	// Find the actual send channel (we receive a receive-only channel)
	for sendCh := range s.subscribers {
		if sendCh == ch {
			close(sendCh)
			delete(s.subscribers, sendCh)
			return
		}
	}
}

// setState updates the state and notifies subscribers.
// Must be called without holding s.mu to avoid deadlock.
func (s *ProcessSupervisor) setState(newState State, err error) {
	s.mu.Lock()
	prev := s.state
	s.state = newState
	restartCount := len(s.restartTimes)
	s.mu.Unlock()

	if prev == newState {
		return
	}

	s.log.WithFields(logrus.Fields{
		"previous": prev,
		"current":  newState,
	}).Info("Supervisor state changed")

	event := StateEvent{
		Previous:     prev,
		Current:      newState,
		RestartCount: restartCount,
		Error:        err,
	}

	s.subsMu.RLock()
	defer s.subsMu.RUnlock()

	for ch := range s.subscribers {
		select {
		case ch <- event:
		default:
			// Don't block if subscriber is slow
		}
	}
}
