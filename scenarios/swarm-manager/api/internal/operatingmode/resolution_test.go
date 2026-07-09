package operatingmode

import (
	"context"
	"strings"
	"testing"
)

// reviewPhase is a declared-output phase used across the ladder tests: it
// requires a structured envelope carrying a required string `verdict`, mirroring
// the holistic-loop / phased-plan-drain review phase.
func reviewPhase() PhaseDefinition {
	return PhaseDefinition{
		Phase: "review",
		Kind:  PhaseKindReview,
		OutputContract: PhaseOutputContract{
			RequiresStructuredResult: true,
			RequiresVerdict:          true,
		},
		DeclaredOutput: &DeclaredOutput{
			EnvelopeKey:              resultEnvelopeKey,
			RequiresStructuredResult: true,
			Fields: []OutputField{
				{Name: "verdict", Type: "string", Required: true},
			},
			Resolution: defaultResolutionPolicy(),
		},
	}
}

// classifyProgressPhase mirrors phased-plan-drain's classify_progress: an
// enum-constrained progress.decision the guards route on.
func classifyProgressPhase() PhaseDefinition {
	return PhaseDefinition{
		Phase: "classify_progress",
		Kind:  PhaseKindExecute,
		OutputContract: PhaseOutputContract{
			RequiresStructuredResult: true,
			RequiresProgress:         true,
		},
		DeclaredOutput: &DeclaredOutput{
			EnvelopeKey:              resultEnvelopeKey,
			RequiresStructuredResult: true,
			Fields: []OutputField{
				{Name: "progress", Type: "object", Required: true},
				{Name: "progress.decision", Type: "string", Required: true, Enum: []any{"continue", "blocked", "replan", "complete"}},
			},
			Resolution: defaultResolutionPolicy(),
		},
	}
}

const acceptedEnvelope = `{"operating_mode_result":{"verdict":"accepted"}}`

func TestResolveCleanResult(t *testing.T) {
	res := resolvePhaseOutput(context.Background(), reviewPhase(), []string{acceptedEnvelope}, nil)
	if res.Outcome != ResolutionResolved {
		t.Fatalf("outcome = %q, want resolved", res.Outcome)
	}
	if res.Layer != ResolutionLayerExtract {
		t.Fatalf("layer = %q, want L1", res.Layer)
	}
	if res.Result.Verdict != "accepted" {
		t.Fatalf("verdict = %q", res.Result.Verdict)
	}
}

func TestResolveMalformedThenValidWithinOneMessage(t *testing.T) {
	// The message wraps prose and a fenced envelope: the whole-body parse fails
	// but the fenced ```json block extracts cleanly (L1).
	msg := "Here is my review.\n```json\n" + acceptedEnvelope + "\n```\nThanks!"
	res := resolvePhaseOutput(context.Background(), reviewPhase(), []string{msg}, nil)
	if res.Outcome != ResolutionResolved || res.Result.Verdict != "accepted" {
		t.Fatalf("res = %+v, want resolved accepted", res)
	}
}

func TestResolveSubagentTailRecoversTrueFinalMessage(t *testing.T) {
	// The real answer is the second-to-last message; a subagent appended a
	// trailing status message with no envelope. L0 must scan back and recover it.
	messages := []string{
		"Investigating the acceptance criteria…",
		acceptedEnvelope,
		"[subagent] cleanup complete, 3 files touched",
	}
	res := resolvePhaseOutput(context.Background(), reviewPhase(), messages, nil)
	if res.Outcome != ResolutionRecovered {
		t.Fatalf("outcome = %q, want recovered", res.Outcome)
	}
	if res.Layer != ResolutionLayerFinalMsg {
		t.Fatalf("layer = %q, want L0 true-final-message", res.Layer)
	}
	if res.ChosenMessageIndex != 1 {
		t.Fatalf("chosen index = %d, want 1", res.ChosenMessageIndex)
	}
	if res.Result.Verdict != "accepted" {
		t.Fatalf("verdict = %q", res.Result.Verdict)
	}
}

func TestResolveMissingEnvelopeAbstainsWithoutClassifier(t *testing.T) {
	res := resolvePhaseOutput(context.Background(), reviewPhase(), []string{"I accept this work, it looks good."}, nil)
	if res.Outcome != ResolutionAbstained {
		t.Fatalf("outcome = %q, want abstained", res.Outcome)
	}
	if len(res.Missing) == 0 {
		t.Fatalf("missing = %v, want the unresolved verdict field", res.Missing)
	}
}

func TestResolveNoOutputAbstains(t *testing.T) {
	res := resolvePhaseOutput(context.Background(), reviewPhase(), []string{"   "}, nil)
	if res.Outcome != ResolutionAbstained {
		t.Fatalf("outcome = %q, want abstained", res.Outcome)
	}
}

// stubClassifier is a deterministic FieldClassifier for the L2 tests: it answers
// declared field paths from a fixed map and abstains on anything else, with no
// live model.
type stubClassifier struct {
	answers map[string]string
	calls   []string
}

