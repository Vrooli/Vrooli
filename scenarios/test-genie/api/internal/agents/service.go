package agents

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

// ProcessChecker is a seam for checking if OS processes are alive.
// This interface enables testing without making actual syscalls.
type ProcessChecker interface {
	IsAlive(pid int) bool
}

// EnvironmentProvider is a seam for accessing environment variables.
// This enables testing configuration without modifying actual environment.
type EnvironmentProvider interface {
	Getenv(key string) string
	Hostname() (string, error)
}

// TimeProvider is a seam for time-dependent operations.
// This enables deterministic testing of time-based logic.
type TimeProvider interface {
	Now() time.Time
}

// --- Default implementations ---

// OSProcessChecker checks process existence using OS signals.
type OSProcessChecker struct{}

// IsAlive checks if a process is running using signal 0.
func (c *OSProcessChecker) IsAlive(pid int) bool {
	return isProcessAlive(pid)
}

// OSEnvironmentProvider uses the real OS environment.
type OSEnvironmentProvider struct{}

// Getenv returns an environment variable value.
func (p *OSEnvironmentProvider) Getenv(key string) string {
	return os.Getenv(key)
}

// Hostname returns the system hostname.
func (p *OSEnvironmentProvider) Hostname() (string, error) {
	return os.Hostname()
}

// RealTimeProvider uses actual system time.
type RealTimeProvider struct{}

// Now returns the current time.
func (p *RealTimeProvider) Now() time.Time {
	return time.Now()
}

// DefaultLockTimeout is the default duration before a scope lock expires.
// Deprecated: Use Config.LockTimeout() instead.
const DefaultLockTimeout = 20 * time.Minute

// DefaultRetentionDays is the default number of days to keep completed agents.
// Deprecated: Use Config.RetentionDays instead.
const DefaultRetentionDays = 7

// HeartbeatInterval is how often agents should send heartbeats.
// This is now derived from Config.HeartbeatInterval() for new code.
var HeartbeatInterval = 5 * time.Minute

