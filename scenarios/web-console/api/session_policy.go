package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// DOC: docs/concepts/ARCHITECTURE.md#session-policy
// [REQ:P1-001a] Expiration Policy Engine

// PolicyMode defines the session expiration behavior.
type PolicyMode string

const (
	PolicyNever  PolicyMode = "never"
	PolicyPreset PolicyMode = "preset"
	PolicyCustom PolicyMode = "custom"
)

// Valid preset TTL durations.
// CROSS-LANGUAGE COUPLING: Must match POLICY_OPTIONS in ui/src/consts/policy-options.ts.
var presetDurations = map[string]time.Duration{
	"1h":  time.Hour,
	"8h":  8 * time.Hour,
	"24h": 24 * time.Hour,
}

// Custom duration bounds.
const (
	customDurationMin = time.Minute
	customDurationMax = 7 * 24 * time.Hour // 7 days
)

// ExpirationPolicy controls when a session expires.
type ExpirationPolicy struct {
	Mode     PolicyMode `json:"mode"`
	Duration string     `json:"duration,omitempty"` // e.g. "1h", "8h", "24h", or custom like "30m"
}

// DefaultPolicy returns the default never-expire policy.
func DefaultPolicy() ExpirationPolicy {
	return ExpirationPolicy{Mode: PolicyNever}
}

// ValidatePolicy checks that a policy is well-formed.
func ValidatePolicy(p ExpirationPolicy) error {
	switch p.Mode {
	case PolicyNever:
		return nil
	case PolicyPreset:
		if _, ok := presetDurations[p.Duration]; !ok {
			return fmt.Errorf("invalid preset duration %q; valid: 1h, 8h, 24h", p.Duration)
		}
		return nil
	case PolicyCustom:
		if p.Duration == "" {
			return fmt.Errorf("custom policy requires a duration")
		}
		d, err := time.ParseDuration(p.Duration)
		if err != nil {
			return fmt.Errorf("invalid custom duration %q: %w", p.Duration, err)
		}
		if d < customDurationMin {
			return fmt.Errorf("custom duration must be at least %s", customDurationMin)
		}
		if d > customDurationMax {
			return fmt.Errorf("custom duration must be at most %s", customDurationMax)
		}
		return nil
	default:
		return fmt.Errorf("invalid policy mode %q; valid: never, preset, custom", p.Mode)
	}
}

// ResolveTTL returns the duration for a policy.
// Returns 0 in three cases, all meaning "never expire":
//  1. Mode is PolicyNever (or any unrecognized mode)
//  2. Preset duration key not found in presetDurations map
//  3. Custom duration string fails to parse
//
// Cases 2 and 3 should be unreachable for policies that passed ValidatePolicy.
// Callers treat 0 as infinite lifetime (see IsExpired, buildPolicyResponse).
func ResolveTTL(p ExpirationPolicy) time.Duration {
	switch p.Mode {
	case PolicyPreset:
		if d, ok := presetDurations[p.Duration]; ok {
			return d
		}
		return 0
	case PolicyCustom:
		d, err := time.ParseDuration(p.Duration)
		if err != nil {
			return 0
		}
		return d
	default:
		return 0
	}
}

// IsExpired checks if a session has expired given its creation time and policy.
// [REQ:P1-001a] Expiration Policy Engine — <10ms evaluation
func IsExpired(createdAt time.Time, policy ExpirationPolicy) bool {
	ttl := ResolveTTL(policy)
	if ttl == 0 {
		return false
	}
	return time.Since(createdAt) > ttl
}

// ExpirationSweeper periodically checks sessions for expiration and removes expired ones.
type ExpirationSweeper struct {
	sessions *SessionManager
	events   *EventLogger
	metrics  *Metrics
	interval time.Duration
	stopCh   chan struct{}
	mu       sync.Mutex
	running  bool
}

// NewExpirationSweeper creates a sweeper that checks for expired sessions.
func NewExpirationSweeper(sm *SessionManager, events *EventLogger, metrics *Metrics) *ExpirationSweeper {
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
		policy := sess.GetPolicy()
		if IsExpired(sess.CreatedAt, policy) {
			log.Printf("session %s: expired (policy=%s, duration=%s)", sess.ID, policy.Mode, policy.Duration)
			if err := es.sessions.Delete(sess.ID); err == nil {
				es.events.Emit(EventSessionTerminated, sess.ID, map[string]string{
					"reason":   "expired",
					"policy":   string(policy.Mode),
					"duration": policy.Duration,
				})
				es.metrics.SessionsDeleted.Add(1)
				es.metrics.ActiveSessions.Add(-1)
			}
		}
	}
}
