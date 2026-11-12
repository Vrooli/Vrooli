package hub

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ProducerKind separates the cooldown budgets, so informational sends never
// use up the budget for decision asks.
type ProducerKind string

const (
	ProducerNotification ProducerKind = "notification"
	ProducerAsk          ProducerKind = "ask"
)

// CooldownLimit admits at most Count new items per producer in any sliding
// Window. A zero Count disables the limit for that kind.
type CooldownLimit struct {
	Count  int
	Window time.Duration
}

func (l CooldownLimit) String() string {
	if l.Count <= 0 || l.Window <= 0 {
		return "off"
	}
	return fmt.Sprintf("%d/%s", l.Count, l.Window)
}

// Default per-producer limits. A producer is one verified agent run or the
// verified owner. Six informational notifications or three decision asks in
// ten minutes covers normal progress reporting; a looping agent stops after
// that many phone alerts instead of sending one per iteration.
var (
	DefaultNotificationCooldown = CooldownLimit{Count: 6, Window: 10 * time.Minute}
	DefaultAskCooldown          = CooldownLimit{Count: 3, Window: 10 * time.Minute}
)

// ErrProducerCooldown marks a refusal by the per-producer cooldown.
var ErrProducerCooldown = errors.New("producer cooldown")

// CooldownError is the refusal a producer receives over its limit. Nothing is
// stored: the producer must retry after RetryAt.
type CooldownError struct {
	Kind    ProducerKind
	Limit   CooldownLimit
	RetryAt time.Time
	// RetryAfter is RetryAt relative to the refusal time, rounded up to whole seconds.
	RetryAfter time.Duration
}

func (e *CooldownError) Error() string {
	what, advice := "notifications", "Combine updates into one notification, or reuse --dedupe-key so repeats collapse."
	if e.Kind == ProducerAsk {
		what, advice = "decision asks", "The ask was not created; retry it after that time, or combine the questions into one ask."
	}
	return fmt.Sprintf("%s: this producer reached %d %s per %s; retry after %s (in %s). %s",
		ErrProducerCooldown, e.Limit.Count, what, e.Limit.Window, e.RetryAt.UTC().Format(time.RFC3339), e.RetryAfter, advice)
}

func (e *CooldownError) Unwrap() error { return ErrProducerCooldown }

// ParseCooldownLimit reads "<count>/<Go duration>", for example "6/10m".
// "off" or "0" disables the limit.
func ParseCooldownLimit(value string) (CooldownLimit, error) {
	value = strings.TrimSpace(value)
	if strings.EqualFold(value, "off") || value == "0" {
		return CooldownLimit{}, nil
	}
	count, window, ok := strings.Cut(value, "/")
	if !ok {
		return CooldownLimit{}, fmt.Errorf("cooldown %q must be <count>/<duration>, for example 6/10m, or off", value)
	}
	n, err := strconv.Atoi(strings.TrimSpace(count))
	if err != nil || n < 1 {
		return CooldownLimit{}, fmt.Errorf("cooldown %q needs a positive count", value)
	}
	d, err := time.ParseDuration(strings.TrimSpace(window))
	if err != nil || d <= 0 {
		return CooldownLimit{}, fmt.Errorf("cooldown %q needs a positive Go duration", value)
	}
	return CooldownLimit{Count: n, Window: d}, nil
}

// producerCooldown is an in-memory sliding window per (producer, kind). It is
// a flood guard, not an audit record: a restart starts every window again.
type producerCooldown struct {
	mu     sync.Mutex
	limits map[ProducerKind]CooldownLimit
	hits   map[string][]time.Time
}

func newProducerCooldown() *producerCooldown {
	return &producerCooldown{
		limits: map[ProducerKind]CooldownLimit{ProducerNotification: DefaultNotificationCooldown, ProducerAsk: DefaultAskCooldown},
		hits:   map[string][]time.Time{},
	}
}

