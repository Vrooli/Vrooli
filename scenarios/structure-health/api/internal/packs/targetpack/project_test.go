package targetpack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func validProjectFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{".vrooli", "templates", "scenarios/demo/.vrooli", "resources/demo", "packages", "cmd", "internal", "docs", ".vrooli/schemas"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write(t, root, ".vrooli/repo-contract.json", projectFixtureJSON(t))
	write(t, root, ".vrooli/service.json", `{}`)
	write(t, root, "scenarios/demo/.vrooli/service.json", `{}`)
	write(t, root, "resources/demo/resource.json", `{}`)
	write(t, root, "go.mod", "module example.test/repo\n\ngo 1.25\n")
	write(t, root, ".vrooli/schemas/resource-definitions.json", `{}`)
	write(t, root, "docs/repo-contract.md", "`vrooli contract validate` `vrooli contract show` `vrooli contract resolve scenario <name> --file service` `vrooli contract match-glob <pattern> <path>` `structure-health-contract` ## Allowed `.vrooli/` Surface `~/.vrooli/secrets.json` ## Landed Consumer Migrations `swarm-manager`")
	return root
}

func projectFixtureJSON(t *testing.T) string {
	t.Helper()
	entries := map[string]any{}
	for _, item := range []struct {
		key, path, kind        string
		regenerable, sensitive bool
	}{
		{"plans", "plans", "dir", false, false}, {"state", "state", "dir", false, false}, {"config", "config", "dir", false, false}, {"data", "data", "dir", false, false}, {"runtime_db", "state/runtime.db", "file", false, false}, {"secrets", "secrets.json", "file", false, true}, {"secrets_enc", "secrets.enc.json", "file", false, true}, {"bin", "bin", "dir", true, false}, {"cache", "cache", "dir", true, false}, {"logs", "logs", "dir", true, false}, {"metrics", "metrics", "dir", true, false}, {"processes", "processes", "dir", true, false}, {"build", "build", "dir", true, false},
	} {
		entries[item.key] = map[string]any{"path": item.path, "kind": item.kind, "regenerable": item.regenerable, "sensitive": item.sensitive}
	}
	doc := map[string]any{
		"$schema": "schemas/repo-contract.schema.json", "version": "1.2.0",
		"platform":     map[string]any{"mode": "cross_platform_go_native", "legacy_project_bash_supported": false},
		"root":         map[string]any{"markers": map[string]any{"required_dirs": []string{".vrooli", "templates", "scenarios", "resources", "packages", "cmd", "internal"}, "required_files": []string{"go.mod"}}},
		"layout":       map[string]any{"project_config_dir": ".vrooli", "scenario_dir": "scenarios", "resource_dir": "resources", "template_dir": "templates", "package_dir": "packages", "command_dir": "cmd", "internal_dir": "internal", "docs_dir": "docs", "project_config_allowlist": []string{"build", "repo-contract.json", "resources", "schemas", "service.json"}},
		"runtime_home": map[string]any{"dir_name": ".vrooli", "env_overrides": []any{}, "entries": entries, "scoped": map[string]string{"scenario_secrets": "scenarios/{scenario}/secrets.json", "project_state": "state/projects/{project_key}"}},
		"scenario":     map[string]any{"required_files": []string{".vrooli/service.json"}, "well_known_paths": map[string]string{"service": ".vrooli/service.json", "orientation": ".vrooli/orientation.json", "docs": "docs", "docs_manifest": "docs/manifest.json", "requirements": "requirements", "api": "api", "ui": "ui", "cli": "cli", "cli_manifest": "cli/manifest.json"}},
		"resource":     map[string]any{"manifest": "resource.json", "well_known_paths": map[string]string{"docs": "docs"}},
		"globs":        map[string]any{"syntax": "doublestar", "root_relative": true, "case_sensitive": true, "allow_absolute": false, "path_format": "slash_normalized"},
		"environment":  map[string]any{"variables": map[string]string{"repo_root": "VROOLI_ROOT", "source_root": "VROOLI_SOURCE_ROOT", "sandbox_id": "VROOLI_SANDBOX_ID", "sandbox_merged": "VROOLI_SANDBOX_MERGED", "sandbox_scope": "VROOLI_SANDBOX_SCOPE"}},
		"sandbox":      map[string]any{"full_repo_scopes": []string{"", ".", "/"}, "scenario_scope_prefix": "scenarios/"},
		"profiles":     map[string]any{"mini_vrooli_bundle": map[string]any{"parameters": []string{"scenario", "resources[*]"}, "include": []string{".vrooli", "cmd", "internal", "packages", "scenarios/{scenario}", "resources/{resources[*]}"}, "optional_include": []string{"docs", "go.mod", "go.sum", "go.work", "go.work.sum", "Makefile", "README.md", "LICENSE"}, "exclude": []string{".git/**", "**/.git/**", "**/node_modules/**", "**/coverage/**", "**/data/**", ".vrooli/secrets.json", "**/.vrooli/secrets.json", "cli/**"}}},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func mutateProjectContract(t *testing.T, root string, mutate func(map[string]any)) {
	t.Helper()
	path := filepath.Join(root, ".vrooli/repo-contract.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	mutate(doc)
	updated, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, updated, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProjectPackPositive(t *testing.T) {
	if got := Evaluate("project", validProjectFixture(t), "repo"); len(got) != 0 {
		t.Fatalf("conforming project findings = %v", got)
	}
}

func TestProjectPackReportsEveryProjectConfigViolation(t *testing.T) {
	root := validProjectFixture(t)
	write(t, root, ".vrooli/agent-manager", "copy")
	write(t, root, ".vrooli/baselines", "scratch")
	got := codes(Evaluate("project", root, "repo"))
	if !got["PROJECT_CONFIG_SURFACE"] {
		t.Fatalf("expected project config surface finding, got %v", got)
	}
	count := 0
	for _, finding := range Evaluate("project", root, "repo") {
		if finding.Code == "PROJECT_CONFIG_SURFACE" {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("project config violations = %d, want 2", count)
	}
}

func TestProjectPackNegativeRuleFixtures(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		mutate func(string)
	}{
		{"phase1", "PROJECT_PHASE1_SEMANTICS", func(root string) { mutateProjectContract(t, root, func(doc map[string]any) { doc["version"] = "" }) }},
		{"canonical", "PROJECT_CANONICAL_LAYOUT", func(root string) {
			mutateProjectContract(t, root, func(doc map[string]any) { doc["layout"].(map[string]any)["docs_dir"] = "manuals" })
		}},
		{"runtime-home", "PROJECT_RUNTIME_HOME", func(root string) {
			mutateProjectContract(t, root, func(doc map[string]any) { doc["runtime_home"].(map[string]any)["dir_name"] = "runtime" })
		}},
		{"live-structure", "PROJECT_LIVE_STRUCTURE", func(root string) {
			if err := os.RemoveAll(filepath.Join(root, "cmd")); err != nil {
				t.Fatal(err)
			}
		}},
		{"config-surface", "PROJECT_CONFIG_SURFACE", func(root string) { write(t, root, ".vrooli/unapproved", "x") }},
		{"excluded-legacy", "PROJECT_EXCLUDED_LEGACY", func(root string) {
			if err := os.MkdirAll(filepath.Join(root, "scripts/lib"), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"profile-roots", "PROJECT_PROFILE_ROOTS", func(root string) {
			mutateProjectContract(t, root, func(doc map[string]any) {
				doc["profiles"].(map[string]any)["mini_vrooli_bundle"].(map[string]any)["include"] = []any{"rogue"}
			})
		}},
		{"bundle-policy", "PROJECT_BUNDLE_PROFILE", func(root string) {
			mutateProjectContract(t, root, func(doc map[string]any) {
				doc["profiles"].(map[string]any)["mini_vrooli_bundle"].(map[string]any)["include"] = []any{".vrooli"}
			})
		}},
		{"docs-alignment", "PROJECT_DOCS_ALIGNMENT", func(root string) { write(t, root, "docs/repo-contract.md", "outdated") }},
		{"resource-artifacts", "PROJECT_RESOURCE_ARTIFACTS", func(root string) {
			if err := os.Remove(filepath.Join(root, ".vrooli/schemas/resource-definitions.json")); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := validProjectFixture(t)
			tc.mutate(root)
			if got := codes(Evaluate("project", root, "repo")); !got[tc.code] {
				t.Fatalf("missing %s in %v", tc.code, got)
			}
		})
	}
}
