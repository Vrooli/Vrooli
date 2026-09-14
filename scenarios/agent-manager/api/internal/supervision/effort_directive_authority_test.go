package supervision

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAutonomousSupervisionDoesNotRequireAnActionEnumeration(t *testing.T) {
	s, _, c := effortFixture(t)
	e := grantFixture(t, s, c, domain.RunStatusRunning)
	e.PermittedActions = nil
	e.AutonomousSupervision = true
	req := directiveFixture(s, e)
	d, err := s.RequestDirective(context.Background(), req, EffortActor{ID: e.SupervisorRunId})
	if err != nil {
		t.Fatal(err)
	}
	if d.Delivery == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED || d.DeliveryReason == "action outside authorized scope" {
		t.Fatalf("autonomous mandate was treated as an action list: %+v", d)
	}
}

func TestAutonomousSupervisionCanCoordinateAnEnrolledWorker(t *testing.T) {
	s, _, c := effortFixture(t)
	e := grantFixture(t, s, c, domain.RunStatusRunning)
	e.AutonomousSupervision = true
	e.PermittedActions = nil
	workerID := uuid.New()
	c.runs[workerID] = &domain.Run{ID: workerID, Status: domain.RunStatusNeedsReview}
	e.Subjects = append(e.Subjects, &pb.EffortSubject{Owner: "agent-manager", Kind: "run", Reference: workerID.String(), RunId: workerID.String(), Role: "worker"})
	enrolled, err := s.Enroll(t.Context(), &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "autonomous-worker"}, EffortActor{ID: "owner", Operator: true})
	if err != nil {
		t.Fatal(err)
	}
	req := directiveFixture(s, enrolled)
	req.ExpectedEnrollmentRevision = enrolled.Revision
	req.Directive.TargetRunId = workerID.String()
	d, err := s.RequestDirective(t.Context(), req, EffortActor{ID: enrolled.SupervisorRunId})
	if err != nil || d.Delivery == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED {
		t.Fatalf("autonomous mandate could not coordinate enrolled worker: directive=%+v err=%v", d, err)
	}
}

func TestAutonomousSupervisionCoordinatesTwoEffortsWithoutCrossingLineage(t *testing.T) {
	s, _, c := effortFixture(t)
	type effortWorker struct {
		enrollment *pb.EffortEnrollment
		workerID   uuid.UUID
	}
	efforts := make([]effortWorker, 0, 2)
	for i := 0; i < 2; i++ {
		e := grantFixture(t, s, c, domain.RunStatusRunning)
		e.AutonomousSupervision = true
		e.PermittedActions = nil
		workerID := uuid.New()
		c.runs[workerID] = &domain.Run{ID: workerID, Status: domain.RunStatusNeedsReview}
		e.Subjects = append(e.Subjects, &pb.EffortSubject{Owner: "agent-manager", Kind: "run", Reference: workerID.String(), RunId: workerID.String(), Role: "worker"})
		enrolled, err := s.Enroll(t.Context(), &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "two-effort-autonomous-" + workerID.String()}, EffortActor{ID: "owner", Operator: true})
		if err != nil {
			t.Fatal(err)
		}
		efforts = append(efforts, effortWorker{enrollment: enrolled, workerID: workerID})
	}

	for _, item := range efforts {
		req := directiveFixture(s, item.enrollment)
		req.ExpectedEnrollmentRevision = item.enrollment.Revision
		req.Directive.TargetRunId = item.workerID.String()
		got, err := s.RequestDirective(t.Context(), req, EffortActor{ID: item.enrollment.SupervisorRunId})
		if err != nil || got.Delivery == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED {
			t.Fatalf("autonomous supervisor could not advance its own worker: directive=%+v err=%v", got, err)
		}
	}
	if c.continued != 2 {
		t.Fatalf("two-effort pilot advanced %d workers, want 2", c.continued)
	}

	cross := directiveFixture(s, efforts[0].enrollment)
	cross.ExpectedEnrollmentRevision = efforts[0].enrollment.Revision
	cross.Directive.TargetRunId = efforts[1].workerID.String()
	got, err := s.RequestDirective(t.Context(), cross, EffortActor{ID: efforts[0].enrollment.SupervisorRunId})
	if err != nil {
		t.Fatal(err)
	}
	if got.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED || c.continued != 2 {
		t.Fatalf("cross-effort target escaped enrollment boundary: directive=%+v continued=%d", got, c.continued)
	}
}

