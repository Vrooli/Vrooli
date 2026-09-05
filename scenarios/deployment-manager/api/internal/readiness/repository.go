package readiness

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	domain "deployment-manager/readiness"
	"deployment-manager/shared"
)

type SQLRepository struct {
	db          shared.RoutedDBTX
	placeholder func(int) string
}

func NewSQLRepository(db shared.RoutedDBTX, dialect string) *SQLRepository {
	p := func(i int) string { return fmt.Sprintf("$%d", i) }
	if strings.EqualFold(dialect, "sqlite") {
		p = func(int) string { return "?" }
	}
	return &SQLRepository{db: db, placeholder: p}
}

func (r *SQLRepository) CreateOrGet(ctx context.Context, review *domain.Review) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("readiness repository is not configured")
	}
	if review == nil {
		return false, errors.New("review is required")
	}
	identity, err := review.Identity.Canonical()
	if err != nil {
		return false, err
	}
	key, err := identity.Key()
	if err != nil {
		return false, err
	}
	if review.Key != "" && review.Key != key {
		return false, errors.New("review key does not match identity")
	}
	review.Key, review.Identity = key, identity
	if review.Status == "" {
		review.Status = domain.ReviewCollecting
	}
	if review.ComparisonMode == "" {
		review.ComparisonMode = domain.ComparisonFirstRelease
	}
	targets, err := json.Marshal(identity.Targets)
	if err != nil {
		return false, err
	}
	now := time.Now().UTC()
	p := r.placeholder
	result, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO readiness_reviews
		(review_key, scenario, profile_id, candidate_commit, artifact_digest, targets_json,
		 channel, policy_version, status, comparison_mode, predecessor_release_id,
		 predecessor_commit, predecessor_artifact_digest, created_at, updated_at)
		VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)
		ON CONFLICT (review_key) DO NOTHING`, p(1), p(2), p(3), p(4), p(5), p(6), p(7), p(8), p(9), p(10), p(11), p(12), p(13), p(14), p(15)),
		key, identity.Scenario, identity.ProfileID, identity.CandidateCommit, identity.ArtifactDigest, string(targets),
		identity.Channel, identity.PolicyVersion, review.Status, review.ComparisonMode,
		nullString(review.PredecessorReleaseID), nullString(review.PredecessorCommit), nullString(review.PredecessorArtifactDigest), now, now)
	if err != nil {
		return false, fmt.Errorf("insert readiness review: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if rows == 0 {
		existing, err := r.Get(ctx, key)
		if err != nil {
			return false, err
		}
		if !reflect.DeepEqual(existing.Identity, identity) {
			return false, errors.New("stored review identity does not match deterministic key")
		}
		*review = *existing
		return true, nil
	}
	review.CreatedAt, review.UpdatedAt = now, now
	return false, nil
}

func (r *SQLRepository) Get(ctx context.Context, key string) (*domain.Review, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("readiness repository is not configured")
	}
	p := r.placeholder
	var review domain.Review
	var targets string
	var predRelease, predCommit, predArtifact, goal, approvedBy sql.NullString
	var goalClosed, approved sql.NullTime
	err := r.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT review_key, scenario, profile_id, candidate_commit,
		artifact_digest, targets_json, channel, policy_version, status, comparison_mode,
		predecessor_release_id, predecessor_commit, predecessor_artifact_digest, goal_ref,
		goal_closed_at, approved_at, approved_by, created_at, updated_at
		FROM readiness_reviews WHERE review_key = %s`, p(1)), key).Scan(
		&review.Key, &review.Identity.Scenario, &review.Identity.ProfileID, &review.Identity.CandidateCommit,
		&review.Identity.ArtifactDigest, &targets, &review.Identity.Channel, &review.Identity.PolicyVersion,
		&review.Status, &review.ComparisonMode, &predRelease, &predCommit, &predArtifact, &goal,
		&goalClosed, &approved, &approvedBy, &review.CreatedAt, &review.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(targets), &review.Identity.Targets); err != nil {
		return nil, fmt.Errorf("decode review targets: %w", err)
	}
	review.PredecessorReleaseID, review.PredecessorCommit, review.PredecessorArtifactDigest = predRelease.String, predCommit.String, predArtifact.String
	review.GoalRef, review.ApprovedBy = goal.String, approvedBy.String
	if goalClosed.Valid {
		value := goalClosed.Time
		review.GoalClosedAt = &value
	}
	if approved.Valid {
		value := approved.Time
		review.ApprovedAt = &value
	}
	return &review, nil
}

