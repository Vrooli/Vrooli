package annotations

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegisterDeclaresEveryAnnotationCommand(t *testing.T) {
	group := Register()
	if group.Name != "annotations" || !group.NeedsAPI {
		t.Fatalf("group = %+v", group)
	}
	want := []string{"packs", "labels", "coverage", "show", "run", "evaluate", "scheduler", "benchmark", "calibration", "gold", "provenance", "series"}
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
	body, err := evaluateBody(evaluateParams{Since: "2026-11-01T00:00:00Z", Limit: 10, RunIDs: runIDs, IncludeOptional: true})
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
	if _, err := evaluateBody(evaluateParams{Since: "yesterday", Limit: 1}); err == nil {
		t.Fatal("non-RFC3339 since was accepted")
	}
	body, err = evaluateBody(evaluateParams{})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"since", "run_ids", "labels", "pin_resource_role", "role"} {
		if _, exists := body[key]; exists {
			t.Fatalf("empty %s was sent: %+v", key, body)
		}
	}
}

func TestEvaluateBodyCarriesThePinnedCandidateAndRole(t *testing.T) {
	body, err := evaluateBody(evaluateParams{Limit: 5, PinResourceRole: "decide.local.accurate", Role: "decide.candidates", Labels: []string{"work.blocked_by_tool"}})
	if err != nil {
		t.Fatal(err)
	}
	if body["pin_resource_role"] != "decide.local.accurate" || body["role"] != "decide.candidates" || body["labels"].([]string)[0] != "work.blocked_by_tool" {
		t.Fatalf("body = %+v", body)
	}
}

func TestExplicitSubjectsUnionNamedRunsAndGoldFilesWithoutDuplicates(t *testing.T) {
	gold := writeFile(t, "gold.jsonl", `{"run_id":"r2","label":"work.blocked_by_tool","positive":true}
{"run_id":"r3","label":"work.blocked_by_tool","positive":false}
{"run_id":"r2","label":"work.blocked_by_tool","positive":false}
`)
	got, err := explicitSubjects("", []string{"r1", " r2 ", ""}, []string{gold})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got.RunIDs, ",") != "r1,r2,r3" || len(got.MessageIDs) != 0 || strings.Join(got.Labels, ",") != "work.blocked_by_tool" {
		t.Fatalf("subjects = %+v, want first-seen runs once and the one label the file judges", got)
	}
	if _, err := explicitSubjects("", nil, []string{filepath.Join(t.TempDir(), "missing.jsonl")}); err == nil || !strings.Contains(err.Error(), "--runs-from") {
		t.Fatalf("err = %v, want the missing gold file named", err)
	}
	unfilled := writeFile(t, "draft.jsonl", `{"run_id":"r9","label":"work.blocked_by_tool","positive":null}
`)
	if _, err := explicitSubjects("", nil, []string{unfilled}); err == nil {
		t.Fatal("an unfilled draft supplied run ids")
	}
}

func TestExplicitSubjectsCarryTheMessagesAMessageGoldNames(t *testing.T) {
	gold := writeFile(t, "mood.jsonl", `{"run_id":"r1","scope_id":"r1-m2","label":"operator.dissatisfied","positive":true}
{"run_id":"r1","scope_id":"r1-m5","labels":{"operator.pleased":false,"operator.dissatisfied":true}}
{"run_id":"r4","scope_id":"r4-m1","label":"operator.dissatisfied","positive":false}
`)
	got, err := explicitSubjects("", nil, []string{gold})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got.RunIDs, ",") != "r1,r4" || strings.Join(got.MessageIDs, ",") != "r1-m2,r1-m5,r4-m1" ||
		strings.Join(got.Labels, ",") != "operator.dissatisfied,operator.pleased" {
		t.Fatalf("subjects = %+v", got)
	}
	body, err := evaluateBody(evaluateParams{Limit: 3, RunIDs: got.RunIDs, MessageIDs: got.MessageIDs, PinResourceRole: "decide.local.accurate"})
	if err != nil {
		t.Fatal(err)
	}
	if ids := body["message_ids"].([]string); len(ids) != 3 || ids[0] != "r1-m2" {
		t.Fatalf("body = %+v, want the named messages sent so the API answers only them", body)
	}
	rows := goldRowsFor([]goldFileRow{{RunID: "r1", ScopeID: "r1-m2", Label: "operator.dissatisfied", Positive: boolPtr(true)}}, "operator.dissatisfied")
	if len(rows) != 1 || rows[0].ScopeID != "r1-m2" {
		t.Fatalf("wire rows = %+v, want the message id carried to the API", rows)
	}
}

