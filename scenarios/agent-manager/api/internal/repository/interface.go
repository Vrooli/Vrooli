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
