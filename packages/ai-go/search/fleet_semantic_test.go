package aisearch

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type fleetSemanticTestEmbedder struct {
	recipe    EmbeddingRecipe
	queries   int
	documents int
}

func (e *fleetSemanticTestEmbedder) Recipe() EmbeddingRecipe        { return e.recipe }
func (e *fleetSemanticTestEmbedder) Available(context.Context) bool { return true }
func (e *fleetSemanticTestEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	return e.EmbedDocument(ctx, text)
}
func (e *fleetSemanticTestEmbedder) EmbedQuery(_ context.Context, _ string) ([]float64, error) {
	e.queries++
	return []float64{1, 0}, nil
}
func (e *fleetSemanticTestEmbedder) EmbedDocument(_ context.Context, _ string) ([]float64, error) {
	e.documents++
	return []float64{0, 1}, nil
}

type fleetSemanticTestAdmission struct {
	mu        sync.Mutex
	deny      bool
	badProof  bool
	expiresIn time.Duration
	requests  []FleetSemanticAdmissionRequest
}

func (g *fleetSemanticTestAdmission) AdmitQdrantFleetSemanticOperation(_ context.Context, request FleetSemanticAdmissionRequest) (FleetSemanticAdmissionReceipt, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.requests = append(g.requests, request)
	if g.deny {
		return FleetSemanticAdmissionReceipt{}, errors.New("semantic operation denied")
	}
	if g.badProof {
		request.Plan.RuntimeRole = "foreign-role"
	}
	now := time.Now().UTC()
	expiresIn := g.expiresIn
	if expiresIn == 0 {
		expiresIn = time.Minute
	}
	return FleetSemanticAdmissionReceipt{Request: request, ReceiptID: "semantic-receipt", ObservedAt: now, ExpiresAt: now.Add(expiresIn)}, nil
}

