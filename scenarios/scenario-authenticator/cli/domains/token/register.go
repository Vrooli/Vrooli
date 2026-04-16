package token

import (
	"fmt"
	"os"
	"strings"

	"scenario-authenticator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `token` subcommand group covering validate/refresh/logout.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "token",
		Description: "Validate, refresh, and invalidate auth tokens",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "validate", Description: "Validate a JWT and show its claims", Run: func(args []string) error { return runValidate(core, args) }},
			{Name: "refresh", Description: "Exchange a refresh token for a fresh JWT", Run: func(args []string) error { return runRefresh(core, args) }},
			{Name: "logout", Description: "Revoke the currently configured token", Run: func(args []string) error { return runLogout(core, args) }},
		},
	}
}

func runValidate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("token validate")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: token validate <token>")
	}
	token := fs.Arg(0)

	// GET /auth/validate reads the Authorization header from the active client.
	// POST /auth/validate accepts the token in the body; the API supports both
	// via the same handler, so we send the token explicitly in the body to
	// decouple from the configured token.
	body, err := core.Request("POST", "/auth/validate", nil, map[string]string{"token": token})
	if err != nil {
		return err
	}

	var resp support.ValidationResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	if !resp.Valid {
		report := cliapp.OperationalReport{
			Status:    []string{"Token is invalid or expired"},
			NextSteps: []string{fmt.Sprintf("%s token refresh <refresh-token>", support.CLIName)},
		}
		if *jsonOutput {
			return cliapp.PrintReportJSON(os.Stdout, report)
		}
		return cliapp.RenderOperationalReport(os.Stdout, report)
	}

	results := []string{
		fmt.Sprintf("User ID: %s", resp.UserID),
		fmt.Sprintf("Email: %s", resp.Email),
		fmt.Sprintf("Roles: %s", support.JoinStrings(resp.Roles, "user")),
		fmt.Sprintf("Expires: %s", support.FormatTimeValue(resp.ExpiresAt)),
	}

	report := cliapp.ListReport{
		Summary:        []string{"Token is valid"},
		ResultsHeading: "Claims",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("%s user get %s", support.CLIName, resp.UserID)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runRefresh(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("token refresh")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: token refresh <refresh-token>")
	}
	refresh := strings.TrimSpace(fs.Arg(0))

	body, err := core.Request("POST", "/auth/refresh", nil, map[string]string{"refresh_token": refresh})
	if err != nil {
		return err
	}

	var resp support.AuthResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	result := []string{"Token refreshed"}
	if resp.Token != "" {
		result = append(result, fmt.Sprintf("Token: %s...", firstN(resp.Token, 20)))
	}
	if resp.RefreshToken != "" {
		result = append(result, fmt.Sprintf("Refresh token: %s...", firstN(resp.RefreshToken, 20)))
	}

	report := cliapp.MutationReport{
		Result:  result,
		Changes: []string{"New access token issued"},
		NextCommand: []string{
			fmt.Sprintf("export SCENARIO_AUTHENTICATOR_API_TOKEN=%q", resp.Token),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runLogout(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("token logout")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Request("POST", "/auth/logout", nil, nil)
	if err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = "Logged out"
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{"Token blacklisted and sessions cleared"},
		NextCommand: []string{"unset SCENARIO_AUTHENTICATOR_API_TOKEN"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func firstN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
