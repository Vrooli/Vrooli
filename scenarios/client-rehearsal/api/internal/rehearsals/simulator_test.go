package rehearsals

import (
	"context"
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
