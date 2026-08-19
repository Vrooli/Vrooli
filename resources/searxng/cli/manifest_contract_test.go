package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManifestDeclaresOnlyTheSupportedManagedContract(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Template    string `json:"template"`
		Driver      string `json:"driver"`
		Description string `json:"description"`
		CLI         struct {
			SourceBuild struct {
				Kind string `json:"kind"`
			} `json:"source_build"`
			Freshness struct {
				Inputs []string `json:"inputs"`
			} `json:"freshness"`
		} `json:"cli"`
		ManagedService struct {
			Artifact struct {
				Layout    string `json:"layout"`
				EntryPath string `json:"entry_path"`
			} `json:"artifact"`
			Acquisition struct {
				Kind string `json:"kind"`
			} `json:"acquisition"`
		} `json:"managed_service"`
		Health []struct {
			Target string `json:"target"`
		} `json:"health_checks"`
		HostTools []struct {
			Name     string `json:"name"`
			Required bool   `json:"required"`
		} `json:"hostTools"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Template != "managed-service" || manifest.Driver != "managed-service" {
		t.Fatalf("template/driver = %q/%q", manifest.Template, manifest.Driver)
	}
	if strings.Contains(strings.ToLower(manifest.Description), "shell") {
		t.Fatalf("description retains shell-era claim: %q", manifest.Description)
	}
	if manifest.ManagedService.Acquisition.Kind != "composed" || manifest.ManagedService.Artifact.Layout != "dir" || manifest.ManagedService.Artifact.EntryPath != "runtime/bin/python" {
		t.Fatalf("managed service artifact/acquisition = %#v", manifest.ManagedService)
	}
	if len(manifest.Health) != 1 || !strings.HasSuffix(manifest.Health[0].Target, "/stats") {
		t.Fatalf("health = %#v", manifest.Health)
	}
	if len(manifest.HostTools) != 0 {
		t.Fatalf("host tools = %#v, want no external runtime requirement", manifest.HostTools)
	}
	if manifest.CLI.SourceBuild.Kind != "go_module" {
		t.Fatalf("cli.source_build.kind = %q, want go_module", manifest.CLI.SourceBuild.Kind)
	}
	if len(manifest.CLI.Freshness.Inputs) == 0 {
		t.Fatal("cli.freshness.inputs must not be empty")
	}
}

func TestDocumentationAndCapabilityContractContainNoLegacyRuntime(t *testing.T) {
	root := ".."
	operations, err := os.ReadFile(filepath.Join(root, "docs", "OPERATIONS.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"managed-service", "RESOURCE_CONFIG_DIR", "RESOURCE_DATA_DIR", "engine-health"} {
		if !strings.Contains(string(operations), want) {
			t.Fatalf("operations document missing %q", want)
		}
	}
	capabilities, err := os.ReadFile(filepath.Join(root, "config", "capabilities.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"redis"} {
		if strings.Contains(strings.ToLower(string(capabilities)), forbidden) {
			t.Fatalf("capability contract retains obsolete %q claim", forbidden)
		}
	}
	for _, legacy := range []string{"lib/common.sh", "test/run-tests.sh", "compose/legacy.yml", "cli/install.sh", "cli/install.ps1", "config/settings.yml.template"} {
		if _, err := os.Stat(filepath.Join(root, legacy)); !os.IsNotExist(err) {
			t.Fatalf("legacy path %q remains or could not be checked: %v", legacy, err)
		}
	}
}
