package chainforensics

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"time"
)

var ErrCaptureLimiterCapacity = errors.New("public capture endpoint limiter capacity reached")

const defaultEndpointLimiterCapacity = 64

// EndpointRateLimiter shares a bounded, single-request gate by provider origin
// across Bitcoin, EVM and Solana clients. It applies the most conservative
// minimum interval requested for each provider and never retries failures.
type EndpointRateLimiter struct {
	mu       sync.Mutex
	capacity int
	gates    map[string]*endpointRateGate
}

type endpointRateGate struct {
	slot        chan struct{}
	interval    time.Duration
	lastRequest time.Time
	cooldownEnd time.Time
	lastUsed    time.Time
	users       int
}

func NewEndpointRateLimiter(capacity int) *EndpointRateLimiter {
	if capacity < 1 {
		capacity = 1
	}
	return &EndpointRateLimiter{capacity: capacity, gates: make(map[string]*endpointRateGate, capacity)}
}

var sharedEndpointRateLimiter = NewEndpointRateLimiter(defaultEndpointLimiterCapacity)

// Acquire waits for the provider's shared slot and returns a release function
// that must be called after the bounded read completes. Requests sharing a
// provider origin serialize, including reads made by separate client objects.
// A cooldown registered with DeferUntil also applies across all those clients.
func (l *EndpointRateLimiter) Acquire(ctx context.Context, endpoint string, minInterval time.Duration) (func(), error) {
	if ctx == nil {
		return nil, errors.New("capture limiter requires a context")
	}
	origin, err := endpointOrigin(endpoint)
	if err != nil {
		return nil, err
	}
	if minInterval < 0 {
		minInterval = 0
	}

	l.mu.Lock()
	gate := l.gates[origin]
	if gate == nil {
		if len(l.gates) >= l.capacity && !l.evictIdleLocked() {
			l.mu.Unlock()
			return nil, ErrCaptureLimiterCapacity
		}
		gate = &endpointRateGate{slot: make(chan struct{}, 1)}
		l.gates[origin] = gate
	}
	if minInterval > gate.interval {
		gate.interval = minInterval
	}
	gate.users++
	gate.lastUsed = time.Now()
	l.mu.Unlock()

	select {
	case <-ctx.Done():
		l.releaseUser(gate)
		return nil, ctx.Err()
	case gate.slot <- struct{}{}:
	}
	if err := ctx.Err(); err != nil {
		<-gate.slot
		l.releaseUser(gate)
		return nil, err
	}

	l.mu.Lock()
	now := time.Now()
	waitUntil := gate.lastRequest.Add(gate.interval)
	if gate.cooldownEnd.After(waitUntil) {
		waitUntil = gate.cooldownEnd
	}
	wait := waitUntil.Sub(now)
	l.mu.Unlock()
	if wait > 0 {
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			<-gate.slot
			l.releaseUser(gate)
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	if err := ctx.Err(); err != nil {
		<-gate.slot
		l.releaseUser(gate)
		return nil, err
	}
	l.mu.Lock()
	gate.lastRequest = time.Now()
	gate.lastUsed = gate.lastRequest
	l.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			<-gate.slot
			l.releaseUser(gate)
		})
	}, nil
}

// DeferUntil shares a provider-supplied retry window with every client using
// the same origin. It does not retry the failed request. Callers should pass a
// bounded server Retry-After timestamp after releasing any response body.
func (l *EndpointRateLimiter) DeferUntil(endpoint string, until time.Time) {
	origin, err := endpointOrigin(endpoint)
	if err != nil || until.IsZero() {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if gate := l.gates[origin]; gate != nil && until.After(gate.cooldownEnd) {
		gate.cooldownEnd = until
		gate.lastUsed = time.Now()
	}
}

func (l *EndpointRateLimiter) releaseUser(gate *endpointRateGate) {
	l.mu.Lock()
	gate.users--
	gate.lastUsed = time.Now()
	l.mu.Unlock()
}

func (l *EndpointRateLimiter) evictIdleLocked() bool {
	oldestOrigin := ""
	var oldest time.Time
	now := time.Now()
	for origin, gate := range l.gates {
		if gate.users == 0 && !gate.cooldownEnd.After(now) && (oldestOrigin == "" || gate.lastUsed.Before(oldest)) {
			oldestOrigin, oldest = origin, gate.lastUsed
		}
	}
	if oldestOrigin == "" {
		return false
	}
	delete(l.gates, oldestOrigin)
	return true
}

func endpointOrigin(endpoint string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", errors.New("capture endpoint must be an absolute HTTP or HTTPS URL")
	}
	return strings.ToLower(u.Scheme + "://" + u.Host), nil
}
