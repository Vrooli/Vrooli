package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"scenario-to-cloud/bundle"
	"scenario-to-cloud/domain"
)

func TestResolveScenarioVersionFallback(t *testing.T) {
	scenarioID := "demo-scenario"
	repoRoot := setupFreshnessFixture(t, scenarioID, "2.1.0", "3.2.1")

	version, source, err := resolveScenarioVersion(repoRoot, scenarioID)
	if err != nil {
		t.Fatalf("resolveScenarioVersion returned error: %v", err)
	}
	if version != "2.1.0" {
		t.Fatalf("expected service.json version 2.1.0, got %q", version)
	}
	if source != versionSourceService {
		t.Fatalf("expected source %q, got %q", versionSourceService, source)
	}

	// Remove service.json to force UI package fallback.
	if err := os.Remove(filepath.Join(repoRoot, "scenarios", scenarioID, ".vrooli", "service.json")); err != nil {
		t.Fatalf("remove service.json: %v", err)
	}

	version, source, err = resolveScenarioVersion(repoRoot, scenarioID)
	if err != nil {
		t.Fatalf("resolveScenarioVersion returned error: %v", err)
	}
	if version != "3.2.1" {
		t.Fatalf("expected package.json version 3.2.1, got %q", version)
	}
	if source != versionSourceUI {
		t.Fatalf("expected source %q, got %q", versionSourceUI, source)
	}

	// Remove ui/package.json to force default.
	if err := os.Remove(filepath.Join(repoRoot, "scenarios", scenarioID, "ui", "package.json")); err != nil {
		t.Fatalf("remove package.json: %v", err)
	}

	version, source, err = resolveScenarioVersion(repoRoot, scenarioID)
	if err != nil {
		t.Fatalf("resolveScenarioVersion returned error: %v", err)
	}
	if version != defaultScenarioVersion {
		t.Fatalf("expected default version %q, got %q", defaultScenarioVersion, version)
	}
	if source != versionSourceDefault {
		t.Fatalf("expected source %q, got %q", versionSourceDefault, source)
	}
}

func TestEvaluateFreshnessOutdatedByVersion(t *testing.T) {
	scenarioID := "demo-scenario"
	repoRoot := setupFreshnessFixture(t, scenarioID, "1.4.0", "1.4.0")
	manifest := testManifest(scenarioID)
	manifest.Scenario.Ref = "1.3.0"

	freshness := evaluateFreshness(repoRoot, &domain.Deployment{}, manifest)
	if freshness.Status != domain.FreshnessOutdated {
		t.Fatalf("expected overall status outdated, got %q", freshness.Status)
	}
	if freshness.VersionStatus != domain.FreshnessOutdated {
		t.Fatalf("expected version status outdated, got %q", freshness.VersionStatus)
	}
}

func TestEvaluateFreshnessOutdatedByFingerprint(t *testing.T) {
	scenarioID := "demo-scenario"
	repoRoot := setupFreshnessFixture(t, scenarioID, "1.0.0", "1.0.0")
	manifest := testManifest(scenarioID)
	manifest.Scenario.Ref = "1.0.0"

	mismatch := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	dep := &domain.Deployment{BundleSHA256: &mismatch}

	freshness := evaluateFreshness(repoRoot, dep, manifest)
	if freshness.Status != domain.FreshnessOutdated {
		t.Fatalf("expected overall status outdated, got %q", freshness.Status)
	}
	if freshness.FingerprintStatus != domain.FreshnessOutdated {
		t.Fatalf("expected fingerprint status outdated, got %q", freshness.FingerprintStatus)
	}
}

func TestEvaluateFreshnessCurrent(t *testing.T) {
	scenarioID := "demo-scenario"
	repoRoot := setupFreshnessFixture(t, scenarioID, "1.0.0", "1.0.0")
	manifest := testManifest(scenarioID)
	manifest.Scenario.Ref = "1.0.0"

	localSHA, _, err := bundle.CalculateBundleSHA(repoRoot, manifest)
	if err != nil {
		t.Fatalf("calculate local bundle sha: %v", err)
	}

	dep := &domain.Deployment{BundleSHA256: &localSHA}
	freshness := evaluateFreshness(repoRoot, dep, manifest)
	if freshness.Status != domain.FreshnessCurrent {
		t.Fatalf("expected overall status current, got %q", freshness.Status)
	}
	if freshness.VersionStatus != domain.FreshnessCurrent {
		t.Fatalf("expected version status current, got %q", freshness.VersionStatus)
	}
	if freshness.FingerprintStatus != domain.FreshnessCurrent {
		t.Fatalf("expected fingerprint status current, got %q", freshness.FingerprintStatus)
	}
}

