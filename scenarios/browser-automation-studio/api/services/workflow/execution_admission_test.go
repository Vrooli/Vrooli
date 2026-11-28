package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/browser-automation-studio/automation/executor"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
)

type testExecutionAdmission struct {
	identity string
	err      error
	calls    int
}

func (a *testExecutionAdmission) ReserveWorkflowExecution(_ context.Context, identity string) (*monetization.QuotaHold, error) {
	a.calls++
	a.identity = identity
	return nil, a.err
}

func TestOrdinaryAdhocCannotUseCaptureServiceAdmission(t *testing.T) {
	denied := errors.New("account is required")
	admission := &testExecutionAdmission{err: denied}
	svc, repo, _ := newCompletionFixture(t, "saved", func(context.Context, executor.Request) error { return nil })
	svc.executionAdmission = admission
	ctx := captureServicePrincipalContext()
	workflow, err := svc.GetWorkflowAPI(ctx, &basapi.GetWorkflowRequest{WorkflowId: repo.workflow.ID.String()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.ExecuteAdhocWorkflowAPI(ctx, &basexecution.ExecuteAdhocRequest{FlowDefinition: workflow.Workflow.FlowDefinition, WaitForCompletion: true})
	if !errors.Is(err, denied) {
		t.Fatalf("ordinary adhoc error = %v, want customer quota admission", err)
	}
	if admission.calls != 1 || admission.identity != "" {
		t.Fatalf("ordinary adhoc admission calls=%d identity=%q; want one empty-identity denial", admission.calls, admission.identity)
	}
}

func TestSystemCaptureUsesGlobalSlotWithoutCustomerIdentityOrQuota(t *testing.T) {
	denied := errors.New("customer quota must not be used for system capture")
	admission := &testExecutionAdmission{err: denied}
	var svc *WorkflowService
	slotObserved, identityObserved := false, ""
	svc, repo, _ := newCompletionFixture(t, "saved", func(ctx context.Context, _ executor.Request) error {
		identityObserved = monetization.UserIdentityFromContext(ctx)
		svc.executionLifecycleMu.Lock()
		slotObserved = svc.executionSlotsInUse == 1
		svc.executionLifecycleMu.Unlock()
		return nil
	})
	svc.executionAdmission = admission
	svc.maxConcurrentExecutions = 1
	ctx := captureServicePrincipalContext()
	workflow, err := svc.GetWorkflowAPI(ctx, &basapi.GetWorkflowRequest{WorkflowId: repo.workflow.ID.String()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.ExecuteSystemCaptureAdhocWorkflowAPIWithOptions(ctx, &basexecution.ExecuteAdhocRequest{FlowDefinition: workflow.Workflow.FlowDefinition, WaitForCompletion: true}, nil)
	if err != nil {
		t.Fatalf("verified system capture failed: %v", err)
	}
	if admission.calls != 0 {
		t.Fatalf("system capture entered customer quota admission %d times", admission.calls)
	}
	if identityObserved != "" {
		t.Fatalf("system capture had customer identity %q", identityObserved)
	}
	if !slotObserved {
		t.Fatal("system capture did not hold the global execution slot while running")
	}
}

func captureServicePrincipalContext() context.Context {
	return apiidentity.WithPrincipal(context.Background(), apiidentity.Principal{
		Kind: apiidentity.ActorService, Subject: "experience-manager-bas-capture", Verified: true,
		Issuer: "scenario-authenticator", Audience: "scenario-authenticator:browser-automation-studio",
		Scopes: []string{"browser-automation-studio:capture"},
	})
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
