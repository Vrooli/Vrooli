package methods_test

import (
	"testing"
	"web-search/internal/evaluation"
	"web-search/internal/methods"
)

func observation(id, method string, effort float64, support bool) evaluation.Observation {
	return evaluation.Observation{TaskID: id, MethodHash: method, Stratum: "default", Effort: effort, Support: support, Coverage: true, Provenance: "test"}
}

func TestCycleRoutesAndRequiresEvidenceAndGrant(t *testing.T) {
	engine := methods.NewEngine()
	cycle, err := engine.Evaluate(methods.Friction{CycleID: "empty", FailureClass: "fetch"}, "candidate", nil, nil, "")
	if err != nil || cycle.State != methods.StateInsufficient || methods.Route(methods.Friction{FailureClass: "fetch"}) != "collection" {
		t.Fatalf("wrong insufficient route: %+v %v", cycle, err)
	}
	base, candidate := []evaluation.Observation{observation("task", "base", 10, true)}, []evaluation.Observation{observation("task", "candidate", 8, true)}
	cycle, err = engine.Evaluate(methods.Friction{CycleID: "proposal", FailureClass: "method", SampleCount: 1}, "candidate", base, candidate, "")
	if err != nil || cycle.State != methods.StateProposal {
		t.Fatalf("expected proposal: %+v %v", cycle, err)
	}
	cycle, err = engine.Evaluate(methods.Friction{CycleID: "promoted", FailureClass: "method", SampleCount: 1}, "candidate", base, candidate, "grant-1")
	if err != nil || cycle.State != methods.StatePromoted {
		t.Fatalf("expected promotion: %+v %v", cycle, err)
	}
}

func TestCycleRejectsFasterIncompleteCandidateAndReplays(t *testing.T) {
	engine := methods.NewEngine()
	base, candidate := []evaluation.Observation{observation("task", "base", 10, true)}, []evaluation.Observation{observation("task", "candidate", 1, false)}
	in := methods.Friction{CycleID: "reject", FailureClass: "verification", SampleCount: 1}
	first, err := engine.Evaluate(in, "candidate", base, candidate, "grant")
	if err != nil || first.State != methods.StateRejected {
		t.Fatalf("expected rejection: %+v %v", first, err)
	}
	second, err := engine.Evaluate(in, "candidate", base, candidate, "grant")
	if err != nil || second.State != first.State || second.Report.Reason != first.Report.Reason {
		t.Fatalf("replay changed cycle: %+v %+v", first, second)
	}
}
