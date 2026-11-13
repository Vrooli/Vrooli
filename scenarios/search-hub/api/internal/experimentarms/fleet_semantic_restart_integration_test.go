package experimentarms

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	aisearch "github.com/vrooli/ai-go/search"
)

type semanticRestartEmbedder struct {
	recipe    aisearch.EmbeddingRecipe
	queries   int
	documents int
}

func (e *semanticRestartEmbedder) Recipe() aisearch.EmbeddingRecipe { return e.recipe }
func (e *semanticRestartEmbedder) Available(context.Context) bool   { return true }
func (e *semanticRestartEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	return e.EmbedDocument(ctx, text)
}
func (e *semanticRestartEmbedder) EmbedQuery(context.Context, string) ([]float64, error) {
	e.queries++
	return []float64{1, 0}, nil
}
func (e *semanticRestartEmbedder) EmbedDocument(context.Context, string) ([]float64, error) {
	e.documents++
	return []float64{0, 1}, nil
}

type semanticRestartAdmission struct {
	calls int
}

func (g *semanticRestartAdmission) AdmitQdrantFleetSemanticOperation(_ context.Context, request aisearch.FleetSemanticAdmissionRequest) (aisearch.FleetSemanticAdmissionReceipt, error) {
	g.calls++
	now := time.Now().UTC()
	return aisearch.FleetSemanticAdmissionReceipt{Request: request, ReceiptID: "semantic-restart-receipt", ObservedAt: now, ExpiresAt: now.Add(time.Minute)}, nil
}

type semanticRestartEstimator int64

func (e semanticRestartEstimator) EstimateFleetSemanticPeakBytes(context.Context, aisearch.FleetSemanticOperationPlan) (int64, error) {
	return int64(e), nil
}

type semanticRestartLegacyAdmission struct{}

func (semanticRestartLegacyAdmission) AdmitQdrantFleetOperation(_ context.Context, operation aisearch.FleetOperation, armID, target string, peak int64) (aisearch.FleetAdmission, error) {
	now := time.Now().UTC()
	return aisearch.FleetAdmission{OperationID: string(operation) + ":" + armID, ReceiptID: "legacy-admission", TargetID: target, PeakBytes: peak, ObservedAt: now, ExpiresAt: now.Add(time.Minute)}, nil
}

type semanticRestartQdrant struct {
	mu         sync.Mutex
	points     map[string]map[string]any
	upserts    int
	retrievals int
}

func (q *semanticRestartQdrant) handler(alias, collection string) http.Handler {
	if q.points == nil {
		q.points = map[string]map[string]any{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/collections/aliases" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"aliases": []map[string]string{{"alias_name": alias, "collection_name": collection}}}})
		case r.URL.Path == "/collections/"+collection+"/points" && r.Method == http.MethodPost:
			var request struct {
				IDs []string `json:"ids"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			q.mu.Lock()
			q.retrievals++
			var points []map[string]any
			for _, id := range request.IDs {
				if payload, ok := q.points[id]; ok {
					points = append(points, map[string]any{"id": id, "payload": payload})
				}
			}
			q.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"result": points})
		case r.URL.Path == "/collections/"+collection+"/points" && r.Method == http.MethodPut:
			var request struct {
				Points []struct {
					ID      string         `json:"id"`
					Payload map[string]any `json:"payload"`
				} `json:"points"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			q.mu.Lock()
			for _, point := range request.Points {
				q.points[point.ID] = point.Payload
				q.upserts++
			}
			q.mu.Unlock()
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "unexpected fake Qdrant request", http.StatusNotFound)
		}
	})
}

