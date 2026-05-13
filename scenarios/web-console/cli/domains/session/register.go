package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions/sessions_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"web-console/cli/internal/support"
)

// Register builds the `session` subcommand group covering session CRUD,
// expiration policy, and persistent-session recovery. All RPCs go through
// the Connect-RPC SessionsService — the legacy REST routes have been
// removed.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "session",
		Description: "Manage PTY terminal sessions",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List active sessions", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one session", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", Description: "Create a session (optionally --body-file PATH)", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Terminate a session", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "policy-get", Description: "Show the expiration policy for a session", Run: func(args []string) error { return runPolicyGet(core, args) }},
			{Name: "policy-set", Description: "Set the expiration policy (--mode, optional --duration)", Run: func(args []string) error { return runPolicySet(core, args) }},
			{Name: "list-recoverable", Description: "List orphaned persistent sessions awaiting recovery", Run: func(args []string) error { return runListRecoverable(core, args) }},
			{Name: "recover", Description: "Recover an orphaned persistent session into a fresh pane", Run: func(args []string) error { return runRecover(core, args) }},
			{Name: "dismiss", Description: "Permanently dismiss an orphaned session row (preserves on-disk state)", Run: func(args []string) error { return runDismiss(core, args) }},
		},
	}
}

func newClient(core *cliapp.ScenarioApp) sessionsconnect.SessionsServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return sessionsconnect.NewSessionsServiceClient(httpClient, baseURL)
}

