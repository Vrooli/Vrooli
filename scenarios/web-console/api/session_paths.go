package main

import (
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/storage"
)

// resolveSessionStateRoot returns the root directory for session-scoped state
// such as per-session CODEX_HOME directories. Tests override this path to keep
// recovery/tailer runs from reading or deleting the live app's session state.
func resolveSessionStateRoot() string {
	if root := strings.TrimSpace(getEnvOrDefault("WC_SESSION_STATE_ROOT", "")); root != "" {
		return ensureDir(filepath.Clean(root))
	}
	return mustResolveScenarioStorageDir(storage.ClassState, "sessions")
}
