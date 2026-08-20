package graph

import (
	"reflect"
	"testing"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/aisearch"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/interfacegraph"
)

// TestInvertGraph verifies the interface graph's directed edges invert into
// per-scenario {depends_on (forward), used_by (reverse)} records, sorted and
// deduplicated, with isolated nodes preserved as empty-connection records.
func TestInvertGraph(t *testing.T) {
	g := interfacegraph.Graph{
		Nodes: []interfacegraph.Node{
			{Scenario: "plan-manager"},
			{Scenario: "prompt-manager"},
			{Scenario: "search-hub"},
			{Scenario: "isolated"},
		},
		Edges: []interfacegraph.Edge{
			{FromScenario: "plan-manager", ToScenario: "prompt-manager"},
			{FromScenario: "plan-manager", ToScenario: "search-hub"},
			{FromScenario: "swarm-manager", ToScenario: "prompt-manager"}, // node introduced by edge
			{FromScenario: "plan-manager", ToScenario: "prompt-manager"},  // duplicate, must dedup
			{FromScenario: "loop", ToScenario: "loop"},                    // self-edge, ignored
		},
	}

	got := invertGraph(g)
	idx := map[string]aisearch.ScenarioConnection{}
	for _, c := range got {
		idx[c.Scenario] = c
	}

	pm := idx["plan-manager"]
	if !reflect.DeepEqual(pm.DependsOn, []string{"prompt-manager", "search-hub"}) {
		t.Errorf("plan-manager depends_on = %v", pm.DependsOn)
	}
	if len(pm.UsedBy) != 0 {
		t.Errorf("plan-manager used_by = %v, want empty", pm.UsedBy)
	}

	prompt := idx["prompt-manager"]
	if !reflect.DeepEqual(prompt.UsedBy, []string{"plan-manager", "swarm-manager"}) {
		t.Errorf("prompt-manager used_by = %v, want [plan-manager swarm-manager]", prompt.UsedBy)
	}

	if iso, ok := idx["isolated"]; !ok {
		t.Error("isolated node dropped")
	} else if len(iso.DependsOn) != 0 || len(iso.UsedBy) != 0 {
		t.Errorf("isolated should have no connections, got depends=%v used=%v", iso.DependsOn, iso.UsedBy)
	}

	// A self-edge contributes no connection and (when the node is not otherwise
	// listed) no scenario — it is fully ignored.
	if _, ok := idx["loop"]; ok {
		t.Error("self-edge-only node should not appear")
	}
}

// TestPayloadStrings tolerates both the text fallback's native []string and
// Qdrant's round-tripped []interface{} payload shape.
func TestPayloadStrings(t *testing.T) {
	native := payloadStrings(map[string]any{"used_by": []string{"a", "b"}}, "used_by")
	if !reflect.DeepEqual(native, []string{"a", "b"}) {
		t.Errorf("native = %v", native)
	}
	roundTripped := payloadStrings(map[string]any{"used_by": []interface{}{"a", "b"}}, "used_by")
	if !reflect.DeepEqual(roundTripped, []string{"a", "b"}) {
		t.Errorf("round-tripped = %v", roundTripped)
	}
	if payloadStrings(nil, "used_by") != nil {
		t.Error("nil payload should yield nil")
	}
}
