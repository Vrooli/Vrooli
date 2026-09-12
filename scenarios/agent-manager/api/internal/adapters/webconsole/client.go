// Package webconsole is agent-manager's read-only consumer of the web-console
// Connect API. It wraps the proto-generated SessionsService and TerminalService
// clients behind a small, proto-free seam ([SessionController]) so the
// interactive execution substrate can create a terminal session, drive its
// stdin, read its screen, and tear it down without importing web-console code
// or its generated proto types.
//
// agent-manager never edits web-console; it consumes the foundation API
// (SessionOrigin, owner/display_label, execute_launch_command) exactly as
// shipped. See scenarios/agent-manager/docs/interactive-runner-design.md §2.
package webconsole

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	sessionsv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions/sessions_v1connect"
	terminalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/terminal"
	terminalv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/terminal/terminal_v1connect"
)

// OwnerAgentManager is the provenance tag agent-manager stamps on every session
// it creates, so the web-console sidebar and any auditor can attribute the
// session back to agent-manager.
const OwnerAgentManager = "agent-manager"

// pasteSubmitDelay is the pause between pasting a prompt and pressing Enter to
// submit it. The paste (bracketed-paste) and the Enter are separate SendInput
// calls; if the Enter races ahead before the TUI has finished ingesting the
// paste, it does not submit and the pasted text just accumulates unsent
// (observed with codex under load). A short settle delay makes the submit
// reliable across claude/codex/grok.
const pasteSubmitDelay = 400 * time.Millisecond

// CreateSessionParams describes a programmatic session agent-manager wants
// web-console to open. LaunchCommand is pasted+executed by the server when
// Execute is true (the foundation recovery-paste seam, no readiness gate).
type CreateSessionParams struct {
	// LaunchCommand is the full shell command line the server pastes into the
	// fresh session's stdin. For interactive runs this is the env-prefixed
	// interactive agent CLI invocation.
	LaunchCommand string
	// Execute asks the server to run LaunchCommand after create.
	Execute bool
	// DisplayLabel is the human-facing sidebar label (e.g. the run tag).
	DisplayLabel string
	// Backend selects the PTY backend; "persistent" gives a tmux-backed pane
	// that survives web-console restarts. Defaults to "persistent" when empty.
	Backend string
	// Cols/Rows size the PTY. Zero values fall back to sensible defaults.
	Cols int32
	Rows int32
}

// SessionInfo is the proto-free projection of a web-console session that the
// substrate needs. It deliberately omits fields no caller here consumes.
type SessionInfo struct {
	ID           string
	Owner        string
	Backend      string
	Origin       string
	DisplayLabel string
}

// SessionController is the proto-free web-console seam the interactive
// substrate depends on. [Client] implements it against the live service; tests
// substitute a fake.
type SessionController interface {
	// CreateSession opens a programmatic session owned by agent-manager and
	// returns its id.
	CreateSession(ctx context.Context, params CreateSessionParams) (string, error)
	// GetSession fetches current session metadata; returns ErrSessionNotFound
	// when the session no longer exists.
	GetSession(ctx context.Context, sessionID string) (SessionInfo, error)
	// DeleteSession tears the session down. It is idempotent: deleting an
	// already-gone session returns nil.
	DeleteSession(ctx context.Context, sessionID string) error
	// SendText types literal text into the session's stdin.
	SendText(ctx context.Context, sessionID, text, source string) error
	// SendPrompt delivers a follow-up prompt to an interactive agent TUI and
	// submits it: the prompt is pasted (bracketed-paste, so embedded newlines in
	// a multi-line prompt land as content, not submits), then a single Enter
	// keypress (carriage return) submits it — the reliable cross-TUI submit path
	// (claude/codex/grok). Used by the interactive Continue flow.
	SendPrompt(ctx context.Context, sessionID, prompt, source string) error
	// Interrupt sends the graceful interrupt key sequence (Escape then
	// Ctrl+C) used to stop an in-flight agent turn.
	Interrupt(ctx context.Context, sessionID, source string) error
	// Screen returns the session's plain-text screen (optionally including
	// scrollback), used to assert launch state.
	Screen(ctx context.Context, sessionID string, includeScrollback bool) (string, error)
}

// ErrSessionNotFound is returned by GetSession when the session is gone.
var ErrSessionNotFound = errors.New("web-console session not found")

// Client is the live SessionController backed by the generated Connect clients.
type Client struct {
	sessions sessionsv1connect.SessionsServiceClient
	terminal terminalv1connect.TerminalServiceClient
}

var _ SessionController = (*Client)(nil)

// NewClient builds a Client for the web-console API at baseURL. When
// httpClient is nil a default client with a bounded timeout is used.
func NewClient(baseURL string, httpClient connect.HTTPClient) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		sessions: sessionsv1connect.NewSessionsServiceClient(httpClient, baseURL),
		terminal: terminalv1connect.NewTerminalServiceClient(httpClient, baseURL),
	}
}

