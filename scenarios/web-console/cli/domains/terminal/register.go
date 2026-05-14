// Package terminal exposes the structured terminal surface as CLI
// subcommands: screen reads, programmatic input, and idle gates. All
// RPCs go through the Connect-RPC TerminalService — there is no REST
// fallback for these methods.
//
// The legacy xterm.js WebSocket bridge (/api/v1/sessions/{id}/ws) and
// multipart upload (/api/v1/sessions/{id}/upload) are NOT exposed here;
// they remain browser-side surfaces.
package terminal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"

	terminalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/terminal"
	terminalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/terminal/terminal_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"web-console/cli/internal/support"
)

// Register builds the `terminal` subcommand group: screen, send-text,
// send-keys, wait-idle. Defaults match memory: human-readable output;
// --json opts into JSON.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "terminal",
		Description: "Programmatic terminal access (screen reads, input, idle gates)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "screen", Description: "Print the decoded screen of a session (--include-scrollback, --json)", Run: func(args []string) error { return runScreen(core, args) }},
			{Name: "send-text", Description: "Send literal text to a session: send-text <session-id> <text...>", Run: func(args []string) error { return runSendText(core, args) }},
			{Name: "send-keys", Description: "Send named keys: send-keys <session-id> Enter Ctrl+C Up", Run: func(args []string) error { return runSendKeys(core, args) }},
			{Name: "wait-idle", Description: "Block until the session is idle (--quiet-window, --timeout)", Run: func(args []string) error { return runWaitIdle(core, args) }},
		},
	}
}

func newClient(core *cliapp.ScenarioApp) terminalconnect.TerminalServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return terminalconnect.NewTerminalServiceClient(httpClient, baseURL)
}

// -----------------------------------------------------------------------------
// screen
// -----------------------------------------------------------------------------

func runScreen(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("terminal screen")
	includeScrollback := fs.Bool("include-scrollback", false, "Include scrollback lines in the plain-text rendering")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) < 1 {
		return fmt.Errorf("usage: terminal screen <session-id>")
	}
	sessionID := rest[0]
	resp, err := newClient(core).GetScreen(context.Background(),
		connect.NewRequest(&terminalv1.GetScreenRequest{
			SessionId:         sessionID,
			IncludeScrollback: *includeScrollback,
		}))
	if err != nil {
		return cliapp.WrapAPIError("terminal screen", err, nil)
	}
	msg := resp.Msg
	if *jsonOutput {
		out := struct {
			SessionID       string `json:"session_id"`
			Cols            int32  `json:"cols"`
			Rows            int32  `json:"rows"`
			InAltBuffer     bool   `json:"in_alt_buffer"`
			ScrollbackLines int32  `json:"scrollback_lines"`
			CursorX         int32  `json:"cursor_x"`
			CursorY         int32  `json:"cursor_y"`
			PlainText       string `json:"plain_text"`
		}{
			SessionID:       sessionID,
			Cols:            msg.GetCols(),
			Rows:            msg.GetRows(),
			InAltBuffer:     msg.GetInAltBuffer(),
			ScrollbackLines: msg.GetScrollbackLines(),
			CursorX:         msg.GetCursor().GetX(),
			CursorY:         msg.GetCursor().GetY(),
			PlainText:       msg.GetPlainText(),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	fmt.Fprintf(os.Stdout, "%s\n", msg.GetPlainText())
	fmt.Fprintf(os.Stderr, "\n[%dx%d cursor=(%d,%d) alt=%v scrollback=%d]\n",
		msg.GetCols(), msg.GetRows(),
		msg.GetCursor().GetX(), msg.GetCursor().GetY(),
		msg.GetInAltBuffer(), msg.GetScrollbackLines())
	return nil
}

// -----------------------------------------------------------------------------
// send-text
// -----------------------------------------------------------------------------

func runSendText(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("terminal send-text")
	isPaste := fs.Bool("paste", false, "Deliver as bracketed paste (when backend supports it)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) < 2 {
		return fmt.Errorf("usage: terminal send-text <session-id> <text...>")
	}
	sessionID := rest[0]
	text := strings.Join(rest[1:], " ")
	_, err := newClient(core).SendInput(context.Background(),
		connect.NewRequest(&terminalv1.SendInputRequest{
			SessionId: sessionID,
			Body:      &terminalv1.SendInputRequest_Text{Text: text},
			IsPaste:   *isPaste,
			Source:    "cli",
		}))
	if err != nil {
		return cliapp.WrapAPIError("terminal send-text", err, nil)
	}
	if *jsonOutput {
		fmt.Println(`{"ok":true}`)
		return nil
	}
	fmt.Fprintf(os.Stdout, "sent %d characters to %s\n", len(text), sessionID)
	return nil
}

// -----------------------------------------------------------------------------
// send-keys
// -----------------------------------------------------------------------------

func runSendKeys(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("terminal send-keys")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) < 2 {
		return fmt.Errorf("usage: terminal send-keys <session-id> <key> [key...]\n  examples: Enter, Tab, Ctrl+C, Alt+Shift+f5")
	}
	sessionID := rest[0]
	keys := make([]*terminalv1.Key, 0, len(rest)-1)
	for _, spec := range rest[1:] {
		k, err := parseKeySpec(spec)
		if err != nil {
			return err
		}
		keys = append(keys, k)
	}
	_, err := newClient(core).SendInput(context.Background(),
		connect.NewRequest(&terminalv1.SendInputRequest{
			SessionId: sessionID,
			Body:      &terminalv1.SendInputRequest_Keys{Keys: &terminalv1.KeySequence{Keys: keys}},
			Source:    "cli",
		}))
	if err != nil {
		return cliapp.WrapAPIError("terminal send-keys", err, nil)
	}
	if *jsonOutput {
		fmt.Println(`{"ok":true}`)
		return nil
	}
	fmt.Fprintf(os.Stdout, "sent %d key(s) to %s\n", len(keys), sessionID)
	return nil
}

