package retrieval

import (
	"context"
	"fmt"
	"os"
	"strings"

	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/api-core/storage"
	"signal-inbox/internal/inference"
)

// QdrantSemanticSearch keeps semantic recall behind the shared ai-go vector
// store. Embeddings are requested exclusively through Signal Inbox's
// ai-gateway boundary, never through ai-go's provider subprocess default.
type QdrantSemanticSearch struct {
	client     inference.Client
	store      aisearch.VectorStore
	collection string
}

func NewQdrantSemanticSearch(client inference.Client) *QdrantSemanticSearch {
	if client == nil {
		return nil
	}
	collection, err := storage.Collection("signals")
	if err != nil {
		return nil
	}
	return &QdrantSemanticSearch{
		client:     client,
		store:      aisearch.NewVectorStore(strings.TrimSpace(os.Getenv("QDRANT_URL")), "", collection),
		collection: collection,
	}
}

func NewSemanticSearchForStore(client inference.Client, store aisearch.VectorStore) *QdrantSemanticSearch {
	if client == nil || store == nil {
		return nil
	}
	return &QdrantSemanticSearch{client: client, store: store, collection: "test-signals"}
}

func (s *QdrantSemanticSearch) Search(ctx context.Context, query string, corpus []Result, limit int) ([]SemanticHit, error) {
	if s == nil || s.client == nil || s.store == nil {
		return nil, fmt.Errorf("semantic retrieval is unavailable")
	}
	if len(corpus) == 0 {
		return []SemanticHit{}, nil
	}
	embedder := inference.Embedder{Client: s.client}
	for index, item := range corpus {
		vector, err := embedder.EmbedDocument(ctx, semanticDocument(item))
		if err != nil {
			return nil, fmt.Errorf("embed journal signal %s: %w", item.Signal.ID, err)
		}
		if index == 0 {
			if err := s.store.EnsureCollection(ctx, aisearch.CollectionSpec{
				Name:          s.collection,
				DenseSize:     len(vector),
				DenseDistance: aisearch.DefaultDenseDistance,
				Model:         "ai-gateway",
				Role:          inference.EmbeddingRole,
			}); err != nil {
				return nil, fmt.Errorf("ensure semantic collection: %w", err)
			}
		}
		if err := s.store.Upsert(ctx, aisearch.Point{ID: item.Signal.ID, Dense: vector, Payload: map[string]any{
			"signal_id": item.Signal.ID,
			"body":      semanticDocument(item),
		}}); err != nil {
			return nil, fmt.Errorf("index journal signal %s: %w", item.Signal.ID, err)
		}
	}
	queryVector, err := embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed semantic query: %w", err)
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	searchResults, err := queryVectorStore(ctx, s.store, aisearch.HybridQuery{Dense: queryVector, Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("query semantic index: %w", err)
	}
	output := make([]SemanticHit, 0, len(searchResults))
	for _, hit := range searchResults {
		id, _ := hit.Payload["signal_id"].(string)
		if id == "" {
			id = hit.ID
		}
		output = append(output, SemanticHit{SignalID: id, Score: hit.Score})
	}
	return output, nil
}

// queryVectorStore adapts ai-go's materialized vector results to the retrieval
// domain. VectorStore.Query returns an owned slice, not a database cursor.
func queryVectorStore(ctx context.Context, store aisearch.VectorStore, query aisearch.HybridQuery) ([]aisearch.SearchResult, error) {
	return store.Query(ctx, query)
}

func semanticDocument(item Result) string {
	return strings.TrimSpace(item.Signal.SourceURL + "\n" + item.Signal.CaptureNote + "\n" + item.Signal.ExtractedContent)
}

var _ SemanticSearch = (*QdrantSemanticSearch)(nil)
