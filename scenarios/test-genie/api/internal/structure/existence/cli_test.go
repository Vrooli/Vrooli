package existence

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectCLIApproach_CrossPlatform(t *testing.T) {
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	mustMkdirCLI(t, cliDir)

	// Create cross-platform indicators
	writeFileCLI(t, filepath.Join(cliDir, "main.go"), "package main\nfunc main() {}")
	writeFileCLI(t, filepath.Join(cliDir, "go.mod"), "module test-scenario/cli")

	approach := DetectCLIApproach(root, "test-scenario")
	if approach != CLIApproachCrossPlatform {
		t.Errorf("expected CLIApproachCrossPlatform, got %s", approach)
	}
}

func TestDetectCLIApproach_Legacy(t *testing.T) {
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	mustMkdirCLI(t, cliDir)

	// Create legacy indicator - bash script
	writeFileCLI(t, filepath.Join(cliDir, "test-scenario"), "#!/usr/bin/env bash\necho hello")

	approach := DetectCLIApproach(root, "test-scenario")
	if approach != CLIApproachLegacy {
		t.Errorf("expected CLIApproachLegacy, got %s", approach)
	}
}

func TestDetectCLIApproach_CrossPlatformWithBinary(t *testing.T) {
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	mustMkdirCLI(t, cliDir)

	// Create cross-platform Go files
	writeFileCLI(t, filepath.Join(cliDir, "main.go"), "package main\nfunc main() {}")
	writeFileCLI(t, filepath.Join(cliDir, "go.mod"), "module test-scenario/cli")

	// Also create a binary (simulated with null bytes)
	writeFileCLI(t, filepath.Join(cliDir, "test-scenario"), string([]byte{0x00, 0x01, 0x02}))

	approach := DetectCLIApproach(root, "test-scenario")
	if approach != CLIApproachCrossPlatform {
		t.Errorf("expected CLIApproachCrossPlatform when Go sources + binary exist, got %s", approach)
	}
}

func TestDetectCLIApproach_Unknown(t *testing.T) {
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	mustMkdirCLI(t, cliDir)

	// Empty cli directory
	approach := DetectCLIApproach(root, "test-scenario")
	if approach != CLIApproachUnknown {
		t.Errorf("expected CLIApproachUnknown for empty cli dir, got %s", approach)
	}
}

func TestDetectCLIApproach_UnknownOnlyMainGo(t *testing.T) {
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	mustMkdirCLI(t, cliDir)

	// Only main.go without go.mod
	writeFileCLI(t, filepath.Join(cliDir, "main.go"), "package main")

	approach := DetectCLIApproach(root, "test-scenario")
	if approach != CLIApproachUnknown {
		t.Errorf("expected CLIApproachUnknown when only main.go exists, got %s", approach)
	}
}

func TestCLIApproach_String(t *testing.T) {
	tests := []struct {
		approach CLIApproach
		expected string
	}{
		{CLIApproachLegacy, "legacy"},
		{CLIApproachCrossPlatform, "cross-platform"},
		{CLIApproachUnknown, "unknown"},
	}

	for _, tc := range tests {
		if got := tc.approach.String(); got != tc.expected {
			t.Errorf("%v.String() = %q, want %q", tc.approach, got, tc.expected)
		}
	}
}

func TestValidateCLI_CrossPlatformValid(t *testing.T) {
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	mustMkdirCLI(t, cliDir)

	// Create valid cross-platform CLI structure
	writeFileCLI(t, filepath.Join(cliDir, "main.go"), "package main\nfunc main() {}")
	writeFileCLI(t, filepath.Join(cliDir, "go.mod"), "module test-scenario/cli")
	writeFileCLI(t, filepath.Join(cliDir, "install.sh"), "#!/bin/bash\necho install")

	result := ValidateCLI(root, "test-scenario", io.Discard)
	if !result.Result.Success {
		t.Fatalf("expected success, got error: %v", result.Result.Error)
	}
	if result.Approach != CLIApproachCrossPlatform {
		t.Errorf("expected cross-platform approach, got %s", result.Approach)
	}
}

func TestValidateCLI_CrossPlatformMissingInstallSh(t *testing.T) {
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	mustMkdirCLI(t, cliDir)

	// Missing install.sh
	writeFileCLI(t, filepath.Join(cliDir, "main.go"), "package main\nfunc main() {}")
	writeFileCLI(t, filepath.Join(cliDir, "go.mod"), "module test-scenario/cli")

	result := ValidateCLI(root, "test-scenario", io.Discard)
	if result.Result.Success {
		t.Fatal("expected failure when install.sh missing")
	}
	if result.Approach != CLIApproachCrossPlatform {
		t.Errorf("expected cross-platform approach, got %s", result.Approach)
	}
}

