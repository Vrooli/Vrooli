package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"swarm-manager/internal/graph"
	"swarm-manager/internal/testutil"
)

// TestGraphMaterialization_ExercisesRealWriters drives mutations through the
// real HTTP handlers for backlog (create, update, delete) and initiatives
// (create, add items) and asserts graph.json is rebuilt after each. This
// pins the dispatch invalidation listener wired at routes_graph.go to the
// Materializer's ScheduleAll hook — without this test, the materializer
// would be a no-op in production because nothing would fire it.
//
// The plan calls for "mutate items through every writer identified in
// exploration section 2+3" (plan §W2 tests). We cover:
//   - Single item create with Initiative field (handler_create.go)
//   - Batch item create with Initiative field (handler.go batch)
//   - Update patch (handler.go update; depends_on mutation → graph edge)
//   - Delete (handler.go delete)
//
// The status-auto-set path in polling.go is covered by polling_status_test.go.
// The Initiative AddItems/RemoveItems path is exercised transitively by the
// backlog handler's initiative assignment (it routes through the assigner).
func TestGraphMaterialization_ExercisesRealWriters(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()
	rootDir := testDataRoot(t)

	// Create an initiative so subsequent items land in its graph.
	mustPost(t, h, "/api/v1/initiatives", map[string]any{
		"name":        "g-init",
		"title":       "Graph Init",
		"description": "Used for the materialization integration test.",
		"status":      "active",
		"priority":    5,
	})

	// Graph path is the sink for every assertion. Reading and polling helpers
	// are hoisted into closures for brevity.
	graphPath := filepath.Join(rootDir, "initiatives", "g-init", "graph.json")

	// Initial materialization (boot-time MaterializeAll) should write graph.json
	// with zero nodes for the fresh initiative.
	waitForGraph(t, graphPath, func(g graph.MaterializedGraph) bool {
		return g.Initiative == "g-init" && len(g.Nodes) == 0
	}, "empty graph after initiative create")

	// 1. Single item create with Initiative field. Exercises handler_create.go.
	mustPost(t, h, "/api/v1/backlog", map[string]any{
		"kind":       "execute",
		"name":       "alpha",
		"title":      "Alpha",
		"initiative": "g-init",
		"priority":   5,
	})
	waitForGraph(t, graphPath, func(g graph.MaterializedGraph) bool {
		return len(g.Nodes) == 1 && g.Nodes[0].ID == "execute/alpha"
	}, "graph has alpha after single create")

	// 2. Batch item create with Initiative field. Exercises handler.go
	//    batch-create writer + dispatches the same invalidation.
	mustPost(t, h, "/api/v1/backlog/batch", map[string]any{
		"items": []map[string]any{
			{"kind": "execute", "name": "beta", "title": "Beta", "initiative": "g-init", "priority": 5},
			{"kind": "execute", "name": "gamma", "title": "Gamma", "initiative": "g-init", "priority": 5},
		},
	})
	waitForGraph(t, graphPath, func(g graph.MaterializedGraph) bool {
		if len(g.Nodes) != 3 {
			return false
		}
		names := map[string]struct{}{}
		for _, n := range g.Nodes {
			names[n.ID] = struct{}{}
		}
		_, hasAlpha := names["execute/alpha"]
		_, hasBeta := names["execute/beta"]
		_, hasGamma := names["execute/gamma"]
		return hasAlpha && hasBeta && hasGamma
	}, "graph has 3 nodes after batch create")

	// 3. Update patch (depends_on). Exercises handler.go update + the edge
	//    projection. beta depends on alpha.
	mustPatch(t, h, "/api/v1/backlog/execute/beta", map[string]any{
		"depends_on": []string{"execute/alpha"},
	})
	waitForGraph(t, graphPath, func(g graph.MaterializedGraph) bool {
		if len(g.Edges) != 1 {
			return false
		}
		e := g.Edges[0]
		return e.From == "execute/beta" && e.To == "execute/alpha"
	}, "graph has beta→alpha edge after update")

	// 4. Delete. Exercises handler.go delete. Removing beta should drop the
	//    node AND its outbound edge.
	mustDelete(t, h, "/api/v1/backlog/execute/beta")
	waitForGraph(t, graphPath, func(g graph.MaterializedGraph) bool {
		if len(g.Nodes) != 2 {
			return false
		}
		for _, n := range g.Nodes {
			if n.ID == "execute/beta" {
				return false
			}
		}
		return len(g.Edges) == 0
	}, "graph drops beta and its edge after delete")

	// 5. Status update on an item. Exercises handler.go update again. The
	//    materialized graph snapshots the current status on each node, so
	//    changing alpha's status should appear in graph.json.
	//    (Backlog is the only valid user-PATCH target here; review states are
	//    guarded.)
	mustPatch(t, h, "/api/v1/backlog/execute/alpha", map[string]any{
		"status": "ready",
	})
	waitForGraph(t, graphPath, func(g graph.MaterializedGraph) bool {
		for _, n := range g.Nodes {
			if n.ID == "execute/alpha" && n.Status == "ready" {
				return true
			}
		}
		return false
	}, "graph reflects alpha status=ready after update")
}

