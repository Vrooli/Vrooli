package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/session"
	"google.golang.org/protobuf/proto"
)

func TestConfiguredRemoteTargetFailsClosedWithoutAllCredentials(t *testing.T) {
	t.Setenv("WEB_CONSOLE_BRIDGE_URL", "http://bridge.test")
	t.Setenv("WEB_CONSOLE_BRIDGE_NODE_ID", "node-1")
	t.Setenv("WEB_CONSOLE_BRIDGE_OWNER_TOKEN", "owner")
	t.Setenv("WEB_CONSOLE_BRIDGE_REAUTH_TOKEN", "")
	target := configuredRemoteTarget()
	if target.Available {
		t.Fatal("target became available without re-authentication proof")
	}
	if target.FailureRung == "" {
		t.Fatal("unavailable target did not explain its failure rung")
	}
}

func TestTargetFromRegistryNodeUsesDispatchabilityAndReadinessRung(t *testing.T) {
	base := remoteTerminalTarget{BaseURL: "http://bridge.test", OwnerToken: "owner", ReauthToken: "reauth"}
	ready := targetFromRegistryNode(base, &registryv1.Node{
		Id:                    "node-ready",
		Name:                  "Swarminator",
		Kind:                  registryv1.NodeKind_NODE_KIND_AGENT,
		RegistryRecordPresent: true,
		HeartbeatFresh:        true,
		ChannelHeld:           true,
		ProtocolCompatible:    true,
		Dispatchable:          true,
	})
	if !ready.Available || ready.ID != "bridge-node:node-ready" || len(ready.Readiness) != 5 {
		t.Fatalf("ready target was not projected correctly: %+v", ready)
	}

	offline := targetFromRegistryNode(base, &registryv1.Node{
		Id:                    "node-offline",
		Name:                  "Offline host",
		Kind:                  registryv1.NodeKind_NODE_KIND_AGENT,
		RegistryRecordPresent: true,
		HeartbeatFresh:        false,
		ChannelHeld:           false,
		ProtocolCompatible:    true,
	})
	if offline.Available || offline.FailureRung != "heartbeat freshness" {
		t.Fatalf("offline target did not fail closed at first rung: %+v", offline)
	}
}

func TestRemoteTerminalProxyTranslatesBrowserInputAndBridgeOutput(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer owner" || r.Header.Get("X-Bridge-Owner-Reauth") != "reauth" {
			http.Error(w, "missing server-side credentials", http.StatusUnauthorized)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		open, _ := proto.Marshal(&sessionv1.Frame{Payload: &sessionv1.Frame_Open{Open: &sessionv1.Open{SessionId: r.URL.Query().Get("session_id")}}})
		_ = conn.WriteMessage(websocket.BinaryMessage, open)
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var incoming sessionv1.Frame
		if proto.Unmarshal(raw, &incoming) != nil || incoming.GetData().GetSequence() != 0 || string(incoming.GetData().GetData()) != "hello" {
			return
		}
		ack, _ := proto.Marshal(&sessionv1.Frame{Payload: &sessionv1.Frame_Ack{Ack: &sessionv1.Ack{Accepted: true, Sequence: 0}}})
		_ = conn.WriteMessage(websocket.BinaryMessage, ack)
		out, _ := proto.Marshal(&sessionv1.Frame{Payload: &sessionv1.Frame_Data{Data: &sessionv1.Data{Sequence: 0, Data: []byte("remote output")}}})
		_ = conn.WriteMessage(websocket.BinaryMessage, out)
		<-time.After(50 * time.Millisecond)
	}))
	defer bridge.Close()

	targetURL := "http://" + strings.TrimPrefix(bridge.URL, "http://")
	srv := &Server{remoteSessions: &remoteTerminalRegistry{sessions: map[string]remoteTerminalSession{
		"remote:s1": {ID: "remote:s1", Target: remoteTerminalTarget{BaseURL: targetURL, NodeID: "node-1", OwnerToken: "owner", ReauthToken: "reauth"}, Cols: 80, Rows: 24},
	}}}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.handleTerminalWS(w, mux.SetURLVars(r, map[string]string{"id": "remote:s1"}))
	})
	web := httptest.NewServer(h)
	defer web.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws://"+strings.TrimPrefix(web.URL, "http://"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	// Consume the federated history/size/ready handshake.
	for i := 0; i < 3; i++ {
		if _, _, err := conn.ReadMessage(); err != nil {
			t.Fatal(err)
		}
	}
	if err := conn.WriteJSON(TerminalMessage{Type: MsgTypeStdin, Data: "hello", Seq: 1}); err != nil {
		t.Fatal(err)
	}
	_, ackRaw, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	var ack TerminalMessage
	if err := json.Unmarshal(ackRaw, &ack); err != nil {
		t.Fatal(err)
	}
	if ack.Type != MsgTypeStdinAck || ack.Seq != 1 || !ack.Ok {
		t.Fatalf("unexpected browser ack: %+v", ack)
	}
}

