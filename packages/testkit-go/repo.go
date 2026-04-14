package testkitgo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

type RepoFixture struct {
	Root        string
	Home        string
	ScenarioDir string
}

type RepoSupportDocs struct {
	RepoContractDoc []string
	ContributingDoc []string
	AgentsDoc       []string
	SkillDoc        []string
}

func NewRepoFixture(t *testing.T, opts ...RepoFixtureOption) RepoFixture {
	t.Helper()
	fixture := RepoFixture{
		Root:        t.TempDir(),
		Home:        t.TempDir(),
		ScenarioDir: "scenarios",
	}
	for _, opt := range opts {
		opt(&fixture)
	}
	return fixture
}

type RepoFixtureOption func(*RepoFixture)

func WithScenarioDir(name string) RepoFixtureOption {
	return func(fixture *RepoFixture) {
		if strings.TrimSpace(name) != "" {
			fixture.ScenarioDir = name
		}
	}
}

func (fixture RepoFixture) WriteRepoContract(t *testing.T) {
	t.Helper()
	WriteRepoContract(t, fixture.Root, fixture.ScenarioDir)
}

func (fixture RepoFixture) WriteRepoSupportDocs(t *testing.T, docs RepoSupportDocs) {
	t.Helper()
	WriteRepoSupportDocs(t, fixture.Root, docs)
}

func (fixture RepoFixture) WriteScenarioStub(t *testing.T, name string) {
	t.Helper()
	WriteScenarioStub(t, fixture.Root, fixture.ScenarioDir, name)
}

func (fixture RepoFixture) WriteResourceStub(t *testing.T, name string) {
	t.Helper()
	WriteResourceStub(t, fixture.Root, name)
}

func ProjectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func WriteRepoContract(t *testing.T, root, scenarioDir string) {
	t.Helper()
	projectRoot := ProjectRoot(t)
	contractPath := repocontractmeta.ContractPath(projectRoot)
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

	sandbox := ensureObject(doc, "sandbox")
	sandbox["scenario_scope_prefix"] = scenarioDir + "/"

	dirsToCreate := []string{".vrooli", scenarioDir, "resources", "packages", "cmd", "internal"}
	if rootMarkers, ok := rootDoc["markers"].(map[string]any); ok {
		if values, ok := rootMarkers["required_dirs"].([]any); ok && len(values) > 0 {
			dirsToCreate = dirsToCreate[:0]
			for _, value := range values {
				text, _ := value.(string)
				if strings.TrimSpace(text) != "" {
					dirsToCreate = append(dirsToCreate, text)
				}
			}
		}
	}

	for _, dir := range dirsToCreate {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	WriteFile(t, filepath.Join(root, "go.mod"), "module test\n")
	WriteJSON(t, repocontractmeta.ContractPath(root), doc)
}

func DefaultRepoSupportDocs() RepoSupportDocs {
	return RepoSupportDocs{
		RepoContractDoc: []string{
			"# Repo Contract",
			"## Adoption Rules",
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

func WriteScenarioStub(t *testing.T, root, scenarioDir, name string) {
	t.Helper()
	WriteJSON(t, filepath.Join(root, scenarioDir, name, repocontractmeta.ServiceManifestPathname), map[string]any{
		"service": map[string]any{
			"name": name,
		},
	})
}

func WriteResourceStub(t *testing.T, root, name string) {
	t.Helper()
	WriteJSON(t, repocontractmeta.ResourceManifestPath(root, name), map[string]any{
		"name": name,
	})
}

func ContainsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func ensureObject(parent map[string]any, key string) map[string]any {
	if value, ok := parent[key].(map[string]any); ok {
		return value
	}
	created := map[string]any{}
	parent[key] = created
	return created
}