func newSemanticRestartRecipe(t *testing.T) aisearch.EmbeddingRecipe {
	t.Helper()
	recipe, err := aisearch.NewEmbeddingRecipe(aisearch.EmbeddingRecipeInput{
		SchemaVersion: "1", ModelID: "notes-test-model", ModelRevision: "rev-1", ArtifactSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		TokenizerRevision: "tok-1", Runtime: "test-runtime", RuntimeBuild: "build-1", Precision: "float32", QueryInstruction: "query", DocumentInstruction: "document",
		Pooling: "mean", Normalization: "l2", Dimensions: 2, Projection: "none", OutputDType: "float32", OutputQuantization: "none", Truncation: "none", ChunkerVersion: "single-document-v1", ComposerVersion: "notes-v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return recipe
}

func semanticRestartAttribution() aisearch.FleetSemanticAttribution {
	return aisearch.FleetSemanticAttribution{CallerID: "notes", RightsEvidenceURI: "rights://notes", RightsEvidenceSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", QualityEvidenceURI: "quality://notes", QualityEvidenceSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
}

func semanticRestartRequest(armID, recipeID string) aisearch.FleetSemanticUpsertRequest {
	return aisearch.FleetSemanticUpsertRequest{OperationID: "note-write-100", TenantID: "tenant-a", ArmID: armID, RecipeID: recipeID, SourceID: "tenant-a/note-a", SourceSequence: 100, Document: aisearch.FleetSemanticDocument{Text: "current note", Payload: map[string]any{"user_id": "tenant-a", "note_id": "note-a"}}, Attribution: semanticRestartAttribution()}
}

func openSemanticRestartCatalog(t *testing.T, path string) (*sql.DB, *aisearch.SQLiteFleetGenerationCatalog) {
	t.Helper()
	db, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := aisearch.NewSQLiteFleetGenerationCatalog(db)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := catalog.Initialize(context.Background()); err != nil {
		db.Close()
		t.Fatal(err)
	}
	return db, catalog
}

func newSemanticRestartStore(t *testing.T, server *httptest.Server, catalog *aisearch.SQLiteFleetGenerationCatalog, boundary *fleetIntegrationAliasBoundary, recipe aisearch.EmbeddingRecipe, embedder aisearch.FleetSemanticEmbedder, admission *semanticRestartAdmission) aisearch.FleetSemanticStore {
	t.Helper()
	adapter, err := aisearch.NewQdrantFleetAdapter(aisearch.QdrantFleetOptions{BaseURL: server.URL, OwnerID: "search-hub", Alias: "search-hub_docs", Client: server.Client(), Admission: semanticRestartLegacyAdmission{}, AliasBoundary: boundary, Source: &fleetIntegrationSource{watermark: "100"}, Catalog: catalog})
	if err != nil {
		t.Fatal(err)
	}
	store, err := adapter.NewSemanticStore(aisearch.FleetSemanticOptions{Embedder: embedder, RuntimeRole: "notes-embedding", Estimator: semanticRestartEstimator(8192), Admission: admission})
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func seedSemanticRestartGeneration(t *testing.T, catalog *aisearch.SQLiteFleetGenerationCatalog, recipe aisearch.EmbeddingRecipe) {
	t.Helper()
	if err := catalog.SaveFleetGenerationCAS(context.Background(), aisearch.FleetGeneration{OwnerID: "search-hub", Alias: "search-hub_docs", ArmID: "arm-a", RecipeID: recipe.ID(), SnapshotID: "snapshot-a", IndexSpecDigest: "sha256:index", Collection: "semantic-current", Dimensions: 2, State: aisearch.FleetActive, Version: 1}, 0); err != nil {
		t.Fatal(err)
	}
}

func TestFleetSemanticRestartReconcilesAppliedUpsertAndBumpsVersionOnce(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "semantic-restart.sqlite")
	db, catalog := openSemanticRestartCatalog(t, path)
	recipe := newSemanticRestartRecipe(t)
	seedSemanticRestartGeneration(t, catalog, recipe)
	qdrant := &semanticRestartQdrant{points: map[string]map[string]any{}}
	server := httptest.NewServer(qdrant.handler("search-hub_docs", "semantic-current"))
	defer server.Close()
	trigger := `CREATE TRIGGER fail_generation_cas BEFORE UPDATE ON qdrant_fleet_generations BEGIN SELECT RAISE(ABORT, 'simulated crash after vector effect'); END;`
	if _, err := db.ExecContext(ctx, trigger); err != nil {
		t.Fatal(err)
	}
	firstEmbedder := &semanticRestartEmbedder{recipe: recipe}
	firstAdmission := &semanticRestartAdmission{}
	firstStore := newSemanticRestartStore(t, server, catalog, &fleetIntegrationAliasBoundary{}, recipe, firstEmbedder, firstAdmission)
	request := semanticRestartRequest("arm-a", recipe.ID())
	if _, err := firstStore.UpsertActive(ctx, request); err == nil {
		t.Fatal("injected generation CAS failure was ignored")
	}
	qdrant.mu.Lock()
	if qdrant.upserts != 1 || firstEmbedder.documents != 1 {
		t.Fatalf("first attempt upserts=%d embeddings=%d, want one applied vector effect", qdrant.upserts, firstEmbedder.documents)
	}
	qdrant.mu.Unlock()
	if _, err := db.ExecContext(ctx, `DROP TRIGGER fail_generation_cas`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, catalog = openSemanticRestartCatalog(t, path)
	defer db.Close()
	secondEmbedder := &semanticRestartEmbedder{recipe: recipe}
	secondAdmission := &semanticRestartAdmission{}
	secondStore := newSemanticRestartStore(t, server, catalog, &fleetIntegrationAliasBoundary{}, recipe, secondEmbedder, secondAdmission)
	result, err := secondStore.UpsertActive(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Generation.Version != 2 || result.Generation.DataVersion != 1 {
		t.Fatalf("recovered generation version=%d data=%d, want exactly one increment", result.Generation.Version, result.Generation.DataVersion)
	}
	if secondAdmission.calls != 1 || secondEmbedder.documents != 0 {
		t.Fatalf("recovery admission=%d embeddings=%d, want fresh admission and no duplicate embedding", secondAdmission.calls, secondEmbedder.documents)
	}
	qdrant.mu.Lock()
	if qdrant.upserts != 1 {
		t.Fatalf("restart replay applied vector upsert %d times, want once", qdrant.upserts)
	}
	qdrant.mu.Unlock()
	replayed, err := secondStore.UpsertActive(ctx, request)
	if err != nil || replayed.Generation.Version != 2 || secondAdmission.calls != 1 {
		t.Fatalf("completed replay=%+v admission calls=%d err=%v", replayed, secondAdmission.calls, err)
	}
	stale := request
	stale.OperationID = "stale-note-write"
	stale.SourceSequence = 99
	if _, err := secondStore.UpsertActive(ctx, stale); !errors.Is(err, aisearch.ErrFleetSemanticSequenceStale) {
		t.Fatalf("stale sequence replay error=%v", err)
	}
	if secondAdmission.calls != 1 || secondEmbedder.documents != 0 {
		t.Fatalf("stale sequence reached admission/embedding: admission=%d embedding=%d", secondAdmission.calls, secondEmbedder.documents)
	}
	changed := request
	changed.Document.Text = "changed under same ID"
	if _, err := secondStore.UpsertActive(ctx, changed); !errors.Is(err, aisearch.ErrFleetSemanticOperationConflict) {
		t.Fatalf("changed request reuse error=%v", err)
	}
}

func TestFleetSemanticRestartCompletesLedgerAfterGenerationCAS(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "semantic-ledger-restart.sqlite")
	db, catalog := openSemanticRestartCatalog(t, path)
	recipe := newSemanticRestartRecipe(t)
	seedSemanticRestartGeneration(t, catalog, recipe)
	qdrant := &semanticRestartQdrant{points: map[string]map[string]any{}}
	server := httptest.NewServer(qdrant.handler("search-hub_docs", "semantic-current"))
	defer server.Close()
	trigger := `CREATE TRIGGER fail_semantic_complete BEFORE UPDATE OF state ON qdrant_fleet_semantic_operations WHEN NEW.state = 'completed' BEGIN SELECT RAISE(ABORT, 'simulated crash before ledger completion'); END;`
	if _, err := db.ExecContext(ctx, trigger); err != nil {
		t.Fatal(err)
	}
	firstEmbedder := &semanticRestartEmbedder{recipe: recipe}
	firstAdmission := &semanticRestartAdmission{}
	firstStore := newSemanticRestartStore(t, server, catalog, &fleetIntegrationAliasBoundary{}, recipe, firstEmbedder, firstAdmission)
	request := semanticRestartRequest("arm-a", recipe.ID())
	if _, err := firstStore.UpsertActive(ctx, request); err == nil {
		t.Fatal("injected operation-ledger completion failure was ignored")
	}
	generation, ok, err := catalog.LoadFleetGeneration(ctx, "arm-a")
	if err != nil || !ok || generation.Version != 2 || generation.LastSemanticOperationID != request.OperationID {
		t.Fatalf("generation CAS did not persist expected recovery marker: %+v ok=%v err=%v", generation, ok, err)
	}
	if _, err := db.ExecContext(ctx, `DROP TRIGGER fail_semantic_complete`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, catalog = openSemanticRestartCatalog(t, path)
	defer db.Close()
	secondEmbedder := &semanticRestartEmbedder{recipe: recipe}
	secondAdmission := &semanticRestartAdmission{}
	secondStore := newSemanticRestartStore(t, server, catalog, &fleetIntegrationAliasBoundary{}, recipe, secondEmbedder, secondAdmission)
	result, err := secondStore.UpsertActive(ctx, request)
	if err != nil || result.Generation.Version != 2 || secondAdmission.calls != 0 || secondEmbedder.documents != 0 {
		t.Fatalf("post-CAS replay=%+v admission=%d embeds=%d err=%v", result, secondAdmission.calls, secondEmbedder.documents, err)
	}
	qdrant.mu.Lock()
	defer qdrant.mu.Unlock()
	if qdrant.upserts != 1 {
		t.Fatalf("post-CAS replay applied Qdrant upserts=%d, want one", qdrant.upserts)
	}
}
