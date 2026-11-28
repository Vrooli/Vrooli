package validation

import (
	"context"
	"testing"

	measures "github.com/vrooli/measures-go"
	"github.com/vrooli/measures-go/manifestscan"
)

// --- fakes ---------------------------------------------------------------

type fakeManifests map[string][]byte

func (f fakeManifests) Manifest(scenario string) ([]byte, error) { return f[scenario], nil }

type fakeDomains map[string][]DerivedDomain

func (f fakeDomains) StatefulDomains(scenario string) ([]DerivedDomain, error) {
	return f[scenario], nil
}

// Mode: a fakeDomains supplying derived domains models a conformant scenario
// (a v1/domain/ folder). Fallback/illegal-declaration behaviour is exercised at
// the pure Classify level (classify_test.go) where Mode is set directly.
func (f fakeDomains) Mode(string) (Mode, error) { return ModeConformant, nil }

// fakeSchema returns a fixed param schema per (service.method).
type fakeSchema map[string][]measures.ParamSchema

func (f fakeSchema) RequestParams(service, method string) ([]measures.ParamSchema, error) {
	return f[service+"."+method], nil
}

type fakeProber struct {
	ok      bool
	skipped bool
	detail  string
}

func (f fakeProber) Probe(_ context.Context, _ string, _ measures.MeasureDeclaration) (bool, string, bool) {
	return f.ok, f.detail, f.skipped
}

type fakeLister []string

func (f fakeLister) Scenarios() ([]string, error) { return []string(f), nil }

// manifestWith builds a one-command manifest declaring a measure on the given
// domain, bound to StatsService.<method>.
const manifestBacklog = `{
  "name": "swarm-manager",
  "groups": [
    {"name": "backlog", "commands": [
      {"name": "completed",
       "binding": {"kind": "connect-rpc", "service": "StatsService", "method": "BacklogCompleted"},
       "governance": {"effect": "read", "run_eligible": true},
       "measure": {
         "intent": "How many backlog items completed in a window.",
         "questions": ["how many backlog items did we complete this week"],
         "params": {"window": {"default": "this_week"}},
         "result": {"kind": "scalar", "value_field": "count", "unit": "items",
                    "summary_template": "{count} backlog items ({window})"}
       }}
    ]}
  ],
  "measures": {"omitted": [{"domain": "queue", "reason": "ephemeral"}]}
}`

func timeWindowSchema() fakeSchema {
	return fakeSchema{
		"StatsService.BacklogCompleted": {
			{Name: "window", Type: measures.ParamTypeTimeWindow, MessageType: measures.TimeWindowMessageName, Optional: true},
		},
	}
}

// --- tests ---------------------------------------------------------------

func TestValidator_FullCoveragePasses(t *testing.T) {
	v := NewValidator(
		fakeManifests{"swarm-manager": []byte(manifestBacklog)},
		fakeDomains{"swarm-manager": {
			{Name: "backlog", Stateful: true},
			{Name: "queue", Stateful: true},
			{Name: "settings", Stateful: false, Note: "stateless"},
		}},
		timeWindowSchema(),
	)
	rep, err := v.ValidateScenario(context.Background(), "swarm-manager", false)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Passed {
		t.Fatalf("expected pass; findings=%+v", rep.Findings)
	}
	if d := domainBy(rep, "backlog"); d == nil || d.Status != StatusCovered || d.Tier != manifestscan.TierFull {
		t.Fatalf("backlog: want covered/full, got %+v", d)
	}
	if d := domainBy(rep, "queue"); d == nil || d.Status != StatusWaived {
		t.Fatalf("queue: want waived, got %+v", d)
	}
	if d := domainBy(rep, "settings"); d == nil || d.Status != StatusNotExpected {
		t.Fatalf("settings: want not_expected, got %+v", d)
	}
}

func TestValidator_UncoveredDomainFails(t *testing.T) {
	v := NewValidator(
		fakeManifests{"swarm-manager": []byte(manifestBacklog)},
		fakeDomains{"swarm-manager": {
			{Name: "backlog", Stateful: true},
			{Name: "captures", Stateful: true}, // neither covered nor waived
		}},
		timeWindowSchema(),
	)
	rep, err := v.ValidateScenario(context.Background(), "swarm-manager", false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Passed {
		t.Fatal("expected fail (captures uncovered)")
	}
	if f := findingBy(rep, "measures.uncovered-domain"); f == nil {
		t.Fatalf("want uncovered-domain ERROR, findings=%+v", rep.Findings)
	}
}

func TestValidator_NoManifestNoMeasures(t *testing.T) {
	// A scenario with no manifest but stateful domains -> all uncovered ERROR.
	v := NewValidator(
		fakeManifests{},
		fakeDomains{"x": {{Name: "orders", Stateful: true}}},
		fakeSchema{},
	)
	rep, err := v.ValidateScenario(context.Background(), "x", false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Passed {
		t.Fatal("expected fail (orders uncovered, no manifest)")
	}
}

func TestValidator_ProbeHollowIsError(t *testing.T) {
	v := NewValidator(
		fakeManifests{"swarm-manager": []byte(manifestBacklog)},
		fakeDomains{"swarm-manager": {{Name: "backlog", Stateful: true}, {Name: "queue", Stateful: true}}},
		timeWindowSchema(),
		WithProber(fakeProber{ok: false, skipped: false, detail: "endpoint 404"}),
	)
	rep, err := v.ValidateScenario(context.Background(), "swarm-manager", true)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Passed {
		t.Fatal("expected fail (hollow declaration)")
	}
	if f := findingBy(rep, "measures.hollow-declaration"); f == nil || f.Severity != SeverityError {
		t.Fatalf("want hollow-declaration ERROR, findings=%+v", rep.Findings)
	}
}

func TestValidator_ProbeSkippedIsNotError(t *testing.T) {
	v := NewValidator(
		fakeManifests{"swarm-manager": []byte(manifestBacklog)},
		fakeDomains{"swarm-manager": {{Name: "backlog", Stateful: true}, {Name: "queue", Stateful: true}}},
		timeWindowSchema(),
		WithProber(fakeProber{skipped: true, detail: "scenario not running"}),
	)
	rep, err := v.ValidateScenario(context.Background(), "swarm-manager", true)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Passed {
		t.Fatalf("skipped probe must not fail; findings=%+v", rep.Findings)
	}
	if len(rep.SkippedScanners) == 0 {
		t.Fatal("want a skipped-scanner note for the unreachable probe")
	}
}

func TestValidator_FleetRollup(t *testing.T) {
	v := NewValidator(
		fakeManifests{"swarm-manager": []byte(manifestBacklog)},
		fakeDomains{
			"swarm-manager":  {{Name: "backlog", Stateful: true}, {Name: "queue", Stateful: true}},
			"empty-scenario": {{Name: "orders", Stateful: true}},
		},
		timeWindowSchema(),
		WithScenarioLister(fakeLister{"swarm-manager", "empty-scenario"}),
	)
	entries, err := v.ListFleetCoverage(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
	var sm, es FleetEntry
	for _, e := range entries {
		switch e.Scenario {
		case "swarm-manager":
			sm = e
		case "empty-scenario":
			es = e
		}
	}
	if !sm.Passed || sm.Covered != 1 || sm.Waived != 1 {
		t.Fatalf("swarm-manager rollup wrong: %+v", sm)
	}
	if es.Passed || es.Uncovered != 1 {
		t.Fatalf("empty-scenario should fail w/ 1 uncovered: %+v", es)
	}
}
