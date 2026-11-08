package inference

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

type AttemptState string

const (
	AttemptDispatching       AttemptState = "dispatching"
	AttemptPreDispatchFailed AttemptState = "pre_dispatch_failed"
	AttemptUncertain         AttemptState = "uncertain"
	AttemptPending           AttemptState = "pending"
	AttemptSettled           AttemptState = "settled"
	AttemptReleased          AttemptState = "released"
	AttemptOutputObserved    AttemptState = "output_observed"
)

var ErrRootConflict = errors.New("root operation id is bound to different request content or device key generation")
var ErrRootRequired = errors.New("root_operation_id must be a high-entropy value of 32 to 128 characters before HostedDoor dispatch")

type AttemptRecord struct {
	Ordinal           int
	ValidationAttempt int
	CandidateIndex    int
	Key               string
	Provider          string
	ResourceRole      string
	State             AttemptState
	AccountingStatus  string
	ChargedCredits    *int64
	Created           bool
}

type LedgerSnapshot struct {
	Generation string
	Attempt    *AttemptRecord
}

type AttemptLedger interface {
	Lookup(context.Context, string, string) (*LedgerSnapshot, error)
	Begin(context.Context, string, string, string, int, int, string, string) (AttemptRecord, error)
	Set(context.Context, string, int, AttemptState, string, *int64) error
}

type SQLAttemptLedger struct{ db *sql.DB }

func NewSQLAttemptLedger(db *sql.DB) *SQLAttemptLedger { return &SQLAttemptLedger{db: db} }

func (l *SQLAttemptLedger) Lookup(ctx context.Context, root, digest string) (*LedgerSnapshot, error) {
	if l == nil || l.db == nil {
		return nil, errors.New("inference attempt ledger is unavailable")
	}
	var savedDigest, generation string
	err := l.db.QueryRowContext(ctx, `SELECT request_sha256,device_key_generation FROM ai_gateway_inference_roots WHERE root_operation_id=$1`, root).Scan(&savedDigest, &generation)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if savedDigest != digest {
		return nil, ErrRootConflict
	}
	snapshot := &LedgerSnapshot{Generation: generation}
	var record AttemptRecord
	var state string
	var credits sql.NullInt64
	err = l.db.QueryRowContext(ctx, `SELECT attempt_ordinal,validation_attempt,candidate_index,attempt_key,provider_id,resource_role,state,accounting_status,charged_credits
FROM ai_gateway_inference_attempts WHERE root_operation_id=$1 ORDER BY attempt_ordinal DESC LIMIT 1`, root).Scan(
		&record.Ordinal, &record.ValidationAttempt, &record.CandidateIndex, &record.Key, &record.Provider, &record.ResourceRole, &state, &record.AccountingStatus, &credits)
	if errors.Is(err, sql.ErrNoRows) {
		return snapshot, nil
	}
	if err != nil {
		return nil, err
	}
	record.State = AttemptState(state)
	if credits.Valid {
		record.ChargedCredits = &credits.Int64
	}
	snapshot.Attempt = &record
	return snapshot, nil
}

func (l *SQLAttemptLedger) Begin(ctx context.Context, root, digest, generation string, validationAttempt, candidateIndex int, provider, resourceRole string) (AttemptRecord, error) {
	if l == nil || l.db == nil {
		return AttemptRecord{}, errors.New("inference attempt ledger is unavailable")
	}
	childKey := make([]byte, 32)
	if _, err := rand.Read(childKey); err != nil {
		return AttemptRecord{}, err
	}
	key := hex.EncodeToString(childKey)
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return AttemptRecord{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO ai_gateway_inference_roots(root_operation_id,request_sha256,device_key_generation)
VALUES($1,$2,$3) ON CONFLICT(root_operation_id) DO NOTHING`, root, digest, generation); err != nil {
		return AttemptRecord{}, err
	}
	var savedDigest, savedGeneration string
	if err = tx.QueryRowContext(ctx, `SELECT request_sha256,device_key_generation FROM ai_gateway_inference_roots WHERE root_operation_id=$1`, root).Scan(&savedDigest, &savedGeneration); err != nil {
		return AttemptRecord{}, err
	}
	if savedDigest != digest || savedGeneration != generation {
		return AttemptRecord{}, ErrRootConflict
	}
	var record AttemptRecord
	var state string
	var credits sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT attempt_ordinal,validation_attempt,candidate_index,attempt_key,provider_id,resource_role,state,accounting_status,charged_credits
FROM ai_gateway_inference_attempts WHERE root_operation_id=$1 AND validation_attempt=$2 AND candidate_index=$3`, root, validationAttempt, candidateIndex).Scan(
		&record.Ordinal, &record.ValidationAttempt, &record.CandidateIndex, &record.Key, &record.Provider, &record.ResourceRole, &state, &record.AccountingStatus, &credits)
	if errors.Is(err, sql.ErrNoRows) {
		if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(attempt_ordinal),-1)+1 FROM ai_gateway_inference_attempts WHERE root_operation_id=$1`, root).Scan(&record.Ordinal); err != nil {
			return AttemptRecord{}, err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO ai_gateway_inference_attempts(root_operation_id,attempt_ordinal,validation_attempt,candidate_index,attempt_key,provider_id,resource_role,state)
VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, root, record.Ordinal, validationAttempt, candidateIndex, key, provider, resourceRole, string(AttemptDispatching))
		if err != nil {
			return AttemptRecord{}, err
		}
		record.ValidationAttempt, record.CandidateIndex, record.Key = validationAttempt, candidateIndex, key
		record.Provider, record.ResourceRole, record.State, record.Created = provider, resourceRole, AttemptDispatching, true
	} else if err != nil {
		return AttemptRecord{}, err
	} else {
		record.State = AttemptState(state)
		if credits.Valid {
			record.ChargedCredits = &credits.Int64
		}
	}
	if err = tx.Commit(); err != nil {
		return AttemptRecord{}, err
	}
	return record, nil
}

func (l *SQLAttemptLedger) Set(ctx context.Context, root string, ordinal int, state AttemptState, status string, credits *int64) error {
	if l == nil || l.db == nil {
		return errors.New("inference attempt ledger is unavailable")
	}
	result, err := l.db.ExecContext(ctx, `UPDATE ai_gateway_inference_attempts
SET state=$1,accounting_status=$2,charged_credits=$3,updated_at=CURRENT_TIMESTAMP
WHERE root_operation_id=$4 AND attempt_ordinal=$5 AND state NOT IN ($6,$7,$8)`,
		string(state), status, credits, root, ordinal, string(AttemptSettled), string(AttemptReleased), string(AttemptOutputObserved))
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 1 {
		return nil
	}
	var exists int
	err = l.db.QueryRowContext(ctx, `SELECT 1 FROM ai_gateway_inference_attempts WHERE root_operation_id=$1 AND attempt_ordinal=$2`, root, ordinal).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("inference attempt %s/%d is missing", root, ordinal)
	}
	return err
}

func validRootOperationID(root string) bool {
	return len(root) >= 32 && len(root) <= 128 && strings.TrimSpace(root) == root && !strings.ContainsAny(root, " \t\r\n")
}

func attemptReference(key string) string {
	digest := sha256.Sum256([]byte(key))
	return hex.EncodeToString(digest[:8])
}
