package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/storage"
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

// resolveSessionStateRoot returns the root directory for session-scoped state
// such as per-session CODEX_HOME directories. Tests override this path to keep
// recovery/tailer runs from reading or deleting the live app's session state.
func resolveSessionStateRoot() string {
	if root := strings.TrimSpace(getEnvOrDefault("WC_SESSION_STATE_ROOT", "")); root != "" {
		return ensureDir(filepath.Clean(root))
	}
	return mustResolveScenarioStorageDir(storage.ClassState, "sessions")
}
