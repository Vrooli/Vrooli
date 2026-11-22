package intake

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"content-desk/internal/artifacts"
	"github.com/google/uuid"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type transactionBeginner interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type service struct {
	db     SQLExecutor
	drafts artifacts.Repository
	now    func() time.Time
}

func NewService(db SQLExecutor) Service {
	return &service{db: db, now: func() time.Time { return time.Now().UTC() }}
}

// NewServiceWithDrafts enables explicit conversion of an admitted receipt to
// ordinary requested drafts. Admission remains usable without this seam.
func NewServiceWithDrafts(db SQLExecutor, drafts artifacts.Repository) Service {
	return &service{db: db, drafts: drafts, now: func() time.Time { return time.Now().UTC() }}
}

func (s *service) Preview(_ context.Context, req Request) (Preview, error) {
	if err := validateRequest(req, false); err != nil {
		return Preview{}, err
	}
	p := Preview{Source: req.Source, Assessments: make([]Assessment, 0, len(req.Items))}
	seen := map[string]bool{}
	for _, item := range req.Items {
		if seen[item.ExternalID] {
			return Preview{}, fmt.Errorf("duplicate external item id %q", item.ExternalID)
		}
		seen[item.ExternalID] = true
		p.Assessments = append(p.Assessments, assess(item))
	}
	return p, nil
}

func (s *service) Admit(ctx context.Context, req Request) ([]StoredItem, error) {
	p, err := s.Preview(ctx, req)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return nil, errors.New("idempotency key is required")
	}
	fingerprint, err := requestFingerprint(req)
	if err != nil {
		return nil, err
	}
	if prior, found, e := s.findBatch(ctx, req.IdempotencyKey); e != nil {
		return nil, e
	} else if found {
		if prior.fingerprint != fingerprint {
			return nil, errors.New("idempotency key was reused with a different article intake request")
		}
		return s.List(ctx, prior.id)
	}
	sourceKey, err := sourceFingerprint(req.Source)
	if err != nil {
		return nil, err
	}
	if prior, found, e := s.findSourceBatch(ctx, sourceKey); e != nil {
		return nil, e
	} else if found {
		if prior.fingerprint == fingerprint {
			return s.List(ctx, prior.id)
		}
		return nil, errors.New("source revision already has an intake batch; reconcile unresolved items instead of importing it again")
	}
	b, ok := s.db.(transactionBeginner)
	if !ok {
		return nil, errors.New("transactional storage is required for article intake")
	}
	tx, err := b.BeginTx(ctx, nil)
	if err != nil {
		return s.recoverAdmission(ctx, req, fingerprint, err)
	}
	defer tx.Rollback()
	batchID := uuid.NewString()
	now := s.now()
	sourceJSON, _ := json.Marshal(req.Source)
	_, err = tx.ExecContext(ctx, `INSERT INTO article_intake_batches(id,idempotency_key,source_key,request_fingerprint,source_json,created_at) VALUES(?,?,?,?,?,?)`, batchID, req.IdempotencyKey, sourceKey, fingerprint, string(sourceJSON), now.Format(time.RFC3339Nano))
	if err != nil {
		_ = tx.Rollback()
		return s.recoverAdmission(ctx, req, fingerprint, err)
	}
	for i, item := range req.Items {
		if err = saveInitial(ctx, tx, batchID, item, p.Assessments[i], now); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		_ = tx.Rollback()
		return s.recoverAdmission(ctx, req, fingerprint, err)
	}
	return s.List(ctx, batchID)
}

