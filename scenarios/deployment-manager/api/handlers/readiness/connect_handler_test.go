package readinesshandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"deployment-manager/internal/authz"
	internalreadiness "deployment-manager/internal/readiness"
	"deployment-manager/internal/testutil"
	domain "deployment-manager/readiness"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/readiness"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestReportEvidenceEnforcesPolicyBindingAndExactIdentity(t *testing.T) {
	db := testutil.OpenSQLite(t)
	if _, err := db.Exec(internalreadiness.Schema()); err != nil {
		t.Fatal(err)
	}
	repo := internalreadiness.NewSQLRepository(db, "sqlite")
	handler := NewConnectHandler(nil, repo, nil).WithEvidenceAuthorization(func(context.Context) error { return nil })
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

func TestReportEvidenceRequiresOwnerAuthorizationWhenConfigured(t *testing.T) {
	handler := NewConnectHandler(nil, nil, nil).WithEvidenceAuthorization(func(context.Context) error {
		return errors.New("missing owner service")
	})
	request := &readinessv1.ReportEvidenceRequest{
		Scenario: "demo", ProfileId: "p1", CandidateCommit: "abc", ArtifactDigest: "sha256:candidate",
		Targets: []string{"linux"}, Channel: "stable", PolicyVersion: 2,
		CriterionId: "suite-state-known", ProducerBinding: "test-genie.runs.report", Status: "passed",
		ObservedAt: timestamppb.New(time.Now().UTC()), EvidenceReference: "run:123",
	}
	if _, err := handler.ReportEvidence(context.Background(), connect.NewRequest(request)); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("unauthorized report code=%v err=%v", connect.CodeOf(err), err)
	}
}

func TestReportEvidenceFailsClosedWithoutAuthorization(t *testing.T) {
	request := &readinessv1.ReportEvidenceRequest{
		Scenario: "demo", ProfileId: "p1", CandidateCommit: "abc", ArtifactDigest: "sha256:candidate",
		Targets: []string{"linux"}, Channel: "stable", PolicyVersion: 2,
		CriterionId: "suite-state-known", ProducerBinding: "test-genie.runs.report", Status: "passed",
		ObservedAt: timestamppb.New(time.Now().UTC()), EvidenceReference: "run:123",
	}
	if _, err := NewConnectHandler(nil, nil, nil).ReportEvidence(context.Background(), connect.NewRequest(request)); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("unconfigured readiness authorization code=%v err=%v", connect.CodeOf(err), err)
	}
}

func TestReportEvidencePassesPolicyBindingToOwnerAuthorization(t *testing.T) {
	var gotBinding string
	handler := NewConnectHandler(nil, nil, nil).WithEvidenceBindingAuthorization(func(_ context.Context, binding string) error {
		gotBinding = binding
		return errors.New("owner mismatch")
	})
	request := &readinessv1.ReportEvidenceRequest{
		Scenario: "demo", ProfileId: "p1", CandidateCommit: "abc", ArtifactDigest: "sha256:candidate",
		Targets: []string{"linux"}, Channel: "stable", PolicyVersion: 2,
		CriterionId: "suite-state-known", ProducerBinding: "test-genie.runs.report", Status: "passed",
		ObservedAt: timestamppb.New(time.Now().UTC()), EvidenceReference: "run:123",
	}
	if _, err := handler.ReportEvidence(context.Background(), connect.NewRequest(request)); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("binding authorization code=%v err=%v", connect.CodeOf(err), err)
	}
	if gotBinding != "test-genie.runs.report" {
		t.Fatalf("binding=%q", gotBinding)
	}
}

func TestReportEvidenceRejectsFutureObservationBeyondClockSkew(t *testing.T) {
	handler := NewConnectHandler(nil, nil, nil).WithEvidenceAuthorization(func(context.Context) error { return nil })
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	handler.now = func() time.Time { return now }
	request := &readinessv1.ReportEvidenceRequest{
		Scenario: "demo", ProfileId: "p1", CandidateCommit: "abc", ArtifactDigest: "sha256:candidate",
		Targets: []string{"linux"}, Channel: "stable", PolicyVersion: 2,
		CriterionId: "suite-state-known", ProducerBinding: "test-genie.runs.report", Status: "passed",
		ObservedAt: timestamppb.New(now.Add(maxOwnerObservationFutureSkew + time.Second)), EvidenceReference: "run:future",
	}
	if _, err := handler.ReportEvidence(context.Background(), connect.NewRequest(request)); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("future observation code=%v err=%v", connect.CodeOf(err), err)
	}
}

