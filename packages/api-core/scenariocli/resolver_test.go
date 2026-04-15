package scenariocli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveExecutableInstallsShellScriptCLI(t *testing.T) {
	fixture := newRepoFixture(t)
	writeShellScenario(t, fixture.root, "scenario-auditor")

	path, err := ResolveExecutable(fixture.root, fixture.home, "scenario-auditor")
	if err != nil {
		t.Fatalf("ResolveExecutable: %v", err)
	}
	if path != filepath.Join(fixture.home, ".vrooli", "bin", "scenario-auditor") {
		t.Fatalf("path = %q", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("installed binary missing: %v", err)
	}
	metaPath := path + ".build.meta"
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	var meta installMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if strings.TrimSpace(meta.Fingerprint) == "" {
		t.Fatal("expected fingerprint metadata")
	}
}

func TestResolveExecutableReinstallsWhenFreshnessInputsChange(t *testing.T) {
	fixture := newRepoFixture(t)
	scenarioRoot := writeShellScenario(t, fixture.root, "scenario-auditor")

	path, err := ResolveExecutable(fixture.root, fixture.home, "scenario-auditor")
	if err != nil {
		t.Fatalf("ResolveExecutable(1): %v", err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read first binary: %v", err)
	}

	if err := os.WriteFile(filepath.Join(scenarioRoot, "cli", "scenario-auditor"), []byte("#!/usr/bin/env bash\necho second\n"), 0o755); err != nil {
		t.Fatalf("update script: %v", err)
	}

	path, err = ResolveExecutable(fixture.root, fixture.home, "scenario-auditor")
	if err != nil {
		t.Fatalf("ResolveExecutable(2): %v", err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read second binary: %v", err)
	}
	if string(first) == string(second) {
		t.Fatal("expected reinstall after freshness input changed")
	}
}

type repoFixture struct {
	root string
	home string
}

func newRepoFixture(t *testing.T) repoFixture {
	t.Helper()
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeRepoContractFixture(t, root)
	return repoFixture{root: root, home: home}
}

func writeRepoContractFixture(t *testing.T, root string) {
	t.Helper()
	projectRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	data, err := os.ReadFile(filepath.Join(projectRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read repo contract: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal repo contract: %v", err)
	}
	layout := ensureObject(doc, "layout")
	layout["scenario_dir"] = "scenarios"
	sandbox := ensureObject(doc, "sandbox")
	sandbox["scenario_scope_prefix"] = "scenarios/"
	rootDoc := ensureObject(doc, "root")
	markers := ensureObject(rootDoc, "markers")
	markers["required_dirs"] = []any{"scenarios", "resources", "packages", "cmd", "internal"}
	for _, dir := range []string{".vrooli", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("marshal repo contract: %v", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), encoded, 0o644); err != nil {
		t.Fatalf("write repo contract: %v", err)
	}
}

func writeShellScenario(t *testing.T, root, name string) string {
	t.Helper()
	scenarioRoot := filepath.Join(root, "scenarios", name)
	if err := os.MkdirAll(filepath.Join(scenarioRoot, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioRoot, "cli"), 0o755); err != nil {
		t.Fatalf("mkdir cli dir: %v", err)
	}
	script := "#!/usr/bin/env bash\nset -e\nmkdir -p \"$HOME/.vrooli/bin\"\ncp \"$(dirname \"$0\")/scenario-auditor\" \"$HOME/.vrooli/bin/scenario-auditor\"\nchmod +x \"$HOME/.vrooli/bin/scenario-auditor\"\n"
	if err := os.WriteFile(filepath.Join(scenarioRoot, "cli", "install.sh"), []byte(script), 0o755); err != nil {
		t.Fatalf("write install.sh: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioRoot, "cli", name), []byte("#!/usr/bin/env bash\necho first\n"), 0o755); err != nil {
		t.Fatalf("write cli binary: %v", err)
	}
	manifest := map[string]any{
		"service": map[string]any{"name": name},
		"cli": map[string]any{
			"enabled": true,
			"command": name,
			"adapter": map[string]any{
				"kind":           "shell_script",
				"script_path":    "cli/" + name,
				"install_script": "cli/install.sh",
			},
			"install": []map[string]any{{
				"kind": "command",
				"run":  "bash ./cli/install.sh",
			}},
			"freshness": map[string]any{
				"inputs": []string{"cli/" + name, "cli/install.sh"},
			},
		},
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(scenarioRoot, ".vrooli", "service.json"), data, 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}
	return scenarioRoot
}

func ensureObject(parent map[string]any, key string) map[string]any {
	if value, ok := parent[key].(map[string]any); ok {
		return value
	}
	created := map[string]any{}
	parent[key] = created
	return created
}
