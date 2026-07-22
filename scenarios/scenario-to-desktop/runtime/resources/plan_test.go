package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
		if file == name+".build.json" {
			body = []byte(`{"resource":"demo","artifact":"` + name + `","os":"` + artifactOS(runtimeOS()) + `","arch":"` + runtime.GOARCH + `"}`)
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

func TestLoadRejectsMismatchedBuildMetadata(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "demo")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	name := "resource-demo_" + artifactOS(runtimeOS()) + "_" + runtime.GOARCH
	files := []string{name, name + ".manifest.json", name + ".build.json"}
	artifacts := make([]Artifact, 0, len(files))
	for _, file := range files {
		body := []byte("binary")
		if file == name+".manifest.json" {
			body = []byte(`{"name":"demo"}`)
		}
		if file == name+".build.json" {
			body = []byte(`{"resource":"demo","artifact":"` + name + `","os":"windows","arch":"amd64"}`)
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
	if _, err := Load(root); err == nil {
		t.Fatal("expected mismatched metadata to fail")
	}
}

func TestLoadReturnsActionableDockerPreflight(t *testing.T) {
	originalFindDocker := findDocker
	findDocker = func(string) (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() { findDocker = originalFindDocker })
	root := t.TempDir()
	plan := Plan{SchemaVersion: "v2", Resources: []Item{{
		RequestedResource: "redis",
		Resource:          "redis",
		OS:                runtimeOS(),
		Architecture:      runtime.GOARCH,
		Mode:              "docker-desktop",
		Support:           "conditional",
		Requires:          []string{"docker-desktop"},
		Limitations:       []string{"Docker must be running"},
	}}}
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "resource-deployment-plan.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "Docker Desktop or Docker Engine") {
		t.Fatalf("expected actionable Docker preflight error, got %v", err)
	}
}

func TestLoadRejectsUnknownModeBeforeHostSelection(t *testing.T) {
	root := t.TempDir()
	plan := Plan{SchemaVersion: "v2", Resources: []Item{{RequestedResource: "demo", Resource: "demo", OS: "windows", Architecture: "arm64", Mode: "server-mode", Support: "supported"}}}
	data, _ := json.Marshal(plan)
	if err := os.WriteFile(filepath.Join(root, "resource-deployment-plan.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "unknown deployment mode") {
		t.Fatalf("expected unknown mode to fail, got %v", err)
	}
}

func TestLoadValidatesSeparatelyPinnedBundledService(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "demo")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	controller := "resource-demo_" + artifactOS(runtimeOS()) + "_" + runtime.GOARCH
	files := []string{controller, controller + ".manifest.json", controller + ".build.json"}
	artifacts := make([]Artifact, 0, len(files))
	for _, file := range files {
		body := []byte("controller")
		if file == controller+".manifest.json" {
			body = []byte(`{"name":"demo"}`)
		}
		if file == controller+".build.json" {
			body = []byte(`{"resource":"demo","artifact":"` + controller + `","os":"` + artifactOS(runtimeOS()) + `","arch":"` + runtime.GOARCH + `"}`)
		}
		if err := os.WriteFile(filepath.Join(resourceDir, file), body, 0o755); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		artifacts = append(artifacts, Artifact{Name: file, SHA256: hex.EncodeToString(sum[:])})
	}
	serverName, serverBody := "server-"+runtime.GOOS+"-"+runtime.GOARCH, []byte("server")
	if err := os.WriteFile(filepath.Join(resourceDir, serverName), serverBody, 0o755); err != nil {
		t.Fatal(err)
	}
	serverSum := sha256.Sum256(serverBody)
	plan := Plan{SchemaVersion: "v2", Resources: []Item{{
		RequestedResource: "demo", Resource: "demo", OS: runtimeOS(), Architecture: runtime.GOARCH,
		Mode: "bundled-service", Support: "supported", Artifact: controller, Files: artifacts,
		Service: &Service{Artifact: serverName, Version: "1.0.0", SHA256: hex.EncodeToString(serverSum[:]), Files: []Artifact{{Name: serverName, SHA256: hex.EncodeToString(serverSum[:])}}},
	}}}
	data, _ := json.Marshal(plan)
	if err := os.WriteFile(filepath.Join(root, "resource-deployment-plan.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, serverName), []byte("tampered"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "bundled service artifact hash mismatch") {
		t.Fatalf("Load after server tamper = %v", err)
	}
}
