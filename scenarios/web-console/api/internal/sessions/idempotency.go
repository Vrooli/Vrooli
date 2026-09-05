package sessions

import (
	"sync"
	"time"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shared"

	"web-console/internal/backend"
	"web-console/internal/policy"
)

const idempotencyTTL = 5 * time.Minute

// Response is the cached JSON shape stored by IdempotencyCache so retried
// session-creation requests return the same payload as the original. The
// on-wire shape itself is defined by the SessionsService proto; this type
// only survives because the cache (and a handful of tests) round-trip
// through it.
type Response struct {
	ID              string        `json:"id"`
	Shell           string        `json:"shell"`
	CreatedAt       string        `json:"created_at"`
	Cols            int           `json:"cols"`
	Rows            int           `json:"rows"`
	Backend         backend.ID    `json:"backend"`
	SurvivesRestart bool          `json:"survives_restart"`
	Policy          policy.Policy `json:"policy"`
	Recovered       bool          `json:"recovered,omitempty"`
	// Provenance. Carried on the cached response so a replayed (idempotent)
	// create returns the same origin/owner/label as the original.
	Origin       string `json:"origin,omitempty"`
	Owner        string `json:"owner,omitempty"`
	DisplayLabel string `json:"display_label,omitempty"`
	// Target is a safe projection only; credentials and transport fields never
	// enter this cache or the SessionsService response.
	Target *sharedv1.Target `json:"target,omitempty"`
	// Fingerprint is an internal request digest used to reject accidental reuse
	// of one idempotency key for a different create request.
	Fingerprint string `json:"-"`
}

type idempotencyEntry struct {
	response  Response
	expiresAt time.Time
}

// IdempotencyCache is a bounded, TTL-scoped cache keyed by the client's
// X-Idempotency-Key header that prevents duplicate session creation on
// retry.
type IdempotencyCache struct {
	mu      sync.Mutex
	entries map[string]idempotencyEntry
	ttl     time.Duration
}

func NewIdempotencyCache() *IdempotencyCache {
	return &IdempotencyCache{
		entries: make(map[string]idempotencyEntry),
		ttl:     idempotencyTTL,
	}
}

func (c *IdempotencyCache) Get(key string) (Response, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		delete(c.entries, key)
		return Response{}, false
	}
	return entry.response, true
}

func (c *IdempotencyCache) Set(key string, resp Response) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = idempotencyEntry{
		response:  resp,
		expiresAt: time.Now().Add(c.ttl),
	}
	if len(c.entries) > 100 {
		now := time.Now()
		for k, v := range c.entries {
			if now.After(v.expiresAt) {
				delete(c.entries, k)
			}
		}
	}
}
