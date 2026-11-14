package readiness

import (
	"context"
	"testing"
	"time"

	"deployment-manager/internal/testutil"
	domain "deployment-manager/readiness"
)

func openRepository(t *testing.T) *SQLRepository {
	t.Helper()
	db := testutil.OpenSQLite(t)
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatalf("create readiness schema: %v", err)
	}
	return NewSQLRepository(db, "sqlite")
}

func testReview() domain.Review {
	return domain.Review{Identity: domain.ReviewIdentity{
		Scenario: "demo", ProfileID: "profile-1", CandidateCommit: "abc",
		ArtifactDigest: "sha256:candidate", Targets: []string{"linux"}, Channel: "stable", PolicyVersion: 2,
	}, ComparisonMode: domain.ComparisonFirstRelease}
}

func TestCreateOrGetIsDeterministicAndDoesNotFabricateApproval(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	first := testReview()
	deduped, err := repo.CreateOrGet(ctx, &first)
	if err != nil || deduped {
		t.Fatalf("first create deduped=%t err=%v", deduped, err)
	}
	second := testReview()
	deduped, err = repo.CreateOrGet(ctx, &second)
	if err != nil || !deduped || second.Key != first.Key {
		t.Fatalf("second=%+v deduped=%t err=%v", second, deduped, err)
	}
	if second.ApprovedAt != nil || second.GoalClosedAt != nil || second.Status != domain.ReviewCollecting {
		t.Fatalf("new review fabricated lifecycle state: %+v", second)
	}
}

func TestCreateOrGetPersistsReleaseBindingAlongsideLegacyReview(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	review := testReview()
	review.Identity.CandidateID = "candidate-1"
	review.Identity.DestinationRevisionID = "destination-1"
	review.Identity.AuthorizationEpoch = 7
	if _, err := repo.CreateOrGet(ctx, &review); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, review.Key)
	if err != nil {
		t.Fatal(err)
	}
	if got.Identity.CandidateID != "candidate-1" || got.Identity.DestinationRevisionID != "destination-1" || got.Identity.AuthorizationEpoch != 7 {
		t.Fatalf("release binding was not persisted: %+v", got.Identity)
	}
	legacy := testReview()
	legacy.Identity.CandidateCommit = "legacy"
	if _, err := repo.CreateOrGet(ctx, &legacy); err != nil {
		t.Fatal(err)
	}
	legacyRead, err := repo.Get(ctx, legacy.Key)
	if err != nil {
		t.Fatal(err)
	}
	if legacyRead.Identity.CandidateID != "" || legacyRead.Identity.AuthorizationEpoch != 0 {
		t.Fatalf("legacy review unexpectedly gained release binding: %+v", legacyRead.Identity)
	}
}

func TestEvaluationInvalidatesClosureAndApproval(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	review := testReview()
	if _, err := repo.CreateOrGet(ctx, &review); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetGoal(ctx, review.Key, "goal-1"); err != nil {
		t.Fatal(err)
	}
	closed := time.Now().UTC()
	if err := repo.RecordGoalClosure(ctx, review.Key, closed); err != nil {
		t.Fatal(err)
	}
	if err := repo.ReplaceEvaluation(ctx, review.Key, []domain.EvidenceItem{{
		ReviewKey: review.Key, CriterionID: "suite-state-known", Status: domain.SignalPassed,
		Applicability: "applicable", Producer: "test-genie", CandidateCommit: "abc",
		ArtifactDigest: "sha256:candidate", Target: "linux", Environment: "test",
		PolicyVersion: 2, ObservedAt: closed, Reference: "run:one",
	}}, nil, domain.ReviewAgentReview); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, review.Key)
	if err != nil {
		t.Fatal(err)
	}
	if got.GoalClosedAt != nil || got.ApprovedAt != nil || got.Status != domain.ReviewAgentReview {
		t.Fatalf("evaluation did not invalidate approval state: %+v", got)
	}
}

func TestEvaluationReplacementPreservesPriorEvidenceForAudit(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	review := testReview()
	if _, err := repo.CreateOrGet(ctx, &review); err != nil {
		t.Fatal(err)
	}
	first := domain.EvidenceItem{
		ReviewKey: review.Key, CriterionID: "suite-state-known", Status: domain.SignalPassed,
		Applicability: "applicable", Producer: "test-genie", CandidateCommit: "abc",
		ArtifactDigest: "sha256:candidate", Target: "linux", Environment: "test",
		PolicyVersion: 2, ObservedAt: time.Now().UTC(), Reference: "run:first",
	}
	if err := repo.ReplaceEvaluation(ctx, review.Key, []domain.EvidenceItem{first}, nil, domain.ReviewAgentReview); err != nil {
		t.Fatal(err)
	}
	second := first
	second.Status = domain.SignalFailed
	second.Reference = "run:second"
	if err := repo.ReplaceEvaluation(ctx, review.Key, []domain.EvidenceItem{second}, nil, domain.ReviewBlocked); err != nil {
		t.Fatal(err)
	}
	current, _, err := repo.ListEvaluation(ctx, review.Key)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != 1 || current[0].Reference != "run:second" {
		t.Fatalf("current evidence = %+v", current)
	}
	var reference string
	if err := repo.db.QueryRowContext(ctx, `SELECT evidence_reference FROM readiness_evidence_history WHERE review_key = ? ORDER BY replaced_at ASC LIMIT 1`, review.Key).Scan(&reference); err != nil {
		t.Fatal(err)
	}
	if reference != "run:first" {
		t.Fatalf("archived evidence reference = %q, want run:first", reference)
	}
}