func TestFleetSemanticStoreBindsQueryUpsertAndDeleteToActiveGeneration(t *testing.T) {
	ctx := context.Background()
	recipe := testFleetRecipe(t)
	catalog := newFleetTestCatalog()
	record := FleetGeneration{OwnerID: "owner", Alias: "owner_docs", ArmID: "arm-active", RecipeID: recipe.ID(), SnapshotID: "snapshot", Dimensions: 2, Collection: "collection-active", State: FleetActive, Version: 5, DataVersion: 2}
	if err := catalog.SaveFleetGenerationCAS(ctx, record, 0); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var aliasCollection = record.Collection
	var queryBody map[string]any
	var upsertBody map[string]any
	var deleteBody map[string]any
	upsertCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.URL.Path == "/collections/aliases" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"aliases": []map[string]string{{"alias_name": "owner_docs", "collection_name": aliasCollection}}}})
		case r.URL.Path == "/collections/collection-active/points/query":
			_ = json.NewDecoder(r.Body).Decode(&queryBody)
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"points": []map[string]any{{"id": "point-a", "score": 0.9, "payload": map[string]any{"source_id": "tenant-a/doc-a"}}}}})
		case r.URL.Path == "/collections/collection-active/points" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"result": []any{}})
		case r.URL.Path == "/collections/collection-active/points" && r.Method == http.MethodPut:
			_ = json.NewDecoder(r.Body).Decode(&upsertBody)
			upsertCalls++
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/collections/collection-active/points/count":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"count": 1}})
		case r.URL.Path == "/collections/collection-active/points/delete":
			_ = json.NewDecoder(r.Body).Decode(&deleteBody)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected semantic Qdrant request: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	adapter, err := NewQdrantFleetAdapter(QdrantFleetOptions{BaseURL: server.URL, OwnerID: "owner", Alias: "owner_docs", Client: server.Client(), Admission: &fleetTestGate{}, AliasBoundary: &fleetTestAliasBoundary{}, Source: &fleetTestSource{watermark: "1"}, Catalog: catalog})
	if err != nil {
		t.Fatal(err)
	}
	embedder := &fleetSemanticTestEmbedder{recipe: recipe}
	gate := &fleetSemanticTestAdmission{}
	store, err := adapter.NewSemanticStore(FleetSemanticOptions{Embedder: embedder, RuntimeRole: "search-hub-embedding", Estimator: fleetSemanticTestEstimator(4096), Admission: gate})
	if err != nil {
		t.Fatal(err)
	}
	attribution := semanticTestAttribution()
	queried, err := store.QueryActive(ctx, FleetSemanticQueryRequest{OperationID: "query-1", TenantID: "tenant-a", ArmID: record.ArmID, RecipeID: recipe.ID(), Text: "find documentation", Limit: 5, Attribution: attribution})
	if err != nil || len(queried.Results) != 1 || queried.Results[0].SourceID != "tenant-a/doc-a" || queried.Admission.ReceiptID != "semantic-receipt" {
		t.Fatalf("query result=%+v err=%v", queried, err)
	}
	if embedder.queries != 1 || embedder.documents != 0 {
		t.Fatalf("query used unexpected encoder calls: query=%d document=%d", embedder.queries, embedder.documents)
	}
	mu.Lock()
	var queryMap map[string]any
	if queryBody != nil {
		encoded, _ := json.Marshal(queryBody)
		_ = json.Unmarshal(encoded, &queryMap)
	}
	mu.Unlock()
	queryFilter := queryMap["filter"].(map[string]any)["must"].([]any)
	if len(queryFilter) != 5 || queryFilter[4].(map[string]any)["key"] != "fleet_tenant_id" || queryFilter[4].(map[string]any)["match"].(map[string]any)["value"] != "tenant-a" {
		t.Fatalf("query was not scoped to exact owner/alias/arm/recipe: %+v", queryMap)
	}

	upsertRequest := FleetSemanticUpsertRequest{OperationID: "upsert-1", TenantID: "tenant-a", ArmID: record.ArmID, RecipeID: recipe.ID(), SourceID: "tenant-a/doc-a", SourceSequence: 10, Document: FleetSemanticDocument{Text: "bounded source content", Payload: map[string]any{"kind": "provider-description", "user_id": "tenant-a"}}, Attribution: attribution}
	upserted, err := store.UpsertActive(ctx, upsertRequest)
	if err != nil || upserted.Generation.Version != 6 {
		t.Fatalf("upsert result=%+v err=%v", upserted, err)
	}
	if embedder.documents != 1 {
		t.Fatalf("upsert document embed calls=%d, want one", embedder.documents)
	}
	mu.Lock()
	encodedUpsert, _ := json.Marshal(upsertBody)
	mu.Unlock()
	var upsert map[string]any
	_ = json.Unmarshal(encodedUpsert, &upsert)
	points := upsert["points"].([]any)
	payload := points[0].(map[string]any)["payload"].(map[string]any)
	for key, want := range map[string]string{"source_id": "tenant-a/doc-a", "embedding_recipe_id": recipe.ID(), "fleet_owner_id": "owner", "fleet_alias": "owner_docs", "fleet_arm_id": "arm-active", "fleet_runtime_role": "search-hub-embedding", "fleet_rights_evidence_sha256": attribution.RightsEvidenceSHA256, "fleet_quality_evidence_sha256": attribution.QualityEvidenceSHA256} {
		if payload[key] != want {
			t.Errorf("upsert payload %s=%v want=%s", key, payload[key], want)
		}
	}
	if points[0].(map[string]any)["id"] != FleetSemanticPointID("owner", "owner_docs", "arm-active", "tenant-a", recipe.ID(), "tenant-a/doc-a") {
		t.Fatalf("semantic point ID is not owner/arm/tenant scoped: %+v", points[0])
	}
	replayed, err := store.UpsertActive(ctx, upsertRequest)
	if err != nil || replayed.Generation.Version != upserted.Generation.Version || replayed.Generation.DataVersion != upserted.Generation.DataVersion {
		t.Fatalf("exact operation replay changed the completed result: replay=%+v err=%v", replayed, err)
	}
	changed := upsertRequest
	changed.Document.Text = "different content under same operation ID"
	if _, err := store.UpsertActive(ctx, changed); !errors.Is(err, ErrFleetSemanticOperationConflict) {
		t.Fatalf("changed request reused completed operation ID: %v", err)
	}
	mu.Lock()
	if upsertCalls != 1 {
		t.Errorf("exact replay reapplied Qdrant upsert: calls=%d", upsertCalls)
	}
	mu.Unlock()

	deleted, err := store.DeleteActive(ctx, FleetSemanticDeleteRequest{OperationID: "delete-1", TenantID: "tenant-a", ArmID: record.ArmID, RecipeID: recipe.ID(), Sources: []FleetSemanticDeleteSource{{SourceID: "tenant-a/doc-a", SourceSequence: 11}}, Attribution: attribution})
	if err != nil || deleted.Generation.Version != 7 {
		t.Fatalf("delete result=%+v err=%v", deleted, err)
	}
	mu.Lock()
	encodedDelete, _ := json.Marshal(deleteBody)
	mu.Unlock()
	var deletion map[string]any
	_ = json.Unmarshal(encodedDelete, &deletion)
	must := deletion["filter"].(map[string]any)["must"].([]any)
	if len(must) != 6 || must[5].(map[string]any)["key"] != "fleet_tenant_id" || must[5].(map[string]any)["match"].(map[string]any)["value"] != "tenant-a" {
		t.Fatalf("delete did not target source IDs scoped to exact owner/alias/arm/recipe/tenant: %+v", deletion)
	}
	stale := upsertRequest
	stale.OperationID = "stale-upsert"
	stale.Document.Text = "stale content"
	if _, err := store.UpsertActive(ctx, stale); !errors.Is(err, ErrFleetSemanticSequenceStale) {
		t.Fatalf("stale source event error=%v, want sequence conflict", err)
	}
	if len(gate.requests) != 3 {
		t.Fatalf("semantic admission requests=%d want=3", len(gate.requests))
	}
	for i, request := range gate.requests {
		plan := request.Plan
		if plan.OwnerID != "owner" || plan.Alias != "owner_docs" || plan.Collection != record.Collection || plan.ArmID != record.ArmID || plan.RecipeID != recipe.ID() || plan.RuntimeRole != "search-hub-embedding" || plan.TenantID != "tenant-a" || plan.Dimensions != 2 || plan.GenerationVersion != []int64{5, 5, 6}[i] || !validFleetEvidenceDigest(plan.RequestSHA256) || request.PeakBytes != 4096 || plan.Attribution != attribution {
			t.Errorf("admission request %d lost active generation or attribution binding: %+v", i, request)
		}
	}
}

