package phasecacheidentity

import (
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/orchestrator/phasecache"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/providerreadiness"
	workspacepkg "test-genie/internal/orchestrator/workspace"
)

func TestProviderBuildChangeInvalidatesCachedPassAndFailure(t *testing.T) {
	for _, status := range []string{"passed", "failed"} {
		t.Run(status, func(t *testing.T) {
			scenarioDir := t.TempDir()
			if err := os.WriteFile(filepath.Join(scenarioDir, "input.txt"), []byte("stable"), 0o644); err != nil {
				t.Fatal(err)
			}
			env := workspacepkg.Environment{
				ScenarioName: "demo", ScenarioDir: scenarioDir, PhaseCacheRoot: filepath.Join(t.TempDir(), "cache"),
				DescriptorSnapshotDigest: "descriptor:v1", ExecutionConfigurationDigest: "config:v1",
			}
			phase := phases.Definition{
				Name: "api", ProviderScenario: "api-health",
				Determinism: phases.Determinism{Default: "file-determined", Inputs: []string{"**"}},
			}
			oldReady := map[string]providerreadiness.Outcome{
				"api": {Phase: "api", ProviderScenario: "api-health", Ready: true, Status: providerreadiness.OutcomeReady,
					SpecVersion: "1", BuildRevision: "old-build", BinaryModifiedAt: "2026-10-17T00:00:00Z", FreshnessDigest: "old-inputs"},
			}
			currentReady := map[string]providerreadiness.Outcome{
				"api": {Phase: "api", ProviderScenario: "api-health", Ready: true, Status: providerreadiness.OutcomeReady,
					SpecVersion: "1", BuildRevision: "new-build", BinaryModifiedAt: "2026-10-18T00:00:00Z", FreshnessDigest: "new-inputs"},
			}
			oldIdentity, ok := Identity(env, phase, oldReady)
			if !ok {
				t.Fatal("old provider build should have a cache identity")
			}
			if err := phasecache.New(env.PhaseCacheRoot).Save(phasecache.Key(oldIdentity), "old-run", phases.ExecutionResult{Name: "api", Status: status}); err != nil {
				t.Fatal(err)
			}
			lookup := Load("", env, "new-run", filepath.Join(t.TempDir(), "api.log"), phase, currentReady)
			if lookup.Found {
				t.Fatalf("%s result from old provider build was reused", status)
			}
		})
	}
}

func TestProviderPhaseWithoutLiveBuildIdentityIsNotCacheable(t *testing.T) {
	scenarioDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(scenarioDir, "input.txt"), []byte("stable"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := workspacepkg.Environment{
		ScenarioName: "demo", ScenarioDir: scenarioDir, PhaseCacheRoot: filepath.Join(t.TempDir(), "cache"),
		DescriptorSnapshotDigest: "descriptor:v1", ExecutionConfigurationDigest: "config:v1",
	}
	phase := phases.Definition{
		Name: "api", ProviderScenario: "api-health",
		Determinism: phases.Determinism{Default: "file-determined", Inputs: []string{"**"}},
	}
	for _, readiness := range []map[string]providerreadiness.Outcome{
		nil,
		{"api": {Phase: "api", ProviderScenario: "api-health", Ready: true, Status: providerreadiness.OutcomeReady, SpecVersion: "1"}},
	} {
		if _, ok := Identity(env, phase, readiness); ok {
			t.Fatalf("provider phase got cache identity without live build evidence: %+v", readiness)
		}
		if lookup := Load("", env, "current-run", filepath.Join(t.TempDir(), "api.log"), phase, readiness); lookup.Found {
			t.Fatalf("provider phase reused cache without live build evidence: %+v", readiness)
		}
	}
}
