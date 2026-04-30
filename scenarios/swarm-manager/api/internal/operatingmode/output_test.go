package operatingmode

import "testing"

func TestParsePhaseResultFromFencedEnvelope(t *testing.T) {
	output := "summary\n```json\n{\"operating_mode_result\":{\"verdict\":\"accepted\",\"replan_needed\":true,\"handoff\":{\"summary\":\"done\"},\"artifacts\":[{\"path\":\"modes/holistic-loop/findings.md\",\"content\":\"# Findings\"}]}}\n```"
	result, ok, err := ParsePhaseResult(output)
	if err != nil {
		t.Fatalf("ParsePhaseResult returned error: %v", err)
	}
	if !ok {
		t.Fatal("ParsePhaseResult ok = false, want true")
	}
	if result.Verdict != "accepted" || !result.ReplanNeeded {
		t.Fatalf("verdict/replan = %q/%v", result.Verdict, result.ReplanNeeded)
	}
	if result.Handoff == nil || result.Handoff.Summary != "done" {
		t.Fatalf("handoff = %+v", result.Handoff)
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0].Path != "modes/holistic-loop/findings.md" {
		t.Fatalf("artifacts = %+v", result.Artifacts)
	}
}

func TestParsePhaseResultValidatesProgressDecision(t *testing.T) {
	_, ok, err := ParsePhaseResult(`{"operating_mode_result":{"progress":{"decision":"sideways"}}}`)
	if err == nil {
		t.Fatal("ParsePhaseResult error = nil, want invalid progress error")
	}
	if ok {
		t.Fatal("ParsePhaseResult ok = true, want false on invalid envelope")
	}
}
