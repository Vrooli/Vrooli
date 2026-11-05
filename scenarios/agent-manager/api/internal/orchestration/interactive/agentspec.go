// Package interactive is the execution substrate for interactive runs: runs
// whose agent CLI runs inside a web-console (persistent/tmux) session rather
// than as an agent-manager-owned child process fed through a codec stdout pipe.
//
// This is the parallel execution path decided in
// scenarios/agent-manager/docs/interactive-runner-design.md §1: it does NOT go
// through runner.Launcher/LaunchedProcess. It creates a web-console session,
// lets the server paste+run an env-scoped interactive launch command, discovers
// the agent-owned on-disk transcript, and (in later phases) tails it with
// runner.Consume + the codec transcript parser. This package owns the session
// lifecycle (create / stop) and the launch-command + transcript-discovery
// contract; transcript event extraction (Phase 3) and completion detection
// (Phase 4) plug in on top of the transcript path this package resolves.
package interactive

import (
	"fmt"

	"agent-manager/internal/domain"
)

// agentSpec captures the per-agent facts that make an interactive launch
// deterministic: where the agent writes its session home, and thus where its
// transcript lands.
//
//   - codex/grok relocate their session home to a run-scoped directory
//     agent-manager owns (homeEnvVar=<VAR>, homeSubdir under the run dir), so
//     the run's rollout/updates file is the only transcript under that home —
//     unambiguous discovery, no cross-run race.
//   - claude has no home relocation (homeEnvVar==""): CLAUDE_CONFIG_DIR would
//     relocate auth and trigger OAuth re-onboarding (design §4), so claude uses
//     the shared, authenticated ~/.claude and its transcript is discovered by
//     cwd-slug + newest-after-launch under ~/.claude/projects.
//
// opencode is intentionally absent — it has no tailable transcript file
// (SQLite-backed) and is descoped from interactive v1 (design §6). Launch
// rejects any runner type without a spec.
type agentSpec struct {
	runnerType domain.RunnerType
	homeEnvVar string // env var relocating the agent's session home; "" for claude
	homeSubdir string // subdir under the run dir for the relocated home; "" for claude

	// sharedHomeBase is the directory under the user's home that holds the
	// agent's real (authenticated) session home. When a run relocates the home to
	// a fresh run-scoped dir, seedFiles are COPIED from here so the CLI stays
	// logged in instead of dropping to a sign-in wall (design §4). "" = no seeding
	// (claude, which keeps the shared home outright).
	//
	// The seed is a copy, not a symlink: the CLI rewrites these files at launch
	// (codex persists trust/NUX state into config.toml), and a symlink would route
	// those writes back into the shared home where dozens of concurrent CLI
	// processes on a busy host contend on the same file — an observed source of
	// flaky launches. A private copy fully isolates the run.
	sharedHomeBase string
	// seedFiles are copied from sharedHomeBase into the relocated home before
	// launch (auth.json for credentials, config.toml for model/NUX settings).
	// Missing sources are skipped, not fatal.
	seedFiles []string
	// trustConfigFile / dirTrustTOML pre-trust the working dir by appending a TOML
	// entry to the named file in the relocated home, so the agent boots past its
	// first-run "trust this directory?" gate deterministically (the operator is
	// not there to accept it). Both empty/nil = no such gate. Preferred over a
	// command-line override because the agent reads directory trust from its
	// on-disk config, not always from a `-c` merge.
	trustConfigFile string
	dirTrustTOML    func(workingDir string) string
}

// relocatesHome reports whether this agent gets a run-scoped session home.
func (s agentSpec) relocatesHome() bool { return s.homeEnvVar != "" && s.homeSubdir != "" }

// agentSpecs is the interactive-support table. Membership == "supported in
// interactive v1". claude, codex, grok are in; opencode is deliberately out.
var agentSpecs = map[domain.RunnerType]agentSpec{
	domain.RunnerTypeClaudeCode: {runnerType: domain.RunnerTypeClaudeCode},
	domain.RunnerTypeCodex: {
		runnerType:      domain.RunnerTypeCodex,
		homeEnvVar:      "CODEX_HOME",
		homeSubdir:      "codex",
		sharedHomeBase:  ".codex",
		seedFiles:       []string{"auth.json", "config.toml"},
		trustConfigFile: "config.toml",
		dirTrustTOML:    codexTrustTOML,
	},
	domain.RunnerTypeGrok: {
		runnerType:     domain.RunnerTypeGrok,
		homeEnvVar:     "GROK_HOME",
		homeSubdir:     "grok",
		sharedHomeBase: ".grok",
	},
}

// codexTrustTOML is the config entry that marks workingDir trusted for codex, in
// codex's own persisted `[projects."<dir>"] trust_level = "trusted"` shape.
// Appending it to the run's private config.toml boots codex straight to its
// input prompt. Verified live against codex v0.144.0.
func codexTrustTOML(workingDir string) string {
	return fmt.Sprintf("\n[projects.%q]\ntrust_level = %q\n", workingDir, "trusted")
}

// SupportsInteractive reports whether a runner type can run in interactive mode.
func SupportsInteractive(rt domain.RunnerType) bool {
	_, ok := agentSpecs[rt]
	return ok
}

// specFor returns the interactive spec for a runner type, or ok=false when the
// runner is not supported in interactive mode (e.g. opencode).
func specFor(rt domain.RunnerType) (agentSpec, bool) {
	spec, ok := agentSpecs[rt]
	return spec, ok
}
