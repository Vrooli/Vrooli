package rehearsals

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type aliasTestTransport struct {
	base  *url.URL
	inner http.RoundTripper
}

func (t aliasTestTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	copyRequest := request.Clone(request.Context())
	copyURL := *request.URL
	copyURL.Scheme, copyURL.Host = t.base.Scheme, t.base.Host
	copyRequest.URL = &copyURL
	return t.inner.RoundTrip(copyRequest)
}

func TestSandboxEngineAdapterRoutesNativeOperationAndRecordsRouteHit(t *testing.T) {
	var hits atomic.Uint64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/messages" || r.Host != "gmail.synthetic.invalid:18080" {
			t.Errorf("unexpected forwarded request: %s %s host=%q", r.Method, r.URL, r.Host)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"subject":"Synthetic"}` {
			t.Errorf("unexpected request body: %s", body)
		}
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"synthetic-message","status":"sent"}`)
	}))
	defer server.Close()
	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: aliasTestTransport{base: base, inner: http.DefaultTransport}}
	selector := HTTPMockProfileSelector{NodeType: "n8n-nodes-base.gmail", Resource: "message", Operation: "send"}
	receipt := WorkspaceSandboxProfileReceipt{Status: "running", OperationAddresses: []HTTPMockOperationAddress{{Selector: selector, Method: http.MethodPost, URL: "http://gmail.synthetic.invalid:18080/v1/messages"}}}
	fallback, err := NewDescriptionSimulator(readFixture(t, "native_service_catalog.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewSandboxEngineAdapter(fallback, receipt, client)
	if err != nil {
		t.Fatal(err)
	}
	response, err := adapter.Invoke(context.Background(), Step{NodeType: selector.NodeType, Resource: selector.Resource, Action: selector.Operation, Input: map[string]any{"subject": "Synthetic"}}, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(response, `"status":"sent"`) || hits.Load() != 1 {
		t.Fatalf("response=%s route hits=%d", response, hits.Load())
	}
	// Read the same receipt the service persists after execution, backed by a
	// local fake Sandbox API whose hit count came from the companion handler.
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/companion-services/workflow-apis/route-hits") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(HTTPMockRouteHitsReceipt{SandboxID: "7d5eb3fd-9847-4cf6-ae0f-f4c063443ce9", CompanionRuntimeID: "http-mock-test", Status: "available", TotalHits: hits.Load(), Routes: []HTTPMockRouteHit{{Host: "gmail.synthetic.invalid", Method: "POST", Path: "/v1/messages", Count: hits.Load()}}})
	}))
	defer api.Close()
	sandbox, err := NewWorkspaceSandboxProfileClient(api.URL, api.Client())
	if err != nil {
		t.Fatal(err)
	}
	routeHits, err := sandbox.ReadRouteHits(context.Background(), "7d5eb3fd-9847-4cf6-ae0f-f4c063443ce9", "workflow-apis")
	if err != nil {
		t.Fatal(err)
	}
	if routeHits.TotalHits != 1 || routeHits.Routes[0].Count != 1 {
		t.Fatalf("expected route-hit receipt for companion traffic, got %#v", routeHits)
	}
}

