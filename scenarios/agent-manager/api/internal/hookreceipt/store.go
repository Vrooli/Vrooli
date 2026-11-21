// Package hookreceipt stores authenticated, single-use policy evaluator
// receipts bound to one Agent Manager run invocation.
package hookreceipt

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"agent-manager/internal/sqlcompat"

	"github.com/google/uuid"
)

var (
	ErrInvalidInput      = errors.New("invalid hook receipt input")
	ErrInvalidCredential = errors.New("invalid hook receipt credential")
	ErrExpiredCredential = errors.New("hook receipt credential expired")
	ErrRevokedCredential = errors.New("hook receipt credential revoked")
	ErrWrongBinding      = errors.New("hook receipt credential binding mismatch")
	ErrCredentialUsed    = errors.New("hook receipt credential already consumed")
)

const credentialBytes = 32

// Consumer is the narrow write contract used by the authenticated HTTP owner.
type Consumer interface {
	Consume(context.Context, string, uuid.UUID, uuid.UUID, Receipt) error
}

// Receipt is the complete persisted evaluator evidence payload. Run and
// invocation binding are carried by the credential row and are deliberately
// not duplicated in this projection.
type Receipt struct {
	ReceiptID   string    `json:"receiptId"`
	SessionID   string    `json:"sessionId"`
	Runner      string    `json:"runner"`
	EvaluatorID string    `json:"evaluatorId"`
	DurationMS  int64     `json:"durationMs"`
	OccurredAt  time.Time `json:"occurredAt"`
}

// Store persists credentials and consumed receipts. now is replaceable from
// package tests so expiry and timestamp boundaries do not depend on wall time.
type Store struct {
	db  sqlcompat.DB
	now func() time.Time
}

// NewStore constructs the store over Agent Manager's routed database seam.
func NewStore(db sqlcompat.DB) *Store { return &Store{db: db, now: time.Now} }

