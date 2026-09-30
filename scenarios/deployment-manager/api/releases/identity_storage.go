package releases

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// CandidateRecord is the immutable candidate projection used by release
// authorization. The canonical JSON is retained so a future reader can
// reconstruct the exact identity without trusting display columns.
type CandidateRecord struct {
	ID                     string    `json:"candidate_id"`
	Candidate              Candidate `json:"candidate"`
	ArtifactManifestDigest string    `json:"artifact_manifest_digest"`
	CreatedAt              time.Time `json:"created_at"`
}

// DestinationRevisionRecord is the immutable, non-secret destination
// projection used by release authorization.
type DestinationRevisionRecord struct {
	ID        string              `json:"destination_revision_id"`
	Revision  DestinationRevision `json:"revision"`
	CreatedAt time.Time           `json:"created_at"`
}

// ReviewRecord adds lifecycle state to the immutable ReviewBinding identity.
// Revocation is represented explicitly and never inferred from a missing row.
type ReviewRecord struct {
	ID         string        `json:"review_id"`
	Binding    ReviewBinding `json:"binding"`
	Status     string        `json:"status"`
	ApprovedAt *time.Time    `json:"approved_at,omitempty"`
	RevokedAt  *time.Time    `json:"revoked_at,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
}

// IdentityRepository is the durable identity lookup seam required before a
// release can be started. Registration methods are intentionally separate
// from Handler so producers can publish immutable records before review.
type IdentityRepository interface {
	RegisterCandidate(context.Context, Candidate) (*CandidateRecord, error)
	GetCandidate(context.Context, string) (*CandidateRecord, error)
	RegisterDestinationRevision(context.Context, DestinationRevision) (*DestinationRevisionRecord, error)
	GetDestinationRevision(context.Context, string) (*DestinationRevisionRecord, error)
	RegisterReview(context.Context, string, ReviewBinding, string, *time.Time) (*ReviewRecord, error)
	GetReview(context.Context, string) (*ReviewRecord, error)
}

// PublicationReceiptRepository stores producer-attributed external effects
// independently of the mutable release standing. A successful status never
// replaces these receipts.
type PublicationReceiptRepository interface {
	RecordPublicationReceipt(context.Context, string, PublicationReceipt) error
	ListPublicationReceipts(context.Context, string) ([]PublicationReceipt, error)
}

func (r *SQLRepository) RecordClientUpdateReceipt(ctx context.Context, releaseID string, receipt ClientUpdateReceipt) error {
	if !receipt.TrustedUpdate() {
		return fmt.Errorf("client update receipt is incomplete or not successful")
	}
	if strings.TrimSpace(releaseID) == "" {
		return fmt.Errorf("release id is required for client update receipt")
	}
	if err := r.validateReleaseReceiptIdentity(ctx, releaseID, receipt.CandidateID, "", receipt.TargetID); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO release_client_update_receipts
			(receipt_id, release_id, candidate_id, predecessor_ref, successor_digest, target_id,
			 verified_version, outcome, producer, external_receipt, observed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (release_id, target_id, external_receipt) DO NOTHING
	`, fmt.Sprintf("update-%s-%s-%s", releaseID, receipt.TargetID, receipt.ExternalReceipt), releaseID,
		receipt.CandidateID, receipt.PredecessorRef, receipt.SuccessorDigest, receipt.TargetID,
		receipt.VerifiedVersion, receipt.Outcome, receipt.Producer, receipt.ExternalReceipt, receipt.ObservedAt)
	if err != nil {
		return err
	}
	var stored ClientUpdateReceipt
	err = r.db.QueryRowContext(ctx, `
		SELECT candidate_id, predecessor_ref, successor_digest, target_id, verified_version,
		       outcome, producer, external_receipt, observed_at
		FROM release_client_update_receipts
		WHERE release_id = $1 AND target_id = $2 AND external_receipt = $3
	`, releaseID, receipt.TargetID, receipt.ExternalReceipt).Scan(
		&stored.CandidateID, &stored.PredecessorRef, &stored.SuccessorDigest, &stored.TargetID,
		&stored.VerifiedVersion, &stored.Outcome, &stored.Producer, &stored.ExternalReceipt, &stored.ObservedAt)
	if err != nil {
		return fmt.Errorf("load stored client update receipt: %w", err)
	}
	if sameClientUpdateReceipt(stored, receipt) {
		return nil
	}
	return fmt.Errorf("conflicting client update receipt for release %q target %q external receipt %q", releaseID, receipt.TargetID, receipt.ExternalReceipt)
}

