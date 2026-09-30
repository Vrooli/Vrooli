package collectors

import (
	"testing"
	"time"
)

func TestDeltaCPUValuesFirstAndResetAreNotMeasurements(t *testing.T) {
	state := platformCPUState{}
	names := []string{"user", "nice", "system", "idle", "iowait", "irq", "softirq", "steal"}
	when := time.Unix(100, 0)
	if _, _, measured := deltaCPUValues(&state, []uint64{10, 0, 10, 80, 0, 0, 0, 0}, when, names, 3, 4); measured {
		t.Fatal("first sample must be pending")
	}
	if usage, modes, measured := deltaCPUValues(&state, []uint64{20, 0, 20, 100, 0, 0, 0, 0}, when.Add(time.Second), names, 3, 4); !measured || usage != 50 || modes["system"] != 25 {
		t.Fatalf("second sample = usage %v modes %#v measured %v", usage, modes, measured)
	}
	if _, _, measured := deltaCPUValues(&state, []uint64{1, 0, 1, 1, 0, 0, 0, 0}, when.Add(2*time.Second), names, 3, 4); measured {
		t.Fatal("counter reset must be pending")
	}
}

func TestCPURefusalCarriesNoNumericValue(t *testing.T) {
	reading := platformCPUUnsupported("cgroup v2 cpu bandwidth backend unavailable")
	values := map[string]interface{}{}
	ensureCPUObservationStates(values, reading)
	if values["quota_throttling_status"] != "unsupported" {
		t.Fatalf("quota status = %v", values["quota_throttling_status"])
	}
	if _, numeric := values["quota_throttling"].(float64); numeric {
		t.Fatal("refused quota observation must not carry a number")
	}
}
