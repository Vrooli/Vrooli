package supervision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func effortFixture(t *testing.T) (*EffortService, *Repository, *fakeActionController) {
	t.Helper()
	r, db := testRepository(t)
	c := &fakeActionController{runs: map[uuid.UUID]*domain.Run{}}
	s := NewEffortService(r, c, NewPolicyStore(db, nil), EffortDiscoveryConfig{Root: t.TempDir(), ScanLimit: 2})
	s.now = r.now
	return s, r, c
}

func TestEffortFreshSupervisorAssessmentBeforeDiscovery(t *testing.T) {
	for _, variant := range []string{"valid", "wrong-revision", "private", "inactive", "unverified", "worker", "wrong-run", "withdrawn"} {
		t.Run(variant, func(t *testing.T) {
			s, r, c := effortFixture(t)
			ctx := context.Background()
			e, err := s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: &pb.EffortEnrollment{EffortRef: "effort:new", TargetRevision: "target"}, IdempotencyKey: "enroll"}, EffortActor{ID: "owner", Operator: true})
			if err != nil {
				t.Fatal(err)
			}
			id := uuid.New()
			ref := &eventpb.WorkReference{Kind: "effort", Id: e.EffortRef, Revision: e.TargetRevision, Relationship: "supervisor", Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE}
			run := &domain.Run{ID: id, WorkReferences: []*eventpb.WorkReference{ref}}
			switch variant {
			case "wrong-revision":
				ref.Revision = "old"
			case "private":
				ref.Visibility = eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_UNSPECIFIED
			case "inactive":
				ref.State = eventpb.WorkReferenceState_WORK_REFERENCE_STATE_UNSPECIFIED
			case "unverified":
				ref.Verified = false
			case "worker":
				ref.Relationship = "worker"
			case "wrong-run":
				run.ID = uuid.New()
			case "withdrawn":
				_, err = s.Withdraw(ctx, &pb.WithdrawEffortRequest{EffortRef: e.EffortRef, ExpectedRevision: e.Revision, Reason: "retired", IdempotencyKey: "withdraw"}, EffortActor{ID: "owner", Operator: true})
				if err != nil {
					t.Fatal(err)
				}
			}
			c.runs[id] = run
			a, err := s.RecordAssessment(ctx, &pb.RecordEffortAssessmentRequest{Assessment: &pb.EffortAssessment{EffortRefs: []string{e.EffortRef}, TargetRevisions: map[string]string{e.EffortRef: e.TargetRevision}, Disposition: "unknown", Rationale: "source uncertainty assessed", EvidenceRefs: []string{"owner:board"}, IdempotencyKey: "fresh", SharedOperationRef: "wake:fresh"}}, EffortActor{ID: id.String()})
			if variant == "valid" {
				if err != nil || a.GetSupervisorRunId() != id.String() {
					t.Fatal(a, err)
				}
				current, _, readErr := r.GetEffort(ctx, e.EffortRef)
				if readErr != nil || current.Revision != e.Revision || len(current.Subjects) != 0 || current.AuthorizedBy != "owner" || len(current.PermittedActions) != 0 {
					t.Fatal("assessment altered enrollment or granted steering", current, readErr)
				}
			} else if err == nil {
				t.Fatalf("invalid membership %s accepted", variant)
			}
		})
	}
}

func TestEffortStableSupervisorDelegationRequiresVerifiedOwnerAndExactScope(t *testing.T) {
	s, _, c := effortFixture(t)
	ctx := context.Background()
	e := grantFixture(t, s, c, domain.RunStatusRunning)
	e.SupervisorRunId = ""
	e.SupervisorOwnerSubject = "owner:standing-supervision"
	e.SupervisorScope = "agent-manager:supervise"
	e, err := s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "stable-grant"}, EffortActor{ID: "owner", Operator: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, actor := range []EffortActor{
		{ID: uuid.NewString()},
		{ID: uuid.NewString(), OwnerSubject: "wrong-owner", Scopes: []string{e.SupervisorScope}},
		{ID: uuid.NewString(), OwnerSubject: e.SupervisorOwnerSubject, Scopes: []string{"*"}},
	} {
		if _, err = s.RequestDirective(ctx, directiveFixture(s, e), actor); err == nil {
			t.Fatal("unbound wake authorized", actor)
		}
	}
	for i := 0; i < 2; i++ {
		// Fresh wakes receive distinct AM-signed identities with the same
		// explicitly granted owner/scope, bounded by the credential expiry.
		expires := time.Now().Add(time.Hour)
		run := &domain.Run{ID: uuid.New(), TaskID: uuid.New(), OwnerSubject: e.SupervisorOwnerSubject, OwnerScopes: []string{e.SupervisorScope, "unrelated:scope"}, OwnerExpiresAt: &expires}
		secret := []byte("effort-wake-fixture-secret")
		token := phases.GenerateIdentityToken(ctx, phases.GenerateIdentityTokenInput{Run: run, Secret: secret, RequestedScopes: []string{e.SupervisorScope}})
		claims, verifyErr := identity.VerifyToken(token, secret)
		if verifyErr != nil || len(claims.Scopes) != 1 {
			t.Fatal("wake identity was not attenuated", verifyErr)
		}
		actor := EffortActor{ID: claims.RunID.String(), OwnerSubject: claims.Subject, Scopes: claims.Scopes}
		d, err := s.RequestDirective(ctx, directiveFixture(s, e), actor)
		if err != nil || d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_PENDING {
			t.Fatal("fresh scoped wake failed", d, err)
		}
	}
}