func (r *SQLRepository) ListReviews(ctx context.Context, status domain.ReviewStatus, limit int) ([]domain.Review, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT review_key FROM readiness_reviews`
	args := []any{}
	if status != "" {
		query += fmt.Sprintf(" WHERE status = %s", r.placeholder(1))
		args = append(args, status)
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %s", r.placeholder(len(args)+1))
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]domain.Review, 0, len(keys))
	for _, key := range keys {
		review, err := r.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		result = append(result, *review)
	}
	return result, nil
}

func (r *SQLRepository) ListEvaluation(ctx context.Context, key string) ([]domain.EvidenceItem, []domain.ReviewFinding, error) {
	if r == nil || r.db == nil {
		return nil, nil, errors.New("readiness repository is not configured")
	}
	p := r.placeholder
	evidenceRows, err := r.db.QueryContext(ctx, fmt.Sprintf(`SELECT review_key, criterion_id, status,
		applicability, COALESCE(applicability_reason, ''), producer, COALESCE(producer_version, ''),
		candidate_commit, artifact_digest, target, environment, policy_version, observed_at,
		evidence_reference, COALESCE(detail, '') FROM readiness_evidence
		WHERE review_key = %s ORDER BY criterion_id`, p(1)), key)
	if err != nil {
		return nil, nil, err
	}
	defer evidenceRows.Close()
	var evidence []domain.EvidenceItem
	for evidenceRows.Next() {
		var item domain.EvidenceItem
		if err := evidenceRows.Scan(&item.ReviewKey, &item.CriterionID, &item.Status, &item.Applicability,
			&item.ApplicabilityReason, &item.Producer, &item.ProducerVersion, &item.CandidateCommit,
			&item.ArtifactDigest, &item.Target, &item.Environment, &item.PolicyVersion, &item.ObservedAt,
			&item.Reference, &item.Detail); err != nil {
			return nil, nil, err
		}
		evidence = append(evidence, item)
	}
	if err := evidenceRows.Err(); err != nil {
		return nil, nil, err
	}
	findingRows, err := r.db.QueryContext(ctx, fmt.Sprintf(`SELECT review_key, criterion_id, severity,
		status, message FROM readiness_findings WHERE review_key = %s ORDER BY criterion_id`, p(1)), key)
	if err != nil {
		return nil, nil, err
	}
	defer findingRows.Close()
	var findings []domain.ReviewFinding
	for findingRows.Next() {
		var finding domain.ReviewFinding
		if err := findingRows.Scan(&finding.ReviewKey, &finding.CriterionID, &finding.Severity, &finding.Status, &finding.Message); err != nil {
			return nil, nil, err
		}
		findings = append(findings, finding)
	}
	if err := findingRows.Err(); err != nil {
		return nil, nil, err
	}
	return evidence, findings, nil
}

func (r *SQLRepository) ListActiveWaivers(ctx context.Context, key string, now time.Time) ([]domain.ReviewWaiver, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("readiness repository is not configured")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`SELECT review_key, criterion_id, actor, reason,
		expires_at, COALESCE(invalidation_trigger, ''), created_at FROM readiness_review_waivers
		WHERE review_key = %s AND expires_at > %s ORDER BY criterion_id`, r.placeholder(1), r.placeholder(2)), key, now.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.ReviewWaiver
	for rows.Next() {
		var waiver domain.ReviewWaiver
		if err := rows.Scan(&waiver.ReviewKey, &waiver.CriterionID, &waiver.Actor, &waiver.Reason, &waiver.ExpiresAt, &waiver.Trigger, &waiver.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, waiver)
	}
	return result, rows.Err()
}

func (r *SQLRepository) ListWaivers(ctx context.Context, key string, limit int) ([]domain.ReviewWaiver, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT review_key, criterion_id, actor, reason, expires_at, COALESCE(invalidation_trigger, ''), created_at FROM readiness_review_waivers`
	args := []any{}
	if strings.TrimSpace(key) != "" {
		query += fmt.Sprintf(" WHERE review_key = %s", r.placeholder(1))
		args = append(args, key)
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %s", r.placeholder(len(args)+1))
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.ReviewWaiver
	for rows.Next() {
		var waiver domain.ReviewWaiver
		if err := rows.Scan(&waiver.ReviewKey, &waiver.CriterionID, &waiver.Actor, &waiver.Reason, &waiver.ExpiresAt, &waiver.Trigger, &waiver.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, waiver)
	}
	return result, rows.Err()
}

func (r *SQLRepository) ReplaceEvaluation(ctx context.Context, key string, evidence []domain.EvidenceItem, findings []domain.ReviewFinding, status domain.ReviewStatus) error {
	if r == nil || r.db == nil {
		return errors.New("readiness repository is not configured")
	}
	if status != domain.ReviewCollecting && status != domain.ReviewBlocked && status != domain.ReviewAgentReview {
		return errors.New("evaluation can only set collecting, blocked, or agent_review")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	p := r.placeholder
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM readiness_evidence WHERE review_key = %s", p(1)), key); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM readiness_findings WHERE review_key = %s", p(1)), key); err != nil {
		return err
	}
	for _, item := range evidence {
		if item.ReviewKey != key || item.CriterionID == "" || item.Producer == "" || item.CandidateCommit == "" || item.ArtifactDigest == "" || item.Target == "" || item.Environment == "" || item.PolicyVersion <= 0 || item.ObservedAt.IsZero() || item.Reference == "" {
			return errors.New("evidence identity, attribution, observation, and reference are required")
		}
		if item.Applicability == "not_applicable" && item.ApplicabilityReason == "" {
			return errors.New("not_applicable evidence requires an attributable reason")
		}
		_, err = tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO readiness_evidence
			(review_key, criterion_id, status, applicability, applicability_reason, producer,
			 producer_version, candidate_commit, artifact_digest, target, environment,
			 policy_version, observed_at, evidence_reference, detail)
			VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)`, p(1), p(2), p(3), p(4), p(5), p(6), p(7), p(8), p(9), p(10), p(11), p(12), p(13), p(14), p(15)),
			key, item.CriterionID, item.Status, item.Applicability, nullString(item.ApplicabilityReason), item.Producer,
			nullString(item.ProducerVersion), item.CandidateCommit, item.ArtifactDigest, item.Target, item.Environment,
			item.PolicyVersion, item.ObservedAt.UTC(), item.Reference, nullString(item.Detail))
		if err != nil {
			return fmt.Errorf("insert readiness evidence: %w", err)
		}
	}
	for _, finding := range findings {
		if finding.ReviewKey != key || finding.CriterionID == "" || finding.Severity == "" || finding.Message == "" {
			return errors.New("finding identity, severity, and message are required")
		}
		_, err = tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO readiness_findings
			(review_key, criterion_id, severity, status, message) VALUES (%s,%s,%s,%s,%s)`, p(1), p(2), p(3), p(4), p(5)), key, finding.CriterionID, finding.Severity, finding.Status, finding.Message)
		if err != nil {
			return fmt.Errorf("insert readiness finding: %w", err)
		}
	}
	result, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE readiness_reviews SET status = %s,
		goal_closed_at = NULL, approved_at = NULL, approved_by = NULL, updated_at = %s WHERE review_key = %s`, p(1), p(2), p(3)), status, time.Now().UTC(), key)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func (r *SQLRepository) SetGoal(ctx context.Context, key, goal string) error {
	if strings.TrimSpace(goal) == "" {
		return errors.New("goal reference is required")
	}
	return r.updateOne(ctx, `UPDATE readiness_reviews SET goal_ref = %s, updated_at = %s WHERE review_key = %s`, goal, time.Now().UTC(), key)
}

func (r *SQLRepository) RecordGoalClosure(ctx context.Context, key string, at time.Time) error {
	if at.IsZero() {
		return errors.New("goal closure time is required")
	}
	return r.updateOne(ctx, `UPDATE readiness_reviews SET goal_closed_at = %s, updated_at = %s WHERE review_key = %s AND goal_ref IS NOT NULL`, at.UTC(), time.Now().UTC(), key)
}

func (r *SQLRepository) Approve(ctx context.Context, key string, identity domain.ReviewIdentity, actor string, at time.Time) error {
	if strings.TrimSpace(actor) == "" || at.IsZero() {
		return errors.New("approval actor and time are required")
	}
	expected, err := identity.Key()
	if err != nil {
		return err
	}
	if expected != key {
		return errors.New("approval identity does not match review key")
	}
	p := r.placeholder
	result, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE readiness_reviews SET status = %s,
		approved_at = %s, approved_by = %s, updated_at = %s
		WHERE review_key = %s AND status = %s AND goal_closed_at IS NOT NULL`, p(1), p(2), p(3), p(4), p(5), p(6)), domain.ReviewApproved, at.UTC(), actor, time.Now().UTC(), key, domain.ReviewAgentReview)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return errors.New("review is not eligible for exact-identity approval")
	}
	return nil
}

