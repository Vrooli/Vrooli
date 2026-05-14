package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"

	terminalH "web-console/handlers/terminal"

	terminalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/terminal"
)

// newTerminalConnectHandler builds the Connect handler against a server
// with pipe-backed fake PTYs, mirroring the sessions test helper.
func newTerminalConnectHandler(srv *Server) *terminalConnectHandlerHelper {
	return &terminalConnectHandlerHelper{
		Handler: terminalH.NewConnectHandler(terminalH.Deps{
			Service: newTerminalAdapter(srv),
		}),
	}
}

type terminalConnectHandlerHelper struct {
	Handler interface {
		GetScreen(context.Context, *connect.Request[terminalv1.GetScreenRequest]) (*connect.Response[terminalv1.GetScreenResponse], error)
		SendInput(context.Context, *connect.Request[terminalv1.SendInputRequest]) (*connect.Response[terminalv1.SendInputResponse], error)
		WaitIdle(context.Context, *connect.Request[terminalv1.WaitIdleRequest]) (*connect.Response[terminalv1.WaitIdleResponse], error)
	}
}

func TestTerminalConnect_GetScreen_UnknownSession(t *testing.T) {
	srv := newFakeTestServer()
	h := newTerminalConnectHandler(srv)
	_, err := h.Handler.GetScreen(context.Background(),
		connect.NewRequest(&terminalv1.GetScreenRequest{SessionId: "no-such-session"}))
	if connectCode(err) != connect.CodeNotFound {
		t.Errorf("expected NotFound, got %v (err=%v)", connectCode(err), err)
	}
}

func TestTerminalConnect_SendInput_RoundTrip(t *testing.T) {
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(sess.ID) }()

	h := newTerminalConnectHandler(srv)
	resp, err := h.Handler.SendInput(context.Background(),
		connect.NewRequest(&terminalv1.SendInputRequest{
			SessionId: sess.ID,
			Body:      &terminalv1.SendInputRequest_Text{Text: "echo hi\n"},
			Source:    "test",
		}))
	if err != nil {
		t.Fatalf("SendInput: %v", err)
	}
	if resp == nil {
		t.Fatal("nil response")
	}
}

func TestTerminalConnect_SendInput_UnknownKey(t *testing.T) {
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(sess.ID) }()

	h := newTerminalConnectHandler(srv)
	_, err = h.Handler.SendInput(context.Background(),
		connect.NewRequest(&terminalv1.SendInputRequest{
			SessionId: sess.ID,
			Body: &terminalv1.SendInputRequest_Keys{Keys: &terminalv1.KeySequence{
				Keys: []*terminalv1.Key{{Name: "NotAKey"}},
			}},
		}))
	if connectCode(err) != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument for unknown key, got %v (err=%v)", connectCode(err), err)
	}
}

func TestTerminalConnect_WaitIdle_BecomesIdle(t *testing.T) {
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(sess.ID) }()

	h := newTerminalConnectHandler(srv)
	resp, err := h.Handler.WaitIdle(context.Background(),
		connect.NewRequest(&terminalv1.WaitIdleRequest{
			SessionId:   sess.ID,
			QuietWindow: durationpb.New(100 * time.Millisecond),
			Timeout:     durationpb.New(2 * time.Second),
		}))
	if err != nil {
		t.Fatalf("WaitIdle: %v", err)
	}
	if got := resp.Msg.GetReason(); got != terminalv1.WaitIdleResponse_REASON_IDLE {
		t.Errorf("reason: got %v, want IDLE", got)
	}
}

func TestTerminalConnect_GetScreen_PlainText(t *testing.T) {
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(sess.ID) }()

	h := newTerminalConnectHandler(srv)
	resp, err := h.Handler.GetScreen(context.Background(),
		connect.NewRequest(&terminalv1.GetScreenRequest{SessionId: sess.ID}))
	if err != nil {
		t.Fatalf("GetScreen: %v", err)
	}
	if resp.Msg.GetCols() != 80 || resp.Msg.GetRows() != 24 {
		t.Errorf("dims: got %dx%d, want 80x24", resp.Msg.GetCols(), resp.Msg.GetRows())
	}
	if len(resp.Msg.GetLines()) != 24 {
		t.Errorf("lines: got %d, want 24", len(resp.Msg.GetLines()))
	}
	// Empty session: plain text should be empty or all-whitespace.
	if strings.TrimSpace(resp.Msg.GetPlainText()) != "" {
		t.Errorf("fresh session plain_text should be empty, got %q", resp.Msg.GetPlainText())
	}
}