func (s *service) Reconcile(ctx context.Context, batchID string, req Request) ([]StoredItem, error) {
	batchID = strings.TrimSpace(batchID)
	if batchID == "" {
		return nil, errors.New("batch id is required")
	}
	if err := validateRequest(req, false); err != nil {
		return nil, err
	}
	var sourceJSON string
	if err := s.db.QueryRowContext(ctx, `SELECT source_json FROM article_intake_batches WHERE id=?`, batchID).Scan(&sourceJSON); err != nil {
		return nil, err
	}
	var source Source
	if err := json.Unmarshal([]byte(sourceJSON), &source); err != nil {
		return nil, err
	}
	if source != req.Source {
		return nil, errors.New("reconcile source artifact, revision, producer and intent must match the batch")
	}
	seen := map[string]bool{}
	for _, item := range req.Items {
		if seen[item.ExternalID] {
			return nil, fmt.Errorf("duplicate external item id %q", item.ExternalID)
		}
		seen[item.ExternalID] = true
	}
	b, ok := s.db.(transactionBeginner)
	if !ok {
		return nil, errors.New("transactional storage is required for article intake reconciliation")
	}
	tx, err := b.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := s.now()
	for _, item := range req.Items {
		var oldStatus string
		err := tx.QueryRowContext(ctx, `SELECT status FROM article_intake_items WHERE batch_id=? AND external_id=?`, batchID, item.ExternalID).Scan(&oldStatus)
		if err == sql.ErrNoRows {
			continue
		} // Reconciliation never silently expands the admitted set.
		if err != nil {
			return nil, err
		}
		if oldStatus != "unresolved" {
			continue
		} // Admitted and rejected work is immutable here.
		a := assess(item)
		if err := updateUnresolved(ctx, tx, batchID, item, a, now); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.List(ctx, batchID)
}

func (s *service) List(ctx context.Context, batchID string) ([]StoredItem, error) {
	query := `SELECT i.batch_id,b.source_json,i.external_id,i.payload_json,i.status,i.unresolved_fields_json,i.rejected_fields_json,i.reason,i.current_receipt_id FROM article_intake_items i JOIN article_intake_batches b ON b.id=i.batch_id`
	args := []any{}
	if batchID != "" {
		query += ` WHERE i.batch_id=?`
		args = append(args, batchID)
	}
	query += ` ORDER BY b.created_at,i.external_id`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	out := []StoredItem{}
	for rows.Next() {
		var x StoredItem
		var sourceText, payload, unresolved, rejected, current string
		if err := rows.Scan(&x.BatchID, &sourceText, &x.Item.ExternalID, &payload, &x.Assessment.Status, &unresolved, &rejected, &x.Assessment.Reason, &current); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(sourceText), &x.Source); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(payload), &x.Item); err != nil {
			return nil, err
		}
		x.Item.RawPayload = json.RawMessage(payload)
		if err := json.Unmarshal([]byte(unresolved), &x.Assessment.UnresolvedFields); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(rejected), &x.Assessment.RejectedFields); err != nil {
			return nil, err
		}
		x.Assessment.ExternalID = x.Item.ExternalID
		x.Receipt.ID = current // Resolve the current receipt after closing this cursor; SQLite uses a single pooled connection.
		out = append(out, x)
	}
	rowsErr := rows.Err()
	closeErr := rows.Close()
	if rowsErr != nil {
		return nil, rowsErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	for i := range out {
		out[i].Receipts, err = s.receipts(ctx, out[i].BatchID, out[i].Item.ExternalID)
		if err != nil {
			return nil, err
		}
		for _, receipt := range out[i].Receipts {
			if receipt.ID == out[i].Receipt.ID {
				out[i].Receipt = receipt
				break
			}
		}
	}
	return out, nil
}