func (r *SQLRepository) SaveWaiver(ctx context.Context, waiver domain.ReviewWaiver) error {
	if waiver.ReviewKey == "" || waiver.CriterionID == "" || strings.TrimSpace(waiver.Actor) == "" || strings.TrimSpace(waiver.Reason) == "" || waiver.ExpiresAt.IsZero() {
		return errors.New("waiver review, criterion, actor, reason, and expiry are required")
	}
	if waiver.CreatedAt.IsZero() {
		waiver.CreatedAt = time.Now().UTC()
	}
	if !waiver.ExpiresAt.After(waiver.CreatedAt) {
		return errors.New("waiver expiry must be after creation")
	}
	p := r.placeholder
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO readiness_review_waivers
		(review_key, criterion_id, actor, reason, expires_at, invalidation_trigger, created_at)
		VALUES (%s,%s,%s,%s,%s,%s,%s)
		ON CONFLICT (review_key, criterion_id) DO UPDATE SET actor=excluded.actor, reason=excluded.reason,
		expires_at=excluded.expires_at, invalidation_trigger=excluded.invalidation_trigger, created_at=excluded.created_at`, p(1), p(2), p(3), p(4), p(5), p(6), p(7)), waiver.ReviewKey, waiver.CriterionID, waiver.Actor, waiver.Reason, waiver.ExpiresAt.UTC(), nullString(waiver.Trigger), waiver.CreatedAt.UTC())
	return err
}

func (r *SQLRepository) SaveObservation(ctx context.Context, observation domain.EvidenceObservation) error {
	identity, err := observation.Identity.Canonical()
	if err != nil {
		return err
	}
	key, err := identity.Key()
	if err != nil {
		return err
	}
	evidence := observation.Evidence
	if observation.CriterionID == "" || observation.ProducerBinding == "" || evidence.Producer == "" || evidence.Reference == "" || evidence.ObservedAt.IsZero() {
		return errors.New("criterion, producer binding, producer, observation time, and evidence reference are required")
	}
	if evidence.Status != domain.SignalPassed && evidence.Status != domain.SignalFailed && evidence.Status != domain.SignalUnknown && evidence.Status != domain.SignalUnavailable {
		return errors.New("producer observation status must be passed, failed, unknown, or unavailable")
	}
	p := r.placeholder
	_, err = r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO readiness_observations
		(identity_key, criterion_id, producer_binding, producer, producer_version,
		 candidate_commit, artifact_digest, target, environment, policy_version,
		 status, observed_at, evidence_reference, detail)
		VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)
		ON CONFLICT (identity_key, criterion_id, producer_binding) DO UPDATE SET
		producer=excluded.producer, producer_version=excluded.producer_version,
		candidate_commit=excluded.candidate_commit, artifact_digest=excluded.artifact_digest,
		target=excluded.target, environment=excluded.environment, policy_version=excluded.policy_version,
		status=excluded.status, observed_at=excluded.observed_at,
		evidence_reference=excluded.evidence_reference, detail=excluded.detail`,
		p(1), p(2), p(3), p(4), p(5), p(6), p(7), p(8), p(9), p(10), p(11), p(12), p(13), p(14)),
		key, observation.CriterionID, observation.ProducerBinding, evidence.Producer, nullString(evidence.ProducerVersion),
		identity.CandidateCommit, identity.ArtifactDigest, strings.Join(identity.Targets, ","), identity.Channel,
		identity.PolicyVersion, evidence.Status, evidence.ObservedAt.UTC(), evidence.Reference, nullString(evidence.Detail))
	return err
}

