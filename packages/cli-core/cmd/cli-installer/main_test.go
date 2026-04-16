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
