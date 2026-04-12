package secrets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewProjectStoreResolvesContractBackedPath(t *testing.T) {
	repoRoot := writeTempRepo(t)
	store, err := NewProjectStore(Config{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("NewProjectStore: %v", err)
	}
	want := filepath.Join(repoRoot, ".vrooli", "secrets.json")
	if got := store.PlaintextPath(); got != want {
		t.Fatalf("PlaintextPath = %q, want %q", got, want)
	}
}

func TestNewProjectStoreFallsBackToConventionalRoot(t *testing.T) {
	root := t.TempDir()
	store, err := NewProjectStore(Config{RepoRoot: root})
	if err != nil {
		t.Fatalf("NewProjectStore: %v", err)
	}
	want := filepath.Join(root, ".vrooli", "secrets.json")
	if got := store.PlaintextPath(); got != want {
		t.Fatalf("PlaintextPath = %q, want %q", got, want)
	}
}

func TestResolvePrefersEnvThenFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".vrooli", "secrets.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{\"API_KEY\":\"file-secret\"}\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	store, err := NewProjectStore(Config{
		RepoRoot: root,
		EnvLookup: func(key string) string {
			if key == "API_KEY" {
				return "env-secret"
			}
			return ""
		},
	})
	if err != nil {
		t.Fatalf("NewProjectStore: %v", err)
	}

	resolved, err := store.Resolve("API_KEY")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Source != SourceEnv || resolved.Value != "env-secret" {
		t.Fatalf("Resolve = %#v, want env-secret from env", resolved)
	}

	store, err = NewProjectStore(Config{RepoRoot: root})
	if err != nil {
		t.Fatalf("NewProjectStore: %v", err)
	}
	resolved, err = store.Resolve("API_KEY")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Source != SourceFile || resolved.Value != "file-secret" || resolved.SourcePath != path {
		t.Fatalf("Resolve = %#v, want file-secret from file", resolved)
	}
}

func TestResolveReturnsMissingWhenUnavailable(t *testing.T) {
	store, err := NewProjectStore(Config{RepoRoot: t.TempDir()})
	if err != nil {
		t.Fatalf("NewProjectStore: %v", err)
	}
	resolved, err := store.Resolve("MISSING")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Source != SourceMissing || resolved.Value != "" {
		t.Fatalf("Resolve = %#v, want missing", resolved)
	}
}

func TestLoadFileDocumentPreservesMetadataAndRejectsBadValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".vrooli", "secrets.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{\"_metadata\":{\"environment\":\"development\"},\"API_KEY\":\"secret\"}\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	doc, err := LoadFileDocument(path)
	if err != nil {
		t.Fatalf("LoadFileDocument: %v", err)
	}
	if doc.Secrets["API_KEY"] != "secret" {
		t.Fatalf("API_KEY = %q, want secret", doc.Secrets["API_KEY"])
	}
	if _, ok := doc.Metadata["_metadata"]; !ok {
		t.Fatal("expected _metadata to be preserved")
	}

	if err := os.WriteFile(path, []byte("{\"API_KEY\":42}\n"), 0o600); err != nil {
		t.Fatalf("write invalid: %v", err)
	}
	_, err = LoadFileDocument(path)
	if err == nil {
		t.Fatal("expected invalid data error")
	}
}

func TestLoadFileDocumentRejectsUnsafeFiles(t *testing.T) {
	t.Run("broad perms", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), ".vrooli", "secrets.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte("{\"API_KEY\":\"secret\"}\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		_, err := LoadFileDocument(path)
		if err == nil {
			t.Fatal("expected insecure permissions error")
		}
	})

	t.Run("symlink", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "target.json")
		if err := os.WriteFile(target, []byte("{\"API_KEY\":\"secret\"}\n"), 0o600); err != nil {
			t.Fatalf("write target: %v", err)
		}
		path := filepath.Join(dir, ".vrooli", "secrets.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.Symlink(target, path); err != nil {
			t.Fatalf("symlink: %v", err)
		}
		_, err := LoadFileDocument(path)
		if err == nil {
			t.Fatal("expected symlink error")
		}
	})
}

