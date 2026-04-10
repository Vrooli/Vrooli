package execute

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"
)

// TestBuildProgressBar verifies the visual progress bar construction.
func TestBuildProgressBar(t *testing.T) {
	tests := []struct {
		name     string
		current  int
		total    int
		width    int
		expected string
	}{
		{
			name:     "empty progress",
			current:  0,
			total:    10,
			width:    10,
			expected: "░░░░░░░░░░",
		},
		{
			name:     "half progress",
			current:  5,
			total:    10,
			width:    10,
			expected: "█████░░░░░",
		},
		{
			name:     "full progress",
			current:  10,
			total:    10,
			width:    10,
			expected: "██████████",
		},
		{
			name:     "zero total returns empty bar",
			current:  5,
			total:    0,
			width:    10,
			expected: "░░░░░░░░░░",
		},
		{
			name:     "current exceeds total clamps to full",
			current:  15,
			total:    10,
			width:    10,
			expected: "██████████",
		},
		{
			name:     "partial fill rounds down",
			current:  1,
			total:    3,
			width:    10,
			expected: "███░░░░░░░", // 1/3 * 10 = 3.33 -> 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildProgressBar(tt.current, tt.total, tt.width)
			if result != tt.expected {
				t.Errorf("buildProgressBar(%d, %d, %d) = %q, want %q",
					tt.current, tt.total, tt.width, result, tt.expected)
			}
		})
	}
}

// TestEstimateCurrentPhase verifies phase estimation based on elapsed time.
func TestEstimateCurrentPhase(t *testing.T) {
	phaseList := []string{"structure", "unit", "integration"}
	targets := map[string]time.Duration{
		"structure":   30 * time.Second,
		"unit":        60 * time.Second,
		"integration": 120 * time.Second,
	}

	tests := []struct {
		name     string
		elapsed  time.Duration
		expected string
	}{
		{
			name:     "start shows first phase",
			elapsed:  0,
			expected: "structure",
		},
		{
			name:     "within first phase",
			elapsed:  15 * time.Second,
			expected: "structure",
		},
		{
			name:     "just past first phase",
			elapsed:  31 * time.Second,
			expected: "unit",
		},
		{
			name:     "in second phase",
			elapsed:  60 * time.Second,
			expected: "unit",
		},
		{
			name:     "in third phase",
			elapsed:  100 * time.Second,
			expected: "integration",
		},
		{
			name:     "past all phases shows last",
			elapsed:  300 * time.Second,
			expected: "integration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimateCurrentPhase(phaseList, targets, tt.elapsed)
			if result != tt.expected {
				t.Errorf("estimateCurrentPhase with elapsed=%v = %q, want %q",
					tt.elapsed, result, tt.expected)
			}
		})
	}
}

// TestEstimateCurrentPhaseWithDefaultDuration verifies fallback for unknown phases.
func TestEstimateCurrentPhaseWithDefaultDuration(t *testing.T) {
	phaseList := []string{"unknown_phase"}
	targets := map[string]time.Duration{} // No targets defined

	// Should use 60s default, so at 30s we should still be in unknown_phase
	result := estimateCurrentPhase(phaseList, targets, 30*time.Second)
	if result != "unknown_phase" {
		t.Errorf("expected unknown_phase, got %q", result)
	}
}

// TestEstimateCurrentPhaseEmptyList handles edge case of no phases.
func TestEstimateCurrentPhaseEmptyList(t *testing.T) {
	result := estimateCurrentPhase(nil, nil, 10*time.Second)
	if result != "running" {
		t.Errorf("expected 'running' for empty phase list, got %q", result)
	}
}

// TestFindPhaseIndex verifies phase index lookup.
func TestFindPhaseIndex(t *testing.T) {
	phaseList := []string{"structure", "unit", "integration"}

	tests := []struct {
		phase    string
		expected int
	}{
		{"structure", 0},
		{"unit", 1},
		{"integration", 2},
		{"nonexistent", 0}, // Returns 0 for not found
	}

	for _, tt := range tests {
		t.Run(tt.phase, func(t *testing.T) {
			result := findPhaseIndex(phaseList, tt.phase)
			if result != tt.expected {
				t.Errorf("findPhaseIndex(%q) = %d, want %d", tt.phase, result, tt.expected)
			}
		})
	}
}

