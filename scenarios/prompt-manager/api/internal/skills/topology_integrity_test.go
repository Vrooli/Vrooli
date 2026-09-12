package skills

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"prompt-manager/internal/store"
)

func TestScenarioPackSkillsResolveThroughReadIndex(t *testing.T) {
	if !contains(Folders, "scenario") {
		t.Fatal(`Folders must include "scenario" so scenario-owned skills are readable`)
	}
	if contains(WritableFolders, "scenario") {
		t.Fatal(`WritableFolders must not include "scenario"`)
	}

	repoRoot := findRepoRoot(t)
	scenarioRoot := filepath.Join(repoRoot, "scenarios")
	configRoot := t.TempDir()
	fileStore := store.NewFileSkillStoreWithScenarioRoots(configRoot, scenarioRoot)
	indexed, err := loadIndexedSkills(NewStoreAdapter(fileStore, store.NewFileContentIO()))
	if err != nil {
		t.Fatalf("load scenario skill index: %v", err)
	}

	want := 0
	entries, err := os.ReadDir(scenarioRoot)
	if err != nil {
		t.Fatalf("read scenarios: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		matches, _ := filepath.Glob(filepath.Join(scenarioRoot, entry.Name(), "skills", "*", "SKILL.md"))
		want += len(matches)
	}
	if want == 0 {
		t.Fatal("expected scenario-owned skills in the repository")
	}

	seen := 0
	for _, skill := range indexed {
		if skill.folder != "scenario" {
			continue
		}
		seen++
		if matches := resolveIdentifier(skill.meta.ID, "id", indexed); len(matches) != 1 {
			t.Fatalf("scenario skill %q resolves to %d entries, want exactly 1", skill.meta.ID, len(matches))
		}
	}
	if seen != want {
		t.Fatalf("scenario skill index contains %d skills, found %d scenario SKILL.md files", seen, want)
	}
}

func TestSkillIDHasOneAuthoritativeSource(t *testing.T) {
	repoRoot := findRepoRoot(t)
	patterns := []string{
		filepath.Join(repoRoot, "scenarios", "*", "skills", "*", "SKILL.md"),
		filepath.Join(repoRoot, "scenarios", "prompt-manager", "store", "skills", "packs", "*", "*", "SKILL.md"),
	}
	sources := make(map[string]string)
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatalf("glob skill sources: %v", err)
		}
		for _, match := range matches {
			id := filepath.Base(filepath.Dir(match))
			if previous, exists := sources[id]; exists {
				t.Fatalf("skill %q has duplicate authoritative sources: %s and %s", id, previous, match)
			}
			sources[id] = match
		}
	}
}

func TestAgentsDocumentHasOneHeadingAndAllCriticalRules(t *testing.T) {
	root := findRepoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	content := string(data)
	if got := strings.Count(content, "# AGENTS.md"); got != 1 {
		t.Fatalf("AGENTS.md headings = %d, want 1", got)
	}
	for i := 1; i <= 7; i++ {
		pattern := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(strconv.Itoa(i)) + `\. `)
		if !pattern.MatchString(content) {
			t.Errorf("AGENTS.md is missing Critical Rule %d", i)
		}
	}
	if !strings.Contains(content, "prompt-manager skill read program-runtime") {
		t.Error("AGENTS.md must route recurring work through prompt-manager skill read program-runtime")
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repository root")
		}
		dir = parent
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
