package supervision

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"

	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type summaryResolver struct{ summaries []SubjectSummary }

func (r summaryResolver) Resolve(context.Context, []*domainpb.WatchSubject) ([]SubjectSummary, error) {
	return r.summaries, nil
}

type evaluatorFunc func(context.Context, EvaluationInput) (*domainpb.WatchDecision, error)

func (f evaluatorFunc) Evaluate(ctx context.Context, input EvaluationInput) (*domainpb.WatchDecision, error) {
	return f(ctx, input)
}

func TestProcessorCommitsBoundedEventDecisionAndCursorTogether(t *testing.T) {
	repo, _ := testRepository(t)
	runID := uuid.New()
	source := &cohortSource{retention: eventlog.RetentionState{Generation: 1, HighRowID: 18}, events: []eventlog.CohortEvent{{ID: uuid.New(), RunID: runID, Rowid: 18, Sequence: 3, EventType: domain.EventTypeStatus, Timestamp: time.Now().UTC()}}}
	service := NewService(repo, source)
	spec := validServiceSpec(runID)
	spec.Triggers.EventCount = 1
	watch, _, err := service.Create(context.Background(), &domainpb.CreateCohortWatchRequest{Spec: spec, IdempotencyKey: "process-events"})
	if err != nil {
		t.Fatal(err)
	}
	processor := NewProcessor(service, summaryResolver{summaries: []SubjectSummary{{RunID: runID.String(), Status: "running"}}}, nil)
	updated, err := processor.Process(context.Background(), watch.GetWatchId())
	if err != nil {
		t.Fatal(err)
	}
	if updated.GetRevision() != 2 || updated.GetLastDecision().GetClassification() != "event_count" || updated.GetCursor().GetToken() == watch.GetCursor().GetToken() {
		t.Fatalf("processed watch = %+v", updated)
	}
	_, checkpoint, err := repo.Get(context.Background(), watch.GetWatchId())
	if err != nil || checkpoint.RowID != 18 {
		t.Fatalf("checkpoint = %+v err=%v", checkpoint, err)
	}
}

func TestProcessorDoesNotConsumeEventsWhenEvaluatorIsUnavailable(t *testing.T) {
	repo, _ := testRepository(t)
	runID := uuid.New()
	source := &cohortSource{retention: eventlog.RetentionState{Generation: 1, HighRowID: 18}, events: []eventlog.CohortEvent{{ID: uuid.New(), RunID: runID, Rowid: 18, Sequence: 3, EventType: domain.EventTypeStatus, Timestamp: time.Now().UTC()}}}
	service := NewService(repo, source)
	watch, _, err := service.Create(context.Background(), &domainpb.CreateCohortWatchRequest{Spec: validServiceSpec(runID), IdempotencyKey: "unavailable-replay"})
	if err != nil {
		t.Fatal(err)
	}
	evaluator := evaluatorFunc(func(context.Context, EvaluationInput) (*domainpb.WatchDecision, error) {
		return &domainpb.WatchDecision{Disposition: domainpb.WatchDisposition_WATCH_DISPOSITION_UNAVAILABLE, Classification: "dependency_unavailable", RecommendedAction: domainpb.WatchActionKind_WATCH_ACTION_KIND_OBSERVE}, nil
	})
	updated, err := NewProcessor(service, nil, evaluator).Process(context.Background(), watch.GetWatchId())
	if err != nil {
		t.Fatal(err)
	}
	_, checkpoint, err := repo.Get(context.Background(), watch.GetWatchId())
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.RowID != 0 || updated.GetCursor().GetToken() != watch.GetCursor().GetToken() {
		t.Fatalf("unavailable evaluation consumed evidence: checkpoint=%+v cursor=%q", checkpoint, updated.GetCursor().GetToken())
	}
}

