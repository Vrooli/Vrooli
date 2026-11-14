package supervision

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func deliveryFixture(t *testing.T) (*EffortService, *Repository, *fakeActionController, *pb.EffortEnrollment, *time.Time) {
	t.Helper()
	s, r, c := effortFixture(t)
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	r.now = s.now
	s.config.ProjectRoot = t.TempDir()
	home := filepath.Join(s.config.ProjectRoot, "delivery")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"CHARTER.md": "# Product\n- Destination: accepted surfaces >= 2\n- Budget: 1h\n", "STATUS.md": "# Delivery\n\n## Scoreboard\n\n| surface | target | state | owner |\n|---|---|---|---|\n| Inbox | selected | close | - |\n| Settings | selected | unassessed | - |\n\n## Handoff\n\nPartial product; no independent acceptance.\n\n## Needs operator\n\n## Next\n\n- [ready] Complete settings.\n", "LOG.md": "# Log\n\n## Accepted units\n\n(none)\n", "FEEDBACK.md": "# Feedback\n\n## Resolved\n"} {
		if err := os.WriteFile(filepath.Join(home, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	e, err := s.Enroll(context.Background(), &pb.EnrollEffortRequest{Enrollment: &pb.EffortEnrollment{EffortRef: "effort:delivery", DestinationRef: "path:delivery/CHARTER.md", TargetRevision: "product-1", Delivery: &pb.EffortDelivery{CommissionId: "disposable", BudgetSeconds: 3600, MaxWorkers: 2, State: "running", StartedAt: timestamppb.New(now.Add(-24 * time.Hour))}}, IdempotencyKey: "start"}, EffortActor{ID: "owner", Operator: true})
	if err != nil {
		t.Fatal(err)
	}
	return s, r, c, e, &now
}

func TestDeliveryPressureBoundariesAndPause(t *testing.T) {
	start := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	d := &pb.EffortDelivery{CommissionId: "test", StartedAt: timestamppb.New(start), BudgetSeconds: 3600, State: "running"}
	for _, tc := range []struct {
		seconds int
		phase   string
	}{{0, "ON TRACK"}, {2699, "ON TRACK"}, {2700, "APPROACHING"}, {3600, "OVER"}, {5400, "CONVERGE"}, {7200, "HARD STOP"}} {
		t.Run(tc.phase+(time.Duration(tc.seconds)*time.Second).String(), func(t *testing.T) {
			if p := DeliveryPressure(d, start.Add(time.Duration(tc.seconds)*time.Second)); p.Phase != tc.phase {
				t.Fatalf("phase=%s want %s", p.Phase, tc.phase)
			}
		})
	}
	d.State = "paused"
	d.PausedAt = timestamppb.New(start.Add(30 * time.Minute))
	p := DeliveryPressure(d, start.Add(10*time.Hour))
	if p.Phase != "PAUSED" || p.ElapsedSeconds != 1800 {
		t.Fatalf("paused clock moved: %+v", p)
	}
}

func TestDeliveryClockAuthorityAndRestart(t *testing.T) {
	s, r, c, e, now := deliveryFixture(t)
	ctx := context.Background()
	if !e.Delivery.StartedAt.AsTime().Equal(*now) {
		t.Fatal("caller supplied clock was accepted")
	}
	*now = now.Add(30 * time.Minute)
	e.Delivery.State = "paused"
	e.Delivery.Reason = "deliberate operator pause"
	e, err := s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "pause"}, EffortActor{ID: "owner", Operator: true})
	if err != nil {
		t.Fatal(err)
	}
	*now = now.Add(10 * time.Hour)
	restart := NewEffortService(r, c, s.config)
	restart.now = s.now
	e, _, _ = r.GetEffort(ctx, e.EffortRef)
	if p := DeliveryPressure(e.Delivery, *now); p.ElapsedSeconds != 1800 {
		t.Fatal(p)
	}
	e.Delivery.State = "running"
	e, err = restart.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "resume"}, EffortActor{ID: "owner", Operator: true})
	if err != nil {
		t.Fatal(err)
	}
	if e.Delivery.PausedSeconds != 36000 || DeliveryPressure(e.Delivery, *now).ElapsedSeconds != 1800 {
		t.Fatalf("restart lost paused time: %+v", e.Delivery)
	}
	amended := proto.Clone(e).(*pb.EffortEnrollment)
	amended.Delivery.BudgetSeconds++
	if _, err := restart.ReconcileMetadata(ctx, &pb.ReconcileEffortMetadataRequest{Enrollment: amended, ExpectedRevision: e.Revision, IdempotencyKey: "forge-clock"}, EffortActor{ID: "owner", Operator: true}); err == nil {
		t.Fatal("metadata altered the clock")
	}
	if _, err := restart.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: amended, ExpectedRevision: e.Revision, IdempotencyKey: "reset-budget"}, EffortActor{ID: "owner", Operator: true}); err == nil {
		t.Fatal("existing commission reset its budget")
	}
}

