package phases

import (
	"context"
	"io"
	"testing"
	"time"

	"test-genie/internal/orchestrator/runnability"
	"test-genie/internal/orchestrator/workspace"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// TestObservationStringGolden locks the public Observation.String() rendering so
// the Phase 2 observation-dedup work (packages emitting shared.Observation
// rather than per-package duplicate types) cannot silently shift the
// human/log-facing output.
func TestObservationStringGolden(t *testing.T) {
	cases := []struct {
		name string
		obs  Observation
		want string
	}{
		{"plain", NewObservation("hello"), "hello"},
		{"section", NewSectionObservation("🔍", "Discovery"), "🔍 Discovery"},
		{"section-no-icon", Observation{Section: "Discovery"}, "Discovery"},
		{"success", NewSuccessObservation("passed"), "[SUCCESS] ✅ passed"},
		{"warning", NewWarningObservation("careful"), "[WARNING] ⚠️ careful"},
		{"error", NewErrorObservation("boom"), "[ERROR] ❌ boom"},
		{"skip", NewSkipObservation("not today"), "[SKIP] ⏭️ not today"},
		{"info", NewInfoObservation("fyi"), "[INFO] ℹ️ fyi"},
		{"empty", Observation{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.obs.String(); got != tc.want {
				t.Fatalf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestSpecToDefinition asserts the single Spec→Definition converter copies every
// projected field, so the centralization in suite_execution stays faithful.
func TestSpecToDefinition(t *testing.T) {
	spec := Spec{
		Name:           Business,
		Runner:         func(context.Context, workspace.Environment, io.Writer) RunReport { return RunReport{} },
		Optional:       true,
		DefaultTimeout: 42 * time.Second,
		SkipEnvVar:     "TEST_GENIE_SKIP_BUSINESS",
		Capabilities:   runnability.PhaseCapabilities{Phase: "business", Optional: true},
		FindingSource:  architecturev1.FindingSource_FINDING_SOURCE_BUSINESS,
	}
	def := spec.ToDefinition()
	if def.Name != spec.Name {
		t.Errorf("Name = %q, want %q", def.Name, spec.Name)
	}
	if def.Timeout != spec.DefaultTimeout {
		t.Errorf("Timeout = %v, want %v", def.Timeout, spec.DefaultTimeout)
	}
	if def.Optional != spec.Optional {
		t.Errorf("Optional = %v, want %v", def.Optional, spec.Optional)
	}
	if def.SkipEnvVar != spec.SkipEnvVar {
		t.Errorf("SkipEnvVar = %q, want %q", def.SkipEnvVar, spec.SkipEnvVar)
	}
	if def.Capabilities.Phase != spec.Capabilities.Phase {
		t.Errorf("Capabilities.Phase = %q, want %q", def.Capabilities.Phase, spec.Capabilities.Phase)
	}
	if def.FindingSource != spec.FindingSource {
		t.Errorf("FindingSource = %v, want %v", def.FindingSource, spec.FindingSource)
	}
	if def.Runner == nil {
		t.Error("Runner not copied")
	}
}
