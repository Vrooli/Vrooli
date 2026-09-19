package supervision

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type freshControllerFixture struct {
	*fakeActionController
	replacement  *domain.Run
	failure      error
	readFailure  error
	calls, reads int
	message      string
}

func TestFreshRecoveryRegistryReconcilesAcceptedHandoffBeforeJoining(t *testing.T) {
	for _, variant := range []string{"lost-response", "delivery-before-binding", "receipt-outage"} {
		t.Run(variant, func(t *testing.T) {
			s, r, c, e, req := freshFixture(t)
			c.failure = errors.New("response lost after admission")
			d, err := s.RequestDirective(t.Context(), req, EffortActor{ID: e.SupervisorRunId})
			if err == nil || d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN {
				t.Fatal("fixture did not retain uncertainty", d, err)
			}
			if variant == "delivery-before-binding" {
				next, err := freshRecoveryResult(s, d, c.replacement)
				if err != nil {
					t.Fatal(err)
				}
				next.Revision++
				if err = r.SaveEffortDirective(t.Context(), next, d.Revision, "simulated-crash-before-binding", effortDigest(next)); err != nil {
					t.Fatal(err)
				}
			}
			if variant == "receipt-outage" {
				c.readFailure = errors.New("owner unavailable")
			}
			c.replacement.WorkReferences = []*eventpb.WorkReference{{Kind: "effort", Id: e.EffortRef, Relationship: "orchestrator", Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE}}
			restarted := NewEffortService(r, c, s.policies, s.config)
			restarted.now = s.now
			restarted.SetRunRegistry(effortRegistryFixture{runs: []*domain.Run{c.replacement}})
			err = restarted.Tick(t.Context())
			if variant != "receipt-outage" && err != nil {
				t.Fatal(err)
			}
			current, _, err := r.GetEffort(t.Context(), e.EffortRef)
			if err != nil {
				t.Fatal(err)
			}
			if current.AuthorizedBy != e.AuthorizedBy || current.MaximumDirectives != e.MaximumDirectives || !proto.Equal(current.AuthorityExpiresAt, e.AuthorityExpiresAt) || !slices.Equal(current.PermittedActions, e.PermittedActions) {
				t.Fatal("discovery erased or renewed original grant", current)
			}
			if c.calls != 1 {
				t.Fatal("discovery repeated admission", c.calls)
			}
			if variant == "receipt-outage" {
				if len(current.Subjects) != 1 || current.Subjects[0].RunId != d.TargetRunId {
					t.Fatal("unproven handoff changed authority", current)
				}
				return
			}
			if len(current.Subjects) != 2 || current.Subjects[0].Role != "previous-orchestrator" || current.Subjects[1].Role != "orchestrator" || current.Subjects[1].RunId != c.replacement.ID.String() {
				t.Fatal("leader handoff not reconciled", current)
			}
			d, err = r.GetEffortDirective(t.Context(), d.DirectiveId)
			if err != nil {
				t.Fatal(err)
			}
			now := s.now().Add(time.Second)
			restarted.now = func() time.Time { return now }
			c.replacement.LastHeartbeat = &now
			verified, err := restarted.UpdateDirective(t.Context(), &pb.UpdateEffortDirectiveRequest{DirectiveId: d.DirectiveId, ExpectedRevision: d.Revision, IdempotencyKey: "verify-after-registry", RecoveryVerification: &pb.EffortRecoveryVerification{State: "progress-observed", Reason: "new passing regression compared to retained baseline", EvidenceRefs: []string{"test-genie:replacement-pass"}, ObservedAt: timestamppb.New(now)}}, EffortActor{ID: e.SupervisorRunId})
			if err != nil || verified.GetRecoveryVerification().GetState() != "progress-observed" {
				t.Fatal("verification grant lost after discovery", verified, err)
			}
		})
	}
}

