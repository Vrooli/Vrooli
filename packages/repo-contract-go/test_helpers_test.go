package repocontract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	root := filepath.Clean(filepath.Join(dir, "..", ".."))
	if _, err := os.Stat(filepath.Join(root, ".vrooli", "repo-contract.json")); err != nil {
		t.Fatalf("repo root missing live contract: %v", err)
	}
	return root
}

func mustLoadDefault(t *testing.T, root string) *Contract {
	t.Helper()
	contract, err := LoadDefault(root)
	if err != nil {
		t.Fatalf("LoadDefault() error = %v", err)
	}
	return contract
}

func writeContractFile(t *testing.T, dir string, doc contractDoc) string {
	t.Helper()
	path := filepath.Join(dir, ".vrooli", "repo-contract.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
	return path
}

func validContractDoc() contractDoc {
	return contractDoc{
		Schema:  "schemas/repo-contract.schema.json",
		Version: "1.0.0",
		Platform: Platform{
			Mode:                       "cross_platform_go_native",
			LegacyProjectBashSupported: false,
		},
		Root: Root{
			Markers: RootMarkers{
				RequiredDirs:  []string{".vrooli", "scenarios", "resources", "packages", "cmd", "internal"},
				RequiredFiles: []string{"go.mod"},
			},
		},
		Layout: Layout{
			ProjectConfigDir: ".vrooli",
			ScenarioDir:      "scenarios",
			ResourceDir:      "resources",
			PackageDir:       "packages",
			CommandDir:       "cmd",
			InternalDir:      "internal",
			DocsDir:          "docs",
		},
		Scenario: ScenarioSpec{
			RequiredFiles: []string{".vrooli/service.json"},
			WellKnownPaths: map[string]string{
				"service":        ".vrooli/service.json",
				"docs":           "docs",
				"requirements":   "requirements",
				"api":            "api",
				"ui":             "ui",
				"cli":            "cli",
				"initialization": "initialization",
			},
		},
		Resource: ResourceSpec{
			Manifest: "resource.json",
			WellKnownPaths: map[string]string{
				"docs":           "docs",
				"initialization": "initialization",
			},
		},
		Globs: GlobSpec{
			Syntax:        "doublestar",
			RootRelative:  true,
			CaseSensitive: true,
			AllowAbsolute: false,
			PathFormat:    "slash_normalized",
		},
		Environment: struct {
			Variables map[string]string `json:"variables"`
		}{
			Variables: map[string]string{
				"repo_root":      "VROOLI_ROOT",
				"source_root":    "VROOLI_SOURCE_ROOT",
				"sandbox_id":     "VROOLI_SANDBOX_ID",
				"sandbox_merged": "VROOLI_SANDBOX_MERGED",
				"sandbox_scope":  "VROOLI_SANDBOX_SCOPE",
			},
		},
		Sandbox: struct {
			FullRepoScopes      []string `json:"full_repo_scopes"`
			ScenarioScopePrefix string   `json:"scenario_scope_prefix"`
		}{
			FullRepoScopes:      []string{"", ".", "/"},
			ScenarioScopePrefix: "scenarios/",
		},
		Profiles: map[string]Profile{
			"mini_vrooli_bundle": {
				Description:     "Repo-aware bundle profile for tests.",
				Parameters:      []string{"scenario", "resources[*]"},
				Include:         []string{"packages", "scenarios/{scenario}", "resources/{resources[*]}"},
				OptionalInclude: []string{"go.mod"},
				Exclude:         []string{"**/coverage/**", ".vrooli/secrets.json"},
			},
		},
	}
}

func validContract() *Contract {
	doc := validContractDoc()
	return &Contract{doc: deepCopyContractDoc(doc)}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
