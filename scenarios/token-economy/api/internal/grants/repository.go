// Package grants owns mandate-shaped token grants and rule evaluation.
package grants

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type Repository interface {
	Create(context.Context, Grant, Credit) (Grant, error)
	Get(context.Context, string) (Grant, error)
	List(context.Context, string, string, bool) ([]Grant, error)
	Revoke(context.Context, string, string, string, time.Time) (Grant, error)
}

type Credit struct {
	ID             string
	TokenTypeID    string
	HolderID       string
	Amount         int64
	CauseReference string
	ActorIdentity  string
	CreatedAt      time.Time
}

type AppendCreditFunc func(context.Context, *sql.Tx, Credit) error

type sqliteRepository struct {
	db           SQLExecutor
	appendCredit AppendCreditFunc
}

func NewSQLiteRepository(db SQLExecutor, appendCredit AppendCreditFunc) Repository {
	return &sqliteRepository{db: db, appendCredit: appendCredit}
}

func (r *sqliteRepository) Create(ctx context.Context, grant Grant, credit Credit) (Grant, error) {
	if err := validateCredit(grant, credit); err != nil {
		return Grant{}, err
	}
	if r.appendCredit == nil {
		return Grant{}, errors.New("grant repository requires a credit appender")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Grant{}, fmt.Errorf("begin grant creation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	existing, err := getGrantByIdempotency(ctx, tx, grant.IdempotencyKey)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrGrantNotFound) {
		return Grant{}, err
	}
	allowed, err := encodeStrings(grant.AllowedCatalogScopes)
	if err != nil {
		return Grant{}, err
	}
	denied, err := encodeStrings(grant.DeniedCatalogScopes)
	if err != nil {
		return Grant{}, err
	}
	evidence, err := encodeStrings(grant.RequiredEvidence)
	if err != nil {
		return Grant{}, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO grants (
			id, token_type_id, grant_source_id, authorizer, holder_id, amount_minor,
			allowed_catalog_scopes, denied_catalog_scopes, expires_at, issued_at, status,
			idempotency_key, required_evidence, recurrence_seconds, next_issue_at, cancelled_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		grant.ID, grant.TokenTypeID, grant.GrantSourceID, grant.Authorizer, grant.HolderID,
		grant.AmountMinor, allowed, denied, formatTime(grant.ExpiresAt), formatTime(grant.IssuedAt),
		grant.Status, grant.IdempotencyKey, evidence, grant.RecurrenceSeconds,
		nullableTime(grant.NextIssueAt), nullableTime(grant.CancelledAt))
	if err != nil {
		return Grant{}, fmt.Errorf("insert grant: %w", err)
	}
	for index, rule := range grant.Rules {
		operands, encodeErr := encodeStrings(rule.Operands)
		if encodeErr != nil {
			return Grant{}, encodeErr
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO grant_rules (id, grant_id, position, condition, operands, amount_limit)
			VALUES (?, ?, ?, ?, ?, ?)`, storedRuleID(grant.ID, rule.ID), grant.ID, index, rule.Condition, operands, rule.AmountLimit)
		if err != nil {
			return Grant{}, fmt.Errorf("insert grant rule %q: %w", rule.ID, err)
		}
	}
	if grant.RecurrenceSeconds > 0 {
		if grant.NextIssueAt == nil {
			return Grant{}, &InvalidGrantError{Reason: "recurring grant requires next_issue_at"}
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO grant_schedules (grant_id, recurrence_seconds, next_issue_at, cancelled_at)
			VALUES (?, ?, ?, ?)`, grant.ID, grant.RecurrenceSeconds, formatTime(*grant.NextIssueAt), nullableTime(grant.CancelledAt))
		if err != nil {
			return Grant{}, fmt.Errorf("insert grant schedule: %w", err)
		}
	}
	if err := r.appendCredit(ctx, tx, credit); err != nil {
		return Grant{}, fmt.Errorf("append grant credit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Grant{}, fmt.Errorf("commit grant creation: %w", err)
	}
	return grant, nil
}

func validateCredit(grant Grant, credit Credit) error {
	if credit.TokenTypeID != grant.TokenTypeID || credit.HolderID != grant.HolderID ||
		credit.Amount != grant.AmountMinor || credit.CauseReference != "grant:"+grant.ID {
		return &InvalidGrantError{Reason: "journal credit must exactly represent the grant"}
	}
	return nil
}

var readGrantSQL = `
	SELECT id, token_type_id, grant_source_id, authorizer, holder_id, amount_minor,
	       allowed_catalog_scopes, denied_catalog_scopes, expires_at, issued_at, status,
	       idempotency_key, required_evidence, recurrence_seconds, next_issue_at, cancelled_at
	FROM grants`

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type rowScanner interface{ Scan(...any) error }

func scanGrant(row rowScanner) (Grant, error) {
	var grant Grant
	var allowed, denied, evidence, expiresAt, issuedAt string
	var nextIssueAt, cancelledAt sql.NullString
	err := row.Scan(
		&grant.ID, &grant.TokenTypeID, &grant.GrantSourceID, &grant.Authorizer, &grant.HolderID,
		&grant.AmountMinor, &allowed, &denied, &expiresAt, &issuedAt, &grant.Status,
		&grant.IdempotencyKey, &evidence, &grant.RecurrenceSeconds, &nextIssueAt, &cancelledAt,
	)
	if err != nil {
		return Grant{}, err
	}
	if err := json.Unmarshal([]byte(allowed), &grant.AllowedCatalogScopes); err != nil {
		return Grant{}, fmt.Errorf("decode allowed catalog scopes: %w", err)
	}
	if err := json.Unmarshal([]byte(denied), &grant.DeniedCatalogScopes); err != nil {
		return Grant{}, fmt.Errorf("decode denied catalog scopes: %w", err)
	}
	if err := json.Unmarshal([]byte(evidence), &grant.RequiredEvidence); err != nil {
		return Grant{}, fmt.Errorf("decode required evidence: %w", err)
	}
	grant.ExpiresAt, err = time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil {
		return Grant{}, fmt.Errorf("parse grant expiry: %w", err)
	}
	grant.IssuedAt, err = time.Parse(time.RFC3339Nano, issuedAt)
	if err != nil {
		return Grant{}, fmt.Errorf("parse grant issue time: %w", err)
	}
	grant.NextIssueAt, err = parseNullableTime(nextIssueAt)
	if err != nil {
		return Grant{}, fmt.Errorf("parse next issue time: %w", err)
	}
	grant.CancelledAt, err = parseNullableTime(cancelledAt)
	if err != nil {
		return Grant{}, fmt.Errorf("parse cancellation time: %w", err)
	}
	return grant, nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Grant, error) {
	return getGrant(ctx, r.db, readGrantSQL+` WHERE id = ?`, id)
}

func (r *sqliteRepository) List(ctx context.Context, holderID, tokenTypeID string, includeInactive bool) ([]Grant, error) {
	query := readGrantSQL + ` WHERE (? = '' OR holder_id = ?) AND (? = '' OR token_type_id = ?)`
	args := []any{strings.TrimSpace(holderID), strings.TrimSpace(holderID), strings.TrimSpace(tokenTypeID), strings.TrimSpace(tokenTypeID)}
	if !includeInactive {
		query += ` AND status = 'live'`
	}
	query += ` ORDER BY issued_at, id`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list grants: %w", err)
	}
	defer rows.Close()
	values := make([]Grant, 0)
	for rows.Next() {
		grant, err := scanGrant(rows)
		if err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan grant: %w", err)
		}
		values = append(values, grant)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate grants: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close grant rows: %w", err)
	}
	for index := range values {
		values[index].Rules, err = readRules(ctx, r.db, values[index].ID)
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}

func (r *sqliteRepository) Revoke(ctx context.Context, id, reason, idempotencyKey string, at time.Time) (Grant, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Grant{}, fmt.Errorf("begin grant revocation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	existing, err := getGrant(ctx, tx, readGrantSQL+`
		WHERE id = (SELECT grant_id FROM grant_revoke_receipts WHERE idempotency_key = ?)`, idempotencyKey)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrGrantNotFound) {
		return Grant{}, err
	}
	grant, err := getGrant(ctx, tx, readGrantSQL+` WHERE id = ?`, id)
	if err != nil {
		return Grant{}, err
	}
	if grant.Status == GrantStatusRevoked {
		return Grant{}, &InvalidGrantError{Reason: "grant is already revoked"}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE grants SET status = 'revoked', cancelled_at = ? WHERE id = ?`, formatTime(at), id); err != nil {
		return Grant{}, fmt.Errorf("revoke grant: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO grant_revoke_receipts (idempotency_key, grant_id, reason, created_at)
		VALUES (?, ?, ?, ?)`, idempotencyKey, id, reason, formatTime(at)); err != nil {
		return Grant{}, fmt.Errorf("store grant revocation receipt: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Grant{}, fmt.Errorf("commit grant revocation: %w", err)
	}
	grant.Status = GrantStatusRevoked
	grant.CancelledAt = &at
	return grant, nil
}

func getGrantByIdempotency(ctx context.Context, db queryer, key string) (Grant, error) {
	return getGrant(ctx, db, readGrantSQL+` WHERE idempotency_key = ?`, key)
}

func getGrant(ctx context.Context, db queryer, query, value string) (Grant, error) {
	grant, err := scanGrant(db.QueryRowContext(ctx, query, value))
	if errors.Is(err, sql.ErrNoRows) {
		return Grant{}, ErrGrantNotFound
	}
	if err != nil {
		return Grant{}, fmt.Errorf("read grant: %w", err)
	}
	rules, err := readRules(ctx, db, grant.ID)
	if err != nil {
		return Grant{}, err
	}
	grant.Rules = rules
	return grant, nil
}

func readRules(ctx context.Context, db queryer, grantID string) ([]GrantRule, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, condition, operands, amount_limit
		FROM grant_rules WHERE grant_id = ? ORDER BY position ASC`, grantID)
	if err != nil {
		return nil, fmt.Errorf("read grant rules: %w", err)
	}
	defer rows.Close()
	rules := make([]GrantRule, 0)
	for rows.Next() {
		var rule GrantRule
		var operands string
		if err := rows.Scan(&rule.ID, &rule.Condition, &operands, &rule.AmountLimit); err != nil {
			return nil, fmt.Errorf("scan grant rule: %w", err)
		}
		rule.ID = strings.TrimPrefix(rule.ID, grantID+"/")
		if err := json.Unmarshal([]byte(operands), &rule.Operands); err != nil {
			return nil, fmt.Errorf("decode grant rule operands: %w", err)
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate grant rules: %w", err)
	}
	return rules, nil
}

func storedRuleID(grantID, ruleID string) string { return grantID + "/" + ruleID }

func encodeStrings(values []string) (string, error) {
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("encode string list: %w", err)
	}
	return string(encoded), nil
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return formatTime(*value)
}

func parseNullableTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value.String)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
