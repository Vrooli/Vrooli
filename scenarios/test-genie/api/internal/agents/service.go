package agents

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

const (
	// DefaultLockTimeout is the default duration before a scope lock expires.
	DefaultLockTimeout = 20 * time.Minute

	// DefaultRetentionDays is the default number of days to keep completed agents.
	DefaultRetentionDays = 7

	// HeartbeatInterval is how often agents should send heartbeats.
	HeartbeatInterval = 5 * time.Minute
)

// AgentService manages agent lifecycle and coordination.
type AgentService struct {
	repo        AgentRepository
	lockTimeout time.Duration

	// In-memory tracking of runtime state (cancel functions, commands)
	// These are not persisted but are needed for stopping agents.
	mu       sync.RWMutex
	runtime  map[string]*runtimeState
	cleanupC chan struct{}
}

type runtimeState struct {
	Cancel context.CancelFunc
	Cmd    interface{} // *exec.Cmd, kept as interface to avoid import cycle
}

// NewAgentService creates a new agent service with the given repository.
func NewAgentService(repo AgentRepository) *AgentService {
	lockTimeout := DefaultLockTimeout
	if envTimeout := os.Getenv("AGENT_LOCK_TIMEOUT_MINUTES"); envTimeout != "" {
		if minutes, err := strconv.Atoi(envTimeout); err == nil && minutes > 0 {
			lockTimeout = time.Duration(minutes) * time.Minute
		}
	}

	s := &AgentService{
		repo:        repo,
		lockTimeout: lockTimeout,
		runtime:     make(map[string]*runtimeState),
		cleanupC:    make(chan struct{}),
	}

	// Clean up any orphaned agents from previous runs
	// This marks agents that were running/pending as failed with an explanation
	s.cleanupOrphanedAgents()

	// Start background cleanup goroutine
	go s.runCleanupLoop()

	return s
}

// Close stops the background cleanup goroutine.
func (s *AgentService) Close() {
	close(s.cleanupC)
}

// runCleanupLoop periodically removes old completed agents.
func (s *AgentService) runCleanupLoop() {
	retentionDays := DefaultRetentionDays
	if envDays := os.Getenv("AGENT_RETENTION_DAYS"); envDays != "" {
		if days, err := strconv.Atoi(envDays); err == nil && days > 0 {
			retentionDays = days
		}
	}

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.cleanupC:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
			_, _ = s.repo.DeleteOlderThan(ctx, cutoff)
			cancel()
		}
	}
}

// cleanupOrphanedAgents marks any agents that were running/pending as failed.
// This should be called on startup to clean up agents from previous server runs
// that were interrupted (e.g., server restart, crash).
// Enhanced: Now checks PID validity to distinguish truly orphaned processes.
func (s *AgentService) cleanupOrphanedAgents() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get all active (non-terminal) agents
	activeAgents, err := s.repo.List(ctx, AgentListOptions{
		ActiveOnly: true,
		Limit:      1000, // Reasonable upper bound
	})
	if err != nil {
		// Log error but don't fail startup - this is best-effort cleanup
		return
	}

	if len(activeAgents) == 0 {
		return
	}

	currentHostname, _ := os.Hostname()
	orphanCount := 0
	stillAliveCount := 0
	failedStatus := AgentStatusFailed

	for _, agent := range activeAgents {
		// Check if the process is still alive
		if agent.PID > 0 && agent.Hostname == currentHostname {
			if isProcessAlive(agent.PID) {
				// Process is still running - don't mark as orphaned
				stillAliveCount++
				continue
			}
		}

		// Agent is orphaned: either no PID recorded, different host, or process is dead
		orphanError := "Agent orphaned: server restarted while agent was active"
		if agent.PID > 0 {
			orphanError = fmt.Sprintf("Agent orphaned: process %d no longer running (hostname: %s)", agent.PID, agent.Hostname)
		}

		err := s.repo.Update(ctx, agent.ID, UpdateAgentInput{
			Status: &failedStatus,
			Error:  &orphanError,
		})
		if err != nil {
			continue
		}

		// Release any scope locks they held
		_ = s.repo.ReleaseLocks(ctx, agent.ID)
		orphanCount++
	}

	// Note: We don't log here to avoid import cycle with logger
	// The count can be observed via the active agents endpoint
	_ = orphanCount
	_ = stillAliveCount
}

