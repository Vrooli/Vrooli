package health

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/sidecar/supervisor"
)

// CircuitBreakerStateFunc returns the current circuit breaker state.
type CircuitBreakerStateFunc func() string

// HealthMonitor monitors the health of the playwright-driver sidecar.
// It polls the sidecar's /health endpoint and broadcasts status changes.
type HealthMonitor struct {
	config     Config
	checker    Checker
	supervisor supervisor.Supervisor
	cbState    CircuitBreakerStateFunc
	log        *logrus.Logger

	current          DriverHealth
	consecutiveFails int
	lastBroadcast    time.Time

	subscribers map[chan DriverHealth]struct{}
	subsMu      sync.RWMutex

	stopCh  chan struct{}
	stopped bool
	mu      sync.RWMutex
}

// NewHealthMonitor creates a new HealthMonitor.
//
// Parameters:
//   - config: health monitoring configuration
//   - checker: interface to check sidecar health
//   - sup: supervisor for restart count and state info
//   - cbState: function returning circuit breaker state
//   - log: logger for health events
func NewHealthMonitor(
	config Config,
	checker Checker,
	sup supervisor.Supervisor,
	cbState CircuitBreakerStateFunc,
	log *logrus.Logger,
) *HealthMonitor {
	return &HealthMonitor{
		config:      config,
		checker:     checker,
		supervisor:  sup,
		cbState:     cbState,
		log:         log,
		subscribers: make(map[chan DriverHealth]struct{}),
		current: DriverHealth{
			Status:    StatusUnhealthy,
			UpdatedAt: time.Now(),
		},
	}
}

// Start begins health monitoring.
func (m *HealthMonitor) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return nil
	}
	m.stopCh = make(chan struct{})
	m.mu.Unlock()

	// Subscribe to supervisor state changes
	supEvents := m.supervisor.Subscribe()

	go m.monitorLoop(supEvents)

	m.log.Info("Health monitor started")
	return nil
}

// Stop stops health monitoring.
func (m *HealthMonitor) Stop() error {
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return nil
	}
	m.stopped = true
	close(m.stopCh)
	m.mu.Unlock()

	// Close all subscriber channels
	m.subsMu.Lock()
	for ch := range m.subscribers {
		close(ch)
		delete(m.subscribers, ch)
	}
	m.subsMu.Unlock()

	m.log.Info("Health monitor stopped")
	return nil
}

// monitorLoop is the main health monitoring loop.
func (m *HealthMonitor) monitorLoop(supEvents <-chan supervisor.StateEvent) {
	ticker := time.NewTicker(m.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return

		case evt, ok := <-supEvents:
			if !ok {
				return
			}
			m.handleSupervisorEvent(evt)

		case <-ticker.C:
			m.checkHealth()
		}
	}
}

// handleSupervisorEvent processes supervisor state changes.
func (m *HealthMonitor) handleSupervisorEvent(evt supervisor.StateEvent) {
	cbState := m.getCBState()

	var newHealth DriverHealth
	switch evt.Current {
	case supervisor.StateRunning:
		// Will be updated by next health check
		return
	case supervisor.StateRestarting:
		newHealth = NewRestartingState(evt.RestartCount, cbState)
	case supervisor.StateUnrecoverable:
		newHealth = NewUnrecoverableState(evt.RestartCount, cbState)
	case supervisor.StateStopped:
		errStr := "sidecar stopped"
		newHealth = DriverHealth{
			Status:         StatusUnhealthy,
			CircuitBreaker: cbState,
			RestartCount:   evt.RestartCount,
			LastError:      &errStr,
			UpdatedAt:      time.Now(),
		}
	default:
		return
	}

	m.updateHealth(newHealth)
}

// checkHealth polls the sidecar's /health endpoint.
func (m *HealthMonitor) checkHealth() {
	m.mu.RLock()
	if m.stopped {
		m.mu.RUnlock()
		return
	}
	m.mu.RUnlock()

	// Skip if supervisor is not running
	supState := m.supervisor.State()
	if supState != supervisor.StateRunning {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.config.Timeout)
	defer cancel()

	resp, err := m.checker.Check(ctx)
	cbState := m.getCBState()
	restartCount := m.supervisor.RestartCount()

	if err != nil {
		m.mu.Lock()
		m.consecutiveFails++
		fails := m.consecutiveFails
		m.mu.Unlock()

		if fails >= m.config.FailureThreshold {
			// Estimate recovery time based on backoff
			var estimatedMS *int64
			if cbState == "open" {
				// Circuit breaker will try again in ~30s (default timeout)
				ms := int64(30000)
				estimatedMS = &ms
			}

			newHealth := NewUnhealthyState(err, restartCount, cbState, estimatedMS)
			m.updateHealth(newHealth)
		}
		return
	}

	// Reset failure counter on success
	m.mu.Lock()
	m.consecutiveFails = 0
	m.mu.Unlock()

	newHealth := NewHealthyState(resp, restartCount, cbState)
	m.updateHealth(newHealth)
}

// updateHealth updates the current health state and broadcasts if changed.
func (m *HealthMonitor) updateHealth(newHealth DriverHealth) {
	m.mu.Lock()
	prev := m.current
	changed := prev.Status != newHealth.Status

	// Debounce rapid changes
	if changed && time.Since(m.lastBroadcast) < m.config.Debounce {
		m.mu.Unlock()
		return
	}

	m.current = newHealth
	if changed {
		m.lastBroadcast = time.Now()
	}
	m.mu.Unlock()

	if changed {
		m.log.WithFields(logrus.Fields{
			"previous": prev.Status,
			"current":  newHealth.Status,
		}).Info("Sidecar health status changed")

		m.broadcast(newHealth)
	}
}

// broadcast sends the health state to all subscribers.
func (m *HealthMonitor) broadcast(health DriverHealth) {
	m.subsMu.RLock()
	defer m.subsMu.RUnlock()

	for ch := range m.subscribers {
		select {
		case ch <- health:
		default:
			// Don't block if subscriber is slow
		}
	}
}

// CurrentHealth returns the current health state.
func (m *HealthMonitor) CurrentHealth() DriverHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// Subscribe returns a channel that receives health state updates.
func (m *HealthMonitor) Subscribe() <-chan DriverHealth {
	ch := make(chan DriverHealth, 16)

	m.subsMu.Lock()
	m.subscribers[ch] = struct{}{}
	m.subsMu.Unlock()

	return ch
}

// Unsubscribe removes a subscriber channel.
func (m *HealthMonitor) Unsubscribe(ch <-chan DriverHealth) {
	m.subsMu.Lock()
	defer m.subsMu.Unlock()

	for sendCh := range m.subscribers {
		if sendCh == ch {
			close(sendCh)
			delete(m.subscribers, sendCh)
			return
		}
	}
}

// getCBState safely gets the circuit breaker state.
func (m *HealthMonitor) getCBState() string {
	if m.cbState == nil {
		return "unknown"
	}
	return m.cbState()
}