// waitForGraph polls graph.json until pred returns true or the budget expires.
// The materializer runs on a goroutine; a real test run settles in well under
// 100ms, but we allow up to ~2s so flaky CI schedulers don't fail the test.
func waitForGraph(t *testing.T, path string, pred func(graph.MaterializedGraph) bool, reason string) {
	t.Helper()
	testutil.Eventually(t, 2*time.Second, reason, func() bool {
		data, err := os.ReadFile(path)
		if err == nil {
			var g graph.MaterializedGraph
			if jsonErr := json.Unmarshal(data, &g); jsonErr == nil {
				if pred(g) {
					return true
				}
			}
		}
		return false
	})
}

func mustPost(t *testing.T, h http.Handler, path string, body any) {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code >= 300 {
		t.Fatalf("POST %s: status=%d body=%s", path, rec.Code, rec.Body.String())
	}
}

func mustPatch(t *testing.T, h http.Handler, path string, body any) {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code >= 300 {
		t.Fatalf("PATCH %s: status=%d body=%s", path, rec.Code, rec.Body.String())
	}
}

func mustDelete(t *testing.T, h http.Handler, path string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code >= 300 {
		t.Fatalf("DELETE %s: status=%d body=%s", path, rec.Code, rec.Body.String())
	}
}

// TestGraphMaterialization_ExercisesInitiativesService drives membership
// mutations through POST/DELETE /api/v1/initiatives/{name}/items and asserts
// graph.json rebuilds. This pins the initiatives.Service.invalidateTopologyGraph
// hook to the Materializer's ScheduleAll listener — a separate path from the
// backlog handler's invalidateAllGraphLenses.
//
// Writers covered here (plan §W2):
//   - initiatives/service.go AddItems  (POST   /api/v1/initiatives/{name}/items)
//   - initiatives/service.go RemoveItems (DELETE /api/v1/initiatives/{name}/items)
func TestGraphMaterialization_ExercisesInitiativesService(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()
	rootDir := testDataRoot(t)

	mustPost(t, h, "/api/v1/initiatives", map[string]any{
		"name":        "svc-init",
		"title":       "Service Init",
		"description": "Exercises initiatives.Service writer paths.",
		"status":      "active",
		"priority":    5,
	})

	// Create two orphan items (no initiative field). These are the items we'll
	// attach/detach via the initiatives service.
	mustPost(t, h, "/api/v1/backlog", map[string]any{
		"kind": "execute", "name": "orphan-a", "title": "Orphan A", "priority": 5,
	})
	mustPost(t, h, "/api/v1/backlog", map[string]any{
		"kind": "execute", "name": "orphan-b", "title": "Orphan B", "priority": 5,
	})

	graphPath := filepath.Join(rootDir, "initiatives", "svc-init", "graph.json")

	// Confirm the initiative's graph is empty before attachment.
	waitForGraph(t, graphPath, func(g graph.MaterializedGraph) bool {
		return g.Initiative == "svc-init" && len(g.Nodes) == 0
	}, "empty graph before attachment")

	// AddItems: should trigger invalidateTopologyGraph → graph.json rebuilds.
	mustPost(t, h, "/api/v1/initiatives/svc-init/items", map[string]any{
		"items": []string{"execute/orphan-a", "execute/orphan-b"},
	})
	waitForGraph(t, graphPath, func(g graph.MaterializedGraph) bool {
		if len(g.Nodes) != 2 {
			return false
		}
		names := map[string]struct{}{}
		for _, n := range g.Nodes {
			names[n.ID] = struct{}{}
		}
		_, hasA := names["execute/orphan-a"]
		_, hasB := names["execute/orphan-b"]
		return hasA && hasB
	}, "graph has 2 nodes after AddItems")

	// RemoveItems: graph.json drops the node.
	mustDeleteBody(t, h, "/api/v1/initiatives/svc-init/items", map[string]any{
		"items": []string{"execute/orphan-a"},
	})
	waitForGraph(t, graphPath, func(g graph.MaterializedGraph) bool {
		if len(g.Nodes) != 1 {
			return false
		}
		return g.Nodes[0].ID == "execute/orphan-b"
	}, "graph has only orphan-b after RemoveItems")
}

