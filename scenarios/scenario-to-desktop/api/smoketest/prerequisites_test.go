package smoketest_test

import (
	"os"
	"path/filepath"
	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/smoketest/mocks"
	"testing"
)

func TestPrerequisiteChecker_CheckDisplayAvailable(t *testing.T) {
	tests := []struct {
		name       string
		display    string
		wayland    string
		xvfbAvail  bool
		wantPassed bool
	}{
		{
			name:       "DISPLAY set",
			display:    ":0",
			wantPassed: true,
		},
		{
			name:       "WAYLAND_DISPLAY set",
			wayland:    "wayland-0",
			wantPassed: true,
		},
		{
			name:       "xvfb-run available",
			xvfbAvail:  true,
			wantPassed: true,
		},
		{
			name:       "no display and no xvfb",
			wantPassed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envReader := mocks.NewMockEnvironmentReader()
			if tt.display != "" {
				envReader.SetEnv("DISPLAY", tt.display)
			}
			if tt.wayland != "" {
				envReader.SetEnv("WAYLAND_DISPLAY", tt.wayland)
			}

			executor := mocks.NewMockProcessExecutor()
			if tt.xvfbAvail {
				executor.AddLookPath("xvfb-run", "/usr/bin/xvfb-run")
			}

			fs := mocks.NewMockFileSystem()
			checker := smoketest.NewPrerequisiteChecker(envReader, fs, executor)

			result := checker.CheckDisplayAvailable()

			if result.Passed != tt.wantPassed {
				t.Errorf("CheckDisplayAvailable() Passed = %v, want %v; message: %s",
					result.Passed, tt.wantPassed, result.Message)
			}

			if result.Kind != smoketest.PrereqDisplay {
				t.Errorf("CheckDisplayAvailable() Kind = %v, want %v",
					result.Kind, smoketest.PrereqDisplay)
			}
		})
	}
}

func TestPrerequisiteChecker_CheckArtifactExecutable(t *testing.T) {
	// Create a real temp file for testing
	tmpDir := t.TempDir()
	executablePath := filepath.Join(tmpDir, "test-executable")
	if err := os.WriteFile(executablePath, []byte("#!/bin/bash\necho test"), 0o755); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	nonExecutablePath := filepath.Join(tmpDir, "test-non-executable")
	if err := os.WriteFile(nonExecutablePath, []byte("data"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name       string
		path       string
		wantPassed bool
		wantFatal  bool
	}{
		{
			name:       "executable file",
			path:       executablePath,
			wantPassed: true,
			wantFatal:  true,
		},
		{
			name:       "non-executable file",
			path:       nonExecutablePath,
			wantPassed: false,
			wantFatal:  true,
		},
		{
			name:       "non-existent file",
			path:       "/nonexistent/path/to/file",
			wantPassed: false,
			wantFatal:  true,
		},
		{
			name:       "directory instead of file",
			path:       tmpDir,
			wantPassed: false,
			wantFatal:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envReader := mocks.NewMockEnvironmentReader()
			executor := mocks.NewMockProcessExecutor()
			fs := smoketest.NewFileSystem() // Use real filesystem

			checker := smoketest.NewPrerequisiteChecker(envReader, fs, executor)

			result := checker.CheckArtifactExecutable(tt.path)

			if result.Passed != tt.wantPassed {
				t.Errorf("CheckArtifactExecutable() Passed = %v, want %v; message: %s",
					result.Passed, tt.wantPassed, result.Message)
			}

			if result.Fatal != tt.wantFatal {
				t.Errorf("CheckArtifactExecutable() Fatal = %v, want %v",
					result.Fatal, tt.wantFatal)
			}

			if result.Kind != smoketest.PrereqArtifactExecutable {
				t.Errorf("CheckArtifactExecutable() Kind = %v, want %v",
					result.Kind, smoketest.PrereqArtifactExecutable)
			}
		})
	}
}

func TestPrerequisiteChecker_CheckDiskSpace(t *testing.T) {
	// Use a real path that exists
	tmpDir := t.TempDir()

	envReader := mocks.NewMockEnvironmentReader()
	executor := mocks.NewMockProcessExecutor()
	fs := smoketest.NewFileSystem()

	checker := smoketest.NewPrerequisiteChecker(envReader, fs, executor)

	// Check with a reasonable minimum (100 bytes - should pass on any system)
	result := checker.CheckDiskSpace(tmpDir, 100)

	if !result.Passed {
		t.Errorf("CheckDiskSpace() Passed = false for small requirement; message: %s", result.Message)
	}

	if result.Kind != smoketest.PrereqDiskSpace {
		t.Errorf("CheckDiskSpace() Kind = %v, want %v", result.Kind, smoketest.PrereqDiskSpace)
	}
}

