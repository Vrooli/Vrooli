package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/executor"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
)

type testExecutionAdmission struct {
	identity string
	err      error
}

func (a *testExecutionAdmission) ReserveWorkflowExecution(_ context.Context, identity string) (*monetization.QuotaHold, error) {
	a.identity = identity
	return nil, a.err
}

func TestExecuteWorkflowAPIRequiresConfiguredAdmissionBeforeCreatingExecution(t *testing.T) {
	denied := errors.New("limit reached")
	admission := &testExecutionAdmission{err: denied}
	svc, repo, _ := newCompletionFixture(t, "saved", func(context.Context, executor.Request) error { return nil })
	svc.executionAdmission = admission
	ctx := monetization.WithUserIdentity(context.Background(), "alice@example.com")
	_, err := svc.ExecuteWorkflowAPIWithOptions(ctx, &basapi.ExecuteWorkflowRequest{WorkflowId: repo.workflow.ID.String()}, nil)
	if !errors.Is(err, denied) {
		t.Fatalf("execution error = %v, want admission denial", err)
	}
	if admission.identity != "alice@example.com" {
		t.Fatalf("admission identity = %q", admission.identity)
	}
	if repo.index.ID != (uuid.UUID{}) {
		t.Fatalf("denied request created execution row: %+v", repo.index)
	}
}
