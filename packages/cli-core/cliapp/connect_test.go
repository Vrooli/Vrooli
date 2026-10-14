package cliapp

import (
	"context"
	"fmt"
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

func TestNewConnectHTTPClientForwardsAgentIdentity(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "run-token")
	handler := connect.NewUnaryHandlerSimple(
		"/test.v1.Echo/Echo",
		func(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
			return wrapperspb.String(req.Value), nil
		},
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(cliutil.HeaderAgentIdentityToken); got != "run-token" {
			t.Fatalf("agent identity = %q", got)
		}
		handler.ServeHTTP(w, r)
	}))
	defer server.Close()

	app := callTestApp(t, server)
	app.tokenSource = func() string { return "" }
	httpClient, baseURL := NewConnectHTTPClient(app)
	client := connect.NewClient[wrapperspb.StringValue, wrapperspb.StringValue](httpClient, baseURL+"/test.v1.Echo/Echo")
	if _, err := client.CallUnary(context.Background(), connect.NewRequest(wrapperspb.String("request"))); err != nil {
		t.Fatalf("CallUnary: %v", err)
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

func TestConstructedConnectClientUsesInvocationAPIOverride(t *testing.T) {
	var staleCalls, selectedCalls int
	stale := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		staleCalls++
		http.Error(w, "wrong instance", http.StatusServiceUnavailable)
	}))
	defer stale.Close()
	handler := connect.NewUnaryHandlerSimple("/test.v1.Echo/Echo", func(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
		selectedCalls++
		return wrapperspb.String(req.Value + "-selected"), nil
	})
	selected := httptest.NewServer(handler)
	defer selected.Close()
	t.Setenv("CONNECT_INVOCATION_CONFIG", t.TempDir())
	app, err := NewScenarioApp(ScenarioOptions{Name: "connect-invocation", DefaultAPIBase: stale.URL, ConfigDirEnvVars: []string{"CONNECT_INVOCATION_CONFIG"}, AllowAnonymous: true})
	if err != nil {
		t.Fatalf("NewScenarioApp: %v", err)
	}
	httpClient, baseURL := NewConnectHTTPClient(app)
	client := connect.NewClient[wrapperspb.StringValue, wrapperspb.StringValue](httpClient, baseURL+"/test.v1.Echo/Echo")
	app.SetCommands([]CommandGroup{{Title: "Test", Commands: []Command{{Name: "echo", Run: func([]string) error {
		resp, err := client.CallUnary(context.Background(), connect.NewRequest(wrapperspb.String("request")))
		if err == nil && resp.Msg.Value != "request-selected" {
			return fmt.Errorf("unexpected selected response %q", resp.Msg.Value)
		}
		return err
	}}}}})
	if err := app.CLI.Run([]string{"--api-base", selected.URL, "echo"}); err != nil {
		t.Fatalf("invocation-selected RPC: %v", err)
	}
	if staleCalls != 0 || selectedCalls != 1 {
		t.Fatalf("stale instance calls=%d, selected instance calls=%d", staleCalls, selectedCalls)
	}
}

func TestConnectWrapAPIError(t *testing.T) {
	err := connect.NewError(connect.CodeInvalidArgument, io.ErrUnexpectedEOF)
	wrapped := WrapAPIError("create note", err, nil)
	if !strings.Contains(wrapped.Error(), "create note: invalid_argument: unexpected EOF") {
		t.Fatalf("wrapped = %v", wrapped)
	}
}

func TestConnectInvocationSelectionPreservesPathQueryAndExplicitTarget(t *testing.T) {
	var selectedPath, selectedQuery string
	selected := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		selectedPath, selectedQuery = r.URL.Path, r.URL.RawQuery
		_, _ = io.WriteString(w, "selected")
	}))
	defer selected.Close()
	explicit := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "explicit")
	}))
	defer explicit.Close()
	app := callTestApp(t, selected)
	app.baseOptions = func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{Override: "http://old.invalid/original"} }
	client, initial := NewConnectHTTPClient(app)
	app.baseOptions = func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{Override: selected.URL + "/chosen"} }
	req, err := http.NewRequest(http.MethodPost, initial+"/test.v1.Echo/Echo?cursor=a%2Fb", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if selectedPath != "/chosen/test.v1.Echo/Echo" || selectedQuery != "cursor=a%2Fb" {
		t.Fatalf("selected request path=%q query=%q", selectedPath, selectedQuery)
	}
	req, err = http.NewRequest(http.MethodGet, explicit.URL+"/explicit", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || string(body) != "explicit" {
		t.Fatalf("explicit target response=%q err=%v", body, err)
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
