package findings

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

func ClaimHash(claim string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(claim)))
	return hex.EncodeToString(sum[:])
}

func (s *sqliteRepository) RecordCorrection(ctx context.Context, in CorrectionInput, actor string) (Correction, error) {
	in.FindingID = strings.TrimSpace(in.FindingID)
	in.Identity = strings.TrimSpace(in.Identity)
	in.OriginalClaimHash = strings.TrimSpace(in.OriginalClaimHash)
	in.Disposition = strings.TrimSpace(in.Disposition)
	if in.FindingID == "" || in.Identity == "" || in.OriginalClaimHash == "" || in.Disposition == "" {
		return Correction{}, ErrInvalidFinding{Field: "correction", Reason: "finding_id, identity, original_claim_hash, and disposition are required"}
	}
	finding, err := s.Get(ctx, in.FindingID)
	if err != nil {
		return Correction{}, err
	}
	if ClaimHash(finding.Claim) != in.OriginalClaimHash {
		return Correction{}, ErrInvalidFinding{Field: "original_claim_hash", Reason: "does not match the finding claim"}
	}
	encode := func(v []string) (string, error) {
		if v == nil {
			v = []string{}
		}
		b, err := json.Marshal(v)
		return string(b), err
	}
	evidence, err := encode(in.EvidenceRefs)
	if err != nil {
		return Correction{}, fmt.Errorf("encode correction evidence: %w", err)
	}
	investigations, err := encode(in.InvestigationIDs)
	if err != nil {
		return Correction{}, fmt.Errorf("encode correction investigations: %w", err)
	}
	methods, err := encode(in.MethodRevisions)
	if err != nil {
		return Correction{}, fmt.Errorf("encode correction methods: %w", err)
	}

	var existing Correction
	row := s.db.QueryRowContext(ctx, `SELECT id, finding_id, identity, original_claim_hash, disposition, reason, evidence_refs, investigation_ids, method_revisions, actor, created_at FROM finding_corrections WHERE finding_id = ? AND identity = ?`, in.FindingID, in.Identity)
	if err := scanCorrection(row, &existing); err == nil {
		if !sameCorrectionBody(existing, in, evidence, investigations, methods) {
			return Correction{}, ErrInvalidFinding{Field: "identity", Reason: "correction replay has a different body"}
		}
		return existing, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Correction{}, fmt.Errorf("lookup correction: %w", err)
	}

	now := s.now()
	id := uuid.NewString()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Correction{}, fmt.Errorf("begin correction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `INSERT INTO finding_corrections (id, finding_id, identity, original_claim_hash, disposition, reason, evidence_refs, investigation_ids, method_revisions, actor, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, in.FindingID, in.Identity, in.OriginalClaimHash, in.Disposition, in.Reason, evidence, investigations, methods, actor, now.Format(findingTimeFormat))
	if err != nil {
		return Correction{}, fmt.Errorf("insert correction: %w", err)
	}
	if err := writeAudit(ctx, tx, now, in.FindingID, MutationCorrection, in.Reason, finding.BriefID, actor); err != nil {
		return Correction{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE findings SET status = ?, updated_at = ? WHERE id = ? AND status = ?`, StatusDisputed, now.Format(findingTimeFormat), in.FindingID, StatusActive); err != nil {
		return Correction{}, fmt.Errorf("invalidate finding reuse: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Correction{}, fmt.Errorf("commit correction: %w", err)
	}
	return Correction{ID: id, FindingID: in.FindingID, Identity: in.Identity, OriginalClaimHash: in.OriginalClaimHash, Disposition: in.Disposition, Reason: in.Reason, EvidenceRefs: in.EvidenceRefs, InvestigationIDs: in.InvestigationIDs, MethodRevisions: in.MethodRevisions, Actor: actor, CreatedAt: now}, nil
}

func sameCorrectionBody(existing Correction, in CorrectionInput, evidence, investigations, methods string) bool {
	actual := func(v []string) string { b, _ := json.Marshal(v); return string(b) }
	return existing.OriginalClaimHash == in.OriginalClaimHash && existing.Disposition == in.Disposition && existing.Reason == in.Reason && actual(existing.EvidenceRefs) == evidence && actual(existing.InvestigationIDs) == investigations && actual(existing.MethodRevisions) == methods
}

func scanCorrection(row interface{ Scan(...any) error }, out *Correction) error {
	var evidence, investigations, methods, created string
	if err := row.Scan(&out.ID, &out.FindingID, &out.Identity, &out.OriginalClaimHash, &out.Disposition, &out.Reason, &evidence, &investigations, &methods, &out.Actor, &created); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(evidence), &out.EvidenceRefs); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(investigations), &out.InvestigationIDs); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(methods), &out.MethodRevisions); err != nil {
		return err
	}
	var err error
	out.CreatedAt, err = time.Parse(findingTimeFormat, created)
	return err
}

func (s *sqliteRepository) ListCorrections(ctx context.Context, findingID string) ([]Correction, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, finding_id, identity, original_claim_hash, disposition, reason, evidence_refs, investigation_ids, method_revisions, actor, created_at FROM finding_corrections WHERE finding_id = ? ORDER BY created_at ASC, id ASC`, findingID)
	if err != nil {
		return nil, fmt.Errorf("list corrections: %w", err)
	}
	defer rows.Close()
	var result []Correction
	for rows.Next() {
		var c Correction
		if err := scanCorrection(rows, &c); err != nil {
			return nil, fmt.Errorf("scan correction: %w", err)
		}
		result = append(result, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate corrections: %w", err)
	}
	return result, nil
}
