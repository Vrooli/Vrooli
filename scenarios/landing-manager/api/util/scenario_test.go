package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetVrooliRoot_UsesEnvVar(t *testing.T) {
	t.Setenv("VROOLI_ROOT", "/tmp/custom-root")
	got := GetVrooliRoot()
	if got != "/tmp/custom-root" {
		t.Fatalf("expected /tmp/custom-root, got %s", got)
	}
}

func TestGetVrooliRoot_DerivesFromExecutablePath(t *testing.T) {
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("HOME", "/home/ignore-me")

	originalExecutable := executablePath
	t.Cleanup(func() {
		executablePath = originalExecutable
	})

	fakeRoot := filepath.Join(os.TempDir(), "vrooli-root")
	fakeExec := filepath.Join(fakeRoot, "scenarios", "landing-manager", "api", "landing-manager-api")
	executablePath = func() (string, error) {
		return fakeExec, nil
	}

	got := GetVrooliRoot()
	if got != fakeRoot {
		t.Fatalf("expected %s, got %s", fakeRoot, got)
	}
}

func TestIsScenarioNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		output   string
		expected bool
	}{
		{
			name:     "scenario not found message",
			output:   "[ERROR] Scenario not found: nonexistent (path: /tmp/nope)",
			expected: true,
		},
		{
			name:     "psql not found should be ignored",
			output:   "psql not found; runtime will auto-init schema",
			expected: false,
		},
		{
			name:     "lifecycle log missing",
			output:   "No lifecycle log found for scenario test-scenario",
			expected: true,
		},
		{
			name:     "no such scenario message",
			output:   "Error: No such scenario 'foo'",
			expected: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsScenarioNotFound(tt.output); got != tt.expected {
				t.Fatalf("expected %v for output %q, got %v", tt.expected, tt.output, got)
			}
		})
	}
}
