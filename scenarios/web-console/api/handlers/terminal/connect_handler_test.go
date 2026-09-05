package terminal

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	terminalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/terminal"
	"google.golang.org/protobuf/types/known/durationpb"
)

type fakeTerminalService struct {
	err  error
	last InputRequest
}

func (f *fakeTerminalService) GetScreen(context.Context, string, bool) (ScreenView, error) {
	return ScreenView{Cols: 2, Rows: 1, Cells: [][]Cell{{{Rune: 'x', SGR: SGR{FG: 1, Bold: true}}}}, Cursor: Cursor{X: 1}, PlainText: "x"}, f.err
}

func (f *fakeTerminalService) SendInput(_ context.Context, _ string, in InputRequest) (int, error) {
	f.last = in
	return 3, f.err
}

func (f *fakeTerminalService) WaitIdle(context.Context, string, time.Duration, time.Duration) (WaitIdleResult, error) {
	return WaitIdleResult{Reason: WaitIdleReasonIdle, Waited: time.Second}, nil
}

func TestConnectHandlerTerminalOperations(t *testing.T) {
	svc := &fakeTerminalService{}
	h := NewConnectHandler(Deps{Service: svc})
	ctx := context.Background()
	if resp, err := h.GetScreen(ctx, connect.NewRequest(&terminalv1.GetScreenRequest{SessionId: "s", IncludeScrollback: true})); err != nil || resp.Msg.Cols != 2 || resp.Msg.Lines[0].Cells[0].Sgr.Bold != true {
		t.Fatalf("screen: %#v %v", resp, err)
	}
	for _, req := range []*terminalv1.SendInputRequest{
		{SessionId: "s", Body: &terminalv1.SendInputRequest_Text{Text: "text"}, Source: "test", IsPaste: true},
		{SessionId: "s", Body: &terminalv1.SendInputRequest_Keys{Keys: &terminalv1.KeySequence{Keys: []*terminalv1.Key{{Name: "enter", Ctrl: true}}}}, Source: "test", IsPaste: true},
		{SessionId: "s", Body: &terminalv1.SendInputRequest_Raw{Raw: []byte{1}}, Source: "test", IsPaste: true},
	} {
		resp, err := h.SendInput(ctx, connect.NewRequest(req))
		if err != nil || resp.Msg.BytesWritten != 3 {
			t.Fatalf("input: %#v %v", resp, err)
		}
	}
	wait, err := h.WaitIdle(ctx, connect.NewRequest(&terminalv1.WaitIdleRequest{SessionId: "s", QuietWindow: durationpb.New(time.Second), Timeout: durationpb.New(2 * time.Second)}))
	if err != nil || wait.Msg.Reason != terminalv1.WaitIdleResponse_REASON_IDLE || wait.Msg.Waited.AsDuration() != time.Second {
		t.Fatalf("wait: %#v %v", wait, err)
	}
}

func TestConnectHandlerTerminalValidationAndClassification(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &fakeTerminalService{}})
	ctx := context.Background()
	for _, call := range []func() error{
		func() error { _, e := h.GetScreen(ctx, connect.NewRequest(&terminalv1.GetScreenRequest{})); return e },
		func() error {
			_, e := h.SendInput(ctx, connect.NewRequest(&terminalv1.SendInputRequest{SessionId: "s"}))
			return e
		},
		func() error { _, e := h.WaitIdle(ctx, connect.NewRequest(&terminalv1.WaitIdleRequest{})); return e },
	} {
		if err := call(); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("got %v", err)
		}
	}
	for _, in := range []error{ErrNotFound, ErrInvalidArgument, ErrFailedPrecondition, ErrInternal, errors.New("x")} {
		if h.classify(in, "test") == nil {
			t.Fatal("classification returned nil")
		}
	}
	for _, reason := range []WaitIdleReason{WaitIdleReasonIdle, WaitIdleReasonTimeout, WaitIdleReasonExited, WaitIdleReasonUnknown} {
		_ = reason
	}
}
