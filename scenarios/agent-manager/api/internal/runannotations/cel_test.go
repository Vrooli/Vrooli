package runannotations

import (
	"testing"
)

func TestCandidateCELUsesDocumentedFeatures(t *testing.T) {
	pack, err := LoadCompiled(packPathForTest(t, `[{
		"id":"agent.spinning","group":"agent","scope":"run","method":"decision",
		"question":{"type":"noul","instructions":"x","true_description":"t","false_description":"f"},
		"evidence":[],"candidates":"duration_min >= 8 || failed_commands >= 2 || repeated_paths >= 3",
		"gate":{"precision":0.5,"recall":0.5,"min_gold":1},
		"search":{"facet":"f"},"compatibility":{"change":"breaking"}}]`))
	if err != nil {
		t.Fatal(err)
	}
	compiled, _ := pack.Label("agent.spinning")
	evaluator := celEvaluator{CostLimit: defaultCostLimit}
	cases := []struct {
		features RunFeatures
		want     bool
	}{
		{RunFeatures{DurationMin: 7}, false},
		{RunFeatures{DurationMin: 8}, true},
		{RunFeatures{FailedCommands: 2}, true},
		{RunFeatures{RepeatedPaths: 3}, true},
		{RunFeatures{RepeatedPaths: 2, FailedCommands: 1, DurationMin: 2}, false},
	}
	for i, tc := range cases {
		got, err := evaluator.evalBool(compiled.candidates, tc.features.Activation())
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if got != tc.want {
			t.Fatalf("case %d: got %t want %t", i, got, tc.want)
		}
	}
}

func TestCELCostLimitRefusesExpensiveRules(t *testing.T) {
	pack, err := LoadPack(packPathForTest(t, `[{
		"id":"costly","group":"agent","scope":"run","method":"deterministic",
		"rule":"[1,2,3,4,5,6,7,8].map(x, [1,2,3,4,5,6,7,8].map(y, x * y)).size() > 0",
		"evidence":[],"gate":{"precision":0.5,"recall":0.5,"min_gold":1},
		"search":{"facet":"f"},"compatibility":{"change":"breaking"}}]`))
	if err != nil {
		t.Fatal(err)
	}
	compiledPack, err := pack.CompileWithCostLimit(1)
	if err != nil {
		t.Fatal(err)
	}
	compiled, _ := compiledPack.Label("costly")
	evaluator := celEvaluator{}
	if _, err := evaluator.evalValue(compiled.rule, (RunFeatures{}).Activation()); err == nil {
		t.Fatal("cost limit 1 accepted an expensive expression")
	}
}

func TestDeterministicRuleDistinguishesFalseFromUnknown(t *testing.T) {
	pack, err := LoadCompiled(packPathForTest(t, `[{
		"id":"run.type","group":"run","scope":"run","method":"deterministic",
		"rule":"workload_kind == 'workflow_node' ? 'workflow' : ''",
		"evidence":[],"gate":{"precision":0.5,"recall":0.5,"min_gold":1},
		"search":{"facet":"f"},"compatibility":{"change":"breaking"}}]`))
	if err != nil {
		t.Fatal(err)
	}
	compiled, _ := pack.Label("run.type")
	evaluator := celEvaluator{}
	value, decided, err := evaluator.evalString(compiled.rule, (RunFeatures{WorkloadKind: "workflow_node"}).Activation())
	if err != nil || !decided || value != "workflow" {
		t.Fatalf("workflow: %q %t %v", value, decided, err)
	}
	value, decided, err = evaluator.evalString(compiled.rule, (RunFeatures{WorkloadKind: "adhoc"}).Activation())
	if err != nil || decided || value != "" {
		t.Fatalf("unknown: %q %t %v", value, decided, err)
	}
}
