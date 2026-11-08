package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/sqlutil"
)

// Admission receipts have no foreign key to events: retention may remove the
// canonical event, but must never make a previously admitted intent fan out again.
// This ledger has no automatic pruning policy. Reclamation requires an explicit
// upstream replay-retirement protocol, which this dormant seam does not provide.
const terminalAdmissionSchema = `
CREATE TABLE IF NOT EXISTS terminal_event_admissions (
 event_id TEXT PRIMARY KEY,
 intent_json TEXT NOT NULL,
 canonical_payload BLOB NOT NULL,
 canonical_sha256 TEXT NOT NULL,
 store_id INTEGER NOT NULL CHECK(store_id > 0)
);
CREATE TRIGGER IF NOT EXISTS terminal_event_admissions_no_replace
 BEFORE INSERT ON terminal_event_admissions WHEN EXISTS (SELECT 1 FROM terminal_event_admissions WHERE event_id=NEW.event_id)
 BEGIN SELECT RAISE(ABORT,'terminal admission receipts are immutable'); END;
CREATE TRIGGER IF NOT EXISTS terminal_event_admissions_no_update
 BEFORE UPDATE ON terminal_event_admissions BEGIN SELECT RAISE(ABORT,'terminal admission receipts are immutable'); END;
CREATE TRIGGER IF NOT EXISTS terminal_event_admissions_no_delete
 BEFORE DELETE ON terminal_event_admissions BEGIN SELECT RAISE(ABORT,'terminal admission receipts are immutable'); END;
`

var ErrTerminalAdmissionConflict = errors.New("terminal event admission content conflict")

// TerminalAdmission retains the exact source intent and acknowledged canonical
// bytes. StoreID is the original event identity, even after event retention.
type TerminalAdmission struct {
	EventID          string
	IntentJSON       string
	CanonicalPayload []byte
	CanonicalSHA256  string
	StoreID          int64
}

// CheckTerminalAdmissionTx returns nil for a new intent, the original receipt
// for an exact replay, or a conflict for inconsistent receipt/canonical bytes.
// The same transaction must subsequently admit and record a new intent.
func (s *SQLiteStore) CheckTerminalAdmissionTx(ctx context.Context, tx *sql.Tx, event Event, intentJSON string) (*TerminalAdmission, error) {
	var receipt TerminalAdmission
	err := tx.QueryRowContext(ctx, `SELECT event_id,intent_json,canonical_payload,canonical_sha256,store_id FROM terminal_event_admissions WHERE event_id=?`, event.EventID).
		Scan(&receipt.EventID, &receipt.IntentJSON, &receipt.CanonicalPayload, &receipt.CanonicalSHA256, &receipt.StoreID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read terminal admission: %w", err)
	}
	digest := sha256.Sum256(event.Payload)
	if receipt.EventID != event.EventID || receipt.IntentJSON != intentJSON || !bytes.Equal(receipt.CanonicalPayload, event.Payload) || receipt.CanonicalSHA256 != hex.EncodeToString(digest[:]) || receipt.StoreID <= 0 {
		return nil, fmt.Errorf("%w: %s", ErrTerminalAdmissionConflict, event.EventID)
	}
	// A retained event must still agree with the receipt. Absence is legitimate
	// after pruning; disagreement is never silently acknowledged as a duplicate.
	var id int64
	var payload []byte
	var source, eventType, created, target, correlation string
	err = tx.QueryRowContext(ctx, `SELECT id,payload,source_scenario,event_type,created_at,target_scenario,correlation_id FROM events WHERE event_id=?`, event.EventID).Scan(&id, &payload, &source, &eventType, &created, &target, &correlation)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("check admitted canonical event: %w", err)
	}
	if err == nil && (id != receipt.StoreID || !bytes.Equal(payload, event.Payload) || source != event.SourceScenario || eventType != event.EventType || created != event.CreatedAt.UTC().Format(sqlutil.TimestampFormat) || target != event.TargetScenario || correlation != event.CorrelationID) {
		return nil, fmt.Errorf("%w: retained canonical event %s", ErrTerminalAdmissionConflict, event.EventID)
	}
	return &receipt, nil
}

// InsertTerminalAdmissionTx joins the canonical event and subscription delivery
// writes in their existing owner transaction. It never commits on its own.
func (s *SQLiteStore) InsertTerminalAdmissionTx(ctx context.Context, tx *sql.Tx, receipt TerminalAdmission) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO terminal_event_admissions (event_id,intent_json,canonical_payload,canonical_sha256,store_id) VALUES (?,?,?,?,?)`, receipt.EventID, receipt.IntentJSON, receipt.CanonicalPayload, receipt.CanonicalSHA256, receipt.StoreID)
	if err != nil {
		return fmt.Errorf("record terminal admission: %w", err)
	}
	return nil
}
