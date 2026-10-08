package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestRunEventsFollowInitializesWebSocketRequest(t *testing.T) {
	for _, override := range []bool{false, true} {
		name := "saved_base"
		if override {
			name = "api_base_override"
		}
		t.Run(name, func(t *testing.T) {
			const runID = "fixture-run"
			t.Setenv("AM_WS_TEST_CONFIG", t.TempDir())
			t.Setenv("AM_WS_TEST_TOKEN", "")
			t.Setenv(cliutil.EnvIdentityToken, "fixture-run-identity")
			var requests atomic.Int32
			closed := make(chan struct{})
			var frames []string
			for _, message := range eventStreamFixture(runID) {
				// Match the API owner's protoconv.JSONMarshalOptions.
				data, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(message)
				if err != nil {
					t.Fatal(err)
				}
				frames = append(frames, string(data))
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodGet || r.URL.Path != "/remote/api/v1/ws" || !websocket.IsWebSocketUpgrade(r) {
					t.Errorf("unexpected request before WebSocket: %s %s", r.Method, r.URL.Path)
					http.Error(w, "WebSocket required", http.StatusBadRequest)
					return
				}
				for header, want := range map[string]string{
					"Authorization":                  "Bearer fixture-saved-token",
					cliutil.HeaderAgentIdentityToken: "fixture-run-identity",
					cliutil.HeaderInvocationScenario: "agent-manager",
					cliutil.HeaderInvocationCommand:  "run events",
					"X-Fixture-Header":               "retained",
				} {
					if r.Header.Get(header) != want {
						t.Errorf("WebSocket handshake did not preserve %s", header)
						http.Error(w, "missing request identity", http.StatusUnauthorized)
						return
					}
				}
				if r.Header.Get(cliutil.HeaderInvocationID) == "" {
					t.Error("WebSocket handshake omitted invocation ID")
				}
				conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
				if err != nil {
					t.Errorf("upgrade: %v", err)
					return
				}
				defer close(closed)
				defer conn.Close()
				_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				_, data, err := conn.ReadMessage()
				if err != nil {
					t.Errorf("read subscription: %v", err)
					return
				}
				var subscription domainpb.AgentManagerWsClientMessage
				if err := protojson.Unmarshal(data, &subscription); err != nil || subscription.Type != domainpb.AgentManagerWsClientMessageType_AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_SUBSCRIBE || subscription.GetRunSubscription().GetRunId() != runID {
					t.Errorf("subscription = %v, err = %v", &subscription, err)
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, []byte(strings.Join(frames, "\n"))); err != nil {
					t.Errorf("send protobuf events: %v", err)
					return
				}
				if err := conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "fixture complete"), time.Now().Add(5*time.Second)); err != nil {
					t.Errorf("close stream: %v", err)
					return
				}
				if _, _, err := conn.ReadMessage(); !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					t.Errorf("client close response = %v", err)
				}
			}))
			defer server.Close()

			opts := cliapp.ScenarioOptions{
				Name: "agent-manager", ConfigDirEnvVars: []string{"AM_WS_TEST_CONFIG"},
				TokenEnvVars: []string{"AM_WS_TEST_TOKEN"}, AllowAnonymous: true,
			}
			core, err := cliapp.NewScenarioApp(opts)
			if err != nil {
				t.Fatal(err)
			}
			core.Config.APIBase = server.URL + "/remote/"
			if override {
				core.Config.APIBase = server.URL + "/wrong"
			}
			core.Config.Token = "fixture-saved-token"
			if err := core.SaveConfig(); err != nil {
				t.Fatal(err)
			}
			configPath := filepath.Join(os.Getenv("AM_WS_TEST_CONFIG"), "config.json")
			before, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatal(err)
			}
			// Reload the saved configuration in a fresh client. The fixture
			// command omits the health probe to exclude any warm-up HTTP call.
			core, err = cliapp.NewScenarioApp(opts)
			if err != nil {
				t.Fatal(err)
			}
			app := &App{core: core}
			core.HTTPClient.SetHeaderSource(func() map[string]string {
				return map[string]string{"X-Fixture-Header": "retained"}
			})
			core.SetCommandsWithSubgroups(nil, []cliapp.SubcommandGroup{{
				Name: "run", Subcommands: []cliapp.Command{{Name: "events", Run: app.runEvents}},
			}})
			args := []string{"run", "events", runID, "--follow"}
			if override {
				args = append([]string{"--api-base", server.URL + "/remote/"}, args...)
			}
			if got := core.HTTPClient.BaseURL(); got != "" {
				t.Fatalf("HTTP client already initialized: %q", got)
			}
			output := captureStdout(t, func() error { return app.Run(args) })
			for _, want := range []string{"EVENT #9007199254740993 message", "pilot observed", "PROGRESS 45% | Phase: executing | checking receipts", "STATUS: running", "STATUS: complete"} {
				if !strings.Contains(output, want) {
					t.Errorf("stream omitted %q: %s", want, output)
				}
			}
			if strings.Contains(output, "foreign message") {
				t.Error("stream displayed a different run")
			}
			select {
			case <-closed:
			case <-time.After(5 * time.Second):
				t.Fatal("WebSocket handler did not finish closing")
			}
			if requests.Load() != 1 {
				t.Fatalf("requests = %d, want only the WebSocket handshake", requests.Load())
			}
			after, err := os.ReadFile(configPath)
			if err != nil || string(after) != string(before) {
				t.Fatal("stream changed the saved API configuration")
			}
		})
	}
}

