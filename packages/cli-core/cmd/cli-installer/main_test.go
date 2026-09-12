package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestCLIInstallerInstallsBinaryManifestAndMetadata(t *testing.T) {
	moduleDir := filepath.Join(t.TempDir(), "demo-cli")
	installDir := t.TempDir()
	manifestPath := filepath.Join(moduleDir, "resource.json")

	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("mkdir module: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte("module example.com/demo-cli\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "main.go"), []byte(`package main

import "fmt"

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func main() { fmt.Println("demo") }
`), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte("{\"name\":\"demo\"}\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	cmd := exec.Command("go", "run", ".", "--module", moduleDir, "--manifest", manifestPath, "--install-dir", installDir, "--name", "demo-cli")
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run cli-installer failed: %v\n%s", err, output)
	}

	binaryPath := filepath.Join(installDir, "demo-cli")
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("installed binary missing: %v", err)
	}
	installedManifest := cliutil.InstalledManifestPath(binaryPath)
	data, err := os.ReadFile(installedManifest)
	if err != nil {
		t.Fatalf("read installed manifest: %v", err)
	}
	if string(data) != "{\"name\":\"demo\"}\n" {
		t.Fatalf("installed manifest = %q", string(data))
	}
	metaPath := cliutil.InstalledBuildMetadataPath(binaryPath)
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read build metadata: %v", err)
	}
	var meta struct {
		BinaryName string `json:"binary_name"`
		ModulePath string `json:"module_path"`
	}
	if err := json.Unmarshal(metaData, &meta); err != nil {
		t.Fatalf("unmarshal build metadata: %v", err)
	}
	if meta.BinaryName != "demo-cli" {
		t.Fatalf("binary_name = %q, want demo-cli", meta.BinaryName)
	}
	if meta.ModulePath == "" {
		t.Fatal("module_path should not be empty")
	}
}

func TestBuildFreshnessSpecUsesDeclaredContextAndInputs(t *testing.T) {
	modulePath := filepath.Join(t.TempDir(), "scenario", "cli")
	contextRoot := filepath.Join(filepath.Dir(modulePath))

	spec, err := buildFreshnessSpec(modulePath, contextRoot, []string{"cli/**", ".vrooli/service.json"}, "demo")
	if err != nil {
		t.Fatalf("buildFreshnessSpec: %v", err)
	}
	if spec.SourceRoot != modulePath {
		t.Fatalf("SourceRoot = %q, want %q", spec.SourceRoot, modulePath)
	}
	if spec.ContextRoot != contextRoot {
		t.Fatalf("ContextRoot = %q, want %q", spec.ContextRoot, contextRoot)
	}
	if len(spec.Inputs) != 2 {
		t.Fatalf("Inputs = %v", spec.Inputs)
	}
	if len(spec.SkipFiles) != 1 || spec.SkipFiles[0] != "demo" {
		t.Fatalf("SkipFiles = %v", spec.SkipFiles)
	}
}

func TestBuildFreshnessSpecCanonicalizesRelativePaths(t *testing.T) {
	root := t.TempDir()
	modulePath := filepath.Join(root, "scenario", "cli")
	contextRoot := filepath.Join(root, "scenario")

	if err := os.MkdirAll(modulePath, 0o755); err != nil {
		t.Fatalf("mkdir module path: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	spec, err := buildFreshnessSpec(filepath.Join("scenario", "cli"), "scenario", []string{"cli/**"}, "demo")
	if err != nil {
		t.Fatalf("buildFreshnessSpec: %v", err)
	}
	if spec.SourceRoot != modulePath {
		t.Fatalf("SourceRoot = %q, want %q", spec.SourceRoot, modulePath)
	}
	if spec.ContextRoot != contextRoot {
		t.Fatalf("ContextRoot = %q, want %q", spec.ContextRoot, contextRoot)
	}
}
