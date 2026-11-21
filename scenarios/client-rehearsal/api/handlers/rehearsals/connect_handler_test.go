package rehearsals

import (
	"context"
	"io"
	"log"
	"testing"

	internal "client-rehearsal/internal/rehearsals"
	"connectrpc.com/connect"
	"github.com/google/uuid"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/rehearsals"
	"google.golang.org/protobuf/types/known/structpb"
)

type taskScopedServiceStub struct {
	task  internal.WorkspaceSandboxTaskRequest
	input internal.CreateInput
	got   bool
}

func (s *taskScopedServiceStub) CreateAndRun(_ context.Context, input internal.CreateInput) (internal.Rehearsal, error) {
	s.input = input
	return internal.Rehearsal{ID: "saved-run", Title: "synthetic route proof"}, nil
}
func (s *taskScopedServiceStub) CreateAndRunTaskScoped(_ context.Context, input internal.CreateInput, task internal.WorkspaceSandboxTaskRequest) (internal.TaskScopedRun, error) {
	s.task, s.input, s.got = task, input, true
	return internal.TaskScopedRun{Rehearsal: internal.Rehearsal{ID: "saved-run", Title: "synthetic route proof", HandoffMarkdown: "Companion route hits: 1"}}, nil
}

func TestCreateAndRunMapsSyntheticTriggerInputFromConnectRequest(t *testing.T) {
	service := &taskScopedServiceStub{}
	handler := NewConnectHandler(service, log.New(io.Discard, "", 0))
	triggerInput, err := structpb.NewStruct(map[string]any{"body": map[string]any{"id": "synthetic-event-1"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = handler.CreateAndRunRehearsal(context.Background(), connect.NewRequest(&v1.CreateAndRunRehearsalRequest{
		Title:        "synthetic trigger input",
		WorkflowJson: `{"nodes":[]}`,
		TriggerInput: triggerInput,
	}))
	if err != nil {
		t.Fatal(err)
	}
	body, ok := service.input.TriggerInput["body"].(map[string]any)
	if !ok || body["id"] != "synthetic-event-1" {
		t.Fatalf("mapped trigger input = %#v", service.input.TriggerInput)
	}
}
func (*taskScopedServiceStub) List(context.Context, int) ([]internal.Rehearsal, error) {
	return nil, nil
}
func (*taskScopedServiceStub) Get(context.Context, string) (internal.Rehearsal, error) {
	return internal.Rehearsal{}, nil
}
func (*taskScopedServiceStub) Review(context.Context, internal.ReviewInput) (internal.Rehearsal, error) {
	return internal.Rehearsal{}, nil
}

func TestCreateAndRunTaskScopedRequestMintsLogicalIdentityAndReturnsHandoff(t *testing.T) {
	service := &taskScopedServiceStub{}
	handler := NewConnectHandler(service, log.New(io.Discard, "", 0))
	response, err := handler.CreateAndRunRehearsal(context.Background(), connect.NewRequest(&v1.CreateAndRunRehearsalRequest{
		Title:      "synthetic route proof",
		TaskScoped: true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !service.got {
		t.Fatal("task-scoped request did not enter task-owned service path")
	}
	if _, err := uuid.Parse(service.task.TaskID); err != nil {
		t.Fatalf("server did not mint a UUID task identity: %q: %v", service.task.TaskID, err)
	}
	if service.task.Name != "client-rehearsal" {
		t.Fatalf("task name = %q, want client-rehearsal", service.task.Name)
	}
	if response.Msg.Rehearsal.GetHandoffMarkdown() != "Companion route hits: 1" {
		t.Fatalf("handoff lost route evidence: %q", response.Msg.Rehearsal.GetHandoffMarkdown())
	}
}
