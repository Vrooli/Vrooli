package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestWriteReleaseChecksumManifestIsDeterministicAndUnsigned(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "resource-vault_linux_amd64"), []byte("controller"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "vault_linux_amd64"), []byte("server"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeReleaseChecksumManifest(dir); err != nil {
		t.Fatalf("write checksum manifest: %v", err)
	}
	manifest, err := os.ReadFile(filepath.Join(dir, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest) == 0 || !strings.Contains(string(manifest), "resource-vault_linux_amd64") || !strings.Contains(string(manifest), "vault_linux_amd64") {
		t.Fatalf("unexpected checksum manifest: %s", manifest)
	}
	if _, err := os.Stat(filepath.Join(dir, "SHA256SUMS.sig")); !os.IsNotExist(err) {
		t.Fatalf("source staging must not create a release signature, stat error: %v", err)
	}
	first, err := os.ReadFile(filepath.Join(dir, "release-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := resourcedeployment.LoadReleaseManifest(dir)
	if err != nil || len(parsed.Artifacts) != 2 {
		t.Fatalf("release manifest = %#v, err=%v", parsed, err)
	}
	if parsed.Artifacts[0].Role != "resource-controller" || parsed.Artifacts[1].Role != "managed-service" {
		t.Fatalf("release roles = %#v", parsed.Artifacts)
	}
	if err := writeReleaseChecksumManifest(dir); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(dir, "release-manifest.json"))
	if err != nil || string(first) != string(second) {
		t.Fatalf("release manifest must be deterministic: %v", err)
	}
}

func TestRunVerifiesGenericReleaseManifestInDevelopmentLocalMode(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "resource-vault_linux_amd64"), []byte("controller"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeReleaseChecksumManifest(dir); err != nil {
		t.Fatal(err)
	}
	if got := run([]string{"--verify-release-manifest", "--release-artifact-root", dir, "--trust-mode", "development-local"}); got != 0 {
		t.Fatalf("development-local verification exit code = %d, want 0", got)
	}
	if got := run([]string{"--verify-release-manifest", "--root", dir, "--release-artifact-root", dir, "--trust-mode", "production"}); got != 1 {
		t.Fatalf("unsigned production verification exit code = %d, want 1", got)
	}
}

func TestResourceArtifactBuildTargetsIncludeOnlyDeclaredBundledPlatforms(t *testing.T) {
	manifest := resourceArtifactManifest{Name: "fixture", Driver: "managed-service"}
	manifest.CLI.Adapter.ModuleDir = "cli"
	manifest.CLI.Distribution.Kind = "prebuilt_artifact"
	manifest.CLI.Distribution.ArtifactName = "resource-fixture_${os}_${arch}"
	manifest.Deployment.Profiles = map[string]resourcedeployment.Profile{
		"desktop": {
			Linux:   &resourcedeployment.Target{Support: "conditional", Mode: "bundled-service", Architectures: []string{"amd64", "arm64"}},
			MacOS:   &resourcedeployment.Target{Support: "unsupported", Mode: "bundled-service", Architectures: []string{"amd64"}},
			Windows: &resourcedeployment.Target{Support: "conditional", Mode: "remote-service", Architectures: []string{"amd64"}},
		},
	}
	targets, err := resourceArtifactBuildTargets(manifest)
	if err != nil {
		t.Fatalf("resourceArtifactBuildTargets: %v", err)
	}
	if len(targets) != 2 || targets[0].Platform.String() != "linux-amd64" || targets[1].Platform.String() != "linux-arm64" {
		t.Fatalf("targets = %#v", targets)
	}
	if targets[0].Artifact != "resource-fixture_linux_amd64" {
		t.Fatalf("artifact = %q", targets[0].Artifact)
	}
}

func TestResourceArtifactBuildTargetsRejectIncompleteBundledContract(t *testing.T) {
	manifest := resourceArtifactManifest{Name: "fixture"}
	manifest.CLI.Distribution.Kind = "prebuilt_artifact"
	manifest.Deployment.Profiles = map[string]resourcedeployment.Profile{
		"desktop": {Linux: &resourcedeployment.Target{Support: "conditional", Mode: "bundled-client", Architectures: []string{"amd64"}}},
	}
	if _, err := resourceArtifactBuildTargets(manifest); err == nil {
		t.Fatal("expected incomplete bundled contract to be rejected")
	}
}
