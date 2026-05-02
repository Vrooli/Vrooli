package main

// E2E coverage for merge_items mutations submitted through the feedback
// pipeline. Pins the contract that:
//
//   1. A feedback round can carry a merge_items mutation alongside other ops.
//   2. /decide → proposals.Applier executes the merge: edges retarget,
//      sources archive, the merged item lands attached to the initiative.
//   3. The on-disk graph projection reflects the merge (sources gone,
//      merged present, retargeted edges).
//
// Mirrors the disk-seeded /agent-turn pattern from the main e2e test so
// agent-manager spawn does not need to be wired.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/graph"
)

func itemArchived(t *testing.T, rootDir, kind, name string) bool {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(rootDir, kind, name, "spec.json"))
	if err != nil {
		t.Fatalf("read spec %s/%s: %v", kind, name, err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse spec %s/%s: %v", kind, name, err)
	}
	v, ok := m["archived_at"].(string)
	return ok && v != ""
}

func itemDependsOn(t *testing.T, rootDir, kind, name string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(rootDir, kind, name, "spec.json"))
	if err != nil {
		t.Fatalf("read spec %s/%s: %v", kind, name, err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse spec %s/%s: %v", kind, name, err)
	}
	raw, _ := m["depends_on"].([]any)
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func TestE2EInitiativeFeedback_MergeItemsAppliesAndRetargets(t *testing.T) {
	t.Setenv("AGENT_MANAGER_ENABLED", "false")

	srv := newTestServer(t)
	h := srv.Handler()
	rootDir := srv.scenarioRoot

	const initiative = "e2e-merge"

	// Seed initiative + four items: alpha+beta will be the merge sources,
	// gamma is an outbound dep (alpha depends on it), delta is an inbound
	// dependent (delta depends on alpha and beta).
	mustPost(t, h, "/api/v1/initiatives", map[string]any{
		"name":        initiative,
		"title":       "E2E Merge",
		"description": "Integration test for merge_items via feedback decide.",
		"status":      "active",
		"priority":    5,
	})
	mustPost(t, h, "/api/v1/backlog/batch", map[string]any{
		"items": []map[string]any{
			{"kind": "execute", "name": "gamma", "title": "Gamma", "initiative": initiative, "priority": 5},
			{"kind": "execute", "name": "alpha", "title": "Alpha", "initiative": initiative, "priority": 5, "depends_on": []string{"execute/gamma"}},
			{"kind": "execute", "name": "beta", "title": "Beta", "initiative": initiative, "priority": 5},
			{"kind": "execute", "name": "delta", "title": "Delta", "initiative": initiative, "priority": 5, "depends_on": []string{"execute/alpha", "execute/beta"}},
		},
	})

	graphPath := filepath.Join(rootDir, "initiatives", initiative, "graph.json")
	waitForGraph(t, graphPath, func(g graph.MaterializedGraph) bool {
		return len(g.Nodes) == 4 && len(g.Edges) == 3
	}, "materialized graph (4 nodes, 3 edges)")

	// Seed a feedback round directly on disk in agent_thinking so we can
	// inject a proposal via /agent-turn (same pattern as the main e2e
	// flow — agent-manager is disabled).
	roundDir := filepath.Join(rootDir, "initiatives", initiative, "feedback", "round-001-merge")
	if err := os.MkdirAll(roundDir, 0o755); err != nil {
		t.Fatalf("make round dir: %v", err)
	}
	seeded := map[string]any{
		"initiative_name": initiative,
		"number":          1,
		"slug":            "merge",
		"type":            "feedback",
		"status":          "agent_thinking",
		"submission":      map[string]any{"text": "fold alpha and beta together", "created_at": "2026-04-28T00:00:00Z"},
		"thread":          []map[string]any{{"role": "user", "content": "fold", "created_at": "2026-04-28T00:00:00Z"}},
		"run_id":          "test-merge-1",
		"created_at":      "2026-04-28T00:00:00Z",
		"updated_at":      "2026-04-28T00:00:00Z",
	}
	seededBytes, _ := json.Marshal(seeded)
	if err := os.WriteFile(filepath.Join(roundDir, "feedback.json"), seededBytes, 0o644); err != nil {
		t.Fatalf("seed round 1: %v", err)
	}

	// Inject a proposal containing one merge_items mutation.
	proposalBody := "Merging alpha and beta:\n```json\n" + `{
		"form": "mutation_list",
		"rationale": "alpha and beta share substrate; collapse them",
		"mutations": [
			{
				"id": "m1",
				"op": "merge_items",
				"sources": ["execute/alpha", "execute/beta"],
				"item": {"kind":"execute","name":"alpha-beta","title":"Alpha+Beta","description":"Combines alpha and beta","priority":5,"effort":"M"},
				"rationale": "shared substrate"
			}
		]
	}` + "\n```\n"
	code, body := postJSON(t, h, fmt.Sprintf("/api/v1/initiatives/%s/feedback/%d/agent-turn", initiative, 1), map[string]any{
		"body": proposalBody,
	})
	if code != http.StatusOK {
		t.Fatalf("agent-turn: status=%d body=%s", code, string(body))
	}

	// Decide: accept the merge mutation.
	code, body = postJSON(t, h, fmt.Sprintf("/api/v1/initiatives/%s/feedback/%d/decide", initiative, 1), map[string]any{
		"kind":                  "partial_accept",
		"accepted_mutation_ids": []string{"m1"},
		"rationale":             "agreed",
	})
	if code != http.StatusOK {
		t.Fatalf("decide: status=%d body=%s", code, string(body))
	}
	var decideResp struct {
		ApplyResult struct {
			Applied int `json:"applied"`
			Failed  int `json:"failed"`
		} `json:"apply_result"`
	}
	if err := json.Unmarshal(body, &decideResp); err != nil {
		t.Fatalf("decode decide: %v (body=%s)", err, string(body))
	}
	if decideResp.ApplyResult.Applied != 1 {
		t.Fatalf("expected applied=1, got applied=%d failed=%d body=%s",
			decideResp.ApplyResult.Applied, decideResp.ApplyResult.Failed, string(body))
	}

	// On-disk: alpha + beta archived, alpha-beta exists with retargeted deps.
	if !itemArchived(t, rootDir, "execute", "alpha") {
		t.Fatalf("alpha should be archived after merge")
	}
	if !itemArchived(t, rootDir, "execute", "beta") {
		t.Fatalf("beta should be archived after merge")
	}
	mergedDeps := itemDependsOn(t, rootDir, "execute", "alpha-beta")
	if !containsString(mergedDeps, "execute/gamma") {
		t.Fatalf("merged item should depend on gamma (retargeted from alpha), got %v", mergedDeps)
	}
	deltaDeps := itemDependsOn(t, rootDir, "execute", "delta")
	mergedRefCount := 0
	for _, d := range deltaDeps {
		if d == "execute/alpha-beta" {
			mergedRefCount++
		}
	}
	if mergedRefCount != 1 {
		t.Fatalf("delta should depend on merged exactly once, got deps=%v", deltaDeps)
	}
	if containsString(deltaDeps, "execute/alpha") || containsString(deltaDeps, "execute/beta") {
		t.Fatalf("delta should no longer reference archived sources, got deps=%v", deltaDeps)
	}

	// Graph projection: merged present and live; sources present but
	// flagged Archived. (The materializer keeps archived items in the
	// projection with Archived=true so the UI can still surface "this
	// item was here".)
	waitForGraph(t, graphPath, func(g graph.MaterializedGraph) bool {
		snapshot := make(map[string]bool, len(g.Nodes))
		for _, n := range g.Nodes {
			snapshot[n.ID] = n.Archived
		}
		mergedLive, mergedFound := snapshot["execute/alpha-beta"]
		alphaArchived, alphaFound := snapshot["execute/alpha"]
		betaArchived, betaFound := snapshot["execute/beta"]
		return mergedFound && !mergedLive && alphaFound && alphaArchived && betaFound && betaArchived
	}, "merge graph projection to converge")
}

func containsString(xs []string, target string) bool {
	for _, x := range xs {
		if x == target {
			return true
		}
	}
	return false
}
