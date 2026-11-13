package experimentarms

import (
	"context"
	"database/sql"
	"math"
	"sync"
	"testing"
	"time"

	aisearch "github.com/vrooli/ai-go/search"
	corestorage "github.com/vrooli/api-core/storage"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	"google.golang.org/protobuf/encoding/protojson"

	ownerregistry "search-hub/internal/registry"
)

type providerFleetTestEncoder struct {
	recipe     aisearch.EmbeddingRecipe
	firstValue float64
}

func (e providerFleetTestEncoder) Recipe() aisearch.EmbeddingRecipe { return e.recipe }
func (providerFleetTestEncoder) Available(context.Context) bool     { return true }
func (e providerFleetTestEncoder) Embed(ctx context.Context, text string) ([]float64, error) {
	return e.EmbedDocument(ctx, text)
}
func (e providerFleetTestEncoder) EmbedQuery(ctx context.Context, text string) ([]float64, error) {
	return e.EmbedDocument(ctx, text)
}
func (e providerFleetTestEncoder) EmbedDocument(_ context.Context, text string) ([]float64, error) {
	input, err := e.recipe.Input()
	if err != nil {
		return nil, err
	}
	vector := make([]float64, input.Dimensions)
	if text != "" {
		value := e.firstValue
		if value == 0 {
			value = 1
		}
		vector[int(text[0])%len(vector)] = value
	}
	return vector, nil
}

