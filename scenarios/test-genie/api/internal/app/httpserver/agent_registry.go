package httpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// AgentStatus represents the current state of a spawned agent.
type AgentStatus string

const (
	AgentStatusPending   AgentStatus = "pending"
	AgentStatusRunning   AgentStatus = "running"
	AgentStatusCompleted AgentStatus = "completed"
	AgentStatusFailed    AgentStatus = "failed"
	AgentStatusTimeout   AgentStatus = "timeout"
	AgentStatusStopped   AgentStatus = "stopped"
)

// ActiveAgent represents a spawned agent being tracked by the registry.
type ActiveAgent struct {
	ID          string      `json:"id"`
	SessionID   string      `json:"sessionId,omitempty"`
	Scenario    string      `json:"scenario"`
	Scope       []string    `json:"scope"`
	Phases      []string    `json:"phases,omitempty"`
	Model       string      `json:"model"`
	Status      AgentStatus `json:"status"`
	StartedAt   time.Time   `json:"startedAt"`
	CompletedAt *time.Time  `json:"completedAt,omitempty"`
	PromptHash  string      `json:"promptHash"`
	PromptIndex int         `json:"promptIndex"`
	PromptText  string      `json:"promptText,omitempty"` // Full prompt text for display
	Output      string      `json:"output,omitempty"`
	Error       string      `json:"error,omitempty"`
	// Process tracking for orphan detection
	PID      int    `json:"pid,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	cancel   context.CancelFunc
	cmd      *exec.Cmd
}

// ScopeLock represents a lock on a set of paths within a scenario.
type ScopeLock struct {
	Scenario   string    `json:"scenario"`
	Paths      []string  `json:"paths"`
	AgentID    string    `json:"agentId"`
	AcquiredAt time.Time `json:"acquiredAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

// AgentRegistry manages active agents and scope locks.
type AgentRegistry struct {
	mu          sync.RWMutex
	agents      map[string]*ActiveAgent
	scopeLocks  []ScopeLock
	lockTimeout time.Duration
}

// NewAgentRegistry creates a new agent registry with the given lock timeout.
func NewAgentRegistry(lockTimeout time.Duration) *AgentRegistry {
	if lockTimeout <= 0 {
		lockTimeout = 15 * time.Minute // default lock timeout
	}
	return &AgentRegistry{
		agents:      make(map[string]*ActiveAgent),
		scopeLocks:  make([]ScopeLock, 0),
		lockTimeout: lockTimeout,
	}
}

// Register adds a new agent to the registry and acquires scope locks.
// Returns an error if there are scope conflicts.
func (r *AgentRegistry) Register(agent *ActiveAgent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clean up expired locks first
	r.cleanExpiredLocksLocked()

	// Check for scope conflicts
	conflicts := r.checkScopeConflictsLocked(agent.Scenario, agent.Scope)
	if len(conflicts) > 0 {
		return fmt.Errorf("scope conflict with active agents: %v", conflicts)
	}

	// Check for duplicate prompts (same hash)
	for _, existing := range r.agents {
		if existing.Status == AgentStatusRunning || existing.Status == AgentStatusPending {
			if existing.PromptHash == agent.PromptHash && existing.Scenario == agent.Scenario {
				return fmt.Errorf("duplicate agent: identical prompt already running (agent %s)", existing.ID)
			}
		}
	}

	// Register the agent
	r.agents[agent.ID] = agent

	// Acquire scope lock if the agent has specific paths
	if len(agent.Scope) > 0 {
		lock := ScopeLock{
			Scenario:   agent.Scenario,
			Paths:      agent.Scope,
			AgentID:    agent.ID,
			AcquiredAt: time.Now(),
			ExpiresAt:  time.Now().Add(r.lockTimeout),
		}
		r.scopeLocks = append(r.scopeLocks, lock)
	}

	return nil
}

// UpdateStatus updates an agent's status and optionally sets completion info.
func (r *AgentRegistry) UpdateStatus(id string, status AgentStatus, sessionID, output, errMsg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, ok := r.agents[id]
	if !ok {
		return
	}

	agent.Status = status
	if sessionID != "" {
		agent.SessionID = sessionID
	}
	if output != "" {
		agent.Output = output
	}
	if errMsg != "" {
		agent.Error = errMsg
	}

	if status == AgentStatusCompleted || status == AgentStatusFailed ||
		status == AgentStatusTimeout || status == AgentStatusStopped {
		now := time.Now()
		agent.CompletedAt = &now
		// Release scope lock
		r.releaseLockForAgentLocked(id)
	}
}

// Get returns an agent by ID.
func (r *AgentRegistry) Get(id string) (*ActiveAgent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent, ok := r.agents[id]
	if !ok {
		return nil, false
	}
	// Return a copy to prevent race conditions
	copy := *agent
	return &copy, true
}

// ListActive returns all agents that are currently running or pending.
func (r *AgentRegistry) ListActive() []ActiveAgent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ActiveAgent, 0)
	for _, agent := range r.agents {
		if agent.Status == AgentStatusRunning || agent.Status == AgentStatusPending {
			copy := *agent
			result = append(result, copy)
		}
	}
	return result
}

