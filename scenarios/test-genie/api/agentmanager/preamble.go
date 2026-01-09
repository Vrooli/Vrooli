package agentmanager

import (
	"fmt"
	"path/filepath"
	"strings"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// Default limits for preamble generation.
const (
	DefaultMaxFilesChanged = 50
	DefaultMaxBytesWritten = 1024 * 1024 // 1MB
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

// GenerateSafetyPreamble creates the immutable safety preamble that will be
// included as a context attachment. This ensures agents operate within
// security boundaries.
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
		maxFiles = DefaultMaxFilesChanged
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytesWritten
	}
	maxBytesKB := maxBytes / 1024

	networkStatus := "DISABLED (agents cannot make outbound requests)"
	if cfg.NetworkEnabled {
		networkStatus = "ENABLED (agents can make outbound requests)"
	}

	allowedCommands := getDefaultAllowedCommands()
	var allowedBashLines []string
	for _, cmd := range allowedCommands {
		allowedBashLines = append(allowedBashLines, fmt.Sprintf("  - %s", cmd))
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

All other bash commands will be blocked. Use the built-in tools (Read, Edit, Write, Glob, Grep) for file operations.

---

`, scenarioPath, scopeDescription, maxFiles, maxBytesKB, networkStatus, scenarioPath, maxFiles, maxBytesKB, allowedBashSection)
}

// GeneratePreambleAttachment creates a context attachment with the safety preamble.
func GeneratePreambleAttachment(cfg PreambleConfig) *domainpb.ContextAttachment {
	preamble := GenerateSafetyPreamble(cfg)
	if preamble == "" {
		return nil
	}

	return &domainpb.ContextAttachment{
		Type:    "note",
		Key:     "test-genie-security-preamble",
		Label:   "Security Constraints",
		Content: preamble,
		Tags:    []string{"security", "constraints", "preamble"},
	}
}

// getDefaultAllowedCommands returns the list of allowed bash command prefixes.
// This is a self-contained list to avoid dependencies on the security package.
func getDefaultAllowedCommands() []string {
	return []string{
		// Test runners
		"pnpm test",
		"pnpm run test",
		"npm test",
		"npm run test",
		"go test",
		"vitest",
		"jest",
		"bats",
		"make test",
		"make check",
		"pytest",
		"python -m pytest",
		// Build commands
		"pnpm build",
		"pnpm run build",
		"npm run build",
		"go build",
		"make build",
		"make",
		// Linters and formatters
		"pnpm lint",
		"pnpm run lint",
		"npm run lint",
		"eslint",
		"prettier",
		"gofmt",
		"gofumpt",
		"golangci-lint",
		// Type checking
		"pnpm typecheck",
		"pnpm run typecheck",
		"tsc",
		// Safe inspection commands
		"ls",
		"pwd",
		"which",
		"wc",
		"diff",
		// Git read-only commands
		"git status",
		"git diff",
		"git log",
		"git show",
	}
}

// GetDefaultSafeTools returns the default list of safe tools for test generation.
func GetDefaultSafeTools() []string {
	return []string{
		"Read",  // Read files
		"Write", // Create/overwrite files
		"Edit",  // Modify files
		"Glob",  // Find files by pattern
		"Grep",  // Search file contents
		"Bash",  // Execute allowed commands
	}
}
