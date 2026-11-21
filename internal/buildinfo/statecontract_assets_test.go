package buildinfo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestHarnessAssetContractTracksPersistentReadersAndHomeLayouts(t *testing.T) {
	root := t.TempDir()
	writeStateContractFixture(t, root)
	writeTestFile(t, root, "go.mod", "module github.com/vrooli/vrooli\n")
	path := filepath.Join(root, filepath.FromSlash(harnessStateContractAsset))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	makeDescriptor := func(resource, id, display string, sessionEnv []string, homeKind, env, stateDir, sharedDir, sessionsEnv, layout, reader string) map[string]any {
		home := map[string]string{"kind": homeKind}
		for key, value := range map[string]string{"env": env, "state_dir": stateDir, "shared_dir": sharedDir, "sessions_env": sessionsEnv, "layout": layout} {
			if value != "" {
				home[key] = value
			}
		}
		features := map[string]string{}
		if reader != "" {
			features["context_window_source"] = reader
		}
		return map[string]any{"resource": resource, "id": id, "display_name": display, "session_env": sessionEnv, "home_isolation": home, "features": features}
	}
	claude := func(display, reader string) map[string]any {
		return makeDescriptor("claude-code", "claude", display, []string{"CLAUDE_SESSION_ID"}, "none", "", "", "", "", "", reader)
	}
	codex := func(display, stateDir string) map[string]any {
		return makeDescriptor("codex", "codex", display, []string{"CODEX_SESSION_ID"}, "env", "CODEX_HOME", stateDir, "shared", "CODEX_SESSIONS", "mirror", "codex-models-cache")
	}
	write := func(descriptors ...map[string]any) string {
		t.Helper()
		data, err := json.Marshal(descriptors)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		digest, err := PersistentStateCodecStamp(root)
		if err != nil {
			t.Fatal(err)
		}
		return digest
	}

	base := write(claude("Claude", "claude-statusline"), codex("Codex", "state"))
	if got := write(claude("Claude CLI", "claude-statusline"), codex("Codex CLI", "state")); got != base {
		t.Fatal("display-only edits changed the persisted-state compatibility projection")
	}
	if got := write(codex("Codex", "state"), claude("Claude", "claude-statusline")); got != base {
		t.Fatal("descriptor ordering changed the persisted-state compatibility projection")
	}
	if got := write(claude("Claude", "claude-statusline"), codex("Codex", "state-v2")); got == base {
		t.Fatal("home-state path change kept the persisted-state compatibility projection")
	}
	if got := write(claude("Claude", "claude-statusline-v2"), codex("Codex", "state")); got == base {
		t.Fatal("persisted-model reader change kept the compatibility projection")
	}
	changedSessionEnv := codex("Codex", "state")
	changedSessionEnv["session_env"] = []string{"CODEX_CONVERSATION_ID"}
	if got := write(claude("Claude", "claude-statusline"), changedSessionEnv); got == base {
		t.Fatal("session identifier selector change kept the compatibility projection")
	}
}