func TestValidateCLI_CrossPlatformWithWindowsSupport(t *testing.T) {
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	mustMkdirCLI(t, cliDir)

	// Create valid cross-platform CLI with Windows support
	writeFileCLI(t, filepath.Join(cliDir, "main.go"), "package main\nfunc main() {}")
	writeFileCLI(t, filepath.Join(cliDir, "go.mod"), "module test-scenario/cli")
	writeFileCLI(t, filepath.Join(cliDir, "install.sh"), "#!/bin/bash\necho install")
	writeFileCLI(t, filepath.Join(cliDir, "install.ps1"), "Write-Host 'Install'")

	result := ValidateCLI(root, "test-scenario", io.Discard)
	if !result.Result.Success {
		t.Fatalf("expected success, got error: %v", result.Result.Error)
	}

	// Check for Windows support observation
	hasWindowsObs := false
	for _, obs := range result.Result.Observations {
		if obs.Type == ObservationSuccess && containsStr(obs.Message, "Windows") {
			hasWindowsObs = true
			break
		}
	}
	if !hasWindowsObs {
		t.Error("expected observation about Windows support")
	}
}

func TestValidateCLI_LegacyValid(t *testing.T) {
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	mustMkdirCLI(t, cliDir)

	// Create valid legacy CLI structure
	writeExecutableCLI(t, filepath.Join(cliDir, "test-scenario"), "#!/bin/bash\necho cli")
	writeFileCLI(t, filepath.Join(cliDir, "install.sh"), "#!/bin/bash\necho install")

	result := ValidateCLI(root, "test-scenario", io.Discard)
	if !result.Result.Success {
		t.Fatalf("expected success, got error: %v", result.Result.Error)
	}
	if result.Approach != CLIApproachLegacy {
		t.Errorf("expected legacy approach, got %s", result.Approach)
	}
}

func TestValidateCLI_LegacyNonExecutable(t *testing.T) {
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	mustMkdirCLI(t, cliDir)

	// Create non-executable script (should produce warning)
	writeFileCLI(t, filepath.Join(cliDir, "test-scenario"), "#!/bin/bash\necho cli")
	writeFileCLI(t, filepath.Join(cliDir, "install.sh"), "#!/bin/bash\necho install")

	result := ValidateCLI(root, "test-scenario", io.Discard)
	if !result.Result.Success {
		t.Fatalf("expected success (with warning), got error: %v", result.Result.Error)
	}

	// Check for warning observation
	hasWarning := false
	for _, obs := range result.Result.Observations {
		if obs.Type == ObservationWarning {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected warning about non-executable script")
	}
}

func TestValidateCLI_MissingCLIDir(t *testing.T) {
	root := t.TempDir()
	// Don't create cli directory

	result := ValidateCLI(root, "test-scenario", io.Discard)
	if result.Result.Success {
		t.Fatal("expected failure when cli directory missing")
	}
	if result.Approach != CLIApproachUnknown {
		t.Errorf("expected unknown approach when dir missing, got %s", result.Approach)
	}
}

func TestValidateCLI_UnknownApproachGuidance(t *testing.T) {
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	mustMkdirCLI(t, cliDir)

	// Only main.go (missing go.mod)
	writeFileCLI(t, filepath.Join(cliDir, "main.go"), "package main")

	result := ValidateCLI(root, "test-scenario", io.Discard)
	if result.Result.Success {
		t.Fatal("expected failure for incomplete CLI structure")
	}
	if result.Result.Remediation == "" {
		t.Error("expected remediation guidance")
	}
}

func TestCLIValidator_Interface(t *testing.T) {
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	mustMkdirCLI(t, cliDir)

	writeFileCLI(t, filepath.Join(cliDir, "main.go"), "package main\nfunc main() {}")
	writeFileCLI(t, filepath.Join(cliDir, "go.mod"), "module test-scenario/cli")
	writeFileCLI(t, filepath.Join(cliDir, "install.sh"), "#!/bin/bash")

	v := NewCLIValidator(root, "test-scenario", io.Discard)
	result := v.Validate()

	if !result.Result.Success {
		t.Errorf("Validate() failed: %v", result.Result.Error)
	}
	if result.Approach != CLIApproachCrossPlatform {
		t.Errorf("expected cross-platform approach, got %s", result.Approach)
	}
}

func TestIsTextFile_Text(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "script.sh")
	writeFileCLI(t, path, "#!/bin/bash\necho hello world\n")

	if !isTextFile(path) {
		t.Error("expected isTextFile to return true for shell script")
	}
}

func TestIsTextFile_Binary(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "binary")

	// Write content with null bytes
	content := []byte{0x7f, 0x45, 0x4c, 0x46, 0x00, 0x01, 0x02}
	if err := os.WriteFile(path, content, 0o755); err != nil {
		t.Fatalf("failed to write binary file: %v", err)
	}

	if isTextFile(path) {
		t.Error("expected isTextFile to return false for binary file")
	}
}

func TestIsTextFile_NonExistent(t *testing.T) {
	if isTextFile("/nonexistent/path") {
		t.Error("expected isTextFile to return false for non-existent file")
	}
}

// Test helpers

func mustMkdirCLI(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func writeFileCLI(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if dir != "." {
		mustMkdirCLI(t, dir)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func writeExecutableCLI(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if dir != "." {
		mustMkdirCLI(t, dir)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("failed to write executable %s: %v", path, err)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
