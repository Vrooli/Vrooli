package supervision

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func recoveryFixture(t *testing.T) (*EffortService, *Repository, *fakeActionController, *pb.EffortEnrollment, *pb.EffortDirective) {
	t.Helper()
	s, r, c := effortFixture(t)
	e := grantFixture(t, s, c, domain.RunStatusNeedsReview)
	req := directiveFixture(s, e)
	req.Directive.RecoveryExpectation = &pb.EffortRecoveryExpectation{
		ProgressCondition:    "the assigned regression changes from failing to passing with a retained owner receipt",
		BaselineEvidenceRefs: []string{"test-genie:regression-before"},
	}
	d, err := s.RequestDirective(context.Background(), req, EffortActor{ID: e.SupervisorRunId})
	if err != nil || d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED {
		t.Fatal(d, err)
	}
	return s, r, c, e, d
}

type recoveryReceiptController struct {
	*fakeActionController
	accepted bool
	err      error
}

func (c *recoveryReceiptController) ContinuationAccepted(context.Context, uuid.UUID, string, string) (bool, error) {
	return c.accepted, c.err
}

func TestEffortRecoveryReconcilesLostDeliveryWithoutResending(t *testing.T) {
	for _, variant := range []string{"accepted-running", "accepted-finished", "missing", "unavailable"} {
		t.Run(variant, func(t *testing.T) {
			s, _, c, e, d := recoveryFixture(t)
			ctx := context.Background()
			d, err := s.transitionDirective(ctx, d, pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN, "simulate response lost after owner acceptance")
			if err != nil {
				t.Fatal(err)
			}
			reader := &recoveryReceiptController{fakeActionController: c, accepted: strings.HasPrefix(variant, "accepted")}
			if variant == "unavailable" {
				reader.err = errors.New("owner receipt unavailable")
			}
			s.controller = reader
			run := c.runs[uuid.MustParse(d.TargetRunId)]
			stamp := s.now().Add(time.Minute)
			run.LastHeartbeat = &stamp
			if variant == "accepted-finished" {
				run.Status, run.EndedAt = domain.RunStatusComplete, &stamp
			}
			observed := stamp.Add(time.Minute)
			s.now = func() time.Time { return observed }
			got, err := s.deliverDirective(ctx, d)
			if variant == "unavailable" && err == nil {
				t.Fatal("receipt outage hidden")
			}
			if variant != "unavailable" && err != nil {
				t.Fatal(err)
			}
			if c.continued != 1 {
				t.Fatal("lost response caused duplicate effects", c.continued)
			}
			if reader.accepted {
				if got.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED {
					t.Fatal(got)
				}
				observed = observed.Add(time.Second)
				_, err = s.UpdateDirective(ctx, &pb.UpdateEffortDirectiveRequest{DirectiveId: got.DirectiveId, ExpectedRevision: got.Revision, IdempotencyKey: "late-verify", RecoveryVerification: &pb.EffortRecoveryVerification{State: "progress-observed", Reason: "new retained regression receipt verifies assignment after original admission", EvidenceRefs: []string{"test-genie:post-recovery-regression"}, ObservedAt: timestamppb.New(observed)}}, EffortActor{ID: e.SupervisorRunId})
				if err != nil {
					t.Fatal("late reconciliation rejected genuine post-request progress", err)
				}
			} else if got.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN {
				t.Fatal("missing receipt invented delivery", got)
			}
		})
	}
}

func TestEffortRecoveryDeliveryIsNotProgress(t *testing.T) {
	s, r, c, e, d := recoveryFixture(t)
	if d.GetRecoveryExpectation().GetProgressCondition() == "" || d.GetRecoveryVerification().GetState() != "pending" {
		t.Fatal("delivery discarded progress contract or invented verification", d)
	}
	c.runs[uuid.MustParse(d.TargetRunId)].Status = domain.RunStatusRunning
	now := s.now()
	c.runs[uuid.MustParse(d.TargetRunId)].LastHeartbeat = &now
	restarted := NewEffortService(r, c, s.policies, s.config)
	restarted.now = s.now
	if err := restarted.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	board, err := restarted.Board(context.Background(), &pb.GetEffortBoardRequest{EffortRef: e.EffortRef})
	if err != nil || board.Rows[0].Directives[0].GetRecoveryVerification().GetState() != "pending" || c.continued != 1 {
		t.Fatal("restart/liveness retried effects or fabricated progress", board, err)
	}
}

