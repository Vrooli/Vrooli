package nodeclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	relayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay"
	relayconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay/relayv1connect"
)

func TestCallRequestPreservesTypedArguments(t *testing.T) {
	// The public request shape itself is the fidelity contract. The transport
	// test below verifies the protobuf projection with a local Connect server in
	// the package's integration suite; this assertion prevents future CSV APIs.
	req := CallRequest{NodeID: "node", Scenario: "demo", Command: "run", Args: []string{"a,b", ""}}
	if len(req.Args) != 2 || req.Args[0] != "a,b" || req.Args[1] != "" {
		t.Fatalf("typed args changed: %#v", req.Args)
	}
}

func TestEndpointUsesResolverAndTimeoutIsExplicit(t *testing.T) {
	called := false
	c := New(Config{HTTPClient: &http.Client{}, ResolveBridgeURL: func(context.Context) (string, error) {
		called = true
		return "http://bridge.test/", nil
	}})
	if _, err := c.endpoint(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("Bridge resolver was not called")
	}
	if got := seconds(1500 * time.Millisecond); got != 2 {
		t.Fatalf("seconds = %d, want 2", got)
	}
}

func TestMissingNodeIsTyped(t *testing.T) {
	c := New(Config{})
	_, err := c.Call(context.Background(), CallRequest{Command: "status"})
	if !IsKind(err, ErrNodeNotFound) {
		t.Fatalf("error = %v, want typed node-not-found", err)
	}
	var typed *Error
	if !errors.As(err, &typed) || typed.Node == "" && typed.Verb == "" {
		t.Fatalf("typed details missing: %#v", typed)
	}
}

func TestOpenReturnsTypedTransportFailure(t *testing.T) {
	c := New(Config{ResolveBridgeURL: func(context.Context) (string, error) {
		return "http://127.0.0.1:1", nil
	}})
	_, err := c.Open(context.Background(), OpenRequest{NodeID: "n", Command: "shell"}, time.Second)
	if !IsKind(err, ErrTransport) {
		t.Fatalf("Open error = %v, want typed transport failure", err)
	}
}

type relayRecorder struct {
	request *relayv1.RelayCallRequest
}

func (r *relayRecorder) Call(_ context.Context, req *connect.Request[relayv1.RelayCallRequest]) (*connect.Response[relayv1.RelayCallResponse], error) {
	r.request = req.Msg
	return connect.NewResponse(&relayv1.RelayCallResponse{Outcome: relayv1.RelayCallOutcome_RELAY_CALL_OUTCOME_COMPLETED, Data: []byte("ok")}), nil
}

func TestCallProjectsRepeatedArgumentsWithoutCSVFlattening(t *testing.T) {
	recorder := &relayRecorder{}
	path, handler := relayconnect.NewRelayServiceHandler(recorder)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, path) {
			handler.ServeHTTP(w, req)
			return
		}
		http.NotFound(w, req)
	}))
	defer server.Close()

	client := New(Config{BridgeURL: server.URL})
	response, err := client.Call(context.Background(), CallRequest{
		NodeID: "node", Scenario: "system-monitor", Command: "metrics current",
		Args: []string{"a,b", "", "--json"}, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(response.Data) != "ok" {
		t.Fatalf("response data = %q", response.Data)
	}
	if got, want := recorder.request.GetArgs(), []string{"a,b", "", "--json"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

func TestAuthTransportUsesProviderScheme(t *testing.T) {
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		authorization = req.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	transport := &authTransport{
		base: http.DefaultClient, ctx: context.Background(),
		tokenProvider: func(context.Context) (string, error) { return "LocalSession OS1.test", nil },
	}
	request, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := transport.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if got, want := authorization, "LocalSession OS1.test"; got != want {
		t.Fatalf("authorization = %q, want %q", got, want)
	}
}