func TestDelegatedDeliveryRetainsOwnerAndWithdrawalFences(t *testing.T) {
	ctx := context.Background()
	actor := EffortActor{ID: "session-agent", DeliveryManager: true, ApprovedBy: "owner"}
	t.Run("retained running owner", func(t *testing.T) {
		s, r, c := effortFixture(t)
		e := enrollmentFixture(t, s, c, domain.RunStatusRunning)
		next := proto.Clone(e).(*pb.EffortEnrollment)
		next.Delivery = &pb.EffortDelivery{CommissionId: "new", BudgetSeconds: 3600, MaxWorkers: 2, State: "running"}
		if _, err := s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: next, ExpectedRevision: e.Revision, IdempotencyKey: "retained-owner"}, actor); err == nil {
			t.Fatal("delivery grant replaced a live retained owner")
		}
		stored, _, _ := r.GetEffort(ctx, e.EffortRef)
		if !proto.Equal(stored, e) {
			t.Fatal("refused commission changed enrollment")
		}
	})
	t.Run("withdrawn enrollment", func(t *testing.T) {
		s, _, c := effortFixture(t)
		e := enrollmentFixture(t, s, c, domain.RunStatusComplete)
		withdrawn, err := s.Withdraw(ctx, &pb.WithdrawEffortRequest{EffortRef: e.EffortRef, ExpectedRevision: e.Revision, IdempotencyKey: "withdraw", Reason: "owner decision"}, EffortActor{ID: "owner", Operator: true})
		if err != nil {
			t.Fatal(err)
		}
		withdrawn.Delivery = &pb.EffortDelivery{CommissionId: "new", BudgetSeconds: 3600, MaxWorkers: 2, State: "running"}
		if _, err := s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: withdrawn, ExpectedRevision: withdrawn.Revision, IdempotencyKey: "resurrect"}, actor); err == nil {
			t.Fatal("delivery grant resurrected a withdrawn effort")
		}
	})
}

func deliveryReference(role string) *eventpb.WorkReference {
	return &eventpb.WorkReference{Kind: "effort", Id: "effort:delivery", Revision: "product-1", Relationship: role, Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE}
}

func TestDeliveryBoundedWorkersAndOneOrchestrator(t *testing.T) {
	s, _, c, e, _ := deliveryFixture(t)
	ctx := context.Background()
	parent := &domain.Run{ID: uuid.New(), Status: domain.RunStatusRunning, ResolvedConfig: &domain.RunConfig{DeliveryCommissions: map[string]string{e.EffortRef: e.Delivery.CommissionId}}, WorkReferences: []*eventpb.WorkReference{deliveryReference("orchestrator")}}
	for i := 0; i < 3; i++ {
		bindings, text, release, err := s.AcquireDeliveryAdmission(ctx, []*eventpb.WorkReference{deliveryReference("worker")}, parent)
		if i == 2 {
			if err == nil {
				release()
				t.Fatal("third worker exceeded the limit")
			}
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if bindings[e.EffortRef] != "disposable" || !strings.Contains(text, "ON TRACK") {
			t.Fatal("missing authoritative context/binding")
		}
		release()
		id := uuid.New()
		run := &domain.Run{ID: id, Status: domain.RunStatusRunning, WorkReferences: []*eventpb.WorkReference{deliveryReference("worker")}, ResolvedConfig: &domain.RunConfig{DeliveryCommissions: bindings}}
		c.runs[id] = run
		e.Subjects = append(e.Subjects, &pb.EffortSubject{Owner: "agent-manager", Kind: "run", RunId: id.String(), Reference: id.String(), Role: "worker"})
		var err2 error
		e, err2 = s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: uuid.NewString()}, EffortActor{ID: "owner", Operator: true})
		if err2 != nil {
			t.Fatal(err2)
		}
	}
	c.runs[parent.ID] = parent
	e.Subjects = append(e.Subjects, &pb.EffortSubject{Owner: "agent-manager", Kind: "run", RunId: parent.ID.String(), Reference: parent.ID.String(), Role: "orchestrator"})
	if _, err := s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "retain-parent"}, EffortActor{ID: "owner", Operator: true}); err != nil {
		t.Fatal(err)
	}
	if _, _, release, err := s.AcquireDeliveryAdmission(ctx, []*eventpb.WorkReference{deliveryReference("orchestrator")}, nil); err == nil {
		release()
		t.Fatal("second orchestrator admitted")
	}
	parent.ResolvedConfig.DeliveryCommissions[e.EffortRef] = "old-commission"
	if _, err := s.DeliveryRunContext(ctx, parent); err == nil {
		t.Fatal("an old commission owner resumed")
	}
	parent.ResolvedConfig = nil
	if _, err := s.DeliveryRunContext(ctx, parent); err == nil {
		t.Fatal("an unbound historical owner joined the new commission")
	}
	if _, _, release, err := s.AcquireDeliveryAdmission(ctx, []*eventpb.WorkReference{deliveryReference("worker")}, parent); err == nil {
		release()
		t.Fatal("old parent joined new commission")
	}
}

