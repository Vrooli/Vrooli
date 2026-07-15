package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeFile writes content to root/rel, creating parents.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

// buildFixture creates a synthetic data/state/cache tree exercising every
// anomaly and referential path the tool must surface.
func buildFixture(t *testing.T) Config {
	t.Helper()
	dir := t.TempDir()
	data := filepath.Join(dir, "data")
	state := filepath.Join(dir, "state")
	cache := filepath.Join(dir, "cache")

	// Valid backlog item with a managed plan-ref, depending on a non-existent item.
	writeFile(t, data, "execute/valid-item/spec.json", `{
		"name":"valid-item","kind":"execute","status":"in_progress",
		"depends_on":["fix/does-not-exist"],
		"plan_ref":{"provider":"plan-manager","plan_id":"p-123","slug":"valid-item","role":"execution_spec"}
	}`)
	// Item pointing at a non-existent initiative + an UNMANAGED plan-ref (empty plan_id).
	writeFile(t, data, "fix/orphan-init/spec.json", `{
		"name":"orphan-init","kind":"fix","status":"backlog","initiative":"ghost-initiative",
		"plan_ref":{"provider":"plan-manager","plan_id":"","slug":"orphan-init","role":"execution_spec"}
	}`)
	// Item with an INVALID status value.
	writeFile(t, data, "chore/bad-status/spec.json", `{"name":"bad-status","kind":"chore","status":"totally-bogus"}`)
	// MALFORMED spec.json (corrupt JSON).
	writeFile(t, data, "ideas/broken/spec.json", `{ this is not json `)
	// Item that an initiative will claim, but whose own initiative field diverges.
	writeFile(t, data, "execute/member-item/spec.json", `{"name":"member-item","kind":"execute","status":"ready","initiative":"other-initiative"}`)

	// Initiative that lists member-item (divergence) and a missing item, with unmanaged
	// (foreign provider) plan-ref, and a dangling initiative dependency.
	writeFile(t, data, "initiatives/real-initiative/initiative.json", `{
		"name":"real-initiative","status":"active","mode":"item-level",
		"items":["execute/member-item","fix/missing-member"],
		"depends_on":["nonexistent-initiative"],
		"plan_ref":{"provider":"legacy-tool","plan_id":"x","slug":"real-initiative","role":"weird_role"}
	}`)
	writeFile(t, data, "initiatives/other-initiative/initiative.json", `{"name":"other-initiative","status":"active","items":[]}`)

	// Goal with one valid + one dangling target.
	writeFile(t, data, "goals/g1/goal.json", `{"name":"g1","status":"active","targets":["execute/valid-item","initiative/ghost"]}`)

	// Record referencing entities.
	writeFile(t, data, "records/swarm-manager/execute/rec-abc.json", `{"id":"rec-abc","kind":"execute","outcome":"shipped","backlog_ref":"execute/valid-item"}`)

	// AMBIGUOUS ownership: one run id mapped to two owners in the global index.
	writeFile(t, data, "operating-mode-run-owners/run-owners.json", `{
		"owners":{"run-xyz":[
			{"target_kind":"plan-manager-plan","scope_id":"plan-a","mode":"phased","execution_id":"e1","round":1},
			{"target_kind":"plan-ref","scope_id":"plan-b","mode":"phased","execution_id":"e2","round":1}
		]}
	}`)

	// Operating-mode manifest (primary) under a mode target.
	writeFile(t, data, "mode-targets/plan-manager-plan/plan-a/modes/phased/executions/e1/manifest.json",
		`{"execution_id":"e1","scope_kind":"plan-manager-plan","scope_id":"plan-a","mode":"phased","status":"active"}`)

	// An UNCLASSIFIED artifact under the data root.
	writeFile(t, data, "mystery-blob.dat", "unknown bytes")

	// Foreign deployment report (ambiguous ownership).
	writeFile(t, data, "deployment/deployment-report.json", `{"report":"foreign"}`)

	// Opaque SQLite marker + JSONL manifest.
	writeFile(t, data, "events.db", "SQLite format 3\x00fake")
	writeFile(t, data, "plan-ref-sweep-manifest.jsonl", "{\"a\":1}\n{\"b\":2}\n")

	// State: a present agent-activities file; execution-runs with an orphaned run.
	writeFile(t, state, "agent-activities.json", `{"activities":[]}`)
	writeFile(t, state, "execution-runs.json", `[{"execution_id":"x1","backlog_kind":"execute","backlog_name":"valid-item","status":"running","run_id":""}]`)

	// Cache: a capture.
	writeFile(t, cache, "captures/cap-1/capture.json", `{"id":"cap-1","status":"classified"}`)

	return Config{DataRoot: data, StateRoot: state, CacheRoot: cache, ResolvedFrom: "test"}
}