func (f *freshControllerFixture) RecoverMissingSession(_ context.Context, id uuid.UUID, message string) (*domain.Run, error) {
	f.calls++
	f.message = message
	if f.replacement != nil {
		f.runs[f.replacement.ID] = f.replacement
	}
	if f.failure != nil {
		return nil, f.failure
	}
	return f.replacement, nil
}
func (f *freshControllerFixture) FreshRecoveryAccepted(_ context.Context, id uuid.UUID, message string) (*domain.Run, error) {
	f.reads++
	if message != f.message {
		return nil, errors.New("receipt payload changed")
	}
	return f.replacement, f.readFailure
}
func freshFixture(t *testing.T) (*EffortService, *Repository, *freshControllerFixture, *pb.EffortEnrollment, *pb.RequestEffortDirectiveRequest) {
	t.Helper()
	s, r, base := effortFixture(t)
	e := grantFixture(t, s, base, domain.RunStatusFailed)
	e.PermittedActions = []pb.WatchActionKind{pb.WatchActionKind_WATCH_ACTION_KIND_RECOVER_FRESH}
	var err error
	e, err = s.Enroll(t.Context(), &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "fresh-grant"}, EffortActor{ID: "owner", Operator: true})
	if err != nil {
		t.Fatal(err)
	}
	c := &freshControllerFixture{fakeActionController: base, replacement: &domain.Run{ID: uuid.New(), Status: domain.RunStatusRunning, SourceRunIDs: []uuid.UUID{uuid.MustParse(e.Subjects[0].RunId)}}}
	s.controller = c
	req := directiveFixture(s, e)
	req.Directive.Kind = pb.WatchActionKind_WATCH_ACTION_KIND_RECOVER_FRESH
	req.Directive.RecoveryExpectation = &pb.EffortRecoveryExpectation{ProgressCondition: "the assigned regression has a new passing receipt", BaselineEvidenceRefs: []string{"test-genie:before"}}
	return s, r, c, e, req
}

func TestFreshEffortRecoveryRetainsLineageAndOriginalGrant(t *testing.T) {
	s, r, c, e, req := freshFixture(t)
	actor := EffortActor{ID: e.SupervisorRunId}
	d, err := s.RequestDirective(t.Context(), req, actor)
	if err != nil {
		t.Fatal(err)
	}
	if d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED || d.RecoveredRunId != c.replacement.ID.String() || c.calls != 1 || c.continued != 0 || d.GetRecoveryVerification().GetState() != "pending" {
		t.Fatal("fresh admission or pending verification wrong", d)
	}
	current, _, err := r.GetEffort(t.Context(), e.EffortRef)
	if err != nil {
		t.Fatal(err)
	}
	if len(current.Subjects) != 2 || current.Subjects[0].Role != "previous-orchestrator" || current.Subjects[1].Role != "orchestrator" || current.Subjects[1].RunId != d.RecoveredRunId || current.AuthorityRef != e.AuthorityRef || current.AuthorizedBy != e.AuthorizedBy || current.MaximumDirectives != e.MaximumDirectives || !proto.Equal(current.AuthorityExpiresAt, e.AuthorityExpiresAt) || !proto.Equal(d.SourceSnapshot.Enrollment, e) {
		t.Fatal("replacement renewed grant, lost predecessor or original baseline", current, d)
	}
	now := s.now().Add(time.Second)
	s.now = func() time.Time { return now }
	c.replacement.LastHeartbeat = &now
	verified, err := s.UpdateDirective(t.Context(), &pb.UpdateEffortDirectiveRequest{DirectiveId: d.DirectiveId, ExpectedRevision: d.Revision, IdempotencyKey: "verify-fresh", RecoveryVerification: &pb.EffortRecoveryVerification{State: "progress-observed", Reason: "compared retained failing baseline with the replacement's new passing regression", EvidenceRefs: []string{"test-genie:after"}, ObservedAt: timestamppb.New(now)}}, actor)
	if err != nil || verified.GetRecoveryVerification().GetState() != "progress-observed" {
		t.Fatal("verification did not use exact replacement owner", verified, err)
	}
	if _, err = s.UpdateDirective(t.Context(), &pb.UpdateEffortDirectiveRequest{DirectiveId: d.DirectiveId, ExpectedRevision: verified.Revision, IdempotencyKey: "ack-fresh", Acknowledgment: pb.EffortDirectiveAcknowledgment_EFFORT_DIRECTIVE_ACKNOWLEDGMENT_ACCEPTED}, EffortActor{ID: d.RecoveredRunId}); err != nil {
		t.Fatal("replacement could not acknowledge its directive", err)
	}
	if _, err = s.RequestDirective(t.Context(), req, actor); err != nil || c.calls != 1 {
		t.Fatal("request replay created another replacement", err, c.calls)
	}
}

