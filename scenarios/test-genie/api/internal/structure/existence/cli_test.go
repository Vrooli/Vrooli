package existence

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectCLIApproach_GoModule(t *testing.T) {
	root := t.TempDir()
	writeCLIManifest(t, root, `{
  "service": {"name": "test-scenario"},
  "cli": {
    "enabled": true,
    "command": "test-scenario",
    "adapter": {"kind": "go_module", "module_dir": "cli"},
    "install": [{"kind": "command", "run": "bash ./cli/install.sh"}],
    "invoke": {"kind": "installed_command", "command": "test-scenario"}
  }
}`)

	approach := DetectCLIApproach(mustLoadManifest(t, root))
	if approach != CLIApproachGoModule {
		t.Errorf("expected CLIApproachGoModule, got %s", approach)
	}
}

func TestDetectCLIApproach_ShellScript(t *testing.T) {
	root := t.TempDir()
	writeCLIManifest(t, root, `{
  "service": {"name": "test-scenario"},
  "cli": {
    "enabled": true,
    "command": "test-scenario",
    "adapter": {
      "kind": "shell_script",
      "script_path": "cli/test-scenario",
      "install_script": "cli/install.sh"
    },
    "install": [{"kind": "command", "run": "bash ./cli/install.sh"}],
    "invoke": {"kind": "installed_command", "command": "test-scenario"}
  }
}`)

	approach := DetectCLIApproach(mustLoadManifest(t, root))
	if approach != CLIApproachShellScript {
		t.Errorf("expected CLIApproachShellScript, got %s", approach)
	}
}

func TestDetectCLIApproach_UnknownWithoutCLIManifest(t *testing.T) {
	root := t.TempDir()
	writeCLIManifest(t, root, `{"service":{"name":"test-scenario"}}`)

	approach := DetectCLIApproach(mustLoadManifest(t, root))
	if approach != CLIApproachUnknown {
		t.Errorf("expected CLIApproachUnknown, got %s", approach)
	}
}

func TestCLIApproach_String(t *testing.T) {
	tests := []struct {
		approach CLIApproach
		expected string
	}{
		{CLIApproachGoModule, "go_module"},
		{CLIApproachShellScript, "shell_script"},
		{CLIApproachUnknown, "unknown"},
	}

	for _, tc := range tests {
		if got := tc.approach.String(); got != tc.expected {
			t.Errorf("%v.String() = %q, want %q", tc.approach, got, tc.expected)
		}
	}
}

func TestValidateCLI_GoModuleValid(t *testing.T) {
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	mustMkdirCLI(t, cliDir)
	writeCLIManifest(t, root, `{
  "service": {"name": "test-scenario"},
  "cli": {
    "enabled": true,
    "command": "test-scenario",
    "adapter": {"kind": "go_module", "module_dir": "cli"},
    "install": [
      {"os": ["linux", "darwin"], "kind": "command", "run": "bash ./cli/install.sh"},
      {"os": ["windows"], "kind": "command", "run": "powershell -File .\\cli\\install.ps1"}
    ],
    "invoke": {"kind": "installed_command", "command": "test-scenario"}
  }
}`)
	writeFileCLI(t, filepath.Join(cliDir, "app.go"), "package main\nfunc main() {}")
	writeFileCLI(t, filepath.Join(cliDir, "go.mod"), "module test-scenario/cli")

	result := ValidateCLI(root, "test-scenario", io.Discard)
	if !result.Result.Success {
		t.Fatalf("expected success, got error: %v", result.Result.Error)
	}
	if result.Approach != CLIApproachGoModule {
		t.Errorf("expected go_module approach, got %s", result.Approach)
	}
}

func TestValidateCLI_GoModuleMissingGoMod(t *testing.T) {
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	mustMkdirCLI(t, cliDir)
	writeCLIManifest(t, root, `{
  "service": {"name": "test-scenario"},
  "cli": {
    "enabled": true,
    "command": "test-scenario",
    "adapter": {"kind": "go_module", "module_dir": "cli"},
    "install": [{"kind": "command", "run": "bash ./cli/install.sh"}],
    "invoke": {"kind": "installed_command", "command": "test-scenario"}
  }
}`)
	writeFileCLI(t, filepath.Join(cliDir, "app.go"), "package main\nfunc main() {}")

	result := ValidateCLI(root, "test-scenario", io.Discard)
	if result.Result.Success {
		t.Fatal("expected failure when go.mod missing")
	}
	if result.Approach != CLIApproachGoModule {
		t.Errorf("expected go_module approach, got %s", result.Approach)
	}
}

