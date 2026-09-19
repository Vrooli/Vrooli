// Package ratelimit is a fixed-window, backend-authoritative limiter for the
// brute-force-sensitive auth endpoints (login/register). Unlike the old
// scenario's limiter — which kept an in-process map as the source of truth and
// only mirrored to Redis as a fire-and-forget side effect (so the "distributed"
// path was a no-op and replicas never shared counts) — this one increments the
// shared store atomically and is the single source of truth, so it actually
// holds across replicas behind a load balancer.
package ratelimit

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"scenario-authenticator/internal/redisstate"
)

// Config tunes the limiter.
type Config struct {
	// Limit is the max requests per window per identifier+path.
	Limit int
	// Window is the fixed window length.
	Window time.Duration
	// PathPrefixes restricts limiting to requests whose path has one of these
	// prefixes. Empty means every request is limited.
	PathPrefixes []string
}

// Limiter enforces a fixed-window limit over a redisstate.Store.
type Limiter struct {
	store redisstate.Store
	cfg   Config
	now   func() time.Time
}

// New constructs a Limiter. Redis is required for this control, so a nil store
// or a store error fails the protected request closed instead of silently
// bypassing the limiter.
func New(store redisstate.Store, cfg Config, now func() time.Time) *Limiter {
	if now == nil {
		now = time.Now
	}
	if cfg.Limit <= 0 {
		cfg.Limit = 20
	}
	if cfg.Window <= 0 {
		cfg.Window = time.Minute
	}
	return &Limiter{store: store, cfg: cfg, now: now}
}

// Middleware wraps next, enforcing the limit on matching paths.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.applies(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if l.store == nil {
			unavailable(w)
			return
		}
		allowed, remaining, resetAt, err := l.allow(r.Context(), identifier(r)+":"+r.URL.Path)
		if err != nil {
			unavailable(w)
			return
		}
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(l.cfg.Limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(time.Until(resetAt).Seconds())+1))
			http.Error(w, `{"code":"resource_exhausted","message":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func unavailable(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	http.Error(w, `{"code":"unavailable","message":"rate limit store unavailable"}`, http.StatusServiceUnavailable)
}

func (l *Limiter) applies(path string) bool {
	if len(l.cfg.PathPrefixes) == 0 {
		return true
	}
	for _, p := range l.cfg.PathPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// allow atomically increments the window counter and reports whether the
// request is within the limit.
func (l *Limiter) allow(ctx context.Context, key string) (allowed bool, remaining int, resetAt time.Time, err error) {
	full := "ratelimit:" + key
	n, err := l.store.Incr(ctx, full)
	if err != nil {
		return false, 0, time.Time{}, err
	}
	if n == 1 {
		// First hit in a new window — set the window TTL.
		_ = l.store.Expire(ctx, full, l.cfg.Window)
	}
	resetAt = l.now().Add(l.cfg.Window)
	remaining = l.cfg.Limit - int(n)
	if remaining < 0 {
		remaining = 0
	}
	return int(n) <= l.cfg.Limit, remaining, resetAt, nil
}

// identifier keys the limit on the client IP (the forwarding headers then the
// peer address).
func identifier(r *http.Request) string {
	if ff := r.Header.Get("X-Forwarded-For"); ff != "" {
		return "ip:" + strings.TrimSpace(strings.Split(ff, ",")[0])
	}
	if ri := r.Header.Get("X-Real-IP"); ri != "" {
		return "ip:" + ri
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return "ip:" + host
	}
	return "ip:" + r.RemoteAddr
}
