package supervision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"google.golang.org/protobuf/types/known/structpb"
)

// scriptedProgramRunner fails with err until it is cleared, then answers with
// a quiet envelope that echoes the caller-supplied cursor.
type scriptedProgramRunner struct {
	err   error
	calls int
}

func (f *scriptedProgramRunner) RunDeclaredProgram(_ context.Context, req *connect.Request[libraryv1.RunDeclaredProgramRequest]) (*connect.Response[libraryv1.RunDeclaredProgramResponse], error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	inputs := req.Msg.GetInputs().AsMap()
	stdout := fmt.Sprintf(`{"status":"ok","signals":{"disposition":"quiet","classification":"quiet","confidence":1,"abstained":false,"recommended_action":"park","next_cursor":%q,"cursor_reset":false,"wake_condition":{"kind":"after","after_seconds":30},"policy_version":%q,"inference_calls":0},"evidence":[]}`, inputs["proposed_next_cursor"], inputs["policy"].(map[string]any)["version"])
	return connect.NewResponse(&libraryv1.RunDeclaredProgramResponse{Terminal: true, Program: &programsv1.Program{Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, Stdout: stdout}}), nil
}

// The message reproduced live on 2026-09-09: the policy pinned a digest that
// program-runtime no longer holds, which it answers with HTTP 400.
var reproducedContractRejection = connect.NewError(connect.CodeFailedPrecondition, errors.New("pinned program artifact unavailable: sql: no rows in result set"))

func TestContractRejectionParksWatchForAnHourWithReason(t *testing.T) {
	now := time.Date(2026, 9, 9, 6, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"invalid-argument", connect.NewError(connect.CodeInvalidArgument, errors.New(`unknown input "watch_ids"`))},
		{"not-found", connect.NewError(connect.CodeNotFound, errors.New("declared program not found"))},
		{"failed-precondition", reproducedContractRejection},
	} {
		t.Run(tc.name, func(t *testing.T) {
			evaluator := NewProgramRuntimeEvaluator()
			evaluator.runner = &scriptedProgramRunner{err: tc.err}
			decision, err := evaluator.Evaluate(context.Background(), EvaluationInput{Watch: evaluatorWatch(now), Now: now, ProposedCursor: "cursor-2"})
			if err != nil {
				t.Fatal(err)
			}
			if decision.GetDisposition() != domainpb.WatchDisposition_WATCH_DISPOSITION_UNAVAILABLE {
				t.Fatalf("disposition=%s", decision.GetDisposition())
			}
			if !strings.HasPrefix(decision.GetClassification(), classificationContractRejected+":") || !strings.Contains(decision.GetClassification(), tc.err.(*connect.Error).Message()) {
				t.Fatalf("classification %q must carry the runtime's message", decision.GetClassification())
			}
			if got := decision.GetNextWakeAt().AsTime().Sub(now); got != contractRejectedRetryInterval {
				t.Fatalf("parked wake = %s, want %s", got, contractRejectedRetryInterval)
			}
		})
	}
}

func TestTransportFailuresBackOffExponentiallyAndCap(t *testing.T) {
	now := time.Date(2026, 9, 9, 6, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		failures int
		want     time.Duration
	}{
		{0, 30 * time.Second}, {1, time.Minute}, {2, 2 * time.Minute}, {3, 4 * time.Minute}, {4, 8 * time.Minute}, {5, 15 * time.Minute}, {12, 15 * time.Minute},
	} {
		for _, cause := range []error{connect.NewError(connect.CodeUnavailable, errors.New("dial tcp 127.0.0.1:19843: connection refused")), errors.New("plain transport failure"), context.DeadlineExceeded} {
			evaluator := NewProgramRuntimeEvaluator()
			evaluator.runner = &scriptedProgramRunner{err: cause}
			decision, err := evaluator.Evaluate(context.Background(), EvaluationInput{Watch: evaluatorWatch(now), Now: now, ProposedCursor: "cursor-2", ConsecutiveFailures: tc.failures})
			if err != nil {
				t.Fatal(err)
			}
			if decision.GetClassification() != classificationRuntimeUnavailable {
				t.Fatalf("classification=%q for %v", decision.GetClassification(), cause)
			}
			if got := decision.GetNextWakeAt().AsTime().Sub(now); got != tc.want {
				t.Fatalf("failures=%d wake=%s want %s", tc.failures, got, tc.want)
			}
		}
	}
}

