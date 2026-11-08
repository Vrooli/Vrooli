package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/protoconv"
	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/vrooli/api-core/apihttp"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	routingpb "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
)

// Actual HTTP publishers and WebSocket upgrades share the routing RPC owner.
// Per-partition barriers prove absence of foreign delivery without relying on
// a timeout read (which permanently poisons a Gorilla WebSocket connection).
func TestWebSocketDeliveryKeepsExactRequestLeasePartition(t *testing.T) {
	t.Setenv(apihttp.TestModeForceEnableEnv, "1")
	db, err := coredb.Open(t.Context(), coredb.Config{Driver: coredb.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	root := t.TempDir()
	roots := filerouting.New(storage.Paths{ConfigDir: filepath.Join(root, "config"), DataDir: filepath.Join(root, "data"), CacheDir: filepath.Join(root, "cache"), LogsDir: filepath.Join(root, "logs"), StateDir: filepath.Join(root, "state")})
	routing := devrouting.NewService(db, roots)
	hub := NewWebSocketHub()
	go hub.Run()
	mux := http.NewServeMux()
	devrouting.RegisterWithFileRootsService(mux, db, roots, routing)
	mux.HandleFunc("/ws", (&Handler{}).HandleWebSocket(hub))
	contexts := make(chan context.Context, 1)
	mux.HandleFunc("/pin", func(w http.ResponseWriter, r *http.Request) {
		contexts <- context.WithoutCancel(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/publish", func(w http.ResponseWriter, r *http.Request) {
		id := uuid.MustParse(r.URL.Query().Get("id"))
		switch r.URL.Query().Get("kind") {
		case "event":
			hub.BroadcastEvent(r.Context(), domain.NewLogEvent(id, "info", "partition fixture"))
		case "run":
			hub.BroadcastRunStatus(r.Context(), &domain.Run{ID: id, Status: domain.RunStatusRunning})
		case "progress":
			hub.BroadcastProgress(r.Context(), id, domain.RunPhaseExecuting, 50, "partition fixture")
		case "task":
			hub.BroadcastTaskStatus(r.Context(), &domain.Task{ID: id, Status: domain.TaskStatusQueued})
		case "workflow":
			hub.BroadcastWorkflowLifecycle(r.Context(), &domain.WorkflowLifecycleEvent{ExecutionID: id, Status: domain.WorkflowExecutionSucceeded})
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(routing.PinnedTestModeMiddleware(mux))
	t.Cleanup(server.Close)
	rpc := routingconnect.NewRoutingServiceClient(server.Client(), server.URL)
	install := func() {
		t.Helper()
		_, err := rpc.InstallTestPool(t.Context(), connect.NewRequest(&routingpb.InstallTestPoolRequest{Dsn: filepath.Join(t.TempDir(), "leased.db"), LeaseId: "same-id", EmptyConfig: true}))
		if err != nil {
			t.Fatal(err)
		}
	}
	clear := func() {
		t.Helper()
		if _, err := rpc.ClearTestPool(t.Context(), connect.NewRequest(&routingpb.ClearTestPoolRequest{LeaseId: "same-id"})); err != nil {
			t.Fatal(err)
		}
	}
	read := func(conn *websocket.Conn) *domainpb.AgentManagerWsMessage {
		t.Helper()
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		var message domainpb.AgentManagerWsMessage
		if err := protoconv.UnmarshalJSON(data, &message); err != nil {
			t.Fatal(err)
		}
		return &message
	}
	dial := func(testMode, queryMarker bool) *websocket.Conn {
		t.Helper()
		headers := http.Header{}
		if testMode && !queryMarker {
			headers.Set(apihttp.TestModeHeader, "1")
		}
		path := "/ws"
		if queryMarker {
			path += "?" + apihttp.TestModeWebSocketQuery + "=1"
		}
		conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+path, headers)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = conn.Close() })
		if read(conn).GetConnected() == nil {
			t.Fatal("missing welcome")
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"subscribe_all"}`)); err != nil {
			t.Fatal(err)
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ping"}`)); err != nil {
			t.Fatal(err)
		}
		if read(conn).GetPong() == nil {
			t.Fatal("missing subscription barrier")
		}
		return conn
	}
	request := func(path string, testMode bool) {
		t.Helper()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		if testMode {
			req.Header.Set(apihttp.TestModeHeader, "1")
		}
		response, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusNoContent {
			t.Fatalf("fixture publisher failed: status=%d", response.StatusCode)
		}
	}
	assertID := func(conn *websocket.Conn, want uuid.UUID) {
		t.Helper()
		message := read(conn)
		got := message.GetRunId()
		if got == "" && message.GetTaskStatus() != nil {
			got = message.GetTaskStatus().GetTaskId()
		}
		if got == "" && message.GetWorkflowLifecycle() != nil {
			got = message.GetWorkflowLifecycle().GetExecutionId()
		}
		if got != want.String() {
			t.Fatalf("cross-partition delivery: got=%s want=%s type=%s", got, want, message.Type)
		}
	}
	primary := dial(false, false)
	install()
	original := dial(true, false)
	request("/pin", true)
	originalContext := <-contexts
	for _, kind := range []string{"event", "run", "progress", "task", "workflow"} {
		testID, primaryID := uuid.New(), uuid.New()
		request("/publish?kind="+kind+"&id="+testID.String(), true)
		request("/publish?kind="+kind+"&id="+primaryID.String(), false)
		assertID(primary, primaryID)
		assertID(original, testID)
	}
	clear()
	install()
	replacement := dial(true, false)
	browserMarker := dial(true, true)
	testID, primaryID, oldBarrier := uuid.New(), uuid.New(), uuid.New()
	request("/publish?kind=run&id="+testID.String(), true)
	request("/publish?kind=run&id="+primaryID.String(), false)
	hub.BroadcastProgress(originalContext, oldBarrier, domain.RunPhaseCompleted, 100, "old partition barrier")
	assertID(replacement, testID)
	assertID(browserMarker, testID)
	assertID(primary, primaryID)
	assertID(original, oldBarrier)
	clear()
	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws?"+apihttp.TestModeWebSocketQuery+"=1", nil)
	if conn != nil {
		conn.Close()
	}
	if response != nil && response.Body != nil {
		response.Body.Close()
	}
	if err == nil || response == nil || response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("unleased test socket was admitted: response=%v err=%v", response, err)
	}
}
