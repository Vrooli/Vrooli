package session

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/policy"
)

// DOC: docs/concepts/GLOSSARY.md#core-terms

// ExpirationSweeper periodically checks sessions against their policy and
// requests archival through the canonical lifecycle owner: age-based modes
// (preset, custom) when the session outlives its TTL, and the idle mode when
// the installed IdleJudge finds the agent idle with nothing running. The
// active process is removed from the manager, but durable evidence is not
// deleted.
type ExpirationSweeper struct {
	sessions *Manager
	events   *events.Logger
	metrics  *metrics.Metrics
	// archive is installed by the API composition root. Expiration must not
	// silently fall back to the process manager, because the manager cannot
	// persist lifecycle receipts or coordinate durable evidence. note is the
	// archive note shown in the archive list.
	archive func(ctx context.Context, id, note string) error
	// idle decides the idle mode; nil keeps every idle-policy session.
	idle IdleJudge
	// idleChecked bounds how often one session's harness evidence is read.
	idleChecked  map[string]time.Time
	idleInterval time.Duration
	interval     time.Duration
	stopCh       chan struct{}
	mu           sync.Mutex
	running      bool
}

// IdleVerdict is an IdleJudge's decision for one session.
type IdleVerdict struct {
	Archive bool
	// Note is the archive note (when Archive) or why the session was kept.
	Note string
}

// IdleJudge decides whether a session under the idle policy may be archived
// now. idleAfter is the policy's window. It must keep the session whenever any
// evidence is missing or unreadable: archiving is the action that needs proof.
type IdleJudge func(ctx context.Context, sess *Session, idleAfter time.Duration) IdleVerdict

// SetArchiveHandler wires expiry to the owning lifecycle service. It must be
// configured before Start is called.
func (es *ExpirationSweeper) SetArchiveHandler(handler func(ctx context.Context, id, note string) error) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.archive = handler
}

// SetIdleJudge installs the idle-mode decision. It must be configured before
// Start is called.
func (es *ExpirationSweeper) SetIdleJudge(judge IdleJudge) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.idle = judge
}

// NewExpirationSweeper creates a sweeper that checks for expired sessions.
func NewExpirationSweeper(sm *Manager, events *events.Logger, metrics *metrics.Metrics) *ExpirationSweeper {
	return &ExpirationSweeper{
		sessions:     sm,
		events:       events,
		metrics:      metrics,
		interval:     30 * time.Second,
		idleInterval: idleCheckInterval,
		idleChecked:  map[string]time.Time{},
		stopCh:       make(chan struct{}),
	}
}

// idleCheckInterval is how often one idle-policy session's evidence (rollout
// tree, goal store, process table) is read. An idle window is hours long, so a
// few minutes of lateness costs nothing.
const idleCheckInterval = 5 * time.Minute

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

// sweep checks all sessions and requests archival for expired or idle ones.
func (es *ExpirationSweeper) sweep() {
	sessions := es.sessions.List()
	es.mu.Lock()
	archive, judge := es.archive, es.idle
	es.mu.Unlock()
	live := make(map[string]bool, len(sessions))
	for _, sess := range sessions {
		live[sess.ID] = true
		p := sess.GetPolicy()
		if policy.IsExpired(sess.CreatedAt, p) {
			log.Printf("session %s: expired (policy=%s, duration=%s)", sess.ID, p.Mode, p.Duration)
			note := fmt.Sprintf("Archived by the %s expiry policy: the session reached its %s lifetime.", p.Mode, p.Duration)
			es.request(archive, sess.ID, note, map[string]string{
				"reason":   "expired",
				"policy":   string(p.Mode),
				"duration": p.Duration,
			})
			continue
		}
		idleAfter, ok := policy.IdleAfter(p)
		if !ok || judge == nil || !es.idleDue(sess.ID) {
			continue
		}
		verdict := judge(context.Background(), sess, idleAfter)
		if !verdict.Archive {
			continue
		}
		log.Printf("session %s: idle (policy=idle, duration=%s): %s", sess.ID, p.Duration, verdict.Note)
		es.request(archive, sess.ID, verdict.Note, map[string]string{
			"reason":   "idle",
			"policy":   string(p.Mode),
			"duration": p.Duration,
			"note":     verdict.Note,
		})
	}
	es.mu.Lock()
	for id := range es.idleChecked {
		if !live[id] {
			delete(es.idleChecked, id)
		}
	}
	es.mu.Unlock()
}

// idleDue reports whether a session's idle evidence is due for a read, and
// records the read.
func (es *ExpirationSweeper) idleDue(id string) bool {
	es.mu.Lock()
	defer es.mu.Unlock()
	if es.idleChecked == nil {
		es.idleChecked = map[string]time.Time{}
	}
	now := time.Now()
	if last, ok := es.idleChecked[id]; ok && now.Sub(last) < es.idleInterval {
		return false
	}
	es.idleChecked[id] = now
	return true
}

// request asks the lifecycle owner to archive one session.
func (es *ExpirationSweeper) request(archive func(context.Context, string, string) error, id, note string, detail map[string]string) {
	if archive == nil {
		log.Printf("session %s: archive skipped; continuity archive handler is not configured", id)
		return
	}
	if err := archive(context.Background(), id, note); err != nil {
		// A governed lifecycle request. Preserve the session and surface a
		// durable-domain failure instead of silently retrying or reporting a
		// deletion that never happened.
		log.Printf("session %s: policy archive failed: %v", id, err)
		es.metrics.ContinuityFailures.Add(1)
		return
	}
	// The canonical archive owner accounts for the active-session transition.
	// The policy is only a trigger; it must not double-count the transition or
	// report archive as deletion.
	es.events.Emit(events.SessionTerminated, id, detail)
}
