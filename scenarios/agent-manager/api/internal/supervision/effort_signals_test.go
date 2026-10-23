package supervision

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
)

type memoryEffortSignalCursorStore struct{ runs map[uuid.UUID]*domain.Run }

func (m memoryEffortSignalCursorStore) QueueEffortSignalCursor(_ context.Context, id uuid.UUID, ref string, through int64) (domain.EffortSignalCursor, bool, error) {
	return m.mutate(id, ref, through)
}

func (m memoryEffortSignalCursorStore) mutate(id uuid.UUID, ref string, through int64) (domain.EffortSignalCursor, bool, error) {
	run := m.runs[id]
	if run == nil || !run.Status.IsValid() || run.Status.IsTerminal() || !runCarriesEffortOrchestrator(run, ref) {
		return domain.EffortSignalCursor{}, false, nil
	}
	if run.EffortSignalCursors == nil {
		run.EffortSignalCursors = map[string]domain.EffortSignalCursor{}
	}
	cursor, exists := run.EffortSignalCursors[ref]
	if !exists {
		cursor = domain.EffortSignalCursor{QueuedSeq: through, DeliveredSeq: 0}
	} else if through > cursor.QueuedSeq && through > cursor.DeliveredSeq {
		cursor.QueuedSeq = through
	}
	run.EffortSignalCursors[ref] = cursor
	return cursor, true, nil
}

func effortReference(ref, relationship string) *eventpb.WorkReference {
	return &eventpb.WorkReference{Kind: "effort", Id: ref, Relationship: relationship, Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE}
}

// endedRunRegistry honours the EndedFrom window the sweep asks for.
type endedRunRegistry struct{ runs []*domain.Run }

func (f endedRunRegistry) List(_ context.Context, filter repository.RunListFilter) ([]*domain.Run, error) {
	var out []*domain.Run
	for _, run := range f.runs {
		if filter.EndedFrom != nil && (run.EndedAt == nil || run.EndedAt.Before(*filter.EndedFrom)) {
			continue
		}
		out = append(out, run)
	}
	if filter.Offset >= len(out) {
		return nil, nil
	}
	end := filter.Offset + filter.Limit
	if end > len(out) {
		end = len(out)
	}
	return out[filter.Offset:end], nil
}

