// Package mints owns token-type declarations and mint authority.
package mints

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type Repository interface {
	Create(context.Context, TokenType) (TokenType, error)
	Get(context.Context, string) (TokenType, error)
	List(context.Context, bool) ([]TokenType, error)
	Retire(context.Context, string, time.Time) (TokenType, error)
	Mint(context.Context, string, int64) (TokenType, error)
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Create(ctx context.Context, tokenType TokenType) (TokenType, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return TokenType{}, fmt.Errorf("begin token type creation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO token_types (id, name, symbol, color, supply_policy, cap_amount, minted_amount, retired, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?)`,
		tokenType.ID, tokenType.Name, tokenType.Symbol, tokenType.Color, tokenType.SupplyPolicy,
		tokenType.CapAmount, tokenType.MintedAmount, formatTime(tokenType.CreatedAt))
	if err != nil {
		return TokenType{}, fmt.Errorf("insert token type: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO minter_authorities (token_type_id, subject) VALUES (?, ?)`, tokenType.ID, tokenType.Authority.Subject)
	if err != nil {
		return TokenType{}, fmt.Errorf("insert minter authority: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return TokenType{}, fmt.Errorf("commit token type creation: %w", err)
	}
	return tokenType, nil
}

type rowScanner interface{ Scan(...any) error }

var readDeclarationSQL = `
	SELECT t.id, t.name, t.symbol, t.color, t.supply_policy, t.cap_amount,
	       t.minted_amount, t.retired, t.created_at, t.retired_at, a.subject
	FROM token_types t
	JOIN minter_authorities a ON a.token_type_id = t.id`

func scanTokenType(row rowScanner) (TokenType, error) {
	var tokenType TokenType
	var createdAt string
	var retiredAt sql.NullString
	err := row.Scan(
		&tokenType.ID, &tokenType.Name, &tokenType.Symbol, &tokenType.Color,
		&tokenType.SupplyPolicy, &tokenType.CapAmount, &tokenType.MintedAmount,
		&tokenType.Retired, &createdAt, &retiredAt, &tokenType.Authority.Subject,
	)
	if err != nil {
		return TokenType{}, err
	}
	tokenType.Authority.TokenTypeID = tokenType.ID
	tokenType.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return TokenType{}, fmt.Errorf("parse token type creation time: %w", err)
	}
	if retiredAt.Valid {
		parsed, parseErr := time.Parse(time.RFC3339Nano, retiredAt.String)
		if parseErr != nil {
			return TokenType{}, fmt.Errorf("parse token type retirement time: %w", parseErr)
		}
		tokenType.RetiredAt = &parsed
	}
	return tokenType, nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (TokenType, error) {
	tokenType, err := scanTokenType(r.db.QueryRowContext(ctx, readDeclarationSQL+` WHERE t.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return TokenType{}, ErrTokenTypeNotFound
	}
	if err != nil {
		return TokenType{}, fmt.Errorf("get token type: %w", err)
	}
	return tokenType, nil
}

func (r *sqliteRepository) List(ctx context.Context, includeRetired bool) ([]TokenType, error) {
	query := readDeclarationSQL
	if !includeRetired {
		query += ` WHERE t.retired = 0`
	}
	query += ` ORDER BY t.created_at ASC, t.id ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list token types: %w", err)
	}
	defer rows.Close()
	out := make([]TokenType, 0)
	for rows.Next() {
		tokenType, scanErr := scanTokenType(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan token type: %w", scanErr)
		}
		out = append(out, tokenType)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) Retire(ctx context.Context, id string, at time.Time) (TokenType, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE token_types SET retired = 1, retired_at = ? WHERE id = ? AND retired = 0`, formatTime(at), id)
	if err != nil {
		return TokenType{}, fmt.Errorf("retire token type: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return TokenType{}, fmt.Errorf("read token type retirement result: %w", err)
	}
	if affected == 0 {
		if _, getErr := r.Get(ctx, id); errors.Is(getErr, ErrTokenTypeNotFound) {
			return TokenType{}, ErrTokenTypeNotFound
		}
	}
	return r.Get(ctx, id)
}

func (r *sqliteRepository) Mint(ctx context.Context, id string, amount int64) (TokenType, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return TokenType{}, fmt.Errorf("begin supply mint: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	tokenType, err := scanTokenType(tx.QueryRowContext(ctx, readDeclarationSQL+` WHERE t.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return TokenType{}, ErrTokenTypeNotFound
	}
	if err != nil {
		return TokenType{}, fmt.Errorf("get token type for mint: %w", err)
	}
	if tokenType.Retired {
		return TokenType{}, ErrTokenTypeRetired
	}
	if amount > math.MaxInt64-tokenType.MintedAmount {
		return TokenType{}, &InvalidTokenTypeError{Reason: "mint amount overflows supply"}
	}
	attempted := tokenType.MintedAmount + amount
	if tokenType.SupplyPolicy != SupplyPolicyUnbounded && attempted > tokenType.CapAmount {
		return TokenType{}, &SupplyCapExceededError{
			TokenTypeID: id, Cap: tokenType.CapAmount, AttemptedAmount: amount, CurrentAmount: tokenType.MintedAmount,
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE token_types SET minted_amount = ? WHERE id = ?`, attempted, id); err != nil {
		return TokenType{}, fmt.Errorf("update minted supply: %w", err)
	}
	tokenType.MintedAmount = attempted
	if err := tx.Commit(); err != nil {
		return TokenType{}, fmt.Errorf("commit supply mint: %w", err)
	}
	return tokenType, nil
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
