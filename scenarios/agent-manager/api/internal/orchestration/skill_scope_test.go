package orchestration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestRunnerSkillScopesArePrivateAndReapable(t *testing.T) {
	root := t.TempDir()
	first, second := uuid.New(), uuid.New()
	firstEnv, err := PrepareRunnerSkillScope(root, first, domain.RunnerTypeClaudeCode)
	if err != nil {
		t.Fatal(err)
	}
	secondEnv, err := PrepareRunnerSkillScope(root, second, domain.RunnerTypeClaudeCode)
	if err != nil {
		t.Fatal(err)
	}
	if firstEnv["CLAUDE_CONFIG_DIR"] == secondEnv["CLAUDE_CONFIG_DIR"] {
		t.Fatal("concurrent runs share a Claude config directory")
	}
	if _, err := os.Stat(firstEnv["CLAUDE_CONFIG_DIR"]); err != nil {
		t.Fatal(err)
	}
	settings, err := os.ReadFile(filepath.Join(firstEnv["CLAUDE_CONFIG_DIR"], "settings.json"))
	if err != nil {
		t.Fatalf("private Claude settings missing: %v", err)
	}
	if !strings.Contains(string(settings), "prompt-manager skill activation-hook") {
		t.Fatalf("private Claude settings missing activation hook: %s", settings)
	}
	if err := CleanupRunnerSkillScope(root, first, domain.RunnerTypeClaudeCode); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(firstEnv["CLAUDE_CONFIG_DIR"]); !os.IsNotExist(err) {
		t.Fatalf("scope survived cleanup: %v", err)
	}
	if err := SweepOrphanedSkillScopes(root, map[uuid.UUID]bool{second: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, first.String(), skillScopeDirName)); !os.IsNotExist(err) {
		t.Fatalf("orphan scope survived sweep: %v", err)
	}
}

func TestClaudeSkillScopePreservesSharedSettingsSnapshot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{"alwaysThinkingEnabled":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	env, err := PrepareRunnerSkillScope(t.TempDir(), uuid.New(), domain.RunnerTypeClaudeCode)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(env["CLAUDE_CONFIG_DIR"], "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "alwaysThinkingEnabled") || !strings.Contains(string(data), "prompt-manager skill activation-hook") {
		t.Fatalf("private settings did not preserve shared settings and add telemetry: %s", data)
	}
}