func TestDeliveryHardStopCheckpointAndDurableHandback(t *testing.T) {
	s, r, c, e, now := deliveryFixture(t)
	ctx := context.Background()
	id := uuid.New()
	run := &domain.Run{ID: id, Status: domain.RunStatusRunning, ResolvedConfig: &domain.RunConfig{DeliveryCommissions: map[string]string{e.EffortRef: e.Delivery.CommissionId}}, WorkReferences: []*eventpb.WorkReference{deliveryReference("orchestrator")}}
	c.runs[id] = run
	e.Subjects = []*pb.EffortSubject{{Owner: "agent-manager", Kind: "run", RunId: id.String(), Reference: id.String(), Role: "orchestrator"}}
	e, err := s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "retain"}, EffortActor{ID: "owner", Operator: true})
	if err != nil {
		t.Fatal(err)
	}
	*now = now.Add(2 * time.Hour)
	if _, _, release, err := s.AcquireDeliveryAdmission(ctx, []*eventpb.WorkReference{deliveryReference("worker")}, run); err == nil {
		release()
		t.Fatal("hard deadline admitted work before scheduler tick")
	}
	if err := s.tickDelivery(ctx); err != nil {
		t.Fatal(err)
	}
	if c.stopped != 0 || c.continued != 1 {
		t.Fatalf("checkpoint not requested before stop: continued=%d stopped=%d", c.continued, c.stopped)
	}
	*now = now.Add(deliveryCheckpointGrace)
	restart := NewEffortService(r, c, s.config)
	restart.now = s.now
	if err := restart.tickDelivery(ctx); err != nil {
		t.Fatal(err)
	}
	closed, _, _ := r.GetEffort(ctx, e.EffortRef)
	if c.stopped != 1 || closed.Delivery.FinalizedAt == nil || closed.Delivery.State != "stopped" {
		t.Fatalf("stop/terminal readback missing: %+v stops=%d", closed.Delivery, c.stopped)
	}
	if closed.Delivery.Outcome == nil || len(closed.Delivery.Outcome.SurfaceScoreboard) != 2 || closed.Delivery.Outcome.SurfaceScoreboard[0].State != "close" {
		t.Fatal("terminal handback changed or omitted the honest surface snapshot")
	}
	status, err := os.ReadFile(filepath.Join(s.config.ProjectRoot, "delivery/STATUS.md"))
	if err != nil || !strings.Contains(string(status), "delivery-outcome:disposable") || !strings.Contains(string(status), "| Inbox | selected | close |") {
		t.Fatal("terminal STATUS lost its checkpoint or changed acceptance", err)
	}
	decisions, err := r.ListEffortDecisions(ctx, "delivery", e.EffortRef, 10)
	if err != nil || len(decisions) != 1 {
		t.Fatalf("operator handback: %v %+v", err, decisions)
	}
	decision := decisions[0]
	for _, want := range []string{e.EffortRef, "Inbox", "close", "Settings", "unassessed", "Accept partial", "Recommission", "Change scope"} {
		if !strings.Contains(decision.BigPicture, want) {
			t.Errorf("delivery handback big picture omits %q: %q", want, decision.BigPicture)
		}
	}
	for _, want := range []string{e.Delivery.CommissionId, "1h0m0s", "2h0m0s", "product-1", "Partial product; no independent acceptance.", "delivery/CHARTER.md", "delivery/STATUS.md", "delivery/LOG.md", "unknown"} {
		if !strings.Contains(decision.TechnicalDetails, want) {
			t.Errorf("delivery handback technical details omit %q: %q", want, decision.TechnicalDetails)
		}
	}
	for name, body := range map[string]string{"big picture": decision.BigPicture, "technical details": decision.TechnicalDetails} {
		if strings.Count(body, "\n\n") < 2 {
			t.Errorf("%s does not explain the decision in paragraphs: %q", name, body)
		}
	}
	event := decisionEvent(decision)
	if event.Payload["big_picture"] != decision.BigPicture || event.Payload["technical_details"] != decision.TechnicalDetails {
		t.Error("published decision does not retain both explanations")
	}
	if decisions[0].State != DecisionPending || decisions[0].PublishedAt != nil {
		t.Fatal("unpublished handback claimed an answer or delivery")
	}
	if _, err := restart.DeliveryRunContext(ctx, run); err == nil {
		t.Fatal("closed commission resumed")
	}
	if err := restart.tickDelivery(ctx); err != nil {
		t.Fatal(err)
	}
	if c.stopped != 1 {
		t.Fatal("repeated tick stopped an owner twice")
	}
}