func TestProviderDescriptionFleetSourceSnapshotCatchUpAndTombstones(t *testing.T) {
	ctx := context.Background()
	db, _ := openRegistryDB(t)
	if _, err := db.ExecContext(ctx, ownerregistry.Schema()); err != nil {
		t.Fatal(err)
	}
	registry, err := NewForNamespace(db, mustSearchHubNamespace(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	recipe := validRecipeForTest(t)
	unbound, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := unbound.NewProviderDescriptionFleetSource(providerFleetTestEncoder{recipe: recipe}); err == nil {
		t.Fatal("source construction must reject a registry without a trusted startup namespace")
	}
	source, err := registry.NewProviderDescriptionFleetSource(providerFleetTestEncoder{recipe: recipe})
	if err != nil {
		t.Fatal(err)
	}
	first := providerFleetDescriptor("docs.alpha", "Alpha docs", []string{"answer alpha", "intent alpha", "example alpha"})
	second := providerFleetDescriptor("docs.beta", "Beta docs", []string{"answer beta"})
	insertProviderDescriptor(t, db, first)
	insertProviderDescriptor(t, db, second)

	snapshot, err := source.CreateFleetSnapshot(ctx, recipe.ID())
	if err != nil {
		t.Fatal(err)
	}
	armSnapshot, err := source.CreateFleetSnapshotForArm(ctx, "arm-a", recipe.ID())
	if err != nil {
		t.Fatal(err)
	}
	armSnapshotRetry, err := source.CreateFleetSnapshotForArm(ctx, "arm-a", recipe.ID())
	if err != nil || armSnapshotRetry != armSnapshot {
		t.Fatalf("same arm/recipe retry must pin the same source snapshot: first=%+v retry=%+v err=%v", armSnapshot, armSnapshotRetry, err)
	}
	pinned, err := source.PinFleetSnapshot(ctx, snapshot.ID)
	if err != nil || pinned != snapshot {
		t.Fatalf("snapshot must survive source reconstruction: pinned=%+v snapshot=%+v err=%v", pinned, snapshot, err)
	}
	points1, cursor, done, err := source.ReadFleetSnapshotPage(ctx, recipe.ID(), snapshot, "", 2)
	if err != nil || done || len(points1) != 2 || cursor != "2" {
		t.Fatalf("first snapshot page points=%d cursor=%q done=%v err=%v", len(points1), cursor, done, err)
	}
	for _, point := range points1 {
		if point.Payload["source_id"] == "" || point.Payload["embedding_recipe_id"] != recipe.ID() || len(point.Vector) != 384 {
			t.Fatalf("fleet point lost stable owner/recipe/vector identity: %+v", point)
		}
	}
	points2, cursor, done, err := source.ReadFleetSnapshotPage(ctx, recipe.ID(), snapshot, cursor, 2)
	if err != nil || !done || len(points2) != 2 || cursor != "4" {
		t.Fatalf("final snapshot page points=%d cursor=%q done=%v err=%v", len(points2), cursor, done, err)
	}

	updated := providerFleetDescriptor("docs.alpha", "Alpha operations", []string{"new alpha intent"})
	updateProviderDescriptor(t, db, updated)
	if _, err := db.ExecContext(ctx, `DELETE FROM providers WHERE provider_id = ?`, second.GetProviderId()); err != nil {
		t.Fatal(err)
	}
	watermark, err := source.FleetWatermark(ctx)
	if err != nil || watermark == snapshot.Watermark {
		t.Fatalf("owner changes did not advance watermark: before=%q after=%q err=%v", snapshot.Watermark, watermark, err)
	}
	changes, next, err := source.ReadFleetChanges(ctx, recipe.ID(), snapshot.Watermark, snapshot.Watermark, 8)
	if err != nil || len(changes) != 2 || next != watermark {
		t.Fatalf("changes=%+v next=%q watermark=%q err=%v", changes, next, watermark, err)
	}
	if len(changes[0].Deletes) != 3 || len(changes[0].Upserts) != 1 {
		t.Fatalf("descriptor change must tombstone old segments before its new upsert: %+v", changes[0])
	}
	if len(changes[1].Deletes) != 1 || len(changes[1].Upserts) != 0 {
		t.Fatalf("provider deletion must emit a stable tombstone: %+v", changes[1])
	}
	count, err := source.ExpectedFleetPointCount(ctx, recipe.ID(), snapshot.ID, watermark)
	if err != nil || count != 1 {
		t.Fatalf("expected point count after update/delete=%d err=%v, want one current alpha segment", count, err)
	}

	fence, err := source.AcquireFleetCutover(ctx, watermark)
	if err != nil {
		t.Fatal(err)
	}
	if current, err := fence.CurrentWatermark(ctx); err != nil || current != watermark {
		t.Fatalf("fenced watermark=%q want=%q err=%v", current, watermark, err)
	}
	blocked := make(chan struct{})
	var once sync.Once
	go func() {
		release, err := ownerregistry.AcquireProviderDescriptionWrite(ctx)
		if err == nil {
			release()
		}
		once.Do(func() { close(blocked) })
	}()
	select {
	case <-blocked:
		t.Fatal("registry writer entered the exclusive source cutover window")
	case <-time.After(20 * time.Millisecond):
	}
	if err := fence.Release(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-blocked:
	case <-time.After(time.Second):
		t.Fatal("registry writer did not resume after source cutover release")
	}
	badEncoderSource, err := registry.NewProviderDescriptionFleetSource(providerFleetTestEncoder{recipe: recipe, firstValue: math.MaxFloat64})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := badEncoderSource.ReadFleetSnapshotPage(ctx, recipe.ID(), snapshot, "", 1); err == nil {
		t.Fatal("source must reject embedding values that overflow Qdrant's float32 point vectors")
	}
	if _, err := db.ExecContext(ctx, `UPDATE provider_description_fleet_snapshot_profiles SET profile_json = ? WHERE snapshot_id = ? AND provider_id = ?`, `{"ProviderID":"docs.alpha","Type":"doc"}`, snapshot.ID, "docs.alpha"); err != nil {
		t.Fatal(err)
	}
	if _, err := source.PinFleetSnapshot(ctx, snapshot.ID); err == nil {
		t.Fatal("snapshot pin must detect profile mutation against its persisted digest")
	}
}

func mustSearchHubNamespace(t *testing.T) corestorage.Namespace {
	t.Helper()
	namespace, err := corestorage.ResolveNamespace(corestorage.NamespaceConfig{FallbackScenario: "search-hub"})
	if err != nil {
		t.Fatal(err)
	}
	return namespace
}

func providerFleetDescriptor(id, description string, intents []string) *registryv1.ProviderDescriptor {
	return &registryv1.ProviderDescriptor{
		ProviderId: id, ProviderGroup: "docs", Type: "doc", Description: description,
		State:          registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
		Endpoint:       &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{ScenarioId: "docs", Path: "/query", Method: registryv1.HttpMethod_HTTP_METHOD_POST}}},
		RoutingProfile: &registryv1.RoutingProfile{AnswerSpaces: intents},
	}
}

func insertProviderDescriptor(t *testing.T, db *sql.DB, descriptor *registryv1.ProviderDescriptor) {
	t.Helper()
	raw, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO providers(provider_id, provider_group, bucket, type, state, scope, descriptor, control_token, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?, '', '', '')`, descriptor.GetProviderId(), descriptor.GetProviderGroup(), descriptor.GetBucket(), descriptor.GetType(), descriptor.GetState(), descriptor.GetScope(), string(raw)); err != nil {
		t.Fatal(err)
	}
}

func updateProviderDescriptor(t *testing.T, db *sql.DB, descriptor *registryv1.ProviderDescriptor) {
	t.Helper()
	raw, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE providers SET provider_group = ?, type = ?, state = ?, scope = ?, descriptor = ? WHERE provider_id = ?`, descriptor.GetProviderGroup(), descriptor.GetType(), descriptor.GetState(), descriptor.GetScope(), string(raw), descriptor.GetProviderId()); err != nil {
		t.Fatal(err)
	}
}
