package orchestration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// The repository's resource manifests decide where each harness discovers
// skills and what relocates that directory; Agent Manager restates neither.
func TestRunnerSkillsDirFollowsTheHarnessManifests(t *testing.T) {
	scope := "/runs/" + uuid.NewString() + "/runtime"
	for runner, tc := range map[domain.RunnerType]struct {
		env  map[string]string
		want string
	}{
		domain.RunnerTypeClaudeCode: {map[string]string{"CLAUDE_CONFIG_DIR": scope + "/claude"}, scope + "/claude/skills"},
		domain.RunnerTypeCodex:      {map[string]string{"CODEX_HOME": scope + "/codex"}, scope + "/codex/skills"},
		domain.RunnerTypeGrok:       {map[string]string{"GROK_HOME": scope + "/grok"}, scope + "/grok/skills"},
		domain.RunnerTypeOpenCode:   {map[string]string{"XDG_CONFIG_HOME": scope + "/opencode"}, scope + "/opencode/opencode/skills"},
	} {
		got, err := RunnerSkillsDir(runner, tc.env)
		if err != nil || got != filepath.FromSlash(tc.want) {
			t.Errorf("RunnerSkillsDir(%s) = %q, %v; want %q", runner, got, err, tc.want)
		}
	}
	// OPENCODE_CONFIG_DIR only adds a config root; it is not the relocation.
	if _, err := RunnerSkillsDir(domain.RunnerTypeOpenCode, map[string]string{"OPENCODE_CONFIG_DIR": scope}); err == nil {
		t.Error("OpenCode skills were placed under OPENCODE_CONFIG_DIR, which does not isolate the global skills directory")
	}
}

// agy reads skills only from ~/.gemini/config/skills and no variable moves it,
// so a skill pack for an Antigravity run fails loudly instead of silently
// missing the run, and no fake scope variable is set.
func TestAntigravitySkillScopingIsRefusedNotFaked(t *testing.T) {
	env, err := PrepareRunnerSkillScope(t.TempDir(), uuid.New(), domain.RunnerTypeAntigravity)
	if err != nil || len(env) != 0 {
		t.Fatalf("Antigravity scope env = %v, %v; want none", env, err)
	}
	if _, err := RunnerSkillsDir(domain.RunnerTypeAntigravity, map[string]string{"ANTIGRAVITY_STATE_DIR": "/runs/x"}); err == nil || !strings.Contains(err.Error(), "unsupported for antigravity") {
		t.Fatalf("RunnerSkillsDir(antigravity) error = %v, want an unsupported refusal", err)
	}
}

// XDG_CONFIG_HOME moves every XDG-aware program, so the private root mirrors
// the operator's ~/.config: other programs and OpenCode's own configuration
// stay visible, and only OpenCode's skills directory is private. Cleanup
// removes the links, never the operator's files.
func TestOpenCodeSkillScopeIsolatesOnlyTheSkillsDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(home, ".config", rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("gh/hosts.yml", "github.com: {}")
	write("opencode/opencode.json", `{"model":"x"}`)
	write("opencode/skills/operator-global/SKILL.md", "operator skill")

	root, runID := t.TempDir(), uuid.New()
	env, err := PrepareRunnerSkillScope(root, runID, domain.RunnerTypeOpenCode)
	if err != nil {
		t.Fatal(err)
	}
	if _, set := env["OPENCODE_CONFIG_DIR"]; set {
		t.Fatal("the scope still sets OPENCODE_CONFIG_DIR")
	}
	config := env["XDG_CONFIG_HOME"]
	if !strings.HasPrefix(config, filepath.Join(root, runID.String())) {
		t.Fatalf("XDG_CONFIG_HOME %q is not run-private", config)
	}
	for rel, want := range map[string]string{"gh/hosts.yml": "github.com: {}", "opencode/opencode.json": `{"model":"x"}`} {
		if data, err := os.ReadFile(filepath.Join(config, rel)); err != nil || string(data) != want {
			t.Errorf("%s through the private config home = %q, %v", rel, data, err)
		}
	}
	skills, err := RunnerSkillsDir(domain.RunnerTypeOpenCode, env)
	if err != nil {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(skills); err != nil || len(entries) != 0 {
		t.Fatalf("private skills directory = %v, %v; want empty, without the operator's global skills", entries, err)
	}
	if _, err := PrepareRunnerSkillScope(root, runID, domain.RunnerTypeOpenCode); err != nil {
		t.Fatalf("continuation could not refresh the scope: %v", err)
	}
	if err := CleanupRunnerSkillScope(root, runID, domain.RunnerTypeOpenCode); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"gh/hosts.yml", "opencode/opencode.json", "opencode/skills/operator-global/SKILL.md"} {
		if _, err := os.Stat(filepath.Join(home, ".config", rel)); err != nil {
			t.Errorf("cleanup removed the operator's %s: %v", rel, err)
		}
	}
}

// Manifest shapes the loader must handle: a harness-home key, an XDG key, a
// pinned directory, and a skills path outside the relocated root.
func TestLoadHarnessSkillStorageReadsRelocationFromManifests(t *testing.T) {
	dir := t.TempDir()
	manifest := func(name, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name, "resource.json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	manifest("homeful", `{"name":"homeful","storage":{"entries":{"home":{"path":"$USER_HOME/.homeful"},"skills":{"path":"$USER_HOME/.homeful/agent/skills","relocation":{"key":"HOMEFUL_HOME"}}}}}`)
	manifest("xdg", `{"name":"xdg","storage":{"entries":{"skills":{"path":"$USER_HOME/.local/share/xdg/skills","relocation":{"key":"XDG_DATA_HOME"}}}}}`)
	manifest("pinned", `{"name":"pinned","storage":{"entries":{"skills":{"path":"$USER_HOME/.pinned/skills","rationale":"no variable"}}}}`)
	manifest("other", `{"name":"other","storage":{"entries":{"data":{"path":{"unusual":"shape"}}}}}`)
	storage, err := loadHarnessSkillStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := storage["homeful"]; got.RelocationKey != "HOMEFUL_HOME" || got.RelativePath != filepath.Join("agent", "skills") || got.SharedBaseDirectory() {
		t.Errorf("homeful = %+v", got)
	}
	if got := storage["xdg"]; got.RelativePath != filepath.Join("xdg", "skills") || !got.SharedBaseDirectory() {
		t.Errorf("xdg = %+v", got)
	}
	if got := storage["pinned"]; got.Relocatable() || got.Rationale != "no variable" {
		t.Errorf("pinned = %+v", got)
	}
	if _, ok := storage["other"]; ok {
		t.Error("a resource without a skills entry was loaded")
	}
	manifest("broken", `{"name":"broken","storage":{"entries":{"skills":{"path":"/elsewhere/skills","relocation":{"key":"BROKEN_HOME"}}}}}`)
	if _, err := loadHarnessSkillStorage(dir); err == nil {
		t.Error("a skills path outside its relocated root was accepted")
	}
}
