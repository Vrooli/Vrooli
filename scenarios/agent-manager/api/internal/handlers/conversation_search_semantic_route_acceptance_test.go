package handlers

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/conversationsearch"
	"connectrpc.com/connect"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
	coredb "github.com/vrooli/api-core/database"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain/domainconnect"
	_ "modernc.org/sqlite"
)

type generatedSemanticRouteCorpus struct {
	relevant   map[string][]conversationsearch.Document
	distractor map[string][]conversationsearch.Document
}

type generatedLabelEmbedder struct{ dimensions map[string]int }

func (embedder generatedLabelEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	return embedder.EmbedVector(text), nil
}

func (embedder generatedLabelEmbedder) EmbedVector(text string) []float64 {
	vector := make([]float64, 10)
	for _, token := range strings.Fields(strings.ToLower(text)) {
		if dimension, ok := embedder.dimensions[token]; ok {
			vector[dimension] = 1
		}
	}
	return vector
}

func generatedSemanticEmbedder() generatedLabelEmbedder {
	return generatedLabelEmbedder{dimensions: map[string]int{
		"portfolio": 0, "investment": 0, "outcome": 1, "result": 1,
		"retention": 2, "renewal": 2, "cadence": 3, "schedule": 3,
		"risk": 4, "hazard": 4, "mitigation": 5, "reduction": 5,
		"partner": 6, "alliance": 6, "pipeline": 7, "funnel": 7,
		"target": 8, "destination": 8, "receipt": 9, "acknowledgement": 9,
	}}
}

func (generatedLabelEmbedder) Available(context.Context) bool { return true }

// generatedDenseStore retains only temporary generated vectors and performs
// cosine ranking from the production ai-go search request. It replaces the
// prior query-to-known-hit oracle while avoiding Ollama, Qdrant, and live data.
type generatedDenseStore struct {
	mu     sync.RWMutex
	points map[string]aisearch.Point
}

func newGeneratedDenseStore() *generatedDenseStore {
	return &generatedDenseStore{points: make(map[string]aisearch.Point)}
}

func (store *generatedDenseStore) EnsureCollection(context.Context, aisearch.CollectionSpec) error {
	return nil
}
func (store *generatedDenseStore) Upsert(_ context.Context, point aisearch.Point) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	point.Dense = append([]float64(nil), point.Dense...)
	store.points[point.ID] = point
	return nil
}
func (store *generatedDenseStore) SetPayload(_ context.Context, id string, payload map[string]any) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	point, ok := store.points[id]
	if !ok {
		return fmt.Errorf("generated dense point %q not found", id)
	}
	point.Payload = payload
	store.points[id] = point
	return nil
}
func (store *generatedDenseStore) BatchDelete(_ context.Context, ids []string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	for _, id := range ids {
		delete(store.points, id)
	}
	return nil
}
func (store *generatedDenseStore) Query(_ context.Context, query aisearch.HybridQuery) ([]aisearch.SearchResult, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	results := make([]aisearch.SearchResult, 0, len(store.points))
	for id, point := range store.points {
		if !generatedFilterMatches(point.Payload, query.Filter) {
			continue
		}
		results = append(results, aisearch.SearchResult{ID: id, Score: generatedCosine(query.Dense, point.Dense), Payload: point.Payload})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].ID < results[j].ID
		}
		return results[i].Score > results[j].Score
	})
	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}
	return results, nil
}
func (store *generatedDenseStore) CountPoints(context.Context) (int, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return len(store.points), nil
}
func (store *generatedDenseStore) ScrollIDs(context.Context) (map[string]aisearch.ScrollItem, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	items := make(map[string]aisearch.ScrollItem, len(store.points))
	for id, point := range store.points {
		sourceID, _ := point.Payload["source_id"].(string)
		items[id] = aisearch.ScrollItem{SourceID: sourceID}
	}
	return items, nil
}
func (*generatedDenseStore) Available(context.Context) bool { return true }

