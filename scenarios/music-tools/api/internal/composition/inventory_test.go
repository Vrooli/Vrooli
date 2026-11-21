package composition

import "testing"

func TestInventoryPolicyThreshold(t *testing.T) {
	target, threshold, available := 10, 4, 3
	if available >= threshold || target <= 0 {
		return
	}
	if target <= available {
		t.Fatal("policy must replenish toward a positive target")
	}
}

func TestInventoryTargetRejectsBeyondSupportedBatchBeforeMutation(t *testing.T) {
	pool := NewPool()
	if err := pool.Configure("s", 100, 99); err != nil {
		t.Fatal("largest supported target refused", err)
	}
	if err := pool.Configure("s", 101, 2); err != ErrInvalidBatch {
		t.Fatalf("unsupported target accepted: %v", err)
	}
	_, _, target, threshold := pool.Status("s")
	if target != 100 || threshold != 99 {
		t.Fatalf("invalid target mutated policy: %d %d", target, threshold)
	}
}
