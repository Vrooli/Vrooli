package user

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"scenario-authenticator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `user` subcommand group covering list/get/create/update/delete.
// Each subcommand is a thin wrapper that parses flags, issues a single API call,
// and renders the canonical report contract.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "user",
		Description: "Manage user accounts",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List users", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Get user details by ID", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", Description: "Register a new user account", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", Description: "Update a user (admin; requires --body-file)", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Delete a user (soft delete; admin)", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("user list")
	role := fs.String("role", "", "Filter users by role (e.g. admin, user)")
	limit := fs.Int("limit", 10, "Maximum users to return")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"role":  *role,
		"limit": strconv.Itoa(*limit),
	})
	body, err := core.Get("/users", query)
	if err != nil {
		return err
	}

	var resp support.UsersListResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Users returned: %d of %d total", len(resp.Users), resp.Total)}
	if *role != "" {
		summary = append(summary, fmt.Sprintf("Role filter: %s", *role))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Users",
		Results:        userRows(resp.Users),
		RetrievalHints: []string{
			fmt.Sprintf("%s user get <id>", support.CLIName),
			fmt.Sprintf("%s user list --role admin", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("user get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: user get <user-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/users/"+id, nil)
	if err != nil {
		return err
	}

	var user support.User
	if err := support.Decode(body, &user); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", user.ID),
		fmt.Sprintf("Email: %s", user.Email),
		fmt.Sprintf("Roles: %s", support.JoinStrings(user.Roles, "user")),
		fmt.Sprintf("Email verified: %t", user.EmailVerified),
		fmt.Sprintf("Created: %s", support.FormatTimeValue(user.CreatedAt)),
		fmt.Sprintf("Last login: %s", support.FormatTimePtr(user.LastLogin)),
	}
	if user.Username != "" {
		results = append(results, fmt.Sprintf("Username: %s", user.Username))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("User: %s", user.Email)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s session list --user-id %s", support.CLIName, user.ID),
			fmt.Sprintf("%s user update %s --body-file payload.json", support.CLIName, user.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("user create")
	email := fs.String("email", "", "Account email (required)")
	password := fs.String("password", "", "Account password (required, min 8 chars)")
	username := fs.String("username", "", "Optional username")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	if strings.TrimSpace(*email) == "" || strings.TrimSpace(*password) == "" {
		return fmt.Errorf("usage: user create --email <addr> --password <pw> [--username <name>]")
	}

	payload := map[string]interface{}{
		"email":    *email,
		"password": *password,
	}
	if strings.TrimSpace(*username) != "" {
		payload["username"] = *username
	}

	body, err := core.Request("POST", "/auth/register", nil, payload)
	if err != nil {
		return err
	}

	var resp support.AuthResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	result := []string{"User created"}
	changes := []string{}
	if resp.User != nil {
		result = append(result, fmt.Sprintf("ID: %s", resp.User.ID))
		result = append(result, fmt.Sprintf("Email: %s", resp.User.Email))
		changes = append(changes, fmt.Sprintf("User %s registered", resp.User.Email))
	}
	if resp.Token != "" {
		result = append(result, fmt.Sprintf("Initial token: %s...", firstN(resp.Token, 20)))
	}

	report := cliapp.MutationReport{
		Result:  result,
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("export SCENARIO_AUTHENTICATOR_API_TOKEN=%q", resp.Token),
			fmt.Sprintf("%s user get %s", support.CLIName, safeID(resp.User)),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("user update")
	bodyFile := fs.String("body-file", "", "Path to JSON file with UpdateUserRequest payload (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: user update <user-id> --body-file <path>")
	}
	id := fs.Arg(0)

	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("PUT", "/users/"+id, nil, raw)
	if err != nil {
		return err
	}

	var resp support.UpdateUserResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	changes := []string{fmt.Sprintf("User %s updated", id)}
	result := []string{"User updated"}
	result = append(result, fmt.Sprintf("Roles: %s", support.JoinStrings(resp.User.Roles, "user")))
	if resp.User.Username != "" {
		result = append(result, fmt.Sprintf("Username: %s", resp.User.Username))
	}
	result = append(result, fmt.Sprintf("Email verified: %t", resp.User.EmailVerified))

	report := cliapp.MutationReport{
		Result:      result,
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s user get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("user delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: user delete <user-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("DELETE", "/users/"+id, nil, nil)
	if err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("User %s deleted", id)
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{fmt.Sprintf("User %s soft-deleted", id)},
		NextCommand: []string{fmt.Sprintf("%s user list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func userRows(users []support.User) []string {
	if len(users) == 0 {
		return []string{"No users found"}
	}
	rows := make([]string, 0, len(users))
	for _, u := range users {
		rows = append(rows, fmt.Sprintf("%s (%s) | roles=%s | verified=%t",
			u.Email, support.ShortID(u.ID), support.JoinStrings(u.Roles, "user"), u.EmailVerified))
	}
	return rows
}

func firstN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func safeID(user *support.User) string {
	if user == nil {
		return "<id>"
	}
	return user.ID
}
