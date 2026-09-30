package health

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// AuditDB is the exact database capability required by the health audit
// store. Keeping it here prevents the health substrate from depending on a
// sibling package merely for a shared test/SQL interface.
type AuditDB interface {
	Exec(string, ...any) (sql.Result, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryxContext(context.Context, string, ...any) (*sqlx.Rows, error)
	GetContext(context.Context, any, string, ...any) error
	SelectContext(context.Context, any, string, ...any) error
	Conn(context.Context) (*sql.Conn, error)
}

// FailureInput contains the probe signals available to a classifier. Health
// owns this minimal observation shape; runner-specific failure taxonomies are
// adapted by the composition root.
type FailureInput struct {
	RunnerType string
	Stderr     string
	Cause      error
}

// ClassifiedFailure is the structured failure observation Health needs to
// persist. An empty reason denotes a classifier that could not classify it.
type ClassifiedFailure struct {
	Reason  string
	Message string
}

// FailureClassifier translates probe failures into durable observations.
type FailureClassifier interface {
	Classify(FailureInput) *ClassifiedFailure
}

type unknownFailureClassifier struct{}

func (unknownFailureClassifier) Classify(FailureInput) *ClassifiedFailure { return nil }