// admit reserves one slot or returns a *CooldownError. release returns the
// slot when the request created nothing.
func (c *producerCooldown) admit(producer string, kind ProducerKind, now time.Time) (release func(), err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	limit := c.limits[kind]
	if limit.Count <= 0 || limit.Window <= 0 {
		return func() {}, nil
	}
	key := string(kind) + "\x00" + producer
	kept := c.hits[key][:0]
	for _, at := range c.hits[key] {
		if now.Sub(at) < limit.Window {
			kept = append(kept, at)
		}
	}
	if len(kept) >= limit.Count {
		c.hits[key] = kept
		retryAt := kept[0].Add(limit.Window)
		wait := retryAt.Sub(now)
		if rounded := wait.Truncate(time.Second); rounded < wait {
			wait = rounded + time.Second
		}
		return nil, &CooldownError{Kind: kind, Limit: limit, RetryAt: retryAt, RetryAfter: wait}
	}
	c.hits[key] = append(kept, now)
	c.pruneLocked(now)
	return func() { c.release(key, now) }, nil
}

func (c *producerCooldown) release(key string, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	hits := c.hits[key]
	for i := len(hits) - 1; i >= 0; i-- {
		if hits[i].Equal(at) {
			c.hits[key] = append(hits[:i], hits[i+1:]...)
			break
		}
	}
	if len(c.hits[key]) == 0 {
		delete(c.hits, key)
	}
}

// pruneLocked drops producers whose window has passed, so finished runs do not
// accumulate. The caller holds c.mu.
func (c *producerCooldown) pruneLocked(now time.Time) {
	longest := time.Duration(0)
	for _, limit := range c.limits {
		if limit.Window > longest {
			longest = limit.Window
		}
	}
	for key, hits := range c.hits {
		if len(hits) == 0 || now.Sub(hits[len(hits)-1]) >= longest {
			delete(c.hits, key)
		}
	}
}

// SetProducerCooldown replaces the limit for one kind. A zero limit disables it.
func (s *Service) SetProducerCooldown(kind ProducerKind, limit CooldownLimit) {
	s.cooldown.mu.Lock()
	defer s.cooldown.mu.Unlock()
	s.cooldown.limits[kind] = limit
}

// ProducerCooldown returns the active limit for one kind.
func (s *Service) ProducerCooldown(kind ProducerKind) CooldownLimit {
	s.cooldown.mu.Lock()
	defer s.cooldown.mu.Unlock()
	return s.cooldown.limits[kind]
}

// SendFrom accepts a direct notification from one producer under its
// cooldown. A replay of an existing idempotency key returns the stored
// notification and uses no budget.
func (s *Service) SendFrom(ctx context.Context, producer string, in SendInput) (Notification, error) {
	if strings.TrimSpace(in.IdempotencyKey) != "" {
		if existing, err := s.findByIdempotency(ctx, in.RequestedBy, in.IdempotencyKey); err == nil {
			return existing, nil
		}
	}
	release, err := s.cooldown.admit(producer, ProducerNotification, s.clock.Now())
	if err != nil {
		return Notification{}, err
	}
	n, created, err := s.persistNotification(ctx, in)
	if err != nil || !created {
		release()
	}
	if err != nil {
		return Notification{}, err
	}
	if created {
		s.dispatch(n.ID)
	}
	return n, nil
}

// AskFrom creates a direct decision ask from one producer under its cooldown.
// A refused ask is never stored, and the producer receives the retry time. A
// replay of an existing idempotency key uses no budget.
func (s *Service) AskFrom(ctx context.Context, producer string, spec AskSpec) (AskRecord, Notification, error) {
	if normalized, err := normalizeAskSpec(spec); err == nil && normalized.IdempotencyKey != "" {
		if _, lookupErr := s.askByRequester(ctx, normalized.Source, normalized.IdempotencyKey); lookupErr == nil {
			return s.AskWithSpec(ctx, spec)
		}
	}
	release, err := s.cooldown.admit(producer, ProducerAsk, s.clock.Now())
	if err != nil {
		return AskRecord{}, Notification{}, err
	}
	ask, n, err := s.AskWithSpec(ctx, spec)
	if err != nil {
		release()
	}
	return ask, n, err
}
