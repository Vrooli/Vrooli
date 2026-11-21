package notes

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type activeSemanticGeneration struct {
	config FleetGenerationConfig
	err    error
}

func (r activeSemanticGeneration) ResolveActiveSemanticGeneration(context.Context) (FleetGenerationConfig, error) {
	return r.config, r.err
}

type semanticAttributionFake struct {
	value SemanticAttribution
	err   error
}

func (f semanticAttributionFake) ResolveSemanticAttribution(context.Context) (SemanticAttribution, error) {
	return f.value, f.err
}

type semanticTenantFake struct{ tenant string }

func (f semanticTenantFake) ResolveSemanticTenant(_ context.Context, requested string) (string, error) {
	if requested != f.tenant {
		return "", errors.New("tenant outside authenticated test principal")
	}
	return f.tenant, nil
}

type semanticFleetStoreFake struct {
	query              SemanticFleetQuery
	upsert             SemanticFleetUpsert
	delete             SemanticFleetDelete
	queryHits          []SemanticSearchHit
	queryResult        SemanticFleetQueryResult
	operationResult    SemanticFleetOperationResult
	queries            int
	upserts            int
	deletes            int
	upsertOperationIDs []string
	deleteOperationIDs []string
	err                error
}

func (f *semanticFleetStoreFake) QueryActive(_ context.Context, request SemanticFleetQuery) (SemanticFleetQueryResult, error) {
	f.queries++
	f.query = request
	if f.err != nil {
		return SemanticFleetQueryResult{}, f.err
	}
	if f.queryResult.Results != nil {
		return f.queryResult, nil
	}
	return SemanticFleetQueryResult{Results: f.queryHits}, nil
}

func (f *semanticFleetStoreFake) UpsertActive(_ context.Context, request SemanticFleetUpsert) (SemanticFleetOperationResult, error) {
	f.upserts++
	f.upsert = request
	f.upsertOperationIDs = append(f.upsertOperationIDs, request.OperationID)
	return f.operationResult, f.err
}

func (f *semanticFleetStoreFake) DeleteActive(_ context.Context, request SemanticFleetDelete) (SemanticFleetOperationResult, error) {
	f.deletes++
	f.delete = request
	f.deleteOperationIDs = append(f.deleteOperationIDs, request.OperationID)
	return f.operationResult, f.err
}

type activeFleetCatalogFake struct {
	item    CatalogIdentity
	found   bool
	err     error
	ownerID string
	alias   string
}

func (f *activeFleetCatalogFake) LoadActive(_ context.Context, ownerID, alias string) (CatalogIdentity, bool, error) {
	f.ownerID, f.alias = ownerID, alias
	return f.item, f.found, f.err
}