// isProcessAlive checks if a process with the given PID is still running.
// Uses kill -0 which doesn't send a signal but checks if the process exists.
func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	// os.FindProcess always succeeds on Unix, need to signal with 0 to check existence
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks if the process exists without actually sending a signal
	err = proc.Signal(os.Signal(syscall.Signal(0)))
	return err == nil
}

// SharedDependencyFiles defines files that should be implicitly locked
// when any agent works on a scenario. These files can cause conflicts
// if modified concurrently by multiple agents.
var SharedDependencyFiles = []string{
	// Go dependency files
	"go.mod",
	"go.sum",
	"go.work",
	"go.work.sum",
	// Node.js dependency files
	"package.json",
	"package-lock.json",
	"pnpm-lock.yaml",
	"yarn.lock",
	// Python dependency files
	"requirements.txt",
	"pyproject.toml",
	"poetry.lock",
	"Pipfile",
	"Pipfile.lock",
	// Rust dependency files
	"Cargo.toml",
	"Cargo.lock",
}

// ExpandScopeWithImplicitPaths adds shared dependency files to the scope
// if the agent might modify them (e.g., working on any path that could
// trigger dependency changes). Returns the expanded scope.
func ExpandScopeWithImplicitPaths(requestedScope []string) []string {
	if len(requestedScope) == 0 {
		// Entire scenario scope - implicit paths already covered
		return requestedScope
	}

	// Create a map for deduplication
	scopeMap := make(map[string]bool)
	for _, path := range requestedScope {
		scopeMap[path] = true
	}

	// Add shared dependency files to the scope
	// These are always at the root of the scenario, so just use the filename
	for _, depFile := range SharedDependencyFiles {
		scopeMap[depFile] = true
	}

	// Convert back to slice
	expanded := make([]string, 0, len(scopeMap))
	for path := range scopeMap {
		expanded = append(expanded, path)
	}

	return expanded
}

// Register creates a new agent and acquires scope locks.
// Returns an error if there are scope conflicts or duplicate prompts.
func (s *AgentService) Register(ctx context.Context, input CreateAgentInput) (*SpawnedAgent, error) {
	// Generate ID if not provided
	if input.ID == "" {
		input.ID = generateAgentID()
	}

	// Expand scope to include implicit shared paths (dependency files)
	// This prevents conflicts when multiple agents might modify shared files
	expandedScope := ExpandScopeWithImplicitPaths(input.Scope)

	// Check for scope conflicts first (using expanded scope)
	if len(expandedScope) > 0 {
		conflicts, err := s.repo.CheckConflicts(ctx, input.Scenario, expandedScope)
		if err != nil {
			return nil, fmt.Errorf("check conflicts: %w", err)
		}
		if len(conflicts) > 0 {
			return nil, fmt.Errorf("scope conflict with active agents")
		}
	}

	// Check for duplicate prompts (same hash, same scenario, running/pending)
	activeAgents, err := s.repo.List(ctx, AgentListOptions{
		Scenario:   input.Scenario,
		ActiveOnly: true,
	})
	if err != nil {
		return nil, fmt.Errorf("list active agents: %w", err)
	}
	for _, existing := range activeAgents {
		if existing.PromptHash == input.PromptHash {
			return nil, fmt.Errorf("duplicate agent: identical prompt already running (agent %s)", existing.ID)
		}
	}

	// Create the agent record (with original scope for display, not expanded)
	agent, err := s.repo.Create(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	// Acquire scope locks using expanded scope (includes shared files)
	if len(expandedScope) > 0 {
		expiresAt := time.Now().Add(s.lockTimeout)
		if err := s.repo.AcquireLocks(ctx, agent.ID, input.Scenario, expandedScope, expiresAt); err != nil {
			// Clean up the agent if lock acquisition fails
			_ = s.repo.Delete(ctx, agent.ID)
			return nil, fmt.Errorf("acquire locks: %w", err)
		}
	}

	return agent, nil
}

// UpdateStatus updates an agent's status and releases locks on terminal states.
func (s *AgentService) UpdateStatus(ctx context.Context, id string, status AgentStatus, sessionID, output, errMsg string) error {
	input := UpdateAgentInput{
		Status: &status,
	}
	if sessionID != "" {
		input.SessionID = &sessionID
	}
	if output != "" {
		input.Output = &output
	}
	if errMsg != "" {
		input.Error = &errMsg
	}

	if err := s.repo.Update(ctx, id, input); err != nil {
		return fmt.Errorf("update agent: %w", err)
	}

	// Release locks on terminal status
	if status.IsTerminal() {
		if err := s.repo.ReleaseLocks(ctx, id); err != nil {
			// Log but don't fail - agent is already updated
			fmt.Printf("warning: failed to release locks for agent %s: %v\n", id, err)
		}

		// Clean up runtime state
		s.mu.Lock()
		delete(s.runtime, id)
		s.mu.Unlock()
	}

	return nil
}

// Get retrieves an agent by ID.
func (s *AgentService) Get(ctx context.Context, id string) (*SpawnedAgent, error) {
	return s.repo.Get(ctx, id)
}

// ListActive returns all running or pending agents.
func (s *AgentService) ListActive(ctx context.Context) ([]*SpawnedAgent, error) {
	return s.repo.List(ctx, AgentListOptions{ActiveOnly: true})
}

// ListAll returns all agents up to the given limit.
func (s *AgentService) ListAll(ctx context.Context, limit int) ([]*SpawnedAgent, error) {
	return s.repo.List(ctx, AgentListOptions{Limit: limit})
}

// Stop attempts to stop an agent by ID.
func (s *AgentService) Stop(ctx context.Context, id string) error {
	agent, err := s.repo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("get agent: %w", err)
	}
	if agent == nil {
		return fmt.Errorf("agent not found: %s", id)
	}

	if agent.Status != AgentStatusRunning && agent.Status != AgentStatusPending {
		return fmt.Errorf("agent %s is not running (status: %s)", id, agent.Status)
	}

	// Cancel runtime state if available
	s.mu.Lock()
	if rt, ok := s.runtime[id]; ok {
		if rt.Cancel != nil {
			rt.Cancel()
		}
		// Kill process if available (type assertion to avoid import cycle)
		if cmd, ok := rt.Cmd.(interface{ Kill() error }); ok {
			_ = cmd.Kill()
		}
		delete(s.runtime, id)
	}
	s.mu.Unlock()

	// Update status and release locks
	errMsg := "stopped by user"
	return s.UpdateStatus(ctx, id, AgentStatusStopped, "", "", errMsg)
}

