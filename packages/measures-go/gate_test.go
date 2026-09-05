package measures

import "testing"

func TestGate(t *testing.T) {
	read := MeasureDeclaration{Name: "m", Effect: EffectRead, RunEligible: true}
	write := MeasureDeclaration{Name: "m", Effect: EffectWrite, RunEligible: true}
	destructive := MeasureDeclaration{Name: "m", Effect: EffectDestructive, RunEligible: true}
	notEligible := MeasureDeclaration{Name: "m", Effect: EffectRead, RunEligible: false}

	complete := ResolveResult{Params: map[string]string{"a": "x"}, Confidence: 1.0}
	lowConf := ResolveResult{Params: map[string]string{"a": "x"}, Confidence: 0.5}
	missing := ResolveResult{Needs: []string{"a"}, Confidence: 1.0}

	cases := []struct {
		name string
		decl MeasureDeclaration
		res  ResolveResult
		want GateAction
	}{
		{"read+complete+high → execute", read, complete, GateExecute},
		{"read+lowconf → confirm", read, lowConf, GateConfirm},
		{"read+missing → needs", read, missing, GateNeedsParams},
		{"write+complete → confirm (never auto)", write, complete, GateConfirm},
		{"destructive+complete → confirm", destructive, complete, GateConfirm},
		{"read+not-eligible → confirm", notEligible, complete, GateConfirm},
		{"write+missing → needs (needs precedes effect)", write, missing, GateNeedsParams},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Gate(c.decl, c.res, DefaultConfidenceThreshold)
			if got.Action != c.want {
				t.Fatalf("Gate action = %d, want %d (reason: %s)", got.Action, c.want, got.Reason)
			}
			if got.Reason == "" {
				t.Fatal("gate decision must carry a reason")
			}
			if (got.Action == GateExecute) != got.Execute() {
				t.Fatal("Execute() inconsistent with Action")
			}
		})
	}
}

func TestGate_ThresholdBoundary(t *testing.T) {
	read := MeasureDeclaration{Name: "m", Effect: EffectRead, RunEligible: true}
	atThreshold := ResolveResult{Params: map[string]string{"a": "x"}, Confidence: DefaultConfidenceThreshold}
	if got := Gate(read, atThreshold, DefaultConfidenceThreshold); got.Action != GateExecute {
		t.Fatalf("confidence == θ should execute (>=), got action %d", got.Action)
	}
	justBelow := ResolveResult{Params: map[string]string{"a": "x"}, Confidence: DefaultConfidenceThreshold - 0.001}
	if got := Gate(read, justBelow, DefaultConfidenceThreshold); got.Action != GateConfirm {
		t.Fatalf("confidence just below θ should confirm, got action %d", got.Action)
	}
}
