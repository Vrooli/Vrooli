package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadValidatesBundledClientArtifacts(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "demo")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	name := "resource-demo_" + runtime.GOOS + "_" + runtime.GOARCH
	if runtime.GOOS == "darwin" {
		name = "resource-demo_darwin_" + runtime.GOARCH
	}
	files := []string{name, name + ".manifest.json", name + ".build.json"}
	artifacts := make([]Artifact, 0, 3)
	for _, file := range files {
		body := []byte(`{"name":"demo"}`)
		if file == name {
			body = []byte("binary")
		}
		if err := os.WriteFile(filepath.Join(resourceDir, file), body, 0o755); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		artifacts = append(artifacts, Artifact{Name: file, SHA256: hex.EncodeToString(sum[:])})
	}
	plan := Plan{SchemaVersion: "v2", Resources: []Item{{RequestedResource: "demo", Resource: "demo", OS: runtimeOS(), Architecture: runtime.GOARCH, Mode: "bundled-client", Support: "supported", Artifact: name, Files: artifacts}}}
	data, _ := json.Marshal(plan)
	if err := os.WriteFile(filepath.Join(root, "resource-deployment-plan.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, name), []byte("tampered"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("expected artifact tampering to fail")
	}
}
