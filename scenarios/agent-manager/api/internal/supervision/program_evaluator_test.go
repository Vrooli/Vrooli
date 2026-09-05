package supervision

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeDeclaredProgramRunner struct {
	stdout string
	req    *libraryv1.RunDeclaredProgramRequest
}

func (f *fakeDeclaredProgramRunner) RunDeclaredProgram(_ context.Context, req *connect.Request[libraryv1.RunDeclaredProgramRequest]) (*connect.Response[libraryv1.RunDeclaredProgramResponse], error) {
	f.req = req.Msg
	return connect.NewResponse(&libraryv1.RunDeclaredProgramResponse{Terminal: true, Program: &programsv1.Program{Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, Stdout: f.stdout}}), nil
}

func evaluatorWatch(now time.Time) *domainpb.CohortWatch {
	return &domainpb.CohortWatch{
		Cursor:    &domainpb.WatchCursor{Token: "cursor-1"},
		UpdatedAt: timestamppb.New(now.Add(-2 * time.Minute)),
		Spec: &domainpb.WatchSpec{PolicyVersion: "policy-v1", Triggers: &domainpb.WatchTriggers{
			EventCount: 3, QuietTime: durationpb.New(time.Minute), Terminal: true,
		}},
	}
}

func TestProgramRuntimeEvaluatorBuildsExactTriggerPolicyAndMapsDecision(t *testing.T) {
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	runner := &fakeDeclaredProgramRunner{stdout: `{"status":"ok","signals":{"disposition":"signal","classification":"quiet_time","confidence":1,"abstained":false,"recommended_action":"escalate","next_cursor":"cursor-2","cursor_reset":false,"wake_condition":{"kind":"immediate","after_seconds":0},"policy_version":"policy-v1","inference_calls":0},"evidence":["evt-1"]}`}
	evaluator := NewProgramRuntimeEvaluator()
	evaluator.runner = runner
	event := eventlog.CohortEvent{ID: uuid.New(), RunID: uuid.New(), EventType: domain.EventTypeStatus, Timestamp: now.Add(-2 * time.Minute)}
	decision, err := evaluator.Evaluate(context.Background(), EvaluationInput{Watch: evaluatorWatch(now), Events: []eventlog.CohortEvent{event}, Now: now, ProposedCursor: "cursor-2"})
	if err != nil || decision.GetDisposition() != domainpb.WatchDisposition_WATCH_DISPOSITION_SIGNAL || decision.GetClassification() != "quiet_time" {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	policy := runner.req.GetInputs().AsMap()["policy"].(map[string]any)
	if policy["event_count_enabled"] != true || policy["friction_enabled"] != false || policy["terminal_enabled"] != true || policy["quiet_reached"] != true {
		t.Fatalf("policy flags=%+v", policy)
	}
}

func TestProgramRuntimeEvaluatorRejectsCursorAndPolicyContractDrift(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct{ name, stdout string }{
		{"cursor", `{"status":"ok","signals":{"disposition":"quiet","classification":"quiet","recommended_action":"park","next_cursor":"invented","cursor_reset":false,"wake_condition":{"kind":"after","after_seconds":30},"policy_version":"policy-v1","inference_calls":0}}`},
		{"policy", `{"status":"ok","signals":{"disposition":"quiet","classification":"quiet","recommended_action":"park","next_cursor":"cursor-2","cursor_reset":false,"wake_condition":{"kind":"after","after_seconds":30},"policy_version":"wrong","inference_calls":0}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			evaluator := NewProgramRuntimeEvaluator()
			evaluator.runner = &fakeDeclaredProgramRunner{stdout: tc.stdout}
			if _, err := evaluator.Evaluate(context.Background(), EvaluationInput{Watch: evaluatorWatch(now), Now: now, ProposedCursor: "cursor-2"}); err == nil {
				t.Fatal("expected contract drift to be rejected")
			}
		})
	}
}

func TestProgramRuntimeEvaluatorCannotInventTerminalCohort(t *testing.T) {
	for _, subjects := range [][]SubjectSummary{nil, {{RunID: "running", Terminal: false}}, {{RunID: "done", Terminal: true}, {RunID: "running", Terminal: false}}} {
		evaluator := NewProgramRuntimeEvaluator()
		evaluator.runner = &fakeDeclaredProgramRunner{stdout: `{"status":"ok","signals":{"disposition":"terminal","classification":"completed","confidence":0.99,"recommended_action":"wake_parent","next_cursor":"cursor-2","wake_condition":{"kind":"terminal"},"policy_version":"policy-v1","inference_calls":1}}`}
		now := time.Now()
		if _, err := evaluator.Evaluate(context.Background(), EvaluationInput{Watch: evaluatorWatch(now), Now: now, Subjects: subjects, ProposedCursor: "cursor-2"}); err == nil {
			t.Fatal("accepted invented terminal cohort")
		}
	}
}
