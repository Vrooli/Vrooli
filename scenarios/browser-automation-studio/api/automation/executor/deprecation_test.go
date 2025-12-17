package executor

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

func TestDeprecationTrackerDisabledByDefault(t *testing.T) {
	// Create a fresh tracker (not the global one)
	tracker := &deprecationTracker{}

	if tracker.IsEnabled() {
		t.Error("Expected tracker to be disabled by default")
	}

	// Recording should be a no-op when disabled
	tracker.RecordTypeUsage("test")
	tracker.RecordParamsUsage("test")

	stats := tracker.Stats()
	if stats.TotalHits != 0 {
		t.Errorf("Expected 0 hits when disabled, got %d", stats.TotalHits)
	}
}

func TestDeprecationTrackerEnable(t *testing.T) {
	tracker := &deprecationTracker{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Suppress warning output in tests

	tracker.Enable(logger)

	if !tracker.IsEnabled() {
		t.Error("Expected tracker to be enabled after Enable()")
	}

	tracker.RecordTypeUsage("test-context")

	stats := tracker.Stats()
	if stats.TotalHits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.TotalHits)
	}
	if stats.UniqueCallers != 1 {
		t.Errorf("Expected 1 unique caller, got %d", stats.UniqueCallers)
	}
	if stats.ByField["Type"] != 1 {
		t.Errorf("Expected 1 Type hit, got %d", stats.ByField["Type"])
	}
}

func TestDeprecationTrackerDisable(t *testing.T) {
	tracker := &deprecationTracker{}
	tracker.Enable(nil)

	if !tracker.IsEnabled() {
		t.Error("Expected tracker to be enabled")
	}

	tracker.Disable()

	if tracker.IsEnabled() {
		t.Error("Expected tracker to be disabled after Disable()")
	}
}

func TestDeprecationTrackerReset(t *testing.T) {
	tracker := &deprecationTracker{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	tracker.Enable(logger)

	tracker.RecordTypeUsage("test1")
	tracker.RecordParamsUsage("test2")

	stats := tracker.Stats()
	if stats.TotalHits != 2 {
		t.Errorf("Expected 2 hits before reset, got %d", stats.TotalHits)
	}

	tracker.Reset()

	stats = tracker.Stats()
	if stats.TotalHits != 0 {
		t.Errorf("Expected 0 hits after reset, got %d", stats.TotalHits)
	}
	if stats.UniqueCallers != 0 {
		t.Errorf("Expected 0 unique callers after reset, got %d", stats.UniqueCallers)
	}
}

func TestDeprecationTrackerByField(t *testing.T) {
	tracker := &deprecationTracker{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	tracker.Enable(logger)

	tracker.RecordTypeUsage("type1")
	tracker.RecordTypeUsage("type2")
	tracker.RecordParamsUsage("params1")

	stats := tracker.Stats()
	if stats.ByField["Type"] != 2 {
		t.Errorf("Expected 2 Type hits, got %d", stats.ByField["Type"])
	}
	if stats.ByField["Params"] != 1 {
		t.Errorf("Expected 1 Params hit, got %d", stats.ByField["Params"])
	}
}

func TestDeprecationTrackerEnvVar(t *testing.T) {
	// Test that environment variable enables tracking
	originalValue := os.Getenv("BAS_DEPRECATION_WARNINGS")
	defer os.Setenv("BAS_DEPRECATION_WARNINGS", originalValue)

	os.Setenv("BAS_DEPRECATION_WARNINGS", "1")

	// Create new tracker that checks env var
	tracker := &deprecationTracker{
		enabled: os.Getenv("BAS_DEPRECATION_WARNINGS") == "1",
	}

	if !tracker.IsEnabled() {
		t.Error("Expected tracker to be enabled via env var")
	}
}

func TestInstructionStepTypeDeprecation(t *testing.T) {
	// Save and restore global tracker state
	originalEnabled := DeprecationTracker.IsEnabled()
	defer func() {
		if !originalEnabled {
			DeprecationTracker.Disable()
		}
	}()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	DeprecationTracker.Enable(logger)
	DeprecationTracker.Reset()

	// Test with deprecated Type field
	instr := contracts.CompiledInstruction{
		NodeID: "test-node",
		Type:   "click",
	}

	result := InstructionStepType(instr)
	if result != "click" {
		t.Errorf("Expected 'click', got %q", result)
	}

	stats := DeprecationTracker.Stats()
	if stats.TotalHits == 0 {
		t.Error("Expected deprecation warning to be recorded")
	}
}

func TestPlanStepParamsDeprecation(t *testing.T) {
	// Save and restore global tracker state
	originalEnabled := DeprecationTracker.IsEnabled()
	defer func() {
		if !originalEnabled {
			DeprecationTracker.Disable()
		}
	}()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	DeprecationTracker.Enable(logger)
	DeprecationTracker.Reset()

	// Test with deprecated Params field
	step := contracts.PlanStep{
		NodeID: "test-node",
		Params: map[string]any{"selector": "#button"},
	}

	result := PlanStepParams(step)
	if result["selector"] != "#button" {
		t.Errorf("Expected selector to be '#button', got %v", result["selector"])
	}

	stats := DeprecationTracker.Stats()
	if stats.TotalHits == 0 {
		t.Error("Expected deprecation warning to be recorded")
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{-1, "-1"},
		{-42, "-42"},
		{12345, "12345"},
		{-12345, "-12345"},
	}

	for _, tt := range tests {
		result := itoa(tt.input)
		if result != tt.expected {
			t.Errorf("itoa(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
