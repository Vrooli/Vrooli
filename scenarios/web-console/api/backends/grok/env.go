// Package grok holds grok-CLI-specific helpers used by the web-console session
// pipeline: per-session GROK_HOME isolation and the parser/tailer that route
// user/assistant text from grok's append-only updates.jsonl back to the owning
// web-console session.
//
// Attribution is by construction: each pane gets its own GROK_HOME, so the only
// session transcripts under it belong to that pane. Shared credentials/config
// from the operator's real ~/.grok are symlinked in so login and configuration
// keep working; only the per-session `sessions/` subtree is isolated.
package grok

import (
	"log"
	"os"
	"path/filepath"
)

// ensureDir mkdirs path (recursive) and returns it. On failure, returns the
// original path unchanged — callers tolerate non-existent dirs.
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

// SharedHome returns the user-level ~/.grok directory (NOT created if missing —
// when grok was never run there is nothing to inherit). This is the source of
// shared auth/config/skills each per-session GROK_HOME symlinks back to.
func SharedHome() string {
	return resolveUserConfigDir(".grok")
}

// isolatedEntries are the shared-home entries that must NOT be symlinked into a
// per-session GROK_HOME: `sessions` is the per-pane transcript tree we isolate,
// and the active-session bookkeeping/locks are process-local state that would
// otherwise alias across panes.
var isolatedEntries = map[string]bool{
	"sessions":             true,
	"active_sessions.json": true,
	"active_sessions.lock": true,
}

// ensureSymlink creates dst as a symlink to src if dst does not already point at
// src. No-op if dst already exists (we never overwrite real user data).
func ensureSymlink(dst, src string) {
	if info, err := os.Lstat(dst); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			if target, readErr := os.Readlink(dst); readErr == nil && target == src {
				return
			}
		}
		return
	} else if !os.IsNotExist(err) {
		return
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		log.Printf("grok session-state: failed creating parent for %s: %v", dst, err)
		return
	}
	if err := os.Symlink(src, dst); err != nil && !os.IsExist(err) {
		log.Printf("grok session-state: failed linking %s -> %s: %v", dst, src, err)
	}
}

// PrepareSessionHome lays out a per-session GROK_HOME under sessionHome and
// symlinks every shared entry except the isolated set from sharedHome.
// Idempotent. Symlinking the full remainder (rather than a fixed allowlist)
// keeps working as grok's home layout evolves; locks/active-session state are
// the only things deliberately left process-local.
func PrepareSessionHome(sessionHome, sharedHome string) string {
	sessionHome = ensureDir(sessionHome)
	if sessionHome == "" {
		return sessionHome
	}
	ensureDir(filepath.Join(sessionHome, "sessions"))

	if sharedHome == "" {
		return sessionHome
	}
	entries, err := os.ReadDir(sharedHome)
	if err != nil {
		return sessionHome
	}
	for _, entry := range entries {
		name := entry.Name()
		if isolatedEntries[name] || filepath.Ext(name) == ".lock" {
			continue
		}
		ensureSymlink(filepath.Join(sessionHome, name), filepath.Join(sharedHome, name))
	}
	return sessionHome
}

// SessionHome returns the per-session GROK_HOME path rooted under
// sessionStateRoot, creating the directory layout and symlinks on first call.
func SessionHome(sessionStateRoot, sessionID string) string {
	return PrepareSessionHome(SessionHomePath(sessionStateRoot, sessionID), SharedHome())
}

// SessionHomePath returns the lazy per-session home path without creating it.
func SessionHomePath(sessionStateRoot, sessionID string) string {
	return filepath.Join(sessionStateRoot, "grok", sessionID)
}

// SessionsDir returns the per-session transcript root (the `sessions` subdir
// under SessionHome). grok writes <SessionsDir>/<url-encoded-cwd>/<session-id>/
// updates.jsonl beneath it.
func SessionsDir(sessionStateRoot, sessionID string) string {
	return ensureDir(SessionsDirPath(sessionStateRoot, sessionID))
}

// SessionsDirPath returns the transcript path without creating it.
func SessionsDirPath(sessionStateRoot, sessionID string) string {
	return filepath.Join(SessionHomePath(sessionStateRoot, sessionID), "sessions")
}
