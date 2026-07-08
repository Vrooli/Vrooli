package operatingmode

import "testing"

func lookup(fields map[string]any) FieldLookup {
	return NewMapFieldLookup(fields)
}

func TestGuardLeafOps(t *testing.T) {
	cases := []struct {
		name  string
		guard Guard
		out   map[string]any
		want  bool
	}{
		{"always matches empty", Guard{Op: GuardOpAlways}, map[string]any{}, true},
		{"eq bool true", Guard{Op: GuardOpEq, Field: "replan_needed", Value: true}, map[string]any{"replan_needed": true}, true},
		{"eq bool false-vs-true", Guard{Op: GuardOpEq, Field: "replan_needed", Value: true}, map[string]any{"replan_needed": false}, false},
		{"eq absent field", Guard{Op: GuardOpEq, Field: "replan_needed", Value: true}, map[string]any{}, false},
		{"ne absent field matches", Guard{Op: GuardOpNe, Field: "verdict", Value: "pass"}, map[string]any{}, true},
		{"ne present differing", Guard{Op: GuardOpNe, Field: "verdict", Value: "pass"}, map[string]any{"verdict": "fail"}, true},
		{"ne present equal", Guard{Op: GuardOpNe, Field: "verdict", Value: "pass"}, map[string]any{"verdict": "pass"}, false},
		{"eq string dotted", Guard{Op: GuardOpEq, Field: "progress.decision", Value: "continue"}, map[string]any{"progress": map[string]any{"decision": "continue"}}, true},
		{"in member", Guard{Op: GuardOpIn, Field: "disposition", Values: []any{"wont_fix", "escalate"}}, map[string]any{"disposition": "escalate"}, true},
		{"in non-member", Guard{Op: GuardOpIn, Field: "disposition", Values: []any{"wont_fix"}}, map[string]any{"disposition": "reproduce"}, false},
		{"not_in non-member matches", Guard{Op: GuardOpNotIn, Field: "disposition", Values: []any{"wont_fix"}}, map[string]any{"disposition": "reproduce"}, true},
		{"not_in absent matches", Guard{Op: GuardOpNotIn, Field: "disposition", Values: []any{"wont_fix"}}, map[string]any{}, true},
		{"gte equal", Guard{Op: GuardOpGte, Field: "severity", Value: float64(3)}, map[string]any{"severity": float64(3)}, true},
		{"gte below", Guard{Op: GuardOpGte, Field: "severity", Value: float64(3)}, map[string]any{"severity": float64(2)}, false},
		{"lt below", Guard{Op: GuardOpLt, Field: "severity", Value: float64(3)}, map[string]any{"severity": float64(2)}, true},
		{"gt non-numeric field", Guard{Op: GuardOpGt, Field: "severity", Value: float64(1)}, map[string]any{"severity": "high"}, false},
		{"exists present", Guard{Op: GuardOpExists, Field: "handoff"}, map[string]any{"handoff": map[string]any{"frontier": "x"}}, true},
		{"exists absent", Guard{Op: GuardOpExists, Field: "handoff"}, map[string]any{}, false},
		{"exists explicit null", Guard{Op: GuardOpExists, Field: "handoff"}, map[string]any{"handoff": nil}, false},
		{"not_exists absent", Guard{Op: GuardOpNotExists, Field: "handoff"}, map[string]any{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.guard.Eval(lookup(tc.out)); got != tc.want {
				t.Fatalf("Eval = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGuardComposites(t *testing.T) {
	allGuard := Guard{All: []Guard{
		{Op: GuardOpEq, Field: "reproduced", Value: true},
		{Op: GuardOpGte, Field: "severity", Value: float64(3)},
	}}
	if !allGuard.Eval(lookup(map[string]any{"reproduced": true, "severity": float64(4)})) {
		t.Fatalf("all guard should match reproduced=true severity=4")
	}
	if allGuard.Eval(lookup(map[string]any{"reproduced": true, "severity": float64(2)})) {
		t.Fatalf("all guard should not match severity below threshold")
	}

	anyGuard := Guard{Any: []Guard{
		{Op: GuardOpEq, Field: "verdict", Value: "accept"},
		{Op: GuardOpEq, Field: "verdict", Value: "accepted"},
	}}
	if !anyGuard.Eval(lookup(map[string]any{"verdict": "accepted"})) {
		t.Fatalf("any guard should match one branch")
	}
	if anyGuard.Eval(lookup(map[string]any{"verdict": "reject"})) {
		t.Fatalf("any guard should not match when no branch matches")
	}

	notGuard := Guard{Not: &Guard{Op: GuardOpEq, Field: "verdict", Value: "pass"}}
	if !notGuard.Eval(lookup(map[string]any{"verdict": "fail"})) {
		t.Fatalf("not(eq pass) should match verdict=fail")
	}
	if notGuard.Eval(lookup(map[string]any{"verdict": "pass"})) {
		t.Fatalf("not(eq pass) should not match verdict=pass")
	}
	if !notGuard.Eval(lookup(map[string]any{})) {
		t.Fatalf("not(eq pass) should match when verdict absent (eq is false)")
	}
}

func TestGuardStructValueCoercion(t *testing.T) {
	// A struct-valued payload field (as the runtime stores ProgressState) must
	// resolve identically to a decoded JSON object.
	payload := map[string]any{"progress": ProgressState{Decision: ProgressContinue}}
	g := Guard{Op: GuardOpEq, Field: "progress.decision", Value: "continue"}
	if !g.Eval(lookup(payload)) {
		t.Fatalf("guard should resolve progress.decision through a struct value")
	}
}

func TestValidateGuardRejectsMalformed(t *testing.T) {
	cases := []struct {
		name  string
		guard Guard
	}{
		{"empty", Guard{}},
		{"unknown op", Guard{Op: "between", Field: "x", Value: 1}},
		{"eq without value", Guard{Op: GuardOpEq, Field: "x"}},
		{"eq without field", Guard{Op: GuardOpEq, Value: 1}},
		{"in without values", Guard{Op: GuardOpIn, Field: "x"}},
		{"always with field", Guard{Op: GuardOpAlways, Field: "x"}},
		{"composite plus op", Guard{Op: GuardOpEq, Field: "x", Value: 1, All: []Guard{{Op: GuardOpAlways}}}},
		{"two composites", Guard{All: []Guard{{Op: GuardOpAlways}}, Any: []Guard{{Op: GuardOpAlways}}}},
		{"bad field path", Guard{Op: GuardOpEq, Field: "Bad.Path", Value: 1}},
		{"nested composite invalid", Guard{Not: &Guard{Op: "nope", Field: "x", Value: 1}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateGuard(tc.guard); err == nil {
				t.Fatalf("validateGuard(%+v) = nil, want error", tc.guard)
			}
		})
	}
}

func TestValidateGuardAcceptsWellFormed(t *testing.T) {
	valid := []Guard{
		{Op: GuardOpAlways},
		{Op: GuardOpEq, Field: "replan_needed", Value: true},
		{Op: GuardOpIn, Field: "disposition", Values: []any{"a", "b"}},
		{Op: GuardOpExists, Field: "handoff"},
		{All: []Guard{{Op: GuardOpEq, Field: "reproduced", Value: true}, {Op: GuardOpGte, Field: "severity", Value: float64(3)}}},
		{Not: &Guard{Op: GuardOpEq, Field: "verdict", Value: "pass"}},
	}
	for i, g := range valid {
		if err := validateGuard(g); err != nil {
			t.Fatalf("validateGuard(valid[%d]) = %v", i, err)
		}
	}
}
