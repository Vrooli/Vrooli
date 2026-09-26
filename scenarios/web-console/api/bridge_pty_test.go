package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"

	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/session"
	"web-console/internal/pty"
	"web-console/internal/ptyfake"
)

func TestBridgePTYUsesTheTypedPTYContract(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()
		writeFrame := func(frame *sessionv1.Frame) error {
			payload, marshalErr := proto.Marshal(frame)
			if marshalErr != nil {
				return marshalErr
			}
			return conn.WriteMessage(websocket.BinaryMessage, payload)
		}
		if err := writeFrame(&sessionv1.Frame{Payload: &sessionv1.Frame_Open{Open: &sessionv1.Open{SessionId: "s1"}}}); err != nil {
			t.Errorf("open: %v", err)
			return
		}
		if err := writeFrame(&sessionv1.Frame{Payload: &sessionv1.Frame_Data{Data: &sessionv1.Data{Data: []byte("abcdef")}}}); err != nil {
			t.Errorf("data: %v", err)
			return
		}
		_, payload, readErr := conn.ReadMessage()
		if readErr != nil {
			t.Errorf("read input: %v", readErr)
			return
		}
		var input sessionv1.Frame
		if err := (proto.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(payload, &input); err != nil {
			t.Errorf("decode input: %v", err)
			return
		}
		data := input.GetData()
		if data == nil || string(data.GetData()) != "ping" || data.GetSequence() != 0 {
			t.Errorf("input = %+v, want sequence 0/ping", data)
		}
		_ = writeFrame(&sessionv1.Frame{Payload: &sessionv1.Frame_Ack{Ack: &sessionv1.Ack{Accepted: true, Sequence: 0}}})
		_, _, _ = conn.ReadMessage()
	}))
	defer server.Close()

	p, err := bridgePTYFactory(pty.LaunchSpec{
		SessionID: "s1", Shell: "/bin/sh", Cols: 80, Rows: 24,
		RemoteURL: server.URL, RemoteNodeID: "node-1", RemoteOwnerToken: "owner", RemoteReauthToken: "reauth",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if err := p.ProbeReady(context.Background()); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 3)
	if n, err := p.Read(buf); err != nil || string(buf[:n]) != "abc" {
		t.Fatalf("first read = %q, %v", buf[:n], err)
	}
	if n, err := p.Read(buf); err != nil || string(buf[:n]) != "def" {
		t.Fatalf("second read = %q, %v", buf[:n], err)
	}
	if err := p.WriteInput([]byte("ping"), pty.KindKeystroke); err != nil {
		t.Fatal(err)
	}
	if err := p.SetSize(100, 40); err != nil {
		t.Fatal(err)
	}
	_ = p.Close()
}

// TestTerminalProtocolConformanceUsesOneHandlerForLocalAndBridgeBackends is
// intentionally table-driven: the browser-facing protocol must not acquire a
// second implementation when the PTY happens to be a Bridge transport.
func TestTerminalProtocolConformanceUsesOneHandlerForLocalAndBridgeBackends(t *testing.T) {
	cases := []struct {
		name        string
		factory     pty.Factory
		bridgeClose func()
	}{
		{name: "local", factory: ptyfake.NewFactory()},
	}

	bridgeURL, bridgeClose := newProtocolBridgeServer(t)
	cases = append(cases, struct {
		name        string
		factory     pty.Factory
		bridgeClose func()
	}{
		name: "bridged",
		factory: func(spec pty.LaunchSpec) (pty.PTY, error) {
			spec.RemoteURL = bridgeURL
			spec.RemoteNodeID = "node-1"
			spec.RemoteOwnerToken = "owner"
			return bridgePTYFactory(spec)
		},
		bridgeClose: bridgeClose,
	})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.bridgeClose != nil {
				t.Cleanup(tc.bridgeClose)
			}
			runSharedTerminalProtocolContract(t, tc.factory)
		})
	}
}

func newProtocolBridgeServer(t *testing.T) (string, func()) {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		writeFrame := func(frame *sessionv1.Frame) error {
			payload, err := proto.Marshal(frame)
			if err != nil {
				return err
			}
			return conn.WriteMessage(websocket.BinaryMessage, payload)
		}
		if err := writeFrame(&sessionv1.Frame{Payload: &sessionv1.Frame_Open{Open: &sessionv1.Open{SessionId: "bridge-session"}}}); err != nil {
			return
		}
		if err := writeFrame(&sessionv1.Frame{Payload: &sessionv1.Frame_Data{Data: &sessionv1.Data{Data: []byte("bridge\r\n")}}}); err != nil {
			return
		}
		for {
			_, payload, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var frame sessionv1.Frame
			if err := (proto.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(payload, &frame); err != nil {
				return
			}
			switch frame := frame.Payload.(type) {
			case *sessionv1.Frame_Data:
				_ = writeFrame(&sessionv1.Frame{Payload: &sessionv1.Frame_Ack{Ack: &sessionv1.Ack{Accepted: true, Sequence: frame.Data.GetSequence()}}})
			case *sessionv1.Frame_Close:
				return
			}
		}
	}))
	return server.URL, server.Close
}

func runSharedTerminalProtocolContract(t *testing.T, factory pty.Factory) {
	t.Helper()
	srv := newFakeTestServerWithFactory(factory)
	srv.setupRoutes()
	httpServer := httptest.NewServer(srv.router)
	t.Cleanup(httpServer.Close)
	t.Cleanup(srv.sessions.Shutdown)

	sess, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(httpServer, "/api/v1/sessions/"+sess.ID+"/ws"), nil)
	if err != nil {
		t.Fatalf("dial terminal: %v", err)
	}
	defer conn.Close()

	skipHistoryEnd(t, conn)
	if err := conn.WriteJSON(TerminalMessage{Type: MsgTypeStdin, Data: "ping", Offset: 0}); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	ack := readUntilType(t, conn, MsgTypeStdinAck)
	if !ack.Ok || ack.AcceptedThrough != 4 {
		t.Fatalf("stdin ack = %+v, want accepted prefix 4", ack)
	}

	if err := conn.WriteJSON(TerminalMessage{Type: "future_message_type"}); err != nil {
		t.Fatalf("write unknown message: %v", err)
	}
	if err := conn.WriteJSON(TerminalMessage{Type: MsgTypePing}); err != nil {
		t.Fatalf("write ping: %v", err)
	}
	if pong := readUntilType(t, conn, MsgTypePong); pong.Type != MsgTypePong {
		t.Fatalf("pong = %+v", pong)
	}
}
