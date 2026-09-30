package retrieval

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

type blockingLexical struct {
	mu      sync.Mutex
	calls   int
	started chan struct{}
	release chan struct{}
}

func (leg *blockingLexical) SearchLexical(ctx context.Context, query Query) ([]Candidate, error) {
	leg.mu.Lock()
	leg.calls++
	if leg.calls == 1 {
		close(leg.started)
	}
	leg.mu.Unlock()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-leg.release:
		return []Candidate{{ID: "shared", Generation: query.Generation}}, nil
	}
}

type fakeLeg struct {
	results []Candidate
	err     error
	delay   time.Duration
	calls   int
}

func (leg *fakeLeg) search(ctx context.Context) ([]Candidate, error) {
	leg.calls++
	if leg.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(leg.delay):
		}
	}
	return append([]Candidate(nil), leg.results...), leg.err
}

type fakeLexical struct{ *fakeLeg }

func (leg fakeLexical) SearchLexical(ctx context.Context, _ Query) ([]Candidate, error) {
	return leg.search(ctx)
}

type fakeSemantic struct{ *fakeLeg }

func (leg fakeSemantic) SearchSemantic(ctx context.Context, _ Query) ([]Candidate, error) {
	return leg.search(ctx)
}

type mappedLeg struct {
	results map[string][]Candidate
	calls   int
}

func (leg *mappedLeg) search(query Query) ([]Candidate, error) {
	leg.calls++
	return append([]Candidate(nil), leg.results[query.Text]...), nil
}

type mappedLexical struct{ leg *mappedLeg }

func (searcher mappedLexical) SearchLexical(_ context.Context, query Query) ([]Candidate, error) {
	return searcher.leg.search(query)
}

type mappedSemantic struct{ leg *mappedLeg }

func (searcher mappedSemantic) SearchSemantic(_ context.Context, query Query) ([]Candidate, error) {
	return searcher.leg.search(query)
}

type fakeReranker struct {
	calls int
	err   error
}

type cardEmbedder struct{ dimensions int }

func (embedder cardEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))
	for index := range vectors {
		vectors[index] = make([]float32, embedder.dimensions)
		vectors[index][index%embedder.dimensions] = 1
	}
	return vectors, nil
}

type cardStore struct{ records []VectorRecord }

func (store *cardStore) Upsert(_ context.Context, records []VectorRecord) error {
	store.records = append(store.records, records...)
	return nil
}
func (store *cardStore) Delete(context.Context, []string) error { return nil }
func (store *cardStore) Query(context.Context, []float32, Query) ([]Candidate, error) {
	return nil, nil
}

type cardAdmission struct {
	acquired int
	released int
}

type fakeGraph struct {
	results []Candidate
	err     error
	calls   int
}

func (graph *fakeGraph) Expand(context.Context, string, string, []string, int) ([]Candidate, error) {
	graph.calls++
	return append([]Candidate(nil), graph.results...), graph.err
}

func (admission *cardAdmission) Acquire(_ context.Context, _ string, weight int) (func(), error) {
	admission.acquired += weight
	return func() { admission.released += weight }, nil
}

func (reranker *fakeReranker) Rerank(_ context.Context, _ Query, candidates []Candidate) ([]Candidate, error) {
	reranker.calls++
	if reranker.err != nil {
		return nil, reranker.err
	}
	for left, right := 0, len(candidates)-1; left < right; left, right = left+1, right-1 {
		candidates[left], candidates[right] = candidates[right], candidates[left]
	}
	return candidates, nil
}