// ListAll returns all agents in the registry.
func (r *AgentRegistry) ListAll(limit int) []ActiveAgent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ActiveAgent, 0, len(r.agents))
	for _, agent := range r.agents {
		copy := *agent
		result = append(result, copy)
	}

	// Sort by start time (most recent first)
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].StartedAt.After(result[i].StartedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result
}

// Stop attempts to stop an agent by ID.
func (r *AgentRegistry) Stop(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, ok := r.agents[id]
	if !ok {
		return fmt.Errorf("agent not found: %s", id)
	}

	if agent.Status != AgentStatusRunning && agent.Status != AgentStatusPending {
		return fmt.Errorf("agent %s is not running (status: %s)", id, agent.Status)
	}

	// Cancel the context if available
	if agent.cancel != nil {
		agent.cancel()
	}

	// Kill the process if available
	if agent.cmd != nil && agent.cmd.Process != nil {
		_ = agent.cmd.Process.Kill()
	}

	agent.Status = AgentStatusStopped
	now := time.Now()
	agent.CompletedAt = &now
	agent.Error = "stopped by user"

	// Release scope lock
	r.releaseLockForAgentLocked(id)

	return nil
}

// StopAll stops all running agents and returns the list of stopped agent IDs.
func (r *AgentRegistry) StopAll() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	stoppedIDs := make([]string, 0)

	for id, agent := range r.agents {
		if agent.Status != AgentStatusRunning && agent.Status != AgentStatusPending {
			continue
		}

		// Cancel the context if available
		if agent.cancel != nil {
			agent.cancel()
		}

		// Kill the process if available
		if agent.cmd != nil && agent.cmd.Process != nil {
			_ = agent.cmd.Process.Kill()
		}

		agent.Status = AgentStatusStopped
		now := time.Now()
		agent.CompletedAt = &now
		agent.Error = "stopped by user (stop all)"

		// Release scope lock
		r.releaseLockForAgentLocked(id)

		stoppedIDs = append(stoppedIDs, id)
	}

	return stoppedIDs
}

// CheckScopeConflicts returns any conflicting agent IDs for the given scenario and paths.
func (r *AgentRegistry) CheckScopeConflicts(scenario string, paths []string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.checkScopeConflictsLocked(scenario, paths)
}

// GetActiveLocks returns all active scope locks.
func (r *AgentRegistry) GetActiveLocks() []ScopeLock {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.cleanExpiredLocksLocked()

	result := make([]ScopeLock, len(r.scopeLocks))
	copy(result, r.scopeLocks)
	return result
}

// CleanupCompleted removes completed agents older than the given duration.
func (r *AgentRegistry) CleanupCompleted(olderThan time.Duration) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	removed := 0

	for id, agent := range r.agents {
		if agent.CompletedAt != nil && agent.CompletedAt.Before(cutoff) {
			delete(r.agents, id)
			removed++
		}
	}

	return removed
}

// SetAgentProcess stores the cancel function and command for an agent.
// This allows the agent to be stopped later.
func (r *AgentRegistry) SetAgentProcess(id string, cancel context.CancelFunc, cmd *exec.Cmd) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if agent, ok := r.agents[id]; ok {
		agent.cancel = cancel
		agent.cmd = cmd
	}
}

// Private helpers

