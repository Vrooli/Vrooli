package evidence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type dbExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type sqliteRepository struct {
	db  dbExecutor
	now func() time.Time
}

func NewSQLiteRepository(db dbExecutor, now func() time.Time) Repository {
	if now == nil {
		now = time.Now
	}
	return &sqliteRepository{db: db, now: now}
}

func (r *sqliteRepository) CreateReceipt(ctx context.Context, in NewObservation) (Receipt, error) {
	if in.Content == nil {
		in.Content = []byte{}
	}
	if err := in.Validate(); err != nil {
		return Receipt{}, err
	}
	if in.RetrievedAt.IsZero() {
		in.RetrievedAt = r.now()
	}
	hash := ContentHash(in.Content)
	artifactID := uuid.NewString()
	redirectJSON, err := json.Marshal(in.RedirectURLs)
	if err != nil {
		return Receipt{}, fmt.Errorf("marshal redirect chain: %w", err)
	}
	receipt := Receipt{ReceiptID: uuid.NewString(), ObservationID: uuid.NewString(), ProducerExecutionID: in.ProducerExecutionID, URL: in.URL, FinalURL: in.FinalURL, RedirectURLs: append([]string(nil), in.RedirectURLs...), RetrievedAt: in.RetrievedAt.UTC(), ContentHash: hash, ArtifactID: artifactID, ExtractionRevision: in.ExtractionRevision, Retention: in.Retention, FailureCode: in.FailureCode, ETag: in.ETag, LastModified: in.LastModified}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Receipt{}, fmt.Errorf("begin evidence receipt: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var existingArtifact string
	err = tx.QueryRowContext(ctx, `SELECT artifact_id FROM evidence_artifacts WHERE content_hash = ?`, hash).Scan(&existingArtifact)
	switch {
	case err == nil:
		// Content-addressed storage is shared, while the receipt below remains a
		// distinct observation with its own URL and retrieval timestamp.
		artifactID = existingArtifact
	case errors.Is(err, sql.ErrNoRows):
		_, err = tx.ExecContext(ctx, `INSERT INTO evidence_artifacts (artifact_id, content_hash, content, created_at) VALUES (?, ?, ?, ?)`, artifactID, hash, in.Content, r.now().UTC().Format(time.RFC3339Nano))
		if err != nil {
			return Receipt{}, fmt.Errorf("insert evidence artifact: %w", err)
		}
	default:
		return Receipt{}, fmt.Errorf("find evidence artifact: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO evidence_receipts (receipt_id, observation_id, producer_execution_id, url, final_url, redirect_urls, retrieved_at, content_hash, artifact_id, extraction_revision, retention, failure_code, etag, last_modified) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, receipt.ReceiptID, receipt.ObservationID, receipt.ProducerExecutionID, receipt.URL, receipt.FinalURL, string(redirectJSON), receipt.RetrievedAt.Format(time.RFC3339Nano), receipt.ContentHash, receipt.ArtifactID, receipt.ExtractionRevision, receipt.Retention, receipt.FailureCode, receipt.ETag, receipt.LastModified)
	if err != nil {
		return Receipt{}, fmt.Errorf("insert evidence receipt: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return Receipt{}, fmt.Errorf("commit evidence receipt: %w", err)
	}
	return receipt, nil
}

func (r *sqliteRepository) GetReceipt(ctx context.Context, id string) (Receipt, error) {
	var out Receipt
	var retrieved string
	var redirects string
	err := r.db.QueryRowContext(ctx, `SELECT receipt_id, observation_id, producer_execution_id, url, final_url, redirect_urls, retrieved_at, content_hash, artifact_id, extraction_revision, retention, failure_code, etag, last_modified FROM evidence_receipts WHERE receipt_id = ?`, id).Scan(&out.ReceiptID, &out.ObservationID, &out.ProducerExecutionID, &out.URL, &out.FinalURL, &redirects, &retrieved, &out.ContentHash, &out.ArtifactID, &out.ExtractionRevision, &out.Retention, &out.FailureCode, &out.ETag, &out.LastModified)
	if errors.Is(err, sql.ErrNoRows) {
		return Receipt{}, fmt.Errorf("evidence receipt %q not found", id)
	}
	if err != nil {
		return Receipt{}, err
	}
	if redirects != "" {
		if err := json.Unmarshal([]byte(redirects), &out.RedirectURLs); err != nil {
			return Receipt{}, fmt.Errorf("invalid redirect chain: %w", err)
		}
	}
	out.RetrievedAt, err = time.Parse(time.RFC3339Nano, retrieved)
	if err != nil {
		return Receipt{}, fmt.Errorf("invalid receipt timestamp: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) CreatePassage(ctx context.Context, receiptID string, start, end int) (Passage, error) {
	receipt, err := r.GetReceipt(ctx, receiptID)
	if err != nil {
		return Passage{}, err
	}
	var content []byte
	err = r.db.QueryRowContext(ctx, `SELECT content FROM evidence_artifacts WHERE artifact_id = ? AND content_hash = ?`, receipt.ArtifactID, receipt.ContentHash).Scan(&content)
	if err != nil {
		return Passage{}, fmt.Errorf("read evidence artifact: %w", err)
	}
	if content == nil {
		return Passage{}, fmt.Errorf("evidence content unavailable after retention expiry")
	}
	if ContentHash(content) != receipt.ContentHash {
		return Passage{}, fmt.Errorf("evidence artifact integrity failure")
	}
	if err = validateSpan(content, start, end); err != nil {
		return Passage{}, err
	}
	passage := Passage{PassageID: uuid.NewString(), ReceiptID: receiptID, StartByte: start, EndByte: end, Content: string(content[start:end]), Hash: ContentHash(content[start:end])}
	_, err = r.db.ExecContext(ctx, `INSERT INTO evidence_passages (passage_id, receipt_id, start_byte, end_byte, passage_hash) VALUES (?, ?, ?, ?, ?)`, passage.PassageID, passage.ReceiptID, start, end, passage.Hash)
	if err != nil {
		return Passage{}, fmt.Errorf("insert evidence passage: %w", err)
	}
	return passage, nil
}

func (r *sqliteRepository) CreateAssessment(ctx context.Context, in Assessment) (Assessment, error) {
	if strings.TrimSpace(in.ClaimID) == "" || strings.TrimSpace(in.Disposition) == "" {
		return Assessment{}, fmt.Errorf("assessment claim and disposition are required")
	}
	if in.AssessmentID == "" {
		in.AssessmentID = uuid.NewString()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO evidence_assessments (assessment_id, claim_id, disposition, policy_revision, reason, evidence_json, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, in.AssessmentID, in.ClaimID, in.Disposition, in.PolicyRevision, in.Reason, in.EvidenceJSON, r.now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		var existing Assessment
		existingErr := r.db.QueryRowContext(ctx, `SELECT assessment_id, claim_id, disposition, policy_revision, reason, evidence_json FROM evidence_assessments WHERE assessment_id = ?`, in.AssessmentID).Scan(&existing.AssessmentID, &existing.ClaimID, &existing.Disposition, &existing.PolicyRevision, &existing.Reason, &existing.EvidenceJSON)
		if existingErr == nil {
			return existing, nil
		}
		return Assessment{}, fmt.Errorf("insert evidence assessment: %w", err)
	}
	return in, nil
}

func (r *sqliteRepository) GetAssessment(ctx context.Context, id string) (Assessment, error) {
	var out Assessment
	err := r.db.QueryRowContext(ctx, `SELECT assessment_id, claim_id, disposition, policy_revision, reason, evidence_json FROM evidence_assessments WHERE assessment_id = ?`, id).Scan(&out.AssessmentID, &out.ClaimID, &out.Disposition, &out.PolicyRevision, &out.Reason, &out.EvidenceJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return Assessment{}, fmt.Errorf("evidence assessment %q not found", id)
	}
	if err != nil {
		return Assessment{}, fmt.Errorf("read evidence assessment: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) GetPassage(ctx context.Context, id string) (Passage, error) {
	var p Passage
	err := r.db.QueryRowContext(ctx, `SELECT passage_id, receipt_id, start_byte, end_byte, passage_hash FROM evidence_passages WHERE passage_id = ?`, id).Scan(&p.PassageID, &p.ReceiptID, &p.StartByte, &p.EndByte, &p.Hash)
	if errors.Is(err, sql.ErrNoRows) {
		return Passage{}, fmt.Errorf("evidence passage %q not found", id)
	}
	if err != nil {
		return Passage{}, err
	}
	receipt, err := r.GetReceipt(ctx, p.ReceiptID)
	if err != nil {
		return Passage{}, err
	}
	var content []byte
	err = r.db.QueryRowContext(ctx, `SELECT content FROM evidence_artifacts WHERE artifact_id = ?`, receipt.ArtifactID).Scan(&content)
	if err != nil {
		return Passage{}, fmt.Errorf("read evidence artifact: %w", err)
	}
	if content == nil {
		return Passage{}, fmt.Errorf("evidence content unavailable after retention expiry")
	}
	if ContentHash(content) != receipt.ContentHash || p.StartByte < 0 || p.EndByte > len(content) {
		return Passage{}, fmt.Errorf("evidence artifact integrity failure")
	}
	span := content[p.StartByte:p.EndByte]
	if err = validateSpan(content, p.StartByte, p.EndByte); err != nil || ContentHash(span) != p.Hash {
		return Passage{}, fmt.Errorf("evidence passage integrity failure")
	}
	p.Content = string(span)
	return p, nil
}

func (r *sqliteRepository) ExpireContent(ctx context.Context, before time.Time) (int, error) {
	return r.ExpireContentExcept(ctx, before, nil)
}

func (r *sqliteRepository) PreviewContentExpiry(ctx context.Context, before time.Time, protected []string) (RetentionPreview, error) {
	query, args := expiryQuery("SELECT COUNT(*), COALESCE(SUM(length(content)), 0) FROM evidence_artifacts WHERE created_at < ? AND content IS NOT NULL", before, protected)
	var out RetentionPreview
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&out.ArtifactCount, &out.ContentBytes)
	if err != nil {
		return RetentionPreview{}, fmt.Errorf("preview evidence expiry: %w", err)
	}
	out.Before = before.UTC()
	return out, nil
}

func (r *sqliteRepository) ExpireContentExcept(ctx context.Context, before time.Time, protected []string) (int, error) {
	query, args := expiryQuery("UPDATE evidence_artifacts SET content = NULL WHERE created_at < ? AND content IS NOT NULL", before, protected)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("expire evidence content: %w", err)
	}
	count, err := result.RowsAffected()
	return int(count), err
}

func expiryQuery(base string, before time.Time, protected []string) (string, []any) {
	args := []any{before.UTC().Format(time.RFC3339Nano)}
	if len(protected) == 0 {
		return base, args
	}
	placeholders := make([]string, len(protected))
	for i, id := range protected {
		placeholders[i], args = "?", append(args, id)
	}
	return base + " AND artifact_id NOT IN (" + strings.Join(placeholders, ",") + ")", args
}
