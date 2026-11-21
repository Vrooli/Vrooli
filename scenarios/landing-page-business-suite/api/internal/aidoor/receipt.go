package aidoor

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// callerOperationID preserves the dispatch identity across response loss. The
// caller's key is an idempotency identity, never a raw Gateway operation ID.
func callerOperationID(key Key, idempotencyKey string) string {
	digest := sha256.Sum256([]byte(key.ID + ":" + idempotencyKey))
	return fmt.Sprintf("%x-%x-%x-%x-%x", digest[:4], digest[4:6], digest[6:8], digest[8:10], digest[10:16])
}

// publicAccountingReceipt deliberately excludes the private Gateway receipt's
// content, routing, cost and model details. Settled describes accounting only;
// it does not claim that the caller received a successful provider response.
type publicAccountingReceipt struct {
	Status         string `json:"status"`
	ChargedCredits *int64 `json:"charged_credits,omitempty"`
}

func (d *Door) publicReceipt(w http.ResponseWriter, r *http.Request, key Key) {
	values := r.Header.Values("Idempotency-Key")
	if len(values) != 1 || values[0] == "" {
		fail(w, http.StatusBadRequest, "original Idempotency-Key required")
		return
	}
	operation := callerOperationID(key, values[0])
	result, err := d.accountingReceipt(r.Context(), key, operation)
	if err != nil {
		fail(w, http.StatusServiceUnavailable, "receipt unavailable")
		return
	}
	if result.Status == "pending" {
		// Only an authenticated, matching durable claim may reach the private
		// recovery owner. Recovery can settle a hold but can never dispatch.
		_ = d.recover(r.Context(), key, operation)
		result, err = d.accountingReceipt(r.Context(), key, operation)
		if err != nil {
			fail(w, http.StatusServiceUnavailable, "receipt unavailable")
			return
		}
	}
	status := http.StatusOK
	switch result.Status {
	case "unknown":
		status = http.StatusNotFound
	case "pending":
		status = http.StatusAccepted
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}

func (d *Door) accountingReceipt(ctx context.Context, key Key, operation string) (publicAccountingReceipt, error) {
	var actual, charged sql.NullInt64
	var status sql.NullString
	// Bind the operation twice so each engine infers the native reservation ID
	// type without casting its indexed column (Postgres UUID, SQLite text).
	err := d.Keys.DB.QueryRowContext(ctx, `SELECT c.actual_credits,r.status,r.charged_amount
 FROM ai_key_charges c
 LEFT JOIN business_account_credit_reservations r ON r.id=$3 AND r.business_account_id=$4 AND r.actor_identity=$5
 WHERE c.operation_id=$1 AND c.key_id=$2`, operation, key.ID, operation, key.Account, key.Actor).Scan(&actual, &status, &charged)
	if errors.Is(err, sql.ErrNoRows) {
		return publicAccountingReceipt{Status: "unknown"}, nil
	}
	if err != nil {
		return publicAccountingReceipt{}, err
	}
	if !status.Valid && !(actual.Valid && actual.Int64 == 0) {
		// A cap claim alone cannot authorize recovery of a hold belonging to a
		// different account/actor, even if persisted data is inconsistent.
		return publicAccountingReceipt{Status: "unknown"}, nil
	}
	if (status.String == "finalized" || status.String == "overrun") && charged.Valid {
		return publicAccountingReceipt{Status: "settled", ChargedCredits: &charged.Int64}, nil
	}
	if status.String == "released" || (!status.Valid && actual.Valid && actual.Int64 == 0) {
		zero := int64(0)
		return publicAccountingReceipt{Status: "released", ChargedCredits: &zero}, nil
	}
	return publicAccountingReceipt{Status: "pending"}, nil
}