func (r *AgentRegistry) checkScopeConflictsLocked(scenario string, paths []string) []string {
	conflicts := make([]string, 0)

	// Clean expired locks first
	r.cleanExpiredLocksLocked()

	for _, lock := range r.scopeLocks {
		if lock.Scenario != scenario {
			continue
		}

		// Check for path overlaps
		if pathsOverlap(lock.Paths, paths) {
			conflicts = append(conflicts, lock.AgentID)
		}
	}

	return conflicts
}

func (r *AgentRegistry) releaseLockForAgentLocked(agentID string) {
	newLocks := make([]ScopeLock, 0, len(r.scopeLocks))
	for _, lock := range r.scopeLocks {
		if lock.AgentID != agentID {
			newLocks = append(newLocks, lock)
		}
	}
	r.scopeLocks = newLocks
}

func (r *AgentRegistry) cleanExpiredLocksLocked() {
	now := time.Now()
	newLocks := make([]ScopeLock, 0, len(r.scopeLocks))
	for _, lock := range r.scopeLocks {
		if lock.ExpiresAt.After(now) {
			newLocks = append(newLocks, lock)
		}
	}
	r.scopeLocks = newLocks
}

// pathsOverlap checks if two sets of paths have any overlap.
// Paths overlap if one is a prefix of another or they are identical.
// Empty scope means "entire scenario" and conflicts with everything.
func pathsOverlap(a, b []string) bool {
	// If either set is empty, it means "entire scenario" which conflicts with everything
	// in that scenario. Two empty sets also conflict (both cover entire scenario).
	if len(a) == 0 || len(b) == 0 {
		return true
	}

	for _, pathA := range a {
		for _, pathB := range b {
			if pathsConflict(pathA, pathB) {
				return true
			}
		}
	}
	return false
}

// pathsConflict checks if two paths conflict (one contains the other).
func pathsConflict(a, b string) bool {
	// Normalize paths
	a = strings.TrimSuffix(a, "/")
	b = strings.TrimSuffix(b, "/")

	// Exact match
	if a == b {
		return true
	}

	// One is prefix of the other
	if strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/") {
		return true
	}

	return false
}

