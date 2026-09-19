// Package holders owns authenticated holder identity and scoped reads.
package holders

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Holder struct {
	ID                   string
	DisplayName          string
	AuthenticatorSubject string
	CreatedAt            time.Time
}

type Scope struct {
	AuthenticatorSubject string
	HolderID             string
}

var (
	ErrHolderNotFound = errors.New("holder not found")
	ErrInvalidHolder  = errors.New("invalid holder")
)

type Repository interface {
	Create(context.Context, Holder) (Holder, error)
	CreateIdempotent(context.Context, Holder, string) (Holder, error)
	Get(context.Context, string) (Holder, error)
	List(context.Context) ([]Holder, error)
	GetScoped(context.Context, Scope) (Holder, bool, error)
	GetBySubject(context.Context, string) (Holder, error)
	Owns(context.Context, string, string) (bool, error)
}

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) *sqliteRepository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Create(ctx context.Context, holder Holder) (Holder, error) {
	if err := validateHolder(holder); err != nil {
		return Holder{}, err
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO holders (id, display_name, authenticator_subject, created_at)
		VALUES (?, ?, ?, ?)`,
		holder.ID, holder.DisplayName, holder.AuthenticatorSubject, formatTime(holder.CreatedAt),
	)
	if err != nil {
		return Holder{}, fmt.Errorf("create holder: %w", err)
	}
	return holder, nil
}

func (r *sqliteRepository) CreateIdempotent(ctx context.Context, holder Holder, idempotencyKey string) (Holder, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return Holder{}, fmt.Errorf("%w: idempotency key is required", ErrInvalidHolder)
	}
	if err := validateHolder(holder); err != nil {
		return Holder{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Holder{}, fmt.Errorf("begin holder creation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	existing, err := scanHolder(tx.QueryRowContext(ctx, `
		SELECT h.id, h.display_name, h.authenticator_subject, h.created_at
		FROM holder_create_receipts receipt
		JOIN holders h ON h.id = receipt.holder_id
		WHERE receipt.idempotency_key = ?`, idempotencyKey))
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Holder{}, fmt.Errorf("read holder creation receipt: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO holders (id, display_name, authenticator_subject, created_at)
		VALUES (?, ?, ?, ?)`, holder.ID, holder.DisplayName, holder.AuthenticatorSubject, formatTime(holder.CreatedAt)); err != nil {
		return Holder{}, fmt.Errorf("create holder: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO holder_create_receipts (idempotency_key, holder_id, created_at)
		VALUES (?, ?, ?)`, idempotencyKey, holder.ID, formatTime(holder.CreatedAt)); err != nil {
		return Holder{}, fmt.Errorf("store holder creation receipt: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Holder{}, fmt.Errorf("commit holder creation: %w", err)
	}
	return holder, nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Holder, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Holder{}, fmt.Errorf("%w: holder id is required", ErrInvalidHolder)
	}
	holder, err := scanHolder(r.db.QueryRowContext(ctx, `
		SELECT id, display_name, authenticator_subject, created_at FROM holders WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Holder{}, ErrHolderNotFound
	}
	if err != nil {
		return Holder{}, fmt.Errorf("read holder: %w", err)
	}
	return holder, nil
}

func (r *sqliteRepository) List(ctx context.Context) ([]Holder, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, display_name, authenticator_subject, created_at FROM holders ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("list holders: %w", err)
	}
	defer rows.Close()
	holders := make([]Holder, 0)
	for rows.Next() {
		holder, err := scanHolder(rows)
		if err != nil {
			return nil, fmt.Errorf("scan holder: %w", err)
		}
		holders = append(holders, holder)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate holders: %w", err)
	}
	return holders, nil
}

// GetScoped applies holder ownership inside the SQL query. A missing holder
// and an existing holder owned by another subject both return (zero, false,
// nil), so repository callers cannot use the result as an existence oracle.
func (r *sqliteRepository) GetScoped(ctx context.Context, scope Scope) (Holder, bool, error) {
	if err := validateScope(scope); err != nil {
		return Holder{}, false, err
	}
	holder, err := scanHolder(r.db.QueryRowContext(ctx, `
		SELECT id, display_name, authenticator_subject, created_at
		FROM holders
		WHERE id = ? AND authenticator_subject = ?`,
		scope.HolderID, scope.AuthenticatorSubject,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return Holder{}, false, nil
	}
	if err != nil {
		return Holder{}, false, fmt.Errorf("read scoped holder: %w", err)
	}
	return holder, true, nil
}

func (r *sqliteRepository) GetBySubject(ctx context.Context, subject string) (Holder, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return Holder{}, fmt.Errorf("%w: authenticator subject is required", ErrInvalidHolder)
	}
	holder, err := scanHolder(r.db.QueryRowContext(ctx, `
		SELECT id, display_name, authenticator_subject, created_at
		FROM holders WHERE authenticator_subject = ?`, subject))
	if errors.Is(err, sql.ErrNoRows) {
		return Holder{}, ErrHolderNotFound
	}
	if err != nil {
		return Holder{}, fmt.Errorf("read holder by subject: %w", err)
	}
	return holder, nil
}

// Owns is the narrow authorization port consumed by the journal's scoped
// history repository. Both absent and foreign holders return false.
func (r *sqliteRepository) Owns(ctx context.Context, holderID, subject string) (bool, error) {
	_, ok, err := r.GetScoped(ctx, Scope{HolderID: holderID, AuthenticatorSubject: subject})
	return ok, err
}

type rowScanner interface{ Scan(...any) error }

func scanHolder(row rowScanner) (Holder, error) {
	var holder Holder
	var createdAt string
	if err := row.Scan(&holder.ID, &holder.DisplayName, &holder.AuthenticatorSubject, &createdAt); err != nil {
		return Holder{}, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Holder{}, fmt.Errorf("parse holder creation time: %w", err)
	}
	holder.CreatedAt = parsed
	return holder, nil
}

func validateHolder(holder Holder) error {
	switch {
	case strings.TrimSpace(holder.ID) == "":
		return fmt.Errorf("%w: id is required", ErrInvalidHolder)
	case strings.TrimSpace(holder.DisplayName) == "":
		return fmt.Errorf("%w: display name is required", ErrInvalidHolder)
	case strings.TrimSpace(holder.AuthenticatorSubject) == "":
		return fmt.Errorf("%w: authenticator subject is required", ErrInvalidHolder)
	case holder.CreatedAt.IsZero():
		return fmt.Errorf("%w: created time is required", ErrInvalidHolder)
	default:
		return nil
	}
}

func validateScope(scope Scope) error {
	if strings.TrimSpace(scope.HolderID) == "" || strings.TrimSpace(scope.AuthenticatorSubject) == "" {
		return fmt.Errorf("%w: holder id and authenticator subject are required", ErrInvalidHolder)
	}
	return nil
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
