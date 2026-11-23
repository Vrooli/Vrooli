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
	"time"

	monetization "github.com/vrooli/vrooli/packages/monetization-go"
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

var attemptStateEdges = map[AttemptState]map[AttemptState]struct{}{
	AttemptDispatching: {
		AttemptPreDispatchFailed: {}, AttemptUncertain: {}, AttemptPending: {},
		AttemptSettled: {}, AttemptReleased: {}, AttemptOutputObserved: {},
	},
	AttemptUncertain: {
		AttemptUncertain: {}, AttemptPending: {}, AttemptSettled: {},
		AttemptReleased: {}, AttemptOutputObserved: {},
	},
	AttemptPending: {
		AttemptUncertain: {}, AttemptPending: {}, AttemptSettled: {},
		AttemptReleased: {}, AttemptOutputObserved: {},
	},
}

// Valid reports whether state is part of the persisted hosted-attempt lifecycle.
func (state AttemptState) Valid() bool {
	switch state {
	case AttemptDispatching, AttemptPreDispatchFailed, AttemptUncertain, AttemptPending,
		AttemptSettled, AttemptReleased, AttemptOutputObserved:
		return true
	default:
		return false
	}
}

func (state AttemptState) Terminal() bool {
	return state == AttemptPreDispatchFailed || state == AttemptSettled || state == AttemptReleased || state == AttemptOutputObserved
}

// CanTransition is the single policy for durable provider-attempt state changes.
func (state AttemptState) CanTransition(next AttemptState) bool {
	if !state.Valid() || !next.Valid() {
		return false
	}
	_, ok := attemptStateEdges[state][next]
	return ok
}

var (
	ErrRootConflict = errors.New("root operation id is bound to different request content or device key generation")
	ErrRootRequired = errors.New("root_operation_id must be a high-entropy value of 32 to 128 characters before HostedDoor dispatch")
)

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

// HostedAttemptRetention exceeds the LPBS credit-door key/receipt validity by
// one day. That grace period accommodates clock skew and delayed lifecycle
// cleanup while keeping a receipt recoverable for every instant its key works.
const HostedAttemptRetention = monetization.CreditDoorReceiptValidity + 24*time.Hour

func NewSQLAttemptLedger(db *sql.DB) *SQLAttemptLedger { return &SQLAttemptLedger{db: db} }

// PruneExpired removes only roots whose complete attempt history is terminal
// and older than the LPBS receipt-validity window. Pending and uncertain work
// is retained indefinitely because deleting it could permit a duplicate
// provider dispatch. The caller supplies now to make lifecycle cleanup
// deterministic in tests.
func (l *SQLAttemptLedger) PruneExpired(ctx context.Context, now time.Time) (int64, error) {
	if l == nil || l.db == nil {
		return 0, errors.New("inference attempt ledger is unavailable")
	}
	cutoff := now.UTC().Add(-HostedAttemptRetention).Format("2006-01-02 15:04:05")
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(ctx, `SELECT root_operation_id FROM ai_gateway_inference_roots r
WHERE EXISTS (SELECT 1 FROM ai_gateway_inference_attempts a WHERE a.root_operation_id=r.root_operation_id)
AND NOT EXISTS (SELECT 1 FROM ai_gateway_inference_attempts a WHERE a.root_operation_id=r.root_operation_id
AND (a.state NOT IN ($1,$2,$3,$4) OR a.updated_at >= $5))`,
		string(AttemptPreDispatchFailed), string(AttemptSettled), string(AttemptReleased), string(AttemptOutputObserved), cutoff)
	if err != nil {
		return 0, err
	}
	var roots []string
	for rows.Next() {
		var root string
		if err := rows.Scan(&root); err != nil {
			_ = rows.Close()
			return 0, err
		}
		roots = append(roots, root)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return 0, err
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	for _, root := range roots {
		if _, err := tx.ExecContext(ctx, `DELETE FROM ai_gateway_inference_attempts WHERE root_operation_id=$1`, root); err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM ai_gateway_inference_roots WHERE root_operation_id=$1`, root); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return int64(len(roots)), nil
}

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
	if !record.State.Valid() {
		return nil, fmt.Errorf("inference attempt %s has unknown state %q", root, record.State)
	}
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
	defer func() { _ = tx.Rollback() }()
	if err = ensureAttemptRoot(ctx, tx, root, digest, generation); err != nil {
		return AttemptRecord{}, err
	}
	record, err := beginAttemptRecord(ctx, tx, root, validationAttempt, candidateIndex, provider, resourceRole, key)
	if err != nil {
		return AttemptRecord{}, err
	}
	if err = tx.Commit(); err != nil {
		return AttemptRecord{}, err
	}
	return record, nil
}

func ensureAttemptRoot(ctx context.Context, tx *sql.Tx, root, digest, generation string) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO ai_gateway_inference_roots(root_operation_id,request_sha256,device_key_generation)
VALUES($1,$2,$3) ON CONFLICT(root_operation_id) DO NOTHING`, root, digest, generation); err != nil {
		return err
	}
	var savedDigest, savedGeneration string
	if err := tx.QueryRowContext(ctx, `SELECT request_sha256,device_key_generation FROM ai_gateway_inference_roots WHERE root_operation_id=$1`, root).Scan(&savedDigest, &savedGeneration); err != nil {
		return err
	}
	if savedDigest != digest || savedGeneration != generation {
		return ErrRootConflict
	}
	return nil
}

func beginAttemptRecord(ctx context.Context, tx *sql.Tx, root string, validationAttempt, candidateIndex int, provider, resourceRole, key string) (AttemptRecord, error) {
	var record AttemptRecord
	var state string
	var credits sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT attempt_ordinal,validation_attempt,candidate_index,attempt_key,provider_id,resource_role,state,accounting_status,charged_credits
FROM ai_gateway_inference_attempts WHERE root_operation_id=$1 AND validation_attempt=$2 AND candidate_index=$3`, root, validationAttempt, candidateIndex).Scan(
		&record.Ordinal, &record.ValidationAttempt, &record.CandidateIndex, &record.Key, &record.Provider, &record.ResourceRole, &state, &record.AccountingStatus, &credits)
	if errors.Is(err, sql.ErrNoRows) {
		return createAttemptRecord(ctx, tx, root, validationAttempt, candidateIndex, provider, resourceRole, key)
	}
	if err != nil {
		return AttemptRecord{}, err
	}
	return restoreAttemptRecord(root, record, state, credits)
}

func createAttemptRecord(ctx context.Context, tx *sql.Tx, root string, validationAttempt, candidateIndex int, provider, resourceRole, key string) (AttemptRecord, error) {
	var record AttemptRecord
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(attempt_ordinal),-1)+1 FROM ai_gateway_inference_attempts WHERE root_operation_id=$1`, root).Scan(&record.Ordinal); err != nil {
		return AttemptRecord{}, err
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO ai_gateway_inference_attempts(root_operation_id,attempt_ordinal,validation_attempt,candidate_index,attempt_key,provider_id,resource_role,state)
VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, root, record.Ordinal, validationAttempt, candidateIndex, key, provider, resourceRole, string(AttemptDispatching))
	if err != nil {
		return AttemptRecord{}, err
	}
	record.ValidationAttempt, record.CandidateIndex, record.Key = validationAttempt, candidateIndex, key
	record.Provider, record.ResourceRole, record.State, record.Created = provider, resourceRole, AttemptDispatching, true
	return record, nil
}

