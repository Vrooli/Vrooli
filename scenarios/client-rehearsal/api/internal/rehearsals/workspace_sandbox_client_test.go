package rehearsals

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkspaceSandboxProfileClientDeclaresStartsAndStopsExactProfile(t *testing.T) {
	profile, err := NewDescriptionSimulator(readFixture(t, "native_service_catalog.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	exported, err := profile.ExportHTTPMockProfile()
	if err != nil {
		t.Fatal(err)
	}
	const sandboxID = "7d5eb3fd-9847-4cf6-ae0f-f4c063443ce9"
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/sandboxes/"+sandboxID+"/companion-services":
			var request struct {
				Services []struct {
					Name        string          `json:"name"`
					Kind        string          `json:"kind"`
					Version     string          `json:"version"`
					HTTPProfile HTTPMockProfile `json:"httpProfile"`
				} `json:"services"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode declaration: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if len(request.Services) != 1 || request.Services[0].Name != "workflow-apis" || request.Services[0].Kind != "http-mock" || len(request.Services[0].HTTPProfile.Operations) != 4 {
				t.Errorf("unexpected Sandbox declaration: %#v", request)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"services":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sandboxes/"+sandboxID+"/companion-services/workflow-apis/start":
			var receipt struct {
				Spec struct {
					Name    string `json:"name"`
					Kind    string `json:"kind"`
					Version string `json:"version"`
				} `json:"spec"`
				Status             string                     `json:"status"`
				Addresses          map[string]string          `json:"addresses"`
				OperationAddresses []HTTPMockOperationAddress `json:"operationAddresses"`
				ProfileSHA256      string                     `json:"profileSha256"`
			}
			receipt.Spec.Name = "workflow-apis"
			receipt.Spec.Kind = "http-mock"
			receipt.Spec.Version = "1.0.0"
			receipt.Status = "running"
			receipt.ProfileSHA256, _ = httpMockProfileDigest(exported)
			receipt.Addresses = map[string]string{}
			for _, operation := range exported.Operations {
				receipt.OperationAddresses = append(receipt.OperationAddresses, HTTPMockOperationAddress{Selector: operation.Selector, Method: operation.Route.Method, URL: "http://" + operation.Route.Host + ":18080" + operation.Route.Path})
				receipt.Addresses[operation.Route.Host] = "http://" + operation.Route.Host + ":18080"
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(receipt)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sandboxes/"+sandboxID+"/companion-services/workflow-apis/stop":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Errorf("unexpected Sandbox request: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewWorkspaceSandboxProfileClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := client.DeclareAndStartProfile(context.Background(), sandboxID, "workflow-apis", "1.0.0", exported)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.SandboxID != sandboxID || receipt.Spec.Name != "workflow-apis" || receipt.Status != "running" || len(receipt.OperationAddresses) != 4 {
		t.Fatalf("unexpected validated receipt: %#v", receipt)
	}
	if err := client.StopProfile(context.Background(), sandboxID, "workflow-apis"); err != nil {
		t.Fatal(err)
	}
	if len(methods) != 3 || methods[0] != "PUT /api/v1/sandboxes/"+sandboxID+"/companion-services" || methods[1] != "POST /api/v1/sandboxes/"+sandboxID+"/companion-services/workflow-apis/start" || methods[2] != "POST /api/v1/sandboxes/"+sandboxID+"/companion-services/workflow-apis/stop" {
		t.Fatalf("unexpected lifecycle calls: %v", methods)
	}
}

func TestWorkspaceSandboxProfileClientStartsRecordedExchangeSource(t *testing.T) {
	simulator, err := NewDescriptionSimulator(`{"kind":"recorded-exchanges","exchanges":[{"method":"POST","host":"crm.synthetic.invalid","path":"/v1/contacts","status":201,"response":{"id":"private-contact-1","email":"person@private.example","status":"created"}}]}`, false)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := simulator.ExportHTTPMockProfile()
	if err != nil {
		t.Fatal(err)
	}
	const sandboxID = "7d5eb3fd-9847-4cf6-ae0f-f4c063443ce9"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/sandboxes/"+sandboxID+"/companion-services":
			var request struct {
				Services []struct {
					HTTPProfile HTTPMockProfile `json:"httpProfile"`
				} `json:"services"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || len(request.Services) != 1 || len(request.Services[0].HTTPProfile.Operations) != 1 {
				t.Errorf("unexpected recorded-exchange declaration: %#v err=%v", request, err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			route := request.Services[0].HTTPProfile.Operations[0].Route
			if route.Host != "crm.synthetic.invalid" || route.Path != "/v1/contacts" || route.Status != http.StatusCreated || strings.Contains(route.Body, "private-contact-1") || strings.Contains(route.Body, "person@private.example") {
				t.Errorf("recorded exchange was not reduced to an exact synthetic route: %#v", route)
			}
			_, _ = w.Write([]byte(`{"services":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sandboxes/"+sandboxID+"/companion-services/workflow-apis/start":
			var receipt WorkspaceSandboxProfileReceipt
			receipt.Spec.Name, receipt.Spec.Kind, receipt.Spec.Version = "workflow-apis", "http-mock", "1.0.0"
			receipt.Status = "running"
			receipt.ProfileSHA256, _ = httpMockProfileDigest(profile)
			receipt.Addresses = map[string]string{"crm.synthetic.invalid": "http://crm.synthetic.invalid:18080"}
			for _, operation := range profile.Operations {
				receipt.OperationAddresses = append(receipt.OperationAddresses, HTTPMockOperationAddress{Selector: operation.Selector, Method: operation.Route.Method, URL: "http://" + operation.Route.Host + ":18080" + operation.Route.Path})
			}
			_ = json.NewEncoder(w).Encode(receipt)
		default:
			t.Errorf("unexpected Sandbox request: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := NewWorkspaceSandboxProfileClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := client.DeclareAndStartProfile(context.Background(), sandboxID, "workflow-apis", "1.0.0", profile)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "running" || len(receipt.OperationAddresses) != 1 {
		t.Fatalf("recorded-exchange Sandbox receipt is incomplete: %#v", receipt)
	}
}

func TestWorkspaceSandboxProfileReceiptPreservesDisjointInputSelectorsAndQueries(t *testing.T) {
	profile := HTTPMockProfile{SchemaVersion: 1, Operations: []HTTPMockProfileOperation{
		{Selector: HTTPMockProfileSelector{NodeType: "n8n-nodes-base.googleSheets", Resource: "sheet", Operation: "read", InputMatch: map[string]any{"sheetName": "products"}}, Route: HTTPMockRoute{Host: "sheets.synthetic.invalid", Method: http.MethodGet, Path: "/v1/spreadsheets", Query: "sheetName=products", Status: http.StatusOK, ContentType: "application/json", Body: `[]`}},
		{Selector: HTTPMockProfileSelector{NodeType: "n8n-nodes-base.googleSheets", Resource: "sheet", Operation: "read", InputMatch: map[string]any{"sheetName": "recipients"}}, Route: HTTPMockRoute{Host: "sheets.synthetic.invalid", Method: http.MethodGet, Path: "/v1/spreadsheets", Query: "sheetName=recipients", Status: http.StatusOK, ContentType: "application/json", Body: `[]`}},
	}}
	if err := validateExportedHTTPMockProfile(profile); err != nil {
		t.Fatalf("disjoint input-conditioned native operations must export: %v", err)
	}
	digest, err := httpMockProfileDigest(profile)
	if err != nil {
		t.Fatal(err)
	}
	receipt := WorkspaceSandboxProfileReceipt{Status: "running", ProfileSHA256: digest, Addresses: map[string]string{"sheets.synthetic.invalid": "http://sheets.synthetic.invalid:18080"}}
	receipt.Spec.Name, receipt.Spec.Kind, receipt.Spec.Version = "workflow-apis", "http-mock", "1.0.0"
	for _, operation := range profile.Operations {
		receipt.OperationAddresses = append(receipt.OperationAddresses, HTTPMockOperationAddress{Selector: operation.Selector, Method: operation.Route.Method, URL: "http://" + operation.Route.Host + ":18080" + operation.Route.Path + "?" + operation.Route.Query})
	}
	if err := validateProfileReceipt("workflow-apis", "1.0.0", digest, profile, receipt); err != nil {
		t.Fatalf("exact selector/address receipt was rejected: %v", err)
	}
	badReceipt := receipt
	badReceipt.OperationAddresses = append([]HTTPMockOperationAddress(nil), receipt.OperationAddresses...)
	badReceipt.OperationAddresses[0].URL = "http://sheets.synthetic.invalid:18080/v1/spreadsheets?sheetName=wrong"
	if err := validateProfileReceipt("workflow-apis", "1.0.0", digest, profile, badReceipt); err == nil {
		t.Fatal("mismatched selector query in Sandbox receipt must fail closed")
	}
}

func TestWorkspaceSandboxProfileRejectsUnsafeSelectorAndQueryValues(t *testing.T) {
	profile := HTTPMockProfile{SchemaVersion: 1, Operations: []HTTPMockProfileOperation{{
		Selector: HTTPMockProfileSelector{NodeType: "n8n-nodes-base.googleSheets", Resource: "sheet", Operation: "read", InputMatch: map[string]any{"sheetName": "Bearer abcdefghijklmno"}},
		Route:    HTTPMockRoute{Host: "sheets.synthetic.invalid", Method: http.MethodGet, Path: "/v1/spreadsheets", Query: "sheetName=Bearer+abcdefghijklmno", Status: http.StatusOK, ContentType: "application/json", Body: `[]`},
	}}}
	if err := validateExportedHTTPMockProfile(profile); err == nil {
		t.Fatal("credential-shaped selector must be rejected before profile declaration")
	}
	profile.Operations[0].Selector.InputMatch = nil
	profile.Operations[0].Route.Query = "sheetName=products&sheetName=other"
	if err := validateExportedHTTPMockProfile(profile); err == nil {
		t.Fatal("multi-valued query must be rejected before profile declaration")
	}
}

func TestWorkspaceSandboxProfileClientRejectsRemoteOriginAndRedirect(t *testing.T) {
	if _, err := NewWorkspaceSandboxProfileClient("https://sandbox.example.com", nil); err == nil {
		t.Fatal("remote Sandbox origin must be rejected")
	}
	redirects := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirects++
		http.Redirect(w, r, "http://example.com/", http.StatusFound)
	}))
	defer server.Close()
	client, err := NewWorkspaceSandboxProfileClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.DeclareAndStartProfile(context.Background(), "7d5eb3fd-9847-4cf6-ae0f-f4c063443ce9", "workflow-apis", "1.0.0", HTTPMockProfile{SchemaVersion: 1, Operations: []HTTPMockProfileOperation{{Selector: HTTPMockProfileSelector{NodeType: "n8n-nodes-base.gmail", Resource: "message", Operation: "send"}, Route: HTTPMockRoute{Host: "gmail.synthetic.invalid", Method: "POST", Path: "/v1/messages", Status: 202, ContentType: "application/json", Body: `{}`}}}})
	if err == nil || redirects != 1 {
		t.Fatalf("expected local redirect response to fail without following it, err=%v requests=%d", err, redirects)
	}
}

func TestWorkspaceSandboxProfileClientRejectsReceiptAddressDrift(t *testing.T) {
	profile := HTTPMockProfile{SchemaVersion: 1, Operations: []HTTPMockProfileOperation{{
		Selector: HTTPMockProfileSelector{NodeType: "n8n-nodes-base.gmail", Resource: "message", Operation: "send"},
		Route:    HTTPMockRoute{Host: "gmail.synthetic.invalid", Method: "POST", Path: "/v1/messages", Status: 202, ContentType: "application/json", Body: `{}`},
	}}}
	var receipt WorkspaceSandboxProfileReceipt
	receipt.Spec.Name = "workflow-apis"
	receipt.Spec.Kind = "http-mock"
	receipt.Spec.Version = "1.0.0"
	receipt.Status = "running"
	receipt.ProfileSHA256, _ = httpMockProfileDigest(profile)
	receipt.Addresses = map[string]string{"gmail.synthetic.invalid": "http://gmail.synthetic.invalid:18080"}
	receipt.OperationAddresses = []HTTPMockOperationAddress{{Selector: profile.Operations[0].Selector, Method: "POST", URL: "http://wrong.synthetic.invalid:18080/v1/messages"}}
	if err := validateProfileReceipt("workflow-apis", "1.0.0", receipt.ProfileSHA256, profile, receipt); err == nil {
		t.Fatal("receipt with a changed operation host must fail closed")
	}
}

func TestWorkspaceSandboxProfileClientCreatesOwnsAndDeletesTaskSandbox(t *testing.T) {
	simulator, err := NewDescriptionSimulator(readFixture(t, "native_service_catalog.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := simulator.ExportHTTPMockProfile()
	if err != nil {
		t.Fatal(err)
	}
	const id = "be8adfe8-89a9-4b47-83fb-c34b5ce4e010"
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sandboxes":
			var request WorkspaceSandboxTaskRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode task create: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if request.ProjectRoot != "/workspace/project" || request.ScopePath != "/workspace/project/.rehearsals/run-1" || request.Owner != "client-rehearsal:run-1" || request.OwnerType != "task" || request.IdempotencyKey != request.Owner {
				t.Errorf("task identity was not retained: %#v", request)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"` + id + `","name":"f7-run-1","projectRoot":"/workspace/project","scopePath":"/workspace/project/.rehearsals/run-1","owner":"client-rehearsal:run-1","ownerType":"task","idempotencyKey":"client-rehearsal:run-1"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/sandboxes/"+id+"/companion-services":
			var request struct {
				Services []struct {
					Name        string          `json:"name"`
					Kind        string          `json:"kind"`
					Version     string          `json:"version"`
					HTTPProfile HTTPMockProfile `json:"httpProfile"`
				} `json:"services"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || len(request.Services) != 1 || len(request.Services[0].HTTPProfile.Operations) != 4 {
				t.Errorf("unexpected task profile declaration: %#v err=%v", request, err)
			}
			_, _ = w.Write([]byte(`{"services":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sandboxes/"+id+"/companion-services/workflow-apis/start":
			var receipt WorkspaceSandboxProfileReceipt
			receipt.Spec.Name = "workflow-apis"
			receipt.Spec.Kind = "http-mock"
			receipt.Spec.Version = "1.0.0"
			receipt.Status = "running"
			receipt.ProfileSHA256, _ = httpMockProfileDigest(profile)
			receipt.Addresses = map[string]string{}
			for _, operation := range profile.Operations {
				receipt.Addresses[operation.Route.Host] = "http://" + operation.Route.Host + ":18080"
				receipt.OperationAddresses = append(receipt.OperationAddresses, HTTPMockOperationAddress{Selector: operation.Selector, Method: operation.Route.Method, URL: "http://" + operation.Route.Host + ":18080" + operation.Route.Path})
			}
			_ = json.NewEncoder(w).Encode(receipt)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sandboxes/"+id+"/companion-services/workflow-apis/stop":
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sandboxes/"+id:
			_, _ = w.Write([]byte(`{"deleted":true}`))
		default:
			t.Errorf("unexpected task lifecycle request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := NewWorkspaceSandboxProfileClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	lease, err := client.CreateAndStartTaskProfile(context.Background(), WorkspaceSandboxTaskRequest{
		Name: "f7-run-1", ProjectRoot: "/workspace/project", ScopePath: "/workspace/project/.rehearsals/run-1",
		Owner: "client-rehearsal:run-1", OwnerType: "task", IdempotencyKey: "client-rehearsal:run-1", Tags: []string{"f7", "task-owned"},
	}, "workflow-apis", "1.0.0", profile)
	if err != nil {
		t.Fatal(err)
	}
	if lease.SandboxID != id || lease.Name != "workflow-apis" || lease.Receipt.Status != "running" {
		t.Fatalf("unexpected task lease: %#v", lease)
	}
	if err := client.StopAndDeleteTaskProfile(context.Background(), lease); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"POST /api/v1/sandboxes",
		"PUT /api/v1/sandboxes/" + id + "/companion-services",
		"POST /api/v1/sandboxes/" + id + "/companion-services/workflow-apis/start",
		"POST /api/v1/sandboxes/" + id + "/companion-services/workflow-apis/stop",
		"DELETE /api/v1/sandboxes/" + id,
	}
	if len(methods) != len(want) {
		t.Fatalf("unexpected lifecycle calls: %v", methods)
	}
	for i := range want {
		if methods[i] != want[i] {
			t.Fatalf("lifecycle call %d = %q, want %q", i, methods[i], want[i])
		}
	}
}

func TestWorkspaceSandboxProfileClientDeletesCreatedSandboxWhenStartFails(t *testing.T) {
	const id = "be8adfe8-89a9-4b47-83fb-c34b5ce4e010"
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sandboxes":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"` + id + `","name":"f7-run-1","projectRoot":"/workspace/project","scopePath":"/workspace/project/.rehearsals/run-1","owner":"client-rehearsal:run-1","ownerType":"task","idempotencyKey":"client-rehearsal:run-1"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/sandboxes/"+id+"/companion-services":
			_, _ = w.Write([]byte(`{"services":[]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/start"):
			http.Error(w, "synthetic owner failure", http.StatusBadGateway)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sandboxes/"+id:
			deleted = true
			_, _ = w.Write([]byte(`{"deleted":true}`))
		default:
			t.Errorf("unexpected cleanup request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := NewWorkspaceSandboxProfileClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	profile := HTTPMockProfile{SchemaVersion: 1, Operations: []HTTPMockProfileOperation{{
		Selector: HTTPMockProfileSelector{NodeType: "n8n-nodes-base.gmail", Resource: "message", Operation: "send"},
		Route:    HTTPMockRoute{Host: "gmail.synthetic.invalid", Method: "POST", Path: "/v1/messages", Status: 202, ContentType: "application/json", Body: `{}`},
	}}}
	_, err = client.CreateAndStartTaskProfile(context.Background(), WorkspaceSandboxTaskRequest{
		Name: "f7-run-1", ProjectRoot: "/workspace/project", ScopePath: "/workspace/project/.rehearsals/run-1",
		Owner: "client-rehearsal:run-1", OwnerType: "task", IdempotencyKey: "client-rehearsal:run-1",
	}, "workflow-apis", "1.0.0", profile)
	if err == nil || !deleted {
		t.Fatalf("expected failed start and deletion of task sandbox, err=%v deleted=%v", err, deleted)
	}
}

func TestWorkspaceSandboxProfileClientRequiresTaskScopeInsideAuthorizedRoot(t *testing.T) {
	client, err := NewWorkspaceSandboxProfileClient("http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.CreateTaskSandbox(context.Background(), WorkspaceSandboxTaskRequest{
		Name: "f7-run-1", ProjectRoot: "/workspace/project", ScopePath: "/workspace/outside",
		Owner: "client-rehearsal:run-1", OwnerType: "task", IdempotencyKey: "client-rehearsal:run-1",
	})
	if err == nil {
		t.Fatal("scope outside project root must be rejected before request")
	}
}
