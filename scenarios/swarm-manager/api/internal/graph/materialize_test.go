package graph

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"swarm-manager/internal/backlog"
)

// stubBacklogStore lets tests supply items without spinning up a FileStore.
type stubBacklogStore struct {
	items map[string]backlog.BacklogItem // key: "kind/name"
}

func (s *stubBacklogStore) LoadItem(kind backlog.BacklogKind, name string) (backlog.BacklogItem, error) {
	key := string(kind) + "/" + name
	if item, ok := s.items[key]; ok {
		return item, nil
	}
	return backlog.BacklogItem{}, os.ErrNotExist
}

// stubInitiativeLister returns a fixed initiative list.
type stubInitiativeLister struct {
	entries []InitiativeEntry
}

func (s *stubInitiativeLister) List() ([]InitiativeEntry, error) {
	return s.entries, nil
}

func TestMaterializeInitiative_WritesGraphJSON(t *testing.T) {
	rootDir := t.TempDir()
	initDir := filepath.Join(rootDir, "initiatives", "my-init")
	if err := os.MkdirAll(initDir, 0o755); err != nil {
		t.Fatal(err)
	}

	lister := &stubInitiativeLister{entries: []InitiativeEntry{
		{Name: "my-init", Title: "My Initiative", Status: "active", Items: []string{"execute/foo", "execute/bar"}},
	}}
	store := &stubBacklogStore{items: map[string]backlog.BacklogItem{
		"execute/foo": {Name: "foo", Kind: backlog.KindExecute, Title: "Foo", Status: backlog.StatusCompleted, Priority: 1, Effort: "M", DependsOn: nil},
		"execute/bar": {Name: "bar", Kind: backlog.KindExecute, Title: "Bar", Status: backlog.StatusBacklog, Priority: 2, DependsOn: []string{"execute/foo"}},
	}}

	m := NewMaterializer(lister, store, func(name string) string {
		return filepath.Join(rootDir, "initiatives", name)
	})

	if err := m.MaterializeInitiative(context.Background(), "my-init"); err != nil {
		t.Fatalf("MaterializeInitiative: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(initDir, "graph.json"))
	if err != nil {
		t.Fatalf("read graph.json: %v", err)
	}
	var g MaterializedGraph
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatalf("unmarshal graph.json: %v", err)
	}

	if g.Initiative != "my-init" {
		t.Errorf("Initiative = %q, want %q", g.Initiative, "my-init")
	}
	if len(g.Nodes) != 2 {
		t.Fatalf("Nodes = %d, want 2", len(g.Nodes))
	}
	if len(g.Edges) != 1 {
		t.Fatalf("Edges = %d, want 1 (bar depends on foo)", len(g.Edges))
	}
	if g.Edges[0].From != "execute/bar" || g.Edges[0].To != "execute/foo" {
		t.Errorf("edge = %s→%s, want execute/bar→execute/foo", g.Edges[0].From, g.Edges[0].To)
	}
}

func TestMaterializeInitiative_DropsCrossInitiativeEdges(t *testing.T) {
	rootDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(rootDir, "initiatives", "a"), 0o755); err != nil {
		t.Fatal(err)
	}

	lister := &stubInitiativeLister{entries: []InitiativeEntry{
		{Name: "a", Status: "active", Items: []string{"execute/inside"}},
	}}
	store := &stubBacklogStore{items: map[string]backlog.BacklogItem{
		// "inside" depends on "outside", which isn't a member of initiative "a".
		"execute/inside":  {Name: "inside", Kind: backlog.KindExecute, DependsOn: []string{"execute/outside"}},
		"execute/outside": {Name: "outside", Kind: backlog.KindExecute},
	}}

	m := NewMaterializer(lister, store, func(name string) string {
		return filepath.Join(rootDir, "initiatives", name)
	})
	if err := m.MaterializeInitiative(context.Background(), "a"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(rootDir, "initiatives", "a", "graph.json"))
	var g MaterializedGraph
	_ = json.Unmarshal(data, &g)
	if len(g.Edges) != 0 {
		t.Errorf("expected 0 edges (outside is not in initiative), got %d", len(g.Edges))
	}
}

