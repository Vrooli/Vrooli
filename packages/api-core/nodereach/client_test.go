package nodereach

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"connectrpc.com/connect"
	relayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay"
	relayconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay/relayv1connect"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/session"
	"google.golang.org/protobuf/proto"
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

func TestCallScenarioUsesBoundedTargetProcedure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/targets/node-1/scenarios/system-monitor/MetricsService/GetCurrentMetrics" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/proto" {
			t.Fatalf("content type = %q", r.Header.Get("Content-Type"))
		}
		_, _ = w.Write([]byte("response"))
	}))
	defer server.Close()
	client := New(Config{BridgeURL: server.URL, Token: "owner"})
	body, err := client.CallScenario(context.Background(), ScenarioRequest{NodeID: "node-1", Scenario: "system-monitor", Service: "MetricsService", Method: "GetCurrentMetrics", Body: []byte("request"), Timeout: time.Second, MaxResponse: 64})
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "response" {
		t.Fatalf("body = %q", body)
	}
}

func TestCallScenarioCarriesNonPostHTTPMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("proxy method = %q, want POST", r.Method)
		}
		if r.Header.Get("X-Vrooli-HTTP-Method") != http.MethodGet {
			t.Fatalf("forwarded method = %q, want GET", r.Header.Get("X-Vrooli-HTTP-Method"))
		}
		_, _ = w.Write([]byte("response"))
	}))
	defer server.Close()
	client := New(Config{BridgeURL: server.URL, Token: "owner"})
	if _, err := client.CallScenario(context.Background(), ScenarioRequest{NodeID: "node-1", Scenario: "vrooli-onboarding", Service: "api", Method: "v2/readiness", HTTPMethod: http.MethodGet, Timeout: time.Second}); err != nil {
		t.Fatal(err)
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

func TestOpenClassifiesHandshakeRejectionAndKeepsBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "node lacks required transport scope vrooli-bridge:write", http.StatusForbidden)
	}))
	defer server.Close()

	client := New(Config{BridgeURL: server.URL})
	_, err := client.Open(context.Background(), OpenRequest{NodeID: "node-1", Command: "shell"}, time.Second)
	if !IsKind(err, ErrMissingScope) {
		t.Fatalf("Open error = %v, want missing-scope classification", err)
	}
	if !strings.Contains(err.Error(), "vrooli-bridge:write") {
		t.Fatalf("Open error = %v, want server rejection body", err)
	}
}

func TestOpenClassifiesHandshakeHTTPStatuses(t *testing.T) {
	for _, tc := range []struct {
		status int
		kind   ErrorKind
	}{
		{http.StatusUnauthorized, ErrMissingReauth},
		{http.StatusForbidden, ErrMissingScope},
		{http.StatusNotFound, ErrNodeNotFound},
		{http.StatusServiceUnavailable, ErrNodeUnavailable},
		{http.StatusBadRequest, ErrTransport},
	} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "handshake diagnostic", tc.status)
			}))
			defer server.Close()

			client := New(Config{BridgeURL: server.URL})
			_, err := client.Open(context.Background(), OpenRequest{NodeID: "node-1"}, time.Second)
			if !IsKind(err, tc.kind) {
				t.Fatalf("status %d classified as %v, want %v (%v)", tc.status, err, tc.kind, err)
			}
			if !strings.Contains(err.Error(), "handshake diagnostic") {
				t.Fatalf("status %d error = %v, want response body", tc.status, err)
			}
		})
	}
}

func TestSessionReadReturnsBridgeCloseReason(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()
		open, _ := proto.Marshal(&sessionv1.Frame{Payload: &sessionv1.Frame_Open{Open: &sessionv1.Open{SessionId: "s1"}}})
		closeFrame, _ := proto.Marshal(&sessionv1.Frame{Payload: &sessionv1.Frame_Close{Close: &sessionv1.Close{Reason: "shell_not_allowed"}}})
		_ = conn.WriteMessage(websocket.BinaryMessage, open)
		_ = conn.WriteMessage(websocket.BinaryMessage, closeFrame)
		select {}
	}))
	defer server.Close()

	client := New(Config{BridgeURL: server.URL})
	sess, err := client.Open(context.Background(), OpenRequest{NodeID: "node-1"}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	buf := make([]byte, 1)
	_, err = sess.Read(buf)
	if err == nil || !strings.Contains(err.Error(), "shell_not_allowed") {
		t.Fatalf("Read error = %v, want close reason", err)
	}
	status, ok := sess.TerminalStatus()
	if !ok || status.Code != "closed" || status.Reason != "shell_not_allowed" {
		t.Fatalf("TerminalStatus() = (%+v, %t), want close reason", status, ok)
	}
}

func TestSessionReconnectsAndPreservesSessionID(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	var connections atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()
		if got := r.URL.Query().Get("session_id"); got != "session-1" {
			t.Errorf("session_id = %q, want session-1", got)
		}
		open, _ := proto.Marshal(&sessionv1.Frame{Payload: &sessionv1.Frame_Open{Open: &sessionv1.Open{SessionId: "session-1"}}})
		if err := conn.WriteMessage(websocket.BinaryMessage, open); err != nil {
			t.Errorf("write open: %v", err)
			return
		}
		if connections.Add(1) == 1 {
			time.Sleep(50 * time.Millisecond)
			return
		}
		data, _ := proto.Marshal(&sessionv1.Frame{Payload: &sessionv1.Frame_Data{Data: &sessionv1.Data{Data: []byte("reconnected")}}})
		_ = conn.WriteMessage(websocket.BinaryMessage, data)
		select {}
	}))
	defer server.Close()

	client := New(Config{BridgeURL: server.URL})
	sess, err := client.Open(context.Background(), OpenRequest{NodeID: "node-1", SessionID: "session-1"}, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	buf := make([]byte, len("reconnected"))
	if _, err := io.ReadFull(sess, buf); err != nil {
		t.Fatalf("read after reconnect: %v", err)
	}
	if string(buf) != "reconnected" {
		t.Fatalf("read after reconnect = %q", buf)
	}
}

func TestSessionReportsReconnectExhaustion(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	var connections atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if connections.Add(1) > 1 {
			http.Error(w, "node offline", http.StatusServiceUnavailable)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		open, _ := proto.Marshal(&sessionv1.Frame{Payload: &sessionv1.Frame_Open{Open: &sessionv1.Open{SessionId: "session-2"}}})
		_ = conn.WriteMessage(websocket.BinaryMessage, open)
		_ = conn.Close()
	}))
	defer server.Close()

	client := New(Config{BridgeURL: server.URL})
	sess, err := client.Open(context.Background(), OpenRequest{NodeID: "node-1", SessionID: "session-2"}, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	result := make(chan error, 1)
	go func() {
		_, readErr := sess.Read(make([]byte, 1))
		result <- readErr
	}()
	select {
	case readErr := <-result:
		if readErr == nil || !strings.Contains(readErr.Error(), "reconnect exhausted") {
			t.Fatalf("Read error = %v, want reconnect exhaustion", readErr)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Read remained blocked after reconnect attempts")
	}
	status, ok := sess.TerminalStatus()
	if !ok || status.Code != "reconnect_exhausted" {
		t.Fatalf("TerminalStatus() = (%+v, %t), want reconnect_exhausted", status, ok)
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
