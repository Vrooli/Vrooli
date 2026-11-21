package library

import (
	"context"
	"database/sql"
	"time"
)

type Repository interface {
	CreateRevision(context.Context, BundleRevision) error
	GetRevision(context.Context, string, uint32) (BundleRevision, error)
	ListRevisions(context.Context, string, uint32, int) ([]BundleRevision, error)
}

type SQLDatabase interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

var _ Repository = (*sqliteRepository)(nil)

type storedRevision struct {
	manifestJSON []byte
	manifestSHA  string
	reviewJSON   []byte
	reviewSHA    string
	publishedAt  time.Time
	artifactKeys map[string]string
}
