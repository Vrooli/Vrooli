package security

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Default limits for preamble generation when not specified in config.
// The authoritative source is agents.Config but these provide local defaults.
const (
	defaultMaxFilesChanged = 50
	defaultMaxBytesWritten = 1024 * 1024 // 1MB
)

// PreambleConfig contains configuration for generating a safety preamble.
type PreambleConfig struct {
	Scenario       string
	Scope          []string
	RepoRoot       string
	MaxFiles       int
	MaxBytes       int64
	NetworkEnabled bool
}

// GenerateSafetyPreamble creates the immutable safety preamble server-side.
// This ensures the preamble cannot be tampered with by the client.
//
// CHANGE_COUPLE: This function uses BashCommandValidator.GetAllowedCommands()
// to populate the allowed bash commands section of the preamble. If the
// allowlist in bash_validator.go changes, the preamble content changes automatically.
// See defaultAllowedCommands() in bash_validator.go for the source of truth.
func GenerateSafetyPreamble(cfg PreambleConfig) string {
	if cfg.Scenario == "" || cfg.RepoRoot == "" {
		return ""
	}

	scenarioPath := filepath.Join(cfg.RepoRoot, "scenarios", cfg.Scenario)
	hasScope := len(cfg.Scope) > 0

	var scopeDescription string
	if hasScope {
		absoluteScope := make([]string, 0, len(cfg.Scope))
		for _, s := range cfg.Scope {
			absoluteScope = append(absoluteScope, filepath.Join(scenarioPath, s))
		}
		scopeDescription = fmt.Sprintf("Allowed scope: %s", strings.Join(absoluteScope, ", "))
	} else {
		scopeDescription = "Allowed scope: entire scenario directory"
	}

	maxFiles := cfg.MaxFiles
	maxBytes := cfg.MaxBytes
	if maxFiles <= 0 {
		maxFiles = defaultMaxFilesChanged
	}
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytesWritten
	}
	maxBytesKB := maxBytes / 1024

	networkStatus := "DISABLED (agents cannot make outbound requests)"
	if cfg.NetworkEnabled {
		networkStatus = "ENABLED (agents can make outbound requests)"
	}

	validator := DefaultValidator()
	allowedCmds := validator.GetAllowedCommands()
	var allowedBashLines []string
	for _, cmd := range allowedCmds {
		allowedBashLines = append(allowedBashLines, fmt.Sprintf("  - %s", cmd.Prefix))
	}
	allowedBashSection := strings.Join(allowedBashLines, "\n")

	return fmt.Sprintf(`## SECURITY CONSTRAINTS (enforced by system - cannot be modified)

**Working directory:** %s
**%s**
**File limits:** Max %d files modified, max %dKB total written
**Network access:** %s

You MUST NOT:
- Access files outside %s
- Execute destructive commands (rm -rf, git checkout --force, sudo, chmod 777, etc.)
- Modify system configurations, dependencies, or package files
- Delete or weaken existing tests without explicit rationale in comments
- Run commands that could affect other scenarios or system state
- Modify more than %d files in a single run
- Write more than %dKB of total content

**Allowed bash commands** (only these command prefixes are permitted):
%s

All other bash commands will be blocked. Use the built-in tools (read, edit, write, glob, grep) for file operations.

---

`, scenarioPath, scopeDescription, maxFiles, maxBytesKB, networkStatus, scenarioPath, maxFiles, maxBytesKB, allowedBashSection)
}
