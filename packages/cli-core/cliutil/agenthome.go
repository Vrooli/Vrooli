package cliutil

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	webConsoleSessionIDEnv     = "WC_WEB_CONSOLE_SESSION_ID"
	webConsoleStateRootEnv     = "WC_SESSION_STATE_ROOT"
	webConsoleCodexHomeEnv     = "CODEX_HOME"
	webConsoleGrokHomeEnv      = "GROK_HOME"
	webConsoleCodexSessionsEnv = "WC_CODEX_SESSIONS_DIR"
)

// PrepareWebConsoleAgentHome materializes the session-scoped home only at the
// native-agent launch boundary. A plain shell receives the identifying
// web-console variables, but it does not create agent state. The function is
// deliberately best-effort: attribution and home isolation must not prevent a
// coding agent from starting when the shared home is unavailable.
func PrepareWebConsoleAgentHome(agent string, environment []string) []string {
	values := environmentValues(environment)
	sessionID := strings.TrimSpace(values[webConsoleSessionIDEnv])
	stateRoot := strings.TrimSpace(values[webConsoleStateRootEnv])
	if sessionID == "" || stateRoot == "" || filepath.Base(sessionID) != sessionID || sessionID == "." || sessionID == ".." {
		return environment
	}

	var home, shared string
	switch strings.ToLower(strings.TrimSpace(agent)) {
	case "codex":
		home = filepath.Join(stateRoot, "codex", sessionID)
		shared = userHomeDir(".codex")
		prepareCodexHome(home, shared)
		environment = withEnvironmentValue(environment, webConsoleCodexHomeEnv, home)
		environment = withEnvironmentValue(environment, webConsoleCodexSessionsEnv, filepath.Join(home, "sessions"))
	case "grok":
		home = filepath.Join(stateRoot, "grok", sessionID)
		shared = userHomeDir(".grok")
		prepareGrokHome(home, shared)
		environment = withEnvironmentValue(environment, webConsoleGrokHomeEnv, home)
	}
	return environment
}

func environmentValues(environment []string) map[string]string {
	values := make(map[string]string, len(environment))
	for _, entry := range environment {
		name, value, ok := strings.Cut(entry, "=")
		if ok {
			values[name] = value
		}
	}
	return values
}

func userHomeDir(name string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, name)
}

func prepareCodexHome(home, shared string) {
	if home == "" {
		return
	}
	_ = os.MkdirAll(filepath.Join(home, "sessions"), 0o755)
	if shared == "" {
		return
	}
	_ = os.MkdirAll(shared, 0o755)
	for _, dir := range []string{"log", "logs", "outputs", "tmp", "cache", "plugins", "shell_snapshots"} {
		sharedPath := filepath.Join(shared, dir)
		_ = os.MkdirAll(sharedPath, 0o755)
		ensureAgentHomeSymlink(filepath.Join(home, dir), sharedPath)
	}
	for _, entry := range []string{
		"auth.json",
		"config.toml",
		"settings.json",
		"skills",
		"rules",
		"version.json",
		".personality_migration",
		"models_cache.json",
		"installation_id",
	} {
		sharedPath := filepath.Join(shared, entry)
		if _, err := os.Lstat(sharedPath); err == nil {
			ensureAgentHomeSymlink(filepath.Join(home, entry), sharedPath)
		}
	}
}

func prepareGrokHome(home, shared string) {
	if home == "" {
		return
	}
	_ = os.MkdirAll(filepath.Join(home, "sessions"), 0o755)
	if shared == "" {
		return
	}
	entries, err := os.ReadDir(shared)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == "sessions" || name == "active_sessions.json" || name == "active_sessions.lock" || strings.HasSuffix(name, ".lock") {
			continue
		}
		ensureAgentHomeSymlink(filepath.Join(home, name), filepath.Join(shared, name))
	}
}

func ensureAgentHomeSymlink(dst, src string) {
	if _, err := os.Lstat(dst); err == nil || !os.IsNotExist(err) {
		return
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return
	}
	_ = os.Symlink(src, dst)
}
