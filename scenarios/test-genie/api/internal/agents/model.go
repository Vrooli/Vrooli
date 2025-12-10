package agents

import (
	"context"
	"os/exec"
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

// IsTerminal returns true if the status represents a final state.
func (s AgentStatus) IsTerminal() bool {
	return s == AgentStatusCompleted || s == AgentStatusFailed ||
		s == AgentStatusTimeout || s == AgentStatusStopped
}

// SpawnedAgent represents an agent tracked in the database.
type SpawnedAgent struct {
	ID             string      `json:"id"`
	IdempotencyKey string      `json:"idempotencyKey,omitempty"`
	SessionID      string      `json:"sessionId,omitempty"`
	Scenario       string      `json:"scenario"`
	Scope          []string    `json:"scope"`
	Phases         []string    `json:"phases,omitempty"`
	Model          string      `json:"model"`
	Status         AgentStatus `json:"status"`
	PromptHash     string      `json:"promptHash"`
	PromptIndex    int         `json:"promptIndex"`
	PromptText     string      `json:"promptText,omitempty"`
	Output         string      `json:"output,omitempty"`
	Error          string      `json:"error,omitempty"`
	StartedAt      time.Time   `json:"startedAt"`
	CompletedAt    *time.Time  `json:"completedAt,omitempty"`
	CreatedAt      time.Time   `json:"createdAt"`
	UpdatedAt      time.Time   `json:"updatedAt"`

	// Process tracking for orphan detection
	PID      int    `json:"pid,omitempty"`      // OS process ID when running
	Hostname string `json:"hostname,omitempty"` // Host where the process runs

	// Runtime-only fields (not persisted)
	Cancel context.CancelFunc `json:"-"`
	Cmd    *exec.Cmd          `json:"-"`
}

// AgentScopeLock represents a lock on paths within a scenario.
type AgentScopeLock struct {
	ID         int       `json:"id"`
	AgentID    string    `json:"agentId"`
	Scenario   string    `json:"scenario"`
	Path       string    `json:"path"`
	AcquiredAt time.Time `json:"acquiredAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
	RenewedAt  time.Time `json:"renewedAt"`
}

// ScopeLockInfo provides detailed lock information for conflict responses.
type ScopeLockInfo struct {
	Path      string    `json:"path"`
	AgentID   string    `json:"agentId"`
	Scenario  string    `json:"scenario"`
	StartedAt time.Time `json:"startedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// ConflictDetail provides information about a specific conflict.
type ConflictDetail struct {
	Path     string        `json:"path"`
	LockedBy ScopeLockInfo `json:"lockedBy"`
}

// CreateAgentInput contains the data needed to create a new agent.
type CreateAgentInput struct {
	ID             string
	IdempotencyKey string // Client-provided key for deduplication
	Scenario       string
	Scope          []string
	Phases         []string
	Model          string
	PromptHash     string
	PromptIndex    int
	PromptText     string
}

// UpdateAgentInput contains fields that can be updated on an agent.
type UpdateAgentInput struct {
	Status    *AgentStatus
	SessionID *string
	Output    *string
	Error     *string
	PID       *int    // Process ID for tracking
	Hostname  *string // Host where process runs
}

// AgentListOptions controls filtering and pagination for agent queries.
type AgentListOptions struct {
	Scenario    string
	StatusIn    []AgentStatus
	ActiveOnly  bool
	Limit       int
	OlderThan   *time.Time
	IncludeText bool
}

// SpawnIntent represents a pending spawn request used for idempotency.
type SpawnIntent struct {
	Key        string    `json:"key"`
	Scenario   string    `json:"scenario"`
	Scope      []string  `json:"scope"`
	AgentID    string    `json:"agentId,omitempty"` // Set after agent is created
	Status     string    `json:"status"`            // "pending", "completed", "failed"
	CreatedAt  time.Time `json:"createdAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
	ResultJSON string    `json:"resultJson,omitempty"` // Cached spawn result for replay
}

// FileOperation represents a tracked file operation by an agent.
type FileOperation struct {
	ID            int       `json:"id"`
	AgentID       string    `json:"agentId"`
	Scenario      string    `json:"scenario"`
	Operation     string    `json:"operation"` // "create", "modify", "delete"
	FilePath      string    `json:"filePath"`
	ContentHash   string    `json:"contentHash,omitempty"`
	ContentBefore string    `json:"contentBefore,omitempty"`
	ContentAfter  string    `json:"contentAfter,omitempty"`
	RecordedAt    time.Time `json:"recordedAt"`
}

// FileOperationInput contains the data needed to record a file operation.
type FileOperationInput struct {
	AgentID       string
	Scenario      string
	Operation     string // "create", "modify", "delete"
	FilePath      string
	ContentHash   string
	ContentBefore string
	ContentAfter  string
}

// SpawnSession represents a server-side session tracking spawn activity.
// This replaces browser-only sessionStorage for cross-tab/browser conflict detection.
type SpawnSession struct {
	ID             int       `json:"id"`
	UserIdentifier string    `json:"userIdentifier"` // IP address, API key, or user ID
	Scenario       string    `json:"scenario"`
	Scope          []string  `json:"scope"`
	Phases         []string  `json:"phases"`
	AgentIDs       []string  `json:"agentIds"`
	Status         string    `json:"status"` // "active", "completed", "failed", "cleared"
	CreatedAt      time.Time `json:"createdAt"`
	ExpiresAt      time.Time `json:"expiresAt"`
	LastActivityAt time.Time `json:"lastActivityAt"`
}

// CreateSpawnSessionInput contains data for creating a spawn session.
type CreateSpawnSessionInput struct {
	UserIdentifier string
	Scenario       string
	Scope          []string
	Phases         []string
	AgentIDs       []string
	TTL            time.Duration
}

// SpawnSessionConflict provides details about a conflicting spawn session.
type SpawnSessionConflict struct {
	SessionID int       `json:"sessionId"`
	Scenario  string    `json:"scenario"`
	Scope     []string  `json:"scope"`
	AgentIDs  []string  `json:"agentIds"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// AgentRepository defines the interface for agent persistence.
type AgentRepository interface {
	// Create inserts a new agent record.
	Create(ctx context.Context, input CreateAgentInput) (*SpawnedAgent, error)

	// Get retrieves an agent by ID.
	Get(ctx context.Context, id string) (*SpawnedAgent, error)

	// GetByIdempotencyKey retrieves an agent by its idempotency key.
	GetByIdempotencyKey(ctx context.Context, key string) (*SpawnedAgent, error)

	// Update modifies an existing agent.
	Update(ctx context.Context, id string, input UpdateAgentInput) error

	// List returns agents matching the given options.
	List(ctx context.Context, opts AgentListOptions) ([]*SpawnedAgent, error)

	// Delete removes an agent and its scope locks.
	Delete(ctx context.Context, id string) error

	// DeleteOlderThan removes completed agents older than the given time.
	DeleteOlderThan(ctx context.Context, completedBefore time.Time) (int64, error)

	// AcquireLocks creates scope locks for an agent.
	AcquireLocks(ctx context.Context, agentID, scenario string, paths []string, expiresAt time.Time) error

	// ReleaseLocks removes all locks for an agent.
	ReleaseLocks(ctx context.Context, agentID string) error

	// RenewLocks extends the expiration of locks for an agent.
	RenewLocks(ctx context.Context, agentID string, newExpiry time.Time) error

	// GetActiveLocks returns all non-expired locks.
	GetActiveLocks(ctx context.Context) ([]*AgentScopeLock, error)

	// GetLocksForScenario returns non-expired locks for a specific scenario.
	GetLocksForScenario(ctx context.Context, scenario string) ([]*AgentScopeLock, error)

	// CheckConflicts returns lock details for paths that overlap with the given scope.
	CheckConflicts(ctx context.Context, scenario string, paths []string) ([]ConflictDetail, error)

	// AcquireSpawnIntent atomically acquires a spawn intent lock.
	// Returns the existing intent if the key already exists (idempotent).
	AcquireSpawnIntent(ctx context.Context, key, scenario string, scope []string, ttl time.Duration) (*SpawnIntent, bool, error)

	// UpdateSpawnIntent updates an existing spawn intent with result info.
	UpdateSpawnIntent(ctx context.Context, key string, agentID, status, resultJSON string) error

	// GetSpawnIntent retrieves a spawn intent by key.
	GetSpawnIntent(ctx context.Context, key string) (*SpawnIntent, error)

	// CleanupSpawnIntents removes expired spawn intents.
	CleanupSpawnIntents(ctx context.Context) (int64, error)

	// RecordFileOperation logs a file operation by an agent.
	RecordFileOperation(ctx context.Context, input FileOperationInput) error

	// GetFileOperations returns file operations for an agent.
	GetFileOperations(ctx context.Context, agentID string) ([]*FileOperation, error)

	// GetFileOperationsForScenario returns file operations for a scenario.
	GetFileOperationsForScenario(ctx context.Context, scenario string, limit int) ([]*FileOperation, error)

	// CreateSpawnSession creates a new server-side spawn session.
	CreateSpawnSession(ctx context.Context, input CreateSpawnSessionInput) (*SpawnSession, error)

	// GetActiveSpawnSessions returns active spawn sessions for a user/scenario.
	GetActiveSpawnSessions(ctx context.Context, userIdentifier, scenario string) ([]*SpawnSession, error)

	// CheckSpawnSessionConflicts checks for conflicting active spawn sessions.
	CheckSpawnSessionConflicts(ctx context.Context, userIdentifier, scenario string, scope []string) ([]SpawnSessionConflict, error)

	// UpdateSpawnSessionStatus updates the status of a spawn session.
	UpdateSpawnSessionStatus(ctx context.Context, sessionID int, status string) error

	// ClearSpawnSessions marks all active spawn sessions for a user as cleared.
	ClearSpawnSessions(ctx context.Context, userIdentifier string) (int64, error)

	// CleanupExpiredSpawnSessions removes expired spawn sessions.
	CleanupExpiredSpawnSessions(ctx context.Context) (int64, error)
}
