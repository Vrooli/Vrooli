package supervision

import (
	"context"
	"testing"

	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
)

func TestTeamEnrollmentPreservesOwnershipWithdrawalAndClock(t *testing.T) {
	s, repo, _ := effortFixture(t)
	ctx := context.Background()
	actor := EffortActor{ID: "creator", Team: &TeamEffortAuthority{TeamID: "owned", MemberRef: "owned/leader", HomePath: "delivery/owned"}}
	e := &pb.EffortEnrollment{EffortRef: "effort:owned", WorkShape: "orchestrated_epochs", DestinationRef: "path:delivery/owned/CHARTER.md", TargetRevision: "target", SourceRevision: "source", AuthorityRef: "instruction", Subjects: []*pb.EffortSubject{{Owner: "prompt-manager", Kind: "team-member", Reference: "owned/leader", Role: "orchestrator"}}}
	current, err := s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: e, IdempotencyKey: "prepare"}, actor)
	if err != nil {
		t.Fatal(err)
	}
	next := proto.Clone(current).(*pb.EffortEnrollment)
	next.Delivery = &pb.EffortDelivery{CommissionId: "commission", State: "running", BudgetSeconds: 3600, MaxWorkers: 2}
	current, err = s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: next, ExpectedRevision: current.Revision, IdempotencyKey: "start"}, actor)
	if err != nil || current.Delivery.StartedAt == nil {
		t.Fatalf("creator commission: %v", err)
	}
	for _, tc := range []struct {
		name  string
		edit  func(*pb.EffortEnrollment)
		actor EffortActor
	}{
		{name: "different owner", actor: EffortActor{ID: "other", Team: actor.Team}, edit: func(*pb.EffortEnrollment) {}},
		{name: "metadata replacement", actor: actor, edit: func(e *pb.EffortEnrollment) { e.TargetRevision = "replacement" }},
		{name: "budget reset", actor: actor, edit: func(e *pb.EffortEnrollment) { e.Delivery.BudgetSeconds++ }},
		{name: "subject replacement", actor: actor, edit: func(e *pb.EffortEnrollment) { e.Subjects[0].Reference = "owned/intruder" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			next := proto.Clone(current).(*pb.EffortEnrollment)
			tc.edit(next)
			if _, err := s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: next, ExpectedRevision: current.Revision, IdempotencyKey: tc.name}, tc.actor); err == nil {
				t.Fatal("team authority changed protected state")
			}
			stored, _, err := repo.GetEffort(ctx, current.EffortRef)
			if err != nil || !proto.Equal(stored, current) {
				t.Fatal("refusal mutated enrollment", err)
			}
		})
	}
	next = proto.Clone(current).(*pb.EffortEnrollment)
	next.Delivery.State = "paused"
	next.Delivery.Reason = "owner pause"
	current, err = s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: next, ExpectedRevision: current.Revision, IdempotencyKey: "pause"}, actor)
	if err != nil || current.Delivery.PausedAt == nil {
		t.Fatalf("creator pause: %v", err)
	}
	inert, inertRepo := s, repo
	e = proto.Clone(e).(*pb.EffortEnrollment)
	e.EffortRef = "effort:withdraw"
	current, err = inert.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: e, IdempotencyKey: "inert-prepare"}, actor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inert.Withdraw(ctx, &pb.WithdrawEffortRequest{EffortRef: current.EffortRef, ExpectedRevision: current.Revision, IdempotencyKey: "withdraw", Reason: "owner withdrawal"}, EffortActor{ID: "owner", Operator: true}); err != nil {
		t.Fatal(err)
	}
	withdrawn, _, _ := inertRepo.GetEffort(ctx, current.EffortRef)
	if _, err := inert.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: withdrawn, ExpectedRevision: withdrawn.Revision, IdempotencyKey: "revive"}, actor); err == nil {
		t.Fatal("creator resurrected withdrawn effort")
	}
}