func validSemanticAttribution() SemanticAttribution {
	return SemanticAttribution{
		CallerID: "notes-api", RightsEvidenceURI: "evidence://rights/notes-v1",
		RightsEvidenceSHA256:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		QualityEvidenceURI:    "evidence://quality/notes-v1",
		QualityEvidenceSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
}

func TestSemanticServingDelegatesTenantQueryAndCanonicalSourceIdentity(t *testing.T) {
	config := FleetGenerationConfig{GenerationID: "arm-8", OwnerID: "notes.api", Alias: "notes-live", Collection: "notes-generation-8", RecipeID: "recipe-8", EmbeddingRole: "embedding.notes.v8", EmbeddingDimensions: 384}
	generation := SemanticFleetGeneration{FleetGenerationConfig: config, Version: 9, QdrantDiskBytesKnown: false}
	receipt := SemanticFleetAdmissionReceipt{Operation: "query", OperationID: "query-operation", ReceiptID: "receipt-query", OwnerID: config.OwnerID, Alias: config.Alias, Collection: config.Collection, TenantID: "tenant-a", ArmID: config.GenerationID, RecipeID: config.RecipeID, RuntimeRole: config.EmbeddingRole, GenerationVersion: 9, Dimensions: config.EmbeddingDimensions, SourceCount: 1, PointCount: 0, RequestBytes: 5, RequestSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", PeakBytes: 2048, Attribution: validSemanticAttribution()}
	store := &semanticFleetStoreFake{
		queryHits:       []SemanticSearchHit{{ID: "point", Score: .9}},
		queryResult:     SemanticFleetQueryResult{Generation: generation, Admission: receipt, Results: []SemanticSearchHit{{ID: "point", Score: .9}}},
		operationResult: SemanticFleetOperationResult{Generation: generation, Admission: SemanticFleetAdmissionReceipt{ReceiptID: "receipt-upsert"}},
	}
	service := &SemanticServing{Generations: activeSemanticGeneration{config: config}, Tenants: semanticTenantFake{tenant: "tenant-a"}, Attribution: semanticAttributionFake{value: validSemanticAttribution()}, Store: store, Mutations: store}
	hits, err := service.Search(context.Background(), " query ", "tenant-a", 7)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(hits.Results) != 1 || store.queries != 1 || store.query.TenantID != "tenant-a" || store.query.Text != "query" || store.query.Limit != 7 || store.query.ArmID != config.GenerationID || store.query.RecipeID != config.RecipeID {
		t.Fatalf("semantic query did not preserve tenant and active generation: hits=%v request=%+v", hits, store.query)
	}
	if store.query.OperationID == "" || store.query.Attribution != validSemanticAttribution() || hits.Generation != generation || !reflect.DeepEqual(hits.Admission, receipt) {
		t.Fatalf("semantic query missing operation identity or trusted evidence: %+v", store.query)
	}

	document := FleetDocument{NoteID: " note-1 ", UserID: " tenant-a ", Title: "Title", Content: "Body", IsPinned: true}
	upsertResult, err := service.UpsertNote(context.Background(), document, "notes-change:19:upsert", 19)
	if err != nil {
		t.Fatalf("UpsertNote() error = %v", err)
	}
	if upsertResult.Generation != generation || upsertResult.Admission.ReceiptID != "receipt-upsert" {
		t.Fatalf("Notes facade discarded returned admission receipt: %+v", upsertResult)
	}
	if store.upserts != 1 || store.upsert.TenantID != "tenant-a" || store.upsert.SourceID != fleetSourceID("tenant-a", "note-1") || store.upsert.Document.Payload["user_id"] != "tenant-a" || store.upsert.Document.Payload["note_id"] != "note-1" || store.upsert.Document.Text != "Title\n\nBody" {
		t.Fatalf("upsert lost canonical source identity or Notes content: %+v", store.upsert)
	}
	deleteResult, err := service.DeleteNote(context.Background(), "tenant-a", "note-1", "notes-change:20:delete", 20)
	if err != nil {
		t.Fatalf("DeleteNote() error = %v", err)
	}
	if deleteResult.Admission.ReceiptID != "receipt-upsert" {
		t.Fatalf("Notes facade discarded returned delete receipt: %+v", deleteResult)
	}
	if store.deletes != 1 || store.delete.TenantID != "tenant-a" || len(store.delete.Sources) != 1 || store.delete.Sources[0].SourceID != store.upsert.SourceID || store.delete.Sources[0].SourceSequence != 20 || store.delete.ArmID != store.upsert.ArmID || store.delete.RecipeID != store.upsert.RecipeID {
		t.Fatalf("delete did not preserve the exact tenant source and generation identity: upsert=%+v delete=%+v", store.upsert, store.delete)
	}
	if store.upsertOperationIDs[0] != "notes-change:19:upsert" || store.deleteOperationIDs[0] != "notes-change:20:delete" {
		t.Fatalf("serving changed durable idempotency keys: upsert=%v delete=%v", store.upsertOperationIDs, store.deleteOperationIDs)
	}
	if store.upsert.SourceSequence != 19 {
		t.Fatalf("upsert sequence = %d, want 19", store.upsert.SourceSequence)
	}
}

func TestSemanticServingRejectsEmptyContentBeforeOwnerAndNormalizesIdentity(t *testing.T) {
	config := FleetGenerationConfig{GenerationID: "arm-8", OwnerID: "notes.api", Alias: "notes-live", Collection: "notes-generation-8", RecipeID: "recipe-8", EmbeddingRole: "embedding.notes.v8", EmbeddingDimensions: 384}
	store := &semanticFleetStoreFake{}
	service := &SemanticServing{Generations: activeSemanticGeneration{config: config}, Tenants: semanticTenantFake{tenant: "tenant-a"}, Attribution: semanticAttributionFake{value: validSemanticAttribution()}, Store: store, Mutations: store}
	if _, err := service.UpsertNote(context.Background(), FleetDocument{NoteID: "note-1", UserID: "tenant-a", Title: "  ", Content: "\n"}, "change-1", 1); err == nil {
		t.Fatal("empty searchable content reached semantic store")
	}
	if store.upserts != 0 {
		t.Fatalf("empty content reached semantic store: %d", store.upserts)
	}
	if _, err := service.UpsertNote(context.Background(), FleetDocument{NoteID: " note-1 ", UserID: " tenant-a ", Content: "body"}, "change-2", 2); err != nil {
		t.Fatalf("UpsertNote() with padded IDs error = %v", err)
	}
	if store.upsert.TenantID != "tenant-a" || store.upsert.Document.Payload["user_id"] != "tenant-a" || store.upsert.SourceID != fleetSourceID("tenant-a", "note-1") {
		t.Fatalf("upsert did not canonicalize tenant and note IDs consistently: %+v", store.upsert)
	}
}

func TestSemanticServingRejectsTenantOutsideAuthenticatedPrincipal(t *testing.T) {
	config := FleetGenerationConfig{GenerationID: "arm-8", OwnerID: "notes.api", Alias: "notes-live", Collection: "notes-generation-8", RecipeID: "recipe-8", EmbeddingRole: "embedding.notes.v8", EmbeddingDimensions: 384}
	store := &semanticFleetStoreFake{}
	service := &SemanticServing{Generations: activeSemanticGeneration{config: config}, Tenants: semanticTenantFake{tenant: "tenant-a"}, Attribution: semanticAttributionFake{value: validSemanticAttribution()}, Store: store, Mutations: store}
	if _, err := service.Search(context.Background(), "query", "tenant-b", 5); err == nil {
		t.Fatal("semantic search accepted a tenant outside the authenticated principal")
	}
	if store.queries != 0 {
		t.Fatalf("unauthorized tenant query reached shared store: %d", store.queries)
	}
	if _, err := service.UpsertNote(context.Background(), FleetDocument{NoteID: "note-1", UserID: "tenant-b", Content: "body"}, "notes-change:1:upsert", 1); err == nil {
		t.Fatal("semantic upsert accepted a tenant outside the authenticated principal")
	}
	if _, err := service.DeleteNote(context.Background(), "tenant-b", "note-1", "notes-change:2:delete", 2); err == nil {
		t.Fatal("semantic delete accepted a tenant outside the authenticated principal")
	}
	if store.upserts != 0 || store.deletes != 0 {
		t.Fatalf("unauthorized tenant mutation reached shared store: upsert=%d delete=%d", store.upserts, store.deletes)
	}
}

func TestSemanticServingMutationRoutesStayDisabledWithoutOrderedOwner(t *testing.T) {
	config := FleetGenerationConfig{GenerationID: "arm-8", OwnerID: "notes.api", Alias: "notes-live", Collection: "notes-generation-8", RecipeID: "recipe-8", EmbeddingRole: "embedding.notes.v8", EmbeddingDimensions: 384}
	store := &semanticFleetStoreFake{}
	service := &SemanticServing{Generations: activeSemanticGeneration{config: config}, Tenants: semanticTenantFake{tenant: "tenant-a"}, Attribution: semanticAttributionFake{value: validSemanticAttribution()}, Store: store}
	if _, err := service.UpsertNote(context.Background(), FleetDocument{NoteID: "note-1", UserID: "tenant-a", Content: "body"}, "notes-fleet:upsert:3", 3); err == nil {
		t.Fatal("mutation route ran without the ordered source-sequence owner")
	}
	if _, err := service.DeleteNote(context.Background(), "tenant-a", "note-1", "notes-fleet:delete:4", 4); err == nil {
		t.Fatal("delete route ran without the ordered source-sequence owner")
	}
	if store.upserts != 0 || store.deletes != 0 {
		t.Fatalf("disabled mutation route reached store: upsert=%d delete=%d", store.upserts, store.deletes)
	}
}

func TestSemanticServingFailsClosedWithoutTrustedAttributionOrStore(t *testing.T) {
	config := FleetGenerationConfig{GenerationID: "arm-8", OwnerID: "notes.api", Alias: "notes-live", Collection: "notes-generation-8", RecipeID: "recipe-8", EmbeddingRole: "embedding.notes.v8", EmbeddingDimensions: 384}
	store := &semanticFleetStoreFake{}
	service := &SemanticServing{Generations: activeSemanticGeneration{config: config}, Tenants: semanticTenantFake{tenant: "tenant-a"}, Attribution: semanticAttributionFake{value: SemanticAttribution{CallerID: "notes-api"}}, Store: store, Mutations: store}
	if _, err := service.Search(context.Background(), "query", "tenant-a", 5); err == nil {
		t.Fatal("Search() succeeded without exact rights and quality evidence")
	}
	if store.queries != 0 {
		t.Fatalf("unattributed request reached semantic store: queries=%d", store.queries)
	}
	service.Attribution = nil
	if _, err := service.UpsertNote(context.Background(), FleetDocument{NoteID: "note-1", UserID: "tenant-a", Content: "body"}, "change-3", 3); err == nil {
		t.Fatal("UpsertNote() succeeded without an attribution resolver")
	}
	if store.upserts != 0 {
		t.Fatalf("missing-attribution upsert reached store: %d", store.upserts)
	}
	service.Attribution = semanticAttributionFake{value: validSemanticAttribution()}
	service.Store = nil
	if _, err := service.DeleteNote(context.Background(), "tenant-a", "note-1", "change-4", 4); err == nil {
		t.Fatal("DeleteNote() succeeded without an admitted semantic store")
	}
}

func TestSemanticServingPropagatesOwnerAndAttributionFailures(t *testing.T) {
	config := FleetGenerationConfig{GenerationID: "arm-8", OwnerID: "notes.api", Alias: "notes-live", Collection: "notes-generation-8", RecipeID: "recipe-8", EmbeddingRole: "embedding.notes.v8", EmbeddingDimensions: 384}
	store := &semanticFleetStoreFake{err: errors.New("owner rejected query")}
	service := &SemanticServing{Generations: activeSemanticGeneration{config: config}, Tenants: semanticTenantFake{tenant: "tenant-a"}, Attribution: semanticAttributionFake{value: validSemanticAttribution()}, Store: store, Mutations: store}
	if _, err := service.Search(context.Background(), "query", "tenant-a", 5); err == nil {
		t.Fatal("Search() hid owner failure")
	}
	service.Attribution = semanticAttributionFake{err: errors.New("rights context unavailable")}
	if _, err := service.UpsertNote(context.Background(), FleetDocument{NoteID: "note-1", UserID: "tenant-a", Content: "body"}, "change-5", 5); err == nil {
		t.Fatal("UpsertNote() hid attribution resolution failure")
	}
	if store.upserts != 0 {
		t.Fatalf("attribution failure reached store: %d", store.upserts)
	}
}

func TestActiveSemanticGenerationResolverRequiresProtectedCatalogRecipe(t *testing.T) {
	item := CatalogIdentity{ArmID: "arm-8", OwnerID: "notes.api", Alias: "notes-live", Collection: "notes-generation-8", RecipeID: "recipe-8", Dimensions: 384, State: "active", Protected: true}
	catalog := &activeFleetCatalogFake{item: item, found: true}
	policy := fixedPolicy{policy: EmbeddingRolePolicy{RecipeID: item.RecipeID, Role: "embedding.notes.v8", Dimensions: 384}}
	resolver := CatalogActiveSemanticGenerationResolver{Catalog: catalog, Policies: policy, OwnerID: item.OwnerID, Alias: item.Alias}
	config, err := resolver.ResolveActiveSemanticGeneration(context.Background())
	if err != nil {
		t.Fatalf("ResolveActiveSemanticGeneration() error = %v", err)
	}
	if catalog.ownerID != item.OwnerID || catalog.alias != item.Alias || config.Collection != item.Collection || config.RecipeID != item.RecipeID || config.EmbeddingRole != "embedding.notes.v8" || config.EmbeddingDimensions != 384 {
		t.Fatalf("resolver did not preserve protected active identity: config=%+v catalog=%+v", config, catalog)
	}

	item.Protected = false
	catalog.item = item
	if _, err := resolver.ResolveActiveSemanticGeneration(context.Background()); err == nil {
		t.Fatal("unprotected active generation was accepted for semantic serving")
	}
}