func (r *SQLRepository) ListClientUpdateReceipts(ctx context.Context, releaseID string) ([]ClientUpdateReceipt, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT candidate_id, predecessor_ref, successor_digest, target_id, verified_version,
		       outcome, producer, external_receipt, observed_at
		FROM release_client_update_receipts
		WHERE release_id = $1 ORDER BY observed_at, target_id
	`, releaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var receipts []ClientUpdateReceipt
	for rows.Next() {
		var receipt ClientUpdateReceipt
		if err := rows.Scan(&receipt.CandidateID, &receipt.PredecessorRef, &receipt.SuccessorDigest, &receipt.TargetID, &receipt.VerifiedVersion, &receipt.Outcome, &receipt.Producer, &receipt.ExternalReceipt, &receipt.ObservedAt); err != nil {
			return nil, err
		}
		receipts = append(receipts, receipt)
	}
	return receipts, rows.Err()
}

func (r *SQLRepository) RecordPublicationReceipt(ctx context.Context, releaseID string, receipt PublicationReceipt) error {
	if !receipt.TrustedPublication() {
		return fmt.Errorf("publication receipt is incomplete or not successful")
	}
	if strings.TrimSpace(releaseID) == "" {
		return fmt.Errorf("release id is required for publication receipt")
	}
	if err := r.validateReleaseReceiptIdentity(ctx, releaseID, receipt.CandidateID, receipt.DestinationRevisionID, receipt.TargetID); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO release_publication_receipts
			(receipt_id, release_id, candidate_id, destination_revision_id, target_id, artifact_digest,
			 destination_object, producer, external_receipt, outcome, observed_at,
			 review_key, predecessor_artifact_digest, evidence_ref)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (receipt_id) DO NOTHING
	`, fmt.Sprintf("pub-%s-%s-%s", releaseID, receipt.TargetID, receipt.ExternalReceipt), releaseID,
		receipt.CandidateID, receipt.DestinationRevisionID, receipt.TargetID, receipt.ArtifactDigest,
		receipt.DestinationObject, receipt.Producer, receipt.ExternalReceipt, receipt.Outcome, receipt.ObservedAt,
		nullableReceiptString(receipt.ReviewKey), nullableReceiptString(receipt.PredecessorArtifactDigest), nullableReceiptString(receipt.EvidenceRef))
	if err != nil {
		return err
	}
	var stored PublicationReceipt
	var reviewKey, predecessor, evidenceRef sql.NullString
	err = r.db.QueryRowContext(ctx, `
		SELECT candidate_id, destination_revision_id, target_id, artifact_digest,
		       destination_object, producer, external_receipt, outcome, observed_at,
		       review_key, predecessor_artifact_digest, evidence_ref
		FROM release_publication_receipts
		WHERE release_id = $1 AND target_id = $2 AND external_receipt = $3
	`, releaseID, receipt.TargetID, receipt.ExternalReceipt).Scan(
		&stored.CandidateID, &stored.DestinationRevisionID, &stored.TargetID, &stored.ArtifactDigest,
		&stored.DestinationObject, &stored.Producer, &stored.ExternalReceipt, &stored.Outcome, &stored.ObservedAt,
		&reviewKey, &predecessor, &evidenceRef)
	stored.ReviewKey, stored.PredecessorArtifactDigest, stored.EvidenceRef = reviewKey.String, predecessor.String, evidenceRef.String
	if err != nil {
		return fmt.Errorf("load stored publication receipt: %w", err)
	}
	if samePublicationReceipt(stored, receipt) {
		return nil
	}
	return fmt.Errorf("conflicting publication receipt for release %q target %q external receipt %q", releaseID, receipt.TargetID, receipt.ExternalReceipt)
}

func sameClientUpdateReceipt(left, right ClientUpdateReceipt) bool {
	return left.CandidateID == right.CandidateID && left.PredecessorRef == right.PredecessorRef &&
		left.SuccessorDigest == right.SuccessorDigest && left.TargetID == right.TargetID &&
		left.VerifiedVersion == right.VerifiedVersion && left.Outcome == right.Outcome &&
		left.Producer == right.Producer && left.ExternalReceipt == right.ExternalReceipt &&
		left.ObservedAt.Equal(right.ObservedAt)
}

func samePublicationReceipt(left, right PublicationReceipt) bool {
	return left.CandidateID == right.CandidateID && left.DestinationRevisionID == right.DestinationRevisionID &&
		left.TargetID == right.TargetID && left.ArtifactDigest == right.ArtifactDigest &&
		left.DestinationObject == right.DestinationObject && left.Producer == right.Producer &&
		left.ExternalReceipt == right.ExternalReceipt && left.Outcome == right.Outcome &&
		left.ObservedAt.Equal(right.ObservedAt)
}