// StopAll stops all running agents and returns the stopped agent IDs.
func (s *AgentService) StopAll(ctx context.Context) ([]string, error) {
	activeAgents, err := s.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active agents: %w", err)
	}

	stoppedIDs := make([]string, 0, len(activeAgents))
	for _, agent := range activeAgents {
		if err := s.Stop(ctx, agent.ID); err != nil {
			// Continue stopping other agents even if one fails
			fmt.Printf("warning: failed to stop agent %s: %v\n", agent.ID, err)
			continue
		}
		stoppedIDs = append(stoppedIDs, agent.ID)
	}

	return stoppedIDs, nil
}

// CheckConflicts returns detailed conflict information for the given scope.
func (s *AgentService) CheckConflicts(ctx context.Context, scenario string, paths []string) ([]ConflictDetail, error) {
	return s.repo.CheckConflicts(ctx, scenario, paths)
}

// CheckConflictsSimple returns just the conflicting agent IDs (for backward compatibility).
func (s *AgentService) CheckConflictsSimple(ctx context.Context, scenario string, paths []string) ([]string, error) {
	conflicts, err := s.repo.CheckConflicts(ctx, scenario, paths)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	ids := make([]string, 0)
	for _, c := range conflicts {
		if !seen[c.LockedBy.AgentID] {
			seen[c.LockedBy.AgentID] = true
			ids = append(ids, c.LockedBy.AgentID)
		}
	}
	return ids, nil
}

// GetActiveLocks returns all non-expired scope locks.
func (s *AgentService) GetActiveLocks(ctx context.Context) ([]*AgentScopeLock, error) {
	return s.repo.GetActiveLocks(ctx)
}

// RenewLocks extends the lock timeout for an agent (heartbeat).
func (s *AgentService) RenewLocks(ctx context.Context, agentID string) error {
	newExpiry := time.Now().Add(s.lockTimeout)
	return s.repo.RenewLocks(ctx, agentID, newExpiry)
}

// CleanupCompleted removes completed agents older than the given duration.
func (s *AgentService) CleanupCompleted(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	return s.repo.DeleteOlderThan(ctx, cutoff)
}