func TestScanReportsAnomaliesAndFindings(t *testing.T) {
	inv := Scan(buildFixture(t))

	anomalyTypes := countByField(t, inv.Anomalies)
	if anomalyTypes["corrupt_json"] < 1 {
		t.Errorf("expected a corrupt_json anomaly for malformed spec.json, got %+v", anomalyTypes)
	}
	if anomalyTypes["invalid_status"] < 1 {
		t.Errorf("expected an invalid_status anomaly, got %+v", anomalyTypes)
	}

	findingTypes := map[string]int{}
	for _, f := range inv.ReferentialFindings {
		findingTypes[f.Type]++
	}
	for _, want := range []string{
		"dangling_dependency",              // valid-item -> fix/does-not-exist
		"dangling_item_initiative",         // orphan-init -> ghost-initiative
		"dangling_initiative_item",         // real-initiative -> fix/missing-member
		"initiative_membership_divergence", // real-initiative vs member-item.initiative
		"dangling_initiative_dependency",   // real-initiative -> nonexistent-initiative
		"dangling_goal_target",             // g1 -> initiative/ghost
		"unclassified_artifact",            // mystery-blob.dat
		"ambiguous_ownership",              // foreign deployment report
		"orphaned_execution",               // running run with empty run_id
	} {
		if findingTypes[want] < 1 {
			t.Errorf("expected referential finding %q, got types %+v", want, findingTypes)
		}
	}

	// Ambiguous run owner surfaced, not dropped.
	if len(inv.Ownership.AmbiguousRunOwners) < 1 {
		t.Errorf("expected an ambiguous run owner, got none")
	}
	found := false
	for _, a := range inv.Ownership.AmbiguousRunOwners {
		if a.RunID == "run-xyz" && len(a.Owners) == 2 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected run-xyz ambiguous owner with 2 owners, got %+v", inv.Ownership.AmbiguousRunOwners)
	}

	// Unmanaged plan-refs: orphan-init (empty plan_id) + real-initiative (foreign provider).
	if inv.PlanRefs.Unmanaged < 2 {
		t.Errorf("expected >=2 unmanaged plan-refs, got %d (%+v)", inv.PlanRefs.Unmanaged, inv.PlanRefs.Details)
	}

	// Expected-absent state surfaced (queue.json, circuit-breaker.json, etc.).
	if len(inv.ExpectedAbsent) == 0 {
		t.Errorf("expected some expected-absent state entries, got none")
	}

	// Corrupt spec still yields a primary object (not silently dropped).
	if !hasPrimaryStatus(inv, "backlog_item", "<unparseable>") {
		t.Errorf("expected corrupt backlog item to appear as an <unparseable> primary object")
	}
}

func TestScanIsByteStable(t *testing.T) {
	cfg := buildFixture(t)
	a := mustJSON(t, Scan(cfg))
	b := mustJSON(t, Scan(cfg))
	if string(a) != string(b) {
		t.Fatalf("inventory JSON is not byte-identical across two runs")
	}
	// Summaries must also be deterministic.
	if renderSummary(Scan(cfg)) != renderSummary(Scan(cfg)) {
		t.Fatalf("summary is not deterministic across two runs")
	}
}

func TestContentHashChangesWhenStateChanges(t *testing.T) {
	cfg := buildFixture(t)
	before := Scan(cfg).Totals.ContentHash
	writeFile(t, cfg.DataRoot, "execute/valid-item/spec.json", `{"name":"valid-item","kind":"execute","status":"completed"}`)
	after := Scan(cfg).Totals.ContentHash
	if before == after {
		t.Fatalf("content hash did not change after mutating state")
	}
}

func countByField(t *testing.T, anomalies []Anomaly) map[string]int {
	t.Helper()
	m := map[string]int{}
	for _, a := range anomalies {
		m[a.Type]++
	}
	return m
}

func hasPrimaryStatus(inv *Inventory, class, status string) bool {
	for _, c := range inv.Classes {
		if c.Class != class {
			continue
		}
		for _, o := range c.Objects {
			if o.Status == status {
				return true
			}
		}
	}
	return false
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return out
}
