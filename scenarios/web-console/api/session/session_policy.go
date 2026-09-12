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

// ExpirationSweeper periodically checks sessions for expiration and requests
// archival through the canonical lifecycle owner. The active process is
// removed from the manager, but durable evidence is not deleted by expiry.
type ExpirationSweeper struct {
	sessions *Manager
	events   *events.Logger
	metrics  *metrics.Metrics
	// archive is installed by the API composition root. Expiration must not
	// silently fall back to the process manager, because the manager cannot
	// persist lifecycle receipts or coordinate durable evidence.
	archive  func(context.Context, string) error
	interval time.Duration
	stopCh   chan struct{}
	mu       sync.Mutex
	running  bool
}

// SetArchiveHandler wires expiry to the owning lifecycle service. It must be
// configured before Start is called.
func (es *ExpirationSweeper) SetArchiveHandler(handler func(context.Context, string) error) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.archive = handler
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

// sweep checks all sessions and requests archival for expired ones.
func (es *ExpirationSweeper) sweep() {
	sessions := es.sessions.List()
	for _, sess := range sessions {
		p := sess.GetPolicy()
		if policy.IsExpired(sess.CreatedAt, p) {
			log.Printf("session %s: expired (policy=%s, duration=%s)", sess.ID, p.Mode, p.Duration)
			es.mu.Lock()
			archive := es.archive
			es.mu.Unlock()
			if archive == nil {
				log.Printf("session %s: expiration skipped; continuity archive handler is not configured", sess.ID)
				continue
			}
			if err := archive(context.Background(), sess.ID); err == nil {
				es.events.Emit(events.SessionTerminated, sess.ID, map[string]string{
					"reason":   "expired",
					"policy":   string(p.Mode),
					"duration": p.Duration,
				})
				// The canonical archive owner accounts for the active-session
				// transition. Expiration is only a policy trigger; it must not
				// double-count the transition or report archive as deletion.
			} else {
				// Expiration is a governed lifecycle request. Preserve the session
				// and surface a durable-domain failure instead of silently retrying
				// or reporting a deletion that never happened.
				log.Printf("session %s: expiration archive failed: %v", sess.ID, err)
				es.metrics.ContinuityFailures.Add(1)
			}
		}
	}
}
