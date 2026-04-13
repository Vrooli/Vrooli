package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

type RepoFixture struct {
	Root string
	Home string
}

type RepoSupportDocs struct {
	RepoContractDoc []string
	ContributingDoc []string
	AgentsDoc       []string
	SkillDoc        []string
}

func NewRepoFixture(t *testing.T) RepoFixture {
	t.Helper()
	return RepoFixture{
		Root: t.TempDir(),
		Home: t.TempDir(),
	}
}

func ProjectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func WriteFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func WriteExecutable(t *testing.T, path, contents string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func WriteJSON(t *testing.T, path string, value any) {
	t.Helper()
	WriteJSONMode(t, path, value, 0o644)
}

func WriteJSONMode(t *testing.T, path string, value any, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func WriteRawJSON(t *testing.T, path, raw string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if !strings.HasSuffix(raw, "\n") {
		raw += "\n"
	}
	if err := os.WriteFile(path, []byte(raw), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func ReadJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return parsed
}

func ContainsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func WriteRepoContract(t *testing.T, root, scenarioDir string) {
	t.Helper()
	projectRoot := ProjectRoot(t)
	contractPath := filepath.Join(projectRoot, ".vrooli", "repo-contract.json")
	data, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read live repo contract: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal live repo contract: %v", err)
	}

	layout := ensureObject(doc, "layout")
	layout["scenario_dir"] = scenarioDir

	rootDoc := ensureObject(doc, "root")
	markers := ensureObject(rootDoc, "markers")
	requiredDirs, _ := markers["required_dirs"].([]any)
	if len(requiredDirs) > 0 {
		updated := make([]any, 0, len(requiredDirs))
		for _, value := range requiredDirs {
			text, _ := value.(string)
			if text == "scenarios" {
				text = scenarioDir
			}
			updated = append(updated, text)
		}
		markers["required_dirs"] = updated
	}

	scenarioSpec := ensureObject(doc, "scenario")
	sandbox := ensureObject(doc, "sandbox")
	sandbox["scenario_scope_prefix"] = scenarioDir + "/"

	if wellKnown, ok := scenarioSpec["well_known_paths"].(map[string]any); ok {
		scenarioSpec["well_known_paths"] = cloneStringMapAny(wellKnown)
	}

	for _, dir := range []string{".vrooli", scenarioDir, "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	WriteFile(t, filepath.Join(root, "go.mod"), "module test\n")
	WriteJSON(t, filepath.Join(root, ".vrooli", "repo-contract.json"), doc)
}

func DefaultRepoSupportDocs() RepoSupportDocs {
	return RepoSupportDocs{
		RepoContractDoc: []string{
			"# Repo Contract",
			"## Adoption Rules",
			"## Grandfathered Debt and Exceptions",
			"`.vrooli/repo-contract-adoption-exceptions.json`",
			"future repo-aware work",
			"`packages/repo-contract-go` directly",
			"`vrooli contract validate`",
			"`vrooli contract show`",
			"`vrooli contract resolve scenario <name> --file service`",
			"`vrooli contract match-glob <pattern> <path>`",
			"`make validate-repo-contract` remains the CI/automation entrypoint",
			"## Landed Consumer Migrations",
			"`swarm-manager`",
		},
		ContributingDoc: []string{
			"# Contributing",
			"**Repo Contract**",
			"Do not add new repo-root heuristics",
			"`make validate-repo-contract`",
			"`vrooli contract show`",
		},
		AgentsDoc: []string{
			"# AGENTS.md",
			"## Repo Contract Adoption",
			"Do not add new independent repo-root detection logic",
			"Do not add new hard-coded canonical scenario path assembly",
			"Run `make validate-repo-contract`",
		},
		SkillDoc: []string{
			"# Cross Platform Readiness",
			"Use repo-contract-backed helpers",
			"`packages/repo-contract-go`",
		},
	}
}

func WriteRepoContractExceptions(t *testing.T, root string) {
	t.Helper()
	WriteJSON(t, filepath.Join(root, ".vrooli", "repo-contract-adoption-exceptions.json"), map[string]any{
		"version":    "1.0.0",
		"exceptions": []any{},
	})
}

func WriteRepoSupportDocs(t *testing.T, root string, docs RepoSupportDocs) {
	t.Helper()
	if len(docs.RepoContractDoc) > 0 {
		WriteFile(t, filepath.Join(root, "docs", "repo-contract.md"), strings.Join(docs.RepoContractDoc, "\n")+"\n")
	}
	if len(docs.ContributingDoc) > 0 {
		WriteFile(t, filepath.Join(root, "docs", "CONTRIBUTING.md"), strings.Join(docs.ContributingDoc, "\n")+"\n")
	}
	if len(docs.AgentsDoc) > 0 {
		WriteFile(t, filepath.Join(root, "AGENTS.md"), strings.Join(docs.AgentsDoc, "\n")+"\n")
	}
	if len(docs.SkillDoc) > 0 {
		WriteFile(t, filepath.Join(root, "scenarios", "prompt-manager", "store", "skills", "packs", "core", "cross-platform-readiness", "SKILL.md"), strings.Join(docs.SkillDoc, "\n")+"\n")
	}
}

func MustLoadDefaultContract(t *testing.T, root string) *repocontract.Contract {
	t.Helper()
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		t.Fatalf("LoadDefault(%s): %v", root, err)
	}
	return contract
}

func ensureObject(parent map[string]any, key string) map[string]any {
	if value, ok := parent[key].(map[string]any); ok {
		return value
	}
	created := map[string]any{}
	parent[key] = created
	return created
}

func cloneStringMapAny(input map[string]any) map[string]any {
	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}
