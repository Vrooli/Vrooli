//go:build !windows

package main

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

// A long-lived tmux server keeps the environment it was born with, and every
// new session takes the server's global environment. REGRESSION 2026-11-08:
// the live console server, started 2026-09-02, handed every new operator pane
// web-console's own process identity (VROOLI_PROCESS_ID, the runtime instance,
// SCENARIO_NAME, LIFECYCLE_PHASE) and a Codex session's markers. The API start
// reconcile must clear everything an operator terminal must not inherit and
// leave ordinary variables alone.
func TestReconcileTmuxGlobalEnvironmentClearsWhatTerminalsMustNotInherit(t *testing.T) {
	requireTmux(t)
	if out, err := tmuxCmd("new-session", "-d", "-s", "wc-scrub-probe", "sleep 60").CombinedOutput(); err != nil {
		t.Fatalf("start isolated tmux server: %v: %s", err, out)
	}
	t.Cleanup(func() { _ = tmuxCmd("kill-server").Run() })

	agentKeys := cliutil.AmbientAgentEnvironment()
	if len(agentKeys) < 2 {
		t.Fatalf("cli-core lists %d ambient agent keys", len(agentKeys))
	}
	cleared := append([]string{
		"VROOLI_PROCESS_ID",
		"VROOLI_RUNTIME_INSTANCE_ID",
		"VROOLI_RUNTIME_GENERATION",
		"VROOLI_SCENARIO_DIR",
		"SCENARIO_NAME",
		"SCENARIO_PATH",
		"LIFECYCLE_PHASE",
		"API_PORT",
		"CODEX_SESSION_ID",
		"CODEX_MANAGED_BY_NPM",
		"CLAUDE_CODE_ENTRYPOINT",
	}, agentKeys[:2]...)
	kept := []string{"WC_SCRUB_KEEP", "GOPATH", "LANG"}
	for _, key := range append(append([]string(nil), cleared...), kept...) {
		if out, err := tmuxCmd("set-environment", "-g", key, "1").CombinedOutput(); err != nil {
			t.Fatalf("seed %s: %v: %s", key, err, out)
		}
	}

	reconcileTmuxGlobalEnvironment()

	out, err := tmuxCmd("show-environment", "-g").Output()
	if err != nil {
		t.Fatalf("show-environment: %v", err)
	}
	global := "\n" + string(out)
	for _, key := range cleared {
		if strings.Contains(global, "\n"+key+"=") {
			t.Errorf("%s survived the reconcile in the tmux global environment", key)
		}
	}
	for _, key := range kept {
		if !strings.Contains(global, "\n"+key+"=1") {
			t.Errorf("the reconcile removed %s, which an operator terminal may inherit", key)
		}
	}
}

func TestReconcileTmuxGlobalEnvironmentWithoutAServerIsHarmless(t *testing.T) {
	requireTmux(t)
	reconcileTmuxGlobalEnvironment()
}

// The pane filter and the server reconcile share one rule, so a variable the
// filter drops can never reach a pane through the server's global environment.
func TestTerminalInheritsEnvFollowsTheForeignScenarioBoundary(t *testing.T) {
	for name, want := range map[string]bool{
		"HOME":                        true,
		"PATH":                        true,
		"GOPATH":                      true,
		"SCENARIO_NAME":               false,
		"VROOLI_SCENARIO_DIR":         false,
		"VROOLI_PROCESS_ID":           false,
		"LIFECYCLE_PHASE":             false,
		"UI_PORT":                     false,
		"CODEX_SESSION_ID":            false,
		"CODEX_CI":                    false,
		"VROOLI_AGENT_IDENTITY_TOKEN": false,
		"TMUX":                        false,
	} {
		if got := terminalInheritsEnv(name); got != want {
			t.Errorf("terminalInheritsEnv(%q) = %v, want %v", name, got, want)
		}
		filtered := filterServiceEnv([]string{name + "=x"})
		if (len(filtered) == 1) != want {
			t.Errorf("filterServiceEnv keeps %q = %v, want %v", name, len(filtered) == 1, want)
		}
	}
}
