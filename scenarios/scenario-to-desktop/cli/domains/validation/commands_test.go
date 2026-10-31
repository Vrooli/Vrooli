package validation

import (
	"context"
	"net/url"
	"testing"
)

type fakeHTTPClient struct {
	method string
	path   string
	result []byte
}

func (f *fakeHTTPClient) DoWithContext(_ context.Context, method, path string, _ url.Values, _ interface{}) ([]byte, error) {
	f.method, f.path = method, path
	return f.result, nil
}

func TestStatusRequestReturnsTerminalValidationState(t *testing.T) {
	client := &fakeHTTPClient{result: []byte(`{"run_id":"run-1","state":"failed","gate":{"disposition":"unavailable"}}`)}
	commands := &Commands{client: client, prefix: "/api/v1"}
	value, err := commands.request(nil, "GET", "/validation/matrices/run-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if client.method != "GET" || client.path != "/api/v1/validation/matrices/run-1" {
		t.Fatalf("request = %s %s, want GET /api/v1/validation/matrices/run-1", client.method, client.path)
	}
	if got := stringField(value, "state"); got != "failed" {
		t.Fatalf("state = %q, want failed", got)
	}
	gate, ok := value.AsMap()["gate"].(map[string]any)
	if !ok || gate["disposition"] != "unavailable" {
		t.Fatalf("gate = %#v, want unavailable disposition", value.AsMap()["gate"])
	}
}
