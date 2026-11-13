package routing_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"

	"search-hub/internal/routing"
)

// rerankFixture mirrors testdata/rerank_queries.json.
type rerankFixture struct {
	Cases []struct {
		Query      string `json:"query"`
		Candidates []struct {
			ID      string `json:"id"`
			Type    string `json:"type"`
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
		} `json:"candidates"`
		ExpectedID string `json:"expected_id"`
	} `json:"cases"`
}

// TestCPULexicalRerankerFixtureMRR is a source regression over the existing
// synthetic ordering fixture. It must not be presented as held-out retrieval
// quality: the cases and relevance labels are constructed test data, not
// owner-reviewed production caller examples.
func TestCPULexicalRerankerFixtureMRR(t *testing.T) {
	fx := loadRerankFixture(t)
	raw, err := os.ReadFile("testdata/rerank_queries.json")
	require.NoError(t, err)
	corpusDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(raw))
	rr := routing.NewCPULexicalReranker()
	measurement := measureMRR(t, rr, fx)
	t.Logf("kind=synthetic_fixture_only fixture_id=search-hub.rerank_queries.v1 candidate=%s fixture_digest=%s fixture_denominator=%d baseline_mrr=%.3f fixture_mrr=%.3f fixture_p95_ms=%.3f test_allocated_bytes=%d cpu_budget_concurrency=1 cpu_budget_candidates=%d result=source_regression_only",
		rr.Name(), corpusDigest, len(fx.Cases), measurement.baselineMRR, measurement.rerankMRR,
		float64(measurement.rerankP95)/float64(time.Millisecond), measurement.allocatedBytes,
		routing.CPULexicalCandidateLimit)
	require.Positive(t, measurement.rerankMRR)
	require.False(t, math.IsNaN(measurement.rerankMRR))
	require.False(t, math.IsInf(measurement.rerankMRR, 0))
}

type mrrMeasurement struct {
	rerankMRR      float64
	baselineMRR    float64
	rerankP95      time.Duration
	allocatedBytes uint64
}

func measureMRR(t *testing.T, rr routing.Reranker, fx rerankFixture) mrrMeasurement {
	t.Helper()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	latencies := make([]time.Duration, 0, len(fx.Cases))
	var sumRerankRR, sumBaselineRR float64
	for _, c := range fx.Cases {
		candidates := make([]*routingv1.SearchHit, len(c.Candidates))
		baselineRank := 0
		for i, cand := range c.Candidates {
			candidates[i] = &routingv1.SearchHit{
				Id: cand.ID, Type: cand.Type, Title: cand.Title, Snippet: cand.Snippet,
			}
			if cand.ID == c.ExpectedID {
				baselineRank = i + 1 // pre-rerank order = input order
			}
		}
		require.Positivef(t, baselineRank, "expected_id %q not among candidates", c.ExpectedID)

		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		started := time.Now()
		ranked, err := rr.Rerank(ctx, c.Query, candidates)
		latencies = append(latencies, time.Since(started))
		cancel()
		require.NoErrorf(t, err, "rerank %q", c.Query)
		require.Len(t, ranked, len(candidates))

		rerankRank := rankOf(ranked, c.ExpectedID)
		require.Positive(t, rerankRank)
		sumRerankRR += 1.0 / float64(rerankRank)
		sumBaselineRR += 1.0 / float64(baselineRank)
		t.Logf("%-55q expected rank: baseline=%d rerank=%d", c.Query, baselineRank, rerankRank)
	}

	n := float64(len(fx.Cases))
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p95Index := (95*len(latencies)+99)/100 - 1
	if p95Index < 0 {
		p95Index = 0
	}
	allocated := uint64(0)
	if after.TotalAlloc >= before.TotalAlloc {
		allocated = after.TotalAlloc - before.TotalAlloc
	}
	return mrrMeasurement{rerankMRR: sumRerankRR / n, baselineMRR: sumBaselineRR / n, rerankP95: latencies[p95Index], allocatedBytes: allocated}
}

func rankOf(hits []*routingv1.SearchHit, id string) int {
	for i, h := range hits {
		if h.GetId() == id {
			return i + 1
		}
	}
	return 0
}

func loadRerankFixture(t *testing.T) rerankFixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/rerank_queries.json")
	require.NoError(t, err)
	var fx rerankFixture
	require.NoError(t, json.Unmarshal(raw, &fx))
	require.NotEmpty(t, fx.Cases)
	return fx
}
