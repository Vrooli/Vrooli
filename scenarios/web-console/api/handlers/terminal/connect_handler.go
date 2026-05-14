// Package terminal: Connect-RPC handler for the structured terminal
// surface (GetScreen, SendInput, WaitIdle). The legacy WS+upload paths
// remain on REST and are mounted alongside in module.go.
package terminal

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"

	terminalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/terminal"
)

func durationProto(d time.Duration) *durationpb.Duration {
	return durationpb.New(d)
}

// Service is the seam the Connect handler depends on. The concrete
// implementation lives in package main and adapts the SessionManager.
// All identifiers passed in are session IDs as produced by the
// SessionsService.
type Service interface {
	// GetScreen returns a structured deep-copy of the active screen for
	// the given session.
	GetScreen(ctx context.Context, sessionID string, includeScrollback bool) (ScreenView, error)
	// SendInput resolves SessionInput payload (text / keys / raw) and
	// writes the bytes to the PTY. Returns the number of bytes written.
	SendInput(ctx context.Context, sessionID string, in InputRequest) (int, error)
	// WaitIdle blocks until the session has produced no output for
	// quietWindow, or timeout elapses, or the session exits.
	WaitIdle(ctx context.Context, sessionID string, quietWindow, timeout time.Duration) (WaitIdleResult, error)
}

// Cell mirrors terminal.Cell in transport-neutral form.
type Cell struct {
	Rune rune
	SGR  SGR
}

// SGR mirrors terminal.SGR.
type SGR struct {
	FG, BG    uint32
	Bold      bool
	Italic    bool
	Underline bool
	Inverse   bool
	Faint     bool
}

// Cursor is the (col, row) cursor position; zero-based.
type Cursor struct {
	X, Y int
}

// ScreenView is the transport-neutral screen snapshot returned by
// GetScreen.
type ScreenView struct {
	Cols, Rows      int
	Cells           [][]Cell
	Cursor          Cursor
	InAltBuffer     bool
	ScrollbackLines int
	PlainText       string
}

// Key is one named key with optional modifiers.
type Key struct {
	Name  string
	Ctrl  bool
	Alt   bool
	Shift bool
}

// InputVariant tags which payload field is set on InputRequest.
type InputVariant uint8

const (
	InputVariantUnset InputVariant = iota
	InputVariantText
	InputVariantKeys
	InputVariantRaw
)

// InputRequest carries the discriminated input payload plus metadata.
type InputRequest struct {
	Variant InputVariant
	Text    string
	Keys    []Key
	Raw     []byte
	Source  string
	IsPaste bool
}

// WaitIdleReason enumerates how the WaitIdle wait ended.
type WaitIdleReason int

const (
	WaitIdleReasonUnknown WaitIdleReason = iota
	WaitIdleReasonIdle
	WaitIdleReasonTimeout
	WaitIdleReasonExited
)

// WaitIdleResult is what WaitIdle returns to the handler.
type WaitIdleResult struct {
	Reason WaitIdleReason
	Waited time.Duration
}

// Sentinel errors mapped via classify().
var (
	ErrNotFound           = errors.New("not found")
	ErrInvalidArgument    = errors.New("invalid argument")
	ErrFailedPrecondition = errors.New("failed precondition")
	ErrInternal           = errors.New("internal error")
)

// Deps wires the seams the Connect terminal handler needs.
type Deps struct {
	Service Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC handler implementing
// TerminalServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetScreen(ctx context.Context, req *connect.Request[terminalv1.GetScreenRequest]) (*connect.Response[terminalv1.GetScreenResponse], error) {
	id := strings.TrimSpace(req.Msg.GetSessionId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id is required"))
	}
	view, err := h.deps.Service.GetScreen(ctx, id, req.Msg.GetIncludeScrollback())
	if err != nil {
		return nil, h.classify(err, "terminal.GetScreen")
	}
	lines := make([]*terminalv1.Line, len(view.Cells))
	for y, row := range view.Cells {
		cells := make([]*terminalv1.Cell, len(row))
		for x, c := range row {
			cells[x] = &terminalv1.Cell{
				Rune: int32(c.Rune),
				Sgr:  sgrToProto(c.SGR),
			}
		}
		lines[y] = &terminalv1.Line{Cells: cells}
	}
	return connect.NewResponse(&terminalv1.GetScreenResponse{
		Lines:           lines,
		Cursor:          &terminalv1.Cursor{X: int32(view.Cursor.X), Y: int32(view.Cursor.Y)},
		Cols:            int32(view.Cols),
		Rows:            int32(view.Rows),
		InAltBuffer:     view.InAltBuffer,
		ScrollbackLines: int32(view.ScrollbackLines),
		PlainText:       view.PlainText,
	}), nil
}

