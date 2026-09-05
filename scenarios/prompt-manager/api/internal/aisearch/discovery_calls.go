package aisearch

import (
	"context"
	"log"
	"time"

	"prompt-manager/internal/store"
)

// DiscoveryCallStore is the per-call telemetry seam: the production
// implementation is *store.DiscoveryCallStore; tests inject a fake so recording
// and metrics are deterministic without a real filesystem home. It is parallel
// to DiscoveryMissStore but records EVERY discover call, not just misses.
type DiscoveryCallStore interface {
	Append(call store.DiscoveryCall) error
	ReadSince(window time.Duration) ([]store.DiscoveryCall, error)
}

// SetDiscoveryCallStore wires the per-call telemetry sink/source. When unset,
// recording is a no-op and metrics return empty.
func (s *Service) SetDiscoveryCallStore(callStore DiscoveryCallStore) {
	s.callStore = callStore
}

// SetDiscoveryProbeSample sets the 1-in-N sampling rate for the threshold
// -clipping probe. n <= 0 disables it (the default). The probe re-runs each
// query's skill search with no score floor to count results the active
// threshold dropped, so it doubles embed+search cost on sampled calls — hence
// opt-in.
func (s *Service) SetDiscoveryProbeSample(n int) {
	if n < 0 {
		n = 0
	}
	s.probeSample = n
}

// recordDiscoveryCall appends one telemetry record for a discover call. It is
// guarded and log-and-continue: a telemetry failure must never fail or slow the
// discover response.
func (s *Service) recordDiscoveryCall(ctx context.Context, resp *DiscoverResponse, queries []string, discoverType, complexity string, trimmedCount int, clipped *int) {
	if s.callStore == nil || resp == nil {
		return
	}
	results := make([]store.DiscoveryCallResult, 0, len(resp.Results))
	for _, r := range resp.Results {
		results = append(results, store.DiscoveryCallResult{
			ID:     r.ID,
			Score:  r.Score,
			Chars:  r.ContentChars,
			Source: r.Source,
			Type:   r.Type,
		})
	}
	callType := discoverType
	if callType == "" {
		callType = "skill"
	}
	call := store.DiscoveryCall{
		Queries:               queries,
		Type:                  callType,
		Complexity:            complexity,
		Threshold:             s.threshold,
		BudgetChars:           resp.BudgetChars,
		TotalContentChars:     resp.TotalContentChars,
		BudgetStatus:          resp.BudgetStatus,
		ReturnedCount:         len(resp.Results),
		TrimmedCount:          trimmedCount,
		ClippedBelowThreshold: clipped,
		Results:               results,
		Caller:                CallerFrom(ctx),
	}
	if err := s.callStore.Append(call); err != nil {
		log.Printf("[aisearch] discovery-call append failed (continuing): %v", err)
	}
}

// maybeProbeClipping returns a non-nil count of clipped results only on sampled
// calls (when probeSample > 0 and this call's sequence number hits the sample).
// A nil return means "not probed", distinct from a probed-zero result.
func (s *Service) maybeProbeClipping(ctx context.Context, queries []string, results []DiscoverResult, limit int) *int {
	if s.probeSample <= 0 || s.callStore == nil {
		return nil
	}
	seq := s.callSeq.Add(1)
	if seq%uint64(s.probeSample) != 0 {
		return nil
	}
	returned := make(map[string]bool, len(results))
	for _, r := range results {
		if r.Type == "action" {
			continue
		}
		returned[r.ID] = true
	}
	count := s.probeClippingBelowThreshold(ctx, queries, returned, limit)
	return &count
}

// probeClippingBelowThreshold re-runs each query's skill vector search with no
// score floor and counts distinct results scoring in [0, threshold) that the
// live threshold would have dropped (excluding results already returned). This
// is the direct measurement of "how much did the threshold cost us".
func (s *Service) probeClippingBelowThreshold(ctx context.Context, queries []string, returned map[string]bool, limit int) int {
	if s.embedder == nil || s.vectorStore == nil {
		return 0
	}
	if limit <= 0 {
		limit = 10
	}
	clipped := make(map[string]bool)
	for _, query := range queries {
		vector, err := s.embedder.Embed(ctx, query)
		if err != nil {
			continue
		}
		hits, err := s.vectorStore.Search(ctx, vector, limit, 0)
		if err != nil {
			continue
		}
		for _, hit := range hits {
			id := hit.ID
			if pid, ok := hit.Payload["skill_id"].(string); ok && pid != "" {
				id = pid
			}
			if hit.Score >= s.threshold || returned[id] || clipped[id] {
				continue
			}
			clipped[id] = true
		}
	}
	return len(clipped)
}