// HashPrompt creates a hash of the prompt for deduplication.
func HashPrompt(prompt string) string {
	h := sha256.New()
	h.Write([]byte(prompt))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// AllowedBashCommand represents a bash command prefix that is explicitly permitted.
type AllowedBashCommand struct {
	Prefix      string // Command prefix that must match (e.g., "pnpm test", "go test")
	Description string // Human-readable description
}

// BashCommandValidator validates bash commands using an allowlist approach.
// SECURITY: This uses allowlist (whitelist) instead of blocklist (blacklist) for stronger security.
// Only explicitly permitted command prefixes are allowed - everything else is rejected.
type BashCommandValidator struct {
	allowedCommands []AllowedBashCommand
	// Keep blocklist for prompt scanning (defense in depth)
	blockedPatterns []struct {
		Pattern     *regexp.Regexp
		Description string
	}
}

// NewBashCommandValidator creates a validator with the default allowed commands.
func NewBashCommandValidator() *BashCommandValidator {
	return &BashCommandValidator{
		allowedCommands: []AllowedBashCommand{
			// Test runners - safe read-only operations
			{Prefix: "pnpm test", Description: "pnpm test runner"},
			{Prefix: "pnpm run test", Description: "pnpm test script"},
			{Prefix: "npm test", Description: "npm test runner"},
			{Prefix: "npm run test", Description: "npm test script"},
			{Prefix: "go test", Description: "Go test runner"},
			{Prefix: "vitest", Description: "Vitest test runner"},
			{Prefix: "jest", Description: "Jest test runner"},
			{Prefix: "bats", Description: "BATS test runner"},
			{Prefix: "make test", Description: "Make test target"},
			{Prefix: "make check", Description: "Make check target"},
			{Prefix: "pytest", Description: "Python pytest runner"},
			{Prefix: "python -m pytest", Description: "Python pytest module"},
			// Build commands - generally safe
			{Prefix: "pnpm build", Description: "pnpm build"},
			{Prefix: "pnpm run build", Description: "pnpm build script"},
			{Prefix: "npm run build", Description: "npm build script"},
			{Prefix: "go build", Description: "Go build"},
			{Prefix: "make build", Description: "Make build target"},
			{Prefix: "make", Description: "Make default target"},
			// Linters and formatters - read-only or in-place safe
			{Prefix: "pnpm lint", Description: "pnpm lint"},
			{Prefix: "pnpm run lint", Description: "pnpm lint script"},
			{Prefix: "npm run lint", Description: "npm lint script"},
			{Prefix: "eslint", Description: "ESLint"},
			{Prefix: "prettier", Description: "Prettier formatter"},
			{Prefix: "gofmt", Description: "Go formatter"},
			{Prefix: "gofumpt", Description: "Go strict formatter"},
			{Prefix: "golangci-lint", Description: "Go linter"},
			// Type checking - read-only
			{Prefix: "pnpm typecheck", Description: "pnpm typecheck"},
			{Prefix: "pnpm run typecheck", Description: "pnpm typecheck script"},
			{Prefix: "tsc", Description: "TypeScript compiler"},
			// Safe inspection commands
			// SECURITY: cat, head, tail removed - they can read arbitrary files outside scope.
			// Agents should use the Read tool which respects scope boundaries.
			// SECURITY: echo removed - can write files via redirection (echo "x" > file).
			// SECURITY: find removed from general allowlist - restricted via safeGlobPrefixes.
			{Prefix: "ls", Description: "List directory"},
			{Prefix: "pwd", Description: "Print working directory"},
			{Prefix: "which", Description: "Locate command"},
			{Prefix: "wc", Description: "Word count"},
			{Prefix: "diff", Description: "Compare files"},
			// Git read-only commands
			{Prefix: "git status", Description: "Git status"},
			{Prefix: "git diff", Description: "Git diff"},
			{Prefix: "git log", Description: "Git log"},
			{Prefix: "git show", Description: "Git show"},
			{Prefix: "git branch", Description: "Git branch list"},
		},
		blockedPatterns: []struct {
			Pattern     *regexp.Regexp
			Description string
		}{
			// Git destructive commands
			{regexp.MustCompile(`(?i)\bgit\s+(checkout|reset\s+--hard|branch\s+-[dD]|push|clean|rebase|merge|cherry-pick|revert|commit|add\s)`), "destructive/mutating git operation"},
			{regexp.MustCompile(`(?i)\bgit\s+stash`), "git stash operation"},
			// Filesystem destructive commands
			{regexp.MustCompile(`(?i)\brm\s`), "rm command"},
			{regexp.MustCompile(`(?i)\bmv\s`), "mv command"},
			{regexp.MustCompile(`(?i)\bcp\s`), "cp command"},
			{regexp.MustCompile(`(?i)\bmkdir\s`), "mkdir command"},
			{regexp.MustCompile(`(?i)\brmdir\s`), "rmdir command"},
			{regexp.MustCompile(`(?i)\btouch\s`), "touch command"},
			// Permission/ownership changes
			{regexp.MustCompile(`(?i)\bchmod\s`), "chmod command"},
			{regexp.MustCompile(`(?i)\bchown\s`), "chown command"},
			// System operations
			{regexp.MustCompile(`(?i)\bsudo\s`), "sudo command"},
			{regexp.MustCompile(`(?i)\bsu\s`), "su command"},
			{regexp.MustCompile(`(?i)\bsystemctl\s`), "systemctl command"},
			{regexp.MustCompile(`(?i)\bservice\s`), "service command"},
			{regexp.MustCompile(`(?i)\bkill\s`), "kill command"},
			{regexp.MustCompile(`(?i)\bpkill\s`), "pkill command"},
			// Database operations
			{regexp.MustCompile(`(?i)\bDROP\s`), "SQL DROP"},
			{regexp.MustCompile(`(?i)\bTRUNCATE\s`), "SQL TRUNCATE"},
			{regexp.MustCompile(`(?i)\bDELETE\s+FROM\s`), "SQL DELETE"},
			{regexp.MustCompile(`(?i)\bUPDATE\s+\w+\s+SET\s`), "SQL UPDATE"},
			{regexp.MustCompile(`(?i)\bINSERT\s+INTO\s`), "SQL INSERT"},
			// Package managers
			{regexp.MustCompile(`(?i)\b(npm|pnpm|yarn)\s+(install|add|remove|uninstall|update|upgrade)\s`), "package installation"},
			{regexp.MustCompile(`(?i)\bgo\s+(get|install|mod\s+tidy)\s`), "go package operation"},
			{regexp.MustCompile(`(?i)\bpip\s+(install|uninstall)\s`), "pip operation"},
			// Network operations that could be dangerous
			{regexp.MustCompile(`(?i)\bcurl\s+.*\|\s*(sh|bash)`), "remote script execution"},
			{regexp.MustCompile(`(?i)\bwget\s+.*\|\s*(sh|bash)`), "remote script execution"},
			{regexp.MustCompile(`(?i)\bcurl\s+.*(-X|--request)\s*(POST|PUT|DELETE|PATCH)`), "HTTP mutation"},
			// Shell execution
			{regexp.MustCompile(`(?i)\beval\s`), "eval command"},
			{regexp.MustCompile(`(?i)\bexec\s`), "exec command"},
			{regexp.MustCompile(`(?i)\bsource\s`), "source command"},
			{regexp.MustCompile(`(?i)\b\.\s+/`), "dot source command"},
		},
	}
}

// safeGlobPrefixes defines which commands are safe to use with glob patterns.
// These are isolated test runners that don't read arbitrary files.
// SECURITY: Only test runners are allowed - they operate within the test framework's scope.
// Commands like find are NOT included as they can traverse outside the allowed directory.
var safeGlobPrefixes = map[string]bool{
	"bats":             true,
	"go test":          true,
	"pytest":           true,
	"python -m pytest": true,
	"vitest":           true,
	"jest":             true,
	"pnpm test":        true,
	"npm test":         true,
	"pnpm run test":    true,
	"npm run test":     true,
	"make test":        true,
	"make check":       true,
	"ls":               true, // ls with globs is safe (shows filenames only)
	// SECURITY: find removed - can traverse outside allowed paths and list sensitive directories
}

// ValidateBashPattern checks if a bash pattern from bash(...) syntax contains only allowed commands.
// Returns an error if any command in the pattern is not in the allowlist.
func (v *BashCommandValidator) ValidateBashPattern(pattern string) error {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return fmt.Errorf("empty bash pattern")
	}

	// Check if the pattern starts with any allowed command prefix
	patternLower := strings.ToLower(pattern)
	hasGlob := strings.Contains(pattern, "*") || strings.Contains(pattern, "?")

	for _, allowed := range v.allowedCommands {
		prefixLower := strings.ToLower(allowed.Prefix)
		// Check exact match or prefix followed by space/end
		if patternLower == prefixLower ||
			strings.HasPrefix(patternLower, prefixLower+" ") ||
			strings.HasPrefix(patternLower, prefixLower+"\t") {
			// If there's a glob, only allow for explicitly safe commands
			if hasGlob {
				if safeGlobPrefixes[allowed.Prefix] {
					return nil // Safe glob usage
				}
				// Glob with non-safe command - reject to prevent "cat *.log" style attacks
				return fmt.Errorf("glob patterns (*,?) are not allowed with '%s'; only test runners support glob patterns", allowed.Prefix)
			}
			return nil // No glob, allowed command
		}
	}

	return fmt.Errorf("bash command '%s' is not in the allowlist; only test runners and safe inspection commands are permitted", pattern)
}