// parseKeySpec parses "Ctrl+Alt+Enter"-style key specifications.
// Modifier prefixes (case-insensitive): Ctrl, Alt, Shift. The final
// segment is the key name.
func parseKeySpec(spec string) (*terminalv1.Key, error) {
	parts := strings.Split(spec, "+")
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty key spec")
	}
	k := &terminalv1.Key{}
	for _, p := range parts[:len(parts)-1] {
		switch strings.ToLower(strings.TrimSpace(p)) {
		case "ctrl", "control":
			k.Ctrl = true
		case "alt", "meta", "option":
			k.Alt = true
		case "shift":
			k.Shift = true
		default:
			return nil, fmt.Errorf("unknown modifier %q in %q", p, spec)
		}
	}
	k.Name = strings.TrimSpace(parts[len(parts)-1])
	if k.Name == "" {
		return nil, fmt.Errorf("missing key name in %q", spec)
	}
	return k, nil
}

// -----------------------------------------------------------------------------
// wait-idle
// -----------------------------------------------------------------------------

func runWaitIdle(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("terminal wait-idle")
	quiet := fs.Duration("quiet-window", 200*time.Millisecond, "Time the session must produce no output to count as idle")
	timeout := fs.Duration("timeout", 5*time.Second, "Maximum total time to wait")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) < 1 {
		return fmt.Errorf("usage: terminal wait-idle <session-id>")
	}
	sessionID := rest[0]
	resp, err := newClient(core).WaitIdle(context.Background(),
		connect.NewRequest(&terminalv1.WaitIdleRequest{
			SessionId:   sessionID,
			QuietWindow: durationpb.New(*quiet),
			Timeout:     durationpb.New(*timeout),
		}))
	if err != nil {
		return cliapp.WrapAPIError("terminal wait-idle", err, nil)
	}
	reason := strings.ToLower(strings.TrimPrefix(resp.Msg.GetReason().String(), "REASON_"))
	waited := resp.Msg.GetWaited().AsDuration()
	if *jsonOutput {
		out := struct {
			Reason string `json:"reason"`
			Waited string `json:"waited"`
		}{Reason: reason, Waited: waited.String()}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	fmt.Fprintf(os.Stdout, "%s after %s\n", reason, waited)
	// Exit non-zero for timeout so shell scripts can branch.
	if resp.Msg.GetReason() == terminalv1.WaitIdleResponse_REASON_TIMEOUT {
		os.Exit(2)
	}
	return nil
}