func TestPrerequisiteChecker_CheckAll(t *testing.T) {
	tmpDir := t.TempDir()
	executablePath := filepath.Join(tmpDir, "test-app")
	if err := os.WriteFile(executablePath, []byte("#!/bin/bash"), 0o755); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	envReader := mocks.NewMockEnvironmentReader()
	envReader.SetEnv("DISPLAY", ":0")

	executor := mocks.NewMockProcessExecutor()
	executor.ExecuteResult.Output = "" // No ports in use

	fs := smoketest.NewFileSystem()

	checker := smoketest.NewPrerequisiteChecker(envReader, fs, executor)

	results := checker.CheckAll(executablePath, "linux", 8080)

	// Should have at least display, artifact, and disk space checks
	if len(results) < 3 {
		t.Errorf("CheckAll() returned %d results, expected at least 3", len(results))
	}

	// Verify no fatal failures
	if checker.HasFatalFailure(results) {
		for _, r := range results {
			if !r.Passed && r.Fatal {
				t.Errorf("Unexpected fatal failure: %s - %s", r.Kind, r.Message)
			}
		}
	}
}

func TestPrerequisiteChecker_HasFatalFailure(t *testing.T) {
	tests := []struct {
		name    string
		results []smoketest.PrerequisiteResult
		wantHas bool
	}{
		{
			name: "all passed",
			results: []smoketest.PrerequisiteResult{
				{Passed: true, Fatal: true},
				{Passed: true, Fatal: false},
			},
			wantHas: false,
		},
		{
			name: "non-fatal failure",
			results: []smoketest.PrerequisiteResult{
				{Passed: true, Fatal: true},
				{Passed: false, Fatal: false},
			},
			wantHas: false,
		},
		{
			name: "fatal failure",
			results: []smoketest.PrerequisiteResult{
				{Passed: true, Fatal: true},
				{Passed: false, Fatal: true},
			},
			wantHas: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envReader := mocks.NewMockEnvironmentReader()
			executor := mocks.NewMockProcessExecutor()
			fs := mocks.NewMockFileSystem()
			checker := smoketest.NewPrerequisiteChecker(envReader, fs, executor)

			got := checker.HasFatalFailure(tt.results)
			if got != tt.wantHas {
				t.Errorf("HasFatalFailure() = %v, want %v", got, tt.wantHas)
			}
		})
	}
}

func TestFormatResults(t *testing.T) {
	results := []smoketest.PrerequisiteResult{
		{Kind: smoketest.PrereqDisplay, Passed: true, Message: "DISPLAY is set"},
		{Kind: smoketest.PrereqArtifactExecutable, Passed: false, Fatal: true, Message: "Not executable", Suggestion: "chmod +x file"},
		{Kind: smoketest.PrereqDiskSpace, Passed: false, Fatal: false, Message: "Low space", Suggestion: "Free up disk"},
	}

	formatted := smoketest.FormatResults(results)

	// Check that it contains expected elements
	if !containsString(formatted, "DISPLAY") {
		t.Error("Expected formatted output to contain 'DISPLAY'")
	}
	if !containsString(formatted, "Not executable") {
		t.Error("Expected formatted output to contain 'Not executable'")
	}
	if !containsString(formatted, "chmod +x") {
		t.Error("Expected formatted output to contain suggestion")
	}
	if !containsString(formatted, "Summary") {
		t.Error("Expected formatted output to contain 'Summary'")
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestPrerequisiteKind_String(t *testing.T) {
	tests := []struct {
		kind smoketest.PrerequisiteKind
		want string
	}{
		{smoketest.PrereqDisplay, "display"},
		{smoketest.PrereqArtifactExecutable, "artifact_executable"},
		{smoketest.PrereqDiskSpace, "disk_space"},
		{smoketest.PrereqPort, "port"},
		{smoketest.PrerequisiteKind(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.want {
				t.Errorf("PrerequisiteKind.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