type recordingHandler struct {
	slog.Handler
	records []slog.Record
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}
func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *recordingHandler) WithGroup(string) slog.Handler      { return h }

func TestFailureIsWarnedOnceWithMessageThenDemoted(t *testing.T) {
	now := time.Date(2026, 9, 9, 6, 0, 0, 0, time.UTC)
	handler := &recordingHandler{}
	evaluator := NewProgramRuntimeEvaluator()
	evaluator.runner = &scriptedProgramRunner{err: reproducedContractRejection}
	evaluator.log = slog.New(handler)
	first, err := evaluator.Evaluate(context.Background(), EvaluationInput{Watch: evaluatorWatch(now), Now: now, ProposedCursor: "cursor-2"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = evaluator.Evaluate(context.Background(), EvaluationInput{Watch: evaluatorWatch(now), Now: now, ProposedCursor: "cursor-2", ConsecutiveFailures: 1, LastFailureReason: first.GetClassification()}); err != nil {
		t.Fatal(err)
	}
	if len(handler.records) != 2 || handler.records[0].Level != slog.LevelWarn || handler.records[1].Level != slog.LevelDebug {
		t.Fatalf("records=%+v", handler.records)
	}
	var message string
	handler.records[0].Attrs(func(a slog.Attr) bool {
		if a.Key == "error" {
			message = a.Value.String()
		}
		return true
	})
	if !strings.Contains(message, "pinned program artifact unavailable") {
		t.Fatalf("first warning must include the runtime message, got %q", message)
	}
}

func retryFixture(t *testing.T, runner *scriptedProgramRunner) (*Repository, *Service, *Scheduler, *domainpb.CohortWatch, *time.Time) {
	t.Helper()
	repo, _ := testRepository(t)
	runID := uuid.New()
	service := NewService(repo, &cohortSource{retention: eventlog.RetentionState{Generation: 1}})
	watch, _, err := service.Create(context.Background(), &domainpb.CreateCohortWatchRequest{Spec: validServiceSpec(runID), IdempotencyKey: "retry-" + t.Name()})
	if err != nil {
		t.Fatal(err)
	}
	evaluator := NewProgramRuntimeEvaluator()
	evaluator.runner = runner
	processor := NewProcessor(service, summaryResolver{summaries: []SubjectSummary{{RunID: runID.String(), Status: "running"}}}, evaluator)
	clock := watch.GetNextWakeAt().AsTime()
	processor.now = func() time.Time { return clock }
	repo.now = processor.now
	scheduler := NewScheduler(repo, processor, func(err error) { t.Errorf("scheduler error: %v", err) })
	scheduler.now = processor.now
	return repo, service, scheduler, watch, &clock
}

func TestSchedulerDoesNotReplayContractRejectedWatchOnTheNextTick(t *testing.T) {
	runner := &scriptedProgramRunner{err: reproducedContractRejection}
	repo, _, scheduler, watch, clock := retryFixture(t, runner)
	first := *clock
	if processed, err := scheduler.RecoverOnce(context.Background()); err != nil || processed != 1 || runner.calls != 1 {
		t.Fatalf("processed=%d calls=%d err=%v", processed, runner.calls, err)
	}
	loaded, checkpoint, err := repo.Get(context.Background(), watch.GetWatchId())
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.GetNextWakeAt().AsTime(); !got.Equal(first.Add(contractRejectedRetryInterval)) {
		t.Fatalf("next wake %s, want %s", got, first.Add(contractRejectedRetryInterval))
	}
	if checkpoint.ConsecutiveFailures != 1 || !strings.HasPrefix(checkpoint.LastFailureReason, classificationContractRejected) || !strings.HasPrefix(loaded.GetLastDecision().GetClassification(), classificationContractRejected) {
		t.Fatalf("checkpoint=%+v decision=%q", checkpoint, loaded.GetLastDecision().GetClassification())
	}
	for _, step := range []time.Duration{30 * time.Second, 5 * time.Minute, 59 * time.Minute} {
		*clock = first.Add(step)
		if processed, err := scheduler.RecoverOnce(context.Background()); err != nil || processed != 0 || runner.calls != 1 {
			t.Fatalf("after %s: processed=%d calls=%d err=%v", step, processed, runner.calls, err)
		}
	}
	*clock = first.Add(contractRejectedRetryInterval)
	if processed, err := scheduler.RecoverOnce(context.Background()); err != nil || processed != 1 || runner.calls != 2 {
		t.Fatalf("hourly retry: processed=%d calls=%d err=%v", processed, runner.calls, err)
	}
	loaded, checkpoint, err = repo.Get(context.Background(), watch.GetWatchId())
	if err != nil || checkpoint.ConsecutiveFailures != 2 || !loaded.GetNextWakeAt().AsTime().Equal(clock.Add(contractRejectedRetryInterval)) {
		t.Fatalf("replayed rejection must still move the wake: checkpoint=%+v wake=%s err=%v", checkpoint, loaded.GetNextWakeAt().AsTime(), err)
	}
}

func TestSchedulerBacksOffTransportFailuresAndResetsAfterSuccess(t *testing.T) {
	runner := &scriptedProgramRunner{err: connect.NewError(connect.CodeUnavailable, errors.New("connection refused"))}
	repo, _, scheduler, watch, clock := retryFixture(t, runner)
	for attempt, want := range []time.Duration{30 * time.Second, time.Minute, 2 * time.Minute, 4 * time.Minute} {
		due := *clock
		if processed, err := scheduler.RecoverOnce(context.Background()); err != nil || processed != 1 || runner.calls != attempt+1 {
			t.Fatalf("attempt %d: processed=%d calls=%d err=%v", attempt, processed, runner.calls, err)
		}
		loaded, checkpoint, err := repo.Get(context.Background(), watch.GetWatchId())
		if err != nil {
			t.Fatal(err)
		}
		if got := loaded.GetNextWakeAt().AsTime().Sub(due); got != want || checkpoint.ConsecutiveFailures != attempt+1 || checkpoint.LastFailureReason != classificationRuntimeUnavailable {
			t.Fatalf("attempt %d: wake +%s want +%s checkpoint=%+v", attempt, got, want, checkpoint)
		}
		if loaded.GetCursor().GetToken() != watch.GetCursor().GetToken() {
			t.Fatalf("attempt %d consumed the cursor", attempt)
		}
		// One tick early the watch is not due; the storm was every tick.
		*clock = due.Add(want - time.Second)
		if processed, err := scheduler.RecoverOnce(context.Background()); err != nil || processed != 0 {
			t.Fatalf("attempt %d replayed before its backoff: processed=%d err=%v", attempt, processed, err)
		}
		*clock = due.Add(want)
	}
	runner.err = nil
	if processed, err := scheduler.RecoverOnce(context.Background()); err != nil || processed != 1 {
		t.Fatalf("recovery: processed=%d err=%v", processed, err)
	}
	loaded, checkpoint, err := repo.Get(context.Background(), watch.GetWatchId())
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.ConsecutiveFailures != 0 || checkpoint.LastFailureReason != "" || loaded.GetLastDecision().GetDisposition() != domainpb.WatchDisposition_WATCH_DISPOSITION_QUIET || loaded.GetCursor().GetToken() == watch.GetCursor().GetToken() {
		t.Fatalf("success must clear the streak and advance the cursor: checkpoint=%+v decision=%+v", checkpoint, loaded.GetLastDecision())
	}
	runner.err = connect.NewError(connect.CodeUnavailable, errors.New("connection refused again"))
	*clock = loaded.GetNextWakeAt().AsTime()
	if _, err := scheduler.RecoverOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if loaded, _, err = repo.Get(context.Background(), watch.GetWatchId()); err != nil || loaded.GetNextWakeAt().AsTime().Sub(*clock) != unavailableBaseBackoff {
		t.Fatalf("a new outage restarts from the base backoff: wake=%s err=%v", loaded.GetNextWakeAt().AsTime().Sub(*clock), err)
	}
}

// contractInputSpec mirrors program-runtime's contracts.InputSpec. Its
// ResolveInputs lives in an internal package of another module, so the
// declared JSON is the shared source of truth this test checks against.
type contractInputSpec struct {
	Type     string          `json:"type"`
	Required bool            `json:"required"`
	Default  json.RawMessage `json:"default"`
	Enum     []any           `json:"enum"`
}

func loadSupervisionContractInputs(t *testing.T) map[string]contractInputSpec {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "program-runtime", "supervision-evaluate.json"))
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Name   string                       `json:"name"`
		Inputs map[string]contractInputSpec `json:"inputs"`
	}
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatal(err)
	}
	if contract.Name != supervisionProgramName {
		t.Fatalf("contract name %q, evaluator calls %q", contract.Name, supervisionProgramName)
	}
	return contract.Inputs
}

