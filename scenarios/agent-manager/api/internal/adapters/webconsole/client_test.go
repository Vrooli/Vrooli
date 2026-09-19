package webconsole

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	sessionsv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions/sessions_v1connect"
	terminalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/terminal"
	terminalv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/terminal/terminal_v1connect"
)

type sessionTestServer struct {
	sessionsv1connect.UnimplementedSessionsServiceHandler
	created *sessionsv1.CreateRequest
	deleted string
	missing bool
}

func (s *sessionTestServer) Create(_ context.Context, req *connect.Request[sessionsv1.CreateRequest]) (*connect.Response[sessionsv1.CreateResponse], error) {
	s.created = req.Msg
	return connect.NewResponse(&sessionsv1.CreateResponse{Session: &sessionsv1.Session{Id: "session-1", Owner: OwnerAgentManager, Backend: req.Msg.Backend, Origin: sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC, DisplayLabel: req.Msg.DisplayLabel}}), nil
}

func (s *sessionTestServer) Get(_ context.Context, req *connect.Request[sessionsv1.GetRequest]) (*connect.Response[sessionsv1.GetResponse], error) {
	if s.missing {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&sessionsv1.GetResponse{Session: &sessionsv1.Session{Id: req.Msg.Id, Owner: OwnerAgentManager, Backend: "persistent", Origin: sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC, DisplayLabel: "drill"}}), nil
}

func (s *sessionTestServer) Delete(_ context.Context, req *connect.Request[sessionsv1.DeleteRequest]) (*connect.Response[sessionsv1.DeleteResponse], error) {
	s.deleted = req.Msg.Id
	if s.missing {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&sessionsv1.DeleteResponse{}), nil
}

type terminalTestServer struct {
	terminalv1connect.UnimplementedTerminalServiceHandler
	inputs []*terminalv1.SendInputRequest
}

func (s *terminalTestServer) SendInput(_ context.Context, req *connect.Request[terminalv1.SendInputRequest]) (*connect.Response[terminalv1.SendInputResponse], error) {
	s.inputs = append(s.inputs, req.Msg)
	return connect.NewResponse(&terminalv1.SendInputResponse{}), nil
}

func (s *terminalTestServer) GetScreen(_ context.Context, req *connect.Request[terminalv1.GetScreenRequest]) (*connect.Response[terminalv1.GetScreenResponse], error) {
	return connect.NewResponse(&terminalv1.GetScreenResponse{PlainText: "agent ready"}), nil
}

func newWebConsoleTestClient(t *testing.T, sessions *sessionTestServer, terminal *terminalTestServer) *Client {
	t.Helper()
	mux := http.NewServeMux()
	sessionsPath, sessionsHandler := sessionsv1connect.NewSessionsServiceHandler(sessions)
	terminalPath, terminalHandler := terminalv1connect.NewTerminalServiceHandler(terminal)
	mux.Handle(sessionsPath, sessionsHandler)
	mux.Handle(terminalPath, terminalHandler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return NewClient(server.URL, server.Client())
}

func TestClientMaintainsInteractiveSessionContract(t *testing.T) {
	sessions := &sessionTestServer{}
	terminal := &terminalTestServer{}
	client := newWebConsoleTestClient(t, sessions, terminal)
	ctx := context.Background()

	id, err := client.CreateSession(ctx, CreateSessionParams{LaunchCommand: "codex", Execute: true, DisplayLabel: "drill"})
	if err != nil || id != "session-1" || sessions.created.GetCols() != 120 || sessions.created.GetRows() != 40 || sessions.created.GetBackend() != "persistent" || !sessions.created.GetExecuteLaunchCommand() {
		t.Fatalf("create id=%q req=%+v err=%v", id, sessions.created, err)
	}
	info, err := client.GetSession(ctx, id)
	if err != nil || info.ID != id || info.Owner != OwnerAgentManager || info.Origin != "SESSION_ORIGIN_PROGRAMMATIC" {
		t.Fatalf("info=%+v err=%v", info, err)
	}
	if err := client.SendText(ctx, id, "continue", "agent-manager"); err != nil {
		t.Fatal(err)
	}
	if err := client.Interrupt(ctx, id, "agent-manager"); err != nil {
		t.Fatal(err)
	}
	if screen, err := client.Screen(ctx, id, true); err != nil || screen != "agent ready" {
		t.Fatalf("screen=%q err=%v", screen, err)
	}
	if len(terminal.inputs) != 2 || terminal.inputs[0].GetText() != "continue" || len(terminal.inputs[1].GetKeys().GetKeys()) != 2 {
		t.Fatalf("inputs=%+v", terminal.inputs)
	}
	if err := client.DeleteSession(ctx, id); err != nil || sessions.deleted != id {
		t.Fatalf("delete=%q err=%v", sessions.deleted, err)
	}

	sessions.missing = true
	if _, err := client.GetSession(ctx, id); err != ErrSessionNotFound {
		t.Fatalf("missing get err=%v", err)
	}
	if err := client.DeleteSession(ctx, id); err != nil {
		t.Fatalf("missing delete err=%v", err)
	}
}
