package launch

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"

	"research-desk/internal/peer"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions/sessions_v1connect"
	terminalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/terminal"
	terminalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/terminal/terminal_v1connect"
)

// Terminal is the Web Console seam (docs/internal/SEAMS.md): create a
// session and drive its terminal. Nothing else is ever sent.
type Terminal interface {
	Create(ctx context.Context, spec SessionSpec) (sessionID string, err error)
	Screen(ctx context.Context, sessionID string) (Screen, error)
	// Type types text; paste uses bracketed paste so newlines do not submit.
	Type(ctx context.Context, sessionID, text string, paste bool) error
	Keys(ctx context.Context, sessionID string, keys ...Key) error
	// WaitIdle waits for quiet output; exited is true when the PTY ended.
	WaitIdle(ctx context.Context, sessionID string, quiet, timeout time.Duration) (exited bool, err error)
}

// SessionSpec is one session to create.
type SessionSpec struct {
	// IdempotencyKey makes a repeated create return the same session.
	IdempotencyKey string
	Label          string
	WorkingDir     string
	Command        string
}

// Screen is the visible terminal text and the cursor row.
type Screen struct {
	Text      string
	CursorRow int
}

// Key is one named key press.
type Key struct {
	Name string
	Ctrl bool
}

// Keys used by the launcher.
var (
	Enter  = Key{Name: "Enter"}
	Escape = Key{Name: "Escape"}
	CtrlC  = Key{Name: "c", Ctrl: true}
)

// WebConsole drives sessions through Web Console's Connect API.
type WebConsole struct {
	Peer *peer.Peer
}

// owner tags sessions the desk creates (Web Console provenance).
const owner = "research-desk"

func (w *WebConsole) sessions(ctx context.Context) (sessionsconnect.SessionsServiceClient, error) {
	client, base, opt, err := w.Peer.Dial(ctx)
	if err != nil {
		return nil, err
	}
	return sessionsconnect.NewSessionsServiceClient(client, base, opt), nil
}

func (w *WebConsole) terminal(ctx context.Context) (terminalconnect.TerminalServiceClient, error) {
	client, base, opt, err := w.Peer.Dial(ctx)
	if err != nil {
		return nil, err
	}
	return terminalconnect.NewTerminalServiceClient(client, base, opt), nil
}

// Create implements Terminal.
func (w *WebConsole) Create(ctx context.Context, spec SessionSpec) (string, error) {
	client, err := w.sessions(ctx)
	if err != nil {
		return "", err
	}
	req := connect.NewRequest(&sessionsv1.CreateRequest{
		LaunchCommand:        spec.Command,
		ExecuteLaunchCommand: true,
		Origin:               sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC,
		Owner:                owner,
		DisplayLabel:         spec.Label,
		WorkingDir:           spec.WorkingDir,
	})
	req.Header().Set("X-Idempotency-Key", spec.IdempotencyKey)
	resp, err := client.Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("web-console Create: %w", err)
	}
	return resp.Msg.GetSession().GetId(), nil
}

// Screen implements Terminal.
func (w *WebConsole) Screen(ctx context.Context, sessionID string) (Screen, error) {
	client, err := w.terminal(ctx)
	if err != nil {
		return Screen{}, err
	}
	resp, err := client.GetScreen(ctx, connect.NewRequest(&terminalv1.GetScreenRequest{SessionId: sessionID}))
	if err != nil {
		return Screen{}, fmt.Errorf("web-console GetScreen: %w", err)
	}
	return Screen{Text: resp.Msg.GetPlainText(), CursorRow: int(resp.Msg.GetCursor().GetY())}, nil
}

// Type implements Terminal.
func (w *WebConsole) Type(ctx context.Context, sessionID, text string, paste bool) error {
	client, err := w.terminal(ctx)
	if err != nil {
		return err
	}
	_, err = client.SendInput(ctx, connect.NewRequest(&terminalv1.SendInputRequest{
		SessionId: sessionID,
		Body:      &terminalv1.SendInputRequest_Text{Text: text},
		Source:    owner,
		IsPaste:   paste,
	}))
	if err != nil {
		return fmt.Errorf("web-console SendInput: %w", err)
	}
	return nil
}

