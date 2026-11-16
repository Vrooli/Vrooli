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

func TestNativeCreateAndRunSelectsTaskBoundVirtualAPIAndExportsHandoff(t *testing.T) {
	root := t.TempDir()
	scope := filepath.Join(root, "virtual-task")
	if err := os.Mkdir(scope, 0o700); err != nil {
		t.Fatal(err)
	}
	const sandboxID = "51cabaf4-4a0a-4a6f-99d0-9c0ce850d31e"
	owner := "trusted-service:virtual-task"
	profile := VirtualAPIProfile{Hosts: []string{"api.notion.com"}, Routes: []VirtualAPIRoute{{Host: "api.notion.com", Method: http.MethodPost, Path: "/v1/pages", Status: http.StatusOK, ContentType: "application/json", Body: `{"id":"synthetic-page","object":"page"}`}}}
	var digest string
	var routeHits atomic.Uint64
	var order []string
	companion := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/pages" || r.Host != "api.notion.com" {
			t.Errorf("unexpected HTTPS alias request: %s %s host=%q", r.Method, r.URL, r.Host)
		}
		order = append(order, "virtual-route")
		routeHits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, profile.Routes[0].Body)
	}))
	defer companion.Close()
	base, err := url.Parse(companion.URL)
	if err != nil {
		t.Fatal(err)
	}
	virtualHTTP := &http.Client{Transport: aliasTestTransport{base: base, inner: http.DefaultTransport}}
	deleted := false
	var declaredKind string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/task-scopes":
			order = append(order, "task-create")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": sandboxID, "name": "virtual-task", "projectRoot": root, "scopePath": scope, "owner": owner, "ownerType": "task", "idempotencyKey": "task-scope:virtual-task"})
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/companion-services"):
			var body struct {
				Services []struct {
					Kind       string            `json:"kind"`
					VirtualAPI VirtualAPIProfile `json:"virtualAPI"`
				} `json:"services"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Services) != 1 {
				t.Errorf("invalid virtual API declaration: err=%v", err)
			} else {
				declaredKind = body.Services[0].Kind
				digest, err = virtualAPIProfileDigest(sandboxID, body.Services[0].VirtualAPI)
				if err != nil {
					t.Errorf("declared profile digest: %v", err)
				}
			}
			order = append(order, "virtual-declare")
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/workflow-apis/start"):
			order = append(order, "virtual-start")
			_ = json.NewEncoder(w).Encode(WorkspaceSandboxProfileReceipt{Spec: struct {
				Name    string `json:"name"`
				Kind    string `json:"kind"`
				Version string `json:"version"`
			}{Name: taskSandboxProfileName, Kind: "virtual-api", Version: taskSandboxProfileVersion}, Status: "running", Addresses: map[string]string{"api.notion.com": "https://api.notion.com:443"}, ProfileSHA256: digest})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sandboxes/"+sandboxID && !deleted:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": sandboxID, "name": "virtual-task", "projectRoot": root, "scopePath": scope, "owner": owner, "ownerType": "task", "idempotencyKey": "task-scope:virtual-task", "status": "active"})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/workflow-apis/route-hits"):
			order = append(order, "route-hits")
			_ = json.NewEncoder(w).Encode(HTTPMockRouteHitsReceipt{SandboxID: sandboxID, CompanionRuntimeID: "n8n-virtual-api:" + sandboxID, Status: "available", TotalHits: routeHits.Load(), Routes: []HTTPMockRouteHit{{Host: "api.notion.com", Method: http.MethodPost, Path: "/v1/pages", QuerySHA256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", Count: routeHits.Load()}}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/workflow-apis/stop"):
			order = append(order, "virtual-stop")
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sandboxes/"+sandboxID:
			order = append(order, "task-delete")
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sandboxes/"+sandboxID && deleted:
			http.NotFound(w, r)
		default:
			http.Error(w, "unexpected Sandbox API request", http.StatusNotFound)
		}
	}))
	defer api.Close()
	client, err := NewWorkspaceSandboxProfileClient(api.URL, api.Client())
	if err != nil {
		t.Fatal(err)
	}
	runtime := &virtualAPIServiceRuntime{events: &order, client: virtualHTTP}
	repo := newMemoryRepository()
	service := NewServiceWithNativeN8N(repo, SyntheticEngine{}, time.Now, client, runtime)
	run, err := service.CreateAndRunTaskScoped(context.Background(), CreateInput{
		Title:               "Notion page sync",
		WorkflowJSON:        `{"name":"Notion page sync","nodes":[{"name":"Start","type":"n8n-nodes-base.manualTrigger","parameters":{}},{"name":"Create page","type":"n8n-nodes-base.httpRequest","parameters":{"method":"POST","url":"https://api.notion.com/v1/pages?api_key=synthetic-but-sensitive"}}],"connections":{"Start":{"main":[[{"node":"Create page"}]]}}}`,
		SimulatorSourceJSON: `{"kind":"openapi","document":{"openapi":"3.0.3","servers":[{"url":"https://api.notion.com/v1"}],"paths":{"/pages":{"post":{"x-n8n-node-type":"n8n-nodes-base.httpRequest","x-n8n-resource":"request","x-n8n-operation":"request","responses":{"200":{"content":{"application/json":{"example":{"id":"source-page","object":"page"}}}}}}}}}}`,
	}, WorkspaceSandboxTaskRequest{TaskID: "virtual-task", Name: "virtual-task"})
	if err != nil {
		t.Fatal(err)
	}
	if declaredKind != "virtual-api" || run.Lifecycle.ProfileSHA256 != digest || run.Lifecycle.OperationCount != 1 || !run.Lifecycle.SyntheticTargets || routeHits.Load() != 1 || run.Lifecycle.RouteHits == nil || run.Lifecycle.RouteHits.TotalHits != 1 || run.Lifecycle.TeardownStatus != "succeeded" {
		t.Fatalf("VirtualAPI native path lacks task-bound route/execution/cleanup evidence: kind=%q run=%#v", declaredKind, run)
	}
	if run.Rehearsal.State != StateNeedsReview || !strings.Contains(run.Rehearsal.HandoffMarkdown, "Companion route-hit receipt: `available` (1 hit(s) across 1 route(s))") || !strings.Contains(run.Rehearsal.HandoffMarkdown, "Native n8n runtime: `stopped`; execution `completed`") {
		t.Fatalf("synthetic owner handoff omitted VirtualAPI or native evidence: %#v", run.Rehearsal)
	}
	want := []string{"task-create", "virtual-declare", "virtual-start", "n8n-attach", "native-execute", "virtual-route", "route-hits", "n8n-stop", "virtual-stop", "task-delete"}
	if strings.Join(order, ",") != strings.Join(want, ",") || !deleted {
		t.Fatalf("VirtualAPI task lifecycle order=%v, deleted=%v; want %v", order, deleted, want)
	}
}

func TestNativeUnboundServiceOperationDeclinesBeforeSandboxOrRuntimeStart(t *testing.T) {
	var apiRequests atomic.Uint64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiRequests.Add(1)
		http.Error(w, "unexpected Sandbox call for unbound native service", http.StatusBadRequest)
	}))
	defer server.Close()
	client, err := NewWorkspaceSandboxProfileClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	var events []string
	runtime := &virtualAPIServiceRuntime{events: &events, client: server.Client()}
	service := NewServiceWithNativeN8N(newMemoryRepository(), SyntheticEngine{}, time.Now, client, runtime)
	workflow := `{"name":"Unbound scheduled Sheets append","nodes":[{"id":"start","name":"Schedule","type":"n8n-nodes-base.scheduleTrigger","parameters":{"rule":{"interval":[{"field":"days","daysInterval":1,"triggerAtHour":8,"triggerAtMinute":0}]}}},{"id":"append","name":"Append row","type":"n8n-nodes-base.googleSheets","typeVersion":4.5,"parameters":{"resource":"sheet","operation":"append","documentId":{"__rl":true,"mode":"id","value":"synthetic-sheet"},"sheetName":{"__rl":true,"mode":"name","value":"Leads"}}}],"connections":{"Schedule":{"main":[[{"node":"Append row"}]]}}}`
	normalized, _, _, normalizeErr := normalizeWorkflow(workflow)
	if normalizeErr != nil || !nativeScheduledWorkflowSupported(normalized) {
		t.Fatalf("scheduled API-bound graph did not pass broad admission: normalize=%v workflow=%#v", normalizeErr, normalized)
	}
	_, err = service.CreateAndRunTaskScoped(context.Background(), CreateInput{Title: "Unbound Sheets operation must decline", WorkflowJSON: workflow}, WorkspaceSandboxTaskRequest{TaskID: "unbound-sheet", Name: "unbound-sheet"})
	var invalid ErrInvalid
	if !errors.As(err, &invalid) || invalid.Field != "workflow.native_service_routes" {
		t.Fatalf("unbound native operation did not return its typed preflight decline: %v", err)
	}
	if apiRequests.Load() != 0 || len(events) != 0 {
		t.Fatalf("unbound native integration started a Sandbox task or runtime: requests=%d events=%v", apiRequests.Load(), events)
	}
}

type virtualAPIServiceRuntime struct {
	events *[]string
	client *http.Client
}

func (f *virtualAPIServiceRuntime) Start(_ context.Context, sandboxID string) (N8NTaskRuntimeLease, error) {
	*f.events = append(*f.events, "n8n-attach")
	return N8NTaskRuntimeLease{SandboxID: sandboxID, RuntimeID: "n8n:" + sandboxID}, nil
}

func (f *virtualAPIServiceRuntime) ImportAndExecuteManual(ctx context.Context, lease N8NTaskRuntimeLease, workflow string) (N8NWorkflowExecutionEvidence, error) {
	*f.events = append(*f.events, "native-execute")
	if lease.RuntimeID == "" || !strings.Contains(workflow, "https://api.notion.com/v1/pages") || strings.Contains(workflow, "api_key") || strings.Contains(workflow, "synthetic-but-sensitive") {
		return N8NWorkflowExecutionEvidence{}, errors.New("native workflow did not retain the exact task VirtualAPI URL")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.notion.com/v1/pages", strings.NewReader(`{"title":"synthetic"}`))
	if err != nil {
		return N8NWorkflowExecutionEvidence{}, err
	}
	response, err := f.client.Do(request)
	if err != nil {
		return N8NWorkflowExecutionEvidence{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return N8NWorkflowExecutionEvidence{}, errors.New("VirtualAPI fixture returned unexpected status")
	}
	return N8NWorkflowExecutionEvidence{OperationID: "00000000-0000-4000-8000-000000000002", Status: "completed", NodeCount: 2, RunCount: 2, ItemCount: 1}, nil
}

func (f *virtualAPIServiceRuntime) ImportAndExecuteScheduled(context.Context, N8NTaskRuntimeLease, string) (N8NWorkflowExecutionEvidence, error) {
	*f.events = append(*f.events, "native-scheduled-execute")
	return N8NWorkflowExecutionEvidence{OperationID: "00000000-0000-4000-8000-000000000004", TriggerKind: string(TriggerScheduled), Status: "completed", NodeCount: 2, RunCount: 1, ItemCount: 1}, nil
}

func (f *virtualAPIServiceRuntime) Stop(context.Context, N8NTaskRuntimeLease) error {
	*f.events = append(*f.events, "n8n-stop")
	return nil
}
