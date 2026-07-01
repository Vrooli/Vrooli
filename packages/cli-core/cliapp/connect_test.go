package cliapp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestNewConnectHTTPClientCallsConnectHandler(t *testing.T) {
	handler := connect.NewUnaryHandlerSimple(
		"/test.v1.Echo/Echo",
		func(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
			return wrapperspb.String(req.Value + "-response"), nil
		},
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q", got)
		}
		handler.ServeHTTP(w, r)
	}))
	defer server.Close()

	app := callTestApp(t, server)
	app.tokenSource = func() string { return "test-token" }
	httpClient, baseURL := NewConnectHTTPClient(app)
	client := connect.NewClient[wrapperspb.StringValue, wrapperspb.StringValue](httpClient, baseURL+"/test.v1.Echo/Echo")

	resp, err := client.CallUnary(context.Background(), connect.NewRequest(wrapperspb.String("request")))
	if err != nil {
		t.Fatalf("CallUnary: %v", err)
	}
	if resp.Msg.Value != "request-response" {
		t.Fatalf("response = %q", resp.Msg.Value)
	}
}

func TestNewConnectHTTPClientUsesRootBase(t *testing.T) {
	handler := connect.NewUnaryHandlerSimple(
		"/test.v1.Echo/Echo",
		func(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
			return wrapperspb.String(req.Value), nil
		},
	)
	server := httptest.NewServer(handler)
	defer server.Close()

	app := callTestApp(t, server)
	app.options.APIPrefix = "/api/v1"
	httpClient, baseURL := NewConnectHTTPClient(app)
	client := connect.NewClient[wrapperspb.StringValue, wrapperspb.StringValue](httpClient, baseURL+"/test.v1.Echo/Echo")

	resp, err := client.CallUnary(context.Background(), connect.NewRequest(wrapperspb.String("request")))
	if err != nil {
		t.Fatalf("CallUnary: %v", err)
	}
	if resp.Msg.Value != "request" {
		t.Fatalf("response = %q", resp.Msg.Value)
	}
	if baseURL != server.URL {
		t.Fatalf("baseURL = %q, want %q", baseURL, server.URL)
	}
}

func TestConnectHTTPClientResolvesRelativeURLAtCallTime(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = io.WriteString(w, "{}")
	}))
	defer server.Close()

	app := &ScenarioApp{
		HTTPClient: cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		options:    ScenarioOptions{APIPrefix: "/api/v1"},
	}
	app.baseOptions = func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{Override: server.URL}
	}
	app.tokenSource = func() string { return "" }

	client := &scenarioConnectHTTPClient{app: app, client: server.Client()}
	req, err := http.NewRequest(http.MethodPost, "/vrooli.plan_manager.v1.plans.PlansService/ReconcilePlans", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	_ = resp.Body.Close()

	if gotPath != "/vrooli.plan_manager.v1.plans.PlansService/ReconcilePlans" {
		t.Fatalf("path = %q, want Connect RPC path", gotPath)
	}
}

func TestConnectWrapAPIError(t *testing.T) {
	err := connect.NewError(connect.CodeInvalidArgument, io.ErrUnexpectedEOF)
	wrapped := WrapAPIError("create note", err, nil)
	if !strings.Contains(wrapped.Error(), "create note: invalid_argument: unexpected EOF") {
		t.Fatalf("wrapped = %v", wrapped)
	}
}

func TestNewConnectHTTPClientRejectsInvalidConfiguration(t *testing.T) {
	httpClient, baseURL := NewConnectHTTPClient(nil)
	if baseURL != "" {
		t.Fatalf("baseURL = %q, want empty", baseURL)
	}
	req := httptest.NewRequest(http.MethodPost, "http://example.test/service/Method", nil)
	if _, err := httpClient.Do(req); err == nil {
		t.Fatal("expected nil app error")
	}

	app := callTestApp(t, httptest.NewServer(http.NotFoundHandler()))
	app.baseOptions = func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{} }
	client := &scenarioConnectHTTPClient{app: app}
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.URL = &url.URL{Path: "/relative"}
	if _, err := client.Do(req); err == nil {
		t.Fatal("expected relative URL error")
	}
	if _, err := client.Do(nil); err == nil {
		t.Fatal("expected nil request error")
	}
}