func TestEvaluateFreshnessAddsVersionFallbackNote(t *testing.T) {
	scenarioID := "demo-scenario"
	repoRoot := setupFreshnessFixture(t, scenarioID, "1.0.0", "1.0.0")
	manifest := testManifest(scenarioID)

	if err := os.Remove(filepath.Join(repoRoot, "scenarios", scenarioID, ".vrooli", "service.json")); err != nil {
		t.Fatalf("remove service.json: %v", err)
	}
	if err := os.Remove(filepath.Join(repoRoot, "scenarios", scenarioID, "ui", "package.json")); err != nil {
		t.Fatalf("remove package.json: %v", err)
	}

	sha, _, err := bundle.CalculateBundleSHA(repoRoot, manifest)
	if err != nil {
		t.Fatalf("calculate local bundle sha: %v", err)
	}
	dep := &domain.Deployment{BundleSHA256: &sha}

	freshness := evaluateFreshness(repoRoot, dep, manifest)
	if freshness.VersionSource != versionSourceDefault {
		t.Fatalf("expected version source %q, got %q", versionSourceDefault, freshness.VersionSource)
	}

	found := false
	for _, note := range freshness.Notes {
		if note == noteVersionFallback {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected fallback warning note %q, got notes=%v", noteVersionFallback, freshness.Notes)
	}
}

func setupFreshnessFixture(t *testing.T, scenarioID, serviceVersion, uiVersion string) string {
	t.Helper()

	repoRoot := t.TempDir()
	writeRepoContractFixture(t, repoRoot)
	for _, dir := range []string{
		filepath.Join("scenarios", scenarioID, ".vrooli"),
		filepath.Join("scenarios", scenarioID, "ui"),
	} {
		if err := os.MkdirAll(filepath.Join(repoRoot, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	writeJSONFile(t, filepath.Join(repoRoot, ".vrooli", "service.json"), map[string]interface{}{
		"version": "2.0.0",
	})
	writeJSONFile(t, filepath.Join(repoRoot, "scenarios", scenarioID, ".vrooli", "service.json"), map[string]interface{}{
		"service": map[string]interface{}{
			"name":    scenarioID,
			"version": serviceVersion,
		},
		"ports": map[string]interface{}{
			"api": map[string]interface{}{
				"port":    8080,
				"env_var": "API_PORT",
			},
		},
	})
	writeJSONFile(t, filepath.Join(repoRoot, "scenarios", scenarioID, "ui", "package.json"), map[string]interface{}{
		"name":    scenarioID + "-ui",
		"version": uiVersion,
	})
	if err := os.WriteFile(filepath.Join(repoRoot, "scenarios", scenarioID, "README.md"), []byte("demo"), 0o644); err != nil {
		t.Fatalf("write scenario readme: %v", err)
	}

	return repoRoot
}

func testManifest(scenarioID string) domain.CloudManifest {
	return domain.CloudManifest{
		Version: "1.0.0",
		Target: domain.ManifestTarget{
			Type: "vps",
			VPS:  &domain.ManifestVPS{Host: "127.0.0.1"},
		},
		Scenario: domain.ManifestScenario{
			ID: scenarioID,
		},
		Dependencies: domain.ManifestDependencies{
			Scenarios: []string{scenarioID},
		},
		Bundle: domain.ManifestBundle{
			Scenarios: []string{scenarioID},
		},
		Ports: domain.ManifestPorts{"api": 8080},
		Edge: domain.ManifestEdge{
			Domain: "example.test",
			Caddy:  domain.ManifestCaddy{Enabled: true},
		},
	}
}

func writeJSONFile(t *testing.T, path string, value interface{}) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json for %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
