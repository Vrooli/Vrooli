package supervision

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
)

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
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.RecordCallerSignal(ctx, &pb.RecordEffortSignalRequest{EffortRef: "effort:goal", Kind: SignalStepBack, RunId: member.String(), Detail: "STEP_BACK flatline", DedupeKey: "E27:flatline:12"})
	if err != nil || again.Seq != first.Seq {
		t.Fatalf("an unchanged step-back was recorded twice: %v %v", again, err)
	}
	feedback, err := s.RecordCallerSignal(ctx, &pb.RecordEffortSignalRequest{EffortRef: "effort:goal", Kind: SignalFeedback, Detail: "FB-070 operator: re-aim at journeys", DedupeKey: "feedback:FB-070"})
	if err != nil || feedback.Seq != first.Seq+1 || feedback.Kind != SignalFeedback {
		t.Fatalf("a FEEDBACK writer could not record a feedback signal: %v %v", feedback, err)
	}
}
