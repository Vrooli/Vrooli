package cliapp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
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

func TestConnectWrapAPIError(t *testing.T) {
	err := connect.NewError(connect.CodeInvalidArgument, io.ErrUnexpectedEOF)
	wrapped := WrapAPIError("create note", err, nil)
	if !strings.Contains(wrapped.Error(), "create note: invalid_argument: unexpected EOF") {
		t.Fatalf("wrapped = %v", wrapped)
	}
}
