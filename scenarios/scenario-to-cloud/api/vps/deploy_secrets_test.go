package vps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"scenario-to-cloud/domain"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestReadLocalSecretsMapIgnoresMetadataAndInvalidData(t *testing.T) {
	t.Run("metadata preserved but excluded", func(t *testing.T) {
		path, err := repocontract.UserPlaintextSecretsPath(t.TempDir())
		if err != nil {
			t.Fatalf("UserPlaintextSecretsPath: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(`{"_metadata":{"environment":"development"},"API_KEY":"secret"}`), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}

		values := readLocalSecretsMap(path)
		if got := values["API_KEY"]; got != "secret" {
			t.Fatalf("API_KEY = %q, want secret", got)
		}
		if _, ok := values["_metadata"]; ok {
			t.Fatalf("expected metadata key to be ignored, got %#v", values)
		}
	})

	t.Run("invalid document returns empty map", func(t *testing.T) {
		path, err := repocontract.UserPlaintextSecretsPath(t.TempDir())
		if err != nil {
			t.Fatalf("UserPlaintextSecretsPath: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(`{"API_KEY":42}`), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}

		values := readLocalSecretsMap(path)
		if len(values) != 0 {
			t.Fatalf("values = %#v, want empty map", values)
		}
	})
}

func TestBuildUserSecretMapPrefersScenarioThenExplicitSecrets(t *testing.T) {
	root := t.TempDir()
	writeRepoContractFixture(t, root)
	t.Setenv("SCENARIO_TO_CLOUD_REPO_ROOT", root)
	home := t.TempDir()
	t.Setenv("HOME", home)

	workspacePath, err := repocontract.UserPlaintextSecretsPath(home)
	if err != nil {
		t.Fatalf("UserPlaintextSecretsPath: %v", err)
	}
	scenarioPath, err := repocontract.UserScenarioPlaintextSecretsPath(home, "demo")
	if err != nil {
		t.Fatalf("UserScenarioPlaintextSecretsPath: %v", err)
	}

	writeJSONFile(t, workspacePath, map[string]interface{}{
		"_metadata": map[string]interface{}{"managed_by": "test"},
		"API_KEY":   "workspace",
	})
	writeJSONFile(t, filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), map[string]interface{}{
		"service": map[string]interface{}{"name": "demo"},
	})
	writeJSONFile(t, scenarioPath, map[string]interface{}{
		"_metadata": map[string]interface{}{"managed_by": "test"},
		"API_KEY":   "scenario",
	})

	manifest := domain.CloudManifest{
		Scenario: domain.ManifestScenario{ID: "demo"},
		Secrets: &domain.ManifestSecrets{
			BundleSecrets: []domain.BundleSecretPlan{
				{
					ID:       "api-key",
					Class:    "user_prompt",
					Required: true,
					Target:   domain.BundleSecretTarget{Name: "API_KEY"},
				},
			},
		},
	}

	got := buildUserSecretMap(manifest, map[string]string{"API_KEY": "provided"})
	if got["API_KEY"] != "provided" {
		t.Fatalf("buildUserSecretMap explicit = %q, want %q", got["API_KEY"], "provided")
	}

	got = buildUserSecretMap(manifest, nil)
	if got["API_KEY"] != "scenario" {
		t.Fatalf("buildUserSecretMap scenario = %q, want %q", got["API_KEY"], "scenario")
	}
}

func TestRequiredResourcesForScenarioUsesContractResolvedServicePath(t *testing.T) {
	root := t.TempDir()
	writeRepoContractFixture(t, root)
	t.Setenv("SCENARIO_TO_CLOUD_REPO_ROOT", root)

	writeJSONFile(t, filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), map[string]interface{}{
		"service": map[string]interface{}{"name": "demo"},
		"dependencies": map[string]interface{}{
			"resources": map[string]interface{}{
				"postgres": map[string]interface{}{"enabled": true},
				"redis":    map[string]interface{}{"enabled": true},
				"vault":    map[string]interface{}{"enabled": false},
			},
		},
	})

	got, err := RequiredResourcesForScenario("demo")
	if err != nil {
		t.Fatalf("RequiredResourcesForScenario: %v", err)
	}
	if len(got) != 2 || got[0] != "postgres" || got[1] != "redis" {
		t.Fatalf("RequiredResourcesForScenario = %#v, want [postgres redis]", got)
	}
}

func writeRepoContractFixture(t *testing.T, root string) {
	t.Helper()

	for _, dir := range []string{".vrooli", "scenarios", "resources", "templates", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	// Copy the live repo's .vrooli/repo-contract.json verbatim rather than
	// hand-typing a literal. This keeps the single source of truth authoritative
	// and prevents the fixture from drifting when the contract schema gains a
	// required field (e.g. runtime_home).
	contract := liveRepoContract(t)
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contract, 0o644); err != nil {
		t.Fatalf("write repo-contract.json: %v", err)
	}
}

// liveRepoContract reads the repository's authoritative
// .vrooli/repo-contract.json by walking up from this source file until the
// contract is found, returning the raw bytes for verbatim copy into a fixture.
func liveRepoContract(t *testing.T) []byte {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed; cannot locate live repo contract")
	}
	dir := filepath.Dir(filename)
	for {
		candidate := filepath.Join(dir, ".vrooli", "repo-contract.json")
		if data, err := os.ReadFile(candidate); err == nil {
			return data
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate .vrooli/repo-contract.json above test package")
		}
		dir = parent
	}
}

func writeJSONFile(t *testing.T, path string, payload map[string]interface{}) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
