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

// OriginProgrammatic is [SessionInfo.Origin] for a session created through the
// API (as every agent-manager session is), as opposed to an operator-opened one.
const OriginProgrammatic = "SESSION_ORIGIN_PROGRAMMATIC"

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
	// CreatedAt is zero when web-console reports no parseable creation time.
	CreatedAt time.Time
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
	// ListSessions returns every session web-console currently holds, from any
	// owner; callers filter by Owner/Origin before acting on one.
	ListSessions(ctx context.Context) ([]SessionInfo, error)
	// ArchiveSession ends the session's process while web-console keeps its
	// metadata and transcript; web-console's archive retention owns any later
	// disposal. agent-manager never permanently deletes a session, so
	// conversation evidence survives. It is idempotent: archiving a session that
	// is already gone or archived returns nil.
	ArchiveSession(ctx context.Context, sessionID string) error
	// SendText types literal text into the session's stdin.
	SendText(ctx context.Context, sessionID, text, source string) error
	// SendPrompt delivers a prompt to an interactive agent TUI and submits it:
	// the prompt is pasted (bracketed-paste, so embedded newlines in a
	// multi-line prompt land as content, not submits), then Enter (carriage
	// return) submits it. Submission is confirmed from the screen, not assumed
	// after a fixed delay; see [Client.SendPrompt]. It returns
	// ErrPromptNotSubmitted when the composer is still seen holding the prompt
	// after bounded Enter retries, and a zero-Verified PromptSubmission when the
	// TUI's composer cannot be observed.
	SendPrompt(ctx context.Context, sessionID, prompt, source string) (PromptSubmission, error)
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

	// wait paces SendPrompt's screen polls; nil uses a real timer. Tests inject
	// an immediate wait so scripted screens advance per poll, not per clock.
	wait func(ctx context.Context, d time.Duration) error
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
	return sessionInfoFromProto(resp.Msg.GetSession()), nil
}

// ListSessions implements SessionController.
func (c *Client) ListSessions(ctx context.Context) ([]SessionInfo, error) {
	resp, err := c.sessions.List(ctx, connect.NewRequest(&sessionsv1.ListRequest{}))
	if err != nil {
		return nil, fmt.Errorf("web-console list sessions: %w", err)
	}
	out := make([]SessionInfo, 0, len(resp.Msg.GetSessions()))
	for _, s := range resp.Msg.GetSessions() {
		if s == nil || s.GetId() == "" {
			continue
		}
		out = append(out, sessionInfoFromProto(s))
	}
	return out, nil
}

func sessionInfoFromProto(s *sessionsv1.Session) SessionInfo {
	info := SessionInfo{
		ID:           s.GetId(),
		Owner:        s.GetOwner(),
		Backend:      s.GetBackend(),
		Origin:       s.GetOrigin().String(),
		DisplayLabel: s.GetDisplayLabel(),
	}
	// web-console stamps created_at as RFC3339 UTC; RFC3339Nano parses both.
	if created, err := time.Parse(time.RFC3339Nano, s.GetCreatedAt()); err == nil {
		info.CreatedAt = created
	}
	return info
}

// ArchiveSession implements SessionController. Archiving a missing session is
// a success so Stop escalation and the retention sweep stay idempotent; an
// already-archived session is not listed live and re-archives as a no-op.
func (c *Client) ArchiveSession(ctx context.Context, sessionID string) error {
	_, err := c.sessions.Archive(ctx, connect.NewRequest(&sessionsv1.ArchiveRequest{Id: sessionID}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return nil
		}
		return fmt.Errorf("web-console archive session: %w", err)
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