func TestEffortSignalSeqIsMonotonicPerEffortAndOneFactIsOneSignal(t *testing.T) {
	repo, _ := testRepository(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make(chan error, 40)
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			_, _, err := repo.AppendEffortSignal(ctx, "effort:a", SignalStepBack, "", fmt.Sprintf("fact %d", i), fmt.Sprintf("fact-%d", i))
			errs <- err
		}(i)
		go func(i int) {
			defer wg.Done()
			// The same fact recorded twice concurrently is one signal.
			_, _, err := repo.AppendEffortSignal(ctx, "effort:a", SignalStepBack, "", fmt.Sprintf("fact %d", i), fmt.Sprintf("fact-%d", i))
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := repo.AppendEffortSignal(ctx, "effort:b", SignalFeedback, "", "other effort", "b-1"); err != nil {
		t.Fatal(err)
	}
	signals, latest, err := repo.ListEffortSignals(ctx, "effort:a", 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 20 || latest != 20 {
		t.Fatalf("want 20 distinct facts with latest seq 20, got %d signals latest %d", len(signals), latest)
	}
	for i, signal := range signals {
		if signal.Seq != int64(i+1) {
			t.Fatalf("seq is not dense and monotonic: position %d has seq %d", i, signal.Seq)
		}
	}
	after, latest, err := repo.ListEffortSignals(ctx, "effort:a", 18, 500)
	if err != nil || len(after) != 2 || after[0].Seq != 19 || latest != 20 {
		t.Fatalf("cursor read: %v latest=%d err=%v", after, latest, err)
	}
	other, _, err := repo.ListEffortSignals(ctx, "effort:b", 0, 10)
	if err != nil || len(other) != 1 || other[0].Seq != 1 {
		t.Fatalf("each effort has its own sequence: %v %v", other, err)
	}
}

func TestEffortSignalTickCoalescesPerEffortAndWaitsForParkedOwner(t *testing.T) {
	s, repo, controller := effortFixture(t)
	ctx := context.Background()
	runID := uuid.New()
	effortRef := "effort:checkin"
	run := &domain.Run{ID: runID, Status: domain.RunStatusParked, WorkReferences: []*eventpb.WorkReference{effortReference(effortRef, "orchestrator")}}
	controller.runs[runID] = run
	s.SetEffortSignalCursorStore(memoryEffortSignalCursorStore{runs: controller.runs})
	enrollment := &pb.EffortEnrollment{EffortRef: effortRef, AuthorizedBy: "owner", Subjects: []*pb.EffortSubject{{Owner: "agent-manager", Kind: "run", Reference: runID.String(), RunId: runID.String(), Role: "orchestrator"}}}
	if err := repo.SaveEffort(ctx, enrollment, &pb.EffortBoardRow{}, 0, "enroll-checkin", "digest-checkin"); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		if _, _, err := repo.AppendEffortSignal(ctx, effortRef, SignalFeedback, "", fmt.Sprintf("feedback %d", i), fmt.Sprintf("feedback-%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.dispatchEffortSignalCheckins(ctx); err != nil {
		t.Fatal(err)
	}
	if len(controller.checkins) != 1 || controller.checkins[0] != (effortCheckinCall{runID: runID, effortRef: effortRef, through: 3}) {
		t.Fatalf("legacy retained signals should coalesce into one payload-neutral catch-up: %+v", controller.checkins)
	}
	if cursor := run.EffortSignalCursors[effortRef]; cursor != (domain.EffortSignalCursor{QueuedSeq: 3, DeliveredSeq: 3}) {
		t.Fatalf("successful wake did not consume exactly the queued range: %+v", cursor)
	}
	if _, _, err := repo.AppendEffortSignal(ctx, effortRef, SignalStepBack, "", "new step back", "step-back-4"); err != nil {
		t.Fatal(err)
	}
	if err := s.dispatchEffortSignalCheckins(ctx); err != nil {
		t.Fatal(err)
	}
	if len(controller.checkins) != 1 || run.EffortSignalCursors[effortRef] != (domain.EffortSignalCursor{QueuedSeq: 4, DeliveredSeq: 3}) {
		t.Fatalf("running owner should retain new signal without wake: calls=%+v cursor=%+v", controller.checkins, run.EffortSignalCursors[effortRef])
	}
	run.Status = domain.RunStatusParked
	if err := s.dispatchEffortSignalCheckins(ctx); err != nil {
		t.Fatal(err)
	}
	if len(controller.checkins) != 2 || controller.checkins[1].through != 4 || run.EffortSignalCursors[effortRef].DeliveredSeq != 4 {
		t.Fatalf("parked owner did not receive the pending range exactly once: calls=%+v cursor=%+v", controller.checkins, run.EffortSignalCursors[effortRef])
	}
}

func TestEffortSignalTickRetainsQueuedRangeWhenWakeFails(t *testing.T) {
	s, repo, controller := effortFixture(t)
	ctx := context.Background()
	runID := uuid.New()
	effortRef := "effort:retry-checkin"
	run := &domain.Run{ID: runID, Status: domain.RunStatusParked, WorkReferences: []*eventpb.WorkReference{effortReference(effortRef, "orchestrator")}}
	controller.runs[runID] = run
	s.SetEffortSignalCursorStore(memoryEffortSignalCursorStore{runs: controller.runs})
	enrollment := &pb.EffortEnrollment{EffortRef: effortRef, AuthorizedBy: "owner", Subjects: []*pb.EffortSubject{{Owner: "agent-manager", Kind: "run", Reference: runID.String(), RunId: runID.String(), Role: "orchestrator"}}}
	if err := repo.SaveEffort(ctx, enrollment, &pb.EffortBoardRow{}, 0, "enroll-retry", "digest-retry"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.AppendEffortSignal(ctx, effortRef, SignalFeedback, "", "private detail must not be transported", "retry-1"); err != nil {
		t.Fatal(err)
	}
	controller.checkinErr = errors.New("temporary wake failure")
	if err := s.dispatchEffortSignalCheckins(ctx); err == nil || !strings.Contains(err.Error(), "temporary wake failure") {
		t.Fatalf("wake failure was hidden: %v", err)
	}
	if cursor := run.EffortSignalCursors[effortRef]; cursor != (domain.EffortSignalCursor{QueuedSeq: 1, DeliveredSeq: 0}) {
		t.Fatalf("failed wake advanced delivered cursor: %+v", cursor)
	}
	controller.checkinErr = nil
	if err := s.dispatchEffortSignalCheckins(ctx); err != nil {
		t.Fatal(err)
	}
	if len(controller.checkins) != 1 || controller.checkins[0].through != 1 || run.EffortSignalCursors[effortRef].DeliveredSeq != 1 {
		t.Fatalf("pending wake was not retried exactly: calls=%+v cursor=%+v", controller.checkins, run.EffortSignalCursors[effortRef])
	}
}

func TestEffortSignalTickContinuesPastFailedOwnerAndRetriesAfterWrap(t *testing.T) {
	s, repo, controller := effortFixture(t)
	ctx := context.Background()
	firstID, secondID := uuid.New(), uuid.New()
	firstRef, secondRef := "effort:a-blocked", "effort:b-valid"
	firstRun := &domain.Run{ID: firstID, Status: domain.RunStatusParked, WorkReferences: []*eventpb.WorkReference{effortReference(firstRef, "orchestrator")}}
	secondRun := &domain.Run{ID: secondID, Status: domain.RunStatusParked, WorkReferences: []*eventpb.WorkReference{effortReference(secondRef, "orchestrator")}}
	controller.runs[firstID], controller.runs[secondID] = firstRun, secondRun
	s.SetEffortSignalCursorStore(memoryEffortSignalCursorStore{runs: controller.runs})
	for _, owner := range []struct {
		ref string
		id  uuid.UUID
	}{
		{firstRef, firstID},
		{secondRef, secondID},
	} {
		enrollment := &pb.EffortEnrollment{EffortRef: owner.ref, Subjects: []*pb.EffortSubject{{Owner: "agent-manager", Kind: "run", Reference: owner.id.String(), RunId: owner.id.String(), Role: "orchestrator"}}}
		if err := repo.SaveEffort(ctx, enrollment, &pb.EffortBoardRow{}, 0, "enroll-"+owner.ref, "digest-"+owner.ref); err != nil {
			t.Fatal(err)
		}
		if _, _, err := repo.AppendEffortSignal(ctx, owner.ref, SignalFeedback, "", "neutral signal detail", "signal-"+owner.ref); err != nil {
			t.Fatal(err)
		}
	}

	wakeErr := errors.New("first owner temporarily unavailable")
	controller.checkinErrByRun = map[uuid.UUID]error{firstID: wakeErr}
	if err := s.dispatchEffortSignalCheckins(ctx); !errors.Is(err, wakeErr) {
		t.Fatalf("batch must retain the owner error for reporting: %v", err)
	}
	if len(controller.checkins) != 1 || controller.checkins[0].runID != secondID {
		t.Fatalf("one failed owner starved a later admissible owner: %+v", controller.checkins)
	}
	if firstRun.EffortSignalCursors[firstRef] != (domain.EffortSignalCursor{QueuedSeq: 1, DeliveredSeq: 0}) {
		t.Fatalf("failed owner's queued signal was cleared or falsely delivered: %+v", firstRun.EffortSignalCursors[firstRef])
	}
	if secondRun.EffortSignalCursors[secondRef] != (domain.EffortSignalCursor{QueuedSeq: 1, DeliveredSeq: 1}) {
		t.Fatalf("later owner's successful delivery was not retained: %+v", secondRun.EffortSignalCursors[secondRef])
	}

	delete(controller.checkinErrByRun, firstID)
	// The current keyset reaches its end, resets, and retries the retained
	// pending owner on the following scheduler tick.
	if err := s.dispatchEffortSignalCheckins(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.dispatchEffortSignalCheckins(ctx); err != nil {
		t.Fatal(err)
	}
	if len(controller.checkins) != 2 || controller.checkins[1] != (effortCheckinCall{runID: firstID, effortRef: firstRef, through: 1}) {
		t.Fatalf("failed owner's pending signal was not retried after keyset wrap: %+v", controller.checkins)
	}
	if firstRun.EffortSignalCursors[firstRef] != (domain.EffortSignalCursor{QueuedSeq: 1, DeliveredSeq: 1}) {
		t.Fatalf("retried owner cursor did not advance after successful wake: %+v", firstRun.EffortSignalCursors[firstRef])
	}
}

func TestEffortSignalTickRequiresExactActiveOrchestrator(t *testing.T) {
	s, repo, controller := effortFixture(t)
	ctx := context.Background()
	validID, mismatchID, withdrawnID := uuid.New(), uuid.New(), uuid.New()
	validRef, mismatchRef, withdrawnRef := "effort:valid", "effort:mismatch", "effort:withdrawn"
	controller.runs[validID] = &domain.Run{ID: validID, Status: domain.RunStatusParked, WorkReferences: []*eventpb.WorkReference{effortReference(validRef, "orchestrator")}}
	controller.runs[mismatchID] = &domain.Run{ID: mismatchID, Status: domain.RunStatusParked, WorkReferences: []*eventpb.WorkReference{effortReference("effort:replacement", "orchestrator")}}
	controller.runs[withdrawnID] = &domain.Run{ID: withdrawnID, Status: domain.RunStatusParked, WorkReferences: []*eventpb.WorkReference{effortReference(withdrawnRef, "orchestrator")}}
	s.SetEffortSignalCursorStore(memoryEffortSignalCursorStore{runs: controller.runs})
	for _, row := range []struct {
		ref       string
		id        uuid.UUID
		withdrawn bool
	}{
		{validRef, validID, false}, // Normal work-reference discovery has no operator authorization field.
		{mismatchRef, mismatchID, false},
		{withdrawnRef, withdrawnID, true},
	} {
		enrollment := &pb.EffortEnrollment{EffortRef: row.ref, Withdrawn: row.withdrawn, Subjects: []*pb.EffortSubject{{Owner: "agent-manager", Kind: "run", Reference: row.id.String(), RunId: row.id.String(), Role: "orchestrator"}}}
		if err := repo.SaveEffort(ctx, enrollment, &pb.EffortBoardRow{}, 0, "enroll-"+row.ref, "digest-"+row.ref); err != nil {
			t.Fatal(err)
		}
		if _, _, err := repo.AppendEffortSignal(ctx, row.ref, SignalFeedback, "", "signal", "signal-"+row.ref); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.dispatchEffortSignalCheckins(ctx); err != nil {
		t.Fatal(err)
	}
	if len(controller.checkins) != 1 || controller.checkins[0].runID != validID || controller.checkins[0].effortRef != validRef {
		t.Fatalf("dispatcher woke an unapproved or mismatched owner: %+v", controller.checkins)
	}
}

func TestEffortSweepSignalsLinkedRunEndsOnceAcrossRestart(t *testing.T) {
	s, repo, c := effortFixture(t)
	ctx := context.Background()
	now := repo.now()
	ended := now.Add(-time.Minute)
	run := func(status domain.RunStatus, stop domain.RunStopReason, refs ...*eventpb.WorkReference) *domain.Run {
		id := uuid.New()
		r := &domain.Run{ID: id, Status: status, StopReason: stop, EndedAt: &ended, WorkReferences: refs}
		c.runs[id] = r
		return r
	}
	worker := run(domain.RunStatusComplete, "", effortReference("effort:goal", "worker"))
	capped := run(domain.RunStatusCancelled, domain.RunStopReasonTokenCap, effortReference("effort:goal", "orchestrator"))
	orchestrator := run(domain.RunStatusComplete, "", effortReference("effort:goal", "orchestrator"))
	unverified := run(domain.RunStatusFailed, "", &eventpb.WorkReference{Kind: "effort", Id: "effort:goal", Relationship: "worker"})
	stillRunning := run(domain.RunStatusRunning, "", effortReference("effort:goal", "worker"))
	stillRunning.EndedAt = nil
	registry := endedRunRegistry{runs: []*domain.Run{worker, capped, orchestrator, unverified, stillRunning}}
	s.SetRunRegistry(registry)

	if err := s.sweepEndedRuns(ctx); err != nil {
		t.Fatal(err)
	}
	restart := NewEffortService(repo, c, s.config)
	restart.now = repo.now
	restart.SetRunRegistry(registry)
	if err := restart.sweepEndedRuns(ctx); err != nil {
		t.Fatal(err)
	}
	got, err := restart.ListSignals(ctx, &pb.ListEffortSignalsRequest{EffortRef: "effort:goal"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{worker.ID.String(): SignalChildRunEnded, capped.ID.String(): SignalCapStop}
	if len(got.Signals) != len(want) || got.LatestSeq != int64(len(want)) {
		t.Fatalf("want one signal per linked terminal fact %v, got %v", want, got.Signals)
	}
	for _, signal := range got.Signals {
		if want[signal.RunId] != signal.Kind {
			t.Fatalf("run %s gave %s, want %q", signal.RunId, signal.Kind, want[signal.RunId])
		}
	}
}

func TestRecordEffortSignalAcceptsOnlyCallerKindsNamingARunOfTheEffort(t *testing.T) {
	s, _, c := effortFixture(t)
	ctx := context.Background()
	member, stranger := uuid.New(), uuid.New()
	c.runs[member] = &domain.Run{ID: member, Status: domain.RunStatusRunning, WorkReferences: []*eventpb.WorkReference{effortReference("effort:goal", "worker")}}
	c.runs[stranger] = &domain.Run{ID: stranger, Status: domain.RunStatusRunning, WorkReferences: []*eventpb.WorkReference{effortReference("effort:other", "worker")}}

	if _, err := s.RecordCallerSignal(ctx, &pb.RecordEffortSignalRequest{EffortRef: "effort:goal", Kind: SignalOperatorAnswer}); !errors.Is(err, ErrSignalKindNotAccepted) {
		t.Fatalf("a caller recorded an Agent Manager-derived kind: %v", err)
	}
	if _, err := s.RecordCallerSignal(ctx, &pb.RecordEffortSignalRequest{EffortRef: "effort:goal", Kind: SignalStepBack, RunId: stranger.String()}); err == nil {
		t.Fatal("a run of another effort recorded a step_back")
	}
	first, err := s.RecordCallerSignal(ctx, &pb.RecordEffortSignalRequest{EffortRef: "effort:goal", Kind: SignalStepBack, RunId: member.String(), Detail: "STEP_BACK flatline", DedupeKey: "E27:flatline:12"})
	if err != nil || first.GetSeq() < 1 || first.NewlyRecorded == nil || !*first.NewlyRecorded {
		t.Fatalf("a first step-back must report newly_recorded=true: %v %v", first, err)
	}
	again, err := s.RecordCallerSignal(ctx, &pb.RecordEffortSignalRequest{EffortRef: "effort:goal", Kind: SignalStepBack, RunId: member.String(), Detail: "STEP_BACK flatline", DedupeKey: "E27:flatline:12"})
	if err != nil || again.Seq != first.Seq || again.NewlyRecorded == nil || *again.NewlyRecorded {
		t.Fatalf("an unchanged step-back must report newly_recorded=false: %v %v", again, err)
	}
	worsened, err := s.RecordCallerSignal(ctx, &pb.RecordEffortSignalRequest{EffortRef: "effort:goal", Kind: SignalStepBack, RunId: member.String(), Detail: "STEP_BACK flatline worsened", DedupeKey: "E27:flatline:13"})
	if err != nil || worsened.NewlyRecorded == nil || !*worsened.NewlyRecorded || worsened.Seq <= first.Seq {
		t.Fatalf("a changed trigger must report a new event: %v %v", worsened, err)
	}
	feedback, err := s.RecordCallerSignal(ctx, &pb.RecordEffortSignalRequest{EffortRef: "effort:goal", Kind: SignalFeedback, Detail: "FB-070 operator: re-aim at journeys", DedupeKey: "feedback:FB-070"})
	if err != nil || feedback.Seq != worsened.Seq+1 || feedback.Kind != SignalFeedback {
		t.Fatalf("a FEEDBACK writer could not record a feedback signal: %v %v", feedback, err)
	}
	listed, err := s.ListSignals(ctx, &pb.ListEffortSignalsRequest{EffortRef: "effort:goal"})
	if err != nil {
		t.Fatal(err)
	}
	for _, stored := range listed.GetSignals() {
		if stored.NewlyRecorded != nil {
			t.Fatalf("response-only insertion metadata leaked into listed event %d", stored.GetSeq())
		}
	}
}