// Issue creates a cryptographically random bearer credential. Its digest,
// run/invocation binding and validity window are persisted; issuing a new
// credential revokes any prior outstanding credential for the same run.
func (s *Store) Issue(ctx context.Context, runID, invocationID uuid.UUID, ttl time.Duration) (string, error) {
	if s == nil || s.db == nil || runID == uuid.Nil || invocationID == uuid.Nil || ttl <= 0 {
		return "", fmt.Errorf("issue evaluator hook credential: %w", ErrInvalidInput)
	}
	var raw [credentialBytes]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate evaluator hook credential: %w", err)
	}
	credential := base64.RawURLEncoding.EncodeToString(raw[:])
	digest := sha256.Sum256([]byte(credential))
	now := s.now().UTC()
	expires := now.Add(ttl)
	if !expires.After(now) {
		return "", fmt.Errorf("issue evaluator hook credential: %w", ErrInvalidInput)
	}

	tx, done, err := s.begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin evaluator hook credential issue: %w", err)
	}
	defer done()
	if _, err := tx.ExecContext(ctx, `UPDATE evaluator_hook_receipts
SET revoked_at = ? WHERE run_id = ? AND consumed_at IS NULL AND revoked_at IS NULL`, formatTime(now), runID.String()); err != nil {
		return "", fmt.Errorf("revoke prior evaluator hook credential: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO evaluator_hook_receipts
(credential_hash, run_id, invocation_id, created_at, expires_at)
VALUES (?, ?, ?, ?, ?)`, digest[:], runID.String(), invocationID.String(), formatTime(now), formatTime(expires)); err != nil {
		return "", fmt.Errorf("persist evaluator hook credential: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit evaluator hook credential: %w", err)
	}
	return credential, nil
}

// Consume atomically validates and stores a receipt against the exact run and
// invocation that received the credential. A receipt ID or token can commit
// only once, including under concurrent requests.
func (s *Store) Consume(ctx context.Context, credential string, runID, invocationID uuid.UUID, receipt Receipt) error {
	if s == nil || s.db == nil || runID == uuid.Nil || invocationID == uuid.Nil || !validCredential(credential) {
		return fmt.Errorf("consume evaluator hook receipt: %w", ErrInvalidInput)
	}
	if err := validateReceipt(receipt); err != nil {
		return err
	}
	digest := sha256.Sum256([]byte(credential))
	now := s.now().UTC()
	tx, done, err := s.begin(ctx)
	if err != nil {
		return fmt.Errorf("begin evaluator hook receipt consume: %w", err)
	}
	defer done()

	var storedRun, storedInvocation, createdAt, expiresAt string
	var consumedAt, revokedAt sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT run_id, invocation_id, created_at, expires_at, consumed_at, revoked_at
FROM evaluator_hook_receipts WHERE credential_hash = ?`, digest[:]).Scan(&storedRun, &storedInvocation, &createdAt, &expiresAt, &consumedAt, &revokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidCredential
	}
	if err != nil {
		return fmt.Errorf("read evaluator hook credential: %w", err)
	}
	if storedRun != runID.String() || storedInvocation != invocationID.String() {
		return ErrWrongBinding
	}
	if revokedAt.Valid {
		return ErrRevokedCredential
	}
	if consumedAt.Valid {
		return ErrCredentialUsed
	}
	expiry, err := time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil {
		return fmt.Errorf("parse evaluator hook credential expiry: %w", err)
	}
	created, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return fmt.Errorf("parse evaluator hook credential creation time: %w", err)
	}
	if !now.Before(expiry) {
		return ErrExpiredCredential
	}
	if receipt.OccurredAt.Before(created) || receipt.OccurredAt.After(expiry) || receipt.OccurredAt.After(now.Add(time.Minute)) {
		return fmt.Errorf("receipt occurredAt outside credential validity window: %w", ErrInvalidInput)
	}

	result, err := tx.ExecContext(ctx, `UPDATE evaluator_hook_receipts SET
consumed_at = ?, receipt_id = ?, session_id = ?, runner = ?, evaluator_id = ?, duration_ms = ?, occurred_at = ?
WHERE credential_hash = ? AND run_id = ? AND invocation_id = ?
  AND consumed_at IS NULL AND revoked_at IS NULL AND expires_at > ?`,
		formatTime(now), receipt.ReceiptID, receipt.SessionID, receipt.Runner, receipt.EvaluatorID,
		receipt.DurationMS, formatTime(receipt.OccurredAt), digest[:], runID.String(), invocationID.String(), formatTime(now))
	if err != nil {
		return fmt.Errorf("persist evaluator hook receipt: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check evaluator hook receipt consume: %w", err)
	}
	if changed != 1 {
		return ErrCredentialUsed
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit evaluator hook receipt: %w", err)
	}
	return nil
}

// RevokeOutstanding revokes every unconsumed, unrevoked credential. Call at
// Agent Manager startup to ensure a credential from a prior process is dead.
func (s *Store) RevokeOutstanding(ctx context.Context) (int64, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("revoke outstanding evaluator hook credentials: %w", ErrInvalidInput)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE evaluator_hook_receipts SET revoked_at = ?
WHERE consumed_at IS NULL AND revoked_at IS NULL`, formatTime(s.now().UTC()))
	if err != nil {
		return 0, fmt.Errorf("revoke outstanding evaluator hook credentials: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count revoked evaluator hook credentials: %w", err)
	}
	return count, nil
}

// RevokeRun revokes any outstanding credential for one terminal run.
func (s *Store) RevokeRun(ctx context.Context, runID uuid.UUID) (int64, error) {
	if s == nil || s.db == nil || runID == uuid.Nil {
		return 0, fmt.Errorf("revoke evaluator hook credential for run: %w", ErrInvalidInput)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE evaluator_hook_receipts SET revoked_at = ?
WHERE run_id = ? AND consumed_at IS NULL AND revoked_at IS NULL`, formatTime(s.now().UTC()), runID.String())
	if err != nil {
		return 0, fmt.Errorf("revoke evaluator hook credential for run %s: %w", runID, err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count revoked evaluator hook credentials: %w", err)
	}
	return count, nil
}

