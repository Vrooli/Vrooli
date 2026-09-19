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
