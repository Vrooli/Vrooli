package terminal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"web-console/session"
	"web-console/terminal"
)

// SessionManager is the slice of session.Manager the Adapter depends on.
type SessionManager interface {
	Get(id string) (*session.Session, bool)
}

// Adapter is the production Service implementation. Constructed in
// api/main.go with a typed *session.Manager (no *Server import) and
// passed to Module.
type Adapter struct {
	Manager SessionManager
}

func (a *Adapter) lookup(id string) (*session.Session, error) {
	sess, ok := a.Manager.Get(id)
	if !ok {
		return nil, fmt.Errorf("session %q: %w", id, ErrNotFound)
	}
	if sess.IsDead() {
		return nil, fmt.Errorf("session %q has exited: %w", id, ErrFailedPrecondition)
	}
	return sess, nil
}

func (a *Adapter) GetScreen(_ context.Context, sessionID string, includeScrollback bool) (ScreenView, error) {
	sess, err := a.lookup(sessionID)
	if err != nil {
		return ScreenView{}, err
	}
	view, plain := sess.ScreenWithText(includeScrollback)
	cells := make([][]Cell, len(view.Cells))
	for y, row := range view.Cells {
		out := make([]Cell, len(row))
		for x, c := range row {
			out[x] = Cell{
				Rune: c.Rune,
				SGR:  sgrToHandler(c.SGR),
			}
		}
		cells[y] = out
	}
	return ScreenView{
		Cols:            view.Cols,
		Rows:            view.Rows,
		Cells:           cells,
		Cursor:          Cursor{X: view.Cursor.X, Y: view.Cursor.Y},
		InAltBuffer:     view.InAltBuffer,
		ScrollbackLines: view.ScrollbackLines,
		PlainText:       plain,
	}, nil
}

func (a *Adapter) SendInput(_ context.Context, sessionID string, in InputRequest) (int, error) {
	sess, err := a.lookup(sessionID)
	if err != nil {
		return 0, err
	}
	var input session.SessionInput
	switch in.Variant {
	case InputVariantText:
		input = session.InputText(in.Text)
	case InputVariantKeys:
		keys := make([]session.Key, 0, len(in.Keys))
		for _, k := range in.Keys {
			keys = append(keys, session.Key{Name: k.Name, Ctrl: k.Ctrl, Alt: k.Alt, Shift: k.Shift})
		}
		input = session.InputKeys(keys...)
	case InputVariantRaw:
		input = session.InputRaw(in.Raw)
	default:
		return 0, fmt.Errorf("body is required: %w", ErrInvalidArgument)
	}
	if in.IsPaste {
		input = input.AsPaste()
	}
	source := in.Source
	if source == "" {
		source = "terminal-rpc"
	}
	input = input.WithSource(source)
	n, err := sess.SendInputCountWithKeyMap(input, DefaultKeyMap{})
	if err != nil {
		if isUnknownKeyError(err) {
			return 0, fmt.Errorf("%v: %w", err, ErrInvalidArgument)
		}
		return 0, fmt.Errorf("%v: %w", err, ErrInternal)
	}
	return n, nil
}

func (a *Adapter) WaitIdle(ctx context.Context, sessionID string, quietWindow, timeout time.Duration) (WaitIdleResult, error) {
	sess, err := a.lookup(sessionID)
	if err != nil {
		return WaitIdleResult{}, err
	}
	reason, waited, werr := sess.WaitIdle(ctx, quietWindow, timeout)
	if werr != nil {
		return WaitIdleResult{}, fmt.Errorf("%v: %w", werr, ErrInternal)
	}
	var r WaitIdleReason
	switch reason {
	case "idle":
		r = WaitIdleReasonIdle
	case "timeout":
		r = WaitIdleReasonTimeout
	case "exited":
		r = WaitIdleReasonExited
	}
	return WaitIdleResult{Reason: r, Waited: waited}, nil
}

func sgrToHandler(s terminal.SGR) SGR {
	return SGR{
		FG: s.FG, BG: s.BG,
		Bold: s.Bold, Italic: s.Italic, Underline: s.Underline,
		Inverse: s.Inverse, Faint: s.Faint,
	}
}

// isUnknownKeyError detects the typed session error for an unrecognized key.
func isUnknownKeyError(err error) bool {
	return errors.Is(err, session.ErrUnknownKey)
}
