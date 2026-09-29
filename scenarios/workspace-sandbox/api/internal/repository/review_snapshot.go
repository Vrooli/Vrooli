package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"workspace-sandbox/internal/types"
)

func (r *SandboxArchiveRepository) CheckReviewCapacity(ctx context.Context) error {
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sandbox_review_snapshots`).Scan(&count); err != nil {
		return err
	}
	if count >= types.MaxReviewSnapshots {
		return errors.New("review snapshot retention capacity reached; release through the evidence owner before further capture")
	}
	return nil
}

func (r *SandboxArchiveRepository) PutReviewSnapshot(ctx context.Context, snapshot *types.ReviewSnapshot) error {
	if snapshot == nil || snapshot.SandboxID == uuid.Nil || snapshot.RequestID == uuid.Nil || snapshot.ID != types.ReviewSnapshotID(snapshot.SandboxID, snapshot.RequestID) || snapshot.SHA256 != snapshot.ContentSHA256() {
		return errors.New("invalid review snapshot identity")
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	// Capacity is checked in the insertion statement as well as before capture.
	// Same-key replay is read below and never replaces another published input.
	_, err = r.db.ExecContext(ctx, `INSERT INTO sandbox_review_snapshots(id, sandbox_id, request_id, snapshot_json)
	 SELECT ?, ?, ?, ? WHERE (SELECT COUNT(*) FROM sandbox_review_snapshots) < ?
	 ON CONFLICT(id) DO NOTHING`, uuidText(snapshot.ID), uuidText(snapshot.SandboxID), uuidText(snapshot.RequestID), string(raw), types.MaxReviewSnapshots)
	if err != nil {
		return err
	}
	var stored string
	err = r.db.QueryRowContext(ctx, `SELECT snapshot_json FROM sandbox_review_snapshots WHERE id = ?`, uuidText(snapshot.ID)).Scan(&stored)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("review snapshot retention capacity reached")
	}
	if err != nil {
		return err
	}
	if stored != string(raw) {
		return errors.New("review request already has different retained evidence")
	}
	return nil
}

func (r *SandboxArchiveRepository) GetReviewSnapshot(ctx context.Context, sandboxID, requestID uuid.UUID) (*types.ReviewSnapshot, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT snapshot_json FROM sandbox_review_snapshots WHERE sandbox_id = ? AND request_id = ?`, uuidText(sandboxID), uuidText(requestID)).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var snapshot types.ReviewSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return nil, fmt.Errorf("decode review snapshot: %w", err)
	}
	if snapshot.SandboxID != sandboxID || snapshot.RequestID != requestID || snapshot.ID != types.ReviewSnapshotID(sandboxID, requestID) || snapshot.SHA256 != snapshot.ContentSHA256() {
		return nil, errors.New("retained review snapshot identity is corrupt")
	}
	return &snapshot, nil
}
