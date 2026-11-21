package runannotations

import (
	"context"
	"testing"
)

func TestProvenanceHashFollowsTheEditablePolicy(t *testing.T) {
	policy := authorshipPolicy(t)
	if policy.Hash() != authorshipPolicy(t).Hash() {
		t.Fatal("the same policy hashed two ways")
	}
	for name, edit := range map[string]func(p *Provenance){
		"a marker":           func(p *Provenance) { p.AgentMarkers = append(append([]string(nil), p.AgentMarkers...), `^New marker`) },
		"an opening":         func(p *Provenance) { p.DelegatingOpenings = nil },
		"a brief workload":   func(p *Provenance) { p.BriefWorkloads = []string{"workflow_node"} },
		"the lineage switch": func(p *Provenance) { p.DelegatedByLineage = false },
	} {
		edited := policy
		edit(&edited)
		if edited.Hash() == policy.Hash() {
			t.Fatalf("editing %s left the policy hash unchanged, so stale rows would pass as current", name)
		}
	}
}

// A policy edit must reach coverage without re-annotating: rows computed under
// the old policy are counted apart, then recomputed from facts alone.
func TestEditingTheProvenancePolicyMakesStoredRowsStaleUntilRefreshed(t *testing.T) {
	ctx := context.Background()
	facts := []RunFacts{
		conversation("chat", "interactive", "fix it", "# E3 do the thing"),
		conversation("flow", "workflow_node", "brief"),
	}
	service, store := testServiceRuntime(t, `{"mode":"passive","provenance":{`+testBriefWorkloads+`,"agent_markers":["^\\s*# E\\d+\\b"]}}`, messageLabels, facts, &fakeDecision{}, Options{})
	if _, err := service.Evaluate(ctx, EvaluateOptions{RunIDs: []string{"chat", "flow"}, TimeBudget: -1}); err != nil {
		t.Fatal(err)
	}
	before, err := service.Coverage(ctx, "7d")
	if err != nil {
		t.Fatal(err)
	}
	if p := before.Provenance; p == nil || p.RunsHumanText != 1 || p.HumanMessages != 1 || p.BriefMessages != 2 || p.RunsStale != 0 {
		t.Fatalf("provenance under the first policy = %+v", p)
	}

	// The same database read by a pack whose policy drops the epoch marker and
	// no longer calls a workflow node's text a brief.
	edited := `{"pack":{"name":"test","version":"2","runtime":{"mode":"passive","provenance":{"brief_workloads":["scheduled"],"seed_brief_workloads":["adhoc"]}},"labels":` + messageLabels + `}}`
	pack, err := LoadCompiled(writePack(t, edited))
	if err != nil {
		t.Fatal(err)
	}
	fixture := NewFixtureSource()
	for _, fact := range facts {
		fixture.Set(fact)
	}
	next, err := NewService(pack, Options{Store: store, Features: fixture, Decisions: &fakeDecision{}, Clock: service.now})
	if err != nil {
		t.Fatal(err)
	}
	stale, err := next.Coverage(ctx, "7d")
	if err != nil {
		t.Fatal(err)
	}
	if p := stale.Provenance; p == nil || p.RunsStale != 2 || p.RunsHumanText != 0 || p.HumanMessages != 0 || p.RunsAnnotated != 2 {
		t.Fatalf("rows computed under another policy must be stale and in no bucket: %+v", p)
	}
	refreshed, err := next.RefreshProvenance(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.Refreshed != 2 || refreshed.Remaining != 0 || refreshed.Failed != 0 {
		t.Fatalf("refresh = %+v", refreshed)
	}
	after, err := next.Coverage(ctx, "7d")
	if err != nil {
		t.Fatal(err)
	}
	// chat: both messages are typed now that the marker is gone; flow: a
	// workflow_node is no longer a brief workload.
	if p := after.Provenance; p == nil || p.RunsStale != 0 || p.RunsHumanText != 2 || p.HumanMessages != 3 {
		t.Fatalf("provenance after refresh = %+v", p)
	}
	again, err := next.RefreshProvenance(ctx, 0)
	if err != nil || again.Refreshed != 0 || again.Remaining != 0 {
		t.Fatalf("a second refresh found work: %+v, %v", again, err)
	}
}
