package session

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"web-console/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `session` subcommand group covering session CRUD,
// expiration policy, and the conversation sub-resource on /sessions/{id}.
// The API is the source of truth; this package is a thin presentation layer.
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
			{Name: "conversation", Description: "Show the conversation feed for a session", Run: func(args []string) error { return runConversation(core, args) }},
			{Name: "conversation-cursor", Description: "Update the conversation cursor (--body-file PATH)", Run: func(args []string) error { return runConversationCursor(core, args) }},
			{Name: "summarize-event", Description: "Trigger summarization of a conversation event", Run: func(args []string) error { return runSummarizeEvent(core, args) }},
			{Name: "list-recoverable", Description: "List orphaned persistent sessions awaiting recovery", Run: func(args []string) error { return runListRecoverable(core, args) }},
			{Name: "recover", Description: "Recover an orphaned persistent session into a fresh pane", Run: func(args []string) error { return runRecover(core, args) }},
			{Name: "dismiss", Description: "Permanently dismiss an orphaned session row (preserves on-disk state)", Run: func(args []string) error { return runDismiss(core, args) }},
		},
	}
}

func runListRecoverable(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("session list-recoverable")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Get("/sessions/recoverable", nil)
	if err != nil {
		return err
	}
	var rows []support.RecoverableSession
	if err := support.Decode(body, &rows); err != nil {
		return err
	}

	results := recoverableRows(rows)
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Recoverable sessions: %d", len(rows))},
		ResultsHeading: "Recoverable",
		Results:        results,
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

	body, err := core.Request("POST", "/sessions/"+id+"/recover", nil, map[string]any{})
	if err != nil {
		return err
	}
	var res support.RecoverResult
	if err := support.Decode(body, &res); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Recovered %s -> %s", support.ShortID(res.OldSessionID), support.ShortID(res.NewSessionID)),
		},
		Changes: []string{
			fmt.Sprintf("Agent: %s", res.AgentType),
			fmt.Sprintf("Pasted: %s", strings.TrimSpace(res.CommandSent)),
			fmt.Sprintf("CODEX_HOME copied: %t", res.CodexHomeCopy),
		},
		NextCommand: []string{fmt.Sprintf("%s session get %s", support.CLIName, res.NewSessionID)},
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
	if _, err := core.Request("DELETE", "/sessions/recoverable/"+id, nil, nil); err != nil {
		return err
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

func recoverableRows(rows []support.RecoverableSession) []string {
	if len(rows) == 0 {
		return []string{"No orphaned persistent sessions awaiting recovery"}
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		flag := ""
		if !r.Recoverable {
			flag = " (NOT RECOVERABLE: " + r.NotRecoverable + ")"
		}
		out = append(out, fmt.Sprintf("%s | agent=%s | session=%s | orphaned=%s%s",
			support.ShortID(r.ID),
			defaultIfEmpty(r.AgentType, "none"),
			defaultIfEmpty(support.ShortID(r.AgentSessionID), "-"),
			support.FormatTime(r.OrphanedAt),
			flag,
		))
	}
	return out
}

func defaultIfEmpty(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("session list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/sessions", nil)
	if err != nil {
		return err
	}
	var sessions []support.Session
	if err := support.Decode(body, &sessions); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Active sessions: %d", len(sessions))},
		ResultsHeading: "Sessions",
		Results:        sessionRows(sessions),
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

	body, err := core.Get("/sessions/"+id, nil)
	if err != nil {
		return err
	}
	var sess support.Session
	if err := support.Decode(body, &sess); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", sess.ID),
		fmt.Sprintf("Shell: %s", sess.Shell),
		fmt.Sprintf("Backend: %s", sess.Backend),
		fmt.Sprintf("Size: %dx%d", sess.Cols, sess.Rows),
		fmt.Sprintf("Busy: %t", sess.Busy),
		fmt.Sprintf("Survives restart: %t", sess.SurvivesRestart),
	}
	if sess.CreatedAt != "" {
		results = append(results, fmt.Sprintf("Created: %s", support.FormatTime(sess.CreatedAt)))
	}
	if len(sess.Policy) > 0 {
		results = append(results, fmt.Sprintf("Policy: %s", string(sess.Policy)))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Session: %s (%s)", sess.ID, sess.Backend)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s session policy-get %s", support.CLIName, sess.ID),
			fmt.Sprintf("%s session conversation %s", support.CLIName, sess.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
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

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		body := map[string]interface{}{}
		if strings.TrimSpace(*shell) != "" {
			body["shell"] = *shell
		}
		if *cols > 0 {
			body["cols"] = *cols
		}
		if *rows > 0 {
			body["rows"] = *rows
		}
		if strings.TrimSpace(*backend) != "" {
			body["backend"] = *backend
		}
		payload = body
	}

	respBody, err := core.Request("POST", "/sessions", nil, payload)
	if err != nil {
		return err
	}
	var sess support.Session
	if err := support.Decode(respBody, &sess); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Created session %s", sess.ID)},
		Changes: []string{
			fmt.Sprintf("Backend: %s", sess.Backend),
			fmt.Sprintf("Shell: %s", sess.Shell),
		},
		NextCommand: []string{fmt.Sprintf("%s session get %s", support.CLIName, sess.ID)},
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

	if _, err := core.Request("DELETE", "/sessions/"+id, nil, nil); err != nil {
		return err
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

	body, err := core.Get("/sessions/"+id+"/policy", nil)
	if err != nil {
		return err
	}
	var resp support.PolicyResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := []string{fmt.Sprintf("Session: %s", resp.SessionID)}
	if len(resp.Policy) > 0 {
		results = append(results, fmt.Sprintf("Policy: %s", string(resp.Policy)))
	}
	if resp.ExpiresAt != "" {
		results = append(results, fmt.Sprintf("Expires at: %s", support.FormatTime(resp.ExpiresAt)))
	}
	if resp.TTL != nil {
		results = append(results, fmt.Sprintf("TTL: %.0fs", *resp.TTL))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Policy for session %s", resp.SessionID)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("%s session policy-set %s --mode <mode>", support.CLIName, resp.SessionID)},
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

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if strings.TrimSpace(*mode) == "" {
			return fmt.Errorf("--mode is required (or provide --body-file)")
		}
		payload = map[string]interface{}{
			"mode":     *mode,
			"duration": *duration,
		}
	}

	body, err := core.Request("PUT", "/sessions/"+id+"/policy", nil, payload)
	if err != nil {
		return err
	}
	var resp support.PolicyResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated policy for session %s", resp.SessionID)},
		Changes:     []string{fmt.Sprintf("Policy: %s", string(resp.Policy))},
		NextCommand: []string{fmt.Sprintf("%s session policy-get %s", support.CLIName, resp.SessionID)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runConversation(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("session conversation")
	limit := fs.Int("limit", 0, "Maximum events to return")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: session conversation <session-id> [--limit N]")
	}
	id := fs.Arg(0)

	query := support.BuildQuery(map[string]string{
		"limit": optInt(*limit),
	})
	body, err := core.Get("/sessions/"+id+"/conversation", query)
	if err != nil {
		return err
	}

	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Conversation for session %s", id)},
		ResultsHeading: "Payload",
		Results:        support.MapRows(payload),
		RetrievalHints: []string{fmt.Sprintf("%s session summarize-event %s <event-id>", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runConversationCursor(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("session conversation-cursor")
	bodyFile := fs.String("body-file", "", "Path to JSON body with the cursor payload (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: session conversation-cursor <session-id> --body-file PATH")
	}
	id := fs.Arg(0)

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	if _, err := core.Request("PUT", "/sessions/"+id+"/conversation/cursor", nil, payload); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated conversation cursor for session %s", id)},
		NextCommand: []string{fmt.Sprintf("%s session conversation %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSummarizeEvent(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("session summarize-event")
	bodyFile := fs.String("body-file", "", "Optional JSON body for the summarize request")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: session summarize-event <session-id> <event-id> [--body-file PATH]")
	}
	sessionID := fs.Arg(0)
	eventID := fs.Arg(1)

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	}

	body, err := core.Request("POST", "/sessions/"+sessionID+"/conversation/"+eventID+"/summarize", nil, payload)
	if err != nil {
		return err
	}
	var payloadMap map[string]interface{}
	_ = support.Decode(body, &payloadMap)

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Summarization requested for event %s (session %s)", eventID, sessionID)},
		Changes:     support.MapRows(payloadMap),
		NextCommand: []string{fmt.Sprintf("%s session conversation %s", support.CLIName, sessionID)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func sessionRows(sessions []support.Session) []string {
	if len(sessions) == 0 {
		return []string{"No active sessions"}
	}
	rows := make([]string, 0, len(sessions))
	for _, s := range sessions {
		rows = append(rows, fmt.Sprintf("%s | shell=%s | backend=%s | %dx%d | busy=%t",
			support.ShortID(s.ID), s.Shell, s.Backend, s.Cols, s.Rows, s.Busy))
	}
	return rows
}

func optInt(v int) string {
	if v <= 0 {
		return ""
	}
	return strconv.Itoa(v)
}
