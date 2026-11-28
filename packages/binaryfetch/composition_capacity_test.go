package binaryfetch

import (
	"math"
	"strings"
	"testing"
)

func boundedComposeStep(role string, download, expanded, entries int64) ComposeStep {
	return ComposeStep{
		Role: role, Kind: "url", Dest: role, URL: "https://example.test/" + role + ".tar.gz",
		SHA256: strings.Repeat("a", 64), Archive: "tar.gz", DownloadBytes: download,
		MaxExpandedBytes: expanded, MaxEntries: entries,
	}
}

func TestPlanCompositionCapacitySumsPerStepBounds(t *testing.T) {
	steps := []ComposeStep{boundedComposeStep("runtime", 100, 1000, 10), boundedComposeStep("source", 200, 2000, 20)}
	plan, err := PlanCompositionCapacity(steps)
	wantPeak := int64(3*(300+3000) + 1<<20)
	if err != nil || plan.DownloadBytes != 300 || plan.ExpandedBytes != 3000 || plan.PeakBytes != wantPeak {
		t.Fatalf("composition plan = %+v, %v; want download=300 expanded=3000 peak=%d", plan, err, wantPeak)
	}
}

func TestPlanCompositionCapacityRejectsUnboundedAndUnsupportedSteps(t *testing.T) {
	bounded := boundedComposeStep("runtime", 10, 20, 2)
	for name, steps := range map[string][]ComposeStep{
		"missing entries":                              {func() ComposeStep { s := bounded; s.MaxEntries = 0; return s }()},
		"mixed bounded and unbounded":                  {bounded, {Role: "source", Kind: "url", Dest: "source", URL: "https://example.test/source.tar.gz", SHA256: strings.Repeat("b", 64), Archive: "tar.gz"}},
		"python wheels have no enforced write ceiling": {bounded, {Role: "wheels", Kind: "python-wheels", Dest: "wheels", Lockfile: "requirements.lock"}},
		"local source has no bounded copy contract":    {bounded, {Role: "local", Kind: "local", Dest: "local", Source: "local.txt"}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := PlanCompositionCapacity(steps); err == nil {
				t.Fatal("accepted composition without an enforceable bound")
			}
		})
	}
}

func TestPlanCompositionCapacityRejectsOverflow(t *testing.T) {
	for name, steps := range map[string][]ComposeStep{
		"sum overflow":  {boundedComposeStep("runtime", math.MaxInt64, 1, 1)},
		"peak overflow": {boundedComposeStep("runtime", math.MaxInt64/4, math.MaxInt64/4, 1)},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := PlanCompositionCapacity(steps); err == nil || !strings.Contains(err.Error(), "overflow") {
				t.Fatalf("overflow result = %v", err)
			}
		})
	}
}
