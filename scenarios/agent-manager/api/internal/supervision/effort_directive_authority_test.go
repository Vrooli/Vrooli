package supervision

import (
	"testing"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
