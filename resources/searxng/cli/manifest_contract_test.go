package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManifestDeclaresOnlyTheSupportedDockerContract(t *testing.T) {
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
				Kind            string   `json:"kind"`
				FreshnessInputs []string `json:"freshness_inputs"`
			} `json:"source_build"`
		} `json:"cli"`
		Runtime struct {
			Image   string `json:"image"`
			Volumes []struct {
				Source string `json:"source"`
				Target string `json:"target"`
			} `json:"volumes"`
		} `json:"runtime"`
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
	if manifest.Template != "docker-service" || manifest.Driver != "docker-service" {
		t.Fatalf("template/driver = %q/%q", manifest.Template, manifest.Driver)
	}
	if strings.Contains(strings.ToLower(manifest.Description), "shell") {
		t.Fatalf("description retains shell-era claim: %q", manifest.Description)
	}
	if manifest.Runtime.Image == "" || len(manifest.Runtime.Volumes) != 2 {
		t.Fatalf("runtime = %#v", manifest.Runtime)
	}
	wantMounts := map[string]string{"${RESOURCE_CONFIG_DIR}": "/etc/searxng", "${RESOURCE_DATA_DIR}": "/var/cache/searxng"}
	for _, mount := range manifest.Runtime.Volumes {
		if wantMounts[mount.Source] != mount.Target {
			t.Fatalf("unexpected mount %#v", mount)
		}
		delete(wantMounts, mount.Source)
	}
	if len(wantMounts) != 0 {
		t.Fatalf("missing mounts: %#v", wantMounts)
	}
	if len(manifest.Health) != 1 || !strings.HasSuffix(manifest.Health[0].Target, "/stats") {
		t.Fatalf("health = %#v", manifest.Health)
	}
	if len(manifest.HostTools) != 1 || manifest.HostTools[0].Name != "docker" || !manifest.HostTools[0].Required {
		t.Fatalf("host tools = %#v, want required Docker preflight", manifest.HostTools)
	}
	if manifest.CLI.SourceBuild.Kind != "go_module" {
		t.Fatalf("cli.source_build.kind = %q, want go_module", manifest.CLI.SourceBuild.Kind)
	}
	if len(manifest.CLI.SourceBuild.FreshnessInputs) == 0 {
		t.Fatal("cli.source_build.freshness_inputs must not be empty")
	}
}

func TestDocumentationAndCapabilityContractContainNoLegacyRuntime(t *testing.T) {
	root := ".."
	operations, err := os.ReadFile(filepath.Join(root, "docs", "OPERATIONS.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Docker", "RESOURCE_CONFIG_DIR", "/etc/searxng", "RESOURCE_DATA_DIR", "/var/cache/searxng", "engine-health"} {
		if !strings.Contains(string(operations), want) {
			t.Fatalf("operations document missing %q", want)
		}
	}
	capabilities, err := os.ReadFile(filepath.Join(root, "config", "capabilities.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"redis", "compose", "docker-compose"} {
		if strings.Contains(strings.ToLower(string(capabilities)), forbidden) {
			t.Fatalf("capability contract retains obsolete %q claim", forbidden)
		}
	}
	for _, legacy := range []string{"lib/common.sh", "test/run-tests.sh", "docker/docker-compose.yml", "cli/install.sh", "cli/install.ps1", "config/settings.yml.template"} {
		if _, err := os.Stat(filepath.Join(root, legacy)); !os.IsNotExist(err) {
			t.Fatalf("legacy path %q remains or could not be checked: %v", legacy, err)
		}
	}
}