func TestFleetSemanticPointIDsSeparateTenantsForSameSourceIdentity(t *testing.T) {
	ctx := context.Background()
	recipe := testFleetRecipe(t)
	catalog := newFleetTestCatalog()
	record := FleetGeneration{OwnerID: "owner", Alias: "owner_docs", ArmID: "arm-active", RecipeID: recipe.ID(), Collection: "collection-active", Dimensions: 2, State: FleetActive, Version: 1}
	if err := catalog.SaveFleetGenerationCAS(ctx, record, 0); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var pointIDs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/collections/aliases" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"aliases": []map[string]string{{"alias_name": "owner_docs", "collection_name": "collection-active"}}}})
		case r.URL.Path == "/collections/collection-active/points" && r.Method == http.MethodPut:
			var body struct {
				Points []struct {
					ID string `json:"id"`
				} `json:"points"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode upsert body: %v", err)
			}
			mu.Lock()
			for _, point := range body.Points {
				pointIDs = append(pointIDs, point.ID)
			}
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/collections/collection-active/points" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"result": []any{}})
		default:
			t.Errorf("unexpected tenant identity request: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	adapter, err := NewQdrantFleetAdapter(QdrantFleetOptions{BaseURL: server.URL, OwnerID: "owner", Alias: "owner_docs", Client: server.Client(), Admission: &fleetTestGate{}, AliasBoundary: &fleetTestAliasBoundary{}, Source: &fleetTestSource{}, Catalog: catalog})
	if err != nil {
		t.Fatal(err)
	}
	store, err := adapter.NewSemanticStore(FleetSemanticOptions{Embedder: &fleetSemanticTestEmbedder{recipe: recipe}, RuntimeRole: "notes-embedding", Estimator: fleetSemanticTestEstimator(1024), Admission: &fleetSemanticTestAdmission{}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tenant := range []string{"tenant-a", "tenant-b"} {
		_, err := store.UpsertActive(ctx, FleetSemanticUpsertRequest{OperationID: "upsert-" + tenant, TenantID: tenant, ArmID: record.ArmID, RecipeID: recipe.ID(), SourceID: "same-note-id", SourceSequence: 1, Document: FleetSemanticDocument{Text: "tenant document", Payload: map[string]any{"user_id": tenant}}, Attribution: semanticTestAttribution()})
		if err != nil {
			t.Fatalf("upsert for %s: %v", tenant, err)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(pointIDs) != 2 || pointIDs[0] == pointIDs[1] {
		t.Fatalf("same source ID across tenants collided: %v", pointIDs)
	}
}

func TestFleetSemanticAdmissionReceiptRejectsUnboundedExpiryBeforeEmbedding(t *testing.T) {
	ctx := context.Background()
	recipe := testFleetRecipe(t)
	catalog := newFleetTestCatalog()
	record := FleetGeneration{OwnerID: "owner", Alias: "owner_docs", ArmID: "arm-active", RecipeID: recipe.ID(), Collection: "collection-active", Dimensions: 2, State: FleetActive, Version: 1}
	if err := catalog.SaveFleetGenerationCAS(ctx, record, 0); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/collections/aliases" {
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"aliases": []map[string]string{{"alias_name": "owner_docs", "collection_name": record.Collection}}}})
			return
		}
		t.Errorf("unbounded receipt reached vector I/O: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	}))
	defer server.Close()
	adapter, err := NewQdrantFleetAdapter(QdrantFleetOptions{BaseURL: server.URL, OwnerID: "owner", Alias: "owner_docs", Client: server.Client(), Admission: &fleetTestGate{}, AliasBoundary: &fleetTestAliasBoundary{}, Source: &fleetTestSource{}, Catalog: catalog})
	if err != nil {
		t.Fatal(err)
	}
	embedder := &fleetSemanticTestEmbedder{recipe: recipe}
	store, err := adapter.NewSemanticStore(FleetSemanticOptions{Embedder: embedder, RuntimeRole: "role", Estimator: fleetSemanticTestEstimator(1024), Admission: &fleetSemanticTestAdmission{expiresIn: 24 * time.Hour}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.QueryActive(ctx, FleetSemanticQueryRequest{OperationID: "unbounded-receipt", TenantID: "tenant-a", ArmID: record.ArmID, RecipeID: recipe.ID(), Text: "query", Limit: 1, Attribution: semanticTestAttribution()})
	if !errors.Is(err, ErrFleetAdmissionRequired) {
		t.Fatalf("unbounded expiry receipt error=%v, want fail-closed admission error", err)
	}
	if embedder.queries != 0 {
		t.Fatalf("unbounded receipt started query embedding: %d calls", embedder.queries)
	}
}

func TestFleetSemanticQueryRejectsWrongRecipeBeforeEmbeddingAndAdmission(t *testing.T) {
	ctx := context.Background()
	recipe := testFleetRecipe(t)
	catalog := newFleetTestCatalog()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("wrong recipe reached Qdrant: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	}))
	defer server.Close()
	adapter, err := NewQdrantFleetAdapter(QdrantFleetOptions{BaseURL: server.URL, OwnerID: "owner", Alias: "owner_docs", Client: server.Client(), Admission: &fleetTestGate{}, AliasBoundary: &fleetTestAliasBoundary{}, Source: &fleetTestSource{}, Catalog: catalog})
	if err != nil {
		t.Fatal(err)
	}
	embedder := &fleetSemanticTestEmbedder{recipe: recipe}
	gate := &fleetSemanticTestAdmission{}
	store, err := adapter.NewSemanticStore(FleetSemanticOptions{Embedder: embedder, RuntimeRole: "role", Estimator: fleetSemanticTestEstimator(4096), Admission: gate})
	if err != nil {
		t.Fatal(err)
	}
	request := FleetSemanticQueryRequest{OperationID: "wrong-recipe", TenantID: "tenant-a", ArmID: "arm", RecipeID: "sha256:foreign", Text: "query", Limit: 1, Attribution: semanticTestAttribution()}
	if _, err := store.QueryActive(ctx, request); err == nil {
		t.Fatal("query accepted a recipe different from the runtime encoder")
	}
	if embedder.queries != 0 || len(gate.requests) != 0 {
		t.Fatalf("wrong recipe reached encoder or admission: embed=%d admission=%d", embedder.queries, len(gate.requests))
	}
}

func TestFleetSemanticUpsertAdmissionDenialPrecedesEmbeddingAndVectorIO(t *testing.T) {
	ctx := context.Background()
	recipe := testFleetRecipe(t)
	catalog := newFleetTestCatalog()
	record := FleetGeneration{OwnerID: "owner", Alias: "owner_docs", ArmID: "arm-active", RecipeID: recipe.ID(), Collection: "collection-active", Dimensions: 2, State: FleetActive, Version: 1}
	if err := catalog.SaveFleetGenerationCAS(ctx, record, 0); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	pointsCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.URL.Path == "/collections/aliases" {
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"aliases": []map[string]string{{"alias_name": "owner_docs", "collection_name": record.Collection}}}})
			return
		}
		if strings.Contains(r.URL.Path, "/points") {
			pointsCalled = true
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	adapter, err := NewQdrantFleetAdapter(QdrantFleetOptions{BaseURL: server.URL, OwnerID: "owner", Alias: "owner_docs", Client: server.Client(), Admission: &fleetTestGate{}, AliasBoundary: &fleetTestAliasBoundary{}, Source: &fleetTestSource{}, Catalog: catalog})
	if err != nil {
		t.Fatal(err)
	}
	embedder := &fleetSemanticTestEmbedder{recipe: recipe}
	gate := &fleetSemanticTestAdmission{deny: true}
	store, err := adapter.NewSemanticStore(FleetSemanticOptions{Embedder: embedder, RuntimeRole: "role", Estimator: fleetSemanticTestEstimator(1024), Admission: gate})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.UpsertActive(ctx, FleetSemanticUpsertRequest{OperationID: "wrong-tenant", TenantID: "tenant-a", ArmID: record.ArmID, RecipeID: recipe.ID(), SourceID: "tenant-a/doc", SourceSequence: 1, Document: FleetSemanticDocument{Text: "content", Payload: map[string]any{"user_id": "tenant-b"}}, Attribution: semanticTestAttribution()})
	if err == nil {
		t.Fatal("upsert accepted a document belonging to another tenant")
	}
	if embedder.documents != 0 || len(gate.requests) != 0 {
		t.Fatalf("cross-tenant upsert reached encoder or admission: embed=%d admission=%d", embedder.documents, len(gate.requests))
	}
	_, err = store.UpsertActive(ctx, FleetSemanticUpsertRequest{OperationID: "deny-upsert", TenantID: "tenant-a", ArmID: record.ArmID, RecipeID: recipe.ID(), SourceID: "tenant/doc", SourceSequence: 1, Document: FleetSemanticDocument{Text: "content", Payload: map[string]any{"user_id": "tenant-a"}}, Attribution: semanticTestAttribution()})
	if err == nil {
		t.Fatal("capacity denial was ignored")
	}
	if embedder.documents != 0 {
		t.Fatalf("denied upsert started embedding: %d calls", embedder.documents)
	}
	mu.Lock()
	defer mu.Unlock()
	if pointsCalled {
		t.Fatal("denied upsert reached Qdrant vector I/O")
	}
}

type fleetSemanticTestEstimator int64

func (e fleetSemanticTestEstimator) EstimateFleetSemanticPeakBytes(_ context.Context, plan FleetSemanticOperationPlan) (int64, error) {
	if plan.GenerationVersion <= 0 || plan.Dimensions <= 0 || plan.OperationID == "" {
		return 0, errors.New("incomplete semantic operation plan")
	}
	return int64(e), nil
}

func semanticTestAttribution() FleetSemanticAttribution {
	return FleetSemanticAttribution{CallerID: "notes-api", RightsEvidenceURI: "rights://notes/role", RightsEvidenceSHA256: strings.Repeat("a", 64), QualityEvidenceURI: "eval://notes/heldout", QualityEvidenceSHA256: strings.Repeat("b", 64)}
}
