package session

import (
	"context"
	"log"
	"sync"
	"time"

	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/policy"
)

// DOC: docs/concepts/GLOSSARY.md#core-terms

// ExpirationSweeper periodically checks sessions for expiration and removes expired ones.
type ExpirationSweeper struct {
	sessions *Manager
	events   *events.Logger
	metrics  *metrics.Metrics
	interval time.Duration
	stopCh   chan struct{}
	mu       sync.Mutex
	running  bool
}

// NewExpirationSweeper creates a sweeper that checks for expired sessions.
func NewExpirationSweeper(sm *Manager, events *events.Logger, metrics *metrics.Metrics) *ExpirationSweeper {
	return &ExpirationSweeper{
		sessions: sm,
		events:   events,
		metrics:  metrics,
		interval: 30 * time.Second,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the periodic sweep loop.
func (es *ExpirationSweeper) Start() {
	es.mu.Lock()
	if es.running {
		es.mu.Unlock()
		return
	}
	es.running = true
	es.mu.Unlock()

	go es.loop()
}

// Stop terminates the sweep loop.
func (es *ExpirationSweeper) Stop() {
	es.mu.Lock()
	defer es.mu.Unlock()
	if es.running {
		close(es.stopCh)
		es.running = false
	}
}

func (es *ExpirationSweeper) loop() {
	ticker := time.NewTicker(es.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			es.sweep()
		case <-es.stopCh:
			return
		}
	}
}

// sweep checks all sessions and removes expired ones.
func (es *ExpirationSweeper) sweep() {
	sessions := es.sessions.List()
	for _, sess := range sessions {
		p := sess.GetPolicy()
		if policy.IsExpired(sess.CreatedAt, p) {
			log.Printf("session %s: expired (policy=%s, duration=%s)", sess.ID, p.Mode, p.Duration)
			if err := es.sessions.Delete(context.Background(), sess.ID); err == nil {
				es.events.Emit(events.SessionTerminated, sess.ID, map[string]string{
					"reason":   "expired",
					"policy":   string(p.Mode),
					"duration": p.Duration,
				})
				es.metrics.SessionsDeleted.Add(1)
				es.metrics.ActiveSessions.Add(-1)
			}
		}
	}
}