func generatedFilterMatches(payload map[string]any, filter *aisearch.QueryFilter) bool {
	if filter == nil {
		return true
	}
	for _, match := range filter.Must {
		actual, exists := payload[match.Key]
		if !exists {
			return false
		}
		if len(match.AnyOf) > 0 {
			found := false
			for _, expected := range match.AnyOf {
				if fmt.Sprint(actual) == fmt.Sprint(expected) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		} else if fmt.Sprint(actual) != fmt.Sprint(match.Value) {
			return false
		}
	}
	for _, span := range filter.Ranges {
		actual, ok := payload[span.Key].(float64)
		if !ok || (span.GT != nil && actual <= *span.GT) || (span.GTE != nil && actual < *span.GTE) || (span.LT != nil && actual >= *span.LT) || (span.LTE != nil && actual > *span.LTE) {
			return false
		}
	}
	return true
}

func generatedCosine(left, right []float64) float64 {
	if len(left) == 0 || len(left) != len(right) {
		return 0
	}
	var dot, leftNorm, rightNorm float64
	for i := range left {
		dot += left[i] * right[i]
		leftNorm += left[i] * left[i]
		rightNorm += right[i] * right[i]
	}
	if leftNorm == 0 || rightNorm == 0 {
		return 0
	}
	return dot / (math.Sqrt(leftNorm) * math.Sqrt(rightNorm))
}

type generatedSemanticCoverage struct{ store *generatedDenseStore }

func (generatedSemanticCoverage) Rebuild(context.Context, string) error { return nil }
func (coverage generatedSemanticCoverage) SemanticStatus(ctx context.Context) (uint64, string, string, string, error) {
	documents, err := coverage.store.CountPoints(ctx)
	return uint64(documents), "generated-only", "generated-layout-v1", "deterministic-generated-embedder", err
}

type emptySemanticCoverageSource struct{}

func (emptySemanticCoverageSource) LoadSourcePage(context.Context, *conversationsearch.SourceCursor, int) (conversationsearch.SourcePage, error) {
	return conversationsearch.SourcePage{}, nil
}

func TestConversationSearchSemanticRouteGeneratedCoverageAndRecallAtTen(t *testing.T) {
	ctx := context.Background()
	db, err := sqlx.Connect("sqlite", "file:generated-semantic-route-acceptance?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, coredb.EnsureSchemas(ctx, db, coredb.SchemaProviderFunc(conversationsearch.Schema)))
	repository := conversationsearch.NewSQLiteRepository(db)
	denseStore := newGeneratedDenseStore()
	corpus := &generatedSemanticRouteCorpus{
		relevant: make(map[string][]conversationsearch.Document), distractor: make(map[string][]conversationsearch.Document),
	}
	harnesses := []string{"claude", "codex", "grok", "opencode", "gemini-cli", "cursor"}
	labels := []struct {
		corpus string
		query  string
	}{
		{corpus: "portfolio outcome", query: "investment result"},
		{corpus: "retention cadence", query: "customer renewal schedule"},
		{corpus: "risk mitigation", query: "hazard reduction"},
		{corpus: "partner pipeline", query: "alliance sales funnel"},
		{corpus: "target receipt", query: "destination acknowledgement"},
	}
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	total := 0
	for _, labelPair := range labels {
		label := labelPair.corpus
		for _, harness := range harnesses {
			key := label + "/" + harness
			for n := 0; n < 5; n++ {
				document := generatedSemanticDocument(fmt.Sprintf("relevant:%s:%s:%d", label, harness, n), label+" generated labeled evidence", harness, n, now)
				require.NoError(t, repository.UpsertDocument(ctx, document))
				require.NoError(t, upsertGeneratedDenseDocument(ctx, denseStore, document))
				corpus.relevant[key] = append(corpus.relevant[key], document)
				total++
			}
			for n := 0; n < 6; n++ {
				// Hard negatives share one label token; the paraphrased query's
				// second concept dimension is what should rank true matches first.
				background := "unrelated generated background note"
				if n%2 == 0 {
					words := strings.Fields(label)
					background = words[(n/2)%len(words)] + " generated partial-match background note"
				}
				document := generatedSemanticDocument(fmt.Sprintf("distractor:%s:%s:%d", label, harness, n), background, harness, n+5, now)
				require.NoError(t, repository.UpsertDocument(ctx, document))
				require.NoError(t, upsertGeneratedDenseDocument(ctx, denseStore, document))
				corpus.distractor[key] = append(corpus.distractor[key], document)
				total++
			}
		}
	}
	require.Equal(t, len(labels)*len(harnesses)*11, total)
	denseEngine := aisearch.NewService(aisearch.ServiceOptions{
		Embedder: generatedSemanticEmbedder(), VectorStore: denseStore,
		ApplyFloor: true, DenseBudget: 250 * time.Millisecond, MaxLimit: 100,
		Project: func(result aisearch.SearchResult) aisearch.SearchResult {
			result.SourceID, _ = result.Payload["source_id"].(string)
			return result
		},
	})
	retriever, err := conversationsearch.NewSharedSemanticRetriever(denseEngine, repository, aisearch.NewWeightedAdmission(2))
	require.NoError(t, err)
	service, err := conversationsearch.NewService(repository, repository, repository, []byte("0123456789abcdef0123456789abcdef"), conversationsearch.WithSemanticRetriever(retriever))
	require.NoError(t, err)
	indexer, err := conversationsearch.NewIndexer(conversationsearch.IndexerOptions{
		Source: emptySemanticCoverageSource{}, Repository: repository,
		Semantic: generatedSemanticCoverage{store: denseStore},
	})
	require.NoError(t, err)
	adapter := NewConversationSearchAdapter(service, indexer)
	path, route := domainconnect.NewConversationSearchServiceHandler(NewConversationSearchConnectHandler(adapter))
	mux := http.NewServeMux()
	mux.Handle(path, route)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := domainconnect.NewConversationSearchServiceClient(server.Client(), server.URL, connect.WithProtoJSON())

	// Indexer status is deliberately asynchronous in production. Wait for its
	// first consistent generated snapshot before evaluating coverage.
	deadline := time.Now().Add(time.Second)
	for {
		status, statusErr := indexer.StatusSnapshot(ctx)
		require.NoError(t, statusErr)
		if status.SemanticDocuments == uint64(total) && status.SemanticEligibleDocuments == uint64(total) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("generated semantic index coverage did not settle: %+v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}

	queries, returned, relevant := 0, 0, 0
	latencies := make([]time.Duration, 0, len(labels)*len(harnesses))
	started := time.Now()
	for _, labelPair := range labels {
		label := labelPair.corpus
		for _, harness := range harnesses {
			queryStarted := time.Now()
			response, searchErr := client.SearchConversations(ctx, connect.NewRequest(&domainpb.SearchConversationsRequest{
				Query: labelPair.query, Mode: domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_SEMANTIC,
				Filters: &domainpb.ConversationSearchFilters{Harnesses: []string{harness}}, PageSize: 10,
			}))
			latencies = append(latencies, time.Since(queryStarted))
			require.NoError(t, searchErr)
			responseMessage := response.Msg
			require.Equal(t, uint64(total), responseMessage.GetCoverage().GetSemanticDocuments())
			require.InDelta(t, 1, responseMessage.GetCoverage().GetSemanticRatio(), 1e-9)
			require.Len(t, responseMessage.GetHits(), 5)
			queryRelevant := 0
			for _, hit := range responseMessage.GetHits() {
				require.Equal(t, harness, hit.GetProvenance().GetHarness())
				hasDense := false
				for _, evidence := range hit.GetRankEvidence() {
					if evidence.GetLeg() == domainpb.ConversationSearchLeg_CONVERSATION_SEARCH_LEG_DENSE {
						hasDense = true
					}
				}
				require.True(t, hasDense, "dense leg evidence must survive adapter mapping")
				for _, want := range corpus.relevant[label+"/"+harness] {
					if hit.GetStableHitId() == want.DocumentID {
						queryRelevant++
					}
				}
			}
			require.Equal(t, 5, queryRelevant)
			queries++
			returned += len(responseMessage.GetHits())
			relevant += queryRelevant
		}
	}
	elapsed := time.Since(started)
	require.Equal(t, len(labels)*len(harnesses), queries)
	require.Equal(t, queries*5, returned)
	require.InDelta(t, 1, float64(relevant)/float64(queries*5), 1e-9)
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p50 := latencies[(len(latencies)-1)/2]
	p95 := latencies[(len(latencies)*95+99)/100-1]
	require.Less(t, p95, 250*time.Millisecond, "Connect route p95 must fit inside the dense leg budget")
	t.Logf("production Agent Manager semantic Connect route: generated_docs=%d queries=%d hits=%d coverage=%d/%d recall@10=%.3f dense_budget=250ms route_p50=%s route_p95=%s total_elapsed=%s", total, queries, returned, total, total, float64(relevant)/float64(queries*5), p50, p95, elapsed)
}

func generatedSemanticDocument(id, content, harness string, sequence int, now time.Time) conversationsearch.Document {
	return conversationsearch.Document{
		DocumentID: id, SourceRunID: "generated-run-" + harness, SourceEventID: "event-" + id,
		SourceMessageID: "message-" + id, ChunkIndex: 0, ChunkTotal: 1, StartByte: 0, EndByte: len(content),
		EventSequence: int64(sequence + 1), Role: "assistant", OccurredAt: now.Add(time.Duration(sequence) * time.Second),
		Content: content, ContentClass: conversationsearch.ContentClassProse, SourceHash: "generated-source", ContentHash: "generated-content-" + id,
		RecipeVersion: "generated-v1", Harness: harness, SourceSessionID: "generated-session-" + harness,
		ProviderOrigin: harness, Importer: "generated-test", RunStatus: "complete", RunLabel: "generated", Visible: true, IndexedAt: now,
	}
}

func upsertGeneratedDenseDocument(ctx context.Context, store *generatedDenseStore, document conversationsearch.Document) error {
	return store.Upsert(ctx, aisearch.Point{
		ID: document.DocumentID, Dense: generatedSemanticEmbedder().EmbedVector(document.Content),
		Payload: map[string]any{
			"source_id": document.DocumentID, "harness": document.Harness,
			"content_class": int(document.ContentClass), "occurred_at_unix": float64(document.OccurredAt.Unix()),
		},
	})
}
