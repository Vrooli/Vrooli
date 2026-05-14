package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	terminalH "web-console/handlers/terminal"
	"web-console/session"
	"web-console/terminal"
)

// terminalAdapter implements terminalH.Service against the server's
// session.Manager. It resolves session IDs, converts SessionInput
// payloads, and forwards GetScreen / SendInput / WaitIdle to the
// underlying *session.Session.
type terminalAdapter struct {
	srv *Server
}

func newTerminalAdapter(s *Server) *terminalAdapter { return &terminalAdapter{srv: s} }

func (a *terminalAdapter) lookup(id string) (*session.Session, error) {
	sess, ok := a.srv.sessions.Get(id)
	if !ok {
		return nil, fmt.Errorf("session %q: %w", id, terminalH.ErrNotFound)
	}
	if sess.IsDead() {
		return nil, fmt.Errorf("session %q has exited: %w", id, terminalH.ErrFailedPrecondition)
	}
	return sess, nil
}

func (a *terminalAdapter) GetScreen(_ context.Context, sessionID string, includeScrollback bool) (terminalH.ScreenView, error) {
	sess, err := a.lookup(sessionID)
	if err != nil {
		return terminalH.ScreenView{}, err
	}
	view := sess.Screen()
	plain := sess.PlainText(includeScrollback)
	cells := make([][]terminalH.Cell, len(view.Cells))
	for y, row := range view.Cells {
		out := make([]terminalH.Cell, len(row))
		for x, c := range row {
			out[x] = terminalH.Cell{
				Rune: c.Rune,
				SGR:  sgrToHandler(c.SGR),
			}
		}
		cells[y] = out
	}
	return terminalH.ScreenView{
		Cols:            view.Cols,
		Rows:            view.Rows,
		Cells:           cells,
		Cursor:          terminalH.Cursor{X: view.Cursor.X, Y: view.Cursor.Y},
		InAltBuffer:     view.InAltBuffer,
		ScrollbackLines: view.ScrollbackLines,
		PlainText:       plain,
	}, nil
}

func (a *terminalAdapter) SendInput(_ context.Context, sessionID string, in terminalH.InputRequest) (int, error) {
	sess, err := a.lookup(sessionID)
	if err != nil {
		return 0, err
	}
	var input session.SessionInput
	switch in.Variant {
	case terminalH.InputVariantText:
		input = session.InputText(in.Text)
	case terminalH.InputVariantKeys:
		keys := make([]session.Key, len(in.Keys))
		for i, k := range in.Keys {
			keys[i] = session.Key{Name: k.Name, Ctrl: k.Ctrl, Alt: k.Alt, Shift: k.Shift}
		}
		input = session.InputKeys(keys...)
	case terminalH.InputVariantRaw:
		input = session.InputRaw(in.Raw)
	default:
		return 0, fmt.Errorf("body is required: %w", terminalH.ErrInvalidArgument)
	}
	if in.IsPaste {
		input = input.AsPaste()
	}
	source := in.Source
	if source == "" {
		source = "terminal-rpc"
	}
	input = input.WithSource(source)
	if err := sess.SendInput(input); err != nil {
		// Surface "unknown key" as InvalidArgument; everything else is
		// internal (PTY write failure, closed pipe, tmux send-keys
		// error, etc.).
		if isUnknownKeyError(err) {
			return 0, fmt.Errorf("%v: %w", err, terminalH.ErrInvalidArgument)
		}
		return 0, fmt.Errorf("%v: %w", err, terminalH.ErrInternal)
	}
	// We don't have visibility into the actual byte count without
	// reaching into resolveBytes; return 0 to signal "delivered, count
	// unspecified". A future revision can plumb the byte count through.
	return 0, nil
}

func (a *terminalAdapter) WaitIdle(ctx context.Context, sessionID string, quietWindow, timeout time.Duration) (terminalH.WaitIdleResult, error) {
	sess, err := a.lookup(sessionID)
	if err != nil {
		return terminalH.WaitIdleResult{}, err
	}
	reason, waited, werr := sess.WaitIdle(ctx, quietWindow, timeout)
	if werr != nil {
		return terminalH.WaitIdleResult{}, fmt.Errorf("%v: %w", werr, terminalH.ErrInternal)
	}
	var r terminalH.WaitIdleReason
	switch reason {
	case "idle":
		r = terminalH.WaitIdleReasonIdle
	case "timeout":
		r = terminalH.WaitIdleReasonTimeout
	case "exited":
		r = terminalH.WaitIdleReasonExited
	}
	return terminalH.WaitIdleResult{Reason: r, Waited: waited}, nil
}

func sgrToHandler(s terminal.SGR) terminalH.SGR {
	return terminalH.SGR{
		FG: s.FG, BG: s.BG,
		Bold: s.Bold, Italic: s.Italic, Underline: s.Underline,
		Inverse: s.Inverse, Faint: s.Faint,
	}
}

// isUnknownKeyError detects the session.SendInput error for unrecognized
// key names. The string prefix is the contract; if it changes,
// terminal_adapter must change too.
func isUnknownKeyError(err error) bool {
	if err == nil {
		return false
	}
	var unknown interface{ Error() string }
	if !errors.As(err, &unknown) {
		return false
	}
	msg := unknown.Error()
	const prefix = "unknown key "
	return len(msg) >= len(prefix) && msg[:len(prefix)] == prefix
}