func TestReadinessWritesRejectUnsupportedStatusesAndHumanVerdicts(t *testing.T) {
	when := timestamppb.New(time.Now().UTC())
	observation := &readinessv1.ReportEvidenceRequest{
		Scenario: "demo", ProfileId: "p1", CandidateCommit: "abc", ArtifactDigest: "sha256:candidate",
		Targets: []string{"linux"}, Channel: "stable", PolicyVersion: 2,
		CriterionId: "suite-state-known", ProducerBinding: "test-genie.runs.report", Status: "waived",
		ObservedAt: when, EvidenceReference: "run:123",
	}
	if _, err := NewConnectHandler(nil, nil, nil).ReportEvidence(context.Background(), connect.NewRequest(observation)); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unsupported producer status code=%v err=%v", connect.CodeOf(err), err)
	}

	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "verified-reviewer", Verified: true, Scopes: []string{authz.WriteCapability}})
	human := &readinessv1.RecordHumanCheckRequest{ReviewKey: "review-1", CriterionId: "update-policy-set", Verdict: "approved", EvidenceReference: "review:123", ReviewedAt: when}
	if _, err := NewConnectHandler(nil, nil, nil).RecordHumanCheck(ctx, connect.NewRequest(human)); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unsupported human verdict code=%v err=%v", connect.CodeOf(err), err)
	}
}

func TestNextActionsProjectPersistedFindings(t *testing.T) {
	policy := domain.DefaultChecklist()
	actions := nextActions(policy, []domain.ReviewFinding{{
		CriterionID: "requirements-cover-sale",
		Message:     "Requirements cover what is sold: owner observation is stale",
	}})
	if len(actions) != 1 || !strings.Contains(actions[0], "requirements-cover-sale:") || !strings.Contains(actions[0], "next action: run requirements-traceability-steer for commercial requirement coverage") {
		t.Fatalf("actions=%v", actions)
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

func TestSensitiveReadinessMutationsRequireVerifiedHuman(t *testing.T) {
	handler := NewConnectHandler(nil, nil, nil)
	request := connect.NewRequest(&readinessv1.SynchronizeGoalClosureRequest{ReviewKey: "review-1"})
	if _, err := handler.SynchronizeGoalClosure(context.Background(), request); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("anonymous closure code=%v err=%v", connect.CodeOf(err), err)
	}

	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "verified-reviewer", Verified: true, Scopes: []string{authz.WriteCapability}})
	actor, err := requireVerifiedReviewer(ctx)
	if err != nil || actor != "verified-reviewer" {
		t.Fatalf("verified reviewer=%q err=%v", actor, err)
	}
	if actor == "caller-supplied-actor" {
		t.Fatal("authorization used a caller-supplied actor")
	}

	agent := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorAgent, Subject: "agent-run", Verified: true, Scopes: []string{authz.WriteCapability}})
	if _, err := requireVerifiedReviewer(agent); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("verified agent reviewer code=%v err=%v", connect.CodeOf(err), err)
	}
}

func TestPolicyProjectionRequiresReadAuthorization(t *testing.T) {
	if _, err := NewConnectHandler(nil, nil, nil).CheckPolicyProjection(context.Background(), connect.NewRequest(&readinessv1.CheckPolicyProjectionRequest{})); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("unconfigured policy projection read code=%v err=%v", connect.CodeOf(err), err)
	}
}

func TestCreateWaiverRequiresReasonAndFutureExpiry(t *testing.T) {
	handler := NewConnectHandler(nil, nil, nil)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "verified-reviewer", Verified: true, Scopes: []string{authz.WriteCapability}})
	base := &readinessv1.CreateWaiverRequest{ReviewKey: "review-1", CriterionId: "update-policy-set", Reason: "temporary owner outage", ExpiresAt: timestamppb.New(time.Now().UTC().Add(time.Hour))}
	missingReason := connect.NewRequest(proto.Clone(base).(*readinessv1.CreateWaiverRequest))
	missingReason.Msg.Reason = ""
	if _, err := handler.CreateWaiver(ctx, missingReason); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("empty waiver reason code=%v err=%v", connect.CodeOf(err), err)
	}
	expired := connect.NewRequest(proto.Clone(base).(*readinessv1.CreateWaiverRequest))
	expired.Msg.ExpiresAt = timestamppb.New(time.Now().UTC().Add(-time.Minute))
	if _, err := handler.CreateWaiver(ctx, expired); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("expired waiver code=%v err=%v", connect.CodeOf(err), err)
	}
}