// -----------------------------------------------------------------------------
// list / get / create / delete
// -----------------------------------------------------------------------------

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("session list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	resp, err := newClient(core).List(context.Background(), connect.NewRequest(&sessionsv1.ListRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("session list", err, nil)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Active sessions: %d", len(resp.Msg.GetSessions()))},
		ResultsHeading: "Sessions",
		Results:        sessionRows(resp.Msg.GetSessions()),
		RetrievalHints: []string{
			fmt.Sprintf("%s session get <session-id>", support.CLIName),
			fmt.Sprintf("%s session policy-get <session-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("session get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: session get <session-id>")
	}
	id := fs.Arg(0)

	resp, err := newClient(core).Get(context.Background(), connect.NewRequest(&sessionsv1.GetRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("session get", err, nil)
	}
	sess := resp.Msg.GetSession()

	results := []string{
		fmt.Sprintf("ID: %s", sess.GetId()),
		fmt.Sprintf("Shell: %s", sess.GetShell()),
		fmt.Sprintf("Backend: %s", sess.GetBackend()),
		fmt.Sprintf("Size: %dx%d", sess.GetCols(), sess.GetRows()),
		fmt.Sprintf("Busy: %t", sess.GetBusy()),
		fmt.Sprintf("Survives restart: %t", sess.GetSurvivesRestart()),
	}
	if t := sess.GetCreatedAt(); t != "" {
		results = append(results, fmt.Sprintf("Created: %s", support.FormatTime(t)))
	}
	results = append(results, fmt.Sprintf("Policy: %s", policyString(sess.GetPolicy())))

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Session: %s (%s)", sess.GetId(), sess.GetBackend())},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s session policy-get %s", support.CLIName, sess.GetId()),
			fmt.Sprintf("%s conversation get --session %s", support.CLIName, sess.GetId()),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// createBody mirrors CreateRequest's JSON form so --body-file is supported
// uniformly. Pointer fields toggle the has_policy flag server-side.
type createBody struct {
	Shell         string         `json:"shell,omitempty"`
	Cols          int32          `json:"cols,omitempty"`
	Rows          int32          `json:"rows,omitempty"`
	Backend       string         `json:"backend,omitempty"`
	Policy        *policyBody    `json:"policy,omitempty"`
	LaunchCommand string         `json:"launchCommand,omitempty"`
	AgentType     string         `json:"agentType,omitempty"`
	Extra         map[string]any `json:"-"`
}

type policyBody struct {
	Mode     string `json:"mode"`
	Duration string `json:"duration,omitempty"`
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("session create")
	shell := fs.String("shell", "", "Shell to run (default: user shell)")
	cols := fs.Int("cols", 0, "Initial columns")
	rows := fs.Int("rows", 0, "Initial rows")
	backend := fs.String("backend", "", "Backend id (e.g. pty, tmux)")
	bodyFile := fs.String("body-file", "", "Path to a JSON body (overrides individual flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body := createBody{
		Shell:   strings.TrimSpace(*shell),
		Cols:    int32(*cols),
		Rows:    int32(*rows),
		Backend: strings.TrimSpace(*backend),
	}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			return fmt.Errorf("decode --body-file: %w", err)
		}
	}

	req := connect.NewRequest(&sessionsv1.CreateRequest{
		Shell:         body.Shell,
		Cols:          body.Cols,
		Rows:          body.Rows,
		Backend:       body.Backend,
		LaunchCommand: body.LaunchCommand,
		AgentType:     body.AgentType,
	})
	if body.Policy != nil {
		req.Msg.Policy = &sessionsv1.ExpirationPolicy{Mode: body.Policy.Mode, Duration: body.Policy.Duration}
		req.Msg.HasPolicy = true
	}

	resp, err := newClient(core).Create(context.Background(), req)
	if err != nil {
		return cliapp.WrapAPIError("session create", err, nil)
	}
	sess := resp.Msg.GetSession()

	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Created session %s", sess.GetId())},
		Changes: []string{
			fmt.Sprintf("Backend: %s", sess.GetBackend()),
			fmt.Sprintf("Shell: %s", sess.GetShell()),
		},
		NextCommand: []string{fmt.Sprintf("%s session get %s", support.CLIName, sess.GetId())},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("session delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: session delete <session-id>")
	}
	id := fs.Arg(0)

	if _, err := newClient(core).Delete(context.Background(),
		connect.NewRequest(&sessionsv1.DeleteRequest{Id: id})); err != nil {
		return cliapp.WrapAPIError("session delete", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted session %s", id)},
		NextCommand: []string{fmt.Sprintf("%s session list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

// -----------------------------------------------------------------------------
// policy
// -----------------------------------------------------------------------------

func runPolicyGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("session policy-get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: session policy-get <session-id>")
	}
	id := fs.Arg(0)

	resp, err := newClient(core).GetPolicy(context.Background(), connect.NewRequest(&sessionsv1.GetPolicyRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("session policy-get", err, nil)
	}
	view := resp.Msg.GetPolicy()

	results := []string{
		fmt.Sprintf("Session: %s", view.GetSessionId()),
		fmt.Sprintf("Policy: %s", policyString(view.GetPolicy())),
	}
	if view.GetHasExpiry() {
		results = append(results,
			fmt.Sprintf("Expires at: %s", support.FormatTime(view.GetExpiresAt())),
			fmt.Sprintf("TTL: %.0fs", view.GetTtlSeconds()),
		)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Policy for session %s", view.GetSessionId())},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("%s session policy-set %s --mode <mode>", support.CLIName, view.GetSessionId())},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runPolicySet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("session policy-set")
	mode := fs.String("mode", "", "Policy mode (required)")
	duration := fs.String("duration", "", "Optional duration (e.g. 1h, 30m)")
	bodyFile := fs.String("body-file", "", "Path to a JSON body (overrides --mode/--duration)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: session policy-set <session-id> --mode <mode> [--duration <dur>]")
	}
	id := fs.Arg(0)

	var policy policyBody
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &policy); err != nil {
			return fmt.Errorf("decode --body-file: %w", err)
		}
	} else {
		if strings.TrimSpace(*mode) == "" {
			return fmt.Errorf("--mode is required (or provide --body-file)")
		}
		policy = policyBody{Mode: *mode, Duration: *duration}
	}

	resp, err := newClient(core).UpdatePolicy(context.Background(), connect.NewRequest(&sessionsv1.UpdatePolicyRequest{
		Id: id,
		Policy: &sessionsv1.ExpirationPolicy{
			Mode:     policy.Mode,
			Duration: policy.Duration,
		},
	}))
	if err != nil {
		return cliapp.WrapAPIError("session policy-set", err, nil)
	}
	view := resp.Msg.GetPolicy()

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated policy for session %s", view.GetSessionId())},
		Changes:     []string{fmt.Sprintf("Policy: %s", policyString(view.GetPolicy()))},
		NextCommand: []string{fmt.Sprintf("%s session policy-get %s", support.CLIName, view.GetSessionId())},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

// -----------------------------------------------------------------------------
// recovery
// -----------------------------------------------------------------------------

func runListRecoverable(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("session list-recoverable")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	resp, err := newClient(core).ListRecoverable(context.Background(),
		connect.NewRequest(&sessionsv1.ListRecoverableRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("session list-recoverable", err, nil)
	}
	rows := resp.Msg.GetSessions()

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Recoverable sessions: %d", len(rows))},
		ResultsHeading: "Recoverable",
		Results:        recoverableRows(rows),
		RetrievalHints: []string{
			fmt.Sprintf("%s session recover <session-id>", support.CLIName),
			fmt.Sprintf("%s session dismiss <session-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runRecover(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("session recover")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: session recover <session-id>")
	}
	id := fs.Arg(0)

	resp, err := newClient(core).Recover(context.Background(), connect.NewRequest(&sessionsv1.RecoverRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("session recover", err, nil)
	}
	res := resp.Msg

	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Recovered %s -> %s", support.ShortID(res.GetOldSessionId()), support.ShortID(res.GetNewSessionId()))},
		Changes: []string{
			fmt.Sprintf("Agent: %s", res.GetAgentType()),
			fmt.Sprintf("Pasted: %s", strings.TrimSpace(res.GetCommandSent())),
			fmt.Sprintf("CODEX_HOME copied: %t", res.GetCodexHomeCopied()),
		},
		NextCommand: []string{fmt.Sprintf("%s session get %s", support.CLIName, res.GetNewSessionId())},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDismiss(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("session dismiss")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: session dismiss <session-id>")
	}
	id := fs.Arg(0)
	if _, err := newClient(core).DismissRecoverable(context.Background(),
		connect.NewRequest(&sessionsv1.DismissRecoverableRequest{Id: id})); err != nil {
		return cliapp.WrapAPIError("session dismiss", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Dismissed orphan session %s (on-disk state preserved)", id)},
		NextCommand: []string{fmt.Sprintf("%s session list-recoverable", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

// -----------------------------------------------------------------------------
// formatting helpers
// -----------------------------------------------------------------------------

func sessionRows(sessions []*sessionsv1.Session) []string {
	if len(sessions) == 0 {
		return []string{"No active sessions"}
	}
	rows := make([]string, 0, len(sessions))
	for _, s := range sessions {
		rows = append(rows, fmt.Sprintf("%s | shell=%s | backend=%s | %dx%d | busy=%t",
			support.ShortID(s.GetId()), s.GetShell(), s.GetBackend(), s.GetCols(), s.GetRows(), s.GetBusy()))
	}
	return rows
}

func recoverableRows(rows []*sessionsv1.RecoverableSession) []string {
	if len(rows) == 0 {
		return []string{"No orphaned persistent sessions awaiting recovery"}
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		flag := ""
		if !r.GetRecoverable() {
			flag = " (NOT RECOVERABLE: " + r.GetNotRecoverableReason() + ")"
		}
		out = append(out, fmt.Sprintf("%s | agent=%s | session=%s | orphaned=%s%s",
			support.ShortID(r.GetId()),
			defaultIfEmpty(r.GetAgentType(), "none"),
			defaultIfEmpty(support.ShortID(r.GetAgentSessionId()), "-"),
			support.FormatTime(r.GetOrphanedAt()),
			flag,
		))
	}
	return out
}

func policyString(p *sessionsv1.ExpirationPolicy) string {
	if p == nil || p.GetMode() == "" {
		return "never"
	}
	if p.GetDuration() != "" {
		return p.GetMode() + "/" + p.GetDuration()
	}
	return p.GetMode()
}

func defaultIfEmpty(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

// Build-time guard: ensures time import isn't accidentally removed when
// the file is regenerated; FormatTime accepts strings, not time.Time, so
// nothing imports it directly today.
var _ = time.RFC3339
