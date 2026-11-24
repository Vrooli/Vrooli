package search

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/ai-go/search/searchtest"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search"
)

func TestUnavailableSemanticProvidesStableDegradedBehavior(t *testing.T) {
	semantic := UnavailableSemantic(nil).(unavailableSemantic)
	require.True(t, semantic.Enabled())
	require.False(t, semantic.Ready(context.Background()))
	require.ErrorContains(t, semantic.Reconcile(context.Background()), "semantic search is unavailable")
	_, err := semantic.SearchSemantic(context.Background(), aisearch.SearchQuery{})
	require.ErrorContains(t, err, "semantic search is unavailable")
}

func TestProjectionSourceUsesRequestRoutedPool(t *testing.T) {
	ctx := context.Background()
	primaryDSN, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "primary.db"), storage.SQLiteTuning{})
	require.NoError(t, err)
	routed, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: primaryDSN, MaxOpenConns: 1, MaxIdleConns: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _ = routed.Close() })
	initSchema := func(ctx context.Context, db *sql.DB) error {
		return apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema))
	}
	require.NoError(t, initSchema(ctx, routed.Primary()))
	routed.SetTestPoolInitializer(initSchema)
	testDSN, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "test.db"), storage.SQLiteTuning{})
	require.NoError(t, err)
	require.NoError(t, routed.InstallTestPool(ctx, testDSN, "semantic-source", time.Minute))

	insert := `INSERT INTO search_docs (id, kind, subject_id, title, subtitle, body, href, availability, badges_json, relations_json, updated_at) VALUES (?, 'asset', ?, ?, '', 'body', '/asset', 'available', '[]', '[]', 'now')`
	_, err = routed.ExecContext(ctx, insert, "primary-id", "primary", "Primary projection")
	require.NoError(t, err)
	_, err = routed.ExecContext(apidb.WithTestMode(ctx), insert, "test-id", "test", "Test projection")
	require.NoError(t, err)

	source := projectionSource{db: routed}
	primary, err := source.LoadAll(ctx)
	require.NoError(t, err)
	require.Len(t, primary, 1)
	require.Equal(t, "primary-id", primary[0].ID)
	testDocs, err := source.LoadAll(apidb.WithTestMode(ctx))
	require.NoError(t, err)
	require.Len(t, testDocs, 1)
	require.Equal(t, "test-id", testDocs[0].ID)
}

func TestSemanticReconcileRejectsMissingConfigurationAndGate(t *testing.T) {
	var unconfigured *semanticIndex
	require.ErrorContains(t, unconfigured.Reconcile(context.Background()), "not configured")

	index := &semanticIndex{reconcileFunc: func(context.Context) error { return nil }}
	require.ErrorContains(t, index.Reconcile(context.Background()), "gate is not configured")
}

func TestSemanticReconcileCancellationAndFailureDegradeUntilRecovery(t *testing.T) {
	index := &semanticIndex{
		reconcileGate:  make(chan struct{}, 1),
		reconcileFunc:  func(context.Context) error { return nil },
		readinessProbe: func(context.Context) bool { return true },
	}
	index.reconcileGate <- struct{}{} // hold the gate as if another reconciliation is active
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, index.Reconcile(ctx), context.Canceled)
	require.Nil(t, index.lastSyncErr.Load(), "a queued cancellation must not poison readiness")
	<-index.reconcileGate

	wantErr := errors.New("embedding provider unavailable")
	index.reconcileFunc = func(context.Context) error { return wantErr }
	require.ErrorIs(t, index.Reconcile(context.Background()), wantErr)
	require.False(t, index.Ready(context.Background()))
	require.Equal(t, wantErr.Error(), *index.lastSyncErr.Load())
	// A successful query against a partially reconciled collection must not
	// erase the failed projection state. Only a complete reconciliation can do so.
	index.setSearchError(nil)
	require.False(t, index.Ready(context.Background()))

	index.reconcileFunc = func(context.Context) error { return nil }
	require.NoError(t, index.Reconcile(context.Background()))
	require.Nil(t, index.lastSyncErr.Load(), "successful reconciliation clears the previous failure")
	require.True(t, index.Ready(context.Background()), "complete reconciliation restores readiness")
}

func TestUnavailableSemanticUsesProvidedCause(t *testing.T) {
	cause := errors.New("qdrant startup failed")
	semantic := UnavailableSemantic(cause).(unavailableSemantic)
	require.Same(t, cause, semantic.Reconcile(context.Background()))
	_, err := semantic.SearchSemantic(context.Background(), aisearch.SearchQuery{})
	require.True(t, errors.Is(err, cause))
}

// TestSemanticRetrievalRunsThroughSharedHybridEngine verifies the full RCL
// retrieval boundary without a live Qdrant/Ollama process: an actual
// ai-go/search Service embeds and encodes the query, issues a hybrid filtered
// vector query, returns a source ID from its payload, and RCL projects that ID
// through the authoritative SQLite index. This protects the seam from
// regressing into a status-only integration or trusting vector payload text.
func TestSemanticRetrievalRunsThroughSharedHybridEngine(t *testing.T) {
	svc := newTestService(t)
	asset := testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "patterns.checkout", "Checkout Path", "A guided sequence for completing a purchase.")
	asset.ID = DocumentID(asset.Kind, asset.SubjectID)
	seedDocuments(t, svc, asset)

	store := searchtest.NewVectorStore()
	store.QueryResults = []aisearch.SearchResult{{
		ID: "vector-point-1", Score: .9,
		Payload: map[string]any{"source_id": asset.ID, "body": "untrusted vector payload text"},
	}}
	engine := aisearch.NewService(aisearch.ServiceOptions{
		Embedder:      &searchtest.Embedder{Vector: []float64{1, 0}, AvailableValue: true},
		SparseEncoder: aisearch.NewBM25SparseEncoder(),
		VectorStore:   store,
		DefaultLimit:  maxSearchLimit,
		MaxLimit:      maxSearchLimit,
	})
	semantic := &semanticIndex{
		engine: engine, store: store, mode: aisearch.ModeHybrid,
		readinessProbe: func(context.Context) bool { return true },
	}
	svc.WithSemantic(semantic)

	response, err := svc.Search(context.Background(), &searchv1.SearchRequest{
		Query: "purchase journey",
		Kinds: []searchv1.SearchKind{searchv1.SearchKind_SEARCH_KIND_ASSET},
	})
	require.NoError(t, err)
	require.Len(t, response.GetResults(), 1)
	require.Equal(t, asset.ID, response.GetResults()[0].GetId(), "the indexed SQLite document is authoritative")
	require.Equal(t, "Checkout Path", response.GetResults()[0].GetTitle())
	require.NotContains(t, response.GetResults()[0].GetSnippet(), "untrusted vector payload text")
	require.Equal(t, searchv1.SemanticLegState_SEMANTIC_LEG_STATE_READY, response.GetStatus().GetSemanticLeg())
	queries := store.QueryHistory()
	require.Len(t, queries, 1)
	require.NotNil(t, queries[0].Sparse, "RCL must execute the configured hybrid engine")
	require.NotNil(t, queries[0].Filter, "kind constraints must reach the vector leg")
	require.Equal(t, "kind", queries[0].Filter.Must[0].Key)
	require.Equal(t, []any{"asset"}, queries[0].Filter.Must[0].AnyOf)
}
