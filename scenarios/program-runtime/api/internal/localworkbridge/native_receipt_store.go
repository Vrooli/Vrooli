package localworkbridge

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	coredb "github.com/vrooli/api-core/database"
)

//go:embed native_receipts.sql
var nativeReceiptSchema string

// NativeReceiptSchema declares only authenticated receipt/selection metadata.
// It is not registered in main and contains no program execution ledger.
type NativeReceiptSchema struct{}

func (NativeReceiptSchema) Schema() string { return nativeReceiptSchema }

// NativeReceiptStore owns trusted native receipt metadata in an explicitly
// supplied SQLite database. The deployment owner must isolate this database
// from all worker-writable state. This constructor performs no I/O, chmod, path
// discovery, routing or automatic production migration.
type NativeReceiptStore struct{ db *sql.DB }

func NewNativeReceiptStore(db *sql.DB) *NativeReceiptStore { return &NativeReceiptStore{db: db} }

type nativeReceipt struct{ ID, Type, Scenario, BodySHA256, ReceivedAt string }

// begin pins the commissioning scope before reading/writing any record. The
// harmless owner-row update acquires the SQLite write lock before snapshot
// reads, serializing distinct store handles and restarts without a Go-only lock.
func (s *NativeReceiptStore) begin(ctx context.Context, digest string) (*sql.Tx, error) {
	if s == nil || s.db == nil || ctx.Err() != nil {
		return nil, ErrSourceAuthority
	}
	if err := coredb.EnsureSchemas(ctx, s.db, NativeReceiptSchema{}); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*sql.Tx, error) { _ = tx.Rollback(); return nil, err }
	if _, err = tx.ExecContext(ctx, "INSERT INTO local_work_native_owner(singleton,scope_sha256) VALUES(1,?) ON CONFLICT(singleton) DO NOTHING", digest); err != nil {
		return fail(err)
	}
	result, err := tx.ExecContext(ctx, "UPDATE local_work_native_owner SET scope_sha256=scope_sha256 WHERE singleton=1 AND scope_sha256=?", digest)
	if err != nil {
		return fail(err)
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return fail(ErrSourceAuthority)
	}
	return tx, nil
}
func (s *NativeReceiptStore) accept(ctx context.Context, digest string, r nativeReceipt) error {
	tx, err := s.begin(ctx, digest)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var old nativeReceipt
	err = tx.QueryRowContext(ctx, "SELECT event_id,event_type,source_scenario,body_sha256,received_at FROM local_work_native_receipts WHERE event_id=?", r.ID).Scan(&old.ID, &old.Type, &old.Scenario, &old.BodySHA256, &old.ReceivedAt)
	if err == nil {
		if old.ID != r.ID || old.Type != r.Type || old.Scenario != r.Scenario || old.BodySHA256 != r.BodySHA256 {
			return ErrNativeReceiptConflict
		}
		return tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	var count int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM local_work_native_receipts").Scan(&count); err != nil {
		return err
	}
	if count >= 10000 {
		return ErrNativeReceiptCapacity
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO local_work_native_receipts(event_id,event_type,source_scenario,body_sha256,received_at) VALUES(?,?,?,?,?)", r.ID, r.Type, r.Scenario, r.BodySHA256, r.ReceivedAt)
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (s *NativeReceiptStore) selectReceipts(ctx context.Context, scope NativeReceiptScope, digest string, now time.Time) (*NativeSelection, error) {
	tx, err := s.begin(ctx, digest)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	out := &NativeSelection{scope: scope, digest: digest}
	var raw string
	var retainedScope string
	err = tx.QueryRowContext(ctx, "SELECT scope_sha256,request_key,admission_deadline,receipts_json FROM local_work_native_selection WHERE singleton=1").Scan(&retainedScope, &out.key, &out.deadline, &raw)
	if err == nil {
		if retainedScope != digest || len(raw) > 32768 || json.Unmarshal([]byte(raw), &out.receipts) != nil || !validNativeSelection(out) {
			return nil, ErrSourceAuthority
		}
		canonical, _ := json.Marshal(out.receipts)
		if string(canonical) != raw {
			return nil, ErrSourceAuthority
		}
		// Retained selection must still correspond to exact authenticated rows.
		for _, r := range out.receipts {
			var present int
			if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM local_work_native_receipts WHERE event_id=? AND event_type=? AND source_scenario=? AND body_sha256=? AND received_at=?", r.ID, r.Type, r.Scenario, r.BodySHA256, r.ReceivedAt).Scan(&present); err != nil || present != 1 {
				return nil, ErrSourceAuthority
			}
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		return out, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, "SELECT event_id,event_type,source_scenario,body_sha256,received_at FROM local_work_native_receipts r WHERE NOT EXISTS (SELECT 1 FROM local_work_native_completed_events c WHERE c.event_id=r.event_id) ORDER BY event_id LIMIT 64")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var r nativeReceipt
		if err = rows.Scan(&r.ID, &r.Type, &r.Scenario, &r.BodySHA256, &r.ReceivedAt); err != nil {
			rows.Close()
			return nil, err
		}
		out.receipts = append(out.receipts, r)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if len(out.receipts) == 0 {
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	out.deadline = now.Add(5 * time.Minute).Format(time.RFC3339Nano)
	out.key = nativeSelectionKey(out)
	if !validNativeSelection(out) {
		return nil, ErrSourceAuthority
	}
	data, _ := json.Marshal(out.receipts)
	_, err = tx.ExecContext(ctx, "INSERT INTO local_work_native_selection(singleton,scope_sha256,request_key,admission_deadline,receipts_json) VALUES(1,?,?,?,?)", digest, out.key, out.deadline, string(data))
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}
func nativeSelectionKey(s *NativeSelection) string {
	// Deadline participates so corruption cannot silently extend one attempt.
	raw, _ := json.Marshal(struct {
		Scope, Deadline string
		Receipts        []nativeReceipt
	}{s.digest, s.deadline, s.receipts})
	sum := sha256.Sum256(raw)
	return "local-work-" + hex.EncodeToString(sum[:])
}
func validNativeSelection(s *NativeSelection) bool {
	if len(s.receipts) == 0 || len(s.receipts) > 64 || !sourceDigest.MatchString(s.digest) || !sourceKey.MatchString(s.key) || s.key != nativeSelectionKey(s) {
		return false
	}
	if _, err := time.Parse(time.RFC3339Nano, s.deadline); err != nil {
		return false
	}
	last := ""
	for _, r := range s.receipts {
		if !sourceAtom.MatchString(r.ID) || r.ID <= last || r.Scenario != "agent-manager" || !sourceDigest.MatchString(r.BodySHA256) {
			return false
		}
		allowed := false
		for _, typ := range s.scope.AllowedTypes {
			if typ == r.Type {
				allowed = true
			}
		}
		if !allowed {
			return false
		}
		if _, err := time.Parse(time.RFC3339Nano, r.ReceivedAt); err != nil {
			return false
		}
		last = r.ID
	}
	return true
}

// completeSelection commits the tombstone, membership and retirement together
// under the same SQLite owner-row serialization used by selection/acceptance.
// The caller holds both native and collector authority boundaries throughout.
func (s *NativeReceiptStore) completeSelection(ctx context.Context, selected *NativeSelection, result SourceBrokerProgram, recheck func() error) error {
	tx, err := s.begin(ctx, selected.digest)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var collectorDigest string
	if err = tx.QueryRowContext(ctx, "SELECT collector_sha256 FROM local_work_native_bindings WHERE request_key=?", selected.key).Scan(&collectorDigest); err != nil || collectorDigest != result.Identity.ScopeSHA256 {
		return ErrSourceAuthority
	}
	receipts, _ := json.Marshal(selected.receipts)
	identity, _ := json.Marshal(result.Identity)
	sum := sha256.Sum256([]byte(result.Stdout))
	hash := hex.EncodeToString(sum[:])
	var scope, deadline, raw, retainedIdentity, status, retainedHash string
	err = tx.QueryRowContext(ctx, "SELECT scope_sha256,admission_deadline,receipts_json,program_identity_json,program_status,stdout_sha256 FROM local_work_native_completed WHERE request_key=?", selected.key).Scan(&scope, &deadline, &raw, &retainedIdentity, &status, &retainedHash)
	if err == nil {
		if scope != selected.digest || deadline != selected.deadline || raw != string(receipts) || retainedIdentity != string(identity) || status != result.Status || retainedHash != hash {
			return ErrSourceAuthority
		}
		// Idempotence never deletes the current selection, which may be a later one.
		if recheck() != nil || ctx.Err() != nil {
			return ErrSourceAuthority
		}
		return tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	err = tx.QueryRowContext(ctx, "SELECT scope_sha256,admission_deadline,receipts_json FROM local_work_native_selection WHERE singleton=1 AND request_key=?", selected.key).Scan(&scope, &deadline, &raw)
	if err != nil || scope != selected.digest || deadline != selected.deadline || raw != string(receipts) {
		return ErrSourceAuthority
	}
	for _, r := range selected.receipts {
		var n int
		if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM local_work_native_receipts WHERE event_id=? AND event_type=? AND source_scenario=? AND body_sha256=? AND received_at=?", r.ID, r.Type, r.Scenario, r.BodySHA256, r.ReceivedAt).Scan(&n); err != nil || n != 1 {
			return ErrSourceAuthority
		}
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO local_work_native_completed(request_key,scope_sha256,admission_deadline,receipts_json,program_identity_json,program_status,stdout_sha256) VALUES(?,?,?,?,?,?,?)", selected.key, selected.digest, selected.deadline, string(receipts), string(identity), result.Status, hash)
	if err != nil {
		return err
	}
	for _, r := range selected.receipts {
		if _, err = tx.ExecContext(ctx, "INSERT INTO local_work_native_completed_events(event_id,request_key) VALUES(?,?)", r.ID, selected.key); err != nil {
			return err
		}
	}
	removed, err := tx.ExecContext(ctx, "DELETE FROM local_work_native_selection WHERE singleton=1 AND request_key=?", selected.key)
	if err != nil {
		return err
	}
	n, err := removed.RowsAffected()
	if err != nil || n != 1 || recheck() != nil || ctx.Err() != nil {
		return ErrSourceAuthority
	}
	return tx.Commit()
}

func (s *NativeReceiptStore) bindSelection(ctx context.Context, selected *NativeSelection, eventsDigest, collectorDigest string) error {
	tx, err := s.begin(ctx, selected.digest)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	raw, _ := json.Marshal(selected.receipts)
	var scope, deadline, receipts string
	err = tx.QueryRowContext(ctx, "SELECT scope_sha256,admission_deadline,receipts_json FROM local_work_native_selection WHERE singleton=1 AND request_key=?", selected.key).Scan(&scope, &deadline, &receipts)
	if err != nil || scope != selected.digest || deadline != selected.deadline || receipts != string(raw) {
		return ErrSourceAuthority
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO local_work_native_bindings(request_key,events_sha256,collector_sha256) VALUES(?,?,?) ON CONFLICT(request_key) DO NOTHING", selected.key, eventsDigest, collectorDigest)
	if err != nil {
		return err
	}
	var oldEvents, oldCollector string
	err = tx.QueryRowContext(ctx, "SELECT events_sha256,collector_sha256 FROM local_work_native_bindings WHERE request_key=?", selected.key).Scan(&oldEvents, &oldCollector)
	if err != nil || oldEvents != eventsDigest || oldCollector != collectorDigest || ctx.Err() != nil {
		return ErrSourceAuthority
	}
	return tx.Commit()
}
