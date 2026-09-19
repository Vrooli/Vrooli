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

func TestOpenCodeRunDataHomeSeesSharedProviderAuthAndSurvivesCleanup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	shared := filepath.Join(home, ".local", "share", "opencode")
	if err := os.MkdirAll(shared, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shared, "auth.json"), []byte(`{"opencode-go":{"type":"api"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	root, runID := t.TempDir(), uuid.New()
	env, err := PrepareRunnerSkillScope(root, runID, domain.RunnerTypeOpenCode)
	if err != nil {
		t.Fatal(err)
	}
	dataHome := env["XDG_DATA_HOME"]
	if !strings.HasPrefix(dataHome, filepath.Join(root, runID.String(), RuntimeDirName)) {
		t.Fatalf("OpenCode data home %q is outside the run runtime folder", dataHome)
	}
	auth, err := os.ReadFile(filepath.Join(dataHome, "opencode", "auth.json"))
	if err != nil {
		t.Fatalf("run cannot see provider credentials: %v", err)
	}
	if !strings.Contains(string(auth), "opencode-go") {
		t.Fatal("run does not see the expected provider auth")
	}
	sessionStore := filepath.Join(dataHome, "opencode", "opencode.db")
	if err := os.WriteFile(sessionStore, []byte("session-store-fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CleanupRunnerSkillScope(root, runID, domain.RunnerTypeOpenCode); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dataHome, "opencode")); err != nil {
		t.Fatalf("terminal cleanup removed the session store that run continue needs: %v", err)
	}
	resumedEnv, err := PrepareRunnerSkillScope(root, runID, domain.RunnerTypeOpenCode)
	if err != nil {
		t.Fatal(err)
	}
	if resumedEnv["XDG_DATA_HOME"] != dataHome {
		t.Fatal("continuation changed the run's data home")
	}
	if data, err := os.ReadFile(sessionStore); err != nil || string(data) != "session-store-fixture" {
		t.Fatal("cleanup and continuation did not preserve session store bytes")
	}
}

func TestOpenCodeRunDataHomeUsesSelectedSharedAuthRoot(t *testing.T) {
	for _, present := range []bool{true, false} {
		name := "selected auth missing"
		if present {
			name = "selected auth present"
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			selected := t.TempDir()
			t.Setenv("XDG_DATA_HOME", selected)
			defaultAuth := filepath.Join(os.Getenv("HOME"), ".local", "share", "opencode", "auth.json")
			selectedAuth := filepath.Join(selected, "opencode", "auth.json")
			for _, p := range []string{defaultAuth, selectedAuth} {
				if p == selectedAuth && !present {
					continue
				}
				if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(`{"opencode-go":{"type":"api"}}`), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			root, runID := t.TempDir(), uuid.New()
			env, err := PrepareRunnerSkillScope(root, runID, domain.RunnerTypeOpenCode)
			if err != nil {
				t.Fatal(err)
			}
			destination := filepath.Join(env["XDG_DATA_HOME"], "opencode", "auth.json")
			if !present {
				if _, err := os.Lstat(destination); !os.IsNotExist(err) {
					t.Fatal("missing selected auth must not expose default-root credentials")
				}
				return
			}
			if target, err := os.Readlink(destination); err != nil || target != selectedAuth {
				t.Fatal("run auth must link to the selected shared store without copying credentials")
			}
			// Removing the selected login before continuation must not leave a
			// stale projection or silently switch to the default store.
			if err := os.Remove(selectedAuth); err != nil {
				t.Fatal(err)
			}
			if _, err := PrepareRunnerSkillScope(root, runID, domain.RunnerTypeOpenCode); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(destination); !os.IsNotExist(err) {
				t.Fatal("continuation retained auth after the selected login was removed")
			}
		})
	}
}
