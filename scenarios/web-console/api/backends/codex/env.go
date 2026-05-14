// Package codex holds codex-CLI-specific helpers used by the web-console
// session pipeline: per-session CODEX_HOME layout, rollout parsing, and
// the tailer that routes assistant/user text back to the owning session.
package codex

import (
	"log"
	"os"
	"path/filepath"
)

// ensureDir mkdirs path (recursive) and returns it. On failure, returns
// the original path unchanged — callers tolerate non-existent dirs.
func ensureDir(path string) string {
	if path == "" {
		return path
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return path
	}
	return path
}

// resolveUserConfigDir returns ~/<name>, or "" if HOME is unresolvable.
func resolveUserConfigDir(name string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, name)
}

// SharedHome returns the user-level ~/.codex directory, creating it if
// missing. This is the source of shared config/auth/skills that each
// per-session CODEX_HOME symlinks back to.
func SharedHome() string {
	return ensureDir(resolveUserConfigDir(".codex"))
}

// ensureSymlink creates dst as a symlink to src if dst does not already
// point at src. No-op if dst exists but is a regular file (we never
// overwrite real user data).
func ensureSymlink(dst, src string) {
	info, err := os.Lstat(dst)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			target, readErr := os.Readlink(dst)
			if readErr == nil && target == src {
				return
			}
		}
		return
	}
	if !os.IsNotExist(err) {
		return
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		log.Printf("session-state: failed creating parent for %s: %v", dst, err)
		return
	}
	if err := os.Symlink(src, dst); err != nil && !os.IsExist(err) {
		log.Printf("session-state: failed linking %s -> %s: %v", dst, src, err)
	}
}

// PrepareSessionHome lays out a per-session CODEX_HOME under sessionHome
// and symlinks shared auth/config from sharedHome. Idempotent.
func PrepareSessionHome(sessionHome, sharedHome string) string {
	sessionHome = ensureDir(sessionHome)
	if sessionHome == "" {
		return sessionHome
	}

	for _, dir := range []string{"sessions", "log", "logs", "outputs", "tmp"} {
		ensureDir(filepath.Join(sessionHome, dir))
	}

	if sharedHome == "" {
		return sessionHome
	}

	for _, entry := range []string{
		"auth.json",
		"config.toml",
		"settings.json",
		"skills",
		"rules",
		"version.json",
		".personality_migration",
	} {
		src := filepath.Join(sharedHome, entry)
		if _, err := os.Lstat(src); err != nil {
			continue
		}
		ensureSymlink(filepath.Join(sessionHome, entry), src)
	}
	return sessionHome
}

// SessionHome returns the per-session CODEX_HOME path rooted under
// sessionStateRoot, creating the directory layout and symlinks on first
// call. sessionStateRoot is the caller-provided root (e.g. the
// web-console state directory) so this package stays independent of
// package-main config helpers.
func SessionHome(sessionStateRoot, sessionID string) string {
	return PrepareSessionHome(
		filepath.Join(sessionStateRoot, "codex", sessionID),
		SharedHome(),
	)
}

// SessionsDir returns the per-session rollout-JSONL directory (the
// "sessions" subdir under SessionHome).
func SessionsDir(sessionStateRoot, sessionID string) string {
	return ensureDir(filepath.Join(SessionHome(sessionStateRoot, sessionID), "sessions"))
}