func TestEffortSubjectTriggerExcludesOwnAssessmentAndFreshSupervisorButTracksWorker(t *testing.T) {
	s, _, c := effortFixture(t)
	ctx := context.Background()
	e := grantFixture(t, s, c, domain.RunStatusRunning)
	board := func() *pb.EffortBoardRow {
		t.Helper()
		b, err := s.Board(ctx, &pb.GetEffortBoardRequest{EffortRef: e.EffortRef})
		if err != nil {
			t.Fatal(err)
		}
		return b.Rows[0]
	}
	before := board()
	_, err := s.RecordAssessment(ctx, &pb.RecordEffortAssessmentRequest{Assessment: &pb.EffortAssessment{
		EffortRefs: []string{e.EffortRef}, TargetRevisions: map[string]string{e.EffortRef: e.TargetRevision},
		Disposition: "quiet", Rationale: "healthy unchanged subject cut", EvidenceRefs: []string{"owner:healthy"},
		SourceLedgerRef: "source-ledger:quiet", SharedOperationRef: "pm:wake/quiet", IdempotencyKey: "quiet-trigger",
	}}, EffortActor{ID: e.SupervisorRunId})
	if err != nil {
		t.Fatal(err)
	}
	c.runs[uuid.MustParse(e.SupervisorRunId)].Status = domain.RunStatusComplete
	after := board()
	if before.ChangeIdentity != after.ChangeIdentity || before.VisibilityChangeIdentity == after.VisibilityChangeIdentity {
		t.Fatal("assessment caused subject feedback loop")
	}
	id := uuid.New()
	wake := &domain.Run{ID: id, Status: domain.RunStatusRunning, WorkReferences: []*eventpb.WorkReference{{Kind: "effort", Id: e.EffortRef, Relationship: "supervisor", Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE}}}
	c.runs[id] = wake
	s.SetRunRegistry(effortRegistryFixture{runs: []*domain.Run{wake}})
	if _, err = s.ReconcileDiscovery(ctx); err != nil {
		t.Fatal(err)
	}
	after = board()
	if before.ChangeIdentity != after.ChangeIdentity || after.Enrollment.AuthorizedBy == "" {
		t.Fatal("fresh observed wake changed subject trigger or revoked unchanged grant")
	}
	c.runs[uuid.MustParse(e.Subjects[0].RunId)].Status = domain.RunStatusNeedsReview
	if board().ChangeIdentity == before.ChangeIdentity {
		t.Fatal("external worker progress swallowed")
	}
}

func TestEffortRegistryKeepsSupervisorGrantAcrossCoordinatorReplacement(t *testing.T) {
	s, _, c := effortFixture(t)
	ctx := context.Background()
	e := grantFixture(t, s, c, domain.RunStatusRunning)
	replacement := uuid.New()
	c.runs[replacement] = &domain.Run{ID: replacement, Status: domain.RunStatusRunning, WorkReferences: []*eventpb.WorkReference{{
		Kind: "effort", Id: e.EffortRef, Revision: e.TargetRevision, Relationship: "orchestrator", Verified: true,
		Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC,
		State:      eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE,
	}}}
	s.SetRunRegistry(effortRegistryFixture{runs: []*domain.Run{c.runs[replacement]}})
	if _, err := s.ReconcileDiscovery(ctx); err != nil {
		t.Fatal(err)
	}
	current, _, err := s.repo.GetEffort(ctx, e.EffortRef)
	if err != nil {
		t.Fatal(err)
	}
	if current.AuthorizedBy == "" || len(current.PermittedActions) == 0 || current.AuthorityRef != e.AuthorityRef {
		t.Fatalf("coordinator replacement revoked the standing supervisor grant: %+v", current)
	}
	if len(current.Subjects) != 2 || current.Subjects[1].RunId != replacement.String() {
		t.Fatalf("replacement coordinator was not retained as runtime evidence: %+v", current.Subjects)
	}
}

func TestEffortWorkspaceEvidenceTimestampAndDerivedJoinsStayStable(t *testing.T) {
	s, r, c := effortFixture(t)
	ctx := context.Background()
	workspaceFixture(t, s.config.Root, "arbitrary", "effort:arbitrary", map[string]any{"target_revision": "", "execution": map[string]any{"approved_source_digest": "accepted-cut"}})
	if _, err := s.ReconcileDiscovery(ctx); err != nil {
		t.Fatal(err)
	}
	e, ob, err := r.GetEffort(ctx, "effort:arbitrary")
	if err != nil || e.TargetRevision != "accepted-cut" {
		t.Fatal(e, err)
	}
	stamp := ob.ObservedAt.AsTime()
	s.now = func() time.Time { return stamp.Add(10 * time.Minute) }
	b, err := s.Board(ctx, nil)
	if err != nil || b.Rows[0].Freshness != pb.EffortFreshness_EFFORT_FRESHNESS_STALE || !b.Rows[0].ObservedAt.AsTime().Equal(stamp) {
		t.Fatal("stale source stamped just now", b, err)
	}
	id := uuid.New()
	run := &domain.Run{ID: id, Status: domain.RunStatusRunning, WorkReferences: []*eventpb.WorkReference{{Kind: "effort", Id: e.EffortRef, Relationship: "supervisor", Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE}}}
	c.runs[id] = run
	s.SetRunRegistry(effortRegistryFixture{runs: []*domain.Run{run}})
	if _, err = s.ReconcileDiscovery(ctx); err != nil {
		t.Fatal(err)
	}
	joined, _, _ := r.GetEffort(ctx, e.EffortRef)
	if _, err = s.ReconcileDiscovery(ctx); err != nil {
		t.Fatal(err)
	}
	next, _, _ := r.GetEffort(ctx, e.EffortRef)
	if len(next.Subjects) != 1 || next.Revision != joined.Revision {
		t.Fatal("unchanged workspace erased/churned derived run join", next)
	}
}

func TestEffortDirectiveReservationsUseFullHistoryAndAtomicAdmission(t *testing.T) {
	for _, mode := range []string{"allowance", "cooldown"} {
		t.Run(mode, func(t *testing.T) {
			s, r, c := effortFixture(t)
			ctx := context.Background()
			e := grantFixture(t, s, c, domain.RunStatusRunning)
			if mode == "allowance" {
				e.MaximumDirectives = 1
			} else {
				e.CooldownSeconds = 300
			}
			var err error
			e, err = s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "limit"}, EffortActor{ID: "owner", Operator: true})
			if err != nil {
				t.Fatal(err)
			}
			first, err := s.RequestDirective(ctx, directiveFixture(s, e), EffortActor{ID: e.SupervisorRunId})
			if err != nil || first.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_PENDING {
				t.Fatal(first, err)
			}
			for i := 0; i < 105; i++ {
				d := &pb.EffortDirective{DirectiveId: fmt.Sprintf("00000000-0000-0000-0000-%012d", i), EffortRef: e.EffortRef, Revision: 1, Delivery: pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED}
				if err = r.SaveEffortDirective(ctx, d, 0, fmt.Sprintf("refused-%d", i), "fixture"); err != nil {
					t.Fatal(err)
				}
			}
			next, err := s.RequestDirective(ctx, directiveFixture(s, e), EffortActor{ID: e.SupervisorRunId})
			if err != nil || next.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED || !strings.Contains(next.DeliveryReason, mode) {
				t.Fatal("display sample bypassed reservation", next, err)
			}
		})
	}
	t.Run("concurrent services", func(t *testing.T) {
		s, r, c := effortFixture(t)
		ctx := context.Background()
		e := grantFixture(t, s, c, domain.RunStatusRunning)
		e.MaximumDirectives = 1
		e, err := s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "limit"}, EffortActor{ID: "owner", Operator: true})
		if err != nil {
			t.Fatal(err)
		}
		second := NewEffortService(r, c, s.policies, s.config)
		second.now = s.now
		var wg sync.WaitGroup
		var mu sync.Mutex
		accepted := 0
		for _, svc := range []*EffortService{s, second} {
			wg.Add(1)
			go func(svc *EffortService) {
				defer wg.Done()
				d, err := svc.RequestDirective(ctx, directiveFixture(svc, e), EffortActor{ID: e.SupervisorRunId})
				if err == nil && d.Delivery == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_PENDING {
					mu.Lock()
					accepted++
					mu.Unlock()
				}
			}(svc)
		}
		wg.Wait()
		if accepted != 1 {
			t.Fatalf("atomic reservation admitted %d requests", accepted)
		}
	})
}

