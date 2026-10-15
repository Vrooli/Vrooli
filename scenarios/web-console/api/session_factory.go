package main

import (
	"log"
	"time"

	"github.com/vrooli/cli-core/cliutil"

	"web-console/internal/config"
	"web-console/internal/pty"
	"web-console/session"
)

// newSessionManager constructs a session.Manager with the package-main default
// PTY factory, tmux integration callbacks, upload-dir resolver, and per-session
// env builder wired in. The session/ sub-package has no knowledge of these
// defaults — this is the single place that bridges the two packages.
func newSessionManager() *session.Manager {
	cfg, err := config.LoadChecked()
	if err != nil {
		log.Printf("configuration: shell unavailable: %v", err)
	} else {
		log.Printf("configuration: resolved shell=%q", cfg.DefaultShell)
	}
	sm := session.NewManager(defaultPTYFactory, cfg)
	wireDefaultManagerHooks(sm)
	return sm
}

// newSessionManagerWithFactory constructs a session.Manager for tests that want
// to substitute a fake PTY factory. Tmux hooks still come from package main so
// tests exercising tmux/recovery code see the same wiring as production.
func newSessionManagerWithFactory(factory pty.Factory) *session.Manager {
	sm := session.NewManagerWithFactory(factory)
	wireDefaultManagerHooks(sm)
	return sm
}

// wireDefaultManagerHooks installs the package-main default tmux helpers,
// upload-dir resolver, and per-session env builder on a bare Manager.
func wireDefaultManagerHooks(sm *session.Manager) {
	sm.SetTmuxHooks(
		tmuxAttachAsPTY,
		DiscoverTmuxSessions,
		refreshTmuxOptions,
		func(name string) { _ = tmuxCmd("kill-session", "-t", name).Run() },
		tmuxSessionPrefix,
	)
	sm.SetUploadDirFunc(resolveUploadDir)
	sm.SetEnvForSessionFunc(defaultSessionEnv)
}

// defaultSessionEnv returns the per-session environment variables injected
// into the spawned PTY. Wired into session.Manager.envForSession by both
// constructors; tests can override the field for hermetic spawning.
func defaultSessionEnv(sessionID string) map[string]string {
	env := map[string]string{
		"WC_WEB_CONSOLE_SESSION_ID": sessionID,
		// The launcher uses this root to materialize CODEX_HOME/GROK_HOME only
		// when the corresponding native agent is actually started. A plain shell
		// therefore carries attribution identity without creating agent state.
		"WC_SESSION_STATE_ROOT": resolveSessionStateRoot(),
	}
	// A verified-human session carries a short-lived human-owner delegation that
	// a coding agent launched here forwards on attach so agent-manager attributes
	// the run to the operator. Absent for plain shells, service/agent sessions,
	// or an expired delegation — then the agent attaches unattributed, exactly as
	// before. This is the ONLY place the delegation enters an agent environment,
	// and it is the derived minted token, never the operator's browser credential.
	if token, ok := activeSessionDelegations.Lookup(sessionID, time.Now()); ok {
		env[cliutil.HumanOwnerDelegationEnv] = token
	}
	return env
}
