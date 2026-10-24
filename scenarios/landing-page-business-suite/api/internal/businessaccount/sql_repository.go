package businessaccount

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

type SQLRepository struct {
	db interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
		ExecContext(context.Context, string, ...any) (sql.Result, error)
	}
}

func NewSQLRepository(db interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
},
) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) ListForUser(ctx context.Context, userID, userEmail string) ([]Account, error) {
	if err := ensureIdentity(userID, userEmail); err != nil {
		return nil, err
	}
	if err := r.ensureDefault(ctx, userID, userEmail); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.display_name, a.billing_email, m.role, a.created_at
		FROM business_accounts a
		JOIN business_account_members m ON m.business_account_id = a.id
		WHERE m.user_id = $1
		ORDER BY a.created_at, a.id`, userID)
	if err != nil {
		return nil, fmt.Errorf("list business accounts: %w", err)
	}
	defer rows.Close()
	var accounts []Account
	for rows.Next() {
		var account Account
		if err := rows.Scan(&account.ID, &account.DisplayName, &account.BillingEmail, &account.Role, &account.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan business account: %w", err)
		}
		accounts = append(accounts, account)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate business accounts: %w", err)
	}
	return accounts, nil
}

func (r *SQLRepository) ResolveForUser(ctx context.Context, userID, userEmail, requestedID string) (Account, error) {
	if err := ensureIdentity(userID, userEmail); err != nil {
		return Account{}, err
	}
	if err := r.ensureDefault(ctx, userID, userEmail); err != nil {
		return Account{}, err
	}
	requestedID = strings.TrimSpace(requestedID)
	query := `
		SELECT a.id, a.display_name, a.billing_email, m.role, a.created_at
		FROM business_accounts a
		JOIN business_account_members m ON m.business_account_id = a.id
		WHERE m.user_id = $1`
	args := []any{userID}
	if requestedID != "" {
		query += ` AND a.id = $2`
		args = append(args, requestedID)
	}
	query += ` ORDER BY a.created_at, a.id LIMIT 1`
	var account Account
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&account.ID, &account.DisplayName, &account.BillingEmail, &account.Role, &account.CreatedAt); err != nil {
		if requestedID != "" {
			return Account{}, ErrNotMember
		}
		return Account{}, ErrNotFound
	}
	return account, nil
}

func (r *SQLRepository) CreateForUser(ctx context.Context, userID, userEmail, displayName string) (Account, error) {
	if err := ensureIdentity(userID, userEmail); err != nil {
		return Account{}, err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || len(displayName) > 200 {
		return Account{}, ErrInvalid
	}
	id, err := randomID()
	if err != nil {
		return Account{}, err
	}
	now := time.Now().UTC()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Account{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `INSERT INTO business_accounts (id, display_name, billing_email, created_at) VALUES ($1,$2,$3,$4)`, id, displayName, strings.ToLower(strings.TrimSpace(userEmail)), now); err != nil {
		return Account{}, fmt.Errorf("create business account: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO business_account_members (business_account_id, user_id, role, created_at) VALUES ($1,$2,'owner',$3)`, id, userID, now); err != nil {
		return Account{}, fmt.Errorf("create business account membership: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Account{}, err
	}
	return Account{ID: id, DisplayName: displayName, BillingEmail: strings.ToLower(strings.TrimSpace(userEmail)), Role: "owner", CreatedAt: now}, nil
}

func (r *SQLRepository) ensureDefault(ctx context.Context, userID, userEmail string) error {
	id := defaultID(userID)
	_, err := r.db.ExecContext(ctx, `INSERT INTO business_accounts (id, display_name, billing_email) VALUES ($1,$2,$3) ON CONFLICT (id) DO NOTHING`, id, "Personal account", strings.ToLower(strings.TrimSpace(userEmail)))
	if err != nil {
		return fmt.Errorf("ensure default business account: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO business_account_members (business_account_id, user_id, role) VALUES ($1,$2,'owner') ON CONFLICT (business_account_id, user_id) DO NOTHING`, id, userID)
	if err != nil {
		return fmt.Errorf("ensure default business account membership: %w", err)
	}
	return nil
}

func ensureIdentity(userID, userEmail string) error {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(userEmail) == "" {
		return ErrInvalid
	}
	return nil
}

func defaultID(userID string) string { return "personal:" + strings.TrimSpace(userID) }

func randomID() (string, error) {
	var raw [18]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "acct:" + base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

var _ Repository = (*SQLRepository)(nil)