func TestProcessorReconcilesRetentionResetWithoutReadingMissingHistory(t *testing.T) {
	repo, _ := testRepository(t)
	runID := uuid.New()
	source := &cohortSource{retention: eventlog.RetentionState{Generation: 2, HighRowID: 30}}
	service := NewService(repo, source)
	watch, _, err := service.Create(context.Background(), &domainpb.CreateCohortWatchRequest{Spec: validServiceSpec(runID), IdempotencyKey: "process-reset"})
	if err != nil {
		t.Fatal(err)
	}
	source.retention = eventlog.RetentionState{Generation: 3, FloorRowID: 20, HighRowID: 40}
	processor := NewProcessor(service, summaryResolver{summaries: []SubjectSummary{{RunID: runID.String(), Status: "running", EvidenceIDs: []string{"run:" + runID.String()}}}}, nil)
	updated, err := processor.Process(context.Background(), watch.GetWatchId())
	if err != nil {
		t.Fatal(err)
	}
	if updated.GetLastDecision().GetDisposition() != domainpb.WatchDisposition_WATCH_DISPOSITION_CURSOR_RESET || source.readCalls != 0 {
		t.Fatalf("reset decision = %+v reads=%d", updated.GetLastDecision(), source.readCalls)
	}
	_, checkpoint, err := repo.Get(context.Background(), watch.GetWatchId())
	if err != nil || checkpoint.RetentionGeneration != 3 || checkpoint.RowID != 40 {
		t.Fatalf("reset checkpoint = %+v err=%v", checkpoint, err)
	}
}

func TestProcessorTerminatesOnlyWhenDurableSubjectSummariesAreTerminal(t *testing.T) {
	repo, _ := testRepository(t)
	runID := uuid.New()
	service := NewService(repo, &cohortSource{retention: eventlog.RetentionState{Generation: 1}})
	watch, _, err := service.Create(context.Background(), &domainpb.CreateCohortWatchRequest{Spec: validServiceSpec(runID), IdempotencyKey: "process-terminal"})
	if err != nil {
		t.Fatal(err)
	}
	processor := NewProcessor(service, summaryResolver{summaries: []SubjectSummary{{RunID: runID.String(), Status: "complete", Terminal: true}}}, nil)
	updated, err := processor.Process(context.Background(), watch.GetWatchId())
	if err != nil || updated.GetStatus() != domainpb.WatchStatus_WATCH_STATUS_TERMINAL || updated.GetLastDecision().GetDisposition() != domainpb.WatchDisposition_WATCH_DISPOSITION_TERMINAL {
		t.Fatalf("terminal watch = %+v err=%v", updated, err)
	}
}

func TestTriggerEvaluatorCoversQuietDeadlineFrictionAndTerminal(t *testing.T) {
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name           string
		triggers       *domainpb.WatchTriggers
		summaries      []SubjectSummary
		updatedAt      time.Time
		want           domainpb.WatchDisposition
		classification string
	}{
		{name: "quiet", triggers: &domainpb.WatchTriggers{QuietTime: durationpb.New(time.Minute)}, updatedAt: now.Add(-2 * time.Minute), want: domainpb.WatchDisposition_WATCH_DISPOSITION_SIGNAL, classification: "quiet_time"},
		{name: "deadline", triggers: &domainpb.WatchTriggers{Deadline: timestamppb.New(now.Add(-time.Second))}, updatedAt: now, want: domainpb.WatchDisposition_WATCH_DISPOSITION_SIGNAL, classification: "deadline"},
		{name: "friction", triggers: &domainpb.WatchTriggers{FrictionScore: .7}, summaries: []SubjectSummary{{FrictionScore: .8}}, updatedAt: now, want: domainpb.WatchDisposition_WATCH_DISPOSITION_SIGNAL, classification: "friction_threshold"},
		{name: "terminal", triggers: &domainpb.WatchTriggers{Terminal: true}, summaries: []SubjectSummary{{Terminal: true}}, updatedAt: now, want: domainpb.WatchDisposition_WATCH_DISPOSITION_TERMINAL, classification: "cohort_terminal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			watch := &domainpb.CohortWatch{Spec: &domainpb.WatchSpec{Triggers: tc.triggers}, UpdatedAt: timestamppb.New(tc.updatedAt)}
			decision, err := (TriggerEvaluator{}).Evaluate(context.Background(), EvaluationInput{Watch: watch, Subjects: tc.summaries, Now: now})
			if err != nil || decision.GetDisposition() != tc.want || decision.GetClassification() != tc.classification {
				t.Fatalf("decision = %+v err=%v", decision, err)
			}
		})
	}
}

