package evidence

import (
	"fmt"
	"testing"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func classScopeSample(id string, class MeasurementClass, phases ...deliveryramp.PhaseDuration) PerformanceSample {
	return PerformanceSample{
		RunID: id, Class: class, HostFingerprint: "class-scope-host", ArtifactDigest: "class-scope-artifact",
		Display: "1920x1080", DeploymentMode: "bundled", ProfilerMode: "disabled", Phases: phases,
	}
}

func classScopePhase(name string, duration int64) deliveryramp.PhaseDuration {
	return deliveryramp.PhaseDuration{Name: name, Available: true, DurationMs: duration}
}

func TestCompareBaselineScopesRequiredPhasesToMeasurementClass(t *testing.T) {
	const coldPhase = "process_to_splash_first_paint"
	const warmOnlyPhase = "server_ready"

	samples := make([]PerformanceSample, 0, 10)
	for i := 0; i < 5; i++ {
		samples = append(samples,
			classScopeSample(fmt.Sprintf("cold-%d", i), MeasurementCold,
				classScopePhase(coldPhase, 100)),
			classScopeSample(fmt.Sprintf("warm-%d", i), MeasurementWarm,
				classScopePhase(coldPhase, 50), classScopePhase(warmOnlyPhase, 20)),
		)
	}

	baseline, err := BuildPerformanceBaseline("class-scoped-phases", samples)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := baseline.Cold[coldPhase]; !ok {
		t.Fatalf("cold baseline lacks %q: %+v", coldPhase, baseline.Cold)
	}
	if _, ok := baseline.Cold[warmOnlyPhase]; ok {
		t.Fatalf("warm-only phase %q leaked into cold baseline: %+v", warmOnlyPhase, baseline.Cold)
	}
	if _, ok := baseline.Warm[warmOnlyPhase]; !ok {
		t.Fatalf("warm baseline lacks warm-only phase %q: %+v", warmOnlyPhase, baseline.Warm)
	}

	current := []PerformanceSample{classScopeSample("current-cold", MeasurementCold,
		classScopePhase(coldPhase, 100))}
	comparison := ComparePerformanceBaseline(baseline, current, MeasurementCold, 0.1)
	if comparison.Status != ComparisonPass {
		t.Fatalf("cold comparison should pass without warm-only phase %q, got %+v", warmOnlyPhase, comparison)
	}
}