func TestPendingDirectiveCannotInheritReplacementAuthority(t *testing.T) {
	for _, variant := range []string{"unchanged", "observation-revision", "new-supervisor", "new-grant", "new-owner", "expired-original-renewed", "stable-new-wake", "changed-stable-owner", "changed-stable-scope"} {
		t.Run(variant, func(t *testing.T) {
			s, _, c := effortFixture(t)
			ctx := t.Context()
			e := grantFixture(t, s, c, domain.RunStatusRunning)
			actor := EffortActor{ID: e.SupervisorRunId}
			if variant == "stable-new-wake" || variant == "changed-stable-owner" || variant == "changed-stable-scope" {
				e.SupervisorRunId = ""
				e.SupervisorOwnerSubject, e.SupervisorScope = "owner:supervision", "agent-manager:supervise"
				var err error
				e, err = s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "stable"}, EffortActor{ID: "owner", Operator: true})
				if err != nil {
					t.Fatal(err)
				}
				actor = EffortActor{ID: uuid.NewString(), OwnerSubject: e.SupervisorOwnerSubject, Scopes: []string{e.SupervisorScope}}
			}
			req := directiveFixture(s, e)
			req.Directive.ExpiresAt = timestamppb.New(s.now().Add(3 * time.Hour))
			d, err := s.RequestDirective(ctx, req, actor)
			if err != nil || d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_PENDING {
				t.Fatal(d, err)
			}
			owner := EffortActor{ID: "owner", Operator: true}
			switch variant {
			case "new-supervisor":
				e.SupervisorRunId = uuid.NewString()
			case "new-grant":
				e.AuthorityRef = "grant:replacement"
			case "new-owner":
				owner.ID = "replacement-owner"
			case "expired-original-renewed":
				now := e.AuthorityExpiresAt.AsTime().Add(time.Second)
				s.now = func() time.Time { return now }
				e.AuthorityExpiresAt = timestamppb.New(now.Add(time.Hour))
			case "changed-stable-owner":
				e.SupervisorOwnerSubject = "owner:replacement"
			case "changed-stable-scope":
				e.SupervisorScope = "agent-manager:effort-supervise"
			}
			if variant != "unchanged" {
				e, err = s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "amend"}, owner)
				if err != nil {
					t.Fatal(err)
				}
			}
			c.runs[uuid.MustParse(d.TargetRunId)].Status = domain.RunStatusNeedsReview
			got, err := s.deliverDirective(ctx, d)
			if err != nil {
				t.Fatal(err)
			}
			allowed := variant == "unchanged" || variant == "observation-revision" || variant == "stable-new-wake"
			if allowed && (c.continued != 1 || got.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED) {
				t.Fatal("valid original grant refused", got)
			}
			if !allowed && (c.continued != 0 || got.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED) {
				t.Fatal("replacement inherited queued instruction", got)
			}
		})
	}
}

func TestUncertainDeliveryReconcilesAfterEffectAuthorityEnds(t *testing.T) {
	for _, variant := range []string{"expired", "withdrawn", "superseded", "changed-target", "legacy-expired", "legacy-refused", "legacy-missing-receipt"} {
		t.Run(variant, func(t *testing.T) {
			s, _, c, e, d := recoveryFixture(t)
			ctx := t.Context()
			d, err := s.transitionDirective(ctx, d, pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN, "owner response lost")
			if err != nil {
				t.Fatal(err)
			}
			s.controller = &recoveryReceiptController{fakeActionController: c, accepted: variant != "legacy-missing-receipt"}
			switch variant {
			case "expired":
				now := e.AuthorityExpiresAt.AsTime().Add(time.Second)
				s.now = func() time.Time { return now }
			case "withdrawn":
				_, err = s.Withdraw(ctx, &pb.WithdrawEffortRequest{EffortRef: e.EffortRef, ExpectedRevision: e.Revision, Reason: "retired", IdempotencyKey: "withdraw"}, EffortActor{ID: "owner", Operator: true})
			case "changed-target":
				e.TargetRevision = "new-target"
				_, err = s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "amend"}, EffortActor{ID: "owner", Operator: true})
			case "superseded":
				replacement, createErr := s.RequestDirective(ctx, directiveFixture(s, e), EffortActor{ID: e.SupervisorRunId})
				if createErr != nil {
					t.Fatal(createErr)
				}
				d, err = s.UpdateDirective(ctx, &pb.UpdateEffortDirectiveRequest{DirectiveId: d.DirectiveId, ExpectedRevision: d.Revision, IdempotencyKey: "supersede", SupersededBy: replacement.DirectiveId}, EffortActor{ID: e.SupervisorRunId})
			default:
				state := pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_EXPIRED
				if variant == "legacy-refused" {
					state = pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED
				}
				d, err = s.transitionDirective(ctx, d, state, "legacy disposition concealed uncertain effect")
			}
			if err != nil {
				t.Fatal(err)
			}
			var got *pb.EffortDirective
			if variant == "legacy-expired" || variant == "legacy-refused" || variant == "legacy-missing-receipt" {
				req := &pb.UpdateEffortDirectiveRequest{DirectiveId: d.DirectiveId, ExpectedRevision: d.Revision, IdempotencyKey: "reconcile", ReconcileDelivery: true}
				got, err = s.UpdateDirective(ctx, req, EffortActor{ID: "owner", Operator: true})
				if err != nil {
					t.Fatal(err)
				}
				again, replayErr := s.UpdateDirective(ctx, req, EffortActor{ID: "owner", Operator: true})
				if replayErr != nil || !proto.Equal(got, again) {
					t.Fatal("receipt request replay not durable", again, replayErr)
				}
				req.Reason = "different request"
				if _, conflict := s.UpdateDirective(ctx, req, EffortActor{ID: "owner", Operator: true}); conflict == nil {
					t.Fatal("receipt identity accepted changed payload")
				}
			} else {
				got, err = s.deliverDirective(ctx, d)
			}
			if err != nil {
				t.Fatal(err)
			}
			want := pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED
			if variant == "legacy-missing-receipt" {
				want = pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN
			}
			if c.continued != 1 || got.Delivery != want || got.SupersededBy != d.SupersededBy || !proto.Equal(got.SourceSnapshot, d.SourceSnapshot) || got.GetRecoveryVerification().GetState() != "pending" {
				t.Fatal("historical receipt resent, lost provenance, or fabricated progress", got, c.continued)
			}
		})
	}
}