func TestMaterializeInitiative_RemovesStaleGraph(t *testing.T) {
	rootDir := t.TempDir()
	initDir := filepath.Join(rootDir, "initiatives", "gone")
	if err := os.MkdirAll(initDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Pre-write a stale graph.json as if the initiative existed previously.
	stalePath := filepath.Join(initDir, "graph.json")
	if err := os.WriteFile(stalePath, []byte(`{"initiative":"gone"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Lister reports no such initiative → materializer should remove the stale file.
	lister := &stubInitiativeLister{entries: []InitiativeEntry{}}
	store := &stubBacklogStore{items: map[string]backlog.BacklogItem{}}
	m := NewMaterializer(lister, store, func(name string) string {
		return filepath.Join(rootDir, "initiatives", name)
	})
	if err := m.MaterializeInitiative(context.Background(), "gone"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Errorf("expected stale graph.json to be removed, stat err = %v", err)
	}
}

// TestReadGraph_ReturnsNilWhenNotExists is the "first ever read" case. The
// materializer has not run yet (e.g., a brand-new initiative with no items
// has been created and the listener hasn't fired). ReadGraph must return
// (nil, nil), not an error — callers need to distinguish "no graph yet"
// from "graph read failed" and the nil/nil convention keeps that cheap.
func TestReadGraph_ReturnsNilWhenNotExists(t *testing.T) {
	rootDir := t.TempDir()
	m := NewMaterializer(nil, nil, func(name string) string {
		return filepath.Join(rootDir, "initiatives", name)
	})

	got, err := m.ReadGraph("never-materialized")
	if err != nil {
		t.Fatalf("ReadGraph err = %v, want nil", err)
	}
	if got != nil {
		t.Fatalf("ReadGraph graph = %+v, want nil", got)
	}
}

// TestReadGraph_ParsesValidJSON exercises the happy path. If the schema
// ever drifts (e.g., a new required field), the test fails and forces
// callers to migrate together.
func TestReadGraph_ParsesValidJSON(t *testing.T) {
	rootDir := t.TempDir()
	initDir := filepath.Join(rootDir, "initiatives", "happy")
	if err := os.MkdirAll(initDir, 0o755); err != nil {
		t.Fatal(err)
	}
	want := MaterializedGraph{
		Initiative:  "happy",
		GeneratedAt: "2026-04-23T12:00:00Z",
		Nodes: []MaterializedGraphNode{
			{ID: "execute/foo", Kind: "execute", Name: "foo", Title: "Foo", Status: "backlog", Priority: 3, Effort: "M"},
		},
		Edges: []MaterializedGraphEdge{},
	}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(initDir, "graph.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}

	m := NewMaterializer(nil, nil, func(name string) string {
		return filepath.Join(rootDir, "initiatives", name)
	})
	got, err := m.ReadGraph("happy")
	if err != nil {
		t.Fatalf("ReadGraph err = %v", err)
	}
	if got == nil {
		t.Fatal("ReadGraph returned nil graph")
	}
	if got.Initiative != want.Initiative {
		t.Errorf("Initiative = %q, want %q", got.Initiative, want.Initiative)
	}
	if len(got.Nodes) != 1 || got.Nodes[0].ID != "execute/foo" {
		t.Errorf("Nodes = %+v, want single execute/foo", got.Nodes)
	}
}

// TestReadGraph_ErrorOnMalformedJSON asserts we do not silently return
// empty data when the on-disk file is corrupted. A silent nil would make
// agents reason about a stale or phantom graph; the error forces the
// caller to decide (re-materialize, skip, etc.).
func TestReadGraph_ErrorOnMalformedJSON(t *testing.T) {
	rootDir := t.TempDir()
	initDir := filepath.Join(rootDir, "initiatives", "broken")
	if err := os.MkdirAll(initDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(initDir, "graph.json"), []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := NewMaterializer(nil, nil, func(name string) string {
		return filepath.Join(rootDir, "initiatives", name)
	})
	got, err := m.ReadGraph("broken")
	if err == nil {
		t.Fatalf("ReadGraph err = nil, want parse error; got graph = %+v", got)
	}
	if got != nil {
		t.Errorf("ReadGraph graph = %+v, want nil on parse error", got)
	}
}

// countingBacklogStore wraps a stubBacklogStore and counts LoadItem calls.
// Used to assert ScheduleAll coalescing actually reduces work rather than
// just eventually converging.
type countingBacklogStore struct {
	*stubBacklogStore
	loads atomic.Int64
}

func (c *countingBacklogStore) LoadItem(kind backlog.BacklogKind, name string) (backlog.BacklogItem, error) {
	c.loads.Add(1)
	return c.stubBacklogStore.LoadItem(kind, name)
}

// TestMaterializer_ScheduleAllCoalesces bounds the work done when many
// invalidations arrive in a burst. With the coalescing design documented on
// ScheduleAll (at most two runs: the one in flight plus one catch-up), a
// burst of 20 calls against 2 initiatives × 1 item each must perform at
// most 2 × 2 × 2 = 8 LoadItem calls total (buildGraph reads items twice —
// once for nodes, once for edges). We assert ≤ 16 to give headroom for
// scheduling overlap without allowing a 20× blowup.
func TestMaterializer_ScheduleAllCoalesces(t *testing.T) {
	rootDir := t.TempDir()
	for _, n := range []string{"one", "two"} {
		if err := os.MkdirAll(filepath.Join(rootDir, "initiatives", n), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	lister := &stubInitiativeLister{entries: []InitiativeEntry{
		{Name: "one", Status: "active", Items: []string{"execute/a"}},
		{Name: "two", Status: "active", Items: []string{"execute/a"}},
	}}
	counting := &countingBacklogStore{stubBacklogStore: &stubBacklogStore{
		items: map[string]backlog.BacklogItem{
			"execute/a": {Name: "a", Kind: backlog.KindExecute, Status: backlog.StatusInReview},
		},
	}}
	m := NewMaterializer(lister, counting, func(name string) string {
		return filepath.Join(rootDir, "initiatives", name)
	})

	const burst = 20
	for i := 0; i < burst; i++ {
		m.ScheduleAll()
	}

	// Wait for materialization to drain. Both graph.json files must exist.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, err1 := os.Stat(filepath.Join(rootDir, "initiatives", "one", "graph.json"))
		_, err2 := os.Stat(filepath.Join(rootDir, "initiatives", "two", "graph.json"))
		if err1 == nil && err2 == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if _, err := os.Stat(filepath.Join(rootDir, "initiatives", "one", "graph.json")); err != nil {
		t.Fatalf("expected initiative one graph.json after burst: %v", err)
	}
	if _, err := os.Stat(filepath.Join(rootDir, "initiatives", "two", "graph.json")); err != nil {
		t.Fatalf("expected initiative two graph.json after burst: %v", err)
	}

	// The critical assertion: total LoadItem calls must not grow linearly
	// with the burst. 20 calls → ≤2 runs means ≤ 2 runs × 2 initiatives ×
	// 2 reads-per-item × 1 item = 8 calls. We allow 16 to absorb any
	// scheduling overlap where a third run sneaks in.
	const maxLoads = 16
	got := counting.loads.Load()
	if got > maxLoads {
		t.Fatalf("ScheduleAll coalescing regressed: %d LoadItem calls for burst of %d, want ≤ %d",
			got, burst, maxLoads)
	}
}

// TestMaterializer_DispatchHookWiring pins the single integration point
// that production relies on: DispatchInvalidate("topology") → Materializer
// re-runs. If this hook breaks, every graph.json update in production
// silently stops working even though every unit test still passes.
func TestMaterializer_DispatchHookWiring(t *testing.T) {
	rootDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(rootDir, "initiatives", "wired"), 0o755); err != nil {
		t.Fatal(err)
	}
	lister := &stubInitiativeLister{entries: []InitiativeEntry{
		{Name: "wired", Status: "active", Items: []string{"execute/a"}},
	}}
	counting := &countingBacklogStore{stubBacklogStore: &stubBacklogStore{
		items: map[string]backlog.BacklogItem{
			"execute/a": {Name: "a", Kind: backlog.KindExecute, Status: backlog.StatusBacklog},
		},
	}}
	m := NewMaterializer(lister, counting, func(name string) string {
		return filepath.Join(rootDir, "initiatives", name)
	})

	// Same wiring shape as routes_graph.go: register a hook that forwards
	// topology invalidations to ScheduleAll.
	dispatch := NewDispatch(nil, nil)
	dispatch.AddInvalidateHook(func(lenses []Lens) {
		for _, lens := range lenses {
			if lens == LensTopology {
				m.ScheduleAll()
				return
			}
		}
	})

	// Firing DispatchInvalidate with a non-topology lens must NOT trigger
	// materialization — otherwise the hook would over-run on every focus
	// update.
	dispatch.DispatchInvalidate(string(LensOperations))
	// Give a brief window for spurious work to appear.
	time.Sleep(50 * time.Millisecond)
	if n := counting.loads.Load(); n != 0 {
		t.Fatalf("operations-only invalidation triggered %d loads; must be 0", n)
	}

	// Topology invalidation must trigger a rebuild.
	dispatch.DispatchInvalidate(string(LensTopology))
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(rootDir, "initiatives", "wired", "graph.json")); err == nil {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("topology invalidation did not trigger materialization (loads=%d)", counting.loads.Load())
}
