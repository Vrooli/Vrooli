// Package repository provides persistence interfaces for domain entities.
//
// This package defines the SEAM for data persistence. All domain entities
// are persisted through these interfaces, enabling:
// - Different storage backends (PostgreSQL, SQLite, in-memory)
// - Easy testing with mock repositories
// - Database migrations without changing domain logic
package repository

import (
	"context"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// Common Types
// -----------------------------------------------------------------------------

// ListFilter specifies pagination for list operations.
type ListFilter struct {
	Limit  int
	Offset int
}

// RunListFilter extends ListFilter with run-specific filters.
type RunListFilter struct {
	ListFilter
	TaskID         *uuid.UUID
	AgentProfileID *uuid.UUID
	Status         *domain.RunStatus
	TagPrefix      string // Filter runs by tag prefix (e.g., "ecosystem-" to get all ecosystem-manager runs)
}

// -----------------------------------------------------------------------------
// ProfileRepository - AgentProfile persistence
// -----------------------------------------------------------------------------

// ProfileRepository provides persistence for AgentProfile entities.
type ProfileRepository interface {
	// Create stores a new agent profile.
	Create(ctx context.Context, profile *domain.AgentProfile) error

	// Get retrieves a profile by ID.
	Get(ctx context.Context, id uuid.UUID) (*domain.AgentProfile, error)

	// GetByName retrieves a profile by name.
	GetByName(ctx context.Context, name string) (*domain.AgentProfile, error)

	// List retrieves profiles with optional filtering.
	List(ctx context.Context, filter ListFilter) ([]*domain.AgentProfile, error)

	// Update modifies an existing profile.
	Update(ctx context.Context, profile *domain.AgentProfile) error

	// Delete removes a profile by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}

// -----------------------------------------------------------------------------
// TaskRepository - Task persistence
// -----------------------------------------------------------------------------

// TaskRepository provides persistence for Task entities.
type TaskRepository interface {
	// Create stores a new task.
	Create(ctx context.Context, task *domain.Task) error

	// Get retrieves a task by ID.
	Get(ctx context.Context, id uuid.UUID) (*domain.Task, error)

	// List retrieves tasks with optional filtering.
	List(ctx context.Context, filter ListFilter) ([]*domain.Task, error)

	// ListByStatus retrieves tasks with a specific status.
	ListByStatus(ctx context.Context, status domain.TaskStatus, filter ListFilter) ([]*domain.Task, error)

	// Update modifies an existing task.
	Update(ctx context.Context, task *domain.Task) error

	// Delete removes a task by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}

// -----------------------------------------------------------------------------
// RunRepository - Run persistence
// -----------------------------------------------------------------------------

// RunRepository provides persistence for Run entities.
type RunRepository interface {
	// Create stores a new run.
	Create(ctx context.Context, run *domain.Run) error

	// Get retrieves a run by ID.
	Get(ctx context.Context, id uuid.UUID) (*domain.Run, error)

	// List retrieves runs with optional filtering.
	List(ctx context.Context, filter RunListFilter) ([]*domain.Run, error)

	// ListByTask retrieves runs for a specific task.
	ListByTask(ctx context.Context, taskID uuid.UUID, filter ListFilter) ([]*domain.Run, error)

	// Update modifies an existing run.
	Update(ctx context.Context, run *domain.Run) error

	// Delete removes a run by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// CountByStatus returns the count of runs by status.
	CountByStatus(ctx context.Context, status domain.RunStatus) (int, error)
}

// -----------------------------------------------------------------------------
// EventRepository - RunEvent persistence
// -----------------------------------------------------------------------------

// EventRepository provides persistence for RunEvent entities.
type EventRepository interface {
	// Append adds events to a run's event stream.
	Append(ctx context.Context, runID uuid.UUID, events ...*domain.RunEvent) error

	// Get retrieves events for a run.
	Get(ctx context.Context, runID uuid.UUID, afterSequence int64, limit int) ([]*domain.RunEvent, error)

	// GetByType retrieves events of specific types.
	GetByType(ctx context.Context, runID uuid.UUID, types []domain.RunEventType, limit int) ([]*domain.RunEvent, error)

	// Count returns the number of events for a run.
	Count(ctx context.Context, runID uuid.UUID) (int64, error)

	// Delete removes all events for a run.
	Delete(ctx context.Context, runID uuid.UUID) error
}

// -----------------------------------------------------------------------------
// PolicyRepository - Policy persistence
// -----------------------------------------------------------------------------

// PolicyRepository provides persistence for Policy entities.
type PolicyRepository interface {
	// Create stores a new policy.
	Create(ctx context.Context, policy *domain.Policy) error

	// Get retrieves a policy by ID.
	Get(ctx context.Context, id uuid.UUID) (*domain.Policy, error)

	// List retrieves policies with optional filtering.
	List(ctx context.Context, filter ListFilter) ([]*domain.Policy, error)

	// ListEnabled retrieves only enabled policies.
	ListEnabled(ctx context.Context) ([]*domain.Policy, error)

	// Update modifies an existing policy.
	Update(ctx context.Context, policy *domain.Policy) error

	// Delete removes a policy by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByScope finds policies matching a scope pattern.
	FindByScope(ctx context.Context, scopePath string) ([]*domain.Policy, error)
}

// -----------------------------------------------------------------------------
// LockRepository - ScopeLock persistence
// -----------------------------------------------------------------------------

// LockRepository provides persistence for ScopeLock entities.
type LockRepository interface {
	// Acquire attempts to acquire a lock.
	Acquire(ctx context.Context, lock *domain.ScopeLock) error

	// Release releases a lock by ID.
	Release(ctx context.Context, id uuid.UUID) error

	// ReleaseByRun releases all locks for a run.
	ReleaseByRun(ctx context.Context, runID uuid.UUID) error

	// Check finds overlapping locks for a scope.
	Check(ctx context.Context, scopePath, projectRoot string) ([]*domain.ScopeLock, error)

	// Refresh extends a lock's expiration.
	Refresh(ctx context.Context, id uuid.UUID, newExpiry int64) error

	// CleanupExpired removes expired locks.
	CleanupExpired(ctx context.Context) (int, error)
}

// -----------------------------------------------------------------------------
// CheckpointRepository - Run checkpoint persistence
// -----------------------------------------------------------------------------

// CheckpointRepository provides persistence for run checkpoints.
// Checkpoints enable safe interruption and resumption of runs.
type CheckpointRepository interface {
	// Save stores a checkpoint, overwriting any existing checkpoint for the run.
	Save(ctx context.Context, checkpoint *domain.RunCheckpoint) error

	// Get retrieves the latest checkpoint for a run.
	Get(ctx context.Context, runID uuid.UUID) (*domain.RunCheckpoint, error)

	// Delete removes all checkpoints for a run.
	Delete(ctx context.Context, runID uuid.UUID) error

	// ListStale finds runs with checkpoints older than the given duration.
	// This helps identify runs that may have stalled.
	ListStale(ctx context.Context, olderThan time.Duration) ([]*domain.RunCheckpoint, error)

	// Heartbeat updates the last heartbeat time for a checkpoint.
	Heartbeat(ctx context.Context, runID uuid.UUID) error
}

// -----------------------------------------------------------------------------
// IdempotencyRepository - Idempotency record persistence
// -----------------------------------------------------------------------------

// IdempotencyRepository provides persistence for idempotency records.
// This enables replay-safe operations that can be safely retried.
type IdempotencyRepository interface {
	// Check looks up an existing idempotency record.
	// Returns nil, nil if no record exists.
	Check(ctx context.Context, key string) (*domain.IdempotencyRecord, error)

	// Reserve creates a pending idempotency record.
	// Returns an error if the key already exists.
	Reserve(ctx context.Context, key string, ttl time.Duration) (*domain.IdempotencyRecord, error)

	// Complete marks an idempotency record as complete with the result.
	Complete(ctx context.Context, key string, entityID uuid.UUID, entityType string, response []byte) error

	// Fail marks an idempotency record as failed (allows retry).
	Fail(ctx context.Context, key string) error

	// CleanupExpired removes expired idempotency records.
	CleanupExpired(ctx context.Context) (int, error)
}
