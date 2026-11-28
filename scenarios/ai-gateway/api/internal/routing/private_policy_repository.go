package routing

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type SQLPrivatePlanRepository struct{ db *sql.DB }

func NewSQLPrivatePlanRepository(db *sql.DB) *SQLPrivatePlanRepository {
	return &SQLPrivatePlanRepository{db: db}
}

func (r *SQLPrivatePlanRepository) PersistPlan(ctx context.Context, operationID, fingerprint string, plan PrivatePlan) (PrivatePlan, error) {
	if r == nil || r.db == nil {
		return PrivatePlan{}, errors.New("provider route plan repository is not configured")
	}
	encoded, err := json.Marshal(plan)
	if err != nil {
		return PrivatePlan{}, err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO private_provider_plans (operation_id, request_sha256, plan_json, created_at) VALUES (?, ?, ?, ?) ON CONFLICT(operation_id) DO NOTHING`, operationID, fingerprint, string(encoded), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return PrivatePlan{}, err
	}
	var existingFingerprint, existingJSON string
	if err := r.db.QueryRowContext(ctx, `SELECT request_sha256, plan_json FROM private_provider_plans WHERE operation_id = ?`, operationID).Scan(&existingFingerprint, &existingJSON); err != nil {
		return PrivatePlan{}, err
	}
	if existingFingerprint != fingerprint {
		return PrivatePlan{}, ErrPrivatePlanConflict
	}
	var existing PrivatePlan
	if err := json.Unmarshal([]byte(existingJSON), &existing); err != nil {
		return PrivatePlan{}, fmt.Errorf("decode provider route plan: %w", err)
	}
	return existing, nil
}

func (r *SQLPrivatePlanRepository) GetPlan(ctx context.Context, operationID string) (PrivatePlan, error) {
	if r == nil || r.db == nil {
		return PrivatePlan{}, errors.New("provider route plan repository is not configured")
	}
	var encoded string
	if err := r.db.QueryRowContext(ctx, `SELECT plan_json FROM private_provider_plans WHERE operation_id = ?`, operationID).Scan(&encoded); err != nil {
		return PrivatePlan{}, err
	}
	var plan PrivatePlan
	if err := json.Unmarshal([]byte(encoded), &plan); err != nil {
		return PrivatePlan{}, fmt.Errorf("decode provider route plan: %w", err)
	}
	return plan, nil
}

type SQLPrivateReceiptRepository struct{ db *sql.DB }

func NewSQLPrivateReceiptRepository(db *sql.DB) *SQLPrivateReceiptRepository {
	return &SQLPrivateReceiptRepository{db: db}
}

func (r *SQLPrivateReceiptRepository) Persist(ctx context.Context, operationID, fingerprint string, receipt PrivateReceipt) (PrivateReceipt, error) {
	if r == nil || r.db == nil {
		return PrivateReceipt{}, errors.New("provider execution receipt repository is not configured")
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		return PrivateReceipt{}, err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO private_provider_receipts (operation_id, request_sha256, receipt_json, created_at) VALUES (?, ?, ?, ?) ON CONFLICT(operation_id) DO NOTHING`, operationID, fingerprint, string(encoded), receipt.CreatedAt)
	if err != nil {
		return PrivateReceipt{}, err
	}
	var existingFingerprint, existingJSON string
	if err := r.db.QueryRowContext(ctx, `SELECT request_sha256, receipt_json FROM private_provider_receipts WHERE operation_id = ?`, operationID).Scan(&existingFingerprint, &existingJSON); err != nil {
		return PrivateReceipt{}, err
	}
	if existingFingerprint != fingerprint {
		return PrivateReceipt{}, ErrPrivateReceiptConflict
	}
	var existing PrivateReceipt
	if err := json.Unmarshal([]byte(existingJSON), &existing); err != nil {
		return PrivateReceipt{}, fmt.Errorf("decode provider execution receipt: %w", err)
	}
	return existing, nil
}

func (r *SQLPrivateReceiptRepository) Get(ctx context.Context, operationID string) (PrivateReceipt, error) {
	if r == nil || r.db == nil {
		return PrivateReceipt{}, errors.New("provider execution receipt repository is not configured")
	}
	var encoded string
	if err := r.db.QueryRowContext(ctx, `SELECT receipt_json FROM private_provider_receipts WHERE operation_id = ?`, operationID).Scan(&encoded); err != nil {
		return PrivateReceipt{}, err
	}
	var receipt PrivateReceipt
	if err := json.Unmarshal([]byte(encoded), &receipt); err != nil {
		return PrivateReceipt{}, fmt.Errorf("decode provider execution receipt: %w", err)
	}
	return receipt, nil
}
