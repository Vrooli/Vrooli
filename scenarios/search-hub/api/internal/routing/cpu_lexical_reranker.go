package routing

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

const (
	cpuLexicalRerankerName   = "cpu-lexical:v1"
	CPULexicalCandidateLimit = 64
	maxCPULexicalQueryBytes  = 512
	maxCPULexicalTextBytes   = 4096
)

// CPULexicalReranker is a bounded, deterministic CPU-only fallback for
// cross-provider ranking. It is deliberately a lexical ranker, not a model:
// callers can see that distinction in its leg identity and degradation reason.
// The single-slot gate caps process-wide ranker concurrency at one CPU worker.
type CPULexicalReranker struct {
	gate chan struct{}
}

// Every instance shares this process-scoped slot so adding routers or chains
// cannot multiply the CPU budget.
var cpuLexicalProcessGate = make(chan struct{}, 1)

func NewCPULexicalReranker() *CPULexicalReranker {
	return &CPULexicalReranker{gate: cpuLexicalProcessGate}
}

func (r *CPULexicalReranker) Name() string { return cpuLexicalRerankerName }

func (r *CPULexicalReranker) Available(context.Context) bool { return r != nil }

func (r *CPULexicalReranker) Rerank(ctx context.Context, query string, candidates []*routingv1.SearchHit) ([]*routingv1.SearchHit, error) {
	if r == nil {
		return nil, fmt.Errorf("CPU lexical reranker unavailable")
	}
	if len(query) == 0 || len(query) > maxCPULexicalQueryBytes || len(tokenizeLexical(query)) == 0 {
		return nil, fmt.Errorf("CPU lexical reranker query must contain terms and be at most %d bytes", maxCPULexicalQueryBytes)
	}
	if len(candidates) == 0 || len(candidates) > CPULexicalCandidateLimit {
		return nil, fmt.Errorf("CPU lexical reranker candidate count %d is outside [1,%d]", len(candidates), CPULexicalCandidateLimit)
	}
	if r.gate == nil {
		return nil, fmt.Errorf("CPU lexical reranker gate unavailable")
	}
	select {
	case r.gate <- struct{}{}:
		defer func() { <-r.gate }()
	case <-ctx.Done():
		return nil, fmt.Errorf("CPU lexical reranker budget exhausted: %w", ctx.Err())
	}

	queryTerms := tokenizeLexical(query)
	docs := make([][]string, len(candidates))
	termDocs := make(map[string]int)
	totalLength := 0
	for i, hit := range candidates {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("CPU lexical reranker budget exhausted: %w", err)
		}
		if hit == nil {
			return nil, fmt.Errorf("CPU lexical candidate %d is nil", i)
		}
		text := rerankCandidateText(hit)
		if len(text) > maxCPULexicalTextBytes {
			return nil, fmt.Errorf("CPU lexical candidate %d exceeds %d bytes", i, maxCPULexicalTextBytes)
		}
		docs[i] = tokenizeLexical(text)
		totalLength += len(docs[i])
		seen := make(map[string]struct{}, len(docs[i]))
		for _, term := range docs[i] {
			seen[term] = struct{}{}
		}
		for term := range seen {
			termDocs[term]++
		}
	}
	avgLength := float64(totalLength) / float64(len(docs))
	if avgLength == 0 {
		avgLength = 1
	}

	type scoredHit struct {
		hit   *routingv1.SearchHit
		score float64
		order int
	}
	ranked := make([]scoredHit, len(candidates))
	for i, hit := range candidates {
		freq := make(map[string]int, len(docs[i]))
		for _, term := range docs[i] {
			freq[term]++
		}
		score := 0.0
		for _, term := range queryTerms {
			df := float64(termDocs[term])
			idf := math.Log(1 + (float64(len(docs))-df+0.5)/(df+0.5))
			tf := float64(freq[term])
			if tf == 0 {
				continue
			}
			lengthNorm := 1.2 * (1 - 0.75 + 0.75*float64(len(docs[i]))/avgLength)
			score += idf * (tf * 2.2) / (tf + lengthNorm)
		}
		// Squash BM25's positive range to [0,1], matching the shared score band.
		hit.RerankScore = score / (score + 1)
		ranked[i] = scoredHit{hit: hit, score: hit.GetRerankScore(), order: i}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		if ranked[i].hit.GetScore() != ranked[j].hit.GetScore() {
			return ranked[i].hit.GetScore() > ranked[j].hit.GetScore()
		}
		return ranked[i].order < ranked[j].order
	})
	out := make([]*routingv1.SearchHit, len(ranked))
	for i := range ranked {
		out[i] = ranked[i].hit
	}
	return out, nil
}

func tokenizeLexical(text string) []string {
	var terms []string
	var current strings.Builder
	flush := func() {
		if current.Len() > 0 {
			terms = append(terms, current.String())
			current.Reset()
		}
	}
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return terms
}