// validateReleaseReceiptIdentity binds producer evidence to the immutable
// identities recorded when the release was created. Receipt completeness is
// not sufficient: a valid receipt for another candidate or destination must
// never be accepted for this release.
func (r *SQLRepository) validateReleaseReceiptIdentity(ctx context.Context, releaseID, candidateID, destinationRevisionID, targetID string) error {
	var storedCandidateID, storedDestinationRevisionID sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT candidate_id, destination_revision_id
		FROM releases WHERE id = $1
	`, strings.TrimSpace(releaseID)).Scan(&storedCandidateID, &storedDestinationRevisionID)
	if err != nil {
		return fmt.Errorf("load release %q for receipt identity: %w", releaseID, err)
	}
	if !storedCandidateID.Valid || strings.TrimSpace(storedCandidateID.String) == "" {
		return fmt.Errorf("release %q has no registered candidate identity", releaseID)
	}
	if strings.TrimSpace(candidateID) != strings.TrimSpace(storedCandidateID.String) {
		return fmt.Errorf("receipt candidate %q does not match release %q candidate %q", candidateID, releaseID, storedCandidateID.String)
	}
	if destinationRevisionID != "" {
		if !storedDestinationRevisionID.Valid || strings.TrimSpace(storedDestinationRevisionID.String) == "" {
			return fmt.Errorf("release %q has no registered destination identity", releaseID)
		}
		if strings.TrimSpace(destinationRevisionID) != strings.TrimSpace(storedDestinationRevisionID.String) {
			return fmt.Errorf("receipt destination %q does not match release %q destination %q", destinationRevisionID, releaseID, storedDestinationRevisionID.String)
		}
	}
	if strings.TrimSpace(targetID) != "" {
		var targetCount int
		if err := r.db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM release_platforms
			WHERE release_id = $1 AND platform = $2
		`, strings.TrimSpace(releaseID), strings.TrimSpace(targetID)).Scan(&targetCount); err != nil {
			return fmt.Errorf("load release target %q for receipt identity: %w", targetID, err)
		}
		if targetCount != 1 {
			return fmt.Errorf("receipt target %q is not part of release %q", targetID, releaseID)
		}
	}
	return nil
}

func (r *SQLRepository) ListPublicationReceipts(ctx context.Context, releaseID string) ([]PublicationReceipt, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT candidate_id, destination_revision_id, target_id, artifact_digest,
		       destination_object, producer, external_receipt, outcome, observed_at,
		       review_key, predecessor_artifact_digest, evidence_ref
		FROM release_publication_receipts
		WHERE release_id = $1 ORDER BY target_id
	`, releaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var receipts []PublicationReceipt
	for rows.Next() {
		var receipt PublicationReceipt
		var reviewKey, predecessor, evidenceRef sql.NullString
		if err := rows.Scan(&receipt.CandidateID, &receipt.DestinationRevisionID, &receipt.TargetID, &receipt.ArtifactDigest, &receipt.DestinationObject, &receipt.Producer, &receipt.ExternalReceipt, &receipt.Outcome, &receipt.ObservedAt, &reviewKey, &predecessor, &evidenceRef); err != nil {
			return nil, err
		}
		receipt.ReviewKey, receipt.PredecessorArtifactDigest, receipt.EvidenceRef = reviewKey.String, predecessor.String, evidenceRef.String
		receipts = append(receipts, receipt)
	}
	return receipts, rows.Err()
}

func (r *SQLRepository) RegisterCandidate(ctx context.Context, candidate Candidate) (*CandidateRecord, error) {
	canonical, err := candidate.Canonical()
	if err != nil {
		return nil, err
	}
	id, err := canonical.Identity()
	if err != nil {
		return nil, err
	}
	manifestDigest, err := canonical.ArtifactManifestDigest()
	if err != nil {
		return nil, err
	}
	payload, err := canonical.CanonicalJSON()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO release_candidates
			(candidate_id, canonical_json, artifact_manifest_digest, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (candidate_id) DO NOTHING
	`, id, string(payload), manifestDigest, now); err != nil {
		return nil, fmt.Errorf("register candidate: %w", err)
	}
	stored, err := r.GetCandidate(ctx, id)
	if err != nil {
		return nil, err
	}
	if stored.ArtifactManifestDigest != manifestDigest {
		return nil, fmt.Errorf("candidate %q has conflicting artifact manifest", id)
	}
	return stored, nil
}

