package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/resources/kopia/cli/internal/version"

	resourceapp "github.com/vrooli/vrooli/resources/kopia/cli/internal/app"
)

func TestNewAppConfiguresResourceApp(t *testing.T) {
	app, err := resourceapp.New(buildFingerprint, buildTimestamp, buildSourceRoot)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if app == nil || app.CLI == nil {
		t.Fatal("New() returned nil app or CLI")
	}
	if app.StaleChecker == nil {
		t.Fatal("New() returned nil stale checker")
	}
}

// TestManifestReferencesPinnedVersion enforces the single-source-of-truth rule:
// the manifest's install block must reference the same kopia version pinned in
// the Go code, so the literal cannot drift between them.
func TestManifestReferencesPinnedVersion(t *testing.T) {
	data, err := os.ReadFile("../resource.json")
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if !strings.Contains(string(data), version.Pinned) {
		t.Fatalf("resource.json does not reference pinned kopia version %q", version.Pinned)
	}
}

// TestManifestContract validates the resource manifest's load-bearing fields.
func TestManifestContract(t *testing.T) {
	data, err := os.ReadFile("../resource.json")
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var m struct {
		Name   string `json:"name"`
		Driver string `json:"driver"`
		Binary string `json:"binary"`
		CLI    struct {
			Command string `json:"command"`
		} `json:"cli"`
		Orchestration struct {
			Dependencies []string `json:"dependencies"`
		} `json:"orchestration"`
		Credentials struct {
			Descriptors []struct {
				Env string `json:"env"`
			} `json:"descriptors"`
		} `json:"credentials"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if m.Name != "kopia" {
		t.Errorf("name = %q, want kopia", m.Name)
	}
	if m.Driver != "external-cli" {
		t.Errorf("driver = %q, want external-cli", m.Driver)
	}
	if m.Binary != "kopia" {
		t.Errorf("binary = %q, want kopia", m.Binary)
	}
	if m.CLI.Command != "resource-kopia" {
		t.Errorf("cli.command = %q, want resource-kopia", m.CLI.Command)
	}
	if contains(m.Orchestration.Dependencies, "vault") {
		t.Errorf("orchestration.dependencies must not include vault, got %v", m.Orchestration.Dependencies)
	}
	var envNames []string
	for _, e := range m.Credentials.Descriptors {
		envNames = append(envNames, e.Env)
	}
	if contains(envNames, "KOPIA_PASSWORD") {
		t.Errorf("credentials.descriptors must not declare the dynamic repository passphrase, got %v", envNames)
	}
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
