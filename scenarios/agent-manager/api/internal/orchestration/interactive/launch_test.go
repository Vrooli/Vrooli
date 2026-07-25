package interactive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/domain"
)

func TestSeedRelocatedHome_CopiesSeedsAndPreTrustsWorkingDir(t *testing.T) {
	userHome := t.TempDir()
	sharedCodex := filepath.Join(userHome, ".codex")
	if err := os.MkdirAll(sharedCodex, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sharedCodex, "auth.json"), []byte(`{"token":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sharedCodex, "config.toml"), []byte("model = 'x'\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	relocated := t.TempDir()
	workingDir := "/work/proj"
	spec, _ := specFor(domain.RunnerTypeCodex)
	if err := seedRelocatedHome(spec, relocated, workingDir, userHome); err != nil {
		t.Fatalf("seedRelocatedHome: %v", err)
	}

	// auth.json is a real copy, not a symlink (isolates the run from the shared
	// home other CLI processes contend on).
	info, err := os.Lstat(filepath.Join(relocated, "auth.json"))
	if err != nil {
		t.Fatalf("auth.json not seeded: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("auth.json should be a copy, not a symlink")
	}

	// config.toml is copied AND carries an appended trust entry for the exact
	// working dir, so codex boots past its directory-trust gate.
	cfg, err := os.ReadFile(filepath.Join(relocated, "config.toml"))
	if err != nil {
		t.Fatalf("config.toml not seeded: %v", err)
	}
	if !strings.Contains(string(cfg), "model = 'x'") {
		t.Error("config.toml should retain the copied shared contents")
	}
	if !strings.Contains(string(cfg), `[projects."/work/proj"]`) || !strings.Contains(string(cfg), `trust_level = "trusted"`) {
		t.Errorf("config.toml should pre-trust the working dir, got:\n%s", cfg)
	}
}

func TestSeedRelocatedHome_TrustEntryWhenConfigAbsent(t *testing.T) {
	userHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(userHome, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Shared home has no config.toml: the trust entry must still be written (the
	// append creates the file) so the dir is trusted.
	relocated := t.TempDir()
	spec, _ := specFor(domain.RunnerTypeCodex)
	if err := seedRelocatedHome(spec, relocated, "/w", userHome); err != nil {
		t.Fatalf("seedRelocatedHome: %v", err)
	}
	cfg, err := os.ReadFile(filepath.Join(relocated, "config.toml"))
	if err != nil {
		t.Fatalf("config.toml should be created for the trust entry: %v", err)
	}
	if !strings.Contains(string(cfg), `[projects."/w"]`) {
		t.Errorf("trust entry missing: %s", cfg)
	}
}

func TestSeedRelocatedHome_NoopForClaude(t *testing.T) {
	relocated := t.TempDir()
	spec, _ := specFor(domain.RunnerTypeClaudeCode)
	if err := seedRelocatedHome(spec, relocated, "/w", t.TempDir()); err != nil {
		t.Fatalf("claude seed should be a no-op, got: %v", err)
	}
	entries, _ := os.ReadDir(relocated)
	if len(entries) != 0 {
		t.Errorf("claude seed should write nothing, found %d entries", len(entries))
	}
}

func TestCleanupSeededHome_RemovesCredentialsKeepsTranscript(t *testing.T) {
	userHome := t.TempDir()
	sharedCodex := filepath.Join(userHome, ".codex")
	if err := os.MkdirAll(sharedCodex, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sharedCodex, "auth.json"), []byte(`{"token":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sharedCodex, "config.toml"), []byte("model = 'x'\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	relocated := t.TempDir()
	spec, _ := specFor(domain.RunnerTypeCodex)
	if err := seedRelocatedHome(spec, relocated, "/work/proj", userHome); err != nil {
		t.Fatalf("seedRelocatedHome: %v", err)
	}
	// A discovered transcript lives alongside the seeded credentials; cleanup must
	// leave it intact for diagnostics.
	rollout := filepath.Join(relocated, "sessions", "rollout-x.jsonl")
	if err := os.MkdirAll(filepath.Dir(rollout), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rollout, []byte(`{"type":"event_msg"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := cleanupSeededHome(spec, relocated); err != nil {
		t.Fatalf("cleanupSeededHome: %v", err)
	}

	for _, name := range spec.seedFiles {
		if _, err := os.Stat(filepath.Join(relocated, name)); !os.IsNotExist(err) {
			t.Errorf("seeded %s should be removed, stat err = %v", name, err)
		}
	}
	if _, err := os.Stat(rollout); err != nil {
		t.Errorf("transcript should survive cleanup: %v", err)
	}

	// Idempotent: a second cleanup (files already gone) is not an error.
	if err := cleanupSeededHome(spec, relocated); err != nil {
		t.Fatalf("second cleanupSeededHome should be a no-op: %v", err)
	}
}

func TestCleanupSeededHome_NoopForClaude(t *testing.T) {
	spec, _ := specFor(domain.RunnerTypeClaudeCode)
	// claude keeps a shared home (no seeding), so cleanup has nothing to remove and
	// an empty relocated path must be a no-op.
	if err := cleanupSeededHome(spec, ""); err != nil {
		t.Fatalf("claude cleanup should be a no-op: %v", err)
	}
}

func TestBuildLaunchCommand_Claude_NoHomeRelocation(t *testing.T) {
	cmd, err := BuildLaunchCommand(LaunchCommandParams{
		RunnerType: domain.RunnerTypeClaudeCode,
		BinaryPath: "/home/u/.local/bin/claude",
		TagEnvKey:  "CLAUDE_CODE_AGENT_TAG",
		Tag:        "run-42",
		WorkingDir: "/home/u/Vrooli",
		RunDir:     "/data/runs/run-42",
	})
	if err != nil {
		t.Fatalf("BuildLaunchCommand: %v", err)
	}
	want := "cd '/home/u/Vrooli' && CLAUDE_CODE_AGENT_TAG='run-42' '/home/u/.local/bin/claude'"
	if cmd != want {
		t.Fatalf("claude command:\n got: %s\nwant: %s", cmd, want)
	}
	if strings.Contains(cmd, "CLAUDE_CONFIG_DIR") {
		t.Error("claude launch must NOT relocate CLAUDE_CONFIG_DIR (design §4)")
	}
}

func TestBuildLaunchCommand_Codex_RelocatesHome(t *testing.T) {
	cmd, err := BuildLaunchCommand(LaunchCommandParams{
		RunnerType: domain.RunnerTypeCodex,
		BinaryPath: "/usr/bin/codex",
		TagEnvKey:  "CODEX_AGENT_TAG",
		Tag:        "run-7",
		WorkingDir: "/work/dir",
		RunDir:     "/data/runs/run-7",
	})
	if err != nil {
		t.Fatalf("BuildLaunchCommand: %v", err)
	}
	// The launch command relocates CODEX_HOME; directory trust and auth are seeded
	// into that home (see seedRelocatedHome), not passed on the command line.
	want := "cd '/work/dir' && CODEX_AGENT_TAG='run-7' CODEX_HOME='/data/runs/run-7/codex' '/usr/bin/codex'"
	if cmd != want {
		t.Fatalf("codex command:\n got: %s\nwant: %s", cmd, want)
	}
}

func TestBuildLaunchCommand_Grok_RelocatesHome(t *testing.T) {
	cmd, err := BuildLaunchCommand(LaunchCommandParams{
		RunnerType: domain.RunnerTypeGrok,
		BinaryPath: "/home/u/.local/bin/grok",
		TagEnvKey:  "GROK_AGENT_TAG",
		Tag:        "run-9",
		WorkingDir: "/work/dir",
		RunDir:     "/data/runs/run-9",
	})
	if err != nil {
		t.Fatalf("BuildLaunchCommand: %v", err)
	}
	if !strings.Contains(cmd, "GROK_HOME='/data/runs/run-9/grok'") {
		t.Fatalf("grok command missing GROK_HOME relocation: %s", cmd)
	}
	if !strings.Contains(cmd, "GROK_AGENT_TAG='run-9'") {
		t.Fatalf("grok command missing tag env: %s", cmd)
	}
}

func TestBuildLaunchCommand_ForwardsModelEffortAndQuotedPrompt(t *testing.T) {
	cases := []struct {
		runner domain.RunnerType
		effort string
		want   string
	}{
		{domain.RunnerTypeClaudeCode, "--effort 'high'", "--model 'claude-sonnet' --effort 'high' 'fix spaces; do not run $(bad)'"},
		{domain.RunnerTypeCodex, "-c 'model_reasoning_effort=high'", "--model 'claude-sonnet' -c 'model_reasoning_effort=high' 'fix spaces; do not run $(bad)'"},
		{domain.RunnerTypeGrok, "--effort 'high'", "--model 'claude-sonnet' --effort 'high' 'fix spaces; do not run $(bad)'"},
	}
	for _, tc := range cases {
		t.Run(string(tc.runner), func(t *testing.T) {
			cmd, err := BuildLaunchCommand(LaunchCommandParams{
				RunnerType: tc.runner, BinaryPath: "/usr/bin/agent", TagEnvKey: "AGENT_TAG", Tag: "run", WorkingDir: "/work", RunDir: "/runs/run",
				Model: "claude-sonnet", Effort: domain.EffortHigh, InitialPrompt: "fix spaces; do not run $(bad)",
			})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(cmd, tc.want) || !strings.Contains(cmd, tc.effort) {
				t.Fatalf("command missing resolved controls: %s", cmd)
			}
		})
	}
}

func TestBuildLaunchCommand_OpenCodeDescoped(t *testing.T) {
	_, err := BuildLaunchCommand(LaunchCommandParams{
		RunnerType: domain.RunnerTypeOpenCode,
		BinaryPath: "/x/opencode",
		TagEnvKey:  "OPENCODE_AGENT_TAG",
		Tag:        "run-1",
		WorkingDir: "/w",
	})
	if err == nil {
		t.Fatal("expected error for opencode (descoped from interactive v1)")
	}
}

func TestBuildLaunchCommand_QuotesInjectionAttempts(t *testing.T) {
	cmd, err := BuildLaunchCommand(LaunchCommandParams{
		RunnerType: domain.RunnerTypeCodex,
		BinaryPath: "/usr/bin/codex",
		TagEnvKey:  "CODEX_AGENT_TAG",
		Tag:        "a'; rm -rf / #",
		WorkingDir: "/work dir/with space",
		RunDir:     "/data/runs/x",
	})
	if err != nil {
		t.Fatalf("BuildLaunchCommand: %v", err)
	}
	// The single quote in the tag must be escaped, not left to close the quote.
	if !strings.Contains(cmd, `CODEX_AGENT_TAG='a'\''; rm -rf / #'`) {
		t.Fatalf("tag not safely quoted: %s", cmd)
	}
	if !strings.Contains(cmd, "cd '/work dir/with space'") {
		t.Fatalf("workdir with space not quoted: %s", cmd)
	}
}

func TestBuildLaunchCommand_RejectsMissingFields(t *testing.T) {
	cases := map[string]LaunchCommandParams{
		"missing binary":  {RunnerType: domain.RunnerTypeClaudeCode, TagEnvKey: "K", WorkingDir: "/w"},
		"missing workdir": {RunnerType: domain.RunnerTypeClaudeCode, BinaryPath: "/b", TagEnvKey: "K"},
		"missing tag key": {RunnerType: domain.RunnerTypeClaudeCode, BinaryPath: "/b", WorkingDir: "/w"},
		"codex missing rundir": {
			RunnerType: domain.RunnerTypeCodex, BinaryPath: "/b", TagEnvKey: "K", WorkingDir: "/w",
		},
	}
	for name, p := range cases {
		if _, err := BuildLaunchCommand(p); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}