func TestEffortSupersededUncertainDispatchNeverSendsAgain(t *testing.T) {
	s, r, c := effortFixture(t)
	ctx := context.Background()
	e := grantFixture(t, s, c, domain.RunStatusRunning)
	actor := EffortActor{ID: e.SupervisorRunId}
	old, err := s.RequestDirective(ctx, directiveFixture(s, e), actor)
	if err != nil {
		t.Fatal(err)
	}
	old, err = s.transitionDirective(ctx, old, pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN, "owner response lost")
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := s.RequestDirective(ctx, directiveFixture(s, e), actor)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := s.UpdateDirective(ctx, &pb.UpdateEffortDirectiveRequest{DirectiveId: old.DirectiveId, ExpectedRevision: old.Revision, IdempotencyKey: "supersede", SupersededBy: replacement.DirectiveId}, actor)
	if err != nil {
		t.Fatal(err)
	}
	c.runs[uuid.MustParse(old.TargetRunId)].Status = domain.RunStatusNeedsReview
	s.controller = &recoveryReceiptController{fakeActionController: c}
	// Deliberately give delivery the stale pre-supersession snapshot.
	if _, err = s.deliverDirective(ctx, old); err != nil {
		t.Fatal(err)
	}
	if c.continued != 0 || updated.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN || !strings.Contains(updated.DeliveryReason, "uncertain") {
		t.Fatal("superseded uncertain directive sent again", updated)
	}
	retained, _ := r.GetEffortDirective(ctx, old.DirectiveId)
	if retained.SupersededBy != replacement.DirectiveId {
		t.Fatal("supersession evidence lost")
	}
}

