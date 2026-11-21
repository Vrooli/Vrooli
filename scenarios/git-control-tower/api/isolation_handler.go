package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"

	eligpb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/eligibility"
)

// eligibilityChecker is the seam between the handler and the Connect-RPC
// client. *TestGenieEligibilityClient satisfies it; tests inject a stub.
type eligibilityChecker interface {
	Check(ctx context.Context, scenario string) (*eligpb.CheckResponse, error)
}

// IsolationStatus enumerates the possible test-genie eligibility states GCT
// surfaces to the UI. The UI's IsolationBadge keys colour and copy off this
// value.
type IsolationStatus string

const (
	IsolationStatusRouted    IsolationStatus = "routed"
	IsolationStatusNotRouted IsolationStatus = "not_routed"
	IsolationStatusUnknown   IsolationStatus = "unknown"
)

// IsolationViolation is the GCT-facing shape of a test-genie violation. It
// is intentionally JSON-tagged for direct UI consumption.
type IsolationViolation struct {
	RuleID   string `json:"rule_id"`
	Severity string `json:"severity"`
	File     string `json:"file,omitempty"`
	Line     uint32 `json:"line,omitempty"`
	Excerpt  string `json:"excerpt,omitempty"`
}

// IsolationResponse is the payload returned by GET /api/v1/scenarios/:slug/isolation.
type IsolationResponse struct {
	Status     IsolationStatus      `json:"status"`
	Reasons    []string             `json:"reasons,omitempty"`
	Violations []IsolationViolation `json:"violations,omitempty"`
}

// IsolationCache memoises per-scenario test-genie eligibility responses for
// a fixed TTL. Bypass by calling Invalidate or by reading the underlying
// client directly.
type IsolationCache struct {
	ttl   time.Duration
	mu    sync.Mutex
	cache map[string]isolationCacheEntry
}

type isolationCacheEntry struct {
	resp      IsolationResponse
	expiresAt time.Time
}

// NewIsolationCache returns a cache with the given TTL. ttl<=0 disables
// caching.
func NewIsolationCache(ttl time.Duration) *IsolationCache {
	return &IsolationCache{ttl: ttl, cache: map[string]isolationCacheEntry{}}
}

// Get returns the cached response and true when present and unexpired.
func (c *IsolationCache) Get(scenario string) (IsolationResponse, bool) {
	if c.ttl <= 0 {
		return IsolationResponse{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.cache[scenario]
	if !ok || time.Now().After(entry.expiresAt) {
		return IsolationResponse{}, false
	}
	return entry.resp, true
}

// Put inserts or refreshes a cached response.
func (c *IsolationCache) Put(scenario string, resp IsolationResponse) {
	if c.ttl <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[scenario] = isolationCacheEntry{
		resp:      resp,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Invalidate drops a scenario's cache entry.
func (c *IsolationCache) Invalidate(scenario string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, scenario)
}

// handleScenarioIsolation proxies test-genie's EligibilityService and
// normalizes the response for the UI. The endpoint never returns a 5xx for
// downstream failures: unreachable test-genie maps to IsolationStatusUnknown
// so the UI can render a third state instead of an error banner.
func (s *Server) handleScenarioIsolation(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(mux.Vars(r)["slug"])
	if slug == "" {
		http.Error(w, "scenario slug is required", http.StatusBadRequest)
		return
	}

	if s.isolationCache != nil {
		if cached, ok := s.isolationCache.Get(slug); ok {
			writeIsolationJSON(w, cached)
			return
		}
	}

	var checker eligibilityChecker
	if s.testGenieEligibility != nil {
		checker = s.testGenieEligibility
	}
	resp := resolveIsolation(r.Context(), checker, slug)

	if s.isolationCache != nil {
		s.isolationCache.Put(slug, resp)
	}
	writeIsolationJSON(w, resp)
}

// resolveIsolation runs the eligibility check and normalizes the response.
// Pulled out of the handler so tests can exercise mapping logic directly.
func resolveIsolation(ctx context.Context, client eligibilityChecker, slug string) IsolationResponse {
	if client == nil {
		return IsolationResponse{Status: IsolationStatusUnknown, Reasons: []string{"test-genie eligibility client is not configured"}}
	}
	pb, err := client.Check(ctx, slug)
	if err != nil {
		return IsolationResponse{
			Status:  IsolationStatusUnknown,
			Reasons: []string{"test-genie unreachable: " + err.Error()},
		}
	}

	var violations []IsolationViolation
	for _, v := range pb.GetViolations() {
		violations = append(violations, IsolationViolation{
			RuleID:   v.GetRuleId(),
			Severity: v.GetSeverity(),
			File:     v.GetFile(),
			Line:     v.GetLine(),
			Excerpt:  v.GetExcerpt(),
		})
	}

	resp := IsolationResponse{
		Reasons:    pb.GetDisqualifyingReasons(),
		Violations: violations,
	}
	if pb.GetRouted() {
		resp.Status = IsolationStatusRouted
	} else {
		resp.Status = IsolationStatusNotRouted
	}
	return resp
}

func writeIsolationJSON(w http.ResponseWriter, resp IsolationResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
