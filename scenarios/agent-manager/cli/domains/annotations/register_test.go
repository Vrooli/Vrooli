package annotations

import (
	"bytes"
	"strings"
	"testing"
)

func TestRegisterDeclaresEveryAnnotationCommand(t *testing.T) {
	group := Register()
	if group.Name != "annotations" || !group.NeedsAPI {
		t.Fatalf("group = %+v", group)
	}
	want := []string{"packs", "labels", "coverage", "show", "run", "evaluate", "scheduler", "benchmark", "gold"}
	if len(group.Subcommands) != len(want) {
		t.Fatalf("commands = %d, want %d", len(group.Subcommands), len(want))
	}
	for i, command := range group.Subcommands {
		if command.Name != want[i] {
			t.Fatalf("command %d = %q, want %q", i, command.Name, want[i])
		}
		if command.RunCtx == nil {
			t.Fatalf("%s has no RunCtx", command.Name)
		}
	}
}

func TestEvaluateBodyValidatesSinceAndCopiesRunIDs(t *testing.T) {
	runIDs := []string{"a", "b"}
	body, err := evaluateBody("2026-11-01T00:00:00Z", 10, runIDs, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if body["since"] != "2026-11-01T00:00:00Z" || body["limit"] != 10 || body["include_optional"] != true {
		t.Fatalf("body = %+v", body)
	}
	runIDs[0] = "changed"
	if body["run_ids"].([]string)[0] == "changed" {
		t.Fatal("run ids were not copied")
	}
	if _, err := evaluateBody("yesterday", 1, nil, false, false); err == nil {
		t.Fatal("non-RFC3339 since was accepted")
	}
	body, err = evaluateBody("", 0, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := body["since"]; exists {
		t.Fatalf("empty since was sent: %+v", body)
	}
	if _, exists := body["run_ids"]; exists {
		t.Fatalf("empty run ids were sent: %+v", body)
	}
}

func TestParseHelpers(t *testing.T) {
	if got := parseLabels(" a, ,b "); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("parseLabels = %#v", got)
	}
	if got := normalizedWindow(" "); got != "7d" {
		t.Fatalf("window = %q", got)
	}
	if _, err := parsePositiveInt("0", "limit"); err == nil {
		t.Fatal("zero limit accepted")
	}
	if value, err := parsePositiveInt("25", "limit"); err != nil || value != 25 {
		t.Fatalf("limit = %d %v", value, err)
	}
}

func TestCoverageRenderingNeverClaimsNoFriction(t *testing.T) {
	var out bytes.Buffer
	renderCoverage(&out, coverageResponse{
		Window:       windowRange{Since: "2026-11-11T00:00:00Z", Until: "2026-11-18T00:00:00Z"},
		Pack:         packRef{Name: "default", Version: "1"},
		EligibleRuns: 0, ObservedRuns: 0, Coverage: 0,
		Weighting: weighting{Mode: "census+sample", SampleRate: 0.2},
		Labels: []coverageLabel{
			{ID: "agent.spinning", State: "not_observed", Gate: gateStatus{State: "unmeasured"}, Eligible: 0},
		},
		Notes: []string{"no annotations recorded in this window: coverage is unknown, not zero friction"},
	})
	text := out.String()
	if !strings.HasPrefix(text, "Coverage:") {
		t.Fatalf("coverage line is not first:\n%s", text)
	}
	if !strings.Contains(text, "unknown, not zero") {
		t.Fatalf("coverage text hides unknown coverage:\n%s", text)
	}
	if strings.Contains(strings.ToLower(text), "no friction\n") {
		t.Fatalf("coverage text claims no friction:\n%s", text)
	}
	if !strings.Contains(text, "gate=unmeasured") {
		t.Fatalf("coverage text hides the gate:\n%s", text)
	}
	if !strings.Contains(text, "not_observed") {
		t.Fatalf("coverage text omits the state:\n%s", text)
	}
}

func TestCoverageRenderingPrintsRateOnlyWhenPresent(t *testing.T) {
	rate := 0.174
	var out bytes.Buffer
	renderCoverage(&out, coverageResponse{
		Pack: packRef{Name: "default", Version: "1"}, EligibleRuns: 818, ObservedRuns: 818, Coverage: 1,
		Labels: []coverageLabel{
			{ID: "agent.spinning", State: "proven", Gate: gateStatus{State: "proven"}, Observed: 142, Eligible: 818, Rate: &rate},
		},
	})
	if !strings.Contains(out.String(), "rate=17.4%") {
		t.Fatalf("proven rate missing:\n%s", out.String())
	}
}

func TestRunRenderingShowsStateGateAndEvidence(t *testing.T) {
	probability := 0.72
	var out bytes.Buffer
	renderRun(&out, runView{
		RunID: "run-1", Pack: "default", Version: "1", State: "read",
		Coverage: runCoverage{Observed: 1, Labels: 11},
		Labels: []annotationView{{
			ID: "agent.spinning", State: "observed", Method: "decision",
			Probability: &probability, GateState: "unmeasured", EvidenceEventIDs: []string{"event-1"},
		}},
		Counts:  stateCounts{Observed: 1},
		Missing: []string{"operator.valence"},
	})
	text := out.String()
	if !strings.HasPrefix(text, "Coverage:") {
		t.Fatalf("run coverage line is not first:\n%s", text)
	}
	for _, want := range []string{"agent.spinning", "observed", "p=0.72", "gate=unmeasured", "event-1", "operator.valence", "unknown, not absent"} {
		if !strings.Contains(text, want) {
			t.Fatalf("run text missing %q:\n%s", want, text)
		}
	}
}

func TestEmptyRunRenderingIsUnknownNotNoFriction(t *testing.T) {
	var out bytes.Buffer
	renderRun(&out, runView{RunID: "run-1", Pack: "default", Version: "1", Coverage: runCoverage{Labels: 11}})
	text := out.String()
	if !strings.HasPrefix(text, "Coverage:") {
		t.Fatalf("run coverage line is not first: %s", text)
	}
	if !strings.Contains(text, "unknown, not zero friction") {
		t.Fatalf("empty run = %s", text)
	}
	if strings.Contains(strings.ToLower(text), "no friction.") {
		t.Fatalf("empty run claims no friction: %s", text)
	}
}

func TestEvaluateRenderingExplainsATruncatedPass(t *testing.T) {
	var out bytes.Buffer
	renderEvaluate(&out, evaluateResult{
		Submitted: 8, Observed: 8, Coverage: 1, Truncated: true, Remaining: 12, Passes: 1,
	})
	text := out.String()
	if !strings.Contains(text, "12 run(s) still queued") || !strings.Contains(text, "run the command again") {
		t.Fatalf("truncated pass output = %q", text)
	}
}

func TestAccumulateEvaluateAddsPassesAndKeepsTheLastSnapshot(t *testing.T) {
	total := evaluateResult{}
	accumulateEvaluate(&total, evaluateResult{
		Submitted: 8, Observed: 8, Counts: stateCounts{Observed: 40}, Truncated: true, Remaining: 12,
	})
	accumulateEvaluate(&total, evaluateResult{
		Submitted: 12, Observed: 10, Unavailable: 2, Counts: stateCounts{Observed: 60, Unavailable: 2},
	})
	if total.Submitted != 20 || total.Observed != 18 || total.Unavailable != 2 || total.Counts.Observed != 100 {
		t.Fatalf("accumulated = %+v, want additive counts", total)
	}
	if total.Truncated || total.Coverage != 0.9 {
		t.Fatalf("accumulated = %+v, want a completed pass at 90%% coverage", total)
	}
}