// SetAgentProcess stores the cancel function and command for an agent.
// This allows the agent to be stopped later. Also records PID in the database.
func (s *AgentService) SetAgentProcess(id string, cancel context.CancelFunc, cmd interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runtime[id] = &runtimeState{
		Cancel: cancel,
		Cmd:    cmd,
	}
}

// SetAgentPID records the process ID and hostname in the database for orphan detection.
// Should be called after the process starts.
func (s *AgentService) SetAgentPID(ctx context.Context, id string, pid int) error {
	hostname, _ := os.Hostname()
	return s.repo.Update(ctx, id, UpdateAgentInput{
		PID:      &pid,
		Hostname: &hostname,
	})
}

// HashPrompt creates a deterministic hash of a prompt for deduplication.
func HashPrompt(prompt string) string {
	h := sha256.New()
	h.Write([]byte(prompt))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// generateAgentID creates a unique 8-character agent ID.
func generateAgentID() string {
	return uuid.New().String()[:8]
}

// GetLockTimeout returns the configured lock timeout duration.
func (s *AgentService) GetLockTimeout() time.Duration {
	return s.lockTimeout
}

// AcquireSpawnIntent atomically acquires a spawn intent lock for idempotency.
// Returns (intent, isNew, error). If isNew is false, the intent already existed.
func (s *AgentService) AcquireSpawnIntent(ctx context.Context, key, scenario string, scope []string, ttl time.Duration) (*SpawnIntent, bool, error) {
	return s.repo.AcquireSpawnIntent(ctx, key, scenario, scope, ttl)
}

// UpdateSpawnIntent updates an existing spawn intent with result info.
func (s *AgentService) UpdateSpawnIntent(ctx context.Context, key string, agentID, status, resultJSON string) error {
	return s.repo.UpdateSpawnIntent(ctx, key, agentID, status, resultJSON)
}

// GetSpawnIntent retrieves a spawn intent by key.
func (s *AgentService) GetSpawnIntent(ctx context.Context, key string) (*SpawnIntent, error) {
	return s.repo.GetSpawnIntent(ctx, key)
}

// RecordFileOperation logs a file operation by an agent for auditing.
func (s *AgentService) RecordFileOperation(ctx context.Context, input FileOperationInput) error {
	return s.repo.RecordFileOperation(ctx, input)
}

// GetFileOperations returns file operations for an agent.
func (s *AgentService) GetFileOperations(ctx context.Context, agentID string) ([]*FileOperation, error) {
	return s.repo.GetFileOperations(ctx, agentID)
}

// GetFileOperationsForScenario returns file operations for a scenario.
func (s *AgentService) GetFileOperationsForScenario(ctx context.Context, scenario string, limit int) ([]*FileOperation, error) {
	return s.repo.GetFileOperationsForScenario(ctx, scenario, limit)
}

// CreateSpawnSession creates a new server-side spawn session for tracking.
func (s *AgentService) CreateSpawnSession(ctx context.Context, input CreateSpawnSessionInput) (*SpawnSession, error) {
	return s.repo.CreateSpawnSession(ctx, input)
}

// GetActiveSpawnSessions returns active spawn sessions for a user.
func (s *AgentService) GetActiveSpawnSessions(ctx context.Context, userIdentifier, scenario string) ([]*SpawnSession, error) {
	return s.repo.GetActiveSpawnSessions(ctx, userIdentifier, scenario)
}

// CheckSpawnSessionConflicts checks for conflicting spawn sessions (server-side).
func (s *AgentService) CheckSpawnSessionConflicts(ctx context.Context, userIdentifier, scenario string, scope []string) ([]SpawnSessionConflict, error) {
	return s.repo.CheckSpawnSessionConflicts(ctx, userIdentifier, scenario, scope)
}

// UpdateSpawnSessionStatus updates the status of a spawn session.
func (s *AgentService) UpdateSpawnSessionStatus(ctx context.Context, sessionID int, status string) error {
	return s.repo.UpdateSpawnSessionStatus(ctx, sessionID, status)
}

// ClearSpawnSessions clears all active spawn sessions for a user.
func (s *AgentService) ClearSpawnSessions(ctx context.Context, userIdentifier string) (int64, error) {
	return s.repo.ClearSpawnSessions(ctx, userIdentifier)
}

// CleanupExpiredSpawnSessions removes expired spawn sessions.
func (s *AgentService) CleanupExpiredSpawnSessions(ctx context.Context) (int64, error) {
	return s.repo.CleanupExpiredSpawnSessions(ctx)
}