func TestWriteFileDocumentWritesPrivateAtomicFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".vrooli", "secrets.json")
	err := WriteFileDocument(path, Document{
		Secrets: map[string]string{"API_KEY": "secret"},
		Metadata: map[string]json.RawMessage{
			"_metadata": json.RawMessage(`{"environment":"development"}`),
		},
	})
	if err != nil {
		t.Fatalf("WriteFileDocument: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 600", got)
	}
	doc, err := LoadFileDocument(path)
	if err != nil {
		t.Fatalf("LoadFileDocument: %v", err)
	}
	if doc.Secrets["API_KEY"] != "secret" {
		t.Fatalf("API_KEY = %q, want secret", doc.Secrets["API_KEY"])
	}
}

func TestNewFileStoreSaveDeleteAndUpdatePreserveMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".vrooli", "secrets.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{\"_metadata\":{\"managed_by\":\"test\"},\"API_KEY\":\"old\"}\n"), 0o600); err != nil {
		t.Fatalf("write seed: %v", err)
	}

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	if err := store.SaveKey("API_KEY", "updated"); err != nil {
		t.Fatalf("SaveKey: %v", err)
	}
	if err := store.Update(func(doc *Document) error {
		doc.Metadata["_metadata"] = json.RawMessage(`{"managed_by":"test","last_updated":"now"}`)
		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	deleted, err := store.DeleteKey("API_KEY")
	if err != nil {
		t.Fatalf("DeleteKey: %v", err)
	}
	if !deleted {
		t.Fatal("expected API_KEY to be deleted")
	}
	doc, err := store.Document()
	if err != nil {
		t.Fatalf("Document: %v", err)
	}
	if len(doc.Secrets) != 0 {
		t.Fatalf("Secrets = %#v, want empty", doc.Secrets)
	}
	if _, ok := doc.Metadata["_metadata"]; !ok {
		t.Fatal("expected metadata to be preserved")
	}
}

func TestNewProjectStoreFromEnvOrCWDFindsNearestConfigDir(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, ".vrooli")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	start := filepath.Join(root, "nested", "deeper")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	origGetwd := getwdFn
	getwdFn = func() (string, error) { return start, nil }
	t.Cleanup(func() { getwdFn = origGetwd })
	origFindRepoRootFromEnvOrCWD := findRepoRootFromEnvOrCWD
	findRepoRootFromEnvOrCWD = func() (string, error) {
		return "", &Error{Kind: ErrResolve, Message: "test fallback discovery"}
	}
	t.Cleanup(func() { findRepoRootFromEnvOrCWD = origFindRepoRootFromEnvOrCWD })

	store, err := NewProjectStoreFromEnvOrCWD(Config{
		EnvLookup: func(string) string { return "" },
	})
	if err != nil {
		t.Fatalf("NewProjectStoreFromEnvOrCWD: %v", err)
	}
	want := filepath.Join(root, ".vrooli", "secrets.json")
	if got := store.PlaintextPath(); got != want {
		t.Fatalf("PlaintextPath = %q, want %q", got, want)
	}
}

func writeTempRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	requiredDirs := []string{".vrooli", "scenarios", "resources", "packages", "cmd", "internal"}
	for _, dir := range requiredDirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	contract := map[string]any{
		"$schema": "schemas/repo-contract.schema.json",
		"version": "1.0.0",
		"platform": map[string]any{
			"mode":                          "cross_platform_go_native",
			"legacy_project_bash_supported": false,
		},
		"root": map[string]any{
			"markers": map[string]any{
				"required_dirs":  requiredDirs,
				"required_files": []string{"go.mod"},
			},
		},
		"layout": map[string]any{
			"project_config_dir": ".vrooli",
			"scenario_dir":       "scenarios",
			"resource_dir":       "resources",
			"package_dir":        "packages",
			"command_dir":        "cmd",
			"internal_dir":       "internal",
			"docs_dir":           "docs",
		},
		"scenario": map[string]any{
			"required_files": []string{".vrooli/service.json"},
			"well_known_paths": map[string]string{
				"service": ".vrooli/service.json",
			},
		},
		"resource": map[string]any{
			"manifest":         "resource.json",
			"well_known_paths": map[string]string{},
		},
		"globs": map[string]any{
			"syntax":         "doublestar",
			"root_relative":  true,
			"case_sensitive": true,
			"allow_absolute": false,
			"path_format":    "slash_normalized",
		},
		"environment": map[string]any{
			"variables": map[string]string{
				"repo_root": "VROOLI_ROOT",
			},
		},
		"sandbox": map[string]any{
			"full_repo_scopes":      []string{"", ".", "/"},
			"scenario_scope_prefix": "scenarios/",
		},
		"profiles": map[string]any{},
	}
	data, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), data, 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}
	return root
}