func (h *connectHandler) SendInput(ctx context.Context, req *connect.Request[terminalv1.SendInputRequest]) (*connect.Response[terminalv1.SendInputResponse], error) {
	id := strings.TrimSpace(req.Msg.GetSessionId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id is required"))
	}
	in := InputRequest{
		Source:  req.Msg.GetSource(),
		IsPaste: req.Msg.GetIsPaste(),
	}
	switch body := req.Msg.GetBody().(type) {
	case *terminalv1.SendInputRequest_Text:
		in.Variant = InputVariantText
		in.Text = body.Text
	case *terminalv1.SendInputRequest_Keys:
		in.Variant = InputVariantKeys
		seq := body.Keys
		if seq != nil {
			for _, k := range seq.GetKeys() {
				in.Keys = append(in.Keys, Key{
					Name:  k.GetName(),
					Ctrl:  k.GetCtrl(),
					Alt:   k.GetAlt(),
					Shift: k.GetShift(),
				})
			}
		}
	case *terminalv1.SendInputRequest_Raw:
		in.Variant = InputVariantRaw
		in.Raw = body.Raw
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("body is required"))
	}
	n, err := h.deps.Service.SendInput(ctx, id, in)
	if err != nil {
		return nil, h.classify(err, "terminal.SendInput")
	}
	return connect.NewResponse(&terminalv1.SendInputResponse{BytesWritten: int32(n)}), nil
}

func (h *connectHandler) WaitIdle(ctx context.Context, req *connect.Request[terminalv1.WaitIdleRequest]) (*connect.Response[terminalv1.WaitIdleResponse], error) {
	id := strings.TrimSpace(req.Msg.GetSessionId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id is required"))
	}
	quiet := req.Msg.GetQuietWindow().AsDuration()
	timeout := req.Msg.GetTimeout().AsDuration()
	res, err := h.deps.Service.WaitIdle(ctx, id, quiet, timeout)
	if err != nil {
		return nil, h.classify(err, "terminal.WaitIdle")
	}
	var reason terminalv1.WaitIdleResponse_Reason
	switch res.Reason {
	case WaitIdleReasonIdle:
		reason = terminalv1.WaitIdleResponse_REASON_IDLE
	case WaitIdleReasonTimeout:
		reason = terminalv1.WaitIdleResponse_REASON_TIMEOUT
	case WaitIdleReasonExited:
		reason = terminalv1.WaitIdleResponse_REASON_EXITED
	default:
		reason = terminalv1.WaitIdleResponse_REASON_UNSPECIFIED
	}
	return connect.NewResponse(&terminalv1.WaitIdleResponse{
		Reason: reason,
		Waited: durationProto(res.Waited),
	}), nil
}

func (h *connectHandler) classify(err error, op string) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrInvalidArgument):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, ErrFailedPrecondition):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, ErrInternal):
		h.deps.Logger.Printf("%s: %v", op, err)
		return connect.NewError(connect.CodeInternal, err)
	default:
		h.deps.Logger.Printf("%s: %v", op, err)
		return connect.NewError(connect.CodeInternal, err)
	}
}

func sgrToProto(s SGR) *terminalv1.SGR {
	if s == (SGR{}) {
		return &terminalv1.SGR{}
	}
	return &terminalv1.SGR{
		Fg:        s.FG,
		Bg:        s.BG,
		Bold:      s.Bold,
		Italic:    s.Italic,
		Underline: s.Underline,
		Inverse:   s.Inverse,
		Faint:     s.Faint,
	}
}
