package session

import (
	"fmt"
	"os"
	"strconv"

	"scenario-authenticator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `session` subcommand group covering list/revoke.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "session",
		Description: "Inspect and revoke active sessions",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List active sessions", Run: func(args []string) error { return runList(core, args) }},
			{Name: "revoke", Aliases: []string{"rm"}, Description: "Revoke a session by ID", Run: func(args []string) error { return runRevoke(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("session list")
	userID := fs.String("user-id", "", "Filter by user ID (admin-only if different from caller)")
	scope := fs.String("scope", "", "Set to 'all' (admin only) to return every active session")
	limit := fs.Int("limit", 200, "Maximum sessions to return")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"user_id": *userID,
		"scope":   *scope,
		"limit":   strconv.Itoa(*limit),
	})
	body, err := core.Get("/sessions", query)
	if err != nil {
		return err
	}

	var resp support.SessionsListResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Active sessions: %d (%d shown)", resp.Total, len(resp.Sessions))},
		ResultsHeading: "Sessions",
		Results:        sessionRows(resp.Sessions),
		RetrievalHints: []string{
			fmt.Sprintf("%s session revoke <session-id>", support.CLIName),
			fmt.Sprintf("%s session list --scope all", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runRevoke(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("session revoke")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: session revoke <session-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("DELETE", "/sessions/"+id, nil, nil)
	if err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Session %s revoked", id)
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{fmt.Sprintf("Session %s revoked", id)},
		NextCommand: []string{fmt.Sprintf("%s session list", support.CLIName)},
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
		rows = append(rows, fmt.Sprintf("%s | user=%s | ip=%s | created=%s | expires=%s",
			s.SessionID,
			support.ShortID(s.UserID),
			fallback(s.IPAddress, "-"),
			support.FormatTimeValue(s.CreatedAt),
			support.FormatTimeValue(s.ExpiresAt),
		))
	}
	return rows
}

func fallback(value, def string) string {
	if value == "" {
		return def
	}
	return value
}
