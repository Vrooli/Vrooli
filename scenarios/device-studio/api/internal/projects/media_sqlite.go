package projects

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

const sourceMediaColumns = `id,project_id,sha256,size_bytes,media_type,kind,readiness,width,height,orientation,color_model,color_profile_status,inspection_job_id,inspection_attempt_id,inspection_issue_code,inspection_issue_message,inspection_receipt_json,inspection_receipt_key,original_filename,imported_at,import_method,decoder,storage_key`

func (r *sqliteRepository) CreateSourceMedia(ctx context.Context, media SourceMedia) (SourceMedia, bool, error) {
	q := `INSERT INTO project_source_media(` + sourceMediaColumns + `) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(project_id,sha256) DO NOTHING RETURNING ` + sourceMediaColumns
	row, err := scanSourceMedia(r.db.QueryRowContext(ctx, q, mediaArgs(media)...))
	if err == nil {
		return row, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return SourceMedia{}, false, fmt.Errorf("insert source media: %w", err)
	}
	row, err = r.GetSourceMediaByDigest(ctx, media.ProjectID, media.SHA256)
	return row, false, err
}

func mediaArgs(m SourceMedia) []any {
	var width, height any
	if m.RasterKnown {
		width, height = m.Width, m.Height
	}
	var receipt string
	if m.InspectionReceipt != nil {
		b, _ := json.Marshal(m.InspectionReceipt)
		receipt = string(b)
	}
	return []any{m.ID, m.ProjectID, m.SHA256, m.SizeBytes, m.MediaType, string(m.Kind), string(m.Readiness), width, height, nullableText(m.Orientation, m.OrientationKnown), nullableText(m.ColorModel, m.ColorModelKnown), nullableText(m.ColorProfileStatus, m.ColorProfileKnown), m.InspectionJobID, m.InspectionAttemptID, m.InspectionIssueCode, m.InspectionIssueMessage, receipt, m.inspectionReceiptKey, m.OriginalFilename, m.ImportedAt.UTC().Format(timeFormat), m.ImportMethod, m.Decoder, m.storageKey}
}

func nullableText(v string, known bool) any {
	if !known {
		return nil
	}
	return v
}