func (s *service) receipts(ctx context.Context, batchID, externalID string) ([]Receipt, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,batch_id,external_id,attempt,status,unresolved_fields_json,rejected_fields_json,reason,payload_json,payload_sha256,created_at FROM article_intake_receipts WHERE batch_id=? AND external_id=? ORDER BY attempt`, batchID, externalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Receipt{}
	for rows.Next() {
		var r Receipt
		var unresolved, rejected, created string
		var payload string
		if err := rows.Scan(&r.ID, &r.BatchID, &r.ExternalID, &r.Attempt, &r.Status, &unresolved, &rejected, &r.Reason, &payload, &r.PayloadSHA256, &created); err != nil {
			return nil, err
		}
		r.RawPayload = json.RawMessage(payload)
		if err := json.Unmarshal([]byte(unresolved), &r.UnresolvedFields); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(rejected), &r.RejectedFields); err != nil {
			return nil, err
		}
		r.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func validateRequest(req Request, requireKey bool) error {
	if strings.TrimSpace(req.Source.ArtifactID) == "" || strings.TrimSpace(req.Source.RevisionID) == "" || strings.TrimSpace(req.Source.Producer) == "" || strings.TrimSpace(req.Source.Intent) == "" {
		return errors.New("source artifact id, revision id, producer and import intent are required")
	}
	if requireKey && strings.TrimSpace(req.IdempotencyKey) == "" {
		return errors.New("idempotency key is required")
	}
	if len(req.Items) == 0 {
		return errors.New("at least one article item is required")
	}
	for _, item := range req.Items {
		if strings.TrimSpace(item.ExternalID) == "" {
			return errors.New("every article item requires an external id")
		}
	}
	return nil
}

func assess(item Item) Assessment {
	a := Assessment{ExternalID: item.ExternalID, Status: "admissible", RejectedFields: append([]string(nil), item.RejectedFields...)}
	if len(a.RejectedFields) > 0 {
		sort.Strings(a.RejectedFields)
		a.Status = "rejected"
		a.Reason = "source fields were explicitly rejected"
		return a
	}
	if strings.TrimSpace(item.Kind) == "" {
		a.UnresolvedFields = append(a.UnresolvedFields, "kind")
	}
	if strings.TrimSpace(item.Title) == "" {
		a.UnresolvedFields = append(a.UnresolvedFields, "title")
	}
	if strings.TrimSpace(item.Body) == "" {
		a.UnresolvedFields = append(a.UnresolvedFields, "body")
	}
	for i, v := range item.Variants {
		prefix := fmt.Sprintf("variants[%d]", i)
		if strings.TrimSpace(v.ID) == "" {
			a.UnresolvedFields = append(a.UnresolvedFields, prefix+".id")
		}
		if strings.TrimSpace(v.Name) == "" {
			a.UnresolvedFields = append(a.UnresolvedFields, prefix+".name")
		}
		if strings.TrimSpace(v.Channel) == "" {
			a.UnresolvedFields = append(a.UnresolvedFields, prefix+".channel")
		}
		if strings.TrimSpace(v.Body) == "" {
			a.UnresolvedFields = append(a.UnresolvedFields, prefix+".body")
		}
	}
	if len(a.UnresolvedFields) > 0 {
		a.Status = "unresolved"
		a.Reason = "required structured article fields are missing"
	}
	return a
}

func itemPayload(item Item) ([]byte, error) {
	if len(item.RawPayload) > 0 {
		if !json.Valid(item.RawPayload) {
			return nil, errors.New("raw payload must be valid JSON")
		}
		return append([]byte(nil), item.RawPayload...), nil
	}
	return json.Marshal(item)
}

func saveInitial(ctx context.Context, tx *sql.Tx, batchID string, item Item, a Assessment, now time.Time) error {
	payload, err := itemPayload(item)
	if err != nil {
		return err
	}
	fields, _ := json.Marshal(a.UnresolvedFields)
	rejected, _ := json.Marshal(a.RejectedFields)
	id := uuid.NewString()
	_, err = tx.ExecContext(ctx, `INSERT INTO article_intake_items(batch_id,external_id,payload_json,status,unresolved_fields_json,rejected_fields_json,reason,current_receipt_id) VALUES(?,?,?,?,?,?,?,?)`, batchID, item.ExternalID, string(payload), a.Status, string(fields), string(rejected), a.Reason, id)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO article_intake_receipts(id,batch_id,external_id,attempt,status,unresolved_fields_json,rejected_fields_json,reason,payload_json,payload_sha256,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, id, batchID, item.ExternalID, 1, a.Status, string(fields), string(rejected), a.Reason, string(payload), digest(payload), now.Format(time.RFC3339Nano))
	return err
}

func updateUnresolved(ctx context.Context, tx *sql.Tx, batchID string, item Item, a Assessment, now time.Time) error {
	payload, err := itemPayload(item)
	if err != nil {
		return err
	}
	var attempt int
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(attempt),0)+1 FROM article_intake_receipts WHERE batch_id=? AND external_id=?`, batchID, item.ExternalID).Scan(&attempt); err != nil {
		return err
	}
	fields, _ := json.Marshal(a.UnresolvedFields)
	rejected, _ := json.Marshal(a.RejectedFields)
	id := uuid.NewString()
	_, err = tx.ExecContext(ctx, `INSERT INTO article_intake_receipts(id,batch_id,external_id,attempt,status,unresolved_fields_json,rejected_fields_json,reason,payload_json,payload_sha256,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, id, batchID, item.ExternalID, attempt, a.Status, string(fields), string(rejected), a.Reason, string(payload), digest(payload), now.Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE article_intake_items SET payload_json=?,status=?,unresolved_fields_json=?,rejected_fields_json=?,reason=?,current_receipt_id=? WHERE batch_id=? AND external_id=? AND status='unresolved'`, string(payload), a.Status, string(fields), string(rejected), a.Reason, id, batchID, item.ExternalID)
	return err
}