func TestProcessorRejectsCursorWhoseFilterBindingWasChanged(t *testing.T) {
	repo, db := testRepository(t)
	runID := uuid.New()
	service := NewService(repo, &cohortSource{retention: eventlog.RetentionState{Generation: 1}})
	watch, _, err := service.Create(context.Background(), &domainpb.CreateCohortWatchRequest{Spec: validServiceSpec(runID), IdempotencyKey: "filter-binding"})
	if err != nil {
		t.Fatal(err)
	}
	tampered := validServiceSpec(uuid.New())
	raw, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(tampered)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE cohort_watches SET spec_json=? WHERE watch_id=?`, string(raw), watch.GetWatchId()); err != nil {
		t.Fatal(err)
	}
	_, err = NewProcessor(service, nil, nil).Process(context.Background(), watch.GetWatchId())
	if err == nil || !strings.Contains(err.Error(), "filter binding") {
		t.Fatalf("filter mismatch err=%v", err)
	}
}

type mutableSummaryResolver struct{ summaries []SubjectSummary }

func (r *mutableSummaryResolver) Resolve(context.Context, []*domainpb.WatchSubject) ([]SubjectSummary, error) {
	return r.summaries, nil
}

func TestQuietWatchParksParentAndTerminalWatchWakesItOnce(t *testing.T) { // [REQ:REQ-P2-009]
	repo, _ := testRepository(t)
	childID, parentID := uuid.New(), uuid.New()
	source := &cohortSource{retention: eventlog.RetentionState{Generation: 1}}
	service := NewService(repo, source)
	controller := &fakeActionController{runs: map[uuid.UUID]*domain.Run{
		childID:  {ID: childID, Status: domain.RunStatusRunning, SessionID: "child-session"},
		parentID: {ID: parentID, Status: domain.RunStatusRunning, SessionID: "parent-session"},
	}}
	actions := NewActionService(repo, controller)
	service.SetActionService(actions)
	spec := &domainpb.WatchSpec{FamilyExecutionId: "family-e2e", ParentRunId: parentID.String(), Subjects: []*domainpb.WatchSubject{{PlanId: "plan-a", RunId: childID.String()}}, Triggers: &domainpb.WatchTriggers{Terminal: true}}
	watch, _, err := service.Create(context.Background(), &domainpb.CreateCohortWatchRequest{Spec: spec, IdempotencyKey: "parent-e2e"})
	if err != nil {
		t.Fatal(err)
	}
	resolver := &mutableSummaryResolver{summaries: []SubjectSummary{{RunID: childID.String(), Status: "running"}}}
	processor := NewProcessor(service, resolver, nil)
	watch, err = processor.Process(context.Background(), watch.GetWatchId())
	if err != nil || controller.parked != 1 || controller.runs[parentID].Status != domain.RunStatusParked {
		t.Fatalf("quiet watch=%+v parked=%d err=%v", watch, controller.parked, err)
	}
	resolver.summaries = []SubjectSummary{{RunID: childID.String(), Status: "complete", Terminal: true}}
	watch, err = processor.Process(context.Background(), watch.GetWatchId())
	if err != nil || watch.GetStatus() != domainpb.WatchStatus_WATCH_STATUS_TERMINAL || controller.woken != 1 || controller.runs[parentID].Status != domain.RunStatusRunning {
		t.Fatalf("terminal watch=%+v woken=%d err=%v", watch, controller.woken, err)
	}
	if _, err := processor.Process(context.Background(), watch.GetWatchId()); err != nil || controller.woken != 1 {
		t.Fatalf("terminal replay woke parent again: woken=%d err=%v", controller.woken, err)
	}
}
