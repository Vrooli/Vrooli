package hub

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// PushBudgetConfig limits device pushes per producer while leaving accepted
// notifications and asks durable in the hub. RefillInterval is the time needed
// to restore one token; Capacity is the burst size. Capacity <= 0 disables the
// limit.
type PushBudgetConfig struct {
	Capacity       int
	RefillInterval time.Duration
}

func (c PushBudgetConfig) String() string {
	if c.Capacity <= 0 || c.RefillInterval <= 0 {
		return "off"
	}
	return fmt.Sprintf("capacity=%d refill=%s/token", c.Capacity, c.RefillInterval)
}

var DefaultPushBudget = PushBudgetConfig{Capacity: 6, RefillInterval: time.Minute}

// ParsePushBudgetConfig parses the capacity and refill rate separately so
// operators can tune burst size and recovery speed independently.
func ParsePushBudgetConfig(capacity, refillInterval string) (PushBudgetConfig, error) {
	count, err := strconv.Atoi(strings.TrimSpace(capacity))
	if err != nil || count < 0 {
		return PushBudgetConfig{}, fmt.Errorf("push bucket capacity must be a non-negative integer")
	}
	if count == 0 {
		return PushBudgetConfig{}, nil
	}
	interval, err := time.ParseDuration(strings.TrimSpace(refillInterval))
	if err != nil || interval <= 0 {
		return PushBudgetConfig{}, fmt.Errorf("push bucket refill interval must be a positive Go duration")
	}
	return PushBudgetConfig{Capacity: count, RefillInterval: interval}, nil
}

type pushBudget struct {
	mu        sync.Mutex
	config    PushBudgetConfig
	buckets   map[string]pushBucket
	lastPrune time.Time
}

type pushBucket struct {
	tokens  float64
	updated time.Time
}

func newPushBudget() *pushBudget {
	return &pushBudget{config: DefaultPushBudget, buckets: make(map[string]pushBucket)}
}

// allowPush spends one token when available. All source kinds share the same
// bucket, so an ask and an informational notification consume equal push
// budget. Identityless host callers share one stable source key.
func (b *pushBudget) allowPush(source string, now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	cfg := b.config
	if cfg.Capacity <= 0 || cfg.RefillInterval <= 0 {
		return true
	}
	if source == "" {
		source = "anonymous"
	}
	state, ok := b.buckets[source]
	if !ok {
		state = pushBucket{tokens: float64(cfg.Capacity), updated: now}
	} else {
		elapsed := now.Sub(state.updated)
		if elapsed > 0 {
			state.tokens += float64(elapsed) / float64(cfg.RefillInterval)
			if state.tokens > float64(cfg.Capacity) {
				state.tokens = float64(cfg.Capacity)
			}
			state.updated = now
		}
	}
	allowed := state.tokens >= 1
	if allowed {
		state.tokens--
	}
	b.buckets[source] = state
	if len(b.buckets) > maxPushBudgetSources || now.Sub(b.lastPrune) >= time.Minute {
		b.prune(now, cfg)
		b.lastPrune = now
	}
	return allowed
}

const maxPushBudgetSources = 10_000

func (b *pushBudget) prune(now time.Time, cfg PushBudgetConfig) {
	// A full bucket has no useful history. Drop idle entries, and cap retained
	// sources so caller-selected identities cannot grow this in-memory guard
	// without bound.
	idleLimit := time.Duration(cfg.Capacity) * cfg.RefillInterval * 2
	if idleLimit < time.Hour {
		idleLimit = time.Hour
	}
	for key, state := range b.buckets {
		if state.tokens >= float64(cfg.Capacity) && now.Sub(state.updated) >= idleLimit {
			delete(b.buckets, key)
		}
	}
	if len(b.buckets) <= maxPushBudgetSources {
		return
	}
	keys := make([]string, 0, len(b.buckets))
	for key := range b.buckets {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return b.buckets[keys[i]].updated.Before(b.buckets[keys[j]].updated)
	})
	for _, key := range keys[:len(keys)-maxPushBudgetSources] {
		delete(b.buckets, key)
	}
}

func (s *Service) SetPushBudget(config PushBudgetConfig) {
	s.pushBudget.mu.Lock()
	defer s.pushBudget.mu.Unlock()
	s.pushBudget.config = config
	for key, state := range s.pushBudget.buckets {
		if state.tokens > float64(config.Capacity) {
			state.tokens = float64(config.Capacity)
			s.pushBudget.buckets[key] = state
		}
	}
}

func (s *Service) PushBudget() PushBudgetConfig {
	s.pushBudget.mu.Lock()
	defer s.pushBudget.mu.Unlock()
	return s.pushBudget.config
}

// SendFrom accepts a direct notification regardless of producer identity. A
// producer's source key only selects its device-push bucket. Idempotent
// replays do not spend budget or schedule another notification.
func (s *Service) SendFrom(ctx context.Context, source string, in SendInput) (Notification, error) {
	if strings.TrimSpace(in.IdempotencyKey) != "" {
		if existing, err := s.findByIdempotency(ctx, in.RequestedBy, in.IdempotencyKey); err == nil {
			return existing, nil
		}
	}
	n, created, err := s.persistNotification(ctx, in)
	if err != nil {
		return Notification{}, err
	}
	if created {
		s.dispatchDirect(n.ID, source)
	}
	return n, nil
}

func (s *Service) dispatchDirect(id, source string) {
	if !s.pushBudget.allowPush(source, s.clock.Now()) {
		s.pushMu.Lock()
		s.noDevicePush[id] = struct{}{}
		s.pushMu.Unlock()
	}
	s.dispatch(id)
}

func (s *Service) takeNoDevicePush(id string) bool {
	s.pushMu.Lock()
	defer s.pushMu.Unlock()
	if _, ok := s.noDevicePush[id]; !ok {
		return false
	}
	delete(s.noDevicePush, id)
	return true
}