// matchesContractInputType applies the same shallow checks as program-runtime
// contracts.matchesInputType to the JSON-decoded value a Struct carries.
func matchesContractInputType(value any, kind string) bool {
	switch kind {
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "number":
		_, ok := value.(float64)
		return ok
	case "integer":
		n, ok := value.(float64)
		return ok && !math.IsNaN(n) && !math.IsInf(n, 0) && math.Trunc(n) == n
	case "array":
		_, ok := value.([]any)
		return ok
	case "object":
		_, ok := value.(map[string]any)
		return ok
	default:
		return false
	}
}

func TestProgramInputMatchesDeclaredContract(t *testing.T) {
	now := time.Date(2026, 9, 9, 6, 0, 0, 0, time.UTC)
	inputs := loadSupervisionContractInputs(t)
	evaluator := NewProgramRuntimeEvaluator()
	watch := evaluatorWatch(now)
	watch.LastDecision = &domainpb.WatchDecision{DecisionId: "decision-1", Disposition: domainpb.WatchDisposition_WATCH_DISPOSITION_QUIET, Classification: "quiet"}
	runID := uuid.New()
	input := EvaluationInput{
		Watch: watch, Now: now, ProposedCursor: "cursor-2", Reset: false, ResetFrom: 1, ResetTo: 1,
		Events:   []eventlog.CohortEvent{{ID: uuid.New(), RunID: runID, Sequence: 7, EventType: domain.EventTypeStatus, Timestamp: now.Add(-time.Minute)}},
		Subjects: []SubjectSummary{{RunID: runID.String(), Status: "running", Friction: []FrictionSummary{{EvidenceID: "friction-1", Score: 0.9, Pattern: "retry loop", Fingerprint: "fp", Owner: "agent-manager"}}}},
	}
	built, err := evaluator.programInput(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	structured, err := structpb.NewStruct(built)
	if err != nil {
		t.Fatal(err)
	}
	provided := structured.AsMap()
	for name := range provided {
		if _, ok := inputs[name]; !ok {
			t.Errorf("unknown input %q", name)
		}
	}
	for name, spec := range inputs {
		value, ok := provided[name]
		if !ok && len(spec.Default) > 0 {
			if err := json.Unmarshal(spec.Default, &value); err != nil {
				t.Fatal(err)
			}
			ok = true
		}
		if !ok {
			if spec.Required {
				t.Errorf("missing required input %q", name)
			}
			continue
		}
		if !matchesContractInputType(value, spec.Type) {
			t.Errorf("input %q must have type %s, got %T", name, spec.Type, value)
		}
	}
}