// ReceiptsForRuns returns consumed receipts for the requested run UUIDs. It
// intentionally exposes no credential digest, invocation binding, or raw
// evaluator input.
func (s *Store) ReceiptsForRuns(ctx context.Context, runIDs []uuid.UUID) (map[uuid.UUID][]Receipt, error) {
	out := make(map[uuid.UUID][]Receipt, len(runIDs))
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("read evaluator hook receipts: %w", ErrInvalidInput)
	}
	ids := make([]uuid.UUID, 0, len(runIDs))
	seen := make(map[uuid.UUID]struct{}, len(runIDs))
	for _, id := range runIDs {
		if id == uuid.Nil {
			return nil, fmt.Errorf("read evaluator hook receipts: %w", ErrInvalidInput)
		}
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			ids = append(ids, id)
			out[id] = []Receipt{}
		}
	}
	for start := 0; start < len(ids); start += 400 {
		end := min(start+400, len(ids))
		chunk := ids[start:end]
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(chunk)), ",")
		args := make([]any, 0, len(chunk))
		for _, id := range chunk {
			args = append(args, id.String())
		}
		rows, err := s.db.QueryxContext(ctx, `SELECT run_id, receipt_id, session_id, runner, evaluator_id, duration_ms, occurred_at
FROM evaluator_hook_receipts WHERE consumed_at IS NOT NULL AND run_id IN (`+placeholders+`)
ORDER BY run_id, occurred_at, receipt_id`, args...)
		if err != nil {
			return nil, fmt.Errorf("read evaluator hook receipts: %w", err)
		}
		for rows.Next() {
			var runIDText, occurredAt string
			var receipt Receipt
			if err := rows.Scan(&runIDText, &receipt.ReceiptID, &receipt.SessionID, &receipt.Runner, &receipt.EvaluatorID, &receipt.DurationMS, &occurredAt); err != nil {
				_ = rows.Close()
				return nil, fmt.Errorf("scan evaluator hook receipt: %w", err)
			}
			runID, err := uuid.Parse(runIDText)
			if err != nil {
				_ = rows.Close()
				return nil, fmt.Errorf("parse stored evaluator hook run ID: %w", err)
			}
			receipt.OccurredAt, err = time.Parse(time.RFC3339Nano, occurredAt)
			if err != nil {
				_ = rows.Close()
				return nil, fmt.Errorf("parse stored evaluator hook receipt time: %w", err)
			}
			out[runID] = append(out[runID], receipt)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("iterate evaluator hook receipts: %w", err)
		}
		if err := rows.Close(); err != nil {
			return nil, fmt.Errorf("close evaluator hook receipt rows: %w", err)
		}
	}
	return out, nil
}

func (s *Store) begin(ctx context.Context) (*sql.Tx, func(), error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return nil, func() {}, err
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		_ = conn.Close()
		return nil, func() {}, err
	}
	done := func() {
		_ = tx.Rollback()
		_ = conn.Close()
	}
	return tx, done, nil
}

func validCredential(credential string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(credential)
	return err == nil && len(decoded) == credentialBytes && base64.RawURLEncoding.EncodeToString(decoded) == credential
}

func validateReceipt(receipt Receipt) error {
	parsedID, err := uuid.Parse(receipt.ReceiptID)
	if err != nil || parsedID == uuid.Nil || parsedID.String() != receipt.ReceiptID {
		return fmt.Errorf("receiptId must be a canonical nonzero UUID: %w", ErrInvalidInput)
	}
	if !validTextID(receipt.SessionID, 512) || !validTextID(receipt.Runner, 128) || !validTextID(receipt.EvaluatorID, 512) {
		return fmt.Errorf("receipt identifiers must be nonempty, bounded, and contain no control characters: %w", ErrInvalidInput)
	}
	if receipt.DurationMS < 0 || receipt.DurationMS > 10*60*1000 || receipt.OccurredAt.IsZero() {
		return fmt.Errorf("receipt requires a bounded nonnegative durationMs and occurredAt: %w", ErrInvalidInput)
	}
	return nil
}

func validTextID(value string, maxBytes int) bool {
	if value == "" || len(value) > maxBytes || strings.TrimSpace(value) != value || !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
