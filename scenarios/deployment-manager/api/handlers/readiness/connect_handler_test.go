package readinesshandler

import (
	"context"
	"testing"
	"time"

	internalreadiness "deployment-manager/internal/readiness"
	"deployment-manager/internal/testutil"
	domain "deployment-manager/readiness"

	"connectrpc.com/connect"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/readiness"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestReportEvidenceEnforcesPolicyBindingAndExactIdentity(t *testing.T) {
	db := testutil.OpenSQLite(t)
	if _, err := db.Exec(internalreadiness.Schema()); err != nil {
		t.Fatal(err)
	}
	repo := internalreadiness.NewSQLRepository(db, "sqlite")
	handler := NewConnectHandler(nil, repo, nil)
	request := &readinessv1.ReportEvidenceRequest{Scenario: "demo", ProfileId: "p1", CandidateCommit: "abc", ArtifactDigest: "sha256:candidate", Targets: []string{"linux"}, Channel: "stable", PolicyVersion: 2, CriterionId: "suite-state-known", ProducerBinding: "wrong.binding", Status: "passed", ObservedAt: timestamppb.New(time.Now().UTC()), EvidenceReference: "run:123"}
	if _, err := handler.ReportEvidence(context.Background(), connect.NewRequest(request)); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("wrong binding code=%v err=%v", connect.CodeOf(err), err)
	}
	request.ProducerBinding = "test-genie.runs.report"
	response, err := handler.ReportEvidence(context.Background(), connect.NewRequest(request))
	if err != nil || !response.Msg.Accepted {
		t.Fatalf("response=%+v err=%v", response, err)
	}
	item, err := repo.FindObservation(context.Background(), domain.ReviewIdentity{Scenario: "demo", ProfileID: "p1", CandidateCommit: "abc", ArtifactDigest: "sha256:candidate", Targets: []string{"linux"}, Channel: "stable", PolicyVersion: 2}, "suite-state-known", "test-genie.runs.report")
	if err != nil || item.Producer != "test-genie" || item.Reference != "run:123" {
		t.Fatalf("stored item=%+v err=%v", item, err)
	}
}

func TestApprovalRevalidationRequiresPassingMechanicalAndHumanEvidence(t *testing.T) {
	db := testutil.OpenSQLite(t)
	if _, err := db.Exec(internalreadiness.Schema()); err != nil {
		t.Fatal(err)
	}
	repo := internalreadiness.NewSQLRepository(db, "sqlite")
	policy := domain.DefaultChecklist()
	identity := domain.ReviewIdentity{Scenario: "demo", ProfileID: "p1", CandidateCommit: "abc", ArtifactDigest: "sha256:candidate", Targets: []string{"linux"}, Channel: "stable", PolicyVersion: policy.Version}
	review := domain.Review{Identity: identity, ComparisonMode: domain.ComparisonFirstRelease}
	if _, err := repo.CreateOrGet(context.Background(), &review); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	evidence := make([]domain.EvidenceItem, 0, len(policy.Items))
	for _, criterion := range policy.Items {
		status := domain.SignalPassed
		if criterion.HumanReview != nil {
			status = domain.SignalUnknown
		}
		evidence = append(evidence, domain.EvidenceItem{ReviewKey: review.Key, CriterionID: criterion.ID, Status: status, Applicability: "applicable", Producer: criterion.Owner, CandidateCommit: identity.CandidateCommit, ArtifactDigest: identity.ArtifactDigest, Target: "linux", Environment: identity.Channel, PolicyVersion: identity.PolicyVersion, ObservedAt: now, Reference: "evidence:" + criterion.ID})
	}
	if err := repo.ReplaceEvaluation(context.Background(), review.Key, evidence, nil, domain.ReviewAgentReview); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetGoal(context.Background(), review.Key, "goal-1"); err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordGoalClosure(context.Background(), review.Key, now); err != nil {
		t.Fatal(err)
	}
	stored, _ := repo.Get(context.Background(), review.Key)
	handler := NewConnectHandler(nil, repo, nil)
	handler.now = func() time.Time { return now }
	if err := handler.revalidate(context.Background(), stored); err == nil {
		t.Fatal("unknown human evidence was accepted without independent checks")
	}
	for _, criterion := range policy.Items {
		if criterion.HumanReview != nil {
			if err := repo.SaveHumanCheck(context.Background(), domain.HumanCheck{ReviewKey: review.Key, CriterionID: criterion.ID, Verdict: "passed", Actor: "reviewer", EvidenceReference: "journey:" + criterion.ID, ReviewedAt: now}); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := handler.revalidate(context.Background(), stored); err != nil {
		t.Fatalf("complete independent checks were rejected: %v", err)
	}
	evidence[0].Status = domain.SignalFailed
	if err := repo.ReplaceEvaluation(context.Background(), review.Key, evidence, nil, domain.ReviewAgentReview); err != nil {
		t.Fatal(err)
	}
	stored, _ = repo.Get(context.Background(), review.Key)
	if err := handler.revalidate(context.Background(), stored); err == nil {
		t.Fatal("failed mechanical evidence was accepted")
	}
}
