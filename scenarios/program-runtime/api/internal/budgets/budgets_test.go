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
