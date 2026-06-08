package measureindex

import (
	"context"
	"testing"

	measures "github.com/vrooli/measures-go"
)

// --- fake harvest seams ---

type fakeManifests map[string][]byte

func (f fakeManifests) Manifest(scenario string) ([]byte, error) { return f[scenario], nil }

type fakeLister []string

func (l fakeLister) Scenarios() ([]string, error) { return []string(l), nil }

// stubSchema returns a fixed proto param schema for the bound RPC so Assemble can
// join the manifest measure block against it without a real descriptor.
type stubSchema struct{}

func (stubSchema) RequestParams(_, _ string) ([]measures.ParamSchema, error) {
	return []measures.ParamSchema{{Name: "window", Type: "string"}}, nil
}

const backlogManifest = `{
  "name": "swarm-manager",
  "groups": [
    {
      "name": "backlog",
      "commands": [
        {
          "name": "completed",
          "binding": { "kind": "connect-rpc", "service": "StatsService", "method": "BacklogCompletionCount" },
          "governance": { "effect": "read", "run_eligible": true },
          "measure": {
            "intent": "How many backlog items were completed in a time window.",
            "questions": ["how many backlog items did we complete this week"],
            "params": { "window": { "type": "time_window", "default": "this_week" } },
            "result": { "kind": "scalar", "value_field": "count", "unit": "items", "summary_template": "{count} backlog items completed ({window})" }
          }
        }
      ]
    }
  ]
}`

func TestHarvest_AttributesScenarioAndAssembles(t *testing.T) {
	h := NewHarvester(
		fakeManifests{"swarm-manager": []byte(backlogManifest), "no-measures": []byte(`{"name":"no-measures","groups":[]}`)},
		fakeLister{"swarm-manager", "no-measures", "missing"},
		stubSchema{},
	)

	decls, err := h.Harvest(context.Background())
	if err != nil {
		t.Fatalf("Harvest: %v", err)
	}
	if len(decls) != 1 {
		t.Fatalf("expected exactly one harvested measure, got %d", len(decls))
	}
	d := decls[0]
	if d.Name != "backlog.completed" {
		t.Fatalf("expected measure name backlog.completed, got %q", d.Name)
	}
	if d.Scenario != "swarm-manager" {
		t.Fatalf("harvested measure must be attributed to its owning scenario, got %q", d.Scenario)
	}
	if d.Effect != measures.EffectRead {
		t.Fatalf("expected effect read, got %q", d.Effect)
	}
	wp, ok := d.Params["window"]
	if !ok {
		t.Fatal("expected a window param on the assembled declaration")
	}
	if wp.Type != measures.ParamTypeTimeWindow {
		t.Fatalf("manifest type annotation should upgrade window to time_window, got %q", wp.Type)
	}
}

func TestHarvest_EmptyFleetYieldsEmptyCorpus(t *testing.T) {
	h := NewHarvester(fakeManifests{}, fakeLister{"a", "b"}, stubSchema{})
	decls, err := h.Harvest(context.Background())
	if err != nil {
		t.Fatalf("Harvest: %v", err)
	}
	if len(decls) != 0 {
		t.Fatalf("expected empty corpus, got %d", len(decls))
	}
}
