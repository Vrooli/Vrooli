package operatingmode

import "testing"

// defaultDeclared is the synthesised declared-output used by extraction unit
// tests: the canonical envelope key with no per-field requirements.
func defaultDeclared() *DeclaredOutput {
	return &DeclaredOutput{EnvelopeKey: resultEnvelopeKey, RequiresStructuredResult: true, Resolution: defaultResolutionPolicy()}
}

func TestExtractDeclaredFromFencedEnvelope(t *testing.T) {
	output := "summary\n```json\n{\"operating_mode_result\":{\"verdict\":\"accepted\",\"replan_needed\":true,\"handoff\":{\"summary\":\"done\"},\"artifacts\":[{\"path\":\"modes/holistic-loop/findings.md\",\"content\":\"# Findings\"}]}}\n```"
	att := extractDeclared(output, defaultDeclared())
	if !att.ok {
		t.Fatal("extractDeclared ok = false, want true for fenced envelope")
	}
	if att.result.Verdict != "accepted" || !att.result.ReplanNeeded {
		t.Fatalf("verdict/replan = %q/%v", att.result.Verdict, att.result.ReplanNeeded)
	}
	if att.result.Handoff == nil || att.result.Handoff.Summary != "done" {
		t.Fatalf("handoff = %+v", att.result.Handoff)
	}
	if len(att.result.Artifacts) != 1 || att.result.Artifacts[0].Path != "modes/holistic-loop/findings.md" {
		t.Fatalf("artifacts = %+v", att.result.Artifacts)
	}
}

func TestExtractDeclaredRejectsInvalidProgressDecision(t *testing.T) {
	att := extractDeclared(`{"operating_mode_result":{"progress":{"decision":"sideways"}}}`, defaultDeclared())
	if att.ok {
		t.Fatal("extractDeclared ok = true, want false for invalid progress decision")
	}
}

func TestExtractDeclaredIgnoresNonEnvelopeProse(t *testing.T) {
	att := extractDeclared("done but not structured", defaultDeclared())
	if att.ok {
		t.Fatal("extractDeclared ok = true, want false for plain prose")
	}
}

func TestExtractDeclaredHonorsEnvelopeKey(t *testing.T) {
	declared := &DeclaredOutput{EnvelopeKey: "custom_result", RequiresStructuredResult: true, Resolution: defaultResolutionPolicy()}
	att := extractDeclared(`{"custom_result":{"verdict":"accepted"}}`, declared)
	if !att.ok || att.result.Verdict != "accepted" {
		t.Fatalf("extractDeclared with custom key = %+v", att)
	}
	// The default envelope key must not match a custom-keyed envelope.
	if att := extractDeclared(`{"custom_result":{"verdict":"accepted"}}`, defaultDeclared()); att.ok {
		t.Fatal("extractDeclared matched custom envelope under default key")
	}
}
