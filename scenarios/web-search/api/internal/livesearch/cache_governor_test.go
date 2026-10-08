package livesearch_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"web-search/internal/livesearch"

	"github.com/vrooli/api-core/scheduletest"
)

func cachedResults() []livesearch.Result {
	return []livesearch.Result{
		{URL: "https://anthropic.com", Title: "Anthropic", Snippet: "Claude maker", Engine: "google", Score: 0.9},
	}
}

// TestCacheKeyEquivalentQueriesHit pins key normalization: queries that differ
// only in surrounding whitespace and letter case are the SAME cache entry, so a
// repeat phrasing never spends budget on a fresh SearXNG call.
func TestCacheKeyEquivalentQueriesHit(t *testing.T) {
	clk := scheduletest.New(time.Time{})
	cache := livesearch.NewCache(time.Minute, clk)

	cache.Put("Anthropic Claude", 5, cachedResults(), nil)

	got, _, ok := cache.Get("  anthropic claude  ", 5)
	require.True(t, ok, "case/whitespace-equivalent query must hit the same cache entry")
	require.Equal(t, cachedResults(), got)

	// A different limit is a different result set — must NOT collide.
	_, _, ok = cache.Get("anthropic claude", 10)
	require.False(t, ok, "same query with a different limit is a distinct entry")
}

// TestCacheHitWithinTTL pins freshness: an entry stays servable for the whole
// TTL window and is gone right after it.
func TestCacheHitWithinTTL(t *testing.T) {
	clk := scheduletest.New(time.Time{})
	cache := livesearch.NewCache(time.Minute, clk)
	cache.Put("q", 5, cachedResults(), nil)

	clk.Advance(59 * time.Second)
	_, _, ok := cache.Get("q", 5)
	require.True(t, ok, "entry must remain a hit within the TTL window")

	clk.Advance(2 * time.Second) // now past the 1m TTL
	_, _, ok = cache.Get("q", 5)
	require.False(t, ok, "entry must expire once the TTL elapses")
}

// TestGovernorExhaustedAfterBurst pins the burst limit: exactly capacity calls
// are allowed in a window; every further call in the same window is declined
// (zero tokens remain) — the caller must degrade instead of hitting SearXNG.
func TestGovernorExhaustedAfterBurst(t *testing.T) {
	clk := scheduletest.New(time.Time{})
	gov := livesearch.NewGovernor(3, time.Minute, clk)

	for i := 0; i < 3; i++ {
		require.True(t, gov.Allow(), "call %d is within the burst capacity", i+1)
	}
	require.False(t, gov.Allow(), "the bucket is exhausted after the burst limit")
	require.False(t, gov.Allow(), "exhaustion persists for the rest of the window")
}

// TestGovernorRefillsToCapacityAfterWindow pins the refill contract: once one
// full window elapses, the bucket is back to FULL capacity (not one token, not
// a partial trickle).
func TestGovernorRefillsToCapacityAfterWindow(t *testing.T) {
	clk := scheduletest.New(time.Time{})
	gov := livesearch.NewGovernor(3, time.Minute, clk)

	for i := 0; i < 3; i++ {
		require.True(t, gov.Allow())
	}
	require.False(t, gov.Allow(), "exhausted before the window rolls")

	clk.Advance(time.Minute)
	for i := 0; i < 3; i++ {
		require.True(t, gov.Allow(), "refill restores the full capacity (call %d)", i+1)
	}
	require.False(t, gov.Allow(), "the refilled window has the same burst limit")
}

// TestGovernorRefillNeverExceedsCapacity pins monotonic refill: several idle
// windows do NOT bank extra tokens — the bucket caps at the configured
// capacity, so a quiet period can never fund a later super-burst.
func TestGovernorRefillNeverExceedsCapacity(t *testing.T) {
	clk := scheduletest.New(time.Time{})
	gov := livesearch.NewGovernor(2, time.Minute, clk)

	require.True(t, gov.Allow()) // partial spend, then a long idle stretch

	clk.Advance(5 * time.Minute)
	require.True(t, gov.Allow())
	require.True(t, gov.Allow())
	require.False(t, gov.Allow(), "refill must cap at capacity, never accumulate across idle windows")
}

// BenchmarkCacheGet measures the cache-hit fast path (REQ-P0-007 performance
// budget: a hit serves with no network I/O, well under 50ms — the real cost is
// one mutex + map lookup).
func BenchmarkCacheGet(b *testing.B) {
	cache := livesearch.NewCache(time.Hour, scheduletest.New(time.Time{}))
	cache.Put("anthropic claude", 5, cachedResults(), nil)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, _, ok := cache.Get("anthropic claude", 5); !ok {
			b.Fatal("expected a cache hit")
		}
	}
}

// BenchmarkGovernorAllow measures the budget check on the request path
// (REQ-P0-007 performance budget: at most 1ms overhead; the real cost is one
// mutex + clock read, both on the allow and the decline branch).
func BenchmarkGovernorAllow(b *testing.B) {
	gov := livesearch.NewGovernor(64, time.Minute, scheduletest.New(time.Time{}))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		gov.Allow()
	}
}