// TestStartProgressWithConfigForceTTY verifies that ForceTTY overrides detection.
func TestStartProgressWithConfigForceTTY(t *testing.T) {
	t.Run("ForceTTY=true produces spinner output", func(t *testing.T) {
		var buf bytes.Buffer
		forceTTY := true

		stop := StartProgressWithConfig(ProgressConfig{
			Writer:    &buf,
			PhaseList: []string{"structure"},
			Targets:   map[string]time.Duration{"structure": 1 * time.Second},
			ForceTTY:  &forceTTY,
		})

		// Wait for at least one tick
		time.Sleep(250 * time.Millisecond)
		stop()

		output := buf.String()

		// TTY mode should contain carriage returns and spinner characters
		if !strings.Contains(output, "\r") {
			t.Error("TTY mode should contain carriage returns for line overwriting")
		}

		// Should contain at least one spinner character
		hasSpinner := false
		for _, char := range spinnerChars {
			if strings.Contains(output, char) {
				hasSpinner = true
				break
			}
		}
		if !hasSpinner {
			t.Error("TTY mode should contain spinner characters")
		}

		// Should contain progress bar characters
		if !strings.Contains(output, "█") && !strings.Contains(output, "░") {
			t.Error("TTY mode should contain progress bar characters")
		}
	})

	t.Run("ForceTTY=false produces phase-change output", func(t *testing.T) {
		var buf bytes.Buffer
		forceTTY := false

		stop := StartProgressWithConfig(ProgressConfig{
			Writer:    &buf,
			PhaseList: []string{"structure", "unit"},
			Targets: map[string]time.Duration{
				"structure": 100 * time.Millisecond,
				"unit":      100 * time.Millisecond,
			},
			Timeouts: map[string]time.Duration{
				"structure": 5 * time.Second,
				"unit":      10 * time.Second,
			},
			ForceTTY: &forceTTY,
		})

		// Wait for initial output
		time.Sleep(50 * time.Millisecond)
		stop()

		output := buf.String()

		// Non-TTY mode should NOT contain carriage returns (except in final message)
		lines := strings.Split(output, "\n")
		for i, line := range lines[:len(lines)-1] { // Exclude last empty line
			if strings.Contains(line, "\r") && i < len(lines)-2 {
				t.Errorf("Non-TTY progress lines should not contain carriage returns: %q", line)
			}
		}

		// Should contain phase indicator format
		if !strings.Contains(output, "[1/") {
			t.Error("Non-TTY mode should contain phase indicator like [1/N]")
		}

		// Should contain "Running ... phase" format
		if !strings.Contains(output, "Running") || !strings.Contains(output, "phase") {
			t.Error("Non-TTY mode should contain 'Running ... phase' format")
		}
		if !strings.Contains(output, "timeout 5s") {
			t.Error("Non-TTY mode should include timeout budget when provided")
		}
	})
}

// TestNonTTYProgressOnlyPrintsOnPhaseChange verifies minimal output in non-TTY mode.
func TestNonTTYProgressOnlyPrintsOnPhaseChange(t *testing.T) {
	var buf bytes.Buffer
	forceTTY := false

	stop := StartProgressWithConfig(ProgressConfig{
		Writer:    &buf,
		PhaseList: []string{"fast_phase"},
		Targets: map[string]time.Duration{
			"fast_phase": 10 * time.Second, // Long duration so phase doesn't change
		},
		ForceTTY: &forceTTY,
	})

	// Wait for multiple tick cycles
	time.Sleep(2500 * time.Millisecond)
	stop()

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have exactly 2 lines: initial phase message + stop message
	// (No repeated updates since phase doesn't change)
	if len(lines) != 2 {
		t.Errorf("Non-TTY mode should only print on phase change, got %d lines:\n%s",
			len(lines), output)
	}
}

