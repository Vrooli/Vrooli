package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	clitest "agent-manager/cli/internal/testutil"

	"github.com/vrooli/cli-core/cliutil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestWorkflowWaitOutlivesOrdinaryHTTPTimeoutWithoutChangingOtherCalls(t *testing.T) {
	for _, timeoutSeconds := range []int{0, 30} {
		t.Run((time.Duration(timeoutSeconds) * time.Second).String(), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workflow-executions/execution-1/wait" || r.Header.Get("Authorization") != "Bearer saved-token" || r.Header.Get(cliutil.HeaderInvocationCommand) != "workflow execution-wait" {
					t.Errorf("lost wait request contract: %s %s %+v", r.Method, r.URL.Path, r.Header)
				}
				var request apipb.WaitWorkflowExecutionRequest
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.ExecutionId != "execution-1" || request.TimeoutSeconds != int32(timeoutSeconds) {
					t.Errorf("changed server wait bound: %+v %v", request, err)
				}
				select {
				case <-time.After(60 * time.Millisecond):
				case <-r.Context().Done():
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"execution":{"id":"execution-1","status":"WORKFLOW_EXECUTION_STATUS_SUCCEEDED"}}`))
			}))
			defer server.Close()
			base := cliutil.NewHTTPClient(cliutil.HTTPClientOptions{Timeout: 10 * time.Millisecond})
			base.SetInvocationHeaderSource(func() map[string]string {
				return map[string]string{cliutil.HeaderInvocationCommand: "workflow execution-wait"}
			})
			api := cliutil.NewAPIClient(base, func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL} }, func() string { return "saved-token" })
			_, response, err := NewServices(api).Workflows.Wait("execution-1", timeoutSeconds)
			if err != nil || response == nil || response.Execution.GetId() != "execution-1" || response.Execution.GetStatus() != domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED || response.TimedOut {
				t.Fatalf("server-owned wait inherited ordinary deadline: %+v %v", response, err)
			}
			if calls.Load() != 1 || base.Timeout() != 10*time.Millisecond {
				t.Fatalf("wait repeated or changed ordinary request timeout: calls=%d timeout=%s", calls.Load(), base.Timeout())
			}
		})
	}
}

func newContractServices(t *testing.T) (*Services, *clitest.RecordingServer) {
	t.Helper()
	server := clitest.NewRecordingServer(t, `{}`)
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL()}
	}, nil)
	return NewServices(api), server
}

func TestServicesUseDocumentedWorkflowAndRunHTTPContracts(t *testing.T) {
	s, recorder := newContractServices(t)
	workflow := &apipb.StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: "workflow"}
	if _, _, err := s.Workflows.Validate(&apipb.ValidateWorkflowRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Workflows.Reconcile("/api/v1/workflows/reconcile", &apipb.ReconcileScenarioWorkflowsRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Workflows.List("owner", "workflow", 10, 2); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Workflows.Get("/api/v1/workflows/active", "owner", "workflow", "digest"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Workflows.StartExecution(workflow); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Workflows.ListExecutions("owner", "workflow", "running", 10, 2); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Workflows.ExecutionResult("execution-1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Workflows.Signal(&apipb.SignalWorkflowExecutionRequest{ExecutionId: "execution-1"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Workflows.Control("cancel", &apipb.WorkflowExecutionOperationRequest{ExecutionId: "execution-1"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Workflows.Execution("execution-1", false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Workflows.Execution("execution-1", true); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Workflows.Wait("execution-1", 30); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Workflows.Trace("execution-1", 3, 10); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Workflows.ExecutionRuns("execution-1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Workflows.Simulate(&apipb.SimulateWorkflowRequest{}); err != nil {
		t.Fatal(err)
	}

	if _, _, err := s.Runs.List(10, 2, "task", "profile", "running", "tag"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Runs.Get("run-1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Runs.Create(&apipb.CreateRunRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Runs.Stop("run-1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Runs.GetByTag("tag"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Runs.StopByTag("tag"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Runs.StopAll(&apipb.StopAllRunsRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Runs.Quiesce(&apipb.QuiesceScenarioRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Runs.Approve("run-1", &apipb.ApproveRunRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Runs.Reject("run-1", &apipb.RejectRunRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Runs.GetDiff("run-1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Runs.GetEvents("run-1", 10, ptr(int64(3))); err != nil {
		t.Fatal(err)
	}
	if err := s.Runs.Delete("run-1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Runs.Continue("run-1", &domainpb.ContinueRunRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Runs.Park("run-1", &domainpb.ParkRunRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Runs.Wake("run-1", &domainpb.WakeRunRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Runs.AwaitResult("run-1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Runs.Recover("run-1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Runs.InvestigationApply(json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Runs.SandboxSync("run-1", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}

	requests := recorder.Requests()
	if len(requests) < 35 {
		t.Fatalf("recorded %d requests, want broad service coverage", len(requests))
	}
	if requests[0].Path != "/api/v1/workflows/validate" || requests[10].Method != http.MethodPost {
		t.Fatalf("workflow routes=%+v", requests[:11])
	}
}

func TestServicesUseDocumentedConfigurationAndReadSideContracts(t *testing.T) {
	s, recorder := newContractServices(t)
	if _, _, err := s.Declarations.Reconcile("/api/v1/declarations/reconcile", &apipb.ReconcileScenarioDeclarationsRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Profiles.List(10, 1); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Profiles.Get("profile-1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Profiles.Create(&domainpb.AgentProfile{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Profiles.Update("profile-1", &domainpb.AgentProfile{}); err != nil {
		t.Fatal(err)
	}
	if err := s.Profiles.Delete("profile-1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Profiles.Ensure(&apipb.EnsureProfileRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Profiles.ReconcileScenario(&apipb.ReconcileScenarioProfilesRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Tasks.List(10, 1, "queued"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Tasks.Get("task-1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Tasks.Create(&domainpb.Task{}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Tasks.Cancel("task-1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Tasks.Update("task-1", &domainpb.Task{}); err != nil {
		t.Fatal(err)
	}
	if err := s.Tasks.Delete("task-1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Runners.GetStatus(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Runners.Probe("codex"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Policy.Status(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Policy.Catalog(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Policy.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Policy.Reload(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Policy.Explain(&apipb.ExplainRolePolicyRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.PermissionPolicy.Status(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.PermissionPolicy.Catalog(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.PermissionPolicy.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.PermissionPolicy.Reload(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.PermissionPolicy.Plan(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.PermissionPolicy.Reconcile(&apipb.ReconcilePermissionPolicyRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.PermissionPolicy.Doctor(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Settings.GetInvestigation(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Settings.UpdateInvestigation(json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Settings.ResetInvestigation(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Maintenance.Purge(&apipb.PurgeDataRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Operational.GetOperational("runner"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Operational.GetFallback(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.HealthAudit.GetModels(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.HealthAudit.GetRunners(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.HealthAudit.QueryAudit(AuditQuery{Scope: "runner", Runner: "codex", Limit: 10}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Events.List(EventsQuery{Run: "run-1", Type: "tool_call", Limit: 10}); err != nil {
		t.Fatal(err)
	}
	if requests := recorder.Requests(); len(requests) < 35 {
		t.Fatalf("recorded %d requests", len(requests))
	}
}

func ptr[T any](value T) *T { return &value }
