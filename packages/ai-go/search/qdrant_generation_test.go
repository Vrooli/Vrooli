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

// fakeQdrant is the smallest Qdrant surface the generation store touches:
// collection create/list/inspect/delete, aliases, and the point endpoints the
// meta sentinel and validation call. It records every request so tests can
// assert on the wire behavior (one delete per retired generation, init_from on
// the candidate create, no client-side scroll of the active generation).
type fakeQdrant struct {
	mu          sync.Mutex
	collections map[string]*fakeCollection
	aliases     map[string]string // alias -> collection
	deletes     []string
	creates     map[string]string // collection -> init_from source
	scrolls     map[string]int    // collection -> scroll requests
	failDelete  map[string]bool
	// shortSeed makes init_from copies land one point short, so a seeded
	// candidate never reaches the active generation's count.
	shortSeed bool
}

type fakeCollection struct {
	points int
}

type generationCatalogFixture struct {
	mu      sync.Mutex
	records map[string]GenerationRecord
	fail    func(GenerationRecord) error
}

func (c *generationCatalogFixture) ListGenerationRecords(_ context.Context, _, _ string) ([]GenerationRecord, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]GenerationRecord, 0, len(c.records))
	for _, record := range c.records {
		out = append(out, record)
	}
	return out, nil
}

func (c *generationCatalogFixture) SaveGenerationRecord(_ context.Context, record GenerationRecord) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail != nil {
		if err := c.fail(record); err != nil {
			return err
		}
	}
	if c.records == nil {
		c.records = map[string]GenerationRecord{}
	}
	c.records[record.Metadata.ID] = record
	return nil
}

func (c *generationCatalogFixture) record(id string) GenerationRecord {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.records[id]
}

func makeCatalogGenerationStore(t *testing.T, serverURL, alias string, fake *fakeQdrant, catalog GenerationCatalog) *qdrantGenerationStore {
	t.Helper()
	owner := strings.SplitN(alias, "_", 2)[0] + ".test"
	store, err := NewQdrantGenerationStore(QdrantGenerationOptions{BaseURL: serverURL, Alias: alias, Owner: owner, Catalog: catalog, AliasBoundary: &fleetTestAliasBoundary{}, Spec: CollectionSpec{DenseSize: 4, DenseDistance: "Cosine"}})
	if err != nil {
		t.Fatal(err)
	}
	return store.(*qdrantGenerationStore)
}

func newFakeQdrant() *fakeQdrant {
	return &fakeQdrant{collections: map[string]*fakeCollection{}, aliases: map[string]string{}, creates: map[string]string{}, scrolls: map[string]int{}, failDelete: map[string]bool{}}
}

func (f *fakeQdrant) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/aliases", func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		aliases := make([]map[string]string, 0, len(f.aliases))
		for alias, collection := range f.aliases {
			aliases = append(aliases, map[string]string{"alias_name": alias, "collection_name": collection})
		}
		writeJSONResponse(w, map[string]any{"result": map[string]any{"aliases": aliases}, "status": "ok"})
	})
	mux.HandleFunc("/collections/aliases", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Actions []map[string]map[string]string `json:"actions"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		defer f.mu.Unlock()
		for _, action := range body.Actions {
			if del, ok := action["delete_alias"]; ok {
				delete(f.aliases, del["alias_name"])
			}
			if create, ok := action["create_alias"]; ok {
				f.aliases[create["alias_name"]] = create["collection_name"]
			}
		}
		writeJSONResponse(w, map[string]any{"result": true, "status": "ok"})
	})
	mux.HandleFunc("/collections", func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		names := make([]map[string]string, 0, len(f.collections))
		for name := range f.collections {
			names = append(names, map[string]string{"name": name})
		}
		writeJSONResponse(w, map[string]any{"result": map[string]any{"collections": names}, "status": "ok"})
	})
	mux.HandleFunc("/collections/", func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/collections/")
		name, suffix, _ := strings.Cut(rest, "/")
		f.mu.Lock()
		defer f.mu.Unlock()
		switch {
		case suffix == "" && r.Method == http.MethodGet:
			collection, ok := f.collections[name]
			if !ok {
				http.Error(w, `{"status":{"error":"not found"}}`, http.StatusNotFound)
				return
			}
			writeJSONResponse(w, map[string]any{"result": map[string]any{
				"points_count": collection.points,
				"config":       map[string]any{"params": map[string]any{"vectors": map[string]any{"dense": map[string]any{"size": 4, "distance": "Cosine"}}}},
			}, "status": "ok"})
		case suffix == "" && r.Method == http.MethodPut:
			var body struct {
				InitFrom *struct {
					Collection string `json:"collection"`
				} `json:"init_from"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			created := &fakeCollection{}
			source := ""
			if body.InitFrom != nil {
				source = body.InitFrom.Collection
				if from, ok := f.collections[source]; ok {
					created.points = from.points
					if f.shortSeed {
						created.points--
					}
				}
			}
			f.collections[name] = created
			f.creates[name] = source
			writeJSONResponse(w, map[string]any{"result": true, "status": "ok"})
		case suffix == "" && r.Method == http.MethodDelete:
			if f.failDelete[name] {
				http.Error(w, `{"status":{"error":"busy"}}`, http.StatusInternalServerError)
				return
			}
			delete(f.collections, name)
			f.deletes = append(f.deletes, name)
			writeJSONResponse(w, map[string]any{"result": true, "status": "ok"})
		case suffix == "points/scroll":
			f.scrolls[name]++
			writeJSONResponse(w, map[string]any{"result": map[string]any{"points": []any{}}, "status": "ok"})
		default:
			// Meta sentinel upsert, filtered deletes, point fetches: accept.
			writeJSONResponse(w, map[string]any{"result": map[string]any{"status": "completed", "points": []any{}}, "status": "ok"})
		}
	})
	return mux
}