// CreateSession implements SessionController.
func (c *Client) CreateSession(ctx context.Context, params CreateSessionParams) (string, error) {
	backend := params.Backend
	if backend == "" {
		backend = "persistent"
	}
	cols := params.Cols
	if cols <= 0 {
		cols = 120
	}
	rows := params.Rows
	if rows <= 0 {
		rows = 40
	}
	req := connect.NewRequest(&sessionsv1.CreateRequest{
		Cols:                 cols,
		Rows:                 rows,
		Backend:              backend,
		LaunchCommand:        params.LaunchCommand,
		Origin:               sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC,
		Owner:                OwnerAgentManager,
		DisplayLabel:         params.DisplayLabel,
		ExecuteLaunchCommand: params.Execute,
	})
	resp, err := c.sessions.Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("web-console create session: %w", err)
	}
	id := resp.Msg.GetSession().GetId()
	if id == "" {
		return "", fmt.Errorf("web-console create session: empty session id in response")
	}
	return id, nil
}

// GetSession implements SessionController.
func (c *Client) GetSession(ctx context.Context, sessionID string) (SessionInfo, error) {
	resp, err := c.sessions.Get(ctx, connect.NewRequest(&sessionsv1.GetRequest{Id: sessionID}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return SessionInfo{}, ErrSessionNotFound
		}
		return SessionInfo{}, fmt.Errorf("web-console get session: %w", err)
	}
	s := resp.Msg.GetSession()
	return SessionInfo{
		ID:           s.GetId(),
		Owner:        s.GetOwner(),
		Backend:      s.GetBackend(),
		Origin:       s.GetOrigin().String(),
		DisplayLabel: s.GetDisplayLabel(),
	}, nil
}

// DeleteSession implements SessionController. Deleting a missing session is a
// success so Stop escalation and cleanup stay idempotent.
func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := c.sessions.Delete(ctx, connect.NewRequest(&sessionsv1.DeleteRequest{Id: sessionID}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return nil
		}
		return fmt.Errorf("web-console delete session: %w", err)
	}
	return nil
}

// SendText implements SessionController.
func (c *Client) SendText(ctx context.Context, sessionID, text, source string) error {
	_, err := c.terminal.SendInput(ctx, connect.NewRequest(&terminalv1.SendInputRequest{
		SessionId: sessionID,
		Body:      &terminalv1.SendInputRequest_Text{Text: text},
		Source:    source,
	}))
	if err != nil {
		return fmt.Errorf("web-console send text: %w", err)
	}
	return nil
}

// SendPrompt implements SessionController. It pastes the prompt via the PTY
// bracketed-paste path (is_paste=true, so a multi-line prompt is delivered as
// one block rather than submitting line-by-line) and then sends a single Enter
// key. Enter resolves to a carriage return (0x0d) via web-console's DefaultKeyMap
// — the submit key claude/codex/grok TUIs expect (the proto notes plain-text
// newlines are forwarded as LF and callers wanting a carriage-return submit
// should use a named Enter key). This mirrors web-console's own launch-command
// paste seam (paste-then-submit) for a reliable follow-up turn.
func (c *Client) SendPrompt(ctx context.Context, sessionID, prompt, source string) error {
	if _, err := c.terminal.SendInput(ctx, connect.NewRequest(&terminalv1.SendInputRequest{
		SessionId: sessionID,
		Body:      &terminalv1.SendInputRequest_Text{Text: prompt},
		Source:    source,
		IsPaste:   true,
	})); err != nil {
		return fmt.Errorf("web-console paste prompt: %w", err)
	}
	// Let the TUI finish ingesting the paste before the Enter, or the submit races
	// ahead and the pasted text is left unsent (see pasteSubmitDelay).
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(pasteSubmitDelay):
	}
	if _, err := c.terminal.SendInput(ctx, connect.NewRequest(&terminalv1.SendInputRequest{
		SessionId: sessionID,
		Body: &terminalv1.SendInputRequest_Keys{Keys: &terminalv1.KeySequence{
			Keys: []*terminalv1.Key{{Name: "enter"}},
		}},
		Source: source,
	})); err != nil {
		return fmt.Errorf("web-console submit prompt: %w", err)
	}
	return nil
}

// Interrupt implements SessionController. It sends Escape followed by Ctrl+C in
// a single key sequence — the interrupt convention the design doc (risk R5)
// specifies for stopping an in-flight interactive agent turn. Key names match
// web-console's DefaultKeyMap ("escape"; single-letter name + Ctrl → control
// byte).
func (c *Client) Interrupt(ctx context.Context, sessionID, source string) error {
	_, err := c.terminal.SendInput(ctx, connect.NewRequest(&terminalv1.SendInputRequest{
		SessionId: sessionID,
		Body: &terminalv1.SendInputRequest_Keys{Keys: &terminalv1.KeySequence{
			Keys: []*terminalv1.Key{
				{Name: "escape"},
				{Name: "c", Ctrl: true},
			},
		}},
		Source: source,
	}))
	if err != nil {
		return fmt.Errorf("web-console interrupt: %w", err)
	}
	return nil
}

// Screen implements SessionController.
func (c *Client) Screen(ctx context.Context, sessionID string, includeScrollback bool) (string, error) {
	resp, err := c.terminal.GetScreen(ctx, connect.NewRequest(&terminalv1.GetScreenRequest{
		SessionId:         sessionID,
		IncludeScrollback: includeScrollback,
	}))
	if err != nil {
		return "", fmt.Errorf("web-console get screen: %w", err)
	}
	return resp.Msg.GetPlainText(), nil
}
