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

func TestParsePhaseResultDetailedStatuses(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   PhaseResultParseStatus
		err    bool
	}{
		{name: "empty output", output: "  ", want: PhaseResultParseNoOutput},
		{name: "plain prose", output: "done but not structured", want: PhaseResultParseNoStructuredResult},
		{name: "empty envelope", output: `{"operating_mode_result":{}}`, want: PhaseResultParseEmpty},
		{name: "malformed envelope", output: `{"operating_mode_result":{"progress":{"decision":"sideways"}}}`, want: PhaseResultParseMalformed, err: true},
		{name: "valid envelope", output: `{"operating_mode_result":{"verdict":"accepted"}}`, want: PhaseResultParseValid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParsePhaseResultDetailed(tt.output)
			if tt.err && err == nil {
				t.Fatal("ParsePhaseResultDetailed error = nil, want error")
			}
			if !tt.err && err != nil {
				t.Fatalf("ParsePhaseResultDetailed returned error: %v", err)
			}
			if parsed.Status != tt.want {
				t.Fatalf("status = %q, want %q", parsed.Status, tt.want)
			}
		})
	}
}
