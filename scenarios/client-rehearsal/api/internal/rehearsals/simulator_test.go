package rehearsals

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// [REQ:CR-P0-001]
func TestDescriptionSimulatorUsesOpenAPICatalogAndInjectedFault(t *testing.T) {
	simulator, err := NewDescriptionSimulator(readFixture(t, "api_catalog_openapi.json"), true)
	if err != nil {
		t.Fatal(err)
	}
	step := Step{Method: "POST", Host: "crm.synthetic.invalid", URL: "/v1/contacts", Operation: "POST /v1/contacts"}
	if _, err = simulator.Invoke(context.Background(), step, 1, true); err == nil {
		t.Fatal("expected seeded first-attempt fault")
	}
	response, err := simulator.Invoke(context.Background(), step, 2, true)
	if err != nil {
		t.Fatal(err)
	}
	if response != `{"id":"synthetic-id","status":"active"}` {
		t.Fatalf("unexpected OpenAPI example response %s", response)
	}

	response, err = simulator.Invoke(context.Background(), Step{Method: "POST", Host: "sheets.synthetic.invalid", URL: "/v1/rows"}, 1, false)
	if err != nil || response != `{"status":"active","updatedRange":"A2:B2"}` {
		t.Fatalf("unexpected second service result %s, %v", response, err)
	}
}

func TestSimulatorSourceFingerprintOrdersEqualNativeSelectorsByRouteAndInputMatch(t *testing.T) {
	profile := `{"kind":"openapi","document":{"openapi":"3.0.3","info":{"title":"Synthetic routed operation","version":"1"},"servers":[{"url":"https://example.synthetic.invalid"}],"paths":{"/v1/a":{"post":{"x-n8n-node-type":"n8n-nodes-base.googleSheets","x-n8n-resource":"sheet","x-n8n-operation":"read","x-n8n-input-match":{"kind":"alpha"},"responses":{"200":{"content":{"application/json":{"example":{"status":"ok"}}}}}}},"/v1/b":{"post":{"x-n8n-node-type":"n8n-nodes-base.googleSheets","x-n8n-resource":"sheet","x-n8n-operation":"read","x-n8n-input-match":{"kind":"beta"},"responses":{"200":{"content":{"application/json":{"example":{"status":"ok"}}}}}}}}}}`
	first, err := simulatorSourceSHA256(profile)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		got, err := simulatorSourceSHA256(profile)
		if err != nil {
			t.Fatal(err)
		}
		if got != first {
			t.Fatalf("equivalent simulator source fingerprint changed on rebuild %d: first=%s got=%s", i+1, first, got)
		}
	}
}

