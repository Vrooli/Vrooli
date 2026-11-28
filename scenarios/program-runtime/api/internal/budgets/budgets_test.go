package budgets

import (
	"testing"
	"time"
)

// TestLadderIsStrictlyNested is the regression guard for the defect that made
// every synchronous call fail at 30 seconds: layers whose budgets were picked
// independently and did not nest.
func TestLadderIsStrictlyNested(t *testing.T) {
	if err := Validate(); err != nil {
		t.Fatalf("shipped ladder must validate: %v", err)
	}
	ladder := Ladder()
	if len(ladder) < 2 {
		t.Fatalf("ladder must have at least two rungs, got %d", len(ladder))
	}
	for index := 1; index < len(ladder); index++ {
		if ladder[index-1].Budget >= ladder[index].Budget {
			t.Fatalf("rung %q (%s) is not shorter than %q (%s)",
				ladder[index-1].Name, ladder[index-1].Budget,
				ladder[index].Name, ladder[index].Budget)
		}
	}
}

// TestServerWriteExceedsEveryInnerBudget states the specific invariant whose
// violation produced `unexpected EOF`: the HTTP write deadline must outlast
// anything a handler can legitimately wait for.
func TestServerWriteExceedsEveryInnerBudget(t *testing.T) {
	for _, rung := range Ladder() {
		if rung.Name == "server_write" {
			continue
		}
		if rung.Budget >= ServerWrite {
			t.Fatalf("%s (%s) is not shorter than the server write deadline (%s); "+
				"a handler that waits this long is killed mid-write and reports an untyped error",
				rung.Name, rung.Budget, ServerWrite)
		}
	}
}

// TestBridgeCallIsShorterThanKernelInvoke states why the kernel sees typed
// bridge errors instead of its own client timeout.
func TestBridgeCallIsShorterThanKernelInvoke(t *testing.T) {
	if BridgeCall >= KernelInvoke {
		t.Fatalf("bridge call budget (%s) must be shorter than the kernel's wait on it (%s)",
			BridgeCall, KernelInvoke)
	}
}

func TestKernelEnvelopeMirrorsTheLadder(t *testing.T) {
	envelope := Kernel()
	if envelope.Telemetry != KernelTelemetry.Seconds() {
		t.Fatalf("telemetry seconds drifted: %v vs %v", envelope.Telemetry, KernelTelemetry.Seconds())
	}
	if envelope.Describe != KernelDescribe.Seconds() {
		t.Fatalf("describe seconds drifted: %v vs %v", envelope.Describe, KernelDescribe.Seconds())
	}
	if envelope.Invoke != KernelInvoke.Seconds() {
		t.Fatalf("invoke seconds drifted: %v vs %v", envelope.Invoke, KernelInvoke.Seconds())
	}
	if got := envelope.BindingInvokes["test-genie/validation/wait"]; got != OwnerWaitKernel.Seconds() {
		t.Fatalf("owner wait profile = %v, want %v", got, OwnerWaitKernel.Seconds())
	}
	if got := BridgeForBinding("test-genie/validation/wait"); got != OwnerWaitBridge {
		t.Fatalf("owner wait bridge = %s, want %s", got, OwnerWaitBridge)
	}
	for _, id := range []string{"test-genie/runs/get", "unknown/binding"} {
		if got := BridgeForBinding(id); got != BridgeCall {
			t.Fatalf("ordinary bridge profile for %q = %s, want %s", id, got, BridgeCall)
		}
	}
}

func TestOwnerWaitProfileFitsExistingServerCeilings(t *testing.T) {
	if BridgeCall != 90*time.Second || KernelInvoke != 100*time.Second {
		t.Fatalf("ordinary profile changed: bridge=%s kernel=%s", BridgeCall, KernelInvoke)
	}
	if OwnerWaitBridge != 12*time.Minute || OwnerWaitKernel != 13*time.Minute {
		t.Fatalf("owner profile = %s/%s, want 12m/13m", OwnerWaitBridge, OwnerWaitKernel)
	}
	if OwnerWaitBridge >= OwnerWaitKernel || OwnerWaitKernel >= ServerWrite {
		t.Fatalf("owner wait ladder is not nested under server write: %s < %s < %s", OwnerWaitBridge, OwnerWaitKernel, ServerWrite)
	}
}

func TestBoundWait(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		requested time.Duration
		want      time.Duration
	}{
		{"unset falls back to the ceiling", 0, MaxWait},
		{"negative falls back to the ceiling", -time.Second, MaxWait},
		{"in-range is honoured", 30 * time.Second, 30 * time.Second},
		{"over-range is clamped, not refused", time.Hour, MaxWait},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := BoundWait(testCase.requested); got != testCase.want {
				t.Fatalf("BoundWait(%s) = %s, want %s", testCase.requested, got, testCase.want)
			}
		})
	}
}

func TestQualifyLatencyEvidenceRequiresCompleteStrataAndPhases(t *testing.T) {
	policy := DefaultLatencyEvidencePolicy()
	if policy.MinSamplesPerStratum != 20 {
		t.Fatalf("default minimum = %d, want 20", policy.MinSamplesPerStratum)
	}
	valid := make([]LatencyObservation, 0, 80)
	for _, status := range []string{"success", "failure"} {
		for _, thermal := range []string{"cold", "warm"} {
			for n := 1; n <= 20; n++ {
				valid = append(valid, LatencyObservation{Status: status, ThermalState: thermal, Total: time.Duration(n) * time.Second, BindingPhase: time.Second, ModelPhase: 2 * time.Second})
			}
		}
	}
	got, err := QualifyLatencyEvidence(valid, policy)
	if err != nil {
		t.Fatalf("complete evidence rejected: %v", err)
	}
	for key, count := range got.Counts {
		if count != 20 || got.P95[key] != 19*time.Second {
			t.Fatalf("stratum %s count/p95 = %d/%s, want 20/19s", key, count, got.P95[key])
		}
	}
	if got.PooledP95 != 19*time.Second {
		t.Fatalf("pooled p95 = %s, want 19s", got.PooledP95)
	}
	for _, tc := range []struct {
		name   string
		mutate func([]LatencyObservation) []LatencyObservation
	}{
		{"missing stratum", func(samples []LatencyObservation) []LatencyObservation { return samples[:60] }},
		{"unknown status", func(samples []LatencyObservation) []LatencyObservation { samples[0].Status = "partial"; return samples }},
		{"missing phase", func(samples []LatencyObservation) []LatencyObservation { samples[0].BindingPhase = 0; return samples }},
		{"unknown thermal state", func(samples []LatencyObservation) []LatencyObservation {
			samples[0].ThermalState = "unknown"
			return samples
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			copySamples := append([]LatencyObservation(nil), valid...)
			if _, err := QualifyLatencyEvidence(tc.mutate(copySamples), policy); err == nil {
				t.Fatal("incomplete evidence unexpectedly qualified")
			}
		})
	}
}