func TestEvaluateRenderingReportsPinningAndEscalationSavings(t *testing.T) {
	var out bytes.Buffer
	renderEvaluate(&out, evaluateResult{
		Submitted: 10, Observed: 10, Coverage: 1, Pinned: "decide.local.accurate",
		Escalation: &escalationStats{EligibleStates: 6, Calls: 1, StatesAsked: 6, SkippedCalls: 3, SkippedStates: 18, SameModelAnswers: 6, CallSeconds: 1.5},
	})
	text := out.String()
	for _, want := range []string{"Pinned to decide.local.accurate", "benchmark --pinned --resource-role decide.local.accurate", "3 call(s) skipped for 18 state(s)", "same model"} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q:\n%s", want, text)
		}
	}
}

func TestCoverageRenderingShowsEachModelAndWhyItIsOrIsNotCounted(t *testing.T) {
	rate := 1.0
	var out bytes.Buffer
	renderCoverage(&out, coverageResponse{
		Pack: packRef{Name: "default", Version: "1"}, EligibleRuns: 4, ObservedRuns: 4, Coverage: 1,
		Provenance: &provenanceSummary{RunsAnnotated: 4, RunsHumanText: 1, RunsBriefOnly: 2, RunsNoOperatorText: 1, HumanMessages: 5, BriefMessages: 2},
		Labels: []coverageLabel{{
			ID: "work.blocked_by_tool", State: "partial", Gate: gateStatus{State: "proven"}, Observed: 2, Eligible: 2, Rate: &rate,
			Population: &population{Basis: "human_operator", Unit: "message", Eligible: 5},
			Models: []modelCoverage{
				{Model: "winnow:e4b", ResourceRole: "decide.local.accurate", Observed: 1, Positives: 1, Proven: true, Gate: gateStatus{State: "proven", Gold: 60, Precision: 0.9, Recall: 0.8}},
				{Model: "laya:en", ResourceRole: "decide.local", Observed: 1, Positives: 1, Gate: gateStatus{State: "unmeasured", Reason: "no measurement for this model"}},
			},
		}},
	})
	text := out.String()
	for _, want := range []string{"1 of 4 annotated run(s) carry a person's words (5 message(s))", "against 5 message(s) a person wrote", "counted in the rate", "not counted in the rate: no measurement for this model"} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q:\n%s", want, text)
		}
	}
}

func TestSeriesRenderingLabelsAnUnprovenMeanAsEvidence(t *testing.T) {
	mean := -0.4
	var out bytes.Buffer
	renderSeries(&out, seriesResult{
		Label: "operator.dissatisfied", Bucket: "day", Messages: 3,
		Points: []seriesPoint{{Start: "2026-11-17T00:00:00Z", Messages: 3, Mean: &mean, FrictionPositives: 1}},
		Note:   "3 of 3 message(s) come from a model with no clearing hold-out proof for this definition; the mean is evidence of mood, not a measurement",
	})
	text := out.String()
	if !strings.Contains(text, "mean=-0.40") || !strings.Contains(text, "proven_mean=-") || !strings.Contains(text, "not a measurement") {
		t.Fatalf("series output = %s", text)
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
			{ID: "work.blocked_by_tool", State: "not_observed", Gate: gateStatus{State: "unmeasured"}, Eligible: 0},
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
			{ID: "work.blocked_by_tool", State: "proven", Gate: gateStatus{State: "proven"}, Observed: 142, Eligible: 818, Rate: &rate},
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
			ID: "work.blocked_by_tool", State: "observed", Method: "decision",
			Probability: &probability, GateState: "unmeasured", EvidenceEventIDs: []string{"event-1"},
		}},
		Counts:  stateCounts{Observed: 1},
		Missing: []string{"operator.dissatisfied"},
	})
	text := out.String()
	if !strings.HasPrefix(text, "Coverage:") {
		t.Fatalf("run coverage line is not first:\n%s", text)
	}
	for _, want := range []string{"work.blocked_by_tool", "observed", "p=0.72", "gate=unmeasured", "event-1", "operator.dissatisfied", "unknown, not absent"} {
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

func TestEvaluateBodyForcesOnlyTheLabelsItNames(t *testing.T) {
	body, err := evaluateBody(evaluateParams{Limit: 5, ForceLabels: []string{"a.b"}})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := body["force_labels"].([]string); len(got) != 1 || got[0] != "a.b" {
		t.Fatalf("force_labels = %v", body["force_labels"])
	}
	if _, bare := body["force"]; bare {
		t.Fatal("a bare force is gone: it re-asked every label a composite reads")
	}
	none, err := evaluateBody(evaluateParams{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := none["force_labels"]; ok {
		t.Fatalf("an evaluation that forces nothing sent %v", none["force_labels"])
	}
}