type batchRef struct{ id, fingerprint string }

func (s *service) findBatch(ctx context.Context, key string) (batchRef, bool, error) {
	var r batchRef
	err := s.db.QueryRowContext(ctx, `SELECT id,request_fingerprint FROM article_intake_batches WHERE idempotency_key=?`, key).Scan(&r.id, &r.fingerprint)
	if err == sql.ErrNoRows {
		return batchRef{}, false, nil
	}
	return r, err == nil, err
}

func (s *service) findSourceBatch(ctx context.Context, key string) (batchRef, bool, error) {
	var r batchRef
	err := s.db.QueryRowContext(ctx, `SELECT id,request_fingerprint FROM article_intake_batches WHERE source_key=?`, key).Scan(&r.id, &r.fingerprint)
	if err == sql.ErrNoRows {
		return batchRef{}, false, nil
	}
	return r, err == nil, err
}
func (s *service) recoverAdmission(ctx context.Context, req Request, fp string, cause error) ([]StoredItem, error) {
	r, found, err := s.findBatch(ctx, req.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if !found {
		sourceKey, e := sourceFingerprint(req.Source)
		if e != nil {
			return nil, e
		}
		r, found, e = s.findSourceBatch(ctx, sourceKey)
		if e != nil {
			return nil, e
		}
		if !found {
			return nil, cause
		}
		if r.fingerprint != fp {
			return nil, errors.New("source revision already has an intake batch; reconcile unresolved items instead of importing it again")
		}
		return s.List(ctx, r.id)
	}
	if r.fingerprint != fp {
		return nil, errors.New("idempotency key was reused with a different article intake request")
	}
	return s.List(ctx, r.id)
}

func requestFingerprint(req Request) (string, error) {
	type fingerprintItem struct {
		ExternalID string
		Payload    json.RawMessage
	}
	items := make([]fingerprintItem, 0, len(req.Items))
	for _, i := range req.Items {
		b, e := itemPayload(i)
		if e != nil {
			return "", e
		}
		items = append(items, fingerprintItem{ExternalID: i.ExternalID, Payload: b})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ExternalID < items[j].ExternalID })
	basis, err := json.Marshal(struct {
		Source Source
		Items  []fingerprintItem
	}{req.Source, items})
	if err != nil {
		return "", err
	}
	return digest(basis), nil
}

func sourceFingerprint(source Source) (string, error) {
	b, err := json.Marshal(source)
	if err != nil {
		return "", err
	}
	return digest(b), nil
}
func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