// AgentService manages agent lifecycle and coordination.
type AgentService struct {
	repo   AgentRepository
	config Config

	// Seams for testability - allow substitution of OS-level dependencies
	processChecker ProcessChecker
	envProvider    EnvironmentProvider
	timeProvider   TimeProvider

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

// AgentServiceOption configures an AgentService during construction.
type AgentServiceOption func(*AgentService)

// WithProcessChecker sets a custom process checker for testing.
func WithProcessChecker(pc ProcessChecker) AgentServiceOption {
	return func(s *AgentService) {
		s.processChecker = pc
	}
}

// WithEnvironmentProvider sets a custom environment provider for testing.
func WithEnvironmentProvider(ep EnvironmentProvider) AgentServiceOption {
	return func(s *AgentService) {
		s.envProvider = ep
	}
}

// WithTimeProvider sets a custom time provider for testing.
func WithTimeProvider(tp TimeProvider) AgentServiceOption {
	return func(s *AgentService) {
		s.timeProvider = tp
	}
}

// WithConfig sets a custom configuration for the agent service.
func WithConfig(cfg Config) AgentServiceOption {
	return func(s *AgentService) {
		s.config = cfg
	}
}

// NewAgentService creates a new agent service with the given repository.
// Optional functional options can customize seam implementations for testing.
func NewAgentService(repo AgentRepository, opts ...AgentServiceOption) *AgentService {
	s := &AgentService{
		repo:           repo,
		config:         LoadConfigFromEnv(),
		runtime:        make(map[string]*runtimeState),
		cleanupC:       make(chan struct{}),
		processChecker: &OSProcessChecker{},
		envProvider:    &OSEnvironmentProvider{},
		timeProvider:   &RealTimeProvider{},
	}

	// Apply any provided options (may override config)
	for _, opt := range opts {
		opt(s)
	}

	// Validate config and apply any derived values
	_ = s.config.Validate()

	// Update the global HeartbeatInterval for backward compatibility
	HeartbeatInterval = s.config.HeartbeatInterval()

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
	ticker := time.NewTicker(s.config.CleanupInterval())
	defer ticker.Stop()

	for {
		select {
		case <-s.cleanupC:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			cutoff := s.timeProvider.Now().Add(-s.config.RetentionDuration())
			_, _ = s.repo.DeleteOlderThan(ctx, cutoff)
			cancel()
		}
	}
}

// OrphanClassification describes why an agent is considered orphaned or still alive.
type OrphanClassification struct {
	IsOrphaned bool
	Reason     string
}

// ClassifyAgentOrphanStatus determines whether an agent should be considered orphaned.
// An agent is orphaned if:
//   - It has no PID recorded (legacy agent or registration failed before process start)
//   - It's running on a different host than the current server
//   - Its process ID is no longer alive on the current host
//
// An agent is NOT orphaned (still alive) if:
//   - It has a valid PID on the current host AND the process is still running
func (s *AgentService) ClassifyAgentOrphanStatus(agent *SpawnedAgent, currentHostname string) OrphanClassification {
	// No PID recorded - agent never fully started or is legacy
	if agent.PID <= 0 {
		return OrphanClassification{
			IsOrphaned: true,
			Reason:     "Agent orphaned: server restarted while agent was active",
		}
	}

	// Different host - we can't check if the process is alive
	if agent.Hostname != currentHostname {
		return OrphanClassification{
			IsOrphaned: true,
			Reason:     fmt.Sprintf("Agent orphaned: process %d no longer running (hostname: %s)", agent.PID, agent.Hostname),
		}
	}

	// Same host - check if process is still running
	if s.processChecker.IsAlive(agent.PID) {
		return OrphanClassification{
			IsOrphaned: false,
			Reason:     "Process still running",
		}
	}

	// Process is dead on this host
	return OrphanClassification{
		IsOrphaned: true,
		Reason:     fmt.Sprintf("Agent orphaned: process %d no longer running (hostname: %s)", agent.PID, agent.Hostname),
	}
}

// cleanupOrphanedAgents marks any agents that were running/pending as failed.
// This should be called on startup to clean up agents from previous server runs
// that were interrupted (e.g., server restart, crash).
// Uses ClassifyAgentOrphanStatus to determine which agents are truly orphaned.
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

	currentHostname, _ := s.envProvider.Hostname()
	orphanCount := 0
	stillAliveCount := 0
	failedStatus := AgentStatusFailed

	for _, agent := range activeAgents {
		classification := s.ClassifyAgentOrphanStatus(agent, currentHostname)

		if !classification.IsOrphaned {
			stillAliveCount++
			continue
		}

		// Mark orphaned agent as failed
		err := s.repo.Update(ctx, agent.ID, UpdateAgentInput{
			Status: &failedStatus,
			Error:  &classification.Reason,
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

// ScopeExpansionDecision describes the outcome of scope expansion analysis.
type ScopeExpansionDecision struct {
	ShouldExpand  bool
	Reason        string
	ImplicitPaths []string
	ExpandedScope []string
	OriginalScope []string
}

// DecideScopeExpansion determines whether and how to expand the requested scope.
// This is the central decision point for implicit path locking.
//
// Decision criteria:
//   - Empty scope (entire scenario): NO EXPANSION - the full scenario already includes all files
//   - Non-empty scope: EXPAND - add shared dependency files to prevent conflicts
//
// Why expand scope?
// Dependency files (go.mod, package.json, etc.) can be modified by any code change.
// If Agent A modifies api/handler.go and Agent B modifies cli/command.go, both might
// trigger changes to go.mod. Without implicit locking, this causes merge conflicts.
//
// The expansion ensures agents automatically coordinate on shared resources without
// requiring explicit specification of every potentially-affected file.
func DecideScopeExpansion(requestedScope []string) ScopeExpansionDecision {
	decision := ScopeExpansionDecision{
		OriginalScope: requestedScope,
		ImplicitPaths: SharedDependencyFiles,
	}

	// Decision: Empty scope means full scenario access
	if len(requestedScope) == 0 {
		decision.ShouldExpand = false
		decision.Reason = "full scenario scope already includes all files; no expansion needed"
		decision.ExpandedScope = requestedScope
		return decision
	}

	// Decision: Non-empty scope needs implicit dependency file protection
	decision.ShouldExpand = true
	decision.Reason = "adding shared dependency files to prevent concurrent modification conflicts"

	// Build expanded scope with deduplication
	scopeMap := make(map[string]bool)
	for _, path := range requestedScope {
		scopeMap[path] = true
	}
	for _, depFile := range SharedDependencyFiles {
		scopeMap[depFile] = true
	}

	expanded := make([]string, 0, len(scopeMap))
	for path := range scopeMap {
		expanded = append(expanded, path)
	}

	decision.ExpandedScope = expanded
	return decision
}

// ExpandScopeWithImplicitPaths adds shared dependency files to the scope
// if the agent might modify them (e.g., working on any path that could
// trigger dependency changes). Returns the expanded scope.
//
// For detailed decision information, use DecideScopeExpansion instead.
func ExpandScopeWithImplicitPaths(requestedScope []string) []string {
	return DecideScopeExpansion(requestedScope).ExpandedScope
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
		expiresAt := s.timeProvider.Now().Add(s.config.LockTimeout())
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
	newExpiry := s.timeProvider.Now().Add(s.config.LockTimeout())
	return s.repo.RenewLocks(ctx, agentID, newExpiry)
}

// CleanupCompleted removes completed agents older than the given duration.
func (s *AgentService) CleanupCompleted(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := s.timeProvider.Now().Add(-olderThan)
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
	hostname, _ := s.envProvider.Hostname()
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
	return s.config.LockTimeout()
}

// GetConfig returns the current agent configuration.
// This allows callers to access tunable levers for operations that need them.
func (s *AgentService) GetConfig() Config {
	return s.config
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
