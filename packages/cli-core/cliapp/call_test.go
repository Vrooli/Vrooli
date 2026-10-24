package cliapp

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/vrooli/cli-core/cliutil"
)

// callTestApp builds a ScenarioApp pointed at a test server. It avoids the
// full NewScenarioApp wiring (config files, port detectors) — Call needs only
// app.Request, which delegates to APIClient.
func callTestApp(t *testing.T, server *httptest.Server) *ScenarioApp {
	t.Helper()
	httpClient := cliutil.NewHTTPClient(cliutil.HTTPClientOptions{
		BaseOptions: cliutil.APIBaseOptions{Override: server.URL},
	})
	app := &ScenarioApp{
		HTTPClient: httpClient,
		options:    ScenarioOptions{APIPrefix: "/"},
	}
	app.baseOptions = func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{Override: server.URL}
	}
	app.tokenSource = func() string { return "" }
	app.APIClient = cliutil.NewAPIClient(app.HTTPClient, app.baseOptions, app.tokenSource)
	return app
}

func TestCallSuccess(t *testing.T) {
	// wrapperspb.StringValue's protojson form is the bare JSON string "hello".
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.TrimSpace(string(body)) != `"hello"` {
			t.Errorf("unexpected request body: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `"world"`)
	}))
	defer server.Close()

	app := callTestApp(t, server)
	req := wrapperspb.String("hello")
	resp, err := Call[*wrapperspb.StringValue, *wrapperspb.StringValue](app, http.MethodPost, "/echo", req)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if resp.Value != "world" {
		t.Errorf("got %q, want %q", resp.Value, "world")
	}
}

func TestCallNonOKWithEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"invalid_request","message":"value required"}`)
	}))
	defer server.Close()

	app := callTestApp(t, server)
	_, err := Call[*wrapperspb.StringValue, *wrapperspb.StringValue](
		app, http.MethodPost, "/echo", wrapperspb.String(""))
	if err == nil {
		t.Fatal("expected error from non-2xx")
	}
	var apiErr *cliutil.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *cliutil.APIError, got %T: %v", err, err)
	}
	wrapped := WrapAPIError("echo", err, nil)
	if !strings.Contains(wrapped.Error(), "echo: invalid_request: value required") {
		t.Errorf("unexpected wrapped error: %v", wrapped)
	}
}

func TestCallNonOKWithoutEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "boom")
	}))
	defer server.Close()

	app := callTestApp(t, server)
	_, err := Call[*wrapperspb.StringValue, *wrapperspb.StringValue](
		app, http.MethodPost, "/echo", wrapperspb.String(""))
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *cliutil.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *cliutil.APIError, got %T", err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("status: got %d", apiErr.StatusCode)
	}
}

func TestCallNilRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if len(body) != 0 {
			t.Errorf("expected empty body for nil request, got %q", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `"ok"`)
	}))
	defer server.Close()

	app := callTestApp(t, server)
	resp, err := Call[*wrapperspb.StringValue, *wrapperspb.StringValue](
		app, http.MethodGet, "/ping", nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if resp.Value != "ok" {
		t.Errorf("got %q", resp.Value)
	}
}

func TestCallQuery(t *testing.T) {
	// structpb.Struct's protojson form is a plain JSON object — keys map to
	// Values inferred from the JSON type ("b" → string Value).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.URL.Query().Get("filter"); got != "all" {
			t.Errorf("filter: got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"a":"b"}`)
	}))
	defer server.Close()

	app := callTestApp(t, server)
	q := make(map[string][]string)
	q["filter"] = []string{"all"}
	resp, err := CallQuery[*structpb.Struct](app, "/items", q)
	if err != nil {
		t.Fatalf("CallQuery: %v", err)
	}
	if resp.Fields["a"].GetStringValue() != "b" {
		t.Errorf("structpb fields: got %v", resp.Fields)
	}
}

func TestIsNilProto(t *testing.T) {
	if !isNilProto(nil) {
		t.Error("untyped nil should be nil-proto")
	}
	var typed *wrapperspb.StringValue
	if !isNilProto(typed) {
		t.Error("typed nil pointer should be nil-proto")
	}
	if isNilProto(wrapperspb.String("x")) {
		t.Error("non-nil proto should not be nil-proto")
	}
}