func restoreAttemptRecord(root string, record AttemptRecord, state string, credits sql.NullInt64) (AttemptRecord, error) {
	record.State = AttemptState(state)
	if !record.State.Valid() {
		return AttemptRecord{}, fmt.Errorf("inference attempt %s/%d has unknown state %q", root, record.Ordinal, record.State)
	}
	if credits.Valid {
		record.ChargedCredits = &credits.Int64
	}
	return record, nil
}

func (l *SQLAttemptLedger) Set(ctx context.Context, root string, ordinal int, state AttemptState, status string, credits *int64) error {
	if l == nil || l.db == nil {
		return errors.New("inference attempt ledger is unavailable")
	}
	from, err := incomingAttemptStates(state)
	if err != nil {
		return err
	}
	placeholders := make([]string, len(from))
	args := []any{string(state), status, credits, root, ordinal}
	for i, current := range from {
		placeholders[i] = fmt.Sprintf("$%d", len(args)+1)
		args = append(args, string(current))
	}
	result, err := l.db.ExecContext(ctx, `UPDATE ai_gateway_inference_attempts
SET state=$1,accounting_status=$2,charged_credits=$3,updated_at=CURRENT_TIMESTAMP
WHERE root_operation_id=$4 AND attempt_ordinal=$5 AND state IN (`+strings.Join(placeholders, ",")+")", args...)
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
	return l.attemptTransitionError(ctx, root, ordinal, state)
}

func incomingAttemptStates(state AttemptState) ([]AttemptState, error) {
	if !state.Valid() {
		return nil, fmt.Errorf("unknown inference attempt state %q", state)
	}
	from := make([]AttemptState, 0, len(attemptStateEdges))
	for current := range attemptStateEdges {
		if current.CanTransition(state) {
			from = append(from, current)
		}
	}
	if len(from) == 0 {
		return nil, fmt.Errorf("inference attempt state %q has no valid incoming transition", state)
	}
	return from, nil
}

func (l *SQLAttemptLedger) attemptTransitionError(ctx context.Context, root string, ordinal int, state AttemptState) error {
	var exists int
	err := l.db.QueryRowContext(ctx, `SELECT 1 FROM ai_gateway_inference_attempts WHERE root_operation_id=$1 AND attempt_ordinal=$2`, root, ordinal).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("inference attempt %s/%d is missing", root, ordinal)
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("inference attempt %s/%d cannot transition to %q from its current state", root, ordinal, state)
}

func validRootOperationID(root string) bool {
	return len(root) >= 32 && len(root) <= 128 && strings.TrimSpace(root) == root && !strings.ContainsAny(root, " \t\r\n")
}

func attemptReference(key string) string {
	digest := sha256.Sum256([]byte(key))
	return hex.EncodeToString(digest[:8])
}