func TestDeliveryLateObservationStillCheckpointsAndFencesQueuedWork(t *testing.T) {
	s, r, c, e, now := deliveryFixture(t)
	ctx := context.Background()
	id := uuid.New()
	run := &domain.Run{ID: id, Status: domain.RunStatusRunning, ResolvedConfig: &domain.RunConfig{DeliveryCommissions: map[string]string{e.EffortRef: e.Delivery.CommissionId}}, WorkReferences: []*eventpb.WorkReference{deliveryReference("orchestrator")}}
	c.runs[id] = run
	e.Subjects = []*pb.EffortSubject{{Owner: "agent-manager", Kind: "run", RunId: id.String(), Reference: id.String(), Role: "orchestrator"}}
	e, err := s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "retain-late"}, EffortActor{ID: "owner", Operator: true})
	if err != nil {
		t.Fatal(err)
	}
	*now = now.Add(3 * time.Hour)
	if _, err := s.DeliveryInitialContext(ctx, run); err == nil {
		t.Fatal("queued product work started after the hard deadline")
	}
	if err := s.tickDelivery(ctx); err != nil {
		t.Fatal(err)
	}
	observed, _, _ := r.GetEffort(ctx, e.EffortRef)
	p := DeliveryPressure(observed.Delivery, *now)
	if c.continued != 1 || c.stopped != 0 || !p.CheckpointDeadline.AsTime().Equal(now.Add(deliveryCheckpointGrace)) || !p.HardStopAt.AsTime().Equal(e.Delivery.StartedAt.AsTime().Add(2*time.Hour)) {
		t.Fatalf("late checkpoint lost original deadline or grace: %+v continued=%d stopped=%d", p, c.continued, c.stopped)
	}
	if _, err := s.DeliveryRunContext(ctx, run); err != nil {
		t.Fatal("checkpoint continuation refused", err)
	}
	if _, err := s.DeliveryInitialContext(ctx, run); err == nil {
		t.Fatal("initial product execution used checkpoint grace")
	}
	*now = now.Add(deliveryCheckpointGrace)
	if err := s.tickDelivery(ctx); err != nil {
		t.Fatal(err)
	}
	if c.stopped != 1 {
		t.Fatal("late owner was not stopped after checkpoint grace")
	}
}

