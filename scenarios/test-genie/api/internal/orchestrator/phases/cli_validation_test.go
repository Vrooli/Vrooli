package phases

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectCLIApproach(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, cliDir, scenarioName string)
		wantApproach CLIApproach
	}{
		{
			name: "cross-platform with main.go and go.mod",
			setup: func(t *testing.T, cliDir, scenarioName string) {
				writeTestFile(t, filepath.Join(cliDir, "main.go"), "package main\nfunc main() {}\n")
				writeTestFile(t, filepath.Join(cliDir, "go.mod"), "module test-cli\n\ngo 1.21\n")
				writeTestFile(t, filepath.Join(cliDir, "install.sh"), "#!/bin/bash\necho install\n")
			},
			wantApproach: CLIApproachCrossPlatform,
		},
		{
			name: "legacy with bash script",
			setup: func(t *testing.T, cliDir, scenarioName string) {
				writeTestFile(t, filepath.Join(cliDir, scenarioName), "#!/bin/bash\necho cli\n")
				writeTestFile(t, filepath.Join(cliDir, "install.sh"), "#!/bin/bash\necho install\n")
			},
			wantApproach: CLIApproachLegacy,
		},
		{
			name: "cross-platform with binary present (build artifact)",
			setup: func(t *testing.T, cliDir, scenarioName string) {
				writeTestFile(t, filepath.Join(cliDir, "main.go"), "package main\nfunc main() {}\n")
				writeTestFile(t, filepath.Join(cliDir, "go.mod"), "module test-cli\n\ngo 1.21\n")
				writeTestFile(t, filepath.Join(cliDir, "install.sh"), "#!/bin/bash\necho install\n")
				// Write a fake binary (contains null bytes)
				writeTestBinary(t, filepath.Join(cliDir, scenarioName))
			},
			wantApproach: CLIApproachCrossPlatform,
		},
		{
			name: "unknown with only install.sh",
			setup: func(t *testing.T, cliDir, scenarioName string) {
				writeTestFile(t, filepath.Join(cliDir, "install.sh"), "#!/bin/bash\necho install\n")
			},
			wantApproach: CLIApproachUnknown,
		},
		{
			name: "unknown with main.go but no go.mod",
			setup: func(t *testing.T, cliDir, scenarioName string) {
				writeTestFile(t, filepath.Join(cliDir, "main.go"), "package main\nfunc main() {}\n")
				writeTestFile(t, filepath.Join(cliDir, "install.sh"), "#!/bin/bash\necho install\n")
			},
			wantApproach: CLIApproachUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenarioDir := t.TempDir()
			scenarioName := "test-scenario"
			cliDir := filepath.Join(scenarioDir, "cli")
			mustMkdir(t, cliDir)
			tt.setup(t, cliDir, scenarioName)

			got := detectCLIApproach(scenarioDir, scenarioName)
			if got != tt.wantApproach {
				t.Errorf("detectCLIApproach() = %v, want %v", got, tt.wantApproach)
			}
		})
	}
}

func TestValidateCLIStructure_CrossPlatform(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := "test-scenario"
	cliDir := filepath.Join(scenarioDir, "cli")
	mustMkdir(t, cliDir)

	// Setup cross-platform CLI structure
	writeTestFile(t, filepath.Join(cliDir, "main.go"), "package main\nfunc main() {}\n")
	writeTestFile(t, filepath.Join(cliDir, "go.mod"), "module test-cli\n\ngo 1.21\n")
	writeTestFile(t, filepath.Join(cliDir, "install.sh"), "#!/bin/bash\necho install\n")

	result := validateCLIStructure(scenarioDir, scenarioName, io.Discard)

	if !result.Valid {
		t.Fatalf("expected valid, got error: %v", result.Error)
	}
	if result.Approach != CLIApproachCrossPlatform {
		t.Errorf("expected cross-platform approach, got %v", result.Approach)
	}
	if len(result.Observations) == 0 {
		t.Error("expected observations to be recorded")
	}

	// Check for info observation about missing install.ps1
	hasInfoObs := false
	for _, obs := range result.Observations {
		if obs.Prefix == "INFO" {
			hasInfoObs = true
			break
		}
	}
	if !hasInfoObs {
		t.Error("expected INFO observation about Windows support")
	}
}

func TestValidateCLIStructure_CrossPlatformWithWindows(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := "test-scenario"
	cliDir := filepath.Join(scenarioDir, "cli")
	mustMkdir(t, cliDir)

	// Setup cross-platform CLI structure with Windows support
	writeTestFile(t, filepath.Join(cliDir, "main.go"), "package main\nfunc main() {}\n")
	writeTestFile(t, filepath.Join(cliDir, "go.mod"), "module test-cli\n\ngo 1.21\n")
	writeTestFile(t, filepath.Join(cliDir, "install.sh"), "#!/bin/bash\necho install\n")
	writeTestFile(t, filepath.Join(cliDir, "install.ps1"), "Write-Host 'install'\n")

	result := validateCLIStructure(scenarioDir, scenarioName, io.Discard)

	if !result.Valid {
		t.Fatalf("expected valid, got error: %v", result.Error)
	}

	// Check for success observation about Windows support
	hasWindowsObs := false
	for _, obs := range result.Observations {
		if obs.Prefix == "SUCCESS" && obs.Text == "Cross-platform CLI with Windows support" {
			hasWindowsObs = true
			break
		}
	}
	if !hasWindowsObs {
		t.Error("expected SUCCESS observation about Windows support")
	}
}