func (r *sqliteRepository) GetSourceMedia(ctx context.Context, projectID, id string) (SourceMedia, error) {
	m, err := scanSourceMedia(r.db.QueryRowContext(ctx, `SELECT `+sourceMediaColumns+` FROM project_source_media WHERE project_id=? AND id=?`, projectID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return SourceMedia{}, ErrSourceMediaNotFound{ID: id}
	}
	if err != nil {
		return SourceMedia{}, fmt.Errorf("get source media: %w", err)
	}
	return m, nil
}

func (r *sqliteRepository) GetSourceMediaByDigest(ctx context.Context, projectID, digest string) (SourceMedia, error) {
	m, err := scanSourceMedia(r.db.QueryRowContext(ctx, `SELECT `+sourceMediaColumns+` FROM project_source_media WHERE project_id=? AND sha256=?`, projectID, digest))
	if errors.Is(err, sql.ErrNoRows) {
		return SourceMedia{}, ErrSourceMediaNotFound{ID: digest}
	}
	if err != nil {
		return SourceMedia{}, fmt.Errorf("get source media by digest: %w", err)
	}
	return m, nil
}

func (r *sqliteRepository) ListSourceMedia(ctx context.Context, projectID string, limit int, after *SourceMedia) ([]SourceMedia, error) {
	q := `SELECT ` + sourceMediaColumns + ` FROM project_source_media WHERE project_id=? ORDER BY imported_at DESC,id DESC LIMIT ?`
	args := []any{projectID, limit}
	if after != nil {
		q = `SELECT ` + sourceMediaColumns + ` FROM project_source_media WHERE project_id=? AND (imported_at<? OR (imported_at=? AND id<?)) ORDER BY imported_at DESC,id DESC LIMIT ?`
		t := after.ImportedAt.UTC().Format(timeFormat)
		args = []any{projectID, t, t, after.ID, limit}
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list source media: %w", err)
	}
	defer rows.Close()
	out := make([]SourceMedia, 0)
	for rows.Next() {
		m, e := scanSourceMedia(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate source media: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) BindInspectionJob(ctx context.Context, projectID, mediaID, sourceDigest, jobID string) (SourceMedia, error) {
	if !validPathComponent(strings.TrimSpace(jobID)) || !validSourceDigest(sourceDigest) {
		return SourceMedia{}, ErrInspectionInvalid
	}
	m, err := r.GetSourceMedia(ctx, projectID, mediaID)
	if err != nil {
		return SourceMedia{}, err
	}
	if m.SHA256 != sourceDigest || m.Kind != MediaKindVideo {
		return SourceMedia{}, ErrInspectionConflict
	}
	if m.InspectionJobID == jobID {
		return m, nil
	}
	if m.InspectionJobID != "" || m.Readiness != ReadinessPending {
		return SourceMedia{}, ErrInspectionConflict
	}
	result, err := r.db.ExecContext(ctx, `UPDATE project_source_media SET inspection_job_id=? WHERE project_id=? AND id=? AND sha256=? AND inspection_job_id='' AND readiness='pending'`, jobID, projectID, mediaID, sourceDigest)
	if err != nil {
		return SourceMedia{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return SourceMedia{}, err
	}
	if n != 1 {
		return SourceMedia{}, ErrInspectionConflict
	}
	m.InspectionJobID = jobID
	return m, nil
}

func (r *sqliteRepository) BindInspectionAttempt(ctx context.Context, projectID, mediaID, sourceDigest, jobID, expectedAttempt, attemptID string) (SourceMedia, error) {
	if !validPathComponent(strings.TrimSpace(jobID)) || !validPathComponent(strings.TrimSpace(attemptID)) || !validSourceDigest(sourceDigest) {
		return SourceMedia{}, ErrInspectionInvalid
	}
	query := `UPDATE project_source_media SET inspection_job_id=?,inspection_attempt_id=?,readiness='pending',inspection_issue_code='',inspection_issue_message='',inspection_receipt_json='',inspection_receipt_key='',width=NULL,height=NULL,orientation=NULL WHERE project_id=? AND id=? AND sha256=? AND kind='video' AND inspection_job_id='' AND inspection_attempt_id='' AND readiness='pending'`
	args := []any{jobID, attemptID, projectID, mediaID, sourceDigest}
	if expectedAttempt != "" {
		query = `UPDATE project_source_media SET inspection_attempt_id=?,readiness='pending',inspection_issue_code='',inspection_issue_message='',inspection_receipt_json='',inspection_receipt_key='',width=NULL,height=NULL,orientation=NULL WHERE project_id=? AND id=? AND sha256=? AND kind='video' AND inspection_job_id=? AND inspection_attempt_id=? AND (readiness='reconciliation_required' OR (readiness='rejected' AND inspection_receipt_json='' AND inspection_issue_code IN ('inspection_execution_failed','inspection_execution_cancelled')))`
		args = []any{attemptID, projectID, mediaID, sourceDigest, jobID, expectedAttempt}
	}
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return SourceMedia{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return SourceMedia{}, err
	}
	if n != 1 {
		m, getErr := r.GetSourceMedia(ctx, projectID, mediaID)
		if getErr != nil {
			return SourceMedia{}, getErr
		}
		if m.SHA256 == sourceDigest && m.InspectionJobID == jobID && m.InspectionAttemptID == attemptID {
			return m, nil
		}
		return SourceMedia{}, ErrInspectionConflict
	}
	return r.GetSourceMedia(ctx, projectID, mediaID)
}

func (r *sqliteRepository) ProjectInspection(ctx context.Context, p InspectionProjection) error {
	if p.Readiness == ReadinessReady && p.Receipt == nil {
		return ErrInspectionInvalid
	}
	m, err := r.GetSourceMedia(ctx, p.ProjectID, p.MediaID)
	if err != nil {
		return err
	}
	if m.SHA256 != p.SourceSHA256 || m.InspectionJobID != p.JobID || m.InspectionAttemptID != p.AttemptID {
		return ErrInspectionConflict
	}
	if m.Kind != MediaKindVideo {
		return ErrInspectionConflict
	}
	if p.Receipt != nil && p.Receipt.SourceSizeBytes != m.SizeBytes {
		return fmt.Errorf("%w: receipt source size differs from stored media", ErrInspectionInvalid)
	}
	if m.Readiness == p.Readiness && projectionMatches(m, p) {
		return nil
	}
	if m.Readiness != ReadinessPending && m.Readiness != ReadinessReconciliationRequired {
		return ErrInspectionConflict
	}
	var receipt string
	if p.Receipt != nil {
		b, e := json.Marshal(p.Receipt)
		if e != nil {
			return e
		}
		receipt = string(b)
	}
	var width, height, orientation any
	if p.Readiness == ReadinessReady {
		width, height = p.Receipt.VideoWidth, p.Receipt.VideoHeight
		orientation = fmt.Sprintf("rotation_%d", p.Receipt.RotationDegrees)
	}
	result, err := r.db.ExecContext(ctx, `UPDATE project_source_media SET readiness=?,inspection_issue_code=?,inspection_issue_message=?,inspection_receipt_json=?,inspection_receipt_key=?,width=?,height=?,orientation=? WHERE project_id=? AND id=? AND sha256=? AND inspection_job_id=? AND inspection_attempt_id=? AND readiness IN ('pending','reconciliation_required')`, string(p.Readiness), p.IssueCode, p.IssueMessage, receipt, p.ReceiptStorageKey, width, height, orientation, p.ProjectID, p.MediaID, p.SourceSHA256, p.JobID, p.AttemptID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrInspectionConflict
	}
	return nil
}

func projectionMatches(m SourceMedia, p InspectionProjection) bool {
	if m.InspectionIssueCode != p.IssueCode || m.InspectionIssueMessage != p.IssueMessage || m.inspectionReceiptKey != p.ReceiptStorageKey {
		return false
	}
	if m.InspectionReceipt == nil || p.Receipt == nil {
		return m.InspectionReceipt == nil && p.Receipt == nil
	}
	return reflect.DeepEqual(m.InspectionReceipt, p.Receipt)
}

type sourceMediaScanner interface{ Scan(...any) error }

func scanSourceMedia(row sourceMediaScanner) (SourceMedia, error) {
	var m SourceMedia
	var width, height sql.NullInt64
	var orientation, colorModel, profile sql.NullString
	var receipt, imported string
	err := row.Scan(&m.ID, &m.ProjectID, &m.SHA256, &m.SizeBytes, &m.MediaType, &m.Kind, &m.Readiness, &width, &height, &orientation, &colorModel, &profile, &m.InspectionJobID, &m.InspectionAttemptID, &m.InspectionIssueCode, &m.InspectionIssueMessage, &receipt, &m.inspectionReceiptKey, &m.OriginalFilename, &imported, &m.ImportMethod, &m.Decoder, &m.storageKey)
	if err != nil {
		return SourceMedia{}, err
	}
	if width.Valid != height.Valid {
		return SourceMedia{}, errors.New("source media raster dimensions are partially present")
	}
	if width.Valid {
		m.Width, m.Height, m.RasterKnown = uint32(width.Int64), uint32(height.Int64), true
	}
	m.Orientation, m.ColorModel, m.ColorProfileStatus = orientation.String, colorModel.String, profile.String
	m.OrientationKnown, m.ColorModelKnown, m.ColorProfileKnown = orientation.Valid, colorModel.Valid, profile.Valid
	if receipt != "" {
		var v InspectionReceipt
		if err := json.Unmarshal([]byte(receipt), &v); err != nil {
			return SourceMedia{}, fmt.Errorf("decode source inspection receipt: %w", err)
		}
		m.InspectionReceipt = &v
	}
	m.ImportedAt, err = time.Parse(parseTimeFormat, imported)
	if err != nil {
		return SourceMedia{}, fmt.Errorf("parse source media imported_at: %w", err)
	}
	return m, nil
}