func TestDeliveryHandbackPublicationRetryKeepsFrozenExplanation(t *testing.T) {
	s, r, c, e, now := deliveryFixture(t)
	ctx := context.Background()
	publisher := &recordingPublisher{fail: errors.New("events unavailable")}
	s.SetDecisionPublisher(publisher)
	*now = now.Add(2 * time.Hour)
	if err := s.tickDelivery(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := r.db.Exec(`CREATE TRIGGER fail_delivery_finalization BEFORE UPDATE ON supervision_efforts WHEN json_extract(NEW.enrollment_json, '$.delivery.finalized_at') IS NOT NULL BEGIN SELECT RAISE(FAIL, 'injected finalization fault'); END`); err != nil {
		t.Fatal(err)
	}
	*now = now.Add(deliveryCheckpointGrace)
	if err := s.tickDelivery(ctx); err == nil {
		t.Fatal("finalization failure was hidden")
	}
	first, _, err := r.GetEffortDecision(ctx, "delivery", "delivery-"+e.Delivery.CommissionId)
	if err != nil {
		t.Fatal(err)
	}
	frozen, _, err := r.GetEffort(ctx, e.EffortRef)
	if err != nil || frozen.Delivery.Outcome == nil || frozen.Delivery.FinalizedAt != nil {
		t.Fatalf("outcome must survive a failed finalization without claiming completion: %v %+v", err, frozen)
	}
	file := filepath.Join(s.config.ProjectRoot, "delivery/STATUS.md")
	status, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(strings.ReplaceAll(string(status), "| Inbox | selected | close |", "| Inbox | selected | accepted |")), 0o644); err != nil {
		t.Fatal(err)
	}
	*now = now.Add(time.Minute)
	if _, err := r.db.Exec(`DROP TRIGGER fail_delivery_finalization`); err != nil {
		t.Fatal(err)
	}
	publisher.fail = nil
	restart := NewEffortService(r, c, s.config)
	restart.now = s.now
	restart.SetDecisionPublisher(publisher)
	if err := restart.tickDelivery(ctx); err != nil {
		t.Fatal(err)
	}
	final, _, _ := r.GetEffort(ctx, e.EffortRef)
	decision, _, _ := r.GetEffortDecision(ctx, "delivery", first.DecisionId)
	if final.Delivery.FinalizedAt == nil || !proto.Equal(final.Delivery.Outcome, frozen.Delivery.Outcome) {
		t.Fatal("retry changed the frozen outcome or failed to finalize")
	}
	if decision.BigPicture != first.BigPicture || decision.TechnicalDetails != first.TechnicalDetails || decision.EventId != first.EventId || publisher.count() != 1 {
		t.Fatal("retry changed the operator context or duplicated publication")
	}
}

func TestDeliveryHandbackContextPreservesUnknownAndBoundedUnicode(t *testing.T) {
	_, _, _, e, now := deliveryFixture(t)
	e.DisplayName = "Notification Hub redesign"
	e.Delivery.StoppedAt = timestamppb.New(*now)
	d := deliveryHandbackDecision(e, "delivery", nil)
	if !strings.Contains(d.Question, e.DisplayName) || !strings.Contains(d.BigPicture, "progress is unknown") || !strings.Contains(d.BigPicture, "assessment is unavailable") {
		t.Fatalf("unknown outcome was hidden or treated as completion: %+v", d)
	}
	zero := 0.0
	outcome := &pb.EffortGoalHome{CurrentValue: &zero, DestinationMetric: "accepted surfaces", NeedsOperator: []string{"Disposable-recipient authority is required."}, Limitations: []string{"Missing owner receipt."}}
	d = deliveryHandbackDecision(e, "delivery", outcome)
	if !strings.Contains(d.BigPicture, "0 accepted surfaces") || !strings.Contains(d.TechnicalDetails, outcome.NeedsOperator[0]) || !strings.Contains(d.TechnicalDetails, outcome.Limitations[0]) {
		t.Fatalf("zero progress, operator requirement or limitation was lost: %+v", d)
	}
	e.DisplayName = strings.Repeat("🧭", 500)
	for range 40 {
		outcome.SurfaceScoreboard = append(outcome.SurfaceScoreboard, &pb.SurfaceProgress{Surface: strings.Repeat("😀", 500), Target: strings.Repeat("🧭", 500), State: "rough"})
	}
	d = deliveryHandbackDecision(e, "delivery", outcome)
	if err := DecisionView(d).Validate(); err != nil {
		t.Fatal("large context is not publishable under the shared decision contract:", err)
	}
	if !utf8.ValidString(d.BigPicture) || !utf8.ValidString(d.TechnicalDetails) || !strings.Contains(d.BigPicture, "truncated") || !strings.Contains(d.TechnicalDetails, "truncated") {
		t.Fatal("large Unicode context was corrupted or silently omitted")
	}
	if !strings.Contains(d.BigPicture, "Accept partial") || !strings.Contains(d.BigPicture, "Recommission") || !strings.Contains(d.BigPicture, "Change scope") || !strings.Contains(d.TechnicalDetails, "delivery/LOG.md") {
		t.Fatal("large surface list displaced decision consequences or evidence references")
	}
}

func TestDeliveryWithdrawalCannotAbandonRunningOwners(t *testing.T) {
	s, _, _, e, _ := deliveryFixture(t)
	if _, err := s.Withdraw(context.Background(), &pb.WithdrawEffortRequest{EffortRef: e.EffortRef, ExpectedRevision: e.Revision, Reason: "operator withdrawal", IdempotencyKey: "withdraw-running"}, EffortActor{ID: "owner", Operator: true}); err == nil {
		t.Fatal("withdrawal abandoned an unfinished delivery commission")
	}
}