func TestSandboxEngineAdapterFailsClosedForUnmatchedNativeOperation(t *testing.T) {
	fallback, err := NewDescriptionSimulator(readFixture(t, "native_service_catalog.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	receipt := WorkspaceSandboxProfileReceipt{Status: "running", OperationAddresses: []HTTPMockOperationAddress{{Selector: HTTPMockProfileSelector{NodeType: "n8n-nodes-base.gmail", Resource: "message", Operation: "send"}, Method: http.MethodPost, URL: "http://gmail.synthetic.invalid:18080/v1/messages"}}}
	adapter, err := NewSandboxEngineAdapter(fallback, receipt, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = adapter.Invoke(context.Background(), Step{NodeType: "n8n-nodes-base.hubspot", Resource: "contact", Action: "upsert"}, 1, false)
	if err == nil || !strings.Contains(err.Error(), "no attached Sandbox operation") {
		t.Fatalf("expected unmatched native operation to fail closed, got %v", err)
	}
}

func TestSandboxEngineAdapterRoutesGenericHTTPRequestThroughCompanion(t *testing.T) {
	var hits atomic.Uint64
	companion := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Host != "crm.synthetic.invalid:18080" || r.URL.Path != "/v1/contacts" {
			t.Errorf("unexpected generic companion request: %s %s host=%q", r.Method, r.URL, r.Host)
		}
		hits.Add(1)
		_, _ = io.WriteString(w, `{"status":"upserted"}`)
	}))
	defer companion.Close()
	base, err := url.Parse(companion.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: aliasTestTransport{base: base, inner: http.DefaultTransport}}
	fallback, err := NewDescriptionSimulator(readFixture(t, "native_service_catalog.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	receipt := WorkspaceSandboxProfileReceipt{Status: "running", OperationAddresses: []HTTPMockOperationAddress{{Selector: HTTPMockProfileSelector{NodeType: "n8n-nodes-base.httpRequest", Resource: "request", Operation: "request"}, Method: http.MethodPost, URL: "http://crm.synthetic.invalid:18080/v1/contacts"}}}
	adapter, err := NewSandboxEngineAdapter(fallback, receipt, client)
	if err != nil {
		t.Fatal(err)
	}
	response, err := adapter.Invoke(context.Background(), Step{Method: http.MethodPost, Host: "crm.synthetic.invalid", URL: "/v1/contacts", Input: map[string]any{"name": "Synthetic"}}, 1, false)
	if err != nil || !strings.Contains(response, "upserted") || hits.Load() != 1 {
		t.Fatalf("response=%q hits=%d err=%v", response, hits.Load(), err)
	}
}

func TestRewriteWorkflowHTTPRequestsToSandboxAliases(t *testing.T) {
	input := `{"name":"CRM intake","nodes":[{"name":"Start","type":"n8n-nodes-base.manualTrigger","parameters":{}},{"name":"Request","type":"n8n-nodes-base.httpRequest","parameters":{"method":"POST","url":"https://crm.synthetic.invalid/v1/contacts"}}],"connections":{}}`
	operations := []HTTPMockOperationAddress{{Selector: HTTPMockProfileSelector{NodeType: "n8n-nodes-base.httpRequest", Resource: "request", Operation: "request"}, Method: http.MethodPost, URL: "http://crm.synthetic.invalid:18080/v1/contacts"}}
	rewritten, err := rewriteWorkflowHTTPRequestsForSandbox(input, operations)
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Nodes []struct {
			Parameters map[string]any `json:"parameters"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(rewritten), &workflow); err != nil {
		t.Fatal(err)
	}
	if got := workflow.Nodes[1].Parameters["url"]; got != operations[0].URL {
		t.Fatalf("rewritten URL = %v, want %q", got, operations[0].URL)
	}
	if _, err := rewriteWorkflowHTTPRequestsForSandbox(input, nil); err == nil {
		t.Fatal("expected unmatched HTTP Request to fail closed")
	}
}

func TestTaskScopedEngineRoutesNativeWorkflowThroughCompanionAndPersistsHitReceipt(t *testing.T) {
	var hits atomic.Uint64
	companion := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Host != "gmail.synthetic.invalid:18080" || r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected companion request: %s %s host=%q", r.Method, r.URL, r.Host)
		}
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"synthetic-message","status":"sent"}`)
	}))
	defer companion.Close()
	companionURL, err := url.Parse(companion.URL)
	if err != nil {
		t.Fatal(err)
	}
	doer := &http.Client{Transport: aliasTestTransport{base: companionURL, inner: http.DefaultTransport}}

	const sandboxID = "4e4c02da-9ec0-4c02-949e-22ee48a06f33"
	const scopeID = "integration-task"
	const taskName = "workflow-run"
	root, taskDir := "/server/tasks", "/server/tasks/integration-task"
	owner := "client-rehearsal:integration-task"
	var profile HTTPMockProfile
	deleted := false
	var eventOrder []string
	controlPlane := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/task-scopes":
			eventOrder = append(eventOrder, "scope-create")
			_ = json.NewEncoder(w).Encode(WorkspaceSandboxTaskRecord{ID: sandboxID, Name: taskName, ProjectRoot: root, ScopePath: taskDir, Owner: owner, OwnerType: "task", IdempotencyKey: "task-scope:" + scopeID})
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/companion-services"):
			var declaration struct {
				Services []struct {
					Name        string          `json:"name"`
					Kind        string          `json:"kind"`
					Version     string          `json:"version"`
					HTTPProfile HTTPMockProfile `json:"httpProfile"`
				} `json:"services"`
			}
			if err := json.NewDecoder(r.Body).Decode(&declaration); err != nil || len(declaration.Services) != 1 || declaration.Services[0].Name != taskSandboxProfileName {
				t.Errorf("invalid companion declaration: %+v err=%v", declaration, err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			profile = declaration.Services[0].HTTPProfile
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/workflow-apis/start"):
			eventOrder = append(eventOrder, "companion-start")
			digest, digestErr := httpMockProfileDigest(profile)
			if digestErr != nil {
				t.Errorf("profile digest: %v", digestErr)
			}
			receipt := WorkspaceSandboxProfileReceipt{Status: "running", ProfileSHA256: digest, Addresses: map[string]string{}}
			receipt.Spec.Name, receipt.Spec.Kind, receipt.Spec.Version = taskSandboxProfileName, "http-mock", taskSandboxProfileVersion
			for _, operation := range profile.Operations {
				route := operation.Route
				receipt.Addresses[route.Host] = "http://" + route.Host + ":18080"
				address := HTTPMockOperationAddress{Selector: operation.Selector, Method: route.Method, URL: "http://" + route.Host + ":18080" + route.Path}
				if route.Query != "" {
					address.URL += "?" + route.Query
				}
				receipt.OperationAddresses = append(receipt.OperationAddresses, address)
			}
			_ = json.NewEncoder(w).Encode(receipt)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sandboxes/"+sandboxID && !deleted:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": sandboxID, "name": taskName, "projectRoot": root, "scopePath": taskDir, "owner": owner, "ownerType": "task", "idempotencyKey": "task-scope:" + scopeID, "status": "active"})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/workflow-apis/route-hits"):
			eventOrder = append(eventOrder, "route-hits")
			_ = json.NewEncoder(w).Encode(HTTPMockRouteHitsReceipt{SandboxID: sandboxID, CompanionRuntimeID: "sandbox.http.mock", Status: "available", TotalHits: hits.Load(), Routes: []HTTPMockRouteHit{{Host: "gmail.synthetic.invalid", Method: http.MethodPost, Path: "/v1/messages", Count: hits.Load()}}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/workflow-apis/stop"):
			eventOrder = append(eventOrder, "companion-stop")
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sandboxes/"+sandboxID:
			eventOrder = append(eventOrder, "scope-delete")
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sandboxes/"+sandboxID && deleted:
			http.NotFound(w, r)
		default:
			http.Error(w, "unexpected Workspace Sandbox request", http.StatusNotFound)
		}
	}))
	defer controlPlane.Close()
	sandbox, err := NewWorkspaceSandboxProfileClient(controlPlane.URL, controlPlane.Client())
	if err != nil {
		t.Fatal(err)
	}
	service := &service{repo: newMemoryRepository(), engine: SyntheticEngine{}, now: time.Now, sandbox: sandbox, sandboxHTTPDoer: doer}
	workflow := `{"name":"Synthetic Gmail delivery","nodes":[{"id":"start","name":"Start","type":"n8n-nodes-base.manualTrigger","parameters":{}},{"id":"send","name":"Send message","type":"n8n-nodes-base.gmail","parameters":{"resource":"message","operation":"send"}}],"connections":{"Start":{"main":[[{"node":"Send message"}]]}}}`
	run, err := service.CreateAndRunTaskScoped(context.Background(), CreateInput{Title: "Synthetic Gmail handoff", WorkflowJSON: workflow, SimulatorSourceJSON: readFixture(t, "native_service_catalog.json")}, WorkspaceSandboxTaskRequest{TaskID: scopeID, Name: taskName})
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 || run.Lifecycle.RouteHits == nil || run.Lifecycle.RouteHits.TotalHits != 1 || len(run.Lifecycle.RouteHits.Routes) != 1 || run.Lifecycle.TeardownStatus != "succeeded" {
		t.Fatalf("workflow companion request or receipt missing: hits=%d lifecycle=%+v", hits.Load(), run.Lifecycle)
	}
	if !strings.Contains(run.Rehearsal.HandoffMarkdown, "Companion route-hit receipt: `available` (1 hit(s) across 1 route(s))") {
		t.Fatalf("handoff did not expose route-hit evidence: %s", run.Rehearsal.HandoffMarkdown)
	}
	if got, want := strings.Join(eventOrder, ","), "scope-create,companion-start,route-hits,companion-stop,scope-delete"; got != want {
		t.Fatalf("lifecycle order %q, want %q", got, want)
	}
}