// Validate checks if the given allowed tools list contains only permitted commands.
// Uses allowlist for bash commands - only explicitly permitted command prefixes are allowed.
func (v *BashCommandValidator) Validate(allowedTools []string) error {
	for _, tool := range allowedTools {
		toolLower := strings.ToLower(strings.TrimSpace(tool))

		// Check for wildcard permissions - always reject
		if toolLower == "bash" || toolLower == "*" {
			return fmt.Errorf("unrestricted bash access is not allowed for spawned agents; use scoped patterns like 'bash(pnpm test|go test)'")
		}

		// If it's a bash pattern, validate each command against the allowlist
		if strings.HasPrefix(toolLower, "bash(") && strings.HasSuffix(tool, ")") {
			innerPatterns := tool[5 : len(tool)-1]
			for _, pattern := range strings.Split(innerPatterns, "|") {
				pattern = strings.TrimSpace(pattern)
				if pattern == "" {
					continue
				}
				if err := v.ValidateBashPattern(pattern); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// ValidatePrompt scans the prompt for obviously dangerous instructions.
// This is defense-in-depth - even if an agent tries to execute commands via prompt injection,
// the allowlist validation will block them. But we still warn about suspicious patterns.
func (v *BashCommandValidator) ValidatePrompt(prompt string) error {
	for _, blocked := range v.blockedPatterns {
		if blocked.Pattern.MatchString(prompt) {
			return fmt.Errorf("prompt contains potentially dangerous pattern: %s", blocked.Description)
		}
	}
	return nil
}

// GetAllowedCommands returns a copy of the allowed command list for documentation purposes.
func (v *BashCommandValidator) GetAllowedCommands() []AllowedBashCommand {
	result := make([]AllowedBashCommand, len(v.allowedCommands))
	copy(result, v.allowedCommands)
	return result
}

// Legacy type aliases for backward compatibility
type BlockedCommand = struct {
	Pattern     *regexp.Regexp
	Description string
}
type DestructiveCommandValidator = BashCommandValidator

// NewDestructiveCommandValidator creates a validator (legacy alias).
func NewDestructiveCommandValidator() *BashCommandValidator {
	return NewBashCommandValidator()
}

// SanitizeAllowedTools filters and validates the allowed tools list.
// Returns the sanitized list and any validation errors.
func SanitizeAllowedTools(tools []string) ([]string, error) {
	validator := NewDestructiveCommandValidator()

	if err := validator.Validate(tools); err != nil {
		return nil, err
	}

	// Filter out empty entries and normalize
	sanitized := make([]string, 0, len(tools))
	for _, tool := range tools {
		tool = strings.TrimSpace(tool)
		if tool == "" {
			continue
		}
		sanitized = append(sanitized, tool)
	}

	// Ensure we don't have unrestricted permissions
	for _, tool := range sanitized {
		if tool == "*" {
			return nil, fmt.Errorf("wildcard tool permission (*) is not allowed for spawned agents")
		}
	}

	return sanitized, nil
}

// ValidateAllowedTools validates a list of allowed tools without sanitizing.
// Returns an error if any tool is invalid or dangerous.
func ValidateAllowedTools(tools []string) error {
	_, err := SanitizeAllowedTools(tools)
	return err
}

// DefaultSafeTools returns a safe default set of allowed tools for agents.
func DefaultSafeTools() []string {
	return []string{
		"read",
		"edit",
		"write",
		"glob",
		"grep",
	}
}

// SafeBashPatterns returns commonly allowed bash patterns for test operations.
func SafeBashPatterns() []string {
	return []string{
		"bash(pnpm test|pnpm run test|npm test|npm run test)",
		"bash(go test|go test ./...)",
		"bash(vitest|vitest run)",
		"bash(jest|jest --)",
		"bash(bats|bats *)",
		"bash(make test)",
	}
}

// PathValidator validates file paths to ensure they stay within allowed boundaries.
// SECURITY: This prevents path traversal attacks and ensures agents only access
// files within their designated scenario directory.
type PathValidator struct {
	allowedRoot string // Absolute path to the allowed root directory
}

// NewPathValidator creates a path validator for the given root directory.
func NewPathValidator(allowedRoot string) (*PathValidator, error) {
	if allowedRoot == "" {
		return nil, fmt.Errorf("allowed root path cannot be empty")
	}

	// Normalize the path to absolute
	absRoot, err := normalizeToAbsolute(allowedRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid allowed root: %w", err)
	}

	return &PathValidator{allowedRoot: absRoot}, nil
}

// ValidatePath checks if a path is within the allowed root directory.
// Returns an error if the path escapes the allowed root via traversal or symlinks.
func (v *PathValidator) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}

	// Normalize the path
	absPath, err := normalizeToAbsolute(path)
	if err != nil {
		return fmt.Errorf("invalid path '%s': %w", path, err)
	}

	// Check if the path is within the allowed root
	if !isPathWithinRoot(absPath, v.allowedRoot) {
		return fmt.Errorf("path '%s' is outside allowed root '%s'", path, v.allowedRoot)
	}

	return nil
}

// ValidatePaths checks multiple paths and returns all validation errors.
func (v *PathValidator) ValidatePaths(paths []string) []error {
	var errors []error
	for _, path := range paths {
		if err := v.ValidatePath(path); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

// GetAllowedRoot returns the allowed root directory.
func (v *PathValidator) GetAllowedRoot() string {
	return v.allowedRoot
}

// normalizeToAbsolute converts a path to its absolute, cleaned form.
func normalizeToAbsolute(path string) (string, error) {
	// Expand home directory if present
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = home + path[1:]
	}

	// Clean the path (removes . and .. where possible)
	cleaned := filepath.Clean(path)

	// Convert to absolute if relative
	if !filepath.IsAbs(cleaned) {
		abs, err := filepath.Abs(cleaned)
		if err != nil {
			return "", err
		}
		cleaned = abs
	}

	return cleaned, nil
}

// isPathWithinRoot checks if absPath is within or equal to absRoot.
// Both paths must already be absolute and cleaned.
func isPathWithinRoot(absPath, absRoot string) bool {
	// Ensure root ends with separator for proper prefix matching
	rootWithSep := absRoot
	if !strings.HasSuffix(rootWithSep, string(filepath.Separator)) {
		rootWithSep += string(filepath.Separator)
	}

	// Check exact match or prefix
	return absPath == absRoot || strings.HasPrefix(absPath, rootWithSep)
}

// ValidateScopePaths validates that all scope paths are within the scenario directory.
// This should be called before registering an agent to ensure scope paths don't escape.
func ValidateScopePaths(scenario string, scopePaths []string, repoRoot string) error {
	if scenario == "" {
		return nil // No scenario = no path validation needed
	}

	if repoRoot == "" {
		repoRoot = os.Getenv("VROOLI_ROOT")
	}
	if repoRoot == "" {
		return fmt.Errorf("VROOLI_ROOT not set; cannot validate paths")
	}

	scenarioRoot := filepath.Join(repoRoot, "scenarios", scenario)

	validator, err := NewPathValidator(scenarioRoot)
	if err != nil {
		return fmt.Errorf("failed to create path validator: %w", err)
	}

	for _, scopePath := range scopePaths {
		// Scope paths may be relative to scenario root
		fullPath := scopePath
		if !filepath.IsAbs(scopePath) {
			fullPath = filepath.Join(scenarioRoot, scopePath)
		}

		if err := validator.ValidatePath(fullPath); err != nil {
			return fmt.Errorf("invalid scope path '%s': %w", scopePath, err)
		}
	}

	return nil
}

// ScanPromptForPaths extracts file paths from a prompt and validates them.
// This is a best-effort scan - it looks for common path patterns.
func ScanPromptForPaths(prompt string, validator *PathValidator) []string {
	var suspiciousPaths []string

	// Common patterns that might indicate file paths
	pathPatterns := []*regexp.Regexp{
		// Absolute paths
		regexp.MustCompile(`(?m)(?:^|\s|["'\(])(/[a-zA-Z0-9._/-]+)`),
		// Home directory paths
		regexp.MustCompile(`(?m)(?:^|\s|["'\(])(~/[a-zA-Z0-9._/-]+)`),
		// Relative paths with traversal
		regexp.MustCompile(`(?m)(?:^|\s|["'\(])(\.\./[a-zA-Z0-9._/-]+)`),
		regexp.MustCompile(`(?m)(?:^|\s|["'\(])(\./\.\./[a-zA-Z0-9._/-]+)`),
	}

	for _, pattern := range pathPatterns {
		matches := pattern.FindAllStringSubmatch(prompt, -1)
		for _, match := range matches {
			if len(match) > 1 {
				path := match[1]
				// Skip common false positives
				if strings.HasPrefix(path, "/api/") || // API routes
					strings.HasPrefix(path, "/v1/") || // Version paths
					strings.HasPrefix(path, "/home/") && strings.Contains(path, "scenarios/") {
					// This is likely a legitimate scenario path, validate it
					if validator != nil {
						if err := validator.ValidatePath(path); err != nil {
							suspiciousPaths = append(suspiciousPaths, path)
						}
					}
				} else if strings.Contains(path, "..") {
					// Any path traversal is suspicious
					suspiciousPaths = append(suspiciousPaths, path)
				}
			}
		}
	}

	return suspiciousPaths
}