func (r *SQLRepository) FindObservation(ctx context.Context, identity domain.ReviewIdentity, criterionID, producerBinding string) (*domain.EvidenceItem, error) {
	key, err := identity.Key()
	if err != nil {
		return nil, err
	}
	var item domain.EvidenceItem
	err = r.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT criterion_id, status, producer,
		COALESCE(producer_version, ''), candidate_commit, artifact_digest, target,
		environment, policy_version, observed_at, evidence_reference, COALESCE(detail, '')
		FROM readiness_observations WHERE identity_key = %s AND criterion_id = %s
		AND producer_binding = %s`, r.placeholder(1), r.placeholder(2), r.placeholder(3)), key, criterionID, producerBinding).Scan(
		&item.CriterionID, &item.Status, &item.Producer, &item.ProducerVersion,
		&item.CandidateCommit, &item.ArtifactDigest, &item.Target, &item.Environment,
		&item.PolicyVersion, &item.ObservedAt, &item.Reference, &item.Detail)
	if err != nil {
		return nil, err
	}
	item.Applicability = "applicable"
	return &item, nil
}

func (r *SQLRepository) MarkPromoted(ctx context.Context, key string, at time.Time) error {
	if at.IsZero() {
		return errors.New("promotion time is required")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	p := r.placeholder
	result, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE readiness_reviews SET status = %s,
		updated_at = %s WHERE review_key = %s AND status = %s`, p(1), p(2), p(3), p(4)),
		domain.ReviewPromoted, at.UTC(), key, domain.ReviewApproved)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return errors.New("only an approved review can be marked promoted")
	}
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`UPDATE readiness_reviews SET status = %s, updated_at = %s
		WHERE status = %s AND review_key <> %s
		AND (scenario, profile_id, channel, targets_json) =
		(SELECT scenario, profile_id, channel, targets_json FROM readiness_reviews WHERE review_key = %s)`,
		p(1), p(2), p(3), p(4), p(5)), domain.ReviewSuperseded, at.UTC(), domain.ReviewPromoted, key, key)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLRepository) SaveHumanCheck(ctx context.Context, check domain.HumanCheck) error {
	if check.ReviewKey == "" || check.CriterionID == "" || check.Actor == "" || check.EvidenceReference == "" || check.ReviewedAt.IsZero() {
		return errors.New("human check review, criterion, actor, evidence reference, and review time are required")
	}
	if check.Verdict != "passed" && check.Verdict != "failed" {
		return errors.New("human check verdict must be passed or failed")
	}
	p := r.placeholder
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO readiness_human_checks
		(review_key, criterion_id, verdict, actor, evidence_reference, reviewed_at)
		VALUES (%s,%s,%s,%s,%s,%s)
		ON CONFLICT (review_key, criterion_id) DO UPDATE SET verdict=excluded.verdict,
		actor=excluded.actor, evidence_reference=excluded.evidence_reference,
		reviewed_at=excluded.reviewed_at`, p(1), p(2), p(3), p(4), p(5), p(6)),
		check.ReviewKey, check.CriterionID, check.Verdict, check.Actor, check.EvidenceReference, check.ReviewedAt.UTC())
	return err
}

func (r *SQLRepository) ListHumanChecks(ctx context.Context, key string) ([]domain.HumanCheck, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`SELECT review_key, criterion_id, verdict,
		actor, evidence_reference, reviewed_at FROM readiness_human_checks
		WHERE review_key = %s ORDER BY criterion_id`, r.placeholder(1)), key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var checks []domain.HumanCheck
	for rows.Next() {
		var check domain.HumanCheck
		if err := rows.Scan(&check.ReviewKey, &check.CriterionID, &check.Verdict, &check.Actor, &check.EvidenceReference, &check.ReviewedAt); err != nil {
			return nil, err
		}
		checks = append(checks, check)
	}
	return checks, rows.Err()
}

func (r *SQLRepository) updateOne(ctx context.Context, query string, args ...any) error {
	p := r.placeholder
	query = fmt.Sprintf(query, p(1), p(2), p(3))
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func nullString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
