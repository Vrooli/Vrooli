package baseusdc

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

//go:embed schema.sql
var schemaSQL string

func Schema() string { return schemaSQL }

type Check struct {
	ID, OperatorIdentity, WalletID, Network, TransactionHash string
	ProviderID, Outcome, ErrorCode                           string
	PriorCheckID, Relation                                   string
	CheckedAt                                                time.Time
	Transfers                                                []Observation
}

type Store interface {
	Append(context.Context, Check) error
	List(context.Context, string, string, string, int) ([]Check, error)
	GetPriorVerified(context.Context, string, string, string, string, string) (Check, error)
}

type DB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLiteStore struct{ db DB }

func NewSQLiteStore(db DB) *SQLiteStore { return &SQLiteStore{db: db} }

func (s *SQLiteStore) Append(ctx context.Context, value Check) error {
	if s == nil || s.db == nil || value.ID == "" || value.OperatorIdentity == "" || value.WalletID == "" || value.Network != Network || value.TransactionHash == "" || value.ProviderID == "" || value.Outcome == "" || value.CheckedAt.IsZero() {
		return errors.New("Base receipt check identity and outcome are required")
	}
	transfers, err := json.Marshal(value.Transfers)
	if err != nil {
		return fmt.Errorf("encode Base receipt observations: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO baseusdc_receipt_checks(id,operator_identity,wallet_id,network,transaction_hash,provider_id,outcome,error_code,checked_at,transfers_json,prior_check_id,relation) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		value.ID, value.OperatorIdentity, value.WalletID, value.Network, value.TransactionHash, value.ProviderID, value.Outcome, value.ErrorCode, value.CheckedAt.UTC().Format(time.RFC3339Nano), string(transfers), value.PriorCheckID, value.Relation)
	return err
}

func (s *SQLiteStore) List(ctx context.Context, operator, walletID, transactionHash string, limit int) ([]Check, error) {
	if s == nil || s.db == nil || operator == "" || walletID == "" || transactionHash == "" {
		return nil, errors.New("operator, wallet, and transaction identity are required")
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,operator_identity,wallet_id,network,transaction_hash,provider_id,outcome,error_code,checked_at,transfers_json,prior_check_id,relation FROM baseusdc_receipt_checks WHERE operator_identity=? AND wallet_id=? AND transaction_hash=? ORDER BY checked_at DESC,id DESC LIMIT ?`, operator, walletID, transactionHash, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]Check, 0)
	for rows.Next() {
		var value Check
		var checkedAt, transfersJSON string
		if err := rows.Scan(&value.ID, &value.OperatorIdentity, &value.WalletID, &value.Network, &value.TransactionHash, &value.ProviderID, &value.Outcome, &value.ErrorCode, &checkedAt, &transfersJSON, &value.PriorCheckID, &value.Relation); err != nil {
			return nil, err
		}
		if value.CheckedAt, err = time.Parse(time.RFC3339Nano, checkedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(transfersJSON), &value.Transfers); err != nil {
			return nil, fmt.Errorf("decode retained Base receipt observations: %w", err)
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (s *SQLiteStore) GetPriorVerified(ctx context.Context, operator, walletID, network, transactionHash, checkID string) (Check, error) {
	if s == nil || s.db == nil || operator == "" || walletID == "" || network == "" || transactionHash == "" || checkID == "" {
		return Check{}, errors.New("operator, wallet, network, transaction, and prior check identity are required")
	}
	var value Check
	var checkedAt, transfersJSON string
	err := s.db.QueryRowContext(ctx, `SELECT id,operator_identity,wallet_id,network,transaction_hash,provider_id,outcome,error_code,checked_at,transfers_json,prior_check_id,relation FROM baseusdc_receipt_checks WHERE id=? AND operator_identity=? AND wallet_id=? AND network=? AND transaction_hash=? AND outcome='verified'`, checkID, operator, walletID, network, transactionHash).Scan(
		&value.ID, &value.OperatorIdentity, &value.WalletID, &value.Network, &value.TransactionHash, &value.ProviderID, &value.Outcome, &value.ErrorCode, &checkedAt, &transfersJSON, &value.PriorCheckID, &value.Relation,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Check{}, ErrPriorCheckNotFound
	}
	if err != nil {
		return Check{}, err
	}
	if value.CheckedAt, err = time.Parse(time.RFC3339Nano, checkedAt); err != nil {
		return Check{}, err
	}
	if err := json.Unmarshal([]byte(transfersJSON), &value.Transfers); err != nil {
		return Check{}, fmt.Errorf("decode retained Base receipt observations: %w", err)
	}
	return value, nil
}
