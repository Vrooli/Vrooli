// Package repository provides data access abstractions for the reference scenario.
// All database interactions flow through repository interfaces, allowing business
// logic to remain independent of storage implementation details.
//
// DOC: docs/concepts/ARCHITECTURE.md#data-access-layer
// DOC: docs/internal/SEAMS.md#repository-seam
// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/ERROR_SEMANTICS.md#repository-errors
package repository

import (
	"context"
	"database/sql"
	"errors"

	"reference-react-vite/api/domain/notes"
	"reference-react-vite/api/domain/projects"
	"reference-react-vite/api/domain/tasks"
)

// ╔════════════════════════════════════════════════════════════════════════════╗
// ║  REPOSITORY ERRORS - Typed error categories for data access operations     ║
// ║                                                                            ║
// ║  Use errors.Is() to check error categories instead of string matching.     ║
// ║  Example: if errors.Is(err, repository.ErrNotFound) { ... }                ║
// ╚════════════════════════════════════════════════════════════════════════════╝

// ErrNotFound is returned when a requested resource does not exist.
// Handlers should return 404 NOT_FOUND to clients.
var ErrNotFound = errors.New("resource not found")

// ErrConflict is returned when an operation would violate a constraint.
// Handlers should return 409 CONFLICT to clients.
var ErrConflict = errors.New("resource conflict")

// IsNotFound checks if an error indicates a missing resource.
// Use this for type-safe error checking instead of string comparison.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// TaskRepository defines the data access contract for tasks.
type TaskRepository interface {
	Create(ctx context.Context, task *tasks.Task) error
	FindByID(ctx context.Context, id string) (*tasks.Task, error)
	List(ctx context.Context, filter tasks.ListFilter) ([]*tasks.Task, int, error)
	Update(ctx context.Context, task *tasks.Task) error
	Delete(ctx context.Context, id string) error
}

// ProjectRepository defines the data access contract for projects.
type ProjectRepository interface {
	Create(ctx context.Context, project *projects.Project) error
	FindByID(ctx context.Context, id string) (*projects.Project, error)
	List(ctx context.Context, filter projects.ListFilter) ([]*projects.Project, int, error)
	Update(ctx context.Context, project *projects.Project) error
	Delete(ctx context.Context, id string) error
}

// NoteRepository defines the data access contract for notes.
type NoteRepository interface {
	Create(ctx context.Context, note *notes.Note) error
	FindByID(ctx context.Context, id string) (*notes.Note, error)
	ListByTask(ctx context.Context, filter notes.ListFilter) ([]*notes.Note, int, error)
	Update(ctx context.Context, note *notes.Note) error
	Delete(ctx context.Context, id string) error
}

// Repositories bundles all repository implementations for dependency injection.
type Repositories struct {
	Tasks    TaskRepository
	Projects ProjectRepository
	Notes    NoteRepository
}

// NewRepositories creates PostgreSQL-backed repositories from a database connection.
func NewRepositories(db *sql.DB) *Repositories {
	return &Repositories{
		Tasks:    NewPostgresTaskRepository(db),
		Projects: NewPostgresProjectRepository(db),
		Notes:    NewPostgresNoteRepository(db),
	}
}
