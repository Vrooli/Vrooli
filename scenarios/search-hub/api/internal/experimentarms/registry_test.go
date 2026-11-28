package experimentarms

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	aisearch "github.com/vrooli/ai-go/search"
	corestorage "github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

func TestRegistryBindsFleetIdentityToResolvedNamespace(t *testing.T) {
	db, _ := openRegistryDB(t)
	namespace, err := corestorage.ResolveNamespace(corestorage.NamespaceConfig{Root: "search-hub_shadow", Variant: "shadow"})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := NewForNamespace(db, namespace)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	if registry.fleetOwnerID != "search-hub_shadow.embedding-fleet" || registry.fleetAlias != "search-hub_shadow_embedding-fleet" {
		t.Fatalf("fleet identity must follow the trusted variant namespace: owner=%q alias=%q", registry.fleetOwnerID, registry.fleetAlias)
	}
	if strings.HasPrefix(registry.fleetAlias, "agent-manager_") {
		t.Fatalf("Search Hub must be statically disjoint from the Agent Manager alias namespace: %q", registry.fleetAlias)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("adapter must be denied by the registry-owned durable boundary before Qdrant I/O: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	}))
	defer server.Close()
	decoyDB, _ := openRegistryDB(t)
	decoyCatalog, err := aisearch.NewSQLiteFleetGenerationCatalog(decoyDB)
	if err != nil {
		t.Fatal(err)
	}
	if err := decoyCatalog.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	adapter, err := registry.newQdrantFleetAdapter(aisearch.QdrantFleetOptions{
		OwnerID: "agent-manager.runs", Alias: "agent-manager_conversation-search",
		BaseURL: server.URL, Admission: &fleetIntegrationGate{}, AliasBoundary: &fleetIntegrationAliasBoundary{},
		Source: &fleetIntegrationSource{}, Catalog: decoyCatalog,
	})
	if err != nil {
		t.Fatal(err)
	}
	ownerID, alias := adapter.FleetIdentity()
	if ownerID != registry.fleetOwnerID || alias != registry.fleetAlias {
		t.Fatalf("caller-supplied identity escaped trusted startup binding: owner=%q alias=%q", ownerID, alias)
	}
	record := aisearch.FleetGeneration{
		OwnerID: ownerID, Alias: alias, ArmID: "arm-1", RecipeID: "recipe-1", SnapshotID: "snapshot-1",
		Dimensions: 2, IndexSpecDigest: "index-1", Collection: "search-hub_shadow_embedding-fleet--fleet--arm-1",
		State: aisearch.FleetCandidate, Version: 1,
	}
	if err := registry.fleetCatalog.SaveFleetGenerationCAS(context.Background(), record, 0); err != nil {
		t.Fatal(err)
	}
	endpointID, err := aisearch.CanonicalQdrantEndpointID(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	foreignOperation := aisearch.FleetAliasOperation{OwnerID: ownerID, EndpointID: endpointID, Alias: alias, OperationID: "held-by-registry-catalog", Kind: aisearch.FleetPromote, ArmID: "arm-1", Target: record.Collection}
	fence, err := registry.fleetBoundary.AcquireFleetAliasMutation(context.Background(), foreignOperation)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Promote(context.Background(), record.ArmID, 128); !errors.Is(err, aisearch.ErrFleetStateConflict) {
		t.Fatalf("caller-supplied catalog/boundary must not bypass the registry-owned lock, got %v", err)
	}
	if err := fence.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestProductionFleetAdapterRequiresTrustedRuntimeAndBuildsOwnerSource(t *testing.T) {
	ctx := context.Background()
	db, _ := openRegistryDB(t)
	namespace, err := corestorage.ResolveNamespace(corestorage.NamespaceConfig{Root: "search-hub_shadow", Variant: "shadow"})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := NewForNamespace(db, namespace)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("adapter construction must not make Qdrant requests: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	}))
	defer server.Close()
	options := aisearch.QdrantFleetOptions{
		BaseURL: server.URL, OwnerID: "agent-manager.runs", Alias: "agent-manager_conversation-search",
		Source: &fleetIntegrationSource{watermark: "foreign"}, Catalog: nil,
	}
	encoder := providerFleetTestEncoder{recipe: validRecipeForTest(t)}
	armInput := validArmCreate(encoder.recipe)
	armInput.OwnerID = "caller-selected-owner"
	armInput.ArmID = "trusted-owner-arm"
	createdArm, err := registry.Create(ctx, armInput)
	if err != nil || createdArm.OwnerID != registry.fleetOwnerID {
		t.Fatalf("arm owner must be injected by lifecycle identity: arm=%+v err=%v", createdArm, err)
	}
	if _, err := registry.NewProviderDescriptionFleetAdapter(options, encoder); !errors.Is(err, aisearch.ErrFleetAdmissionRequired) {
		t.Fatalf("production adapter must remain closed without the capacity owner's operation gate, got %v", err)
	}
	options.Admission = &fleetIntegrationGate{}
	adapter, err := registry.NewProviderDescriptionFleetAdapter(options, encoder)
	if err != nil {
		t.Fatalf("construct trusted production adapter with owner encoder and admission gate: %v", err)
	}
	ownerID, alias := adapter.FleetIdentity()
	if ownerID != registry.fleetOwnerID || alias != registry.fleetAlias {
		t.Fatalf("production adapter escaped its lifecycle-bound owner identity: owner=%q alias=%q", ownerID, alias)
	}
	if _, err := registry.NewProviderDescriptionFleetAdapter(aisearch.QdrantFleetOptions{Admission: &fleetIntegrationGate{}}, nil); err == nil {
		t.Fatal("production adapter must reject a missing trusted recipe encoder")
	}
	if _, err := registry.NewFleetGenerationStage(aisearch.QdrantFleetOptions{BaseURL: server.URL}, encoder, staticFleetPeakEstimator(4096)); !errors.Is(err, aisearch.ErrFleetAdmissionRequired) {
		t.Fatalf("stage service must reject absent exact-operation admission: %v", err)
	}
	options.Admission = &fleetIntegrationGate{}
	stage, err := registry.NewFleetGenerationStage(options, encoder, staticFleetPeakEstimator(4096))
	if err != nil {
		t.Fatalf("construct lifecycle stage service with all trusted dependencies: %v", err)
	}
	ownerID, alias = stage.adapter.FleetIdentity()
	if ownerID != registry.fleetOwnerID || alias != registry.fleetAlias || stage.source.db != db || stage.encoder.Recipe().ID() != encoder.Recipe().ID() {
		t.Fatalf("stage service did not bind its source/recipe/adapter to owner lifecycle: owner=%q alias=%q source=%T encoder=%s", ownerID, alias, stage.source.db, stage.encoder.Recipe().ID())
	}
}

type fleetSemanticIntegrationGate struct{}

type fleetSemanticIntegrationEstimator int64

func (e fleetSemanticIntegrationEstimator) EstimateFleetSemanticPeakBytes(context.Context, aisearch.FleetSemanticOperationPlan) (int64, error) {
	return int64(e), nil
}

func (fleetSemanticIntegrationGate) AdmitQdrantFleetSemanticOperation(_ context.Context, request aisearch.FleetSemanticAdmissionRequest) (aisearch.FleetSemanticAdmissionReceipt, error) {
	now := time.Now().UTC()
	return aisearch.FleetSemanticAdmissionReceipt{Request: request, ReceiptID: "semantic-owner-receipt", ObservedAt: now, ExpiresAt: now.Add(time.Minute)}, nil
}

func TestProductionSemanticStoreRequiresTrustedDependenciesAndBindsOwnerSource(t *testing.T) {
	ctx := context.Background()
	db, _ := openRegistryDB(t)
	namespace, err := corestorage.ResolveNamespace(corestorage.NamespaceConfig{Root: "search-hub_shadow", Variant: "shadow"})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := NewForNamespace(db, namespace)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("semantic-store construction must not call Qdrant: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	}))
	defer server.Close()
	encoder := providerFleetTestEncoder{recipe: validRecipeForTest(t)}
	options := aisearch.QdrantFleetOptions{BaseURL: server.URL, OwnerID: "caller-owner", Alias: "caller-alias", Source: &fleetIntegrationSource{watermark: "foreign"}}
	semanticOptions := aisearch.FleetSemanticOptions{RuntimeRole: "search-hub-embedding", Estimator: fleetSemanticIntegrationEstimator(2048)}
	if _, err := registry.NewProviderDescriptionFleetSemanticStore(options, encoder, semanticOptions); !errors.Is(err, aisearch.ErrFleetAdmissionRequired) {
		t.Fatalf("semantic store accepted missing owner admission: %v", err)
	}
	semanticOptions.Admission = fleetSemanticIntegrationGate{}
	if _, err := registry.NewProviderDescriptionFleetSemanticStore(options, nil, semanticOptions); err == nil {
		t.Fatal("semantic store accepted missing lifecycle recipe encoder")
	}
	store, err := registry.NewProviderDescriptionFleetSemanticStore(options, encoder, semanticOptions)
	if err != nil {
		t.Fatal(err)
	}
	semantic, ok := store.(*aisearch.QdrantFleetSemanticStore)
	if !ok {
		t.Fatalf("production semantic store type=%T", store)
	}
	owner, alias := semantic.FleetIdentity()
	if owner != registry.fleetOwnerID || alias != registry.fleetAlias {
		t.Fatalf("semantic store escaped trusted lifecycle identity: owner=%q alias=%q", owner, alias)
	}
}

type staticFleetPeakEstimator int64

func (e staticFleetPeakEstimator) EstimateFleetStagePeakBytes(context.Context, string, aisearch.EmbeddingRecipe, uint64) (int64, error) {
	return int64(e), nil
}

func TestRegistryFactorySuccessfulPromotionReleasesIndependentCatalogAndFence(t *testing.T) {
	ctx := context.Background()
	db, _ := openRegistryDB(t)
	namespace, err := corestorage.ResolveNamespace(corestorage.NamespaceConfig{Root: "search-hub", Variant: "live"})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := NewForNamespace(db, namespace)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	ownerID, alias := registry.fleetOwnerID, registry.fleetAlias
	old := validFleetRecord(ownerID, alias, "old-arm", "old-collection", aisearch.FleetActive)
	candidate := validFleetRecord(ownerID, alias, "candidate-arm", "candidate-collection", aisearch.FleetCandidate)
	if err := registry.fleetCatalog.SaveFleetGenerationCAS(ctx, old, 0); err != nil {
		t.Fatal(err)
	}
	if err := registry.fleetCatalog.SaveFleetGenerationCAS(ctx, candidate, 0); err != nil {
		t.Fatal(err)
	}
	activeCollection := old.Collection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/collections/aliases" && r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"aliases": []map[string]string{{"alias_name": alias, "collection_name": activeCollection}}}})
			return
		}
		if r.URL.Path == "/collections/aliases" && r.Method == http.MethodPost {
			var body struct {
				Actions []map[string]map[string]string `json:"actions"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode alias actions: %v", err)
			}
			for _, action := range body.Actions {
				if create, ok := action["create_alias"]; ok {
					activeCollection = create["collection_name"]
				}
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		t.Errorf("unexpected Qdrant request: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	}))
	defer server.Close()
	decoyDB, _ := openRegistryDB(t)
	decoyCatalog, err := aisearch.NewSQLiteFleetGenerationCatalog(decoyDB)
	if err != nil {
		t.Fatal(err)
	}
	if err := decoyCatalog.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	adapter, err := registry.newQdrantFleetAdapter(aisearch.QdrantFleetOptions{
		BaseURL: server.URL, OwnerID: "untrusted", Alias: "untrusted_alias", Admission: &fleetIntegrationGate{},
		AliasBoundary: &fleetIntegrationAliasBoundary{}, Source: &fleetIntegrationSource{watermark: "w1"}, Catalog: decoyCatalog,
		Client: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Promote(ctx, candidate.ArmID, 1024); err != nil {
		t.Fatalf("successful factory-bound promotion should release catalog operation and distinct alias fence exactly once: %v", err)
	}
	if activeCollection != candidate.Collection {
		t.Fatalf("active collection=%q, want promoted candidate %q", activeCollection, candidate.Collection)
	}
	operation := aisearch.FleetAliasOperation{OwnerID: ownerID, EndpointID: mustEndpointID(t, server.URL), Alias: alias, OperationID: "probe-after-completion", Kind: aisearch.FleetPromote, ArmID: candidate.ArmID, Target: candidate.Collection}
	fence, err := registry.fleetBoundary.AcquireFleetAliasMutation(ctx, operation)
	if err != nil {
		t.Fatalf("completed promotion left its alias fence held: %v", err)
	}
	if err := fence.Release(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteFleetAliasFenceRejectsConcurrentDuplicateAndReclaimsOnlyDeadExactOperation(t *testing.T) {
	ctx := context.Background()
	db, _ := openRegistryDB(t)
	first, err := aisearch.NewSQLiteFleetAliasMutationBoundary(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	second, err := aisearch.NewSQLiteFleetAliasMutationBoundary(db)
	if err != nil {
		t.Fatal(err)
	}
	operation := aisearch.FleetAliasOperation{OwnerID: "search-hub.embedding-fleet", EndpointID: "http://qdrant:6333", Alias: "search-hub_embedding-fleet", OperationID: "promote:arm-a:collection-a", Kind: aisearch.FleetPromote, ArmID: "arm-a", Target: "collection-a"}
	fence, err := first.AcquireFleetAliasMutation(ctx, operation)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := second.AcquireFleetAliasMutation(ctx, operation); !errors.Is(err, aisearch.ErrFleetStateConflict) {
		t.Fatalf("same-operation concurrent retry must be excluded while its original holder is alive, got %v", err)
	}
	if err := fence.Release(ctx); err != nil {
		t.Fatal(err)
	}
	completedFence, err := second.AcquireFleetAliasMutation(ctx, operation)
	if err != nil {
		t.Fatalf("completed operation should release its fence: %v", err)
	}
	if err := completedFence.Release(ctx); err != nil {
		t.Fatal(err)
	}

	// A restart leaves a fence only if the prior process died before defer
	// cleanup. Simulate that durable row, then verify only its exact operation
	// can reclaim it; a different target remains blocked.
	if _, err := db.ExecContext(ctx, `UPDATE qdrant_fleet_alias_fences SET holder_pid = ?, holder_start = ? WHERE endpoint_id = ? AND alias_name = ?`, 2147483647, "dead-process", operation.EndpointID, operation.Alias); err != nil {
		t.Fatal(err)
	}
	if _, err := first.AcquireFleetAliasMutation(ctx, operation); err != nil {
		t.Fatalf("same durable operation could not recover dead holder: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE qdrant_fleet_alias_fences SET holder_pid = ?, holder_start = ? WHERE endpoint_id = ? AND alias_name = ?`, 2147483647, "dead-process", operation.EndpointID, operation.Alias); err != nil {
		t.Fatal(err)
	}
	foreign := operation
	foreign.OperationID = "promote:arm-b:collection-b"
	foreign.ArmID = "arm-b"
	foreign.Target = "collection-b"
	if _, err := second.AcquireFleetAliasMutation(ctx, foreign); !errors.Is(err, aisearch.ErrFleetStateConflict) {
		t.Fatalf("different operation must not take over dead holder fence: %v", err)
	}
	reclaimedFence, err := second.AcquireFleetAliasMutation(ctx, operation)
	if err != nil {
		t.Fatalf("exact operation must reclaim the dead-holder fence: %v", err)
	}
	if err := reclaimedFence.Release(ctx); err != nil {
		t.Fatal(err)
	}
}

func validFleetRecord(ownerID, alias, armID, collection string, state aisearch.FleetGenerationState) aisearch.FleetGeneration {
	record := aisearch.FleetGeneration{
		OwnerID: ownerID, Alias: alias, ArmID: armID, RecipeID: "recipe-" + armID, SnapshotID: "snapshot-" + armID,
		SnapshotWatermark: "w1", CatchupWatermark: "w1", Dimensions: 2, IndexSpecDigest: "index-" + armID,
		Collection: collection, State: state, Version: 1, DataVersion: 1,
		ValidationDataVersion: 1, ValidationWatermark: "w1", ValidationExpected: 1, ValidationActual: 1, ValidationValid: true,
	}
	validation, _ := json.Marshal(aisearch.FleetValidation{ArmID: armID, RecipeID: record.RecipeID, Watermark: "w1", DataVersion: 1, ExpectedPointCount: 1, ActualPointCount: 1, Valid: true})
	digest := sha256.Sum256(validation)
	record.ValidationDigest = "sha256:" + hex.EncodeToString(digest[:])
	return record
}

func mustEndpointID(t *testing.T, endpoint string) string {
	t.Helper()
	id, err := aisearch.CanonicalQdrantEndpointID(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestRegistryCreateRetryAndRestartPreserveImmutableEvidence(t *testing.T) {
	ctx := context.Background()
	db, path := openRegistryDB(t)
	registry, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	input := validRecipe()
	recipe, err := aisearch.NewEmbeddingRecipe(input)
	if err != nil {
		t.Fatal(err)
	}
	create := validArmCreate(recipe)
	created, err := registry.Create(ctx, create)
	if err != nil {
		t.Fatal(err)
	}
	retried, err := registry.Create(ctx, create)
	if err != nil || !sameImmutableArm(created, retried) || retried.Version != 1 {
		t.Fatalf("idempotent create did not return same arm: arm=%+v err=%v", retried, err)
	}
	create.BuildID = "changed-build"
	if _, err := registry.Create(ctx, create); !errors.Is(err, ErrImmutableConflict) {
		t.Fatalf("expected immutable identity conflict, got %v", err)
	}
	firstTransition := TransitionRequest{ArmID: created.ArmID, ExpectedState: StateDiscovered, ExpectedVersion: 1, To: StateCapabilityVerified, IdempotencyKey: "after-restart", Receipt: testReceipt("capability", "capability-1")}
	firstResult, err := registry.Transition(ctx, firstTransition)
	if err != nil {
		t.Fatal(err)
	}
	input.ModelRevision = "caller-mutated-after-create"
	create.BuildID = "search-hub-build-9"
	create.Evidence[0].SHA256 = strings.Repeat("b", 64)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	reopened, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	recovered, err := reopened.Get(ctx, "e5-small-arm")
	if err != nil {
		t.Fatal(err)
	}
	if recovered.RecipeID != recipe.ID() || recovered.BuildID != "search-hub-build-9" || recovered.Evidence[0].SHA256 != strings.Repeat("a", 64) {
		t.Fatalf("restart or caller mutation changed immutable identity/evidence: %+v", recovered)
	}
	recoveredRetry, err := reopened.Transition(ctx, firstTransition)
	if err != nil || recoveredRetry.State != firstResult.State || recoveredRetry.Version != firstResult.Version {
		t.Fatalf("transition idempotency did not survive reopen: result=%+v err=%v", recoveredRetry, err)
	}
	if _, err := reopened.Transition(ctx, TransitionRequest{ArmID: created.ArmID, ExpectedState: StateCapabilityVerified, ExpectedVersion: 2, To: StatePlanned, IdempotencyKey: "advance-after-reopen", Receipt: testReceipt("plan", "plan-after-reopen")}); err != nil {
		t.Fatal(err)
	}
	recoveredRetry, err = reopened.Transition(ctx, firstTransition)
	if err != nil || recoveredRetry.State != firstResult.State || recoveredRetry.Version != firstResult.Version {
		t.Fatalf("old idempotent retry did not return its original receipt after later transition: result=%+v err=%v", recoveredRetry, err)
	}
}

func TestRegistryTransitionCASConflictAndIdempotentRetry(t *testing.T) {
	ctx := context.Background()
	db, _ := openRegistryDB(t)
	registry, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	arm := createTestArm(t, registry, ctx)
	request := TransitionRequest{ArmID: arm.ArmID, ExpectedState: StateDiscovered, ExpectedVersion: 1, To: StateCapabilityVerified, IdempotencyKey: "capability-check-1", Receipt: testReceipt("capability", "capability-1")}
	first, err := registry.Transition(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := registry.Transition(ctx, request)
	if err != nil || retry.State != first.State || retry.Version != first.Version {
		t.Fatalf("transition retry changed outcome: first=%+v retry=%+v err=%v", first, retry, err)
	}
	changed := request
	changed.To = StateFailed
	if _, err := registry.Transition(ctx, changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("expected idempotency key conflict, got %v", err)
	}
	stale := TransitionRequest{ArmID: arm.ArmID, ExpectedState: StateDiscovered, ExpectedVersion: 1, To: StateCapabilityVerified, IdempotencyKey: "different-key", Receipt: testReceipt("capability", "capability-1")}
	if _, err := registry.Transition(ctx, stale); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected stale-version conflict, got %v", err)
	}
	if _, err := registry.Transition(ctx, TransitionRequest{ArmID: arm.ArmID, ExpectedState: StateCapabilityVerified, ExpectedVersion: 2, To: StateActive, IdempotencyKey: "illegal"}); !errors.Is(err, ErrIllegalTransition) {
		t.Fatalf("expected illegal state transition rejection, got %v", err)
	}

	var gate sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		gate.Add(1)
		go func(index int) {
			defer gate.Done()
			<-start
			candidate := TransitionRequest{ArmID: arm.ArmID, ExpectedState: StateCapabilityVerified, ExpectedVersion: 2, To: StatePlanned, IdempotencyKey: fmt.Sprintf("plan-race-%d", index), Receipt: testReceipt("plan", fmt.Sprintf("plan-%d", index))}
			_, err := registry.Transition(ctx, candidate)
			errs <- err
		}(i)
	}
	close(start)
	gate.Wait()
	close(errs)
	passed, conflicted := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			passed++
		case errors.Is(err, ErrVersionConflict):
			conflicted++
		default:
			t.Fatalf("unexpected concurrent transition result: %v", err)
		}
	}
	if passed != 1 || conflicted != 1 {
		t.Fatalf("expected exactly one concurrent CAS winner and one conflict, passed=%d conflicted=%d", passed, conflicted)
	}
}

func TestRegistryRejectsTamperedRecipeAndEvidenceReference(t *testing.T) {
	ctx := context.Background()
	db, _ := openRegistryDB(t)
	registry, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	badEvidence := validArmCreate(validRecipeForTest(t))
	badEvidence.Evidence[0].SHA256 = "not-a-hash"
	if _, err := registry.Create(ctx, badEvidence); err == nil {
		t.Fatal("accepted malformed evidence digest")
	}
	arm := createTestArm(t, registry, ctx)
	_, err = db.ExecContext(ctx, `UPDATE embedding_experiment_arms SET recipe_canonical = ? WHERE arm_id = ?`, []byte(`{"model_id":"tampered"}`), arm.ArmID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Get(ctx, arm.ArmID); err == nil || !strings.Contains(err.Error(), "corrupt") {
		t.Fatalf("expected tampered persisted recipe rejection, got %v", err)
	}
}

func TestRegistryDetectsTamperedEvidenceReferences(t *testing.T) {
	ctx := context.Background()
	db, _ := openRegistryDB(t)
	registry, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	arm := createTestArm(t, registry, ctx)
	_, err = db.ExecContext(ctx, `UPDATE embedding_experiment_arms SET evidence_json = ? WHERE arm_id = ?`, []byte(`[ {"kind":"test-genie","uri":"test-genie://tampered","sha256":"`+strings.Repeat("a", 64)+`"} ]`), arm.ArmID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Get(ctx, arm.ArmID); err == nil || !strings.Contains(err.Error(), "evidence references are corrupt") {
		t.Fatalf("expected tampered evidence reference rejection, got %v", err)
	}
}

func TestRegistryRequiresLegalTransitionAndEvidenceIdentity(t *testing.T) {
	ctx := context.Background()
	db, _ := openRegistryDB(t)
	registry, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	create := validArmCreate(validRecipeForTest(t))
	create.Evidence = nil
	if _, err := registry.Create(ctx, create); err == nil {
		t.Fatal("accepted arm without source-evidence references")
	}
	if err := validateTransitionRequest(TransitionRequest{ArmID: "arm", ExpectedState: StateDiscovered, ExpectedVersion: 1, To: StateDeleted, IdempotencyKey: "bad"}); !errors.Is(err, ErrIllegalTransition) {
		t.Fatalf("expected illegal transition to be rejected, got %v", err)
	}
	if err := validateTransitionRequest(TransitionRequest{ArmID: "arm", ExpectedState: StateDiscovered, ExpectedVersion: 1, To: StateCapabilityVerified, IdempotencyKey: "no-receipt"}); err == nil || !strings.Contains(err.Error(), "receipt is required") {
		t.Fatalf("expected transition evidence requirement, got %v", err)
	}
}

func TestProviderGenerationRequestRejectsChangedRecipeAndMissingSnapshot(t *testing.T) {
	recipe, err := aisearch.NewEmbeddingRecipe(validRecipe())
	if err != nil {
		t.Fatal(err)
	}
	request := GenerationRequest{ArmID: "arm-1", ExperimentID: "trial-1", RecipeID: recipe.ID(), RecipeCanonical: recipe.Canonical(), SourceSnapshot: "snapshot-sha256:abc", Watermark: "source-sequence:42", IdempotencyKey: "stage-arm-1"}
	if err := request.Validate(); err != nil {
		t.Fatalf("valid provider generation request rejected: %v", err)
	}
	request.RecipeCanonical = append([]byte(nil), request.RecipeCanonical...)
	request.RecipeCanonical[0] = '!'
	if err := request.Validate(); err == nil {
		t.Fatal("adapter request accepted changed canonical recipe bytes")
	}
	request.RecipeCanonical = recipe.Canonical()
	request.SourceSnapshot = ""
	if err := request.Validate(); err == nil {
		t.Fatal("adapter request accepted missing source snapshot")
	}
}

func openRegistryDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "arms.sqlite")
	db, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(4)
	t.Cleanup(func() { _ = db.Close() })
	return db, path
}

func sqliteDSN(path string) string {
	return "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
}

func createTestArm(t *testing.T, registry *Registry, ctx context.Context) Arm {
	t.Helper()
	arm, err := registry.Create(ctx, validArmCreate(validRecipeForTest(t)))
	if err != nil {
		t.Fatal(err)
	}
	return arm
}

func validRecipeForTest(t *testing.T) aisearch.EmbeddingRecipe {
	t.Helper()
	recipe, err := aisearch.NewEmbeddingRecipe(validRecipe())
	if err != nil {
		t.Fatal(err)
	}
	return recipe
}

func validRecipe() aisearch.EmbeddingRecipeInput {
	return aisearch.EmbeddingRecipeInput{SchemaVersion: "embedding-recipe-v1", ModelID: "e5-small", ModelRevision: "revision-7", ArtifactSHA256: strings.Repeat("c", 64), TokenizerRevision: "tokenizer-7", Runtime: "onnx", RuntimeBuild: "ort-1.24", Precision: "fp32", QueryInstruction: "query: ", DocumentInstruction: "passage: ", Pooling: "mean", Normalization: "l2", Dimensions: 384, Projection: "none", OutputDType: "float32", OutputQuantization: "none", Truncation: "512-tokens", ChunkerVersion: "chunk-v4", ComposerVersion: "search-hub-v7"}
}

func validArmCreate(recipe aisearch.EmbeddingRecipe) ArmCreate {
	return ArmCreate{ExperimentID: "efficiency-2026-11", ArmID: "e5-small-arm", OwnerID: "search-hub", ProviderID: "provider-docs", CallerID: "search-hub", BuildID: "search-hub-build-9", Recipe: recipe, Evidence: []EvidenceRef{{Kind: "e5-source-run", URI: "test-genie://search-hub/20261111-103509-40f70848", SHA256: strings.Repeat("a", 64)}, {Kind: "w10-caller-build", URI: "source://w10-searchhub-build-identity", SHA256: strings.Repeat("d", 64)}}}
}

func testReceipt(kind, id string) EvidenceRef {
	return EvidenceRef{Kind: kind, URI: "test://" + id, SHA256: strings.Repeat("e", 64)}
}