func writeJSONResponse(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

func newTestGenerationStore(t *testing.T, fake *fakeQdrant, alias string) (GenerationStore, *qdrantGenerationStore) {
	t.Helper()
	server := httptest.NewServer(fake.handler())
	t.Cleanup(server.Close)
	owner := strings.SplitN(alias, "_", 2)[0] + ".test"
	store, err := NewQdrantGenerationStore(QdrantGenerationOptions{BaseURL: server.URL, Alias: alias, Owner: owner, AliasBoundary: &fleetTestAliasBoundary{}, Spec: CollectionSpec{DenseSize: 4, DenseDistance: "Cosine"}})
	if err != nil {
		t.Fatal(err)
	}
	return store, store.(*qdrantGenerationStore)
}

func TestQdrantGenerationStoreSeedsIncrementalCandidateServerSide(t *testing.T) {
	fake := newFakeQdrant()
	const alias = "fixture_search"
	active := generationCollectionName(alias, "gen-1")
	fake.collections[active] = &fakeCollection{points: 3}
	fake.aliases[alias] = active
	store, _ := newTestGenerationStore(t, fake, alias)

	if err := store.BeginGeneration(context.Background(), GenerationMetadata{ID: "gen-2"}); err != nil {
		t.Fatal(err)
	}
	candidate := generationCollectionName(alias, "gen-2")
	if got := fake.creates[candidate]; got != active {
		t.Fatalf("candidate must be seeded with init_from=%q, got %q", active, got)
	}
	if fake.collections[candidate].points != 3 {
		t.Fatalf("seeded candidate holds %d points, want 3", fake.collections[candidate].points)
	}
	if fake.scrolls[active] != 0 {
		t.Fatalf("active generation was scrolled %d times; the server-side seed must not move points through the client", fake.scrolls[active])
	}
}

func TestQdrantGenerationStoreFullGenerationStartsEmpty(t *testing.T) {
	fake := newFakeQdrant()
	const alias = "fixture_search"
	active := generationCollectionName(alias, "gen-1")
	fake.collections[active] = &fakeCollection{points: 3}
	fake.aliases[alias] = active
	store, _ := newTestGenerationStore(t, fake, alias)

	if err := store.BeginGeneration(context.Background(), GenerationMetadata{ID: "gen-2", Full: true}); err != nil {
		t.Fatal(err)
	}
	candidate := generationCollectionName(alias, "gen-2")
	if got := fake.creates[candidate]; got != "" {
		t.Fatalf("full generation must not be seeded, got init_from=%q", got)
	}
}

func TestQdrantGenerationStoreCleanupRetiresBeyondKeep(t *testing.T) {
	fake := newFakeQdrant()
	const alias = "fixture_search"
	for _, id := range []string{"gen-1", "gen-2", "gen-3", "gen-4", "gen-5"} {
		fake.collections[generationCollectionName(alias, id)] = &fakeCollection{points: 1}
	}
	fake.collections["unrelated_collection"] = &fakeCollection{points: 9}
	fake.collections["fixture_search_other--generation--gen-0"] = &fakeCollection{points: 9}
	fake.aliases[alias] = generationCollectionName(alias, "gen-5")
	store, _ := newTestGenerationStore(t, fake, alias)

	// An in-flight candidate is never retired, even though it carries the
	// generation prefix and is not the alias target.
	if err := store.BeginGeneration(context.Background(), GenerationMetadata{ID: "gen-9", Full: true}); err != nil {
		t.Fatal(err)
	}
	if err := store.CleanupGenerations(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	want := []string{generationCollectionName(alias, "gen-2"), generationCollectionName(alias, "gen-1")}
	if strings.Join(fake.deletes, ",") != strings.Join(want, ",") {
		t.Fatalf("deleted %v, want newest-first retirement beyond keep=2: %v", fake.deletes, want)
	}
	for _, kept := range []string{generationCollectionName(alias, "gen-5"), generationCollectionName(alias, "gen-9"), "unrelated_collection", "fixture_search_other--generation--gen-0"} {
		if _, ok := fake.collections[kept]; !ok {
			t.Fatalf("%q must survive cleanup", kept)
		}
	}

	// Once promoted, the previous alias target becomes retired and the newest
	// keep survive: gen-5 and gen-4 stay, gen-3 goes.
	if err := store.PromoteGeneration(context.Background(), "gen-9"); err != nil {
		t.Fatal(err)
	}
	fake.deletes = nil
	if err := store.CleanupGenerations(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(fake.deletes, ","); got != generationCollectionName(alias, "gen-3") {
		t.Fatalf("after promotion deleted %q, want only gen-3", got)
	}
	if fake.aliases[alias] != generationCollectionName(alias, "gen-9") {
		t.Fatalf("alias must point at the promoted candidate, got %q", fake.aliases[alias])
	}
}

func TestQdrantGenerationStoreRollbackRestoresPreviousAliasAfterReconstruction(t *testing.T) {
	fake := newFakeQdrant()
	const alias = "fixture_search"
	previous := generationCollectionName(alias, "gen-1")
	candidate := generationCollectionName(alias, "gen-2")
	fake.collections[previous] = &fakeCollection{points: 1}
	fake.collections[candidate] = &fakeCollection{points: 2}
	fake.aliases[alias] = previous
	catalog := &generationCatalogFixture{records: map[string]GenerationRecord{
		"gen-1": {Metadata: GenerationMetadata{ID: "gen-1", Owner: "fixture.test", Namespace: alias, Alias: alias, Model: "nomic-embed-text:latest", EmbeddingRole: "embedding.default", EmbeddingDimensions: 768, PolicySchemaVersion: "policy/v4", MigrationPhase: "active"}, CollectionName: previous, State: GenerationStateActive},
		"gen-2": {Metadata: GenerationMetadata{ID: "gen-2", Owner: "fixture.test", Namespace: alias, Alias: alias}, CollectionName: candidate, State: GenerationStateCandidate},
	}}
	server := httptest.NewServer(fake.handler())
	defer server.Close()
	store := makeCatalogGenerationStore(t, server.URL, alias, fake, catalog)

	// Simulate process death after Qdrant switched aliases but before the
	// catalog's active-state commit succeeded.
	catalog.fail = func(record GenerationRecord) error {
		if record.Metadata.ID == "gen-2" && record.State == GenerationStateActive {
			return errors.New("injected post-switch catalog failure")
		}
		return nil
	}
	if err := store.PromoteGeneration(context.Background(), "gen-2"); err == nil {
		t.Fatal("promotion should surface the interrupted catalog commit")
	}
	if got := catalog.record("gen-2").PreviousCollection; got != previous {
		t.Fatalf("promotion intent stored previous target %q, want %q", got, previous)
	}
	retired := catalog.record("gen-1")
	if retired.Metadata.Model != "nomic-embed-text:latest" || retired.Metadata.EmbeddingRole != "embedding.default" || retired.Metadata.EmbeddingDimensions != 768 || retired.Metadata.PolicySchemaVersion != "policy/v4" || retired.Metadata.MigrationPhase != "active" {
		t.Fatalf("promotion discarded previous recipe identity: %+v", retired.Metadata)
	}
	catalog.fail = nil

	// A newly constructed store has no in-memory promoted/candidate history.
	restarted := makeCatalogGenerationStore(t, server.URL, alias, fake, catalog)
	if err := restarted.RollbackGeneration(context.Background(), "gen-2"); err != nil {
		t.Fatal(err)
	}
	if got := fake.aliases[alias]; got != previous {
		t.Fatalf("rollback alias target = %q, want previous generation %q", got, previous)
	}
	if _, exists := fake.collections[candidate]; exists {
		t.Fatal("rollback should delete the promoted candidate after restoring the old alias")
	}
	restored := catalog.record("gen-1")
	if restored.State != GenerationStateActive || restored.Metadata.Model != "nomic-embed-text:latest" || restored.Metadata.EmbeddingDimensions != 768 || restored.Metadata.PolicySchemaVersion != "policy/v4" {
		t.Fatalf("rollback did not restore the prior recipe record as active: %+v", restored)
	}
}

func TestQdrantGenerationStorePromotionReplayRetiresDurablePriorRecord(t *testing.T) {
	fake := newFakeQdrant()
	const alias = "fixture_search"
	previous := generationCollectionName(alias, "gen-1")
	candidate := generationCollectionName(alias, "gen-2")
	fake.collections[previous] = &fakeCollection{points: 1}
	fake.collections[candidate] = &fakeCollection{points: 2}
	fake.aliases[alias] = previous
	catalog := &generationCatalogFixture{records: map[string]GenerationRecord{
		"gen-1": {Metadata: GenerationMetadata{ID: "gen-1", Model: "nomic-embed-text:latest", EmbeddingRole: "embedding.default", EmbeddingDimensions: 768, PolicySchemaVersion: "policy/v4"}, CollectionName: previous, State: GenerationStateActive},
		"gen-2": {Metadata: GenerationMetadata{ID: "gen-2"}, CollectionName: candidate, State: GenerationStateCandidate},
	}}
	server := httptest.NewServer(fake.handler())
	defer server.Close()
	catalog.fail = func(record GenerationRecord) error {
		if record.Metadata.ID == "gen-1" && record.State == GenerationStateRetired {
			return errors.New("injected prior-retirement failure")
		}
		return nil
	}
	store := makeCatalogGenerationStore(t, server.URL, alias, fake, catalog)
	if err := store.PromoteGeneration(context.Background(), "gen-2"); err == nil {
		t.Fatal("promotion should surface prior-record failure")
	}
	if fake.aliases[alias] != candidate {
		t.Fatalf("alias = %q, want switched candidate %q", fake.aliases[alias], candidate)
	}
	catalog.fail = nil
	restarted := makeCatalogGenerationStore(t, server.URL, alias, fake, catalog)
	if err := restarted.PromoteGeneration(context.Background(), "gen-2"); err != nil {
		t.Fatal(err)
	}
	prior := catalog.record("gen-1")
	if prior.State != GenerationStateRetired || prior.Metadata.Model != "nomic-embed-text:latest" || prior.Metadata.EmbeddingRole != "embedding.default" || prior.Metadata.EmbeddingDimensions != 768 || prior.Metadata.PolicySchemaVersion != "policy/v4" {
		t.Fatalf("replay lost prior record identity/state: %+v", prior)
	}
}

func TestQdrantGenerationStoreRollbackReplayFinishesPriorCatalogRestore(t *testing.T) {
	fake := newFakeQdrant()
	const alias = "fixture_search"
	previous := generationCollectionName(alias, "gen-1")
	candidate := generationCollectionName(alias, "gen-2")
	fake.collections[previous] = &fakeCollection{points: 1}
	fake.collections[candidate] = &fakeCollection{points: 2}
	fake.aliases[alias] = candidate
	catalog := &generationCatalogFixture{records: map[string]GenerationRecord{
		"gen-1": {Metadata: GenerationMetadata{ID: "gen-1"}, CollectionName: previous, State: GenerationStateRetired},
		"gen-2": {Metadata: GenerationMetadata{ID: "gen-2"}, CollectionName: candidate, PreviousCollection: previous, State: GenerationStateActive},
	}}
	server := httptest.NewServer(fake.handler())
	defer server.Close()
	// Simulate death after alias restoration but before catalog restoration.
	fake.aliases[alias] = previous
	store := makeCatalogGenerationStore(t, server.URL, alias, fake, catalog)
	if err := store.RollbackGeneration(context.Background(), "gen-2"); err != nil {
		t.Fatal(err)
	}
	if got := catalog.record("gen-1").State; got != GenerationStateActive {
		t.Fatalf("replayed prior state = %q, want active", got)
	}
	if _, exists := fake.collections[candidate]; exists {
		t.Fatal("replayed rollback should remove candidate")
	}
}

func TestQdrantGenerationStoreRollbackRetryFinishesInterruptedDelete(t *testing.T) {
	fake := newFakeQdrant()
	const alias = "fixture_search"
	previous := generationCollectionName(alias, "gen-1")
	candidate := generationCollectionName(alias, "gen-2")
	fake.collections[previous] = &fakeCollection{points: 1}
	fake.collections[candidate] = &fakeCollection{points: 2}
	fake.aliases[alias] = candidate
	catalog := &generationCatalogFixture{records: map[string]GenerationRecord{
		"gen-2": {Metadata: GenerationMetadata{ID: "gen-2", Owner: "fixture.test", Namespace: alias, Alias: alias}, CollectionName: candidate, PreviousCollection: previous, State: GenerationStateActive},
	}}
	server := httptest.NewServer(fake.handler())
	defer server.Close()
	fake.failDelete[candidate] = true
	store := makeCatalogGenerationStore(t, server.URL, alias, fake, catalog)
	if err := store.RollbackGeneration(context.Background(), "gen-2"); err == nil {
		t.Fatal("first rollback should report the injected delete failure")
	}
	if got := fake.aliases[alias]; got != previous {
		t.Fatalf("alias should be restored even when candidate deletion fails, got %q", got)
	}
	delete(fake.failDelete, candidate)
	restarted := makeCatalogGenerationStore(t, server.URL, alias, fake, catalog)
	if err := restarted.RollbackGeneration(context.Background(), "gen-2"); err != nil {
		t.Fatal(err)
	}
	if _, exists := fake.collections[candidate]; exists {
		t.Fatal("reconstructed retry should finish candidate deletion")
	}
}

func TestQdrantGenerationStoreCleanupAttemptsEveryRetiredCollection(t *testing.T) {
	fake := newFakeQdrant()
	const alias = "fixture_search"
	for _, id := range []string{"gen-1", "gen-2", "gen-3"} {
		fake.collections[generationCollectionName(alias, id)] = &fakeCollection{points: 1}
	}
	fake.aliases[alias] = generationCollectionName(alias, "gen-3")
	fake.failDelete[generationCollectionName(alias, "gen-2")] = true
	store, _ := newTestGenerationStore(t, fake, alias)

	err := store.CleanupGenerations(context.Background(), 0)
	if err == nil || !strings.Contains(err.Error(), "gen-2") {
		t.Fatalf("cleanup must report the failed retirement, got %v", err)
	}
	if strings.Join(fake.deletes, ",") != generationCollectionName(alias, "gen-1") {
		t.Fatalf("a failed delete must not stop the remaining retirements, deleted %v", fake.deletes)
	}
}

func TestQdrantGenerationStoreAbandonsHalfSeededCandidate(t *testing.T) {
	fake := newFakeQdrant()
	const alias = "fixture_search"
	active := generationCollectionName(alias, "gen-1")
	fake.collections[active] = &fakeCollection{points: 3}
	fake.aliases[alias] = active
	fake.shortSeed = true
	store, raw := newTestGenerationStore(t, fake, alias)
	raw.seedTimeout = 20 * time.Millisecond

	err := store.BeginGeneration(context.Background(), GenerationMetadata{ID: "gen-2"})
	if err == nil {
		t.Fatal("a candidate that never reaches the active point count must fail BeginGeneration")
	}
	candidate := generationCollectionName(alias, "gen-2")
	if _, exists := fake.collections[candidate]; exists {
		t.Fatalf("half-seeded candidate %q must be deleted, not left as an orphan", candidate)
	}
	if strings.Join(fake.deletes, ",") != candidate {
		t.Fatalf("expected exactly one delete of the candidate, got %v", fake.deletes)
	}
}