// TestIsTTYFuncTestingSeam verifies the testing seam works correctly.
func TestIsTTYFuncTestingSeam(t *testing.T) {
	// Save original function
	originalIsTTYFunc := isTTYFunc
	defer func() { isTTYFunc = originalIsTTYFunc }()

	t.Run("mock returns true triggers TTY mode", func(t *testing.T) {
		// Override with mock that always returns true
		isTTYFunc = func(w io.Writer) bool { return true }

		var buf bytes.Buffer
		stop := StartProgress(&buf, []string{"test"}, map[string]time.Duration{"test": 1 * time.Second}, nil)
		time.Sleep(250 * time.Millisecond)
		stop()

		output := buf.String()
		// TTY mode uses carriage returns
		if !strings.Contains(output, "\r") {
			t.Error("When isTTYFunc returns true, should use TTY mode with carriage returns")
		}
	})

	t.Run("mock returns false triggers non-TTY mode", func(t *testing.T) {
		// Override with mock that always returns false
		isTTYFunc = func(w io.Writer) bool { return false }

		var buf bytes.Buffer
		stop := StartProgress(&buf, []string{"test"}, map[string]time.Duration{"test": 1 * time.Second}, nil)
		time.Sleep(50 * time.Millisecond)
		stop()

		output := buf.String()
		// Non-TTY mode uses newlines, not carriage returns for progress
		if !strings.Contains(output, "[1/") {
			t.Error("When isTTYFunc returns false, should use non-TTY mode with phase format")
		}
	})
}

// TestDefaultIsTTYWithNonFileWriter verifies non-file writers return false.
func TestDefaultIsTTYWithNonFileWriter(t *testing.T) {
	// bytes.Buffer is not an *os.File, so should return false
	var buf bytes.Buffer
	result := defaultIsTTY(&buf)
	if result {
		t.Error("defaultIsTTY should return false for non-*os.File writers")
	}
}

// TestStartProgressDefaultPhases verifies default phases when none provided.
func TestStartProgressDefaultPhases(t *testing.T) {
	var buf bytes.Buffer
	forceTTY := false

	stop := StartProgressWithConfig(ProgressConfig{
		Writer:    &buf,
		PhaseList: nil, // No phases provided
		Targets:   nil,
		ForceTTY:  &forceTTY,
	})

	time.Sleep(50 * time.Millisecond)
	stop()

	output := buf.String()
	// Should use default phases: structure, unit, integration
	if !strings.Contains(output, "structure") {
		t.Error("Should use 'structure' as first default phase")
	}
	if !strings.Contains(output, "[1/3]") {
		t.Error("Should have 3 default phases")
	}
}

// TestTTYProgressContainsExpectedElements verifies all TTY output components.
func TestTTYProgressContainsExpectedElements(t *testing.T) {
	var buf bytes.Buffer
	forceTTY := true

	stop := StartProgressWithConfig(ProgressConfig{
		Writer:    &buf,
		PhaseList: []string{"structure", "unit"},
		Targets: map[string]time.Duration{
			"structure": 30 * time.Second,
			"unit":      60 * time.Second,
		},
		ForceTTY: &forceTTY,
	})

	time.Sleep(250 * time.Millisecond)
	stop()

	output := buf.String()

	checks := []struct {
		name     string
		contains string
	}{
		{"phase count", "[1/2]"},
		{"phase name", "structure"},
		{"running text", "Running tests"},
		{"elapsed label", "elapsed"},
		{"remaining label", "remaining"},
	}

	for _, check := range checks {
		if !strings.Contains(output, check.contains) {
			t.Errorf("TTY output should contain %s (%q), got:\n%s",
				check.name, check.contains, output)
		}
	}
}

// TestNonTTYStopMessageIncludesElapsed verifies the stop message format.
func TestNonTTYStopMessageIncludesElapsed(t *testing.T) {
	var buf bytes.Buffer
	forceTTY := false

	stop := StartProgressWithConfig(ProgressConfig{
		Writer:    &buf,
		PhaseList: []string{"test"},
		Targets:   map[string]time.Duration{"test": 1 * time.Second},
		ForceTTY:  &forceTTY,
	})

	time.Sleep(50 * time.Millisecond)
	stop()

	output := buf.String()
	if !strings.Contains(output, "Progress tracking stopped after") {
		t.Errorf("Stop message should include elapsed time, got:\n%s", output)
	}
}
