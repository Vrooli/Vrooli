package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CapturePolicy mirrors the Events-owned ReceiptCapturePolicy wire contract.
// Projection paths are explicit canonical paths (for example, plan.id).
type CapturePolicy struct {
	PolicyID string `json:"policy_id"`
	Enabled  bool   `json:"enabled"`
	Selector struct {
		TargetScenario string `json:"target_scenario"`
		Operation      string `json:"operation"`
		Protocol       string `json:"protocol"`
		EventType      string `json:"event_type"`
	} `json:"selector"`
	ResponseType            string   `json:"response_type"`
	ResponseProjectionPaths []string `json:"response_projection_paths"`
	RetentionDays           uint32   `json:"retention_days"`
	Version                 string   `json:"version"`
}

type PolicySnapshot struct {
	Version                string          `json:"version"`
	ReceiptCapturePolicies []CapturePolicy `json:"receipt_capture_policies"`
}

type Cache struct {
	mu         sync.RWMutex
	snapshot   PolicySnapshot
	receivedAt time.Time
	maxAge     time.Duration
}

// RuntimeState is the bounded health representation of the receipt runtime.
// It exposes connectivity and policy readiness without exposing policy bodies.
type RuntimeState struct {
	State       string    `json:"state"`
	Armed       bool      `json:"armed"`
	PolicyCount int       `json:"policyCount"`
	LastRefresh time.Time `json:"lastRefresh,omitempty"`
}

const defaultPolicyMaxAge = 5 * time.Minute

func NewCache() *Cache { return NewCacheWithMaxAge(defaultPolicyMaxAge) }
func NewCacheWithMaxAge(maxAge time.Duration) *Cache {
	if maxAge <= 0 {
		maxAge = defaultPolicyMaxAge
	}
	return &Cache{maxAge: maxAge}
}

func (c *Cache) Replace(next PolicySnapshot, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if next.Version == "" {
		return false
	}
	if next.Version == c.snapshot.Version {
		// A snapshot with the current version is still a successful refresh.
		// Keep the cache armed while Events confirms that the policy set remains
		// current; callers must not mistake a quiet policy stream for a stale one.
		c.receivedAt = now.UTC()
		return false
	}
	c.snapshot = next
	c.receivedAt = now.UTC()
	return true
}

func (c *Cache) Health(now time.Time) (version string, age time.Duration, usable bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.receivedAt.IsZero() {
		return "", 0, false
	}
	return c.snapshot.Version, now.Sub(c.receivedAt), now.Sub(c.receivedAt) <= c.maxAge
}

func (c *Cache) RuntimeState(now time.Time) RuntimeState {
	if c == nil {
		return RuntimeState{State: "never_connected"}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.receivedAt.IsZero() {
		return RuntimeState{State: "never_connected"}
	}
	state := "armed"
	if len(c.snapshot.ReceiptCapturePolicies) == 0 {
		state = "connected_empty"
	}
	if now.Sub(c.receivedAt) > c.maxAge {
		state = "stale"
	}
	return RuntimeState{State: state, Armed: now.Sub(c.receivedAt) <= c.maxAge, PolicyCount: len(c.snapshot.ReceiptCapturePolicies), LastRefresh: c.receivedAt}
}

// ProjectReceipt returns only declared response paths. An unavailable or stale
// policy deliberately means no observation, never a failed business request.
func (c *Cache) ProjectReceipt(_ string, target, operation, protocol string, candidate map[string]any) (map[string]any, string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.receivedAt.IsZero() || time.Since(c.receivedAt) > c.maxAge {
		return nil, "", false
	}
	for _, p := range c.snapshot.ReceiptCapturePolicies {
		if !p.Enabled || p.Selector.TargetScenario != target || p.Selector.Operation != operation || p.Selector.Protocol != protocol || p.Selector.EventType != ReceiptEventType {
			continue
		}
		projection := make(map[string]any, len(p.ResponseProjectionPaths))
		for _, path := range p.ResponseProjectionPaths {
			if value, ok := valueAtPath(candidate, path); ok {
				projection[path] = value
			}
		}
		return projection, p.Version, true
	}
	return nil, "", false
}

func valueAtPath(value map[string]any, path string) (any, bool) {
	var current any = value
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func (c Client) Refresh(ctx context.Context, cache *Cache) error {
	if cache == nil || !c.Enabled() {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.baseURL(), "/")+"/api/v1/policies/snapshot", nil)
	if err != nil {
		return err
	}
	h := c.HTTPClient
	if h == nil {
		h = http.DefaultClient
	}
	resp, err := h.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("vrooli-events snapshot: %s", resp.Status)
	}
	var snapshot PolicySnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		return err
	}
	cache.Replace(snapshot, time.Now())
	return nil
}