func (s *stubClassifier) ClassifyField(_ context.Context, req ClassifyFieldRequest) (ClassifyFieldResult, error) {
	s.calls = append(s.calls, req.FieldPath)
	if v, ok := s.answers[req.FieldPath]; ok {
		return ClassifyFieldResult{Found: true, Value: v}, nil
	}
	return ClassifyFieldResult{Found: false}, nil
}

func TestResolveClassifierReconstructsVerdict(t *testing.T) {
	classifier := &stubClassifier{answers: map[string]string{"verdict": "accepted"}}
	res := resolvePhaseOutput(context.Background(), reviewPhase(), []string{"After review I accept the work."}, classifier)
	if res.Outcome != ResolutionRecovered {
		t.Fatalf("outcome = %q, want recovered via classifier", res.Outcome)
	}
	if res.Layer != ResolutionLayerClassifier {
		t.Fatalf("layer = %q, want L2 classifier", res.Layer)
	}
	if res.Result.Verdict != "accepted" {
		t.Fatalf("verdict = %q, want accepted", res.Result.Verdict)
	}
	if len(classifier.calls) == 0 {
		t.Fatal("classifier was never consulted")
	}
}

func TestResolveClassifierReconstructsEnumDecision(t *testing.T) {
	classifier := &stubClassifier{answers: map[string]string{"progress.decision": "replan"}}
	res := resolvePhaseOutput(context.Background(), classifyProgressPhase(), []string{"The plan is wrong, we need to replan."}, classifier)
	if res.Outcome != ResolutionRecovered {
		t.Fatalf("outcome = %q, want recovered", res.Outcome)
	}
	if res.Result.Progress == nil || res.Result.Progress.Decision != ProgressReplan {
		t.Fatalf("progress = %+v, want decision=replan", res.Result.Progress)
	}
}

func TestResolveClassifierPreservesNonRoutingDeclaredScalar(t *testing.T) {
	phase := PhaseDefinition{
		Phase: "assess",
		Kind:  PhaseKindReview,
		DeclaredOutput: &DeclaredOutput{
			EnvelopeKey:              resultEnvelopeKey,
			RequiresStructuredResult: true,
			Fields: []OutputField{
				{Name: "confidence", Type: "number", Required: true},
			},
			Resolution: defaultResolutionPolicy(),
		},
	}
	classifier := &stubClassifier{answers: map[string]string{"confidence": "0.75"}}
	res := resolvePhaseOutput(context.Background(), phase, []string{"This looks mostly correct."}, classifier)
	if res.Outcome != ResolutionRecovered {
		t.Fatalf("outcome = %q, want recovered", res.Outcome)
	}
	envelope := resultEnvelopeMap(res.Result)
	got, ok := envelope["confidence"].(float64)
	if !ok || got != 0.75 {
		t.Fatalf("confidence = %#v (ok=%v), want 0.75", envelope["confidence"], ok)
	}
}

func TestResolveClassifierAbstainsRatherThanGuessing(t *testing.T) {
	// The classifier finds nothing → the ladder abstains on the required field
	// instead of fabricating a verdict.
	classifier := &stubClassifier{answers: map[string]string{}}
	res := resolvePhaseOutput(context.Background(), reviewPhase(), []string{"Some prose without a decision."}, classifier)
	if res.Outcome != ResolutionAbstained {
		t.Fatalf("outcome = %q, want abstained", res.Outcome)
	}
	if res.Result.Verdict != "" {
		t.Fatalf("verdict = %q, want empty (no guess)", res.Result.Verdict)
	}
}

func TestResolveL3RejectsEnumViolation(t *testing.T) {
	// A present-but-invalid enum value is a contract violation: extraction parses
	// it but L3 validation rejects it, and with no classifier the ladder abstains.
	msg := `{"operating_mode_result":{"progress":{"decision":"sideways"}}}`
	res := resolvePhaseOutput(context.Background(), classifyProgressPhase(), []string{msg}, nil)
	if res.Outcome != ResolutionAbstained {
		t.Fatalf("outcome = %q, want abstained on invalid enum", res.Outcome)
	}
}

func TestResolveL3RejectsBadEnumEvenWhenParsed(t *testing.T) {
	// A structurally-valid decision value outside the declared enum: the progress
	// state itself validates (decision is one of the four), but if a mode declared
	// a narrower enum the L3 check must still reject out-of-enum values.
	narrow := classifyProgressPhase()
	narrow.DeclaredOutput.Fields[1].Enum = []any{"continue", "complete"}
	msg := `{"operating_mode_result":{"progress":{"decision":"replan"}}}`
	res := resolvePhaseOutput(context.Background(), narrow, []string{msg}, nil)
	if res.Outcome != ResolutionAbstained {
		t.Fatalf("outcome = %q, want abstained on out-of-declared-enum", res.Outcome)
	}
	if len(res.Violations) == 0 {
		t.Fatalf("violations = %v, want the enum violation reported", res.Violations)
	}
}

