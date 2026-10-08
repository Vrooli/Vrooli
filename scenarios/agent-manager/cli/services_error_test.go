package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/cli-core/cliutil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func newFailingServices(t *testing.T) *Services {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL}
	}, nil)
	return NewServices(api)
}

func TestServicesPropagateAPITransportFailuresWithoutFabricatingResponses(t *testing.T) {
	s := newFailingServices(t)
	if _, response, err := s.Workflows.Validate(&apipb.ValidateWorkflowRequest{}); err == nil || response != nil {
		t.Fatal("workflow validation hid API failure")
	}
	if _, response, err := s.Workflows.StartExecution(&apipb.StartWorkflowExecutionRequest{}); err == nil || response != nil {
		t.Fatal("workflow start hid API failure")
	}
	if _, response, err := s.Workflows.Execution("execution", true); err == nil || response != nil {
		t.Fatal("workflow advance hid API failure")
	}
	if _, response, err := s.Workflows.Wait("execution", 10); err == nil || response != nil {
		t.Fatal("workflow wait hid API failure")
	}
	if _, response, err := s.Profiles.Create(&domainpb.AgentProfile{}); err == nil || response != nil {
		t.Fatal("profile create hid API failure")
	}
	if _, response, err := s.Profiles.Update("profile", &domainpb.AgentProfile{}); err == nil || response != nil {
		t.Fatal("profile update hid API failure")
	}
	if _, response, err := s.Profiles.Ensure(&apipb.EnsureProfileRequest{}); err == nil || response != nil {
		t.Fatal("profile ensure hid API failure")
	}
	if _, response, err := s.Tasks.Create(&domainpb.Task{}); err == nil || response != nil {
		t.Fatal("task create hid API failure")
	}
	if _, response, err := s.Tasks.Update("task", &domainpb.Task{}); err == nil || response != nil {
		t.Fatal("task update hid API failure")
	}
	if _, response, err := s.Runs.Create(&apipb.CreateRunRequest{}); err == nil || response != nil {
		t.Fatal("run create hid API failure")
	}
	if _, response, err := s.Runs.Continue("run", &domainpb.ContinueRunRequest{}); err == nil || response != nil {
		t.Fatal("run continue hid API failure")
	}
	if _, response, err := s.Runs.Park("run", &domainpb.ParkRunRequest{}); err == nil || response != nil {
		t.Fatal("run park hid API failure")
	}
	if _, response, err := s.Runs.Wake("run", &domainpb.WakeRunRequest{}); err == nil || response != nil {
		t.Fatal("run wake hid API failure")
	}
	if _, response, err := s.Runs.StopAll(&apipb.StopAllRunsRequest{}); err == nil || response != nil {
		t.Fatal("stop all hid API failure")
	}
	if _, response, err := s.Runs.Quiesce(&apipb.QuiesceScenarioRequest{}); err == nil || response != nil {
		t.Fatal("quiesce hid API failure")
	}
	if _, response, err := s.Runs.Approve("run", &apipb.ApproveRunRequest{}); err == nil || response != nil {
		t.Fatal("approve hid API failure")
	}
	if _, response, err := s.Runners.GetStatus(); err == nil || response != nil {
		t.Fatal("runner status hid API failure")
	}
	if _, response, err := s.Runners.Probe("codex"); err == nil || response != nil {
		t.Fatal("runner probe hid API failure")
	}
	if _, response, err := s.Policy.Validate(); err == nil || response != nil {
		t.Fatal("role-policy validation hid API failure")
	}
	if _, response, err := s.PermissionPolicy.Reconcile(&apipb.ReconcilePermissionPolicyRequest{}); err == nil || response != nil {
		t.Fatal("permission-policy reconcile hid API failure")
	}
}
