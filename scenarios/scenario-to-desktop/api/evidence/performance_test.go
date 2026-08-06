package evidence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"scenario-to-desktop-api/smoketest"
)

func TestLaunchPhaseDurationsKeepsUnavailableSegmentsExplicit(t *testing.T) {
	trace := smoketest.LaunchTrace{Events: []smoketest.LaunchEvent{
		{Name: smoketest.EventDemoSpawn, MonotonicNs: 10},
		{Name: smoketest.EventSplashFirstPaint, MonotonicNs: 110},
		{Name: smoketest.EventAppReady, MonotonicNs: 510},
	}}
	phases := LaunchPhaseDurations(trace)
	if !phases[0].Available || phases[0].DurationMs != 0 {
		t.Fatalf("process-to-splash phase = %+v", phases[0])
	}
	// The duration is sub-millisecond by design in this fixture; no phase
	// should be turned into a fabricated non-zero duration.
	if phases[3].Available || phases[3].Reason == "" {
		t.Fatalf("missing server/application phase should be explicit: %+v", phases[3])
	}
}

func TestSummarizeDurationsUsesNearestRankPercentiles(t *testing.T) {
	distribution, err := SummarizeDurations([]time.Duration{
		10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond,
		40 * time.Millisecond, 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if distribution.Count != 5 || distribution.P50Ms != 30 || distribution.P95Ms != 100 || distribution.Spread != 90 {
		t.Fatalf("distribution = %+v", distribution)
	}
	if _, err := SummarizeDurations([]time.Duration{-time.Millisecond}); err == nil {
		t.Fatal("negative duration should be rejected")
	}
}

func baselineSample(id string, class MeasurementClass, duration int64) PerformanceSample {
	return PerformanceSample{
		RunID: id, Class: class, HostFingerprint: "host-a", ArtifactDigest: "artifact-a",
		Display: "1920x1080", DeploymentMode: "bundled", ProfilerMode: "disabled",
		Phases: []PhaseDuration{{Name: "process_to_splash_first_paint", Available: true, DurationMs: duration}},
	}
}

func TestBuildBaselineRequiresRepeatedColdAndWarmSamples(t *testing.T) {
	samples := make([]PerformanceSample, 0, 10)
	for i := 0; i < 5; i++ {
		samples = append(samples, baselineSample(fmt.Sprintf("cold-%d", i), MeasurementCold, int64(100+i)))
		samples = append(samples, baselineSample(fmt.Sprintf("warm-%d", i), MeasurementWarm, int64(50+i)))
	}
	baseline, err := BuildPerformanceBaseline("hello-desktop-linux-xvfb", samples)
	if err != nil {
		t.Fatal(err)
	}
	if baseline.Cold["process_to_splash_first_paint"].Count != 5 || baseline.Warm["process_to_splash_first_paint"].Count != 5 {
		t.Fatalf("baseline distributions = cold=%+v warm=%+v", baseline.Cold, baseline.Warm)
	}
}

func TestCompareBaselineFailsClosedAndFindsRegression(t *testing.T) {
	samples := make([]PerformanceSample, 0, 10)
	for i := 0; i < 5; i++ {
		samples = append(samples, baselineSample(fmt.Sprintf("cold-%d", i), MeasurementCold, 100))
		samples = append(samples, baselineSample(fmt.Sprintf("warm-%d", i), MeasurementWarm, 50))
	}
	baseline, err := BuildPerformanceBaseline("baseline", samples)
	if err != nil {
		t.Fatal(err)
	}
	current := []PerformanceSample{baselineSample("current-1", MeasurementCold, 200)}
	comparison := ComparePerformanceBaseline(baseline, current, MeasurementCold, 0.1)
	if comparison.Status != ComparisonRegression || comparison.SlowestSegment != "process_to_splash_first_paint" {
		t.Fatalf("comparison = %+v", comparison)
	}
	current[0].HostFingerprint = "different-host"
	comparison = ComparePerformanceBaseline(baseline, current, MeasurementCold, 0.1)
	if comparison.Status != ComparisonNonComparable {
		t.Fatalf("identity drift should be non-comparable: %+v", comparison)
	}
}

func TestDurableHelloDesktopBaselineIsComparableInput(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "internal", "performance-baseline.json"))
	if err != nil {
		t.Fatal(err)
	}
	var baseline PerformanceBaseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		t.Fatal(err)
	}
	if len(baseline.Samples) != 10 || baseline.Cold["process_to_splash_first_paint"].P95Ms != 1745 || baseline.Warm["process_to_splash_first_paint"].P95Ms != 651 {
		t.Fatalf("unexpected durable baseline: %+v", baseline)
	}
	if _, err := BuildPerformanceBaseline(baseline.Name, baseline.Samples); err != nil {
		t.Fatalf("durable baseline samples are not rebuildable: %v", err)
	}
}
