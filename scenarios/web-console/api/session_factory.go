package main

import (
	"web-console/internal/config"
	"web-console/internal/pty"
	"web-console/session"
)

// newSessionManager constructs a session.Manager with the package-main default
// PTY factory, tmux integration callbacks, upload-dir resolver, and per-session
// env builder wired in. The session/ sub-package has no knowledge of these
// defaults — this is the single place that bridges the two packages.
func newSessionManager() *session.Manager {
	sm := session.NewManager(defaultPTYFactory, config.Load())
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
		applyTmuxOptions,
		func(name string) { _ = tmuxCmd("kill-session", "-t", name).Run() },
		tmuxSessionPrefix,
	)
	sm.SetUploadDirFunc(resolveUploadDir)
	sm.SetEnvForSessionFunc(defaultSessionEnv)
}

// init installs the package-main implementations of the ANSI hooks the
// session package exposes for dependency injection. Doing this in init keeps
// the session/ package side of the seam free of package-main symbols.
func init() {
	session.AnsiResponderFunc = generateAnsiResponses
	session.SanitizeForClientFunc = sanitizeForClient
}
