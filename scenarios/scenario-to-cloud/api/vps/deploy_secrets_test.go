package vps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"scenario-to-cloud/domain"
)

func TestReadLocalSecretsMapIgnoresMetadataAndInvalidData(t *testing.T) {
	t.Run("metadata preserved but excluded", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), ".vrooli", "secrets.json")
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
		path := filepath.Join(t.TempDir(), ".vrooli", "secrets.json")
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

	writeJSONFile(t, filepath.Join(root, ".vrooli", "secrets.json"), map[string]interface{}{
		"_metadata": map[string]interface{}{"managed_by": "test"},
		"API_KEY":   "workspace",
	})
	writeJSONFile(t, filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), map[string]interface{}{
		"service": map[string]interface{}{"name": "demo"},
	})
	writeJSONFile(t, filepath.Join(root, "scenarios", "demo", ".vrooli", "secrets.json"), map[string]interface{}{
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

	for _, dir := range []string{".vrooli", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	contract := `{
  "$schema": "schemas/repo-contract.schema.json",
  "version": "1.0.0",
  "platform": {"mode": "cross_platform_go_native", "legacy_project_bash_supported": false},
  "root": {"markers": {"required_dirs": [".vrooli", "scenarios", "resources", "packages", "cmd", "internal"], "required_files": ["go.mod"]}},
  "layout": {"project_config_dir": ".vrooli", "scenario_dir": "scenarios", "resource_dir": "resources", "package_dir": "packages", "command_dir": "cmd", "internal_dir": "internal", "docs_dir": "docs"},
  "scenario": {"required_files": [".vrooli/service.json"], "well_known_paths": {"service": ".vrooli/service.json", "docs": "docs", "requirements": "requirements", "api": "api", "ui": "ui", "cli": "cli", "initialization": "initialization"}},
  "resource": {"manifest": "resource.json", "well_known_paths": {"docs": "docs", "initialization": "initialization"}},
  "globs": {"syntax": "doublestar", "root_relative": true, "case_sensitive": true, "allow_absolute": false, "path_format": "slash_normalized"},
  "environment": {"variables": {"repo_root": "VROOLI_ROOT", "source_root": "VROOLI_SOURCE_ROOT", "sandbox_id": "VROOLI_SANDBOX_ID", "sandbox_merged": "VROOLI_SANDBOX_MERGED", "sandbox_scope": "VROOLI_SANDBOX_SCOPE"}},
  "sandbox": {"full_repo_scopes": ["", ".", "/"], "scenario_scope_prefix": "scenarios/"},
  "profiles": {
    "mini_vrooli_bundle": {
      "description": "fixture profile",
      "parameters": ["scenario", "resources[*]"],
      "include": [".vrooli", "cmd", "internal", "packages", "scenarios/{scenario}", "resources/{resources[*]}"],
      "optional_include": ["docs", "go.mod"],
      "exclude": [".git/**"]
    }
  }
}`
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), []byte(contract), 0o644); err != nil {
		t.Fatalf("write repo-contract.json: %v", err)
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