func eventStreamFixture(runID string) []*domainpb.AgentManagerWsMessage {
	return []*domainpb.AgentManagerWsMessage{
		{Type: domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_CONNECTED, Payload: &domainpb.AgentManagerWsMessage_Connected{Connected: &domainpb.WsConnected{Message: "connected"}}},
		{Type: domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT, RunId: &runID, Payload: &domainpb.AgentManagerWsMessage_RunEvent{RunEvent: &domainpb.RunEvent{
			RunId: runID, Sequence: 9007199254740993, EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE,
			Data: &domainpb.RunEvent_Message{Message: &domainpb.MessageEventData{Role: "assistant", Content: "pilot observed"}},
		}}},
		{Type: domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_PROGRESS, RunId: &runID, Payload: &domainpb.AgentManagerWsMessage_RunProgress{RunProgress: &domainpb.ProgressEventData{
			Phase: domainpb.RunPhase_RUN_PHASE_EXECUTING, PercentComplete: 45, CurrentAction: "checking receipts",
		}}},
		{Type: domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS, RunId: &runID, Payload: &domainpb.AgentManagerWsMessage_RunStatus{RunStatus: &domainpb.RunStatusUpdate{RunId: runID, Status: domainpb.RunStatus_RUN_STATUS_RUNNING}}},
		{Type: domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT, Payload: &domainpb.AgentManagerWsMessage_RunEvent{RunEvent: &domainpb.RunEvent{
			RunId: "other", Sequence: 2, EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE,
			Data: &domainpb.RunEvent_Message{Message: &domainpb.MessageEventData{Content: "foreign message"}},
		}}},
		{Type: domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS, RunId: &runID, Payload: &domainpb.AgentManagerWsMessage_RunStatus{RunStatus: &domainpb.RunStatusUpdate{RunId: runID, Status: domainpb.RunStatus_RUN_STATUS_COMPLETE}}},
	}
}

func TestHTTPToWSURLPreservesEndpointAndMapsSupportedSchemes(t *testing.T) {
	cases := map[string]string{
		"http://example.test/api/v1/ws?token=x": "ws://example.test/api/v1/ws?token=x",
		"https://example.test/api/v1/ws":        "wss://example.test/api/v1/ws",
		"ws://example.test/api/v1/ws":           "ws://example.test/api/v1/ws",
		"wss://example.test/api/v1/ws":          "wss://example.test/api/v1/ws",
	}
	for in, want := range cases {
		got, err := httpToWSURL(in)
		if err != nil || got != want {
			t.Fatalf("httpToWSURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := httpToWSURL("ftp://example.test/events"); err == nil {
		t.Fatal("unsupported scheme unexpectedly succeeded")
	}
}

func TestHandleWSMessageFiltersOtherRunsAndRejectsMalformedKnownPayloads(t *testing.T) {
	app := &App{}
	other := "other"
	if err := app.handleWSMessage(&domainpb.AgentManagerWsMessage{Type: domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT, RunId: &other}, "target"); err != nil {
		t.Fatalf("message for another run should be ignored: %v", err)
	}
	for _, kind := range []domainpb.AgentManagerWsMessageType{
		domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT,
		domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_PROGRESS,
		domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS,
	} {
		if err := app.handleWSMessage(&domainpb.AgentManagerWsMessage{Type: kind}, "target"); err == nil {
			t.Fatalf("malformed %s payload unexpectedly succeeded", kind)
		}
	}
	for _, kind := range []domainpb.AgentManagerWsMessageType{domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_CONNECTED, domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_PONG} {
		if err := app.handleWSMessage(&domainpb.AgentManagerWsMessage{Type: kind}, "target"); err != nil {
			t.Fatalf("%s = %v", kind, err)
		}
	}
}

func TestHandleWSMessageDisplaysEachSupportedPayloadShape(t *testing.T) {
	app := &App{}
	for _, msg := range eventStreamFixture("target") {
		if err := app.handleWSMessage(msg, "target"); err != nil {
			t.Fatalf("handleWSMessage(%s): %v", msg.Type, err)
		}
	}
}

func TestReadWSMessagesCancellationUnblocksDelivery(t *testing.T) {
	data, err := protojson.Marshal(eventStreamFixture("target")[1])
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	read := make(chan struct{})
	done := make(chan error, 1)
	messages := make(chan *domainpb.AgentManagerWsMessage) // Deliberately no consumer.
	go func() {
		done <- readWSMessages(ctx, func() (int, []byte, error) {
			close(read)
			return websocket.TextMessage, data, nil
		}, messages)
	}()
	<-read
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("cancelled reader: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("reader stayed blocked delivering to an interrupted stream")
	}
}

func TestReadWSMessagesReportsMalformedEnvelopeAndAbnormalClose(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
		err  error
		want string
	}{
		{name: "malformed", data: []byte(`{`), want: "failed to decode WebSocket message"},
		{name: "abnormal_close", err: &websocket.CloseError{Code: websocket.CloseAbnormalClosure}, want: "connection closed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := readWSMessages(context.Background(), func() (int, []byte, error) {
				return websocket.TextMessage, tc.data, tc.err
			}, make(chan *domainpb.AgentManagerWsMessage))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}
