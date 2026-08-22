package collectors

import (
	"testing"
	"time"
)

func TestForkRateTrackerNeedsTwoSamples(t *testing.T) {
	var tracker forkRateTracker
	base := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)

	if _, ok := tracker.observe(1000, base); ok {
		t.Fatal("first sample reported a rate; there is no interval to divide by yet")
	}
	rate, ok := tracker.observe(1600, base.Add(2*time.Second))
	if !ok {
		t.Fatal("second sample did not produce a rate")
	}
	if rate != 300 {
		t.Fatalf("rate=%v, want 300", rate)
	}
}

// A counter that goes backwards means the host rebooted. Reporting the wrapped
// delta would invent an enormous spike out of a reboot.
func TestForkRateTrackerRejectsCounterReset(t *testing.T) {
	var tracker forkRateTracker
	base := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	tracker.observe(9_000_000, base)
	if _, ok := tracker.observe(12, base.Add(time.Second)); ok {
		t.Fatal("counter reset reported a rate")
	}
	// The reset sample becomes the new baseline rather than being discarded.
	rate, ok := tracker.observe(112, base.Add(2*time.Second))
	if !ok || rate != 100 {
		t.Fatalf("rate=%v ok=%v, want 100 and true after re-baselining", rate, ok)
	}
}

func TestForkRateTrackerIgnoresZeroInterval(t *testing.T) {
	var tracker forkRateTracker
	base := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	tracker.observe(10, base)
	if _, ok := tracker.observe(20, base); ok {
		t.Fatal("zero elapsed time produced a rate instead of being skipped")
	}
}

// The first cycle must be explicitly pending, never a silent zero: "0 forks/sec"
// and "no rate yet" are opposite diagnoses during an incident.
func TestForkRateValuesMarksFirstCyclePending(t *testing.T) {
	var tracker forkRateTracker
	base := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	reading := forkRateReading{total: 500, supported: true, provenance: "test"}

	first := forkRateValues(&tracker, reading, base)
	if first["fork_rate_pending"] != true || first["fork_rate_primed"] != false {
		t.Fatalf("first cycle not marked pending: %v", first)
	}
	if first["fork_rate_status"] != "measured" {
		t.Fatalf("status=%v, want measured", first["fork_rate_status"])
	}

	second := forkRateValues(&tracker, forkRateReading{total: 1500, supported: true, provenance: "test"}, base.Add(time.Second))
	if second["fork_rate_pending"] != false || second["fork_rate_primed"] != true {
		t.Fatalf("second cycle still pending: %v", second)
	}
	if second["forks_per_second"] != float64(1000) {
		t.Fatalf("forks_per_second=%v, want 1000", second["forks_per_second"])
	}
}

func TestForkRateValuesReportsUnsupported(t *testing.T) {
	var tracker forkRateTracker
	values := forkRateValues(&tracker, forkRateUnsupported("no counter here"), time.Now())
	if values["fork_rate_status"] != "unsupported" {
		t.Fatalf("status=%v, want unsupported", values["fork_rate_status"])
	}
	if _, present := values["forks_per_second"]; present {
		t.Fatal("unsupported reading emitted a rate value; absence must not read as zero")
	}
}
