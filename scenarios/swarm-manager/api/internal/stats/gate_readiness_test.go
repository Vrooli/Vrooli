package stats

import "testing"

func TestEvaluateGateReadinessRequiresAttributedSample(t *testing.T) {
	if got := EvaluateGateReadiness(1, 2, 3, 0.9); got != GateReadinessInsufficientSample {
		t.Fatalf("readiness = %q, want insufficient-sample", got)
	}
	if got := EvaluateGateReadiness(0.89, 3, 3, 0.9); got != GateReadinessBelowThreshold {
		t.Fatalf("readiness = %q, want below-threshold", got)
	}
	if got := EvaluateGateReadiness(0.9, 3, 3, 0.9); got != GateReadinessReady {
		t.Fatalf("readiness = %q, want ready", got)
	}
}