func (r *SQLRepository) GetCandidate(ctx context.Context, id string) (*CandidateRecord, error) {
	var raw string
	var record CandidateRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT candidate_id, canonical_json, artifact_manifest_digest, created_at
		FROM release_candidates WHERE candidate_id = $1
	`, strings.TrimSpace(id)).Scan(&record.ID, &raw, &record.ArtifactManifestDigest, &record.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(raw), &record.Candidate); err != nil {
		return nil, fmt.Errorf("decode candidate %q: %w", id, err)
	}
	canonical, err := record.Candidate.Canonical()
	if err != nil {
		return nil, fmt.Errorf("stored candidate %q is invalid: %w", id, err)
	}
	computedID, err := canonical.Identity()
	if err != nil || computedID != record.ID {
		return nil, fmt.Errorf("stored candidate %q failed identity verification", id)
	}
	computedDigest, err := canonical.ArtifactManifestDigest()
	if err != nil || computedDigest != record.ArtifactManifestDigest {
		return nil, fmt.Errorf("stored candidate %q failed artifact verification", id)
	}
	record.Candidate = canonical
	return &record, nil
}

func (r *SQLRepository) RegisterDestinationRevision(ctx context.Context, revision DestinationRevision) (*DestinationRevisionRecord, error) {
	canonical, err := revision.Canonical()
	if err != nil {
		return nil, err
	}
	id, err := canonical.Identity()
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO release_destination_revisions
			(destination_revision_id, canonical_json, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (destination_revision_id) DO NOTHING
	`, id, string(payload), now); err != nil {
		return nil, fmt.Errorf("register destination revision: %w", err)
	}
	return r.GetDestinationRevision(ctx, id)
}

func (r *SQLRepository) GetDestinationRevision(ctx context.Context, id string) (*DestinationRevisionRecord, error) {
	var raw string
	var record DestinationRevisionRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT destination_revision_id, canonical_json, created_at
		FROM release_destination_revisions WHERE destination_revision_id = $1
	`, strings.TrimSpace(id)).Scan(&record.ID, &raw, &record.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(raw), &record.Revision); err != nil {
		return nil, fmt.Errorf("decode destination revision %q: %w", id, err)
	}
	canonical, err := record.Revision.Canonical()
	if err != nil {
		return nil, fmt.Errorf("stored destination revision %q is invalid: %w", id, err)
	}
	computedID, err := canonical.Identity()
	if err != nil || computedID != record.ID {
		return nil, fmt.Errorf("stored destination revision %q failed identity verification", id)
	}
	record.Revision = canonical
	return &record, nil
}

func (r *SQLRepository) RegisterReview(ctx context.Context, id string, binding ReviewBinding, status string, approvedAt *time.Time) (*ReviewRecord, error) {
	binding, err := binding.Canonical()
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	status = strings.TrimSpace(status)
	if id == "" || status == "" {
		return nil, fmt.Errorf("review id and status are required")
	}
	if _, err := r.GetCandidate(ctx, binding.CandidateID); err != nil {
		return nil, fmt.Errorf("register review candidate: %w", err)
	}
	if _, err := r.GetDestinationRevision(ctx, binding.DestinationRevisionID); err != nil {
		return nil, fmt.Errorf("register review destination: %w", err)
	}
	payload, err := json.Marshal(binding)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO release_review_bindings
			(review_id, canonical_json, candidate_id, destination_revision_id, status, approved_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (review_id) DO NOTHING
	`, id, string(payload), binding.CandidateID, binding.DestinationRevisionID, status, approvedAt, now); err != nil {
		return nil, fmt.Errorf("register review: %w", err)
	}
	stored, err := r.GetReview(ctx, id)
	if err != nil {
		return nil, err
	}
	storedBindingID, err := stored.Binding.Identity()
	if err != nil {
		return nil, fmt.Errorf("derive stored review binding identity: %w", err)
	}
	bindingID, err := binding.Identity()
	if err != nil {
		return nil, fmt.Errorf("derive review binding identity: %w", err)
	}
	if storedBindingID != bindingID || stored.Status != status || !sameApprovalTime(stored.ApprovedAt, approvedAt) {
		return nil, fmt.Errorf("review %q already exists with a different immutable binding or lifecycle state", id)
	}
	return stored, nil
}

func sameApprovalTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func (r *SQLRepository) GetReview(ctx context.Context, id string) (*ReviewRecord, error) {
	var raw, status string
	var approvedAt, revokedAt sql.NullTime
	var record ReviewRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT review_id, canonical_json, status, approved_at, revoked_at, created_at
		FROM release_review_bindings WHERE review_id = $1
	`, strings.TrimSpace(id)).Scan(&record.ID, &raw, &status, &approvedAt, &revokedAt, &record.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(raw), &record.Binding); err != nil {
		return nil, fmt.Errorf("decode review %q: %w", id, err)
	}
	binding, err := record.Binding.Canonical()
	if err != nil {
		return nil, fmt.Errorf("stored review %q is invalid: %w", id, err)
	}
	record.Binding, record.Status = binding, status
	if approvedAt.Valid {
		value := approvedAt.Time
		record.ApprovedAt = &value
	}
	if revokedAt.Valid {
		value := revokedAt.Time
		record.RevokedAt = &value
	}
	return &record, nil
}

func nullableReceiptString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