func TestValidateCLIStructure_Legacy(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := "test-scenario"
	cliDir := filepath.Join(scenarioDir, "cli")
	mustMkdir(t, cliDir)

	// Setup legacy CLI structure
	cliScript := filepath.Join(cliDir, scenarioName)
	writeTestExecutable(t, cliScript, "#!/bin/bash\necho cli\n")
	writeTestFile(t, filepath.Join(cliDir, "install.sh"), "#!/bin/bash\necho install\n")

	result := validateCLIStructure(scenarioDir, scenarioName, io.Discard)

	if !result.Valid {
		t.Fatalf("expected valid, got error: %v", result.Error)
	}
	if result.Approach != CLIApproachLegacy {
		t.Errorf("expected legacy approach, got %v", result.Approach)
	}

	// Check for info observation about cross-platform alternative
	hasInfoObs := false
	for _, obs := range result.Observations {
		if obs.Prefix == "INFO" && obs.Text != "" {
			hasInfoObs = true
			break
		}
	}
	if !hasInfoObs {
		t.Error("expected INFO observation about cross-platform alternative")
	}
}

func TestValidateCLIStructure_LegacyNotExecutable(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := "test-scenario"
	cliDir := filepath.Join(scenarioDir, "cli")
	mustMkdir(t, cliDir)

	// Setup legacy CLI structure without executable permission
	cliScript := filepath.Join(cliDir, scenarioName)
	writeTestFile(t, cliScript, "#!/bin/bash\necho cli\n") // Not executable
	writeTestFile(t, filepath.Join(cliDir, "install.sh"), "#!/bin/bash\necho install\n")

	result := validateCLIStructure(scenarioDir, scenarioName, io.Discard)

	if !result.Valid {
		t.Fatalf("expected valid (with warning), got error: %v", result.Error)
	}

	// Check for warning observation about executable permission
	hasWarningObs := false
	for _, obs := range result.Observations {
		if obs.Prefix == "WARNING" {
			hasWarningObs = true
			break
		}
	}
	if !hasWarningObs {
		t.Error("expected WARNING observation about executable permission")
	}
}

func TestValidateCLIStructure_Unknown(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := "test-scenario"
	cliDir := filepath.Join(scenarioDir, "cli")
	mustMkdir(t, cliDir)

	// Setup incomplete CLI structure
	writeTestFile(t, filepath.Join(cliDir, "install.sh"), "#!/bin/bash\necho install\n")

	result := validateCLIStructure(scenarioDir, scenarioName, io.Discard)

	if result.Valid {
		t.Fatal("expected invalid for incomplete CLI structure")
	}
	if result.Approach != CLIApproachUnknown {
		t.Errorf("expected unknown approach, got %v", result.Approach)
	}
	if result.Remediation == "" {
		t.Error("expected remediation message")
	}
}

func TestValidateCLIStructure_MissingCLIDirectory(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := "test-scenario"
	// Don't create cli/ directory

	result := validateCLIStructure(scenarioDir, scenarioName, io.Discard)

	if result.Valid {
		t.Fatal("expected invalid for missing cli directory")
	}
	if result.Error == nil {
		t.Error("expected error for missing cli directory")
	}
}

func TestValidateCLIStructure_CrossPlatformMissingGoMod(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := "test-scenario"
	cliDir := filepath.Join(scenarioDir, "cli")
	mustMkdir(t, cliDir)

	// main.go without go.mod should be unknown
	writeTestFile(t, filepath.Join(cliDir, "main.go"), "package main\nfunc main() {}\n")
	writeTestFile(t, filepath.Join(cliDir, "install.sh"), "#!/bin/bash\necho install\n")

	result := validateCLIStructure(scenarioDir, scenarioName, io.Discard)

	if result.Valid {
		t.Fatal("expected invalid for incomplete cross-platform structure")
	}
	if result.Approach != CLIApproachUnknown {
		t.Errorf("expected unknown approach, got %v", result.Approach)
	}
}

func TestIsTextFile(t *testing.T) {
	dir := t.TempDir()

	// Text file
	textPath := filepath.Join(dir, "text.sh")
	writeTestFile(t, textPath, "#!/bin/bash\necho hello\n")
	if !isTextFile(textPath) {
		t.Error("expected text file to be detected as text")
	}

	// Binary file (contains null bytes)
	binaryPath := filepath.Join(dir, "binary")
	writeTestBinary(t, binaryPath)
	if isTextFile(binaryPath) {
		t.Error("expected binary file to be detected as binary")
	}

	// Non-existent file
	if isTextFile(filepath.Join(dir, "nonexistent")) {
		t.Error("expected non-existent file to return false")
	}
}

func TestCLIApproachString(t *testing.T) {
	tests := []struct {
		approach CLIApproach
		want     string
	}{
		{CLIApproachUnknown, "unknown"},
		{CLIApproachLegacy, "legacy"},
		{CLIApproachCrossPlatform, "cross-platform"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.approach.String(); got != tt.want {
				t.Errorf("CLIApproach.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test helpers

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func writeTestExecutable(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("failed to write executable %s: %v", path, err)
	}
}

func writeTestBinary(t *testing.T, path string) {
	t.Helper()
	// Create a file with null bytes to simulate a binary
	content := []byte{0x7f, 'E', 'L', 'F', 0x00, 0x00, 0x00, 0x00}
	if err := os.WriteFile(path, content, 0o755); err != nil {
		t.Fatalf("failed to write binary %s: %v", path, err)
	}
}