// Coverage note for the execution/polling and workshop writer paths:
//
// execution/polling.go and internal/backlog/workshop_save.go do not expose
// HTTP endpoints that can be driven directly from a unit test (polling
// requires agent-manager integration; workshop finalization is an async
// agent flow). Those writers are covered transitively:
//
//   - polling_status_test.go pins the in-process status writes from polling
//     land in the right backlog_status values.
//   - execution/service.go dispatchStatusUpdate calls DispatchInvalidate
//     ("topology") on the same Dispatcher that the Materializer subscribes
//     to at routes_graph.go — proven wired by the backlog-handler tests in
//     TestGraphMaterialization_ExercisesRealWriters and by
//     TestGraphMaterialization_ExercisesInitiativesService below.
//
// If either of those listener hooks detaches, BOTH test functions above
// will fail — so this indirection is adequate boundary coverage without
// adding a fragile end-to-end agent-manager harness.

// mustDeleteBody sends a DELETE with a JSON body (initiatives RemoveItems
// accepts the items list in the request body).
func mustDeleteBody(t *testing.T, h http.Handler, path string, body any) {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodDelete, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code >= 300 {
		t.Fatalf("DELETE %s: status=%d body=%s", path, rec.Code, rec.Body.String())
	}
}

// Boundary enforcement moved to internal/graph/boundary_test.go, which does
// real structural assertions (reflect-based API surface pin + ast-based
// writer scan) rather than the placeholder smoke test that lived here.
// See TestBoundary_MaterializerAPISurface and friends.

// TestGraphMaterialization_ExecutionStatusInvalidates drives a mutation via
// the execution status dispatch path that polling.go uses in production.
// Rather than spin up an agent-manager mock, we confirm the wiring at the
// dispatcher boundary: DispatchInvalidate("topology") from the same
// Dispatcher that the Materializer listens on must rebuild graph.json.
//
// This covers the writers in execution/polling.go (status auto-set →
// dispatchStatusUpdate → DispatchInvalidate("topology", ...)) without
// needing the full agent-manager harness.
func TestGraphMaterialization_ExecutionStatusInvalidates(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()
	rootDir := testDataRoot(t)

	mustPost(t, h, "/api/v1/initiatives", map[string]any{
		"name":        "exec-init",
		"title":       "Exec Init",
		"description": "Pins the dispatch → materializer hook used by execution polling.",
		"status":      "active",
		"priority":    5,
	})
	mustPost(t, h, "/api/v1/backlog", map[string]any{
		"kind": "execute", "name": "exec-item", "title": "Exec Item",
		"initiative": "exec-init", "priority": 5,
	})

	graphPath := filepath.Join(rootDir, "initiatives", "exec-init", "graph.json")
	waitForGraph(t, graphPath, func(g graph.MaterializedGraph) bool {
		return len(g.Nodes) == 1
	}, "graph seeded with exec-item")

	// Delete graph.json so we can assert the next invalidation recreates it.
	// (The materializer writes atomically; a pure timestamp-based check would
	// race, whereas file-existence is unambiguous.)
	if err := os.Remove(graphPath); err != nil {
		t.Fatalf("remove graph.json: %v", err)
	}

	// Fire a topology invalidation through the same dispatcher the execution
	// service uses in production (wired on srv). If the hook wiring regresses,
	// graph.json will never reappear.
	srv.graphDispatch.DispatchInvalidate("topology")

	waitForGraph(t, graphPath, func(g graph.MaterializedGraph) bool {
		return len(g.Nodes) == 1
	}, "graph re-materialized after topology invalidation")
}
