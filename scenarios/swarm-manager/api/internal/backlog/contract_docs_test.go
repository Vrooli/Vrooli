package backlog

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRootFromContractDocsTest(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."))
}

func readRepoFile(t *testing.T, root, rel string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

func TestAuthoritativeDocsRejectLegacyScopeAndBatchMilestoneFlag(t *testing.T) {
	root := repoRootFromContractDocsTest(t)

	files := []string{
		"scenarios/prompt-manager/store/skills/packs/core/swarm-manager-meta-orchestrator/SKILL.md",
		"scenarios/prompt-manager/store/skills/packs/core/swarm-manager-backlog-tools/SKILL.md",
		"scenarios/swarm-manager/docs/reference/cli-commands.md",
	}

	for _, rel := range files {
		content := readRepoFile(t, root, rel)
		if strings.Contains(content, `"scope":`) {
			t.Fatalf("%s still contains legacy backlog scope examples", rel)
		}
		for _, line := range strings.Split(content, "\n") {
			if strings.Contains(line, "backlog batch-create") && strings.Contains(line, "--milestone") {
				t.Fatalf("%s still teaches the legacy backlog batch-create --milestone flag", rel)
			}
		}
	}
}

func TestAuthoritativeDocsDescribeCanonicalBacklogImport(t *testing.T) {
	root := repoRootFromContractDocsTest(t)

	metaSkill := readRepoFile(t, root, "scenarios/prompt-manager/store/skills/packs/core/swarm-manager-meta-orchestrator/SKILL.md")
	for _, required := range []string{
		"acceptance_allow",
		"acceptance_deny",
		`"milestones": [`,
		"--preview",
		"orchestration-summary.md",
	} {
		if !strings.Contains(metaSkill, required) {
			t.Fatalf("meta-orchestrator skill missing %q", required)
		}
	}

	backlogTools := readRepoFile(t, root, "scenarios/prompt-manager/store/skills/packs/core/swarm-manager-backlog-tools/SKILL.md")
	for _, required := range []string{
		"acceptance_allow",
		"acceptance_deny",
		"--preview",
		"goals targets-add",
	} {
		if !strings.Contains(backlogTools, required) {
			t.Fatalf("backlog-tools skill missing %q", required)
		}
	}

	manifest := readRepoFile(t, root, "scenarios/swarm-manager/docs/manifest.json")
	for _, required := range []string{
		"reference/api-endpoints.md",
		"reference/cli-commands.md",
	} {
		if !strings.Contains(manifest, required) {
			t.Fatalf("docs manifest missing %q", required)
		}
	}
}
