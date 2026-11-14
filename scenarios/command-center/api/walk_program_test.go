package main

import (
	"os/exec"
	"testing"
	"time"
)

func TestWalkReadingPreservesEvidenceAndExcludesSamples(t *testing.T) { // [REQ:CC-P0-015]
	now := time.Now().UTC()
	got := walkReading(MetricEntry{ID: "zero", Coverage: CoverageNow, Trust: TrustValid, Empirical: EmpiricalMiss, Value: 0, ObservedAt: &now, Sample: &Sample{Value: 999}})
	if got.Value == nil || got.Value.GetNumberValue() != 0 || got.Trust != "VALID" || got.Empirical != "MISS" || got.ObservedAt == "" {
		t.Fatalf("evidence lost: %v", got)
	}
	sample := walkReading(MetricEntry{ID: "missing", Coverage: CoverageMissing, Trust: TrustUnavailable, Sample: &Sample{Value: 999}})
	if sample.Value != nil {
		t.Fatal("sample became a measurement")
	}
}

func TestMorningWalkProgramBehavior(t *testing.T) { // [REQ:CC-P0-015]
	cmd := exec.Command("python3", "testdata/walk_program_behavior.py")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("morning walk behavioral fixtures: %v\n%s", err, output)
	}
}
