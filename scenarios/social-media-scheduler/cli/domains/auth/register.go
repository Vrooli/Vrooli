package auth

import (
	"fmt"
	"os"
	"strings"

	"social-media-scheduler/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `auth` subcommand group covering login, register, logout,
// and whoami. login/register persist the returned JWT to the cli-core config
// so later commands can authenticate automatically.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "auth",
		Description: "Authenticate with the scheduler API",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "login", Description: "Log in and store the returned token", Run: func(args []string) error { return runLogin(core, args) }},
			{Name: "register", Description: "Register a new user and store the returned token", Run: func(args []string) error { return runRegister(core, args) }},
			{Name: "logout", Description: "Clear the stored auth token", NeedsAPI: false, Run: func(args []string) error { return runLogout(core, args) }},
			{Name: "whoami", Description: "Show the currently authenticated user", Run: func(args []string) error { return runWhoami(core, args) }},
		},
	}
}

func runLogin(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("auth login")
	email := fs.String("email", "", "Account email (required)")
	password := fs.String("password", "", "Account password (required)")
	bodyFile := fs.String("body-file", "", "Optional JSON file with the login payload (overrides --email/--password)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := buildAuthBody(*bodyFile, map[string]interface{}{
		"email":    strings.TrimSpace(*email),
		"password": *password,
	}, []string{"email", "password"})
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/auth/login", nil, payload)
	if err != nil {
		return err
	}
	return handleAuthResponse(core, body, "Login successful", *jsonOutput)
}

func runRegister(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("auth register")
	email := fs.String("email", "", "Account email (required)")
	password := fs.String("password", "", "Account password (required, min 8 chars)")
	firstName := fs.String("first-name", "", "First name")
	lastName := fs.String("last-name", "", "Last name")
	timezone := fs.String("timezone", "", "IANA timezone (e.g. UTC)")
	bodyFile := fs.String("body-file", "", "Optional JSON file with the register payload (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := buildAuthBody(*bodyFile, map[string]interface{}{
		"email":      strings.TrimSpace(*email),
		"password":   *password,
		"first_name": strings.TrimSpace(*firstName),
		"last_name":  strings.TrimSpace(*lastName),
		"timezone":   strings.TrimSpace(*timezone),
	}, []string{"email", "password"})
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/auth/register", nil, payload)
	if err != nil {
		return err
	}
	return handleAuthResponse(core, body, "Registration successful", *jsonOutput)
}

func runLogout(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("auth logout")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	core.Config.Token = ""
	if err := core.SaveConfig(); err != nil {
		return fmt.Errorf("clear token: %w", err)
	}

	report := cliapp.MutationReport{
		Result:      []string{"Cleared stored auth token"},
		Changes:     []string{"Removed token from CLI config"},
		NextCommand: []string{fmt.Sprintf("%s auth login --email <email> --password <password>", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runWhoami(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("auth whoami")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/auth/me", nil)
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
	}
	if user.FirstName != "" || user.LastName != "" {
		results = append(results, fmt.Sprintf("Name: %s %s", user.FirstName, user.LastName))
	}
	if user.SubscriptionTier != "" {
		results = append(results, fmt.Sprintf("Tier: %s", user.SubscriptionTier))
	}
	if user.Timezone != "" {
		results = append(results, fmt.Sprintf("Timezone: %s", user.Timezone))
	}
	results = append(results, fmt.Sprintf("Created: %s", support.FormatTimeValue(user.CreatedAt)))

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Logged in as %s", user.Email)},
		ResultsHeading: "User",
		Results:        results,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func handleAuthResponse(core *cliapp.ScenarioApp, body []byte, summary string, asJSON bool) error {
	var env support.AuthEnvelope
	if err := support.Decode(body, &env); err != nil {
		return err
	}
	if strings.TrimSpace(env.Token) == "" {
		return fmt.Errorf("auth response missing token")
	}

	core.Config.Token = env.Token
	if err := core.SaveConfig(); err != nil {
		return fmt.Errorf("persist token: %w", err)
	}

	report := cliapp.MutationReport{
		Result: []string{
			summary,
			fmt.Sprintf("Logged in as %s", env.User.Email),
		},
		Changes:     []string{"Stored auth token in CLI config"},
		NextCommand: []string{fmt.Sprintf("%s auth whoami", support.CLIName)},
	}
	if asJSON {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

// buildAuthBody merges flag-derived fields with an optional --body-file payload.
// When bodyFile is set, its JSON wins. Otherwise required fields must be
// supplied via flags and non-empty optional fields are included.
func buildAuthBody(bodyFile string, fields map[string]interface{}, required []string) (interface{}, error) {
	raw, err := support.ReadJSONFile(bodyFile, false)
	if err != nil {
		return nil, err
	}
	if raw != nil {
		return raw, nil
	}
	for _, key := range required {
		value, _ := fields[key].(string)
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("--%s is required (or pass --body-file)", strings.ReplaceAll(key, "_", "-"))
		}
	}
	payload := map[string]interface{}{}
	for k, v := range fields {
		switch val := v.(type) {
		case string:
			if strings.TrimSpace(val) != "" {
				payload[k] = val
			}
		default:
			payload[k] = v
		}
	}
	return payload, nil
}