// Keys implements Terminal.
func (w *WebConsole) Keys(ctx context.Context, sessionID string, keys ...Key) error {
	client, err := w.terminal(ctx)
	if err != nil {
		return err
	}
	seq := &terminalv1.KeySequence{}
	for _, k := range keys {
		seq.Keys = append(seq.Keys, &terminalv1.Key{Name: k.Name, Ctrl: k.Ctrl})
	}
	_, err = client.SendInput(ctx, connect.NewRequest(&terminalv1.SendInputRequest{
		SessionId: sessionID,
		Body:      &terminalv1.SendInputRequest_Keys{Keys: seq},
		Source:    owner,
	}))
	if err != nil {
		return fmt.Errorf("web-console SendInput keys: %w", err)
	}
	return nil
}

// WaitIdle implements Terminal.
func (w *WebConsole) WaitIdle(ctx context.Context, sessionID string, quiet, timeout time.Duration) (bool, error) {
	client, err := w.terminal(ctx)
	if err != nil {
		return false, err
	}
	resp, err := client.WaitIdle(ctx, connect.NewRequest(&terminalv1.WaitIdleRequest{
		SessionId:   sessionID,
		QuietWindow: durationpb.New(quiet),
		Timeout:     durationpb.New(timeout),
	}))
	if err != nil {
		return false, fmt.Errorf("web-console WaitIdle: %w", err)
	}
	return resp.Msg.GetReason() == terminalv1.WaitIdleResponse_REASON_EXITED, nil
}

// Codex screen reading. Codex draws its input row as "› …", shows
// "esc to interrupt" while working, and a numbered list with "› 1." for a
// choice it needs answered (web-console backends/codex/prompt_detector.go).
var (
	codexChoice  = regexp.MustCompile(`^\s*[›❯>▌]\s*1\.\s+\S`)
	codexReplied = regexp.MustCompile(`(?i)^[\s•›>*-]*"?ready"?[.!]?\s*$`)
)

const codexWorking = "esc to interrupt"

// composer reports whether Codex waits at its input row, or the choice line
// it shows instead.
func composer(s Screen) (ready bool, choice string) {
	return inputRow(s, false)
}

// inputRow reports whether the cursor is in Codex's input row and no choice
// is shown; duringTurn accepts Codex's working line.
func inputRow(s Screen, duringTurn bool) (ready bool, choice string) {
	lines := strings.Split(s.Text, "\n")
	for _, line := range lines[max(0, len(lines)-24):] {
		if codexChoice.MatchString(line) {
			return false, strings.TrimSpace(line)
		}
	}
	if !duringTurn && strings.Contains(strings.ToLower(s.Text), codexWorking) {
		return false, ""
	}
	if s.CursorRow < 0 || s.CursorRow >= len(lines) {
		return false, ""
	}
	return strings.HasPrefix(strings.TrimSpace(lines[s.CursorRow]), "›"), ""
}

// holds reports whether Codex's input row (from the last "›" line up to the
// cursor) still shows the start of text.
func holds(s Screen, text string) bool {
	lines := strings.Split(s.Text, "\n")
	if s.CursorRow < 0 || s.CursorRow >= len(lines) {
		return false
	}
	start := -1
	for i := s.CursorRow; i >= 0; i-- {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "›") {
			start = i
			break
		}
	}
	if start < 0 {
		return false
	}
	row := strings.TrimPrefix(strings.TrimSpace(strings.Join(lines[start:s.CursorRow+1], " ")), "›")
	return strings.Contains(strings.Join(strings.Fields(row), " "), excerpt(text))
}

// excerpt is the start of a text, spaces collapsed, as a screen shows it.
func excerpt(text string) string {
	words := strings.Join(strings.Fields(text), " ")
	if r := []rune(words); len(r) > 40 {
		return string(r[:40])
	}
	return words
}

// repliedReady reports whether a line of the screen is Codex's "ready".
func repliedReady(s Screen) bool {
	for _, line := range strings.Split(s.Text, "\n") {
		if codexReplied.MatchString(line) {
			return true
		}
	}
	return false
}

// lastLine is the screen's last non-blank line, for error messages.
func lastLine(s Screen) string {
	lines := strings.Split(strings.TrimRight(s.Text, "\n "), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
