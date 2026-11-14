package rehearsals

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const lifecycleSandboxID = "9c54f876-6b12-4a55-a11d-65e18af3d223"

func TestTaskScopedRunCreatesOwnsStartsAndVerifiesTeardown(t *testing.T) {
	root := t.TempDir()
	scope := filepath.Join(root, "task-1")
	if err := os.Mkdir(scope, 0o700); err != nil {
		t.Fatal(err)
	}
	owner := "trusted-service:task-1"
	profile, digest := lifecycleTestProfile(t)
	var companionHits atomic.Uint64
	companion := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Host != "crm.synthetic.invalid:18080" || r.URL.Path != "/v1/contacts" {
			t.Errorf("unexpected companion request: %s %s host=%q", r.Method, r.URL, r.Host)
		}
		companionHits.Add(1)
		_, _ = io.WriteString(w, `{"id":"synthetic-contact","status":"created"}`)
	}))
	defer companion.Close()
	companionURL, err := url.Parse(companion.URL)
	if err != nil {
		t.Fatal(err)
	}
	sandboxHTTPClient := &http.Client{Transport: aliasTestTransport{base: companionURL, inner: http.DefaultTransport}}
	var calls []string
	var eventOrder []string
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/task-scopes":
			eventOrder = append(eventOrder, "sandbox-create")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": lifecycleSandboxID, "name": "task-rehearsal", "projectRoot": root, "scopePath": scope, "owner": owner, "ownerType": "task", "idempotencyKey": "task-scope:task-1"})
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/companion-services"):
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/workflow-apis/start"):
			eventOrder = append(eventOrder, "companion-start")
			_ = json.NewEncoder(w).Encode(WorkspaceSandboxProfileReceipt{Spec: struct {
				Name    string `json:"name"`
				Kind    string `json:"kind"`
				Version string `json:"version"`
			}{Name: taskSandboxProfileName, Kind: "http-mock", Version: taskSandboxProfileVersion}, Status: "running", Addresses: map[string]string{"crm.synthetic.invalid": "http://crm.synthetic.invalid:18080"}, OperationAddresses: []HTTPMockOperationAddress{{Selector: profile.Operations[0].Selector, Method: "POST", URL: "http://crm.synthetic.invalid:18080/v1/contacts"}}, ProfileSHA256: digest})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sandboxes/"+lifecycleSandboxID && !deleted:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": lifecycleSandboxID, "name": "task-rehearsal", "projectRoot": root, "scopePath": scope, "owner": owner, "ownerType": "task", "idempotencyKey": "task-scope:task-1", "status": "active"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sandboxes/"+lifecycleSandboxID+"/companion-services/workflow-apis/route-hits":
			eventOrder = append(eventOrder, "route-hits")
			_ = json.NewEncoder(w).Encode(HTTPMockRouteHitsReceipt{SandboxID: lifecycleSandboxID, CompanionRuntimeID: "n8n-httpmock:" + lifecycleSandboxID, Status: "available", TotalHits: companionHits.Load(), Routes: []HTTPMockRouteHit{{Host: "crm.synthetic.invalid", Method: "POST", Path: "/v1/contacts", Count: companionHits.Load()}}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/workflow-apis/stop"):
			eventOrder = append(eventOrder, "companion-stop")
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sandboxes/"+lifecycleSandboxID:
			eventOrder = append(eventOrder, "sandbox-delete")
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sandboxes/"+lifecycleSandboxID && deleted:
			http.NotFound(w, r)
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()
	client, err := NewWorkspaceSandboxProfileClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	repo := newMemoryRepository()
	svc := NewServiceWithWorkspaceSandbox(repo, SyntheticEngine{}, time.Now, client)
	svc.(*service).sandboxHTTPDoer = sandboxHTTPClient
	workflow := `{"name":"Contact intake","nodes":[{"name":"Start","type":"n8n-nodes-base.manualTrigger","parameters":{}},{"name":"Create contact","type":"n8n-nodes-base.httpRequest","parameters":{"method":"POST","url":"https://crm.synthetic.invalid/v1/contacts"}}],"connections":{"Start":{"main":[[{"node":"Create contact"}]]}}}`
	run, err := svc.CreateAndRunTaskScoped(context.Background(), CreateInput{Title: "Synthetic contact intake", WorkflowJSON: workflow, SimulatorSourceJSON: `{"kind":"recorded-exchanges","exchanges":[{"method":"POST","host":"crm.synthetic.invalid","path":"/v1/contacts","status":201,"response":{"id":"contact-real-ish","status":"created"}}]}`}, WorkspaceSandboxTaskRequest{TaskID: "task-1", Name: "task-rehearsal"})
	if err != nil {
		t.Fatal(err)
	}
	if run.Rehearsal.State != StateNeedsReview || !strings.Contains(run.Rehearsal.HandoffMarkdown, "Workspace Sandbox task:") || !strings.Contains(run.Rehearsal.HandoffMarkdown, "Companion route-hit receipt: `available` (1 hit(s) across 1 route(s))") {
		t.Fatalf("task scoped run lacks handoff or expected state: %#v", run.Rehearsal)
	}
	if run.Lifecycle.SandboxID != lifecycleSandboxID || run.Lifecycle.Owner != owner || run.Lifecycle.ProfileSHA256 != digest || !run.Lifecycle.SyntheticTargets || run.Lifecycle.RouteHitsStatus != "available" || run.Lifecycle.RouteHits == nil || run.Lifecycle.RouteHits.TotalHits != 1 || companionHits.Load() != 1 || run.Lifecycle.TeardownStatus != "succeeded" || run.Lifecycle.CleanupConfirmedAt.IsZero() {
		t.Fatalf("incomplete lifecycle evidence: %#v", run.Lifecycle)
	}
	if stored := repo.items[run.Rehearsal.ID]; len(stored.Trace) == 0 || lifecycleEvidenceFromTrace(stored.Trace).TeardownStatus != "succeeded" {
		t.Fatalf("lifecycle receipt was not persisted in trace: %#v", stored.Trace)
	}
	if len(calls) != 8 || !deleted {
		t.Fatalf("unexpected lifecycle path: %#v; deleted=%t", calls, deleted)
	}
	deleted = false
	calls = nil
	eventOrder = nil
	companionHits.Store(0)
	runtime := &taskN8NRuntimeFake{events: &eventOrder, routeClient: sandboxHTTPClient}
	nativeService := NewServiceWithNativeN8N(newMemoryRepository(), SyntheticEngine{}, time.Now, client, runtime)
	nativeRun, err := nativeService.CreateAndRunTaskScoped(context.Background(), CreateInput{Title: "Native synthetic contact intake", WorkflowJSON: workflow, SimulatorSourceJSON: `{"kind":"recorded-exchanges","exchanges":[{"method":"POST","host":"crm.synthetic.invalid","path":"/v1/contacts","status":201,"response":{"id":"contact-real-ish","status":"created"}}]}`}, WorkspaceSandboxTaskRequest{TaskID: "task-1", Name: "task-rehearsal"})
	if err != nil {
		t.Fatal(err)
	}
	if nativeRun.Rehearsal.State != StateNeedsReview || nativeRun.Lifecycle.N8NRuntimeStatus != "stopped" || nativeRun.Lifecycle.N8NExecution == nil || nativeRun.Lifecycle.RouteHitsStatus != "available" || nativeRun.Lifecycle.RouteHits.TotalHits != 1 || !strings.Contains(nativeRun.Rehearsal.HandoffMarkdown, "Native n8n runtime: `stopped`; execution `completed` (2 nodes, 2 run(s), 1 item(s))") {
		t.Fatalf("native owner receipts were not retained: %#v", nativeRun.Lifecycle)
	}
	wantOrder := []string{"sandbox-create", "n8n-start", "companion-start", "n8n-execute", "route-hits", "n8n-stop", "companion-stop", "sandbox-delete"}
	if strings.Join(eventOrder, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("native task ordering = %v, want %v", eventOrder, wantOrder)
	}
	deleted = false
	calls = nil
	eventOrder = nil
	companionHits.Store(0)
	noRouteService := NewServiceWithNativeN8N(newMemoryRepository(), SyntheticEngine{}, time.Now, client, &taskN8NRuntimeFake{events: &eventOrder})
	noRouteRun, err := noRouteService.CreateAndRunTaskScoped(context.Background(), CreateInput{Title: "Native workflow without companion traffic", WorkflowJSON: workflow, SimulatorSourceJSON: `{"kind":"recorded-exchanges","exchanges":[{"method":"POST","host":"crm.synthetic.invalid","path":"/v1/contacts","status":201,"response":{"id":"contact-real-ish","status":"created"}}]}`}, WorkspaceSandboxTaskRequest{TaskID: "task-1", Name: "task-rehearsal"})
	if err != nil {
		t.Fatal(err)
	}
	if noRouteRun.Rehearsal.State != StateFailed || noRouteRun.Lifecycle.RouteHitsStatus != "available" || noRouteRun.Lifecycle.RouteHits.TotalHits != 0 || !strings.Contains(noRouteRun.Rehearsal.FaultSummary, "without a matching verified request through its Workspace Sandbox companion route") {
		t.Fatalf("native HTTP workflow without route evidence must fail closed: run=%#v lifecycle=%#v", noRouteRun.Rehearsal, noRouteRun.Lifecycle)
	}
	deleted = false
	calls = nil
	eventOrder = nil
	startupFailure := NewServiceWithNativeN8N(newMemoryRepository(), SyntheticEngine{}, time.Now, client, taskN8NRuntimeStartFailure{events: &eventOrder})
	_, err = startupFailure.CreateAndRunTaskScoped(context.Background(), CreateInput{Title: "Runtime startup failure", WorkflowJSON: workflow, SimulatorSourceJSON: `{"kind":"recorded-exchanges","exchanges":[{"method":"POST","host":"crm.synthetic.invalid","path":"/v1/contacts","status":201,"response":{"id":"synthetic-contact","status":"created"}}]}`}, WorkspaceSandboxTaskRequest{TaskID: "task-1", Name: "task-rehearsal"})
	if err == nil {
		t.Fatal("n8n startup failure must fail the task rehearsal")
	}
	wantFailureOrder := []string{"sandbox-create", "n8n-start-failed", "sandbox-delete"}
	if strings.Join(eventOrder, ",") != strings.Join(wantFailureOrder, ",") || !deleted {
		t.Fatalf("failed native startup cleanup order = %v; deleted=%t, want %v", eventOrder, deleted, wantFailureOrder)
	}
	deleted = false
	calls = nil
	eventOrder = nil
	failingService := NewServiceWithWorkspaceSandbox(newMemoryRepository(), taskLifecycleFailingEngine{}, time.Now, client)
	failedRun, err := failingService.CreateAndRunTaskScoped(context.Background(), CreateInput{Title: "Synthetic contact intake failure", WorkflowJSON: workflow, SimulatorSourceJSON: `{"kind":"recorded-exchanges","exchanges":[{"method":"POST","host":"crm.synthetic.invalid","path":"/v1/contacts","status":201,"response":{"id":"contact-real-ish","status":"created"}}]}`}, WorkspaceSandboxTaskRequest{TaskID: "task-1", Name: "task-rehearsal"})
	if err != nil {
		t.Fatal(err)
	}
	if failedRun.Rehearsal.State != StateFailed || failedRun.Lifecycle.TeardownStatus != "succeeded" || !deleted {
		t.Fatalf("execution failure did not retain teardown evidence: %#v", failedRun)
	}
}

func TestNativeWorkflowRouteEvidenceMustMatchDeclaredHTTPRoute(t *testing.T) {
	workflow, _, _, err := normalizeWorkflow(`{"name":"Route evidence","nodes":[{"name":"Start","type":"n8n-nodes-base.manualTrigger","parameters":{}},{"name":"Create contact","type":"n8n-nodes-base.httpRequest","parameters":{"method":"POST","url":"https://crm.synthetic.invalid/v1/contacts"}}],"connections":{"Start":{"main":[[{"node":"Create contact"}]]}}}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		hit  HTTPMockRouteHit
		want bool
	}{
		{name: "exact declared route", hit: HTTPMockRouteHit{Host: "crm.synthetic.invalid", Method: "POST", Path: "/v1/contacts", Count: 1}, want: true},
		{name: "unrelated host", hit: HTTPMockRouteHit{Host: "gmail.synthetic.invalid", Method: "POST", Path: "/v1/contacts", Count: 1}},
		{name: "unrelated path", hit: HTTPMockRouteHit{Host: "crm.synthetic.invalid", Method: "POST", Path: "/v1/other", Count: 1}},
		{name: "no observed calls", hit: HTTPMockRouteHit{Host: "crm.synthetic.invalid", Method: "POST", Path: "/v1/contacts"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			receipt := HTTPMockRouteHitsReceipt{Routes: []HTTPMockRouteHit{test.hit}}
			if got := nativeWorkflowHasCompanionRouteHit(workflow, receipt); got != test.want {
				t.Fatalf("native route evidence match = %t, want %t", got, test.want)
			}
		})
	}
}

type taskN8NRuntimeFake struct {
	events      *[]string
	routeClient sandboxHTTPDoer
}

type taskN8NRuntimeStartFailure struct{ events *[]string }

func (f taskN8NRuntimeStartFailure) Start(context.Context, string) (N8NTaskRuntimeLease, error) {
	*f.events = append(*f.events, "n8n-start-failed")
	return N8NTaskRuntimeLease{}, errors.New("synthetic n8n startup failure")
}

func (taskN8NRuntimeStartFailure) ImportAndExecuteManual(context.Context, N8NTaskRuntimeLease, string) (N8NWorkflowExecutionEvidence, error) {
	return N8NWorkflowExecutionEvidence{}, errors.New("runtime must not execute after startup failure")
}

func (taskN8NRuntimeStartFailure) Stop(context.Context, N8NTaskRuntimeLease) error {
	return errors.New("runtime stop must not run without a lease")
}

func (f *taskN8NRuntimeFake) Start(_ context.Context, sandboxID string) (N8NTaskRuntimeLease, error) {
	*f.events = append(*f.events, "n8n-start")
	return N8NTaskRuntimeLease{SandboxID: sandboxID, RuntimeID: "n8n:" + sandboxID}, nil
}

func (f *taskN8NRuntimeFake) ImportAndExecuteManual(ctx context.Context, lease N8NTaskRuntimeLease, workflow string) (N8NWorkflowExecutionEvidence, error) {
	*f.events = append(*f.events, "n8n-execute")
	if lease.RuntimeID == "" || !strings.Contains(workflow, "manualTrigger") || !strings.Contains(workflow, "crm.synthetic.invalid:18080/v1/contacts") {
		return N8NWorkflowExecutionEvidence{}, errors.New("native test received invalid workflow/runtime")
	}
	if f.routeClient != nil {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://crm.synthetic.invalid:18080/v1/contacts", strings.NewReader(`{"name":"Synthetic contact"}`))
		if err != nil {
			return N8NWorkflowExecutionEvidence{}, err
		}
		request.Header.Set("Content-Type", "application/json")
		response, err := f.routeClient.Do(request)
		if err != nil {
			return N8NWorkflowExecutionEvidence{}, err
		}
		defer response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return N8NWorkflowExecutionEvidence{}, errors.New("synthetic companion route returned a non-success status")
		}
	}
	return N8NWorkflowExecutionEvidence{OperationID: "00000000-0000-4000-8000-000000000001", Status: "completed", NodeCount: 2, RunCount: 2, ItemCount: 1}, nil
}

func (f *taskN8NRuntimeFake) Stop(_ context.Context, _ N8NTaskRuntimeLease) error {
	*f.events = append(*f.events, "n8n-stop")
	return nil
}

type taskLifecycleFailingEngine struct{}

func (taskLifecycleFailingEngine) Execute(context.Context, Workflow, string, Simulator, bool, time.Time) ([]TraceEvent, error) {
	return nil, errors.New("synthetic execution failure")
}

func TestTaskScopedRunRequiresLogicalTaskIDAndRejectsTraversal(t *testing.T) {
	svc := NewService(newMemoryRepository(), SyntheticEngine{}, time.Now)
	_, err := svc.CreateAndRunTaskScoped(context.Background(), CreateInput{Title: "no sandbox", WorkflowJSON: `{}`}, WorkspaceSandboxTaskRequest{})
	if err == nil || !strings.Contains(err.Error(), "configured Workspace Sandbox client") {
		t.Fatalf("unconfigured opt-in must fail closed, got %v", err)
	}
	if err := validateWorkspaceSandboxTaskScope(WorkspaceSandboxTaskRequest{TaskID: "../escape"}); err == nil {
		t.Fatal("caller-selected traversal task ID must fail closed")
	}
}

func lifecycleTestProfile(t *testing.T) (HTTPMockProfile, string) {
	t.Helper()
	simulator, err := NewDescriptionSimulator(`{"kind":"recorded-exchanges","exchanges":[{"method":"POST","host":"crm.synthetic.invalid","path":"/v1/contacts","status":201,"response":{"id":"synthetic-contact-01","status":"created"}}]}`, false)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := simulator.ExportHTTPMockProfile()
	if err != nil {
		t.Fatal(err)
	}
	digest, err := httpMockProfileDigest(profile)
	if err != nil {
		t.Fatal(err)
	}
	return profile, digest
}