func TestDescriptionSimulatorMatchesRecordedExchangeByHostMethodAndPath(t *testing.T) {
	simulator, err := NewDescriptionSimulator(readFixture(t, "error_path_recorded_exchanges.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := simulator.Invoke(context.Background(), Step{Method: "POST", Host: "crm.synthetic.invalid", URL: "/v1/events"}, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"id":"synthetic-id","status":"active"}` {
		t.Fatalf("got %s", got)
	}
	if _, err = simulator.Invoke(context.Background(), Step{Method: "POST", Host: "wrong.synthetic.invalid", URL: "/v1/events"}, 1, false); err == nil {
		t.Fatal("recorded exchange must not match a different host")
	}
}

func TestNativeServiceBindingUsesProfileRouteAndRejectsUnmappedOperations(t *testing.T) {
	simulator, err := NewDescriptionSimulator(readFixture(t, "native_service_catalog.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	response, err := simulator.Invoke(context.Background(), Step{NodeType: "n8n-nodes-base.gmail", Resource: "message", Action: "send", Host: "attacker.invalid", URL: "/exfiltrate"}, 1, false)
	if err != nil || response != `{"id":"synthetic-id","status":"active"}` {
		t.Fatalf("expected profile route to ignore workflow destination and use synthetic fixture: %s, %v", response, err)
	}
	if _, err := simulator.Invoke(context.Background(), Step{NodeType: "n8n-nodes-base.gmail", Resource: "message", Action: "delete"}, 1, false); err == nil {
		t.Fatal("unmapped native operation must fail closed")
	}
}

func TestNativeServiceCatalogFingerprintsEveryMappedOperation(t *testing.T) {
	simulator, err := NewDescriptionSimulator(readFixture(t, "native_service_catalog.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := json.Marshal(simulator.sourceFingerprint())
	if err != nil {
		t.Fatal(err)
	}
	var projected struct {
		NativeOperations []struct {
			Service  string `json:"service"`
			NodeType string `json:"node_type"`
			Resource string `json:"resource"`
			Action   string `json:"action"`
		} `json:"native_operations"`
	}
	if err := json.Unmarshal(fingerprint, &projected); err != nil {
		t.Fatal(err)
	}
	if len(projected.NativeOperations) != 4 {
		t.Fatalf("fingerprint must retain all mapped native operations, got %#v", projected.NativeOperations)
	}
	seen := map[string]bool{}
	for _, operation := range projected.NativeOperations {
		key := operation.NodeType + "/" + operation.Resource + "/" + operation.Action
		seen[key] = true
	}
	for _, key := range []string{
		"n8n-nodes-base.gmail/message/send",
		"n8n-nodes-base.googleSheets/sheet/append",
		"n8n-nodes-base.slack/message/post",
		"n8n-nodes-base.hubspot/contact/upsert",
	} {
		if !seen[key] {
			t.Errorf("fingerprint omitted native selector %s", key)
		}
	}
}

func TestDescriptionSimulatorSanitizesRecordedValuesAndHashesOnlyProfileShape(t *testing.T) {
	first := `{"kind":"recorded-exchanges","exchanges":[{"method":"GET","host":"crm.synthetic.invalid","path":"/v1/contacts/123","status":200,"response":{"id":"client-123","email":"person@private.example","name":"Avery Client"}}]}`
	second := `{"kind":"recorded-exchanges","exchanges":[{"method":"GET","host":"crm.synthetic.invalid","path":"/v1/contacts/123","status":200,"response":{"id":"other-client","email":"someone@different.example","name":"Morgan Client"}}]}`
	sim, err := NewDescriptionSimulator(first, false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := sim.Invoke(context.Background(), Step{Method: "GET", Host: "crm.synthetic.invalid", URL: "/v1/contacts/123"}, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"client-123", "person@private.example", "Avery Client"} {
		if strings.Contains(got, private) {
			t.Fatalf("simulator returned recorded value %q: %s", private, got)
		}
	}
	firstHash, err := simulatorSourceSHA256(first)
	if err != nil {
		t.Fatal(err)
	}
	secondHash, err := simulatorSourceSHA256(second)
	if err != nil {
		t.Fatal(err)
	}
	if firstHash != secondHash {
		t.Fatal("simulator provenance should fingerprint operation structure, not response values")
	}
}

func TestDescriptionSimulatorRejectsUnmatchedOperationWithoutForwarding(t *testing.T) {
	simulator, err := NewDescriptionSimulator(readFixture(t, "api_catalog_openapi.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = simulator.Invoke(context.Background(), Step{Method: "POST", Host: "unknown.synthetic.invalid", URL: "/collect"}, 1, false); err == nil {
		t.Fatal("unknown destination must fail locally")
	}
}

func TestDescriptionSimulatorHTTPBridgeServesDeclaredHostOnly(t *testing.T) {
	simulator, err := NewDescriptionSimulator(readFixture(t, "api_catalog_openapi.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(simulator)
	defer server.Close()

	request, err := http.NewRequest(http.MethodPost, server.URL+"/v1/contacts", strings.NewReader(`{"synthetic":true}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "crm.synthetic.invalid"
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusCreated || string(body) != `{"id":"synthetic-id","status":"active"}` {
		t.Fatalf("unexpected bridge response status=%d body=%s", response.StatusCode, body)
	}
	if response.Header.Get("X-Workflow-Simulator") != "synthetic" || response.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("bridge did not identify synthetic, non-cacheable result: %v", response.Header)
	}

	request, err = http.NewRequest(http.MethodPost, server.URL+"/v1/contacts", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "unlisted.invalid"
	response, err = server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("unlisted host must fail closed, got %d", response.StatusCode)
	}
	if got := simulator.Hosts(); len(got) != 2 || got[0] != "crm.synthetic.invalid" || got[1] != "sheets.synthetic.invalid" {
		t.Fatalf("unexpected exact host allowlist: %v", got)
	}
}

func TestDescriptionSimulatorHTTPBridgeEnforcesRequestSize(t *testing.T) {
	simulator, err := NewDescriptionSimulator(readFixture(t, "api_catalog_openapi.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(simulator)
	defer server.Close()
	request, err := http.NewRequest(http.MethodPost, server.URL+"/v1/contacts", strings.NewReader(strings.Repeat("x", maxSimulatorRequestBytes+1)))
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "crm.synthetic.invalid"
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body should be rejected, got %d", response.StatusCode)
	}
}

func TestDescriptionSimulatorExportsBoundedSandboxHTTPProfile(t *testing.T) {
	simulator, err := NewDescriptionSimulator(readFixture(t, "native_service_catalog.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := simulator.ExportHTTPMockProfile()
	if err != nil {
		t.Fatal(err)
	}
	if profile.SchemaVersion != 1 || len(profile.Operations) != 4 {
		t.Fatalf("expected schema-v1 profile with three explicit native operations: %#v", profile)
	}
	want := map[string]HTTPMockRoute{
		"n8n-nodes-base.gmail|message|send":        {Host: "gmail.synthetic.invalid", Method: "POST", Path: "/v1/messages", Status: http.StatusOK, ContentType: "application/json", Body: `{"id":"synthetic-id","status":"active"}`},
		"n8n-nodes-base.googlesheets|sheet|append": {Host: "sheets.synthetic.invalid", Method: "POST", Path: "/v1/rows", Status: http.StatusOK, ContentType: "application/json", Body: `{"status":"active","updatedRange":"A2:B2"}`},
		"n8n-nodes-base.slack|message|post":        {Host: "slack.synthetic.invalid", Method: "POST", Path: "/v1/messages", Status: http.StatusOK, ContentType: "application/json", Body: `{"id":"synthetic-id","status":"active"}`},
		"n8n-nodes-base.hubspot|contact|upsert":    {Host: "crm.synthetic.invalid", Method: "POST", Path: "/v1/contacts", Status: http.StatusOK, ContentType: "application/json", Body: `{"id":"synthetic-id","status":"active"}`},
	}
	for _, operation := range profile.Operations {
		key := nativeOperationKey(operation.Selector.NodeType, operation.Selector.Resource, operation.Selector.Operation)
		route, ok := want[key]
		if !ok || route != operation.Route {
			t.Fatalf("unexpected exported operation %q: %#v", key, operation)
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("profile omitted expected selectors: %#v", want)
	}
}

func TestDescriptionSimulatorExportsInputConditionedOperationsAsExactSandboxQueries(t *testing.T) {
	simulator, err := NewDescriptionSimulator(readFixture(t, "public_workflow_16364_catalog.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := simulator.ExportHTTPMockProfile()
	if err != nil {
		t.Fatalf("export disjoint native selectors: %v", err)
	}
	if len(profile.Operations) != 4 {
		t.Fatalf("expected two declared Sheets read selectors plus generic Sheets/Gmail operations, got %#v", profile.Operations)
	}
	seen := map[string]bool{}
	for _, operation := range profile.Operations {
		if operation.Selector.Operation != "read" {
			continue
		}
		if operation.Selector.NodeType != "n8n-nodes-base.googleSheets" || operation.Selector.Resource != "sheet" || operation.Selector.Operation != "read" || len(operation.Selector.InputMatch) != 1 {
			t.Fatalf("operation lost generic native selector identity: %#v", operation)
		}
		sheetName, ok := operation.Selector.InputMatch["sheetName"].(string)
		if !ok || operation.Route.Query != "sheetName="+sheetName || operation.Route.Host != "sheets.synthetic.invalid" {
			t.Fatalf("Sandbox route did not preserve exact selector query: %#v", operation)
		}
		seen[sheetName] = true
	}
	if !seen["products"] || !seen["recipients"] {
		t.Fatalf("profile omitted an input-conditioned Sheets operation: %#v", seen)
	}
}

func TestDescriptionSimulatorHTTPMockExportFailsClosedOnNonSyntheticOrDynamicRoute(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"real host", `{"kind":"openapi","document":{"servers":[{"url":"https://api.example.com"}],"paths":{"/v1/contacts":{"post":{"x-n8n-node-type":"n8n-nodes-base.hubspot","x-n8n-resource":"contact","x-n8n-operation":"upsert","responses":{"200":{"content":{"application/json":{"example":{"id":"source-id"}}}}}}}}}}`},
		{"parameterized path", `{"kind":"openapi","document":{"servers":[{"url":"https://crm.synthetic.invalid"}],"paths":{"/v1/contacts/{contactId}":{"get":{"x-n8n-node-type":"n8n-nodes-base.hubspot","x-n8n-resource":"contact","x-n8n-operation":"get","responses":{"200":{"content":{"application/json":{"example":{"id":"source-id"}}}}}}}}}}`},
		{"query parameter", `{"kind":"openapi","document":{"servers":[{"url":"https://crm.synthetic.invalid"}],"paths":{"/v1/contacts":{"get":{"parameters":[{"name":"filter","in":"query","schema":{"type":"string"}}],"x-n8n-node-type":"n8n-nodes-base.hubspot","x-n8n-resource":"contact","x-n8n-operation":"getAll","responses":{"200":{"content":{"application/json":{"example":{"items":[]}}}}}}}}}}`},
		{"secret-shaped response", `{"kind":"openapi","document":{"servers":[{"url":"https://crm.synthetic.invalid"}],"paths":{"/v1/contacts":{"get":{"x-n8n-node-type":"n8n-nodes-base.hubspot","x-n8n-resource":"contact","x-n8n-operation":"getAll","responses":{"200":{"content":{"application/json":{"example":{"access_token":"must-not-export"}}}}}}}}}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			simulator, err := NewDescriptionSimulator(tc.raw, false)
			if err != nil {
				t.Fatalf("description should be valid for in-process simulation: %v", err)
			}
			if _, err := simulator.ExportHTTPMockProfile(); err == nil {
				t.Fatal("unsafe or dynamic route must fail closed for Sandbox export")
			}
		})
	}
}

func TestSyntheticizePreservesOnlyExplicitlySyntheticURLs(t *testing.T) {
	got := syntheticize(map[string]any{
		"synthetic": "https://feeds.synthetic.invalid/industry",
		"public":    "https://example.com/feed",
		"query":     "https://feeds.synthetic.invalid/feed?token=secret",
	}, "").(map[string]any)
	if got["synthetic"] != "https://feeds.synthetic.invalid/industry" {
		t.Fatalf("deterministic synthetic URL should retain route semantics: %#v", got)
	}
	if got["public"] != "synthetic-value" || got["query"] != "synthetic-value" {
		t.Fatalf("real or credential-bearing URLs must be replaced: %#v", got)
	}
}

func TestSimulatorPreservesOnlySanitizedBoundedBase64JSONFixtures(t *testing.T) {
	payload := map[string]any{"id": "real-secret-id", "name": "Client Person", "status": "open", "email": "client@example.com", "token": "private-token"}
	plain, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	profileValue := map[string]any{"encoding": "base64-json", "data": base64.StdEncoding.EncodeToString(plain)}
	cleaned, ok := syntheticize(profileValue, "").(map[string]any)
	if !ok || cleaned["encoding"] != "base64-json" {
		t.Fatalf("encoded JSON marker was not preserved: %#v", cleaned)
	}
	encoded, ok := cleaned["data"].(string)
	if !ok {
		t.Fatalf("sanitized binary payload missing: %#v", cleaned)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("sanitized output is not base64: %v", err)
	}
	var got map[string]any
	if err = json.Unmarshal(decoded, &got); err != nil {
		t.Fatalf("sanitized payload is not JSON: %v", err)
	}
	if got["id"] != "synthetic-id" || got["name"] != "synthetic-value" || got["status"] != "open" || got["email"] != "person@example.invalid" || got["token"] != "synthetic-value" {
		t.Fatalf("binary JSON values were not safely syntheticized: %#v", got)
	}
	bad, _ := syntheticize(map[string]any{"encoding": "base64-json", "data": "not base64"}, "").(map[string]any)
	if bad["data"] == "not base64" {
		t.Fatal("invalid encoded source bytes were retained")
	}
}
