package aisearch

import (
	"context"
	"encoding/json"
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
	store, err := NewQdrantGenerationStore(QdrantGenerationOptions{BaseURL: server.URL, Alias: alias, Spec: CollectionSpec{DenseSize: 4, DenseDistance: "Cosine"}})
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
