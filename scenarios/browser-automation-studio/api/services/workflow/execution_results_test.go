package workflow

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/events"
	"github.com/vrooli/browser-automation-studio/automation/executor"
	"github.com/vrooli/browser-automation-studio/database"
	uxcollector "github.com/vrooli/browser-automation-studio/services/uxmetrics/collector"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

type lifecycleRepository struct {
	database.Repository
	execution *database.ExecutionIndex
}

func (r *lifecycleRepository) GetExecution(context.Context, uuid.UUID) (*database.ExecutionIndex, error) {
	return r.execution, nil
}

func (r *lifecycleRepository) UpdateExecutionStatus(_ context.Context, _ uuid.UUID, status string, _ *string, _ *time.Time, _ time.Time) error {
	r.execution.Status = status
	return nil
}

type lifecycleSink struct {
	*events.MemorySink
	closed []uuid.UUID
}

func (s *lifecycleSink) CloseExecution(id uuid.UUID) {
	s.closed = append(s.closed, id)
	s.MemorySink.CloseExecution(id)
}

type lifecycleExecutor struct{ err error }

func (e lifecycleExecutor) Execute(context.Context, executor.Request) error { return e.err }

// [REQ:BAS-RH-J07] The workflow owner retires decorated sinks on all exits.
func TestWorkflowClosesDecoratedSinkOnEveryExit(t *testing.T) {
	for _, mode := range []string{"fresh", "resumed"} {
		for _, outcome := range []string{"completed", "failed", "cancelled", "compile-failed"} {
			t.Run(mode+"/"+outcome, func(t *testing.T) {
				id, workflowID := uuid.New(), uuid.New()
				repo := &lifecycleRepository{execution: &database.ExecutionIndex{ID: id, WorkflowID: workflowID}}
				sink := &lifecycleSink{MemorySink: events.NewMemorySink(contracts.DefaultEventBufferLimits)}
				runner := lifecycleExecutor{}
				if outcome == "failed" {
					runner.err = errors.New("synthetic action failure")
				} else if outcome == "cancelled" {
					runner.err = context.Canceled
				}
				service := &WorkflowService{
					repo: repo, executor: runner, executionDataRoot: t.TempDir(),
					eventSinkFactory: func() events.Sink { return uxcollector.NewCollector(sink, nil) },
				}
				workflow := &basapi.WorkflowSummary{Id: workflowID.String(), FlowDefinition: &basworkflows.WorkflowDefinitionV2{
					Nodes: []*basworkflows.WorkflowNodeV2{{Id: "navigate", Action: &basactions.ActionDefinition{
						Type:   basactions.ActionType_ACTION_TYPE_NAVIGATE,
						Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://fixture.invalid"}},
					}}},
				}}
				if outcome == "compile-failed" {
					workflow.FlowDefinition = nil
				}
				if mode == "resumed" {
					service.executeResumedWorkflowAsync(context.Background(), workflow, id, &CheckpointState{}, nil, nil, "")
				} else {
					service.executeWorkflowAsyncWithOptions(context.Background(), workflow, id, nil, nil, nil, nil, nil, nil, nil, "", "", "", false, nil, "", nil)
				}
				if len(sink.closed) != 1 || sink.closed[0] != id {
					t.Fatalf("sink closed for %v; wanted exactly %s", sink.closed, id)
				}
				want := outcome
				if outcome == "compile-failed" {
					want = "failed"
				}
				if repo.execution.Status != want {
					t.Fatalf("execution status = %s, want %s", repo.execution.Status, want)
				}
			})
		}
	}
}

func TestRequiredVideoArtifactContract(t *testing.T) {
	if err := requiredVideoArtifactError(true, nil, nil); err == nil {
		t.Fatal("required video with no artifact must fail")
	}
	if err := requiredVideoArtifactError(true, []ExecutionVideoArtifact{{ArtifactID: "video-1"}}, nil); err != nil {
		t.Fatalf("required video with an artifact failed: %v", err)
	}
	want := errors.New("artifact lookup failed")
	if err := requiredVideoArtifactError(true, nil, want); !errors.Is(err, want) {
		t.Fatalf("artifact lookup error = %v, want wrapped %v", err, want)
	}
	if err := requiredVideoArtifactError(false, nil, nil); err != nil {
		t.Fatalf("optional video must not fail: %v", err)
	}
}

func TestListExecutionArtifactsDoesNotExposeCapturePath(t *testing.T) {
	// enforces invariant: protectedEvidenceHasNoPublicLocation
	root := t.TempDir()
	executionID := uuid.New()
	dir := filepath.Join(root, executionID.String(), "artifacts", "har")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "capture.har"), []byte(`{"log":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	service := &WorkflowService{executionDataRoot: root}
	artifacts, err := service.listExecutionArtifacts(context.Background(), executionID, "har")
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("artifacts = %d", len(artifacts))
	}
	artifact := artifacts[0]
	if _, ok := artifact.Payload["path"]; ok {
		t.Fatalf("payload leaked local path: %#v", artifact.Payload)
	}
	if artifact.StorageURL != "" || artifact.Payload["access_policy"] != "ACCESS_POLICY_PROTECTED_STORAGE_ONLY" || artifact.Payload["sha256"] == "" {
		t.Fatalf("artifact evidence metadata = %#v", artifact)
	}
}

func TestExecutionOutcomeDistinguishesCancellationFromFailure(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, tc := range []struct {
		name string
		ctx  context.Context
		err  error
		want string
	}{
		{"success", context.Background(), nil, "completed"},
		{"completion wins stop race", cancelled, nil, "completed"},
		{"typed cancellation", context.Background(), context.Canceled, "cancelled"},
		{"driver error after cancellation", cancelled, errors.New("browser request interrupted"), "cancelled"},
		{"deadline is failure", context.Background(), context.DeadlineExceeded, "failed"},
		{"cancel text is not cancellation", context.Background(), errors.New("cancel button selector missing"), "failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, detail := executionOutcome(tc.ctx, tc.err)
			if got != tc.want {
				t.Fatalf("status = %q, want %q", got, tc.want)
			}
			if tc.want == "completed" && detail != "" {
				t.Fatalf("success has error %q", detail)
			}
			if tc.want == "failed" && detail != tc.err.Error() {
				t.Fatalf("failure detail lost: %q", detail)
			}
		})
	}
}