func TestHybridEngineRunsLegsConcurrentlyAndExplainsFusion(t *testing.T) {
	lexical := &fakeLeg{delay: 40 * time.Millisecond, results: []Candidate{{ID: "lexical", Role: "implementation", Authority: "authoritative"}, {ID: "shared", Role: "implementation", Authority: "authoritative"}}}
	semantic := &fakeLeg{delay: 40 * time.Millisecond, results: []Candidate{{ID: "shared", Role: "implementation", Authority: "authoritative"}, {ID: "semantic", Role: "implementation", Authority: "authoritative"}}}
	started := time.Now()
	response, err := (HybridEngine{Lexical: fakeLexical{lexical}, Semantic: fakeSemantic{semantic}}).Search(context.Background(), Query{Text: "find request routing implementation", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed >= 75*time.Millisecond {
		t.Fatalf("retrieval legs did not overlap: %s", elapsed)
	}
	if len(response.Results) != 3 || response.Results[0].ID != "shared" || len(response.Results[0].RankEvidence) != 2 || response.Results[0].ScoreFactors["rrf"] == 0 {
		t.Fatalf("shared fusion evidence mismatch: %+v", response)
	}
}

func TestHybridEngineFinalCacheIsGenerationKeyedAndDefensive(t *testing.T) {
	lexical := &fakeLeg{results: []Candidate{{ID: "cached", ScoreFactors: map[string]float64{"lexical": 1}}}}
	engine := HybridEngine{Lexical: fakeLexical{lexical}, Cache: NewResultCache(2, 4096)}
	query := Query{Text: "Symbol", Generation: "g1", Limit: 5}
	first, err := engine.Search(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	first.Results[0].ID = "mutated"
	first.Results[0].ScoreFactors["lexical"] = 0
	second, err := engine.Search(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if lexical.calls != 1 || second.Results[0].ID != "cached" || second.Results[0].ScoreFactors["lexical"] != 1 {
		t.Fatalf("cache was not generation-keyed/defensive: calls=%d response=%+v", lexical.calls, second)
	}
	query.Generation = "g2"
	if _, err := engine.Search(context.Background(), query); err != nil {
		t.Fatal(err)
	}
	if lexical.calls != 2 {
		t.Fatalf("new generation reused old result: calls=%d", lexical.calls)
	}
}

func TestHybridEngineCoalescesIdenticalGenerationQueries(t *testing.T) {
	lexical := &blockingLexical{started: make(chan struct{}), release: make(chan struct{})}
	engine := HybridEngine{Lexical: lexical, Flights: NewQueryFlights()}
	query := Query{Text: "Symbol", Generation: "g1", Limit: 5}
	results := make(chan HybridResponse, 2)
	for range 2 {
		go func() {
			response, _ := engine.Search(context.Background(), query)
			results <- response
		}()
	}
	<-lexical.started
	time.Sleep(10 * time.Millisecond)
	close(lexical.release)
	for range 2 {
		if response := <-results; len(response.Results) != 1 {
			t.Fatalf("coalesced response = %+v", response)
		}
	}
	lexical.mu.Lock()
	defer lexical.mu.Unlock()
	if lexical.calls != 1 {
		t.Fatalf("lexical calls = %d, want 1", lexical.calls)
	}
}

func TestHybridEngineDegradesTruthfullyAndExactSkipsAI(t *testing.T) {
	lexical := &fakeLeg{results: []Candidate{{ID: "exact", Role: "implementation", Authority: "authoritative"}}}
	semantic := &fakeLeg{err: errors.New("qdrant unavailable")}
	reranker := &fakeReranker{err: errors.New("reranker unavailable")}
	engine := HybridEngine{Lexical: fakeLexical{lexical}, Semantic: fakeSemantic{semantic}, Reranker: reranker}
	degraded, err := engine.Search(context.Background(), Query{Text: "find persistence implementation", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(degraded.Degraded, []string{"semantic"}) || len(degraded.Results) != 1 {
		t.Fatalf("lexical degradation mismatch: %+v", degraded)
	}
	exact, err := engine.Search(context.Background(), Query{Text: "Service.Search", Limit: 5})
	if err != nil || len(exact.Results) != 1 {
		t.Fatalf("exact search failed: %+v err=%v", exact, err)
	}
	if semantic.calls != 1 || reranker.calls != 0 {
		t.Fatalf("exact search invoked optional AI: semantic=%d reranker=%d", semantic.calls, reranker.calls)
	}
}

func TestHybridEngineNamesRerankerFailure(t *testing.T) {
	lexical := &fakeLeg{results: []Candidate{{ID: "one"}, {ID: "two"}}}
	semantic := &fakeLeg{results: []Candidate{{ID: "two"}, {ID: "three"}}}
	reranker := &fakeReranker{err: errors.New("model capacity exhausted")}
	response, err := (HybridEngine{Lexical: fakeLexical{lexical}, Semantic: fakeSemantic{semantic}, Reranker: reranker}).Search(context.Background(), Query{Text: "natural language request flow", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(response.Degraded, []string{"reranker"}) || len(response.Results) != 3 {
		t.Fatalf("reranker degradation mismatch: %+v", response)
	}
}

func TestHybridEngineExpandsCurrentGraphProofAndNamesGraphDegradation(t *testing.T) {
	lexical := &fakeLeg{results: []Candidate{{ID: "service", Title: "Service.Search", Generation: "g1"}}}
	semantic := &fakeLeg{results: []Candidate{{ID: "service", Title: "Service.Search", Generation: "g1"}}}
	graph := &fakeGraph{results: []Candidate{{ID: "edge:handler", Title: "Handler.Search", Evidence: "current_source_hash"}}}
	response, err := (HybridEngine{Lexical: fakeLexical{lexical}, Semantic: fakeSemantic{semantic}, Graph: graph}).Search(context.Background(), Query{Text: "who calls Service Search", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if graph.calls != 1 || len(response.Results) != 2 || response.Results[1].Proof != "analyzer_confirmed" || response.Results[1].Regime != RegimeRelationship {
		t.Fatalf("graph expansion mismatch: %+v", response)
	}
	withoutGraph, err := (HybridEngine{Lexical: fakeLexical{lexical}, Semantic: fakeSemantic{semantic}}).Search(context.Background(), Query{Text: "who calls Service Search", Limit: 5})
	if err != nil || !reflect.DeepEqual(withoutGraph.Degraded, []string{"graph"}) {
		t.Fatalf("graph degradation mismatch: %+v err=%v", withoutGraph, err)
	}
}

func TestHybridDegradedMatrixPreservesLexicalResults(t *testing.T) {
	for _, stage := range []string{"embedder", "vector_store"} {
		t.Run(stage, func(t *testing.T) {
			lexical := &fakeLeg{results: []Candidate{{ID: "lexical"}}}
			semantic := &fakeLeg{err: errors.New(stage + " unavailable")}
			response, err := (HybridEngine{Lexical: fakeLexical{lexical}, Semantic: fakeSemantic{semantic}}).Search(context.Background(), Query{Text: "natural implementation lookup", Limit: 5})
			if err != nil || len(response.Results) != 1 || !reflect.DeepEqual(response.Degraded, []string{"semantic"}) {
				t.Fatalf("%s degradation lost lexical service: %+v err=%v", stage, response, err)
			}
		})
	}
}

func TestHybridEvaluationMeetsRecallMRRAndDoesNotRegressExact(t *testing.T) {
	cases := []struct{ query, expected string }{
		{"where provider demotion state is persisted", "demotion"},
		{"incrementally reconcile vector search documents", "reconciler"},
		{"Code Facts Search RPC request response", "contract"},
		{"secure request router implementation", "router"},
		{"cache quota garbage collection", "cache"},
	}
	lexical := &mappedLeg{results: map[string][]Candidate{
		cases[0].query:   {{ID: "noise-a"}, {ID: "demotion"}, {ID: "noise-b"}},
		cases[1].query:   {{ID: "noise-a"}, {ID: "reconciler"}},
		cases[2].query:   {{ID: "contract"}, {ID: "noise-c"}},
		cases[3].query:   {{ID: "noise-b"}, {ID: "router"}},
		cases[4].query:   {{ID: "cache"}},
		"Service.Search": {{ID: "exact-service"}},
	}}
	semantic := &mappedLeg{results: map[string][]Candidate{
		cases[0].query: {{ID: "demotion"}, {ID: "noise-a"}},
		cases[1].query: {{ID: "reconciler"}, {ID: "noise-b"}},
		cases[2].query: {{ID: "noise-c"}, {ID: "contract"}},
		cases[3].query: {{ID: "router"}, {ID: "noise-a"}},
		cases[4].query: {{ID: "cache"}, {ID: "noise-b"}},
	}}
	engine := HybridEngine{Lexical: mappedLexical{lexical}, Semantic: mappedSemantic{semantic}}
	hits, reciprocalRank := 0, 0.0
	for _, testCase := range cases {
		response, err := engine.Search(context.Background(), Query{Text: testCase.query, Limit: 5})
		if err != nil {
			t.Fatal(err)
		}
		rank := 0
		for index, candidate := range response.Results {
			if candidate.ID == testCase.expected {
				rank = index + 1
				break
			}
		}
		t.Logf("query=%q expected=%s rank=%d results=%v", testCase.query, testCase.expected, rank, candidateIDs(response.Results))
		if rank > 0 && rank <= 5 {
			hits++
		}
		if rank > 0 && rank <= 3 {
			reciprocalRank += 1 / float64(rank)
		}
	}
	if recall := float64(hits) / float64(len(cases)); recall < 0.95 {
		t.Fatalf("hybrid recall@5 %.2f below 0.95", recall)
	}
	if mrr := reciprocalRank / float64(len(cases)); mrr < 0.85 {
		t.Fatalf("hybrid MRR@3 %.2f below 0.85", mrr)
	}
	beforeSemantic := semantic.calls
	exact, err := engine.Search(context.Background(), Query{Text: "Service.Search", Limit: 5})
	if err != nil || len(exact.Results) != 1 || exact.Results[0].ID != "exact-service" || semantic.calls != beforeSemantic {
		t.Fatalf("exact category regressed or invoked semantic search: %+v err=%v", exact, err)
	}
}

func TestCardPolicyIsSelectiveStableAndStorageBounded(t *testing.T) {
	policy := DefaultCardPolicy()
	extractor := CardExtractor{Policy: policy}
	documents := []Document{
		{ID: "symbol:search", SourceFileID: "file:service", SourceHash: "sha256:service", Path: "service.go", Language: "go", Role: "implementation", Scope: "scenario:code-facts", Kind: "symbol", Title: "Service.Search", Body: "Search returns current implementation evidence.", Aliases: []string{"code search"}},
		{ID: "fixture:noise", SourceFileID: "file:fixture", SourceHash: "sha256:fixture", Path: "testdata/noise.json", Language: "json", Role: "fixture", Scope: "scenario:code-facts", Kind: "file", Title: "noise", Body: "must not embed"},
		{ID: "alias:search", SourceFileID: "file:proto", SourceHash: "sha256:proto", Path: "facts.proto", Language: "protobuf", Role: "generated_alias", Scope: "scenario:code-facts", Kind: "contract", Title: "Search client alias", Body: "Points to the authoritative Search RPC."},
	}
	first, stats, err := extractor.Extract("g1", documents)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := extractor.Extract("g1", documents)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || stats.RejectedIneligible != 1 || first[0].EmbeddingHash != second[0].EmbeddingHash {
		t.Fatalf("selective stable card policy mismatch: cards=%+v stats=%+v", first, stats)
	}
	if first[0].DisplayText == first[0].EmbeddingText || stats.ByRole["fixture"] != 0 || stats.ByKind["contract"] != 1 {
		t.Fatalf("card display/embedding or measurements mismatch: cards=%+v stats=%+v", first, stats)
	}
	projected := stats.EstimatedBytes * 80000 / int64(stats.Total)
	if projected >= 1536*1024*1024 {
		t.Fatalf("80k-card storage projection %d exceeds 1.5 GiB", projected)
	}
}

func TestCardIndexerBatchesEmbeddingsAndPersistsStablePayload(t *testing.T) {
	policy := DefaultCardPolicy()
	policy.Dimensions = 4
	documents := []Document{
		{ID: "one", SourceFileID: "file:one", SourceHash: "sha256:one", Path: "one.go", Language: "go", Role: "implementation", Scope: "repo", Kind: "symbol", Title: "One", Body: "first card"},
		{ID: "two", SourceFileID: "file:two", SourceHash: "sha256:two", Path: "two.go", Language: "go", Role: "implementation", Scope: "repo", Kind: "symbol", Title: "Two", Body: "second card"},
		{ID: "three", SourceFileID: "file:three", SourceHash: "sha256:three", Path: "three.go", Language: "go", Role: "implementation", Scope: "repo", Kind: "symbol", Title: "Three", Body: "third card"},
	}
	cards, _, err := (CardExtractor{Policy: policy}).Extract("g1", documents)
	if err != nil {
		t.Fatal(err)
	}
	store := &cardStore{}
	admission := &cardAdmission{}
	count, err := (CardIndexer{Embedder: cardEmbedder{dimensions: 4}, Store: store, Admission: admission, BatchSize: 2}).Index(context.Background(), cards)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 || len(store.records) != 3 || admission.acquired != 3 || admission.released != 3 {
		t.Fatalf("bounded card indexing mismatch: count=%d records=%d admission=%+v", count, len(store.records), admission)
	}
	for _, record := range store.records {
		if record.ID == "" || record.Generation != "g1" || record.SourceHash == "" || record.Payload["embedding_hash"] == "" || record.Payload["display_text"] == "" {
			t.Fatalf("card vector lost identity or display evidence: %+v", record)
		}
	}
}

func TestModelBakeoffPinsCardPolicyToMeasuredProfile(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("testdata", "model-bakeoff-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Version  string `json:"version"`
		Model    string `json:"model"`
		Cases    int    `json:"cases"`
		Profiles []struct {
			Dimensions int     `json:"dimensions"`
			RecallAt5  float64 `json:"recall_at_5"`
			MRRAt3     float64 `json:"mrr_at_3"`
		} `json:"profiles"`
		Selected struct {
			Dimensions int    `json:"dimensions"`
			Storage    string `json:"storage"`
		} `json:"selected"`
	}
	if err := json.Unmarshal(payload, &report); err != nil {
		t.Fatal(err)
	}
	policy := DefaultCardPolicy()
	if report.Version != "code-facts-embedding-bakeoff-v1" || report.Model != policy.Model || report.Selected.Dimensions != policy.Dimensions || report.Selected.Storage != policy.Storage || report.Cases < 5 {
		t.Fatalf("card policy drifted from bake-off: report=%+v policy=%+v", report, policy)
	}
	for _, profile := range report.Profiles {
		if profile.RecallAt5 < 0.95 || profile.MRRAt3 < 0.85 {
			t.Fatalf("unevaluated embedding profile selected: %+v", profile)
		}
	}
}