func TestRemoteTerminalProxySurfacesUnpairedBridgeRejection(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		open, _ := proto.Marshal(&sessionv1.Frame{Payload: &sessionv1.Frame_Open{Open: &sessionv1.Open{SessionId: r.URL.Query().Get("session_id")}}})
		_ = conn.WriteMessage(websocket.BinaryMessage, open)
		reject, _ := proto.Marshal(&sessionv1.Frame{Payload: &sessionv1.Frame_Ack{Ack: &sessionv1.Ack{Code: "node_unavailable", Reason: "node channel unavailable"}}})
		_, _, _ = conn.ReadMessage()
		_ = conn.WriteMessage(websocket.BinaryMessage, reject)
		<-time.After(50 * time.Millisecond)
	}))
	defer bridge.Close()

	srv := &Server{remoteSessions: &remoteTerminalRegistry{sessions: map[string]remoteTerminalSession{
		"remote:rejected": {ID: "remote:rejected", LaunchCommand: "printf ready", Target: remoteTerminalTarget{BaseURL: "http://" + strings.TrimPrefix(bridge.URL, "http://"), NodeID: "node-1", OwnerToken: "owner", ReauthToken: "reauth"}, Cols: 80, Rows: 24},
	}}}
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.handleTerminalWS(w, mux.SetURLVars(r, map[string]string{"id": "remote:rejected"}))
	}))
	defer web.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws://"+strings.TrimPrefix(web.URL, "http://"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	for i := 0; i < 3; i++ {
		if _, _, err := conn.ReadMessage(); err != nil {
			t.Fatal(err)
		}
	}
	var msg TerminalMessage
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatal(err)
	}
	if msg.Type != MsgTypeError || msg.Data != "node_unavailable" {
		t.Fatalf("unexpected rejection message: %+v", msg)
	}
}

func TestRemoteTerminalProxySurfacesBridgeCloseReason(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		open, _ := proto.Marshal(&sessionv1.Frame{Payload: &sessionv1.Frame_Open{Open: &sessionv1.Open{SessionId: r.URL.Query().Get("session_id")}}})
		_ = conn.WriteMessage(websocket.BinaryMessage, open)
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "node_channel_lost"), time.Now().Add(time.Second))
	}))
	defer bridge.Close()

	srv := &Server{remoteSessions: &remoteTerminalRegistry{sessions: map[string]remoteTerminalSession{
		"remote:closed": {ID: "remote:closed", Target: remoteTerminalTarget{BaseURL: "http://" + strings.TrimPrefix(bridge.URL, "http://"), NodeID: "node-1", OwnerToken: "owner", ReauthToken: "reauth"}, Cols: 80, Rows: 24},
	}}}
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.handleTerminalWS(w, mux.SetURLVars(r, map[string]string{"id": "remote:closed"}))
	}))
	defer web.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws://"+strings.TrimPrefix(web.URL, "http://"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	for i := 0; i < 3; i++ {
		if _, _, err := conn.ReadMessage(); err != nil {
			t.Fatal(err)
		}
	}
	var msg TerminalMessage
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatal(err)
	}
	if msg.Type != MsgTypeExit || msg.Data != "bridge_session_closed" {
		t.Fatalf("unexpected close message: %+v", msg)
	}
}

func TestRemoteTerminalCreateStoresServerSideSessionRecord(t *testing.T) {
	t.Setenv("WEB_CONSOLE_BRIDGE_URL", "http://bridge.test")
	t.Setenv("WEB_CONSOLE_BRIDGE_NODE_ID", "node-1")
	t.Setenv("WEB_CONSOLE_BRIDGE_OWNER_TOKEN", "owner")
	t.Setenv("WEB_CONSOLE_BRIDGE_REAUTH_TOKEN", "reauth")
	target := configuredRemoteTarget()
	srv := &Server{
		remoteSessions: &remoteTerminalRegistry{sessions: make(map[string]remoteTerminalSession)},
		remoteTargetCatalog: func() []remoteTerminalTarget {
			return []remoteTerminalTarget{target}
		},
	}
	req := httptest.NewRequest("POST", "/api/v1/remote-sessions", strings.NewReader(`{"target_id":"`+target.ID+`","launch_command":"printf ready"}`))
	resp := httptest.NewRecorder()
	srv.handleRemoteTerminalCreate(resp, req)
	if resp.Code != 201 {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	var body struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(body.ID, "remote:") {
		t.Fatalf("id=%q is not a remote session id", body.ID)
	}
	stored, ok := srv.remoteSessions.get(body.ID)
	if !ok || stored.LaunchCommand != "printf ready" {
		t.Fatalf("server-side launch command was not retained: %+v", stored)
	}
}