func TestFreshEffortRecoveryRefusesWithoutSeparateAuthorityOrOwnerProof(t *testing.T) {
	for _, variant := range []string{"no-grant", "no-expectation", "no-comparison", "cancelled", "complete", "pre-effect-refusal", "wrong-lineage"} {
		t.Run(variant, func(t *testing.T) {
			s, _, c, e, req := freshFixture(t)
			source := c.runs[uuid.MustParse(req.Directive.TargetRunId)]
			switch variant {
			case "no-grant":
				e.PermittedActions = []pb.WatchActionKind{pb.WatchActionKind_WATCH_ACTION_KIND_CONTINUE}
				var err error
				e, err = s.Enroll(t.Context(), &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "no-fresh"}, EffortActor{ID: "owner", Operator: true})
				if err != nil {
					t.Fatal(err)
				}
				req.ExpectedEnrollmentRevision = e.Revision
			case "no-expectation":
				req.Directive.RecoveryExpectation = nil
			case "no-comparison":
				req.Directive.Comparison = ""
			case "cancelled":
				source.Status = domain.RunStatusCancelled
			case "complete":
				source.Status = domain.RunStatusComplete
			case "pre-effect-refusal":
				c.failure = domain.RefuseBeforeEffects(errors.New("executor exclusion incomplete"))
			case "wrong-lineage":
				c.replacement.SourceRunIDs = []uuid.UUID{uuid.New()}
			}
			d, err := s.RequestDirective(t.Context(), req, EffortActor{ID: e.SupervisorRunId})
			if variant == "wrong-lineage" {
				if err == nil || d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN {
					t.Fatal("unproven replacement was accepted", d, err)
				}
			} else if err != nil || d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED {
				t.Fatal("unsafe fresh recovery not refused", d, err)
			}
			if c.continued != 0 || (variant != "pre-effect-refusal" && variant != "wrong-lineage" && c.calls != 0) {
				t.Fatal("refusal produced an effect", c.calls, c.continued)
			}
			if variant == "pre-effect-refusal" && !d.RefusalBeforeEffects {
				t.Fatal("owner refusal proof lost")
			}
		})
	}
}

func TestFreshEffortRecoveryLostResponseReconcilesWithoutResendOrGrantRenewal(t *testing.T) {
	for _, variant := range []string{"accepted", "withdrawn", "expired", "missing", "unavailable"} {
		t.Run(variant, func(t *testing.T) {
			s, r, c, e, req := freshFixture(t)
			c.failure = errors.New("response lost after owner admission")
			d, err := s.RequestDirective(t.Context(), req, EffortActor{ID: e.SupervisorRunId})
			if err == nil || d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN {
				t.Fatal(d, err)
			}
			switch variant {
			case "withdrawn":
				_, err = s.Withdraw(t.Context(), &pb.WithdrawEffortRequest{EffortRef: e.EffortRef, ExpectedRevision: e.Revision, IdempotencyKey: "withdraw", Reason: "owner withdrew"}, EffortActor{ID: "owner", Operator: true})
				if err != nil {
					t.Fatal(err)
				}
			case "expired":
				now := e.AuthorityExpiresAt.AsTime().Add(time.Second)
				s.now = func() time.Time { return now }
			case "missing":
				c.replacement = nil
			case "unavailable":
				c.readFailure = errors.New("owner unavailable")
			}
			restarted := NewEffortService(r, c, s.policies, s.config)
			restarted.now = s.now
			got, err := restarted.deliverDirective(t.Context(), d)
			if variant == "unavailable" {
				if err == nil {
					t.Fatal("receipt outage hidden")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if c.calls != 1 || c.continued != 0 {
				t.Fatal("uncertain fresh effect resent", c.calls, c.continued)
			}
			current, _, err := r.GetEffort(t.Context(), e.EffortRef)
			if err != nil {
				t.Fatal(err)
			}
			if variant == "accepted" {
				if got.RecoveredRunId != c.replacement.ID.String() || current.Subjects[1].RunId != got.RecoveredRunId {
					t.Fatal("recovery identity not reconciled", got, current)
				}
			} else if len(current.Subjects) != 1 || current.Subjects[0].RunId != d.TargetRunId {
				t.Fatal("historical receipt renewed or transferred authority", current)
			}
			if variant == "missing" || variant == "unavailable" {
				if got.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN {
					t.Fatal("missing receipt invented admission", got)
				}
			} else if got.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED || got.GetRecoveryVerification().GetState() != "pending" {
				t.Fatal("receipt lost identity or invented progress", got)
			}
		})
	}
}