func TestEffortContradictoryRunStateKeepsAccountingUnknown(t *testing.T) {
	s, _, c := effortFixture(t)
	e := grantFixture(t, s, c, domain.RunStatusRunning)
	start := s.now().Add(-time.Hour)
	ended := s.now().Add(-time.Minute)
	heartbeat := s.now()
	run := c.runs[uuid.MustParse(e.Subjects[0].RunId)]
	run.StartedAt = &start
	run.EndedAt = &ended
	run.LastHeartbeat = &heartbeat
	run.ErrorMsg = "runner exited before terminal event"
	run.Summary = &domain.RunSummary{TokensUsed: 396462006}
	b, err := s.Board(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	a := b.Rows[0].Assignments[0]
	if a.RuntimeState != "unknown" || a.Usage.AgentSeconds != nil || a.Usage.Tokens != nil || a.UnavailableReason == "" || len(b.Rows[0].Blockers) == 0 {
		t.Fatal("contradictory lifecycle became settled state/accounting", a)
	}
	if c.continued != 0 || c.stopped != 0 {
		t.Fatal("observation repaired or replaced run")
	}
}

type effortRegistryFixture struct{ runs []*domain.Run }

func (f effortRegistryFixture) List(_ context.Context, filter repository.RunListFilter) ([]*domain.Run, error) {
	if filter.Offset >= len(f.runs) {
		return nil, nil
	}
	end := filter.Offset + filter.Limit
	if end > len(f.runs) {
		end = len(f.runs)
	}
	return f.runs[filter.Offset:end], nil
}

func TestEffortRuntimeSupervisorJoinsAndSharedQuietAssessmentIsRetainedOnce(t *testing.T) {
	s, r, c := effortFixture(t)
	ctx := context.Background()
	id := uuid.New()
	run := &domain.Run{ID: id, Status: domain.RunStatusRunning}
	c.runs[id] = run
	for _, ref := range []string{"effort:a", "effort:b"} {
		run.WorkReferences = append(run.WorkReferences, &eventpb.WorkReference{Kind: "effort", Id: ref, Relationship: "supervisor", Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE})
	}
	s.SetRunRegistry(effortRegistryFixture{runs: []*domain.Run{run}})
	if _, err := s.ReconcileDiscovery(ctx); err != nil {
		t.Fatal(err)
	}
	a := &pb.EffortAssessment{EffortRefs: []string{"effort:a", "effort:b"}, TargetRevisions: map[string]string{"effort:a": "", "effort:b": ""}, Disposition: "quiet", Rationale: "required owner waits remain valid; no independently evidenced deviation", EvidenceRefs: []string{"owner:wait/a", "owner:wait/b"}, SourceLedgerRef: "source-ledger:assessment/one", SharedOperationRef: "pm:wake/one", IdempotencyKey: "quiet-one"}
	req := &pb.RecordEffortAssessmentRequest{Assessment: a}
	saved, err := s.RecordAssessment(ctx, req, EffortActor{ID: id.String()})
	if err != nil {
		t.Fatal(err)
	}
	restarted := NewEffortService(r, c, s.policies, s.config)
	restarted.now = s.now
	b, err := restarted.Board(ctx, nil)
	if err != nil || len(b.Rows) != 2 {
		t.Fatal(b, err)
	}
	for _, row := range b.Rows {
		if row.LastAssessment.GetAssessmentId() != saved.AssessmentId || row.LastAssessment.GetBenefit() != "unknown" || row.LastAssessment.GetObservedUsage().ReportedCostUsd != nil || row.Assignments[0].Subject.Role != "supervisor" {
			t.Fatal("quiet assessment/unknown shared cost not integrated", row)
		}
	}
	replay, err := restarted.RecordAssessment(ctx, req, EffortActor{ID: id.String()})
	if err != nil || replay.AssessmentId != saved.AssessmentId {
		t.Fatal("shared operation duplicated", replay, err)
	}
	a.IdempotencyKey = "duplicate-operation"
	if _, err = restarted.RecordAssessment(ctx, req, EffortActor{ID: id.String()}); !errors.Is(err, ErrConflict) {
		t.Fatal("shared operation admitted twice", err)
	}
}

func TestEffortAssessmentRepairLinkValidationKeepsCanonicalAssignmentBounded(t *testing.T) {
	valid := &pb.EffortRepairLink{WorkRef: "swarm-manager:backlog/chore/adoption", AssigningOwnerRef: "owner:root", NextOperation: "vrooli scenario restart agent-manager", CompletionEvidenceRefs: []string{"test-genie:adoption"}, StoppingCondition: "stop after one attempt", State: "assigned"}
	for _, tc := range []struct {
		name  string
		links []*pb.EffortRepairLink
		ok    bool
	}{
		{name: "assigned", links: []*pb.EffortRepairLink{valid}, ok: true},
		{name: "resolved", links: []*pb.EffortRepairLink{{WorkRef: "swarm:item", AssigningOwnerRef: "owner", NextOperation: "verify", CompletionEvidenceRefs: []string{"proof"}, StoppingCondition: "stop", State: "resolved"}}, ok: true},
		{name: "needs_assignment", links: []*pb.EffortRepairLink{{AssigningOwnerRef: "owner:infra", NextOperation: "reconcile", StoppingCondition: "one escalation", State: "needs_assignment"}}, ok: true},
		{name: "needs_assignment_claims_work", links: []*pb.EffortRepairLink{{WorkRef: "swarm:item", AssigningOwnerRef: "owner", NextOperation: "reconcile", StoppingCondition: "stop", State: "needs_assignment"}}},
		{name: "assigned_without_proof", links: []*pb.EffortRepairLink{{WorkRef: "swarm:item", AssigningOwnerRef: "owner", NextOperation: "reconcile", StoppingCondition: "stop", State: "assigned"}}},
		{name: "duplicate_work", links: []*pb.EffortRepairLink{valid, valid}},
		{name: "unnamed_assigning_owner", links: []*pb.EffortRepairLink{{AssigningOwnerRef: " ", NextOperation: "reconcile", StoppingCondition: "one escalation", State: "needs_assignment"}}},
		{name: "empty_next_operation", links: []*pb.EffortRepairLink{{AssigningOwnerRef: "owner:infra", NextOperation: "\t", StoppingCondition: "one escalation", State: "needs_assignment"}}},
		{name: "unnamed_stopping_condition", links: []*pb.EffortRepairLink{{AssigningOwnerRef: "owner:infra", NextOperation: "reconcile", StoppingCondition: " ", State: "needs_assignment"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRepairLinks(tc.links)
			if (err == nil) != tc.ok {
				t.Fatalf("repair link validation ok=%t want=%t: %v", err == nil, tc.ok, err)
			}
		})
	}
}

func TestEffortActualScanClearsNotScannedButRetainsCurrentFailure(t *testing.T) {
	s, _, _ := effortFixture(t)
	workspaceFixture(t, s.config.Root, "good", "effort:good", nil)
	workspaceFixture(t, s.config.Root, "bad", "effort:bad", map[string]any{"schema_version": 99})
	d, err := s.ReconcileDiscovery(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !d.Partial || d.LastScanAt == nil {
		t.Fatal(d)
	}
	for _, f := range d.Findings {
		if f.Code == "not_scanned" {
			t.Fatal("bootstrap finding persisted after scan", d)
		}
	}
}

func workspaceFixture(t *testing.T, root, name, ref string, changes map[string]any) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	m := map[string]any{"schema_version": 1, "slug": name, "repository": "repo:test", "effort_ref": ref, "destination_ref": "doc:accepted-target", "target_revision": "accepted-1", "work_shape": "bounded_task", "owners": map[string]any{}}
	for k, v := range changes {
		m[k] = v
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "effort.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func grantFixture(t *testing.T, s *EffortService, c *fakeActionController, status domain.RunStatus) *pb.EffortEnrollment {
	t.Helper()
	id := uuid.New()
	c.runs[id] = &domain.Run{ID: id, Status: status}
	e := &pb.EffortEnrollment{EffortRef: "effort:" + uuid.NewString(), DisplayName: "Arbitrary bounded effort", DestinationRef: "doc:target", TargetRevision: "accepted-1", AuthorityRef: "grant:operator-accepted", SupervisorRunId: uuid.NewString(), Subjects: []*pb.EffortSubject{{Owner: "agent-manager", Kind: "run", Reference: id.String(), RunId: id.String(), Role: "orchestrator"}}, PermittedActions: []pb.WatchActionKind{pb.WatchActionKind_WATCH_ACTION_KIND_NUDGE}, MaximumDirectives: 5, AuthorityExpiresAt: timestamppb.New(s.now().Add(time.Hour))}
	supervisorID := uuid.MustParse(e.SupervisorRunId)
	c.runs[supervisorID] = &domain.Run{ID: supervisorID, Status: status}
	got, err := s.Enroll(context.Background(), &pb.EnrollEffortRequest{Enrollment: e, IdempotencyKey: uuid.NewString()}, EffortActor{ID: "owner", Operator: true})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func directiveFixture(s *EffortService, e *pb.EffortEnrollment) *pb.RequestEffortDirectiveRequest {
	return &pb.RequestEffortDirectiveRequest{ExpectedEnrollmentRevision: e.Revision, Directive: &pb.EffortDirective{EffortRef: e.EffortRef, TargetRevision: e.TargetRevision, TargetRunId: e.Subjects[0].RunId, Kind: pb.WatchActionKind_WATCH_ACTION_KIND_NUDGE, Scope: "orchestrator assignment only", EvidenceRefs: []string{"owner:validation-failure"}, Adjustment: "Reconcile the failed acceptance check with its assigned owner", ExpectedResult: "Evidence for the required outcome", ExpiresAt: timestamppb.New(s.now().Add(10 * time.Minute)), IdempotencyKey: uuid.NewString(), Hypothesis: "unchanged failing validation is avoidable", Comparison: "prior evidence cut versus owner repair result"}}
}

func TestEffortDiscoveryRotatesAcrossRestartAndSuppressesUnchangedCuts(t *testing.T) {
	s, r, c := effortFixture(t)
	ctx := context.Background()
	for _, name := range []string{"alpha", "beta", "gamma", "zeta"} {
		workspaceFixture(t, s.config.Root, name, "effort:"+name, nil)
	}
	first, err := s.ReconcileDiscovery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if first.ScannedCount != 2 || first.ScanCursor != "beta" || !first.Partial {
		t.Fatalf("bounded page=%v", first)
	}
	restart := NewEffortService(r, c, s.policies, s.config)
	restart.now = s.now
	second, err := restart.ReconcileDiscovery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	list, _ := restart.List(ctx, nil)
	if len(list.Efforts) != 4 || second.ScanCursor != "zeta" {
		t.Fatalf("restart starved later sources: %v %v", second, list)
	}
	third, err := restart.ReconcileDiscovery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fourth, err := restart.ReconcileDiscovery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if third.Generation != second.Generation || fourth.Generation != second.Generation || third.ChangeIdentity != fourth.ChangeIdentity {
		t.Fatalf("unchanged rotations changed generation: %v %v %v", second, third, fourth)
	}
	workspaceFixture(t, s.config.Root, "omega", "effort:new-shape", map[string]any{"work_shape": "investigation"})
	for i := 0; i < 3; i++ {
		if _, err = restart.ReconcileDiscovery(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err = r.GetEffort(ctx, "effort:new-shape"); err != nil {
		t.Fatal("new arbitrary effort was not discovered", err)
	}
}

func TestEffortDiscoveryPathSchemaConflictAndRemoval(t *testing.T) {
	s, r, _ := effortFixture(t)
	s.config.ScanLimit = 100
	ctx := context.Background()
	workspaceFixture(t, s.config.Root, "first", "effort:stable", nil)
	if _, err := s.ReconcileDiscovery(ctx); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(s.config.Root, "first"), filepath.Join(s.config.Root, "moved")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReconcileDiscovery(ctx); err != nil {
		t.Fatal(err)
	}
	e, _, _ := r.GetEffort(ctx, "effort:stable")
	if e.Workspace != "moved" {
		t.Fatal("move changed identity or failed", e)
	}
	workspaceFixture(t, s.config.Root, "duplicate", "effort:stable", nil)
	report, err := s.ReconcileDiscovery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, o, _ := r.GetEffort(ctx, e.EffortRef)
	if !report.Partial || o.Freshness != pb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE {
		t.Fatal("conflict not visible", report, o)
	}
	workspaceFixture(t, s.config.Root, "bad-version", "effort:bad", map[string]any{"schema_version": 99})
	workspaceFixture(t, s.config.Root, "unsafe", "effort:unsafe", map[string]any{"checkpoint": "../../credential.json"})
	if err = os.Symlink(t.TempDir(), filepath.Join(s.config.Root, "symlink")); err != nil {
		t.Fatal(err)
	}
	report, err = s.ReconcileDiscovery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) < 3 {
		t.Fatal("unsafe/incompatible sources accepted", report)
	}
	root, err := openSafeRoot(s.config.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err = safeRead(root, "../credentials", 100); err == nil {
		t.Fatal("path escape accepted")
	}
}

func TestEffortDiscoveryMissingAndMalformedNeverCompletes(t *testing.T) {
	s, r, _ := effortFixture(t)
	ctx := context.Background()
	workspaceFixture(t, s.config.Root, "one", "effort:one", nil)
	s.ReconcileDiscovery(ctx)
	if err := os.WriteFile(filepath.Join(s.config.Root, "one", "effort.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	s.ReconcileDiscovery(ctx)
	e, o, err := r.GetEffort(ctx, "effort:one")
	if err != nil || e.Withdrawn || o.Freshness != pb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE {
		t.Fatalf("malformed erased enrollment: %v %v %v", e, o, err)
	}
	if err = os.Rename(filepath.Join(s.config.Root, "one"), filepath.Join(t.TempDir(), "retained")); err != nil {
		t.Fatal(err)
	}
	s.ReconcileDiscovery(ctx)
	board, err := s.Board(ctx, nil)
	if err != nil || len(board.Rows) != 1 || board.Rows[0].OutcomeStanding.State == "accepted" {
		t.Fatal("removal inferred successful completion", board, err)
	}
}

func TestEffortBoardReadOnlyUnknownUsageAndDistinctOutcome(t *testing.T) {
	s, r, c := effortFixture(t)
	ctx := context.Background()
	workspaceFixture(t, s.config.Root, "one", "effort:one", nil)
	board, err := s.Board(ctx, nil)
	if err != nil || len(board.Rows) != 0 {
		t.Fatal("read implicitly discovered", board, err)
	}
	e := grantFixture(t, s, c, domain.RunStatusComplete)
	c.runs[uuid.MustParse(e.Subjects[0].RunId)].Summary = &domain.RunSummary{CostEstimate: 12}
	before, _, _ := r.GetEffort(ctx, e.EffortRef)
	board, err = s.Board(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	row := board.Rows[0]
	if row.RuntimeState != "finished" || row.OutcomeStanding.State != "unknown" || row.Usage.Tokens != nil || row.Usage.ReportedCostUsd != nil {
		t.Fatal("activity/cost confused with evidence", row)
	}
	after, _, _ := r.GetEffort(ctx, e.EffortRef)
	if before.Revision != after.Revision {
		t.Fatal("board mutated enrollment")
	}
	delete(c.runs, uuid.MustParse(e.Subjects[0].RunId))
	board, _ = s.Board(ctx, nil)
	if len(board.Rows) != 1 || board.Rows[0].Assignments[0].UnavailableReason == "" {
		t.Fatal("owner outage hid enrollment", board)
	}
}

func TestEffortDirectiveDeliveryAcknowledgmentAndAssessmentAreSeparate(t *testing.T) {
	s, r, c := effortFixture(t)
	ctx := context.Background()
	e := grantFixture(t, s, c, domain.RunStatusRunning)
	req := directiveFixture(s, e)
	actor := EffortActor{ID: e.SupervisorRunId}
	d, err := s.RequestDirective(ctx, req, actor)
	if err != nil || d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_PENDING || c.continued != 0 {
		t.Fatal("active turn interrupted", d, err)
	}
	c.runs[uuid.MustParse(d.TargetRunId)].Status = domain.RunStatusNeedsReview
	restarted := NewEffortService(r, c, s.policies, s.config)
	restarted.now = s.now
	if err = restarted.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	d, _ = r.GetEffortDirective(ctx, d.DirectiveId)
	if d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED || d.Acknowledgment != 0 || d.Assessment != "unknown" || c.continued != 1 {
		t.Fatal("delivery invented acknowledgment/benefit", d)
	}
	replay, err := restarted.RequestDirective(ctx, req, actor)
	if err != nil || replay.DirectiveId != d.DirectiveId || c.continued != 1 {
		t.Fatal("retry restarted work", replay, err)
	}
	snapshot := proto.Clone(d.SourceSnapshot)
	update := &pb.UpdateEffortDirectiveRequest{DirectiveId: d.DirectiveId, ExpectedRevision: d.Revision, IdempotencyKey: "defer", Acknowledgment: pb.EffortDirectiveAcknowledgment_EFFORT_DIRECTIVE_ACKNOWLEDGMENT_DEFERRED, Reason: "validation producer pending"}
	if _, err = restarted.UpdateDirective(ctx, update, EffortActor{ID: d.TargetRunId}); err == nil {
		t.Fatal("defer accepted without owner wait")
	}
	update.OwnerWaitRef = "test-genie:run/one"
	d, err = restarted.UpdateDirective(ctx, update, EffortActor{ID: d.TargetRunId})
	if err != nil {
		t.Fatal(err)
	}
	d, err = restarted.UpdateDirective(ctx, &pb.UpdateEffortDirectiveRequest{DirectiveId: d.DirectiveId, ExpectedRevision: d.Revision, IdempotencyKey: "challenge", Acknowledgment: pb.EffortDirectiveAcknowledgment_EFFORT_DIRECTIVE_ACKNOWLEDGMENT_CHALLENGED, Reason: "required validation is useful work", EvidenceRefs: []string{"validation:required"}}, EffortActor{ID: d.TargetRunId})
	if err != nil {
		t.Fatal(err)
	}
	d, err = restarted.UpdateDirective(ctx, &pb.UpdateEffortDirectiveRequest{DirectiveId: d.DirectiveId, ExpectedRevision: d.Revision, IdempotencyKey: "assess", Assessment: "contradicted", EvidenceRefs: []string{"comparison:required-validation"}}, actor)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(snapshot, d.SourceSnapshot) || d.SupervisionUsage.ReportedCostUsd != nil {
		t.Fatal("assessment overwrote input cut or invented cost")
	}
}

func TestEffortDirectiveAuthorityRevisionExpiryAndWithdrawal(t *testing.T) {
	for _, which := range []string{"unknown_grant", "wrong_subject", "revision", "expiry", "withdrawal", "disabled"} {
		t.Run(which, func(t *testing.T) {
			s, r, c := effortFixture(t)
			ctx := context.Background()
			e := grantFixture(t, s, c, domain.RunStatusRunning)
			req := directiveFixture(s, e)
			if which == "wrong_subject" {
				req.Directive.TargetRunId = uuid.NewString()
			}
			if which == "revision" {
				req.Directive.TargetRevision = "other"
			}
			if which == "unknown_grant" {
				e.AuthorizedBy = ""
				_, o, _ := r.GetEffort(ctx, e.EffortRef)
				e.Revision++
				if err := r.SaveEffort(ctx, e, o, e.Revision-1, "revoke", e.EffortRef); err != nil {
					t.Fatal(err)
				}
				req.ExpectedEnrollmentRevision = e.Revision
			}
			d, err := s.RequestDirective(ctx, req, EffortActor{ID: e.SupervisorRunId})
			if err != nil {
				t.Fatal(err)
			}
			if which == "expiry" {
				s.now = func() time.Time { return req.Directive.ExpiresAt.AsTime().Add(time.Second) }
			}
			if which == "withdrawal" {
				if _, err = s.Withdraw(ctx, &pb.WithdrawEffortRequest{EffortRef: e.EffortRef, ExpectedRevision: e.Revision, IdempotencyKey: "withdraw", Reason: "owner retired"}, EffortActor{ID: "owner", Operator: true}); err != nil {
					t.Fatal(err)
				}
			}
			if which == "disabled" {
				if _, err = s.policies.db.ExecContext(ctx, `INSERT INTO supervision_policy_control(singleton,disabled,reason,updated_by,updated_at) VALUES(1,1,'test','owner','2026-09-04T12:00:00Z')`); err != nil {
					t.Fatal(err)
				}
			}
			c.runs[uuid.MustParse(e.Subjects[0].RunId)].Status = domain.RunStatusNeedsReview
			if _, err = s.deliverDirective(ctx, d); err != nil {
				t.Fatal(err)
			}
			if c.continued != 0 {
				t.Fatal("refused/retired effect delivered")
			}
		})
	}
}

func TestEffortEnrollmentIdempotencyScopeAndWithdrawalSurviveRediscovery(t *testing.T) {
	s, r, c := effortFixture(t)
	ctx := context.Background()
	e := grantFixture(t, s, c, domain.RunStatusRunning)
	req := &pb.EnrollEffortRequest{Enrollment: proto.Clone(e).(*pb.EffortEnrollment), ExpectedRevision: e.Revision, IdempotencyKey: "amend"}
	req.Enrollment.TargetRevision = "accepted-2"
	if _, err := s.Enroll(ctx, req, EffortActor{ID: e.SupervisorRunId}); err == nil {
		t.Fatal("supervisor amended its own grant")
	}
	first, err := s.Enroll(ctx, req, EffortActor{ID: "owner", Operator: true})
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.Enroll(ctx, req, EffortActor{ID: "owner", Operator: true})
	if err != nil || first.Revision != again.Revision {
		t.Fatal("enrollment retry changed revision", err)
	}
	req.Enrollment.TargetRevision = "accepted-3"
	if _, err = s.Enroll(ctx, req, EffortActor{ID: "owner", Operator: true}); !errors.Is(err, ErrConflict) {
		t.Fatal("changed request reused key", err)
	}
	workspaceFixture(t, s.config.Root, "retired", "effort:retired", nil)
	s.ReconcileDiscovery(ctx)
	discovered, _, _ := r.GetEffort(ctx, "effort:retired")
	s.Withdraw(ctx, &pb.WithdrawEffortRequest{EffortRef: discovered.EffortRef, ExpectedRevision: discovered.Revision, IdempotencyKey: "retire", Reason: "explicit closure"}, EffortActor{ID: "owner", Operator: true})
	s.ReconcileDiscovery(ctx)
	discovered, _, _ = r.GetEffort(ctx, discovered.EffortRef)
	if !discovered.Withdrawn {
		t.Fatal("rediscovery revived retired effort")
	}
}

func TestEffortMetadataReconciliationIsScopedAndPreservesAuthority(t *testing.T) {
	s, r, c := effortFixture(t)
	ctx := context.Background()
	e := grantFixture(t, s, c, domain.RunStatusRunning)
	e.DispatchAuthorization = &pb.SupervisorDispatchAuthorization{AuthorizationId: "dispatch-1", TargetRevision: e.TargetRevision, MaximumRuns: 3}
	_, observation, err := r.GetEffort(ctx, e.EffortRef)
	if err != nil {
		t.Fatal(err)
	}
	if err = r.SaveEffort(ctx, e, observation, e.Revision, "fixture-dispatch", "fixture-dispatch"); err != nil {
		t.Fatal(err)
	}

	request := func(key string) *pb.ReconcileEffortMetadataRequest {
		enrollment := proto.Clone(e).(*pb.EffortEnrollment)
		// Metadata requests omit dispatch state; the server retains its stored
		// authorization rather than accepting a caller-supplied replacement.
		enrollment.DispatchAuthorization = nil
		return &pb.ReconcileEffortMetadataRequest{
			Enrollment:       enrollment,
			ExpectedRevision: e.Revision,
			IdempotencyKey:   key,
		}
	}
	request("worker").Enrollment.DisplayName = "worker attempt"
	if _, err = s.ReconcileMetadata(ctx, request("worker"), EffortActor{ID: uuid.NewString(), MetadataReconciler: true, Scopes: []string{EffortMetadataReconcileScope}}); err == nil {
		t.Fatal("unrelated run reconciled effort metadata")
	}

	coordinator := EffortActor{ID: e.Subjects[0].RunId, MetadataReconciler: true, Scopes: []string{EffortMetadataReconcileScope}}
	coordinatorRequest := request("coordinator")
	coordinatorRequest.Enrollment.DisplayName = "reconciled metadata"
	updated, err := s.ReconcileMetadata(ctx, coordinatorRequest, coordinator)
	if err != nil {
		t.Fatal(err)
	}
	if updated.DisplayName != "reconciled metadata" || updated.Revision != e.Revision+1 || updated.DispatchAuthorization == nil || updated.DispatchAuthorization.AuthorizationId != "dispatch-1" {
		t.Fatalf("metadata reconciliation lost retained state: %+v", updated)
	}

	replay, err := s.ReconcileMetadata(ctx, coordinatorRequest, coordinator)
	if err != nil || replay.Revision != updated.Revision {
		t.Fatalf("metadata reconciliation was not idempotent: revision=%d err=%v", replay.GetRevision(), err)
	}

	forged := request("forged-authority")
	forged.Enrollment.AuthorityRef = "forged"
	if _, err = s.ReconcileMetadata(ctx, forged, coordinator); err == nil {
		t.Fatal("metadata reconciliation accepted an authority change")
	}
	retargeted := request("retargeted")
	retargeted.Enrollment.TargetRevision = "accepted-2"
	if _, err = s.ReconcileMetadata(ctx, retargeted, coordinator); err == nil {
		t.Fatal("metadata reconciliation retargeted an actively supervised effort")
	}

	stale := request("stale")
	stale.ExpectedRevision = e.Revision
	if _, err = s.ReconcileMetadata(ctx, stale, EffortActor{ID: "owner", Operator: true}); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale metadata reconciliation returned %v, want conflict", err)
	}
}

func TestEffortDeclaredCheckpointKeepsIndependentBlocker(t *testing.T) {
	s, _, _ := effortFixture(t)
	workspaceFixture(t, s.config.Root, "mixed", "effort:mixed", map[string]any{"checkpoint": "handoffs/current.json"})
	dir := filepath.Join(s.config.Root, "mixed", "handoffs")
	os.Mkdir(dir, 0o700)
	os.WriteFile(filepath.Join(dir, "current.json"), []byte(`{"next_action":"investigate outcome B","rationale":"fresh acceptance failure","blockers":["outcome B failed"],"pending_operations":["test-genie:A"]}`), 0o600)
	if _, err := s.ReconcileDiscovery(context.Background()); err != nil {
		t.Fatal(err)
	}
	board, _ := s.Board(context.Background(), nil)
	if len(board.Rows) != 1 || !strings.Contains(board.Rows[0].NextAction, "outcome B") || len(board.Rows[0].Blockers) != 1 {
		t.Fatal("independent deviation masked", board)
	}
}

func legacyDriverFixture(t *testing.T, root, name, body string) string {
	t.Helper()
	dir := filepath.Join(root, name, "handoffs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEffortLegacyAdapterDiscoversArbitraryDriversWithAndWithoutManifest(t *testing.T) {
	s, r, c := effortFixture(t)
	ctx := context.Background()
	worker := uuid.New()
	c.runs[worker] = &domain.Run{ID: worker, Status: domain.RunStatusRunning}
	workspaceFixture(t, s.config.Root, "copper-kite", "effort:canonical-kite", map[string]any{"destination_ref": "", "target_revision": ""})
	legacyDriverFixture(t, s.config.Root, "copper-kite", fmt.Sprintf(`{"schema_version":1,"effort":"driver label unrelated to directory","next_action":"wait for required validation","loop_state":"waiting","children":{"worker":{"run_id":%q,"execution_id":"not-a-run","status":"complete","attempt":4}},"authorized_by":"forged","target_revision":"not-accepted"}`, worker.String()))
	legacyDriverFixture(t, s.config.Root, "violet-bridge", `{"effort":"independent investigation","next_action":"inspect evidence gap","orchestrator":{"loop_state":"waiting on owner"},"assignment":"../../credentials/must-not-read","children":{"missing":{"execution_id":"not-an-AM-run"}}}`)
	first, err := s.ReconcileDiscovery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Board(ctx, nil)
	if err != nil || len(b.Rows) != 2 {
		t.Fatal("arbitrary discovery missing rows", b, err)
	}
	for _, row := range b.Rows {
		if row.Enrollment.AuthorizedBy != "" || len(row.Enrollment.PermittedActions) != 0 || row.Enrollment.TargetRevision != "" || row.Enrollment.DestinationRef != "" || row.OutcomeStanding.State != "unknown" || row.Usage.Tokens != nil || row.Usage.ReportedCostUsd != nil {
			t.Fatal("driver declaration invented grant, acceptance or costs", row)
		}
		if row.NextAction == "" || !strings.Contains(row.Rationale, "self-report") || len(row.EvidenceRefs) == 0 {
			t.Fatal("names-only discovery", row)
		}
		if row.Enrollment.EffortRef == "effort:canonical-kite" {
			if len(row.Assignments) != 1 || row.Assignments[0].Subject.RunId != worker.String() || row.Assignments[0].RuntimeState != "running" {
				t.Fatal("child declaration replaced AM state", row)
			}
		} else if row.Enrollment.EffortRef != "legacy-effort:independent investigation" || len(row.Assignments) != 0 || !strings.Contains(strings.Join(row.Limitations, " "), "run reference is unknown") {
			t.Fatal("manifestless source not attributed", row)
		}
	}
	second, err := s.ReconcileDiscovery(ctx)
	if err != nil || second.Generation != first.Generation {
		t.Fatal("unchanged driver cut churned", first, second, err)
	}
	legacyDriverFixture(t, s.config.Root, "violet-bridge", `{"effort":"independent investigation","next_action":"reconcile changed evidence","loop_state":"reviewing"}`)
	if _, err = s.ReconcileDiscovery(ctx); err != nil {
		t.Fatal(err)
	}
	_, observation, err := r.GetEffort(ctx, "legacy-effort:independent investigation")
	if err != nil || observation.NextAction != "reconcile changed evidence" {
		t.Fatal("driver progress not observed", observation, err)
	}
}

func TestEffortLegacyAdapterPrefersDeclaredCheckpointAndExactSubjects(t *testing.T) {
	s, _, _ := effortFixture(t)
	id := uuid.NewString()
	workspaceFixture(t, s.config.Root, "silver-orbit", "effort:declared", map[string]any{"checkpoint": "handoffs/current.json", "supervision": map[string]any{"subjects": []any{map[string]any{"owner": "agent-manager", "kind": "run", "reference": id, "run_id": id, "role": "orchestrator"}}}})
	legacyDriverFixture(t, s.config.Root, "silver-orbit", `{"effort":"ignored fallback","next_action":"wrong action","loop_state":"wrong loop"}`)
	declared := fmt.Sprintf(`{"effort":"declared driver","next_action":"declared action","orchestrator":{"loop_state":"declared wait"},"children":{"duplicate":{"run_id":%q}}}`, id)
	path := filepath.Join(s.config.Root, "silver-orbit", "handoffs", "current.json")
	if err := os.WriteFile(path, []byte(declared), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReconcileDiscovery(context.Background()); err != nil {
		t.Fatal(err)
	}
	b, err := s.Board(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	row := b.Rows[0]
	if row.NextAction != "declared action" || !strings.Contains(row.Rationale, "declared wait") || len(row.Enrollment.Subjects) != 1 || row.Enrollment.Subjects[0].Role != "orchestrator" {
		t.Fatal("fallback/child overrode explicit declaration", row)
	}
	if strings.Contains(strings.Join(row.EvidenceRefs, " "), "/state.json@") {
		t.Fatal("read undeclared fallback despite explicit checkpoint")
	}
}

func TestEffortLegacyAdapterPreservesMissingAndMalformedUncertainty(t *testing.T) {
	for _, manifest := range []bool{false, true} {
		t.Run(fmt.Sprint("manifest=", manifest), func(t *testing.T) {
			s, r, _ := effortFixture(t)
			ctx := context.Background()
			ref := "legacy-effort:recoverable"
			if manifest {
				ref = "effort:recoverable"
				workspaceFixture(t, s.config.Root, "amber-river", ref, nil)
			}
			path := legacyDriverFixture(t, s.config.Root, "amber-river", `{"effort":"recoverable","next_action":"wait for owner","loop_state":"waiting"}`)
			if _, err := s.ReconcileDiscovery(ctx); err != nil {
				t.Fatal(err)
			}
			for _, broken := range []string{"missing", "malformed"} {
				if broken == "missing" {
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
				} else {
					legacyDriverFixture(t, s.config.Root, "amber-river", "{broken")
				}
				d, err := s.ReconcileDiscovery(ctx)
				if err != nil {
					t.Fatal(err)
				}
				e, o, err := r.GetEffort(ctx, ref)
				if err != nil || e.Withdrawn || o.Freshness != pb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE || !d.Partial || len(d.Findings) == 0 {
					t.Fatal("lost driver became success or disappeared", e, o, d, err)
				}
				if _, err = s.ReconcileDiscovery(ctx); err != nil {
					t.Fatal(err)
				}
				_, again, err := r.GetEffort(ctx, ref)
				if err != nil || again.Freshness != pb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE {
					t.Fatal("repeated missing/malformed scan cleared uncertainty", again, err)
				}
			}
		})
	}
	// A missing manifest may use the fallback without duplicating its retained identity.
	s, _, _ := effortFixture(t)
	workspaceFixture(t, s.config.Root, "retained", "effort:retained", nil)
	legacyDriverFixture(t, s.config.Root, "retained", `{"effort":"different label","loop_state":"waiting"}`)
	if _, err := s.ReconcileDiscovery(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(s.config.Root, "retained", "effort.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReconcileDiscovery(context.Background()); err != nil {
		t.Fatal(err)
	}
	b, err := s.Board(context.Background(), nil)
	if err != nil || len(b.Rows) != 1 || b.Rows[0].Enrollment.EffortRef != "effort:retained" || b.Rows[0].Enrollment.TargetRevision != "" {
		t.Fatal("manifest loss duplicated identity or preserved approval claim", b, err)
	}
}

func TestEffortLegacyAdapterBoundsAndManifestFailureAreNotBypassed(t *testing.T) {
	for _, mode := range []string{"version", "shape", "bytes", "symlink", "manifest"} {
		t.Run(mode, func(t *testing.T) {
			s, _, _ := effortFixture(t)
			body := `{"effort":"arbitrary","loop_state":"waiting"}`
			switch mode {
			case "version":
				body = `{"schema_version":2,"effort":"arbitrary","loop_state":"waiting"}`
			case "shape":
				body = `{"effort":"arbitrary","loop_state":{"state":"not-a-string"}}`
			case "bytes":
				body = strings.Repeat(" ", int(s.config.FileBytes)+1)
			case "manifest":
				workspaceFixture(t, s.config.Root, "sandboxed", "effort:no-bypass", map[string]any{"schema_version": 99})
			}
			path := legacyDriverFixture(t, s.config.Root, "sandboxed", body)
			if mode == "symlink" {
				outside := filepath.Join(t.TempDir(), "credential-file")
				if err := os.WriteFile(outside, []byte(body), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, path); err != nil {
					t.Fatal(err)
				}
			}
			d, err := s.ReconcileDiscovery(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			b, err := s.Board(context.Background(), nil)
			if err != nil || len(b.Rows) != 0 || !d.Partial || len(d.Findings) == 0 {
				t.Fatal("unsafe/incompatible source became a fresh row", b, d, err)
			}
		})
	}
}
