package main

import "testing"

func TestCapacityRungLabelsStayAlignedWithResourceContract(t *testing.T) {
	steps := capacitySteps()
	if len(steps) != 2 || steps[0].Label != "full" || steps[1].Label != "offload-dit" {
		t.Fatalf("capacity steps = %+v", steps)
	}
}
