package contracts

import (
	"strings"
	"testing"
)

func TestEngineCapabilitiesValidate(t *testing.T) {
	valid := EngineCapabilities{
		SchemaVersion:         EventEnvelopeSchemaVersion,
		Engine:                "browserless",
		MaxConcurrentSessions: 2,
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid capabilities, got error: %v", err)
	}

	invalid := valid
	invalid.Engine = ""
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected error when engine is missing")
	}

	invalid = valid
	invalid.MaxConcurrentSessions = 0
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected error when max_concurrent_sessions <= 0")
	}

	invalid = valid
	invalid.MaxViewportWidth = -1
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected error when max_viewport_width is negative")
	}
}

func TestEngineCapabilitiesCheckCompatibility(t *testing.T) {
	caps := EngineCapabilities{
		SchemaVersion:         EventEnvelopeSchemaVersion,
		Engine:                "browserless",
		MaxConcurrentSessions: 1,
		AllowsParallelTabs:    false,
		SupportsHAR:           false,
		SupportsVideo:         true,
		SupportsIframes:       true,
		MaxViewportWidth:      1200,
		MaxViewportHeight:     800,
	}

	req := CapabilityRequirement{
		NeedsParallelTabs: true,
		NeedsHAR:          true,
		NeedsVideo:        true,
		MinViewportWidth:  1300,
		MinViewportHeight: 700,
	}

	gap := caps.CheckCompatibility(req)
	if gap.Satisfied() {
		t.Fatalf("expected gaps, got satisfied result")
	}

	expectedMissing := []string{"parallel_tabs", "har", "viewport_width>=1300"}
	for _, missing := range expectedMissing {
		if !contains(gap.Missing, missing) {
			t.Fatalf("expected missing capability %q, got %+v", missing, gap.Missing)
		}
	}

	// Height should be satisfied; video supported; warnings should be empty.
	if len(gap.Warnings) > 0 {
		t.Fatalf("expected no warnings, got %+v", gap.Warnings)
	}
}

func TestEventBufferLimitsValidate(t *testing.T) {
	if err := DefaultEventBufferLimits.Validate(); err != nil {
		t.Fatalf("expected default limits to be valid, got %v", err)
	}

	limits := EventBufferLimits{PerExecution: 0, PerAttempt: 10}
	if err := limits.Validate(); err == nil {
		t.Fatal("expected per-execution validation error")
	}

	limits = EventBufferLimits{PerExecution: 10, PerAttempt: -1}
	if err := limits.Validate(); err == nil {
		t.Fatal("expected per-attempt validation error")
	}
}

func contains(list []string, target string) bool {
	for _, item := range list {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}