func TestApprovalRequiresExactIdentityAndGoalClosure(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	review := testReview()
	if _, err := repo.CreateOrGet(ctx, &review); err != nil {
		t.Fatal(err)
	}
	if err := repo.ReplaceEvaluation(ctx, review.Key, nil, nil, domain.ReviewAgentReview); err != nil {
		t.Fatal(err)
	}
	if err := repo.Approve(ctx, review.Key, review.Identity, "operator", time.Now().UTC()); err == nil {
		t.Fatal("expected open goal refusal")
	}
	if err := repo.SetGoal(ctx, review.Key, "goal-1"); err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordGoalClosure(ctx, review.Key, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	changed := review.Identity
	changed.ArtifactDigest = "sha256:other"
	if err := repo.Approve(ctx, review.Key, changed, "operator", time.Now().UTC()); err == nil {
		t.Fatal("expected changed identity refusal")
	}
	if err := repo.Approve(ctx, review.Key, review.Identity, "operator", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, review.Key)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.ReviewApproved || got.ApprovedAt == nil || got.ApprovedBy != "operator" {
		t.Fatalf("approval not recorded: %+v", got)
	}
}

func TestNotApplicableEvidenceRequiresReasonAndWaiverExpires(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	review := testReview()
	if _, err := repo.CreateOrGet(ctx, &review); err != nil {
		t.Fatal(err)
	}
	evidence := domain.EvidenceItem{ReviewKey: review.Key, CriterionID: "migration-upgrade-proven", Status: domain.SignalPassed, Applicability: "not_applicable", Producer: "storage-manager", CandidateCommit: "abc", ArtifactDigest: "sha256:candidate", Target: "linux", Environment: "test", PolicyVersion: 2, ObservedAt: time.Now().UTC(), Reference: "schema:no-change"}
	if err := repo.ReplaceEvaluation(ctx, review.Key, []domain.EvidenceItem{evidence}, nil, domain.ReviewAgentReview); err == nil {
		t.Fatal("expected missing applicability reason refusal")
	}
	now := time.Now().UTC()
	if err := repo.SaveWaiver(ctx, domain.ReviewWaiver{ReviewKey: review.Key, CriterionID: "compatibility-proven", Actor: "operator", Reason: "bounded beta exception", CreatedAt: now, ExpiresAt: now.Add(-time.Minute)}); err == nil {
		t.Fatal("expected expired waiver refusal")
	}
}

func TestObservationIsBoundToExactIdentityCriterionAndProducer(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	identity := testReview().Identity
	observed := time.Now().UTC().Add(-time.Minute)
	observation := domain.EvidenceObservation{Identity: identity, CriterionID: "suite-state-known", ProducerBinding: "test-genie.runs.report", Evidence: domain.EvidenceItem{
		Status: domain.SignalPassed, Producer: "test-genie", ProducerVersion: "2", ObservedAt: observed, Reference: "run:123", Detail: "release suite passed",
	}}
	if err := repo.SaveObservation(ctx, observation); err != nil {
		t.Fatal(err)
	}
	got, err := repo.FindObservation(ctx, identity, observation.CriterionID, observation.ProducerBinding)
	if err != nil {
		t.Fatal(err)
	}
	if got.CandidateCommit != identity.CandidateCommit || got.ArtifactDigest != identity.ArtifactDigest || got.Target != "linux" || got.Producer != "test-genie" || got.Reference != "run:123" {
		t.Fatalf("observation lost exact identity or attribution: %+v", got)
	}
	changed := identity
	changed.ArtifactDigest = "sha256:other"
	if _, err := repo.FindObservation(ctx, changed, observation.CriterionID, observation.ProducerBinding); err == nil {
		t.Fatal("observation leaked across artifact identity")
	}
	if _, err := repo.FindObservation(ctx, identity, observation.CriterionID, "other.binding"); err == nil {
		t.Fatal("observation leaked across producer binding")
	}
}

func TestObservationUpdateCannotMoveBackwardInTime(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	identity := testReview().Identity
	newer := time.Now().UTC()
	base := domain.EvidenceObservation{Identity: identity, CriterionID: "suite-state-known", ProducerBinding: "test-genie.runs.report", Evidence: domain.EvidenceItem{
		Status: domain.SignalPassed, Producer: "test-genie", ObservedAt: newer, Reference: "run:newer",
	}}
	if err := repo.SaveObservation(ctx, base); err != nil {
		t.Fatal(err)
	}
	older := base
	older.Evidence.Status = domain.SignalFailed
	older.Evidence.ObservedAt = newer.Add(-time.Minute)
	older.Evidence.Reference = "run:older"
	if err := repo.SaveObservation(ctx, older); err != nil {
		t.Fatal(err)
	}
	got, err := repo.FindObservation(ctx, identity, base.CriterionID, base.ProducerBinding)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.SignalPassed || got.Reference != "run:newer" || !got.ObservedAt.Equal(newer) {
		t.Fatalf("older observation replaced newer evidence: %+v", got)
	}
}

func TestOnlyApprovedReviewCanBeMarkedPromoted(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	review := testReview()
	if _, err := repo.CreateOrGet(ctx, &review); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkPromoted(ctx, review.Key, time.Now().UTC()); err == nil {
		t.Fatal("collecting review was marked promoted")
	}
	if err := repo.ReplaceEvaluation(ctx, review.Key, nil, nil, domain.ReviewAgentReview); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetGoal(ctx, review.Key, "goal-1"); err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordGoalClosure(ctx, review.Key, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := repo.Approve(ctx, review.Key, review.Identity, "operator", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkPromoted(ctx, review.Key, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, review.Key)
	if err != nil || got.Status != domain.ReviewPromoted {
		t.Fatalf("promoted review=%+v err=%v", got, err)
	}
}
