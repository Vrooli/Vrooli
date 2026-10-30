package routing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	"testing"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"time"
)

func TestDefaultFanoutBudgetFitsActiveFleet(t *testing.T) {
	// The honest baseline counted 34 active leaves. This is intentionally a
	// fleet-sized invariant rather than a two-provider unit fixture.
	const activeProviders = 34
	worst := fanoutWorstCase(activeProviders, defaultConcurrency, defaultPerProviderTimeout)
	if worst >= defaultQueryTimeout {
		t.Fatalf("fan-out budget exceeds query budget: %s >= %s (providers=%d concurrency=%d per-provider=%s)", worst, defaultQueryTimeout, activeProviders, defaultConcurrency, defaultPerProviderTimeout)
	}
	if got, want := worst, 20*time.Second; got != want {
		t.Fatalf("default fleet budget = %s, want %s", got, want)
	}
}

func TestFanoutWorstCaseUsesConcurrencyWaves(t *testing.T) {
	if got, want := fanoutWorstCase(9, 4, 2*time.Second), 6*time.Second; got != want {
		t.Fatalf("worst case = %s, want %s", got, want)
	}
}

// TestDefaultCrossEncoderBudgetCoversMeasuredLoadedLatency guards the
// re-measured cross-encoder bound. On 2026-09-12 the TEI reranker served
// 32x700-char docs to 32 concurrent callers at p95 ~1.73s (32x120-char p95
// ~0.38s), so the previous 500ms bound produced false timeouts and opened the
// 60s rerank breaker under concurrent host load (e.g. the eval scheduler).
func TestDefaultCrossEncoderBudgetCoversMeasuredLoadedLatency(t *testing.T) {
	if defaultCrossEncoderRerankTimeout < 2*time.Second {
		t.Fatalf("cross-encoder budget %s is below the measured loaded latency (2s)", defaultCrossEncoderRerankTimeout)
	}
	if defaultCrossEncoderRerankTimeout >= defaultLLMRerankTimeout {
		t.Fatalf("cross-encoder budget %s must stay below the LLM fallback budget %s", defaultCrossEncoderRerankTimeout, defaultLLMRerankTimeout)
	}
	if defaultCrossEncoderRerankTimeout+defaultResponseCushion >= defaultQueryTimeout {
		t.Fatalf("cross-encoder budget %s + response cushion %s must fit the query budget %s", defaultCrossEncoderRerankTimeout, defaultResponseCushion, defaultQueryTimeout)
	}
}

type probeReranker struct{}

func (probeReranker) Available(context.Context) bool { return true }
func (probeReranker) Rerank(_ context.Context, _ string, hits []*routingv1.SearchHit) ([]*routingv1.SearchHit, error) {
	return hits, nil
}

func TestRerankInsufficientBudgetDoesNotStrandHalfOpenProbe(t *testing.T) {
	now := time.Now()
	r := NewRouter(Deps{Reranker: probeReranker{}, Now: func() time.Time { return now }, RerankBreaker: RerankBreakerConfig{FailureThreshold: 1, Cooldown: time.Second}})
	r.rerankBreaker.recordFailure(now)
	now = now.Add(2 * time.Second)
	groups := []*routingv1.ProviderResultGroup{{Hits: []*routingv1.SearchHit{{Id: "a"}}}, {Hits: []*routingv1.SearchHit{{Id: "b"}}}}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, _, degraded, reason, _, _ := r.maybeRerank(ctx, "q", groups, nil)
	if !degraded || reason != "reranker_budget_exhausted" {
		t.Fatalf("unexpected short budget result: %t %s", degraded, reason)
	}
	_, reranked, degraded, reason, _, _ := r.maybeRerank(context.Background(), "q", groups, nil)
	if !reranked || degraded {
		t.Fatalf("healthy probe stranded: reranked=%t degraded=%t reason=%s", reranked, degraded, reason)
	}
}

func TestCallerPartitionIncludesForwardedVerifiedToken(t *testing.T) {
	var contexts []context.Context
	verifier := provenance.VerifierFunc(func(string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: "run", Subject: "same"}}, nil
	})
	handler := provenance.Middleware(verifier)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { contexts = append(contexts, r.Context()) }))
	for _, token := range []string{"secret-a", "secret-b", "secret-a"} {
		req := httptest.NewRequest("POST", "/query", nil)
		req.Header.Set(cliutil.HeaderAgentIdentityToken, token)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}
	a, b, again := callerPartition(contexts[0]), callerPartition(contexts[1]), callerPartition(contexts[2])
	if a == b || a != again || strings.Contains(a, "secret") {
		t.Fatal("caller partition did not separate verified token identities")
	}
}

type partitionPairVerifier struct{}

func (partitionPairVerifier) Verify(string) (*cliutil.VerifyResult, error) {
	return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: "run", Subject: "same"}}, nil
}
func (partitionPairVerifier) VerifyPair(_ context.Context, _, _ string) (*cliutil.VerifyPairResult, error) {
	grant := identity.OperationGrant{ID: "grant", Approver: "owner", ExpiresAt: time.Now().Add(time.Hour)}
	b, _ := json.Marshal(grant)
	return &cliutil.VerifyPairResult{Valid: true, Operation: &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: "run", Meta: map[string]string{"operation_grant": string(b)}}}}, nil
}
func TestCallerPartitionIncludesForwardedOperationProof(t *testing.T) {
	var contexts []context.Context
	handler := provenance.Middleware(partitionPairVerifier{})(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { contexts = append(contexts, r.Context()) }))
	for _, token := range []string{"operation-a", "operation-b", "operation-a"} {
		req := httptest.NewRequest("POST", "/query", nil)
		req.Header.Set(cliutil.HeaderAgentIdentityToken, "same-base")
		req.Header.Set(cliutil.HeaderAgentOperationToken, token)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}
	a, b, again := callerPartition(contexts[0]), callerPartition(contexts[1]), callerPartition(contexts[2])
	if a == b || a != again || strings.Contains(a, "operation-") {
		t.Fatal("operation proof was not partitioned safely")
	}
}