func TestResolveOptOutOfTrueFinalMessageOnlyScansLast(t *testing.T) {
	phase := reviewPhase()
	phase.DeclaredOutput.Resolution.DetectTrueFinalMessage = false
	messages := []string{acceptedEnvelope, "[subagent] trailing noise"}
	res := resolvePhaseOutput(context.Background(), phase, messages, nil)
	// With true-final-message detection off, only the last (noise) message is
	// examined, so the earlier clean envelope is not recovered → abstain.
	if res.Outcome != ResolutionAbstained {
		t.Fatalf("outcome = %q, want abstained when L0 disabled", res.Outcome)
	}
}

func TestResolveNotRequiredPhaseNeverAbstains(t *testing.T) {
	phase := PhaseDefinition{
		Phase:          "investigate",
		Kind:           PhaseKindInvestigate,
		OutputContract: PhaseOutputContract{RequiresStructuredResult: false},
		DeclaredOutput: &DeclaredOutput{
			EnvelopeKey:              resultEnvelopeKey,
			RequiresStructuredResult: false,
			Resolution:               defaultResolutionPolicy(),
		},
	}
	res := resolvePhaseOutput(context.Background(), phase, []string{"free-form investigation notes"}, nil)
	if res.Outcome != ResolutionNotRequired {
		t.Fatalf("outcome = %q, want not_required", res.Outcome)
	}
	if !res.Resolved() {
		t.Fatal("not-required phase should count as resolved")
	}
}

func TestResolveEmptyEnvelopeAbstainsWhenStructuredRequired(t *testing.T) {
	// A phase requiring a structured result but declaring no specific fields
	// (investigate/plan style) still needs a non-empty body.
	phase := PhaseDefinition{
		Phase:          "investigate",
		Kind:           PhaseKindInvestigate,
		OutputContract: PhaseOutputContract{RequiresStructuredResult: true},
	}
	res := resolvePhaseOutput(context.Background(), phase, []string{`{"operating_mode_result":{}}`}, nil)
	if res.Outcome != ResolutionAbstained {
		t.Fatalf("outcome = %q, want abstained on empty envelope", res.Outcome)
	}
	// A non-empty body (a handoff) satisfies the same phase.
	ok := resolvePhaseOutput(context.Background(), phase, []string{`{"operating_mode_result":{"handoff":{"summary":"done"}}}`}, nil)
	if !ok.Resolved() {
		t.Fatalf("outcome = %q, want resolved for non-empty envelope", ok.Outcome)
	}
}

func TestValidateDeclaredOutputBounds(t *testing.T) {
	min := 1.0
	declared := &DeclaredOutput{
		Fields: []OutputField{{Name: "score", Type: "number", Required: true, Minimum: &min}},
	}
	if missing, violations := validateDeclaredOutput(declared, map[string]any{"score": 5.0}); len(missing) != 0 || len(violations) != 0 {
		t.Fatalf("valid score reported missing=%v violations=%v", missing, violations)
	}
	if _, violations := validateDeclaredOutput(declared, map[string]any{"score": 0.0}); len(violations) == 0 {
		t.Fatal("score below minimum did not report a violation")
	}
	if missing, _ := validateDeclaredOutput(declared, map[string]any{}); len(missing) == 0 {
		t.Fatal("absent required score not reported missing")
	}
}

func TestValidateDeclaredOutputRejectsMissingHandoffSubfield(t *testing.T) {
	declared := &DeclaredOutput{
		Fields: []OutputField{
			{
				Name:     "handoff",
				Type:     "object",
				Required: true,
				Fields: []OutputField{
					{Name: "summary", Type: "string", Required: true},
					{Name: "frontier", Type: "string", Required: true},
				},
			},
		},
	}
	missing, violations := validateDeclaredOutput(declared, map[string]any{
		"handoff": map[string]any{"summary": "done"},
	})
	if len(violations) != 0 {
		t.Fatalf("violations = %v, want none", violations)
	}
	if got := strings.Join(missing, ","); !strings.Contains(got, "handoff.frontier") {
		t.Fatalf("missing = %v, want handoff.frontier", missing)
	}
}

func TestValidateDeclaredOutputRejectsOutOfEnumVerdict(t *testing.T) {
	declared := &DeclaredOutput{
		Fields: []OutputField{
			{Name: "verdict", Type: "string", Required: true, Enum: []any{"accepted", "changes_requested", "rejected"}},
		},
	}
	_, violations := validateDeclaredOutput(declared, map[string]any{"verdict": "needs_work"})
	if len(violations) == 0 {
		t.Fatal("out-of-enum verdict did not report a violation")
	}
}

func TestResolveClassifierErrorDoesNotPanic(t *testing.T) {
	classifier := errorClassifier{}
	res := resolvePhaseOutput(context.Background(), reviewPhase(), []string{"prose"}, classifier)
	if res.Outcome != ResolutionAbstained {
		t.Fatalf("outcome = %q, want abstained when classifier errors", res.Outcome)
	}
	if !strings.Contains(strings.Join(res.Notes, " "), "classify") {
		t.Fatalf("notes = %v, want a classify error note", res.Notes)
	}
}

type errorClassifier struct{}

func (errorClassifier) ClassifyField(context.Context, ClassifyFieldRequest) (ClassifyFieldResult, error) {
	return ClassifyFieldResult{}, context.DeadlineExceeded
}