func TestEffortRecoveryVerificationRequiresFreshEvidenceAndExactAuthority(t *testing.T) {
	for _, variant := range []string{"progress", "same-baseline", "no-evidence", "future", "before-delivery", "wrong-verifier", "withdrawn", "expired-grant", "wrong-target", "owner-unavailable", "active-ended", "future-heartbeat", "completed", "future-completion", "prior-completion", "failed-run", "owner-wait", "wait-without-condition", "failed-verification"} {
		t.Run(variant, func(t *testing.T) {
			s, r, c, e, d := recoveryFixture(t)
			ctx := context.Background()
			stamp := s.now().Add(time.Second)
			s.now = func() time.Time { return stamp }
			actor := EffortActor{ID: e.SupervisorRunId}
			v := &pb.EffortRecoveryVerification{State: "progress-observed", Reason: "owner regression receipt now passes the assigned check", EvidenceRefs: []string{"test-genie:regression-after"}, ObservedAt: timestamppb.New(stamp), Verifier: "forged-request-identity"}
			run := c.runs[uuid.MustParse(d.TargetRunId)]
			run.Status = domain.RunStatusRunning
			run.LastHeartbeat = &stamp
			switch variant {
			case "same-baseline":
				v.EvidenceRefs = d.RecoveryExpectation.GetBaselineEvidenceRefs()
			case "no-evidence":
				v.EvidenceRefs = nil
			case "future":
				v.ObservedAt = timestamppb.New(stamp.Add(time.Minute))
			case "before-delivery":
				v.ObservedAt = timestamppb.New(d.DeliveredAt.AsTime().Add(-time.Second))
			case "wrong-verifier":
				actor.ID = uuid.NewString()
			case "withdrawn":
				_, err := s.Withdraw(ctx, &pb.WithdrawEffortRequest{EffortRef: e.EffortRef, ExpectedRevision: e.Revision, Reason: "withdrawn", IdempotencyKey: "withdraw"}, EffortActor{ID: "owner", Operator: true})
				if err != nil {
					t.Fatal(err)
				}
			case "expired-grant":
				stamp = e.AuthorityExpiresAt.AsTime().Add(time.Second)
			case "wrong-target":
				e.TargetRevision = "amended"
				_, err := s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "amend"}, EffortActor{ID: "owner", Operator: true})
				if err != nil {
					t.Fatal(err)
				}
			case "owner-unavailable":
				delete(c.runs, run.ID)
			case "active-ended":
				run.EndedAt = &stamp
			case "future-heartbeat":
				future := stamp.Add(time.Minute)
				run.LastHeartbeat = &future
			case "completed", "future-completion", "prior-completion":
				run.Status = domain.RunStatusComplete
				ended := stamp
				if variant == "future-completion" {
					ended = stamp.Add(time.Minute)
				} else if variant == "prior-completion" {
					ended = d.DeliveredAt.AsTime().Add(-time.Second)
				}
				run.EndedAt = &ended
			case "failed-run":
				run.Status = domain.RunStatusFailed
			case "owner-wait":
				v.State = "owner-wait"
				v.NextOwnerCondition = "test-genie:run/wait-for-regression"
			case "wait-without-condition":
				v.State = "owner-wait"
			case "failed-verification":
				v.State = "failed"
				v.NextOwnerCondition = "assignment:repair-regression"
				run.Status = domain.RunStatusFailed
			}
			request := &pb.UpdateEffortDirectiveRequest{DirectiveId: d.DirectiveId, ExpectedRevision: d.Revision, IdempotencyKey: "verify", RecoveryVerification: v}
			got, err := s.UpdateDirective(ctx, request, actor)
			wantOK := variant == "progress" || variant == "completed" || variant == "owner-wait" || variant == "failed-verification"
			if !wantOK {
				if err == nil {
					t.Fatal("invalid verification accepted", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.GetRecoveryVerification().GetVerifier() != actor.ID || got.Assessment != "unknown" || !proto.Equal(got.SourceSnapshot, d.SourceSnapshot) {
				t.Fatal("verification lost attribution/baseline or invented causal benefit", got)
			}
			again, err := s.UpdateDirective(ctx, request, actor)
			if err != nil || !proto.Equal(got, again) || c.continued != 1 {
				t.Fatal("verification replay caused another recovery", again, err)
			}
			persisted, err := r.GetEffortDirective(ctx, d.DirectiveId)
			if err != nil || !proto.Equal(persisted, got) {
				t.Fatal("verification not durable", persisted, err)
			}
		})
	}
}
