package usage

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

// annotationGateFixture is what usage.annotation_gate returns for a window with
// a label proven for one model, a mood label measured only on its tuning rows,
// and a label that was never observed.
var annotationGateFixture = []map[string]string{
	{"kind": "coverage", "pack": "default", "pack_version": "2026-11-20.7", "eligible_runs": "818", "observed_runs": "557", "coverage": "0.681", "human_text_runs": "26", "human_messages": "436", "stale_runs": "4"},
	{"kind": "label", "label": "operator.dissatisfied", "group": "operator", "scope": "message", "state": "partial", "friction": "true",
		"observed": "40", "eligible": "436", "proven_observed": "30", "rate": "0.1", "population_unit": "message",
		"gate_state": "proven", "gate_model": "winnow:e4b", "gate_role": "decide.local.accurate", "split": "holdout",
		"precision": "0.9", "recall": "0.8", "gold": "128", "gold_positives": "15", "ece": "0.042", "temperature": "1.4", "calibration": "calibrated",
		"target_precision": "0.85", "target_recall": "0.7", "min_gold": "40"},
	{"kind": "model", "label": "operator.dissatisfied", "model": "winnow:e4b", "resource_role": "decide.local.accurate", "observed": "30", "proven": "true", "gate_state": "proven"},
	{"kind": "model", "label": "operator.dissatisfied", "model": "laya:en", "resource_role": "decide.local", "observed": "10", "proven": "false", "gate_state": "unmeasured"},
	{"kind": "label", "label": "operator.pleased", "group": "operator", "scope": "message", "state": "unproven", "friction": "false",
		"observed": "40", "eligible": "436", "proven_observed": "0", "population_unit": "message",
		"gate_state": "tuning_only", "gate_model": "laya:en", "gate_role": "decide.local", "split": "tuning", "gold": "127",
		"target_precision": "0.85", "target_recall": "0.7", "min_gold": "40"},
	{"kind": "label", "label": "agent.discovery_slow", "group": "agent", "scope": "run", "state": "not_observed", "friction": "true",
		"observed": "0", "eligible": "0", "gate_state": "unmeasured", "target_precision": "0.85", "target_recall": "0.7", "min_gold": "40"},
}

func runAnnotations(t *testing.T, rows []map[string]string, args ...string) (string, map[string]string) {
	t.Helper()
	body, err := json.Marshal(map[string]any{"fields": rows})
	if err != nil {
		t.Fatal(err)
	}
	var sent map[string]string
	var measure string
	cmd := annotationsCommand(func(_ cliapp.RunContext, id string, params map[string]string) ([]byte, error) {
		measure, sent = id, params
		return body, nil
	})
	var stdout bytes.Buffer
	ctx, err := cliapp.NewTestRunContextFromArgs(cmd.Args, args, nil, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatalf("usage annotations %v: %v", args, err)
	}
	if measure != annotationGateMeasure {
		t.Fatalf("measure = %q, want %q", measure, annotationGateMeasure)
	}
	return stdout.String(), sent
}

func TestAnnotationsCommandNamesTheModelBehindEveryRateAndNeverRatesAnUnprovenLabel(t *testing.T) {
	out, sent := runAnnotations(t, annotationGateFixture)
	if sent["window"] != "last_7d" {
		t.Fatalf("window = %v", sent)
	}
	for _, want := range []string{
		"Inferred annotations: pack default 2026-11-20.7",
		"Coverage: 68% (557 of 818 runs observed)",
		"Human-written operator text: 26 runs, 436 messages",
		"4 runs not yet classified for operator text",
		"[partial] operator.dissatisfied · observed 40/436 messages · 30 from a proven model",
		"proof: winnow:e4b@decide.local.accurate, calibrated T=1.4 · hold-out precision 0.90 recall 0.80 ECE 0.042 · gold 128 (15 positive)",
		"rate 10.0% of 30 proven observations",
		"laya:en@decide.local · 10 observed · unmeasured",
		"[unproven] operator.pleased",
		"no rate: laya:en measured on its tuning rows only; no hold-out",
		"1 label(s) have no observation in this window (unknown, not zero)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("human output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "operator.pleased") && strings.Contains(strings.SplitAfter(out, "operator.pleased")[1], "rate 0") {
		t.Fatalf("an unproven label was rendered as a rate:\n%s", out)
	}
}

func TestAnnotationsCommandPassesTheWindowAndPrintsTheMeasureForJSON(t *testing.T) {
	out, sent := runAnnotations(t, annotationGateFixture, "--window", "last_30d", "--json")
	if sent["window"] != "last_30d" {
		t.Fatalf("window = %v", sent)
	}
	var decoded struct {
		Fields []map[string]string `json:"fields"`
	}
	if err := json.Unmarshal([]byte(out), &decoded); err != nil || len(decoded.Fields) != len(annotationGateFixture) {
		t.Fatalf("--json did not print the measure rows (%v): %s", err, out)
	}
}

func TestAnnotationsCommandEmptyWindowIsUnknownNotZeroFriction(t *testing.T) {
	out, _ := runAnnotations(t, []map[string]string{{"kind": "coverage", "pack": "default", "pack_version": "1", "eligible_runs": "0", "observed_runs": "0", "coverage": "0"}})
	if !strings.Contains(out, "unknown, not zero friction") {
		t.Fatalf("empty coverage was not stated honestly:\n%s", out)
	}
}

func TestInferredAnnotationLinesQuoteARateOnlyForAProvenGate(t *testing.T) {
	rate := 0.174
	section := &measurepb.UsageInferredAnnotations{
		PackName: "default", PackVersion: "1", EligibleRuns: 50, ObservedRuns: 34, Coverage: 0.68,
		Labels: []*measurepb.UsageInferredAnnotation{
			{Id: "work.blocked_by_tool", Group: "agent", State: "observed", Observed: 34, Eligible: 50, Coverage: 0.68, GateState: "proven", Gold: 40, Rate: &rate, Friction: true, TargetPrecision: 0.8, TargetRecall: 0.7, ProvenObserved: 30, GateModel: "winnow:e4b", GateRole: "decide.local.accurate", Calibration: "calibrated", Temperature: 1.4, Ece: 0.042},
			{Id: "operator.dissatisfied", Group: "operator", State: "observed", Observed: 34, Eligible: 50, Coverage: 0.68, GateState: "insufficient_gold", Gold: 2, Friction: true, TargetPrecision: 0.85, TargetRecall: 0.7},
			{Id: "run.type", Group: "run", State: "observed", Observed: 34, Eligible: 50, GateState: "unmeasured", Friction: false},
		},
	}
	joined := strings.Join(inferredAnnotationLines(section), "\n")
	for _, want := range []string{"coverage 68% (34 of 50 runs observed)", "[proven] work.blocked_by_tool", "17.4% of 30 proven observations (of 34/50 observed)", "winnow:e4b@decide.local.accurate, calibrated T=1.4, ECE 0.042", "no rate (unproven): operator.dissatisfied (insufficient_gold)", "usage annotations"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("section missing %q:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "run.type") {
		t.Fatalf("a non-friction label was rendered in the friction lens:\n%s", joined)
	}
}

func TestInferredAnnotationLinesReadAnEmptyWindowAsUnknown(t *testing.T) {
	lines := inferredAnnotationLines(&measurepb.UsageInferredAnnotations{Note: "no annotations recorded in this window"})
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"coverage 0% (0 of 0 runs observed)", "no annotations recorded in this window", "unknown, not zero friction"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("empty section missing %q:\n%s", want, joined)
		}
	}
}
