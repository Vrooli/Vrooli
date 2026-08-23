package targetpack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	write(t, root, "pnpm-workspace.yaml", "packages:\n  - packages/*\nautoInstallPeers: false\nlink-workspace-packages: false\n")
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

func TestProjectPackRejectsDuplicateCredentialDescriptorInOneManifest(t *testing.T) {
	root := validProjectFixture(t)
	write(t, root, "scenarios/demo/.vrooli/service.json", `{"credentials":{"descriptors":[{"logical_id":"vrooli/demo","field":"token"},{"logical_id":"vrooli/demo","field":"token"}]}}`)
	got := Evaluate("project", root, "repo")
	for _, finding := range got {
		if finding.Code == "PROJECT_CREDENTIAL_DESCRIPTOR_DUPLICATE" {
			if !strings.Contains(finding.Message, "vrooli/demo:token") || !strings.Contains(finding.Location, "/credentials/descriptors/1") {
				t.Fatalf("duplicate finding lacks descriptor evidence: %+v", finding)
			}
			return
		}
	}
	t.Fatalf("expected duplicate credential descriptor finding, got %v", got)
}

func TestProjectPackRejectsDuplicateSchemaID(t *testing.T) {
	root := validProjectFixture(t)
	write(t, root, "schemas/first.schema.json", `{"$schema":"http://json-schema.org/draft-07/schema#","$id":"https://example.test/shared"}`)
	write(t, root, "scenarios/demo/schemas/second.schema.json", `{"$schema":"http://json-schema.org/draft-07/schema#","$id":"https://example.test/shared"}`)

	got := Evaluate("project", root, "repo")
	for _, finding := range got {
		if finding.Code == "REPO_SCHEMA_ID_UNIQUE" {
			if !strings.Contains(finding.Message, "schemas/first.schema.json") || !strings.Contains(finding.Message, "scenarios/demo/schemas/second.schema.json") {
				t.Fatalf("duplicate finding does not name both owners: %+v", finding)
			}
			return
		}
	}
	t.Fatalf("expected duplicate schema id finding, got %v", got)
}

func TestRepositorySchemaIDsAreUnique(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(root, ".vrooli", "schemas", "service.schema.json")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Skip("repository root is unavailable")
		}
		root = parent
	}
	if findings := projectSchemaIDRules(root); len(findings) != 0 {
		t.Fatalf("repository schema IDs are not unique: %v", findings)
	}
}

func TestProjectPackReportsOwningScenarioManifestSchemaViolation(t *testing.T) {
	root := validProjectFixture(t)
	write(t, root, ".vrooli/schemas/cli-manifest.schema.json", `{"type":"object","required":["name"],"properties":{"name":{"type":"string"}},"additionalProperties":true}`)
	write(t, root, "cli/manifest.json", `{"name":"repo"}`)
	write(t, root, "scenarios/demo/cli/manifest.json", `{"name":42}`)

	for _, finding := range Evaluate("project", root, "repo") {
		if finding.Code == "PROJECT_CLI_MANIFEST_SCHEMA" && finding.Location == "scenarios/demo/cli/manifest.json" {
			if !strings.Contains(finding.Message, "scenario/demo") {
				t.Fatalf("manifest finding lacks owning scenario: %+v", finding)
			}
			return
		}
	}
	t.Fatalf("expected owning scenario manifest schema finding, got %v", Evaluate("project", root, "repo"))
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
		{"root-lock", "PROJECT_ROOT_PNPM_LOCK", func(root string) { write(t, root, "pnpm-lock.yaml", "lockfileVersion: 9") }},
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