func TestValidateCLI_ShellScriptValid(t *testing.T) {
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	mustMkdirCLI(t, cliDir)
	writeCLIManifest(t, root, `{
  "service": {"name": "test-scenario"},
  "cli": {
    "enabled": true,
    "command": "test-scenario",
    "adapter": {
      "kind": "shell_script",
      "script_path": "cli/test-scenario",
      "install_script": "cli/install.sh"
    },
    "install": [{"kind": "command", "run": "bash ./cli/install.sh"}],
    "invoke": {"kind": "installed_command", "command": "test-scenario"}
  }
}`)
	writeExecutableCLI(t, filepath.Join(cliDir, "test-scenario"), "#!/bin/bash\necho cli")
	writeFileCLI(t, filepath.Join(cliDir, "install.sh"), "#!/bin/bash\necho install")

	result := ValidateCLI(root, "test-scenario", io.Discard)
	if !result.Result.Success {
		t.Fatalf("expected success, got error: %v", result.Result.Error)
	}
	if result.Approach != CLIApproachShellScript {
		t.Errorf("expected shell_script approach, got %s", result.Approach)
	}
}

func TestValidateCLI_ShellScriptNonExecutable(t *testing.T) {
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	mustMkdirCLI(t, cliDir)
	writeCLIManifest(t, root, `{
  "service": {"name": "test-scenario"},
  "cli": {
    "enabled": true,
    "command": "test-scenario",
    "adapter": {
      "kind": "shell_script",
      "script_path": "cli/test-scenario",
      "install_script": "cli/install.sh"
    },
    "install": [{"kind": "command", "run": "bash ./cli/install.sh"}],
    "invoke": {"kind": "installed_command", "command": "test-scenario"}
  }
}`)
	writeFileCLI(t, filepath.Join(cliDir, "test-scenario"), "#!/bin/bash\necho cli")
	writeFileCLI(t, filepath.Join(cliDir, "install.sh"), "#!/bin/bash\necho install")

	result := ValidateCLI(root, "test-scenario", io.Discard)
	if !result.Result.Success {
		t.Fatalf("expected success (with warning), got error: %v", result.Result.Error)
	}

	hasWarning := false
	for _, obs := range result.Result.Observations {
		if obs.Type == ObservationWarning {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected warning about non-executable shell script")
	}
}

func TestValidateCLI_MissingManifest(t *testing.T) {
	root := t.TempDir()

	result := ValidateCLI(root, "test-scenario", io.Discard)
	if result.Result.Success {
		t.Fatal("expected failure when service manifest missing")
	}
	if result.Approach != CLIApproachUnknown {
		t.Errorf("expected unknown approach, got %s", result.Approach)
	}
}

func TestValidateCLI_InvalidCommonContract(t *testing.T) {
	root := t.TempDir()
	mustMkdirCLI(t, filepath.Join(root, "cli"))
	writeCLIManifest(t, root, `{
  "service": {"name": "test-scenario"},
  "cli": {
    "enabled": true,
    "command": "test-scenario",
    "adapter": {"kind": "go_module", "module_dir": "cli"},
    "install": [{"kind": "command", "run": "bash ./cli/install.sh"}],
    "invoke": {"kind": "installed_command", "command": "other-name"}
  }
}`)
	writeFileCLI(t, filepath.Join(root, "cli", "app.go"), "package main\nfunc main() {}")
	writeFileCLI(t, filepath.Join(root, "cli", "go.mod"), "module test-scenario/cli")

	result := ValidateCLI(root, "test-scenario", io.Discard)
	if result.Result.Success {
		t.Fatal("expected failure for mismatched invoke.command")
	}
	if result.Result.Remediation == "" {
		t.Error("expected remediation guidance")
	}
}

func TestCLIValidator_Interface(t *testing.T) {
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	mustMkdirCLI(t, cliDir)
	writeCLIManifest(t, root, `{
  "service": {"name": "test-scenario"},
  "cli": {
    "enabled": true,
    "command": "test-scenario",
    "adapter": {"kind": "go_module", "module_dir": "cli"},
    "install": [{"kind": "command", "run": "bash ./cli/install.sh"}],
    "invoke": {"kind": "installed_command", "command": "test-scenario"}
  }
}`)
	writeFileCLI(t, filepath.Join(cliDir, "app.go"), "package main\nfunc main() {}")
	writeFileCLI(t, filepath.Join(cliDir, "go.mod"), "module test-scenario/cli")

	v := NewCLIValidator(root, "test-scenario", io.Discard)
	result := v.Validate()

	if !result.Result.Success {
		t.Errorf("Validate() failed: %v", result.Result.Error)
	}
	if result.Approach != CLIApproachGoModule {
		t.Errorf("expected go_module approach, got %s", result.Approach)
	}
}

func writeCLIManifest(t *testing.T, root, content string) {
	t.Helper()
	serviceDir := filepath.Join(root, ".vrooli")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "service.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}
}

func mustLoadManifest(t *testing.T, root string) serviceManifest {
	t.Helper()
	manifest, err := loadServiceManifest(root)
	if err != nil {
		t.Fatalf("load service manifest: %v", err)
	}
	return manifest
}

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
