package oauth

import (
	"fmt"
	"os"
	"strings"

	"scenario-authenticator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `oauth` subcommand group covering providers/login.
// The callback endpoints are browser-only and are intentionally not wrapped.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "oauth",
		Description: "Inspect configured OAuth providers and generate login URLs",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "providers", Description: "List configured OAuth providers", Run: func(args []string) error { return runProviders(core, args) }},
			{Name: "login", Description: "Generate an authorization URL for a provider", Run: func(args []string) error { return runLogin(core, args) }},
		},
	}
}

func runProviders(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("oauth providers")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/auth/oauth/providers", nil)
	if err != nil {
		return err
	}

	var resp support.OAuthProvidersResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Providers))
	for _, p := range resp.Providers {
		results = append(results, fmt.Sprintf("%s (%s) | enabled=%t", p.Display, p.Name, p.Enabled))
	}
	if len(results) == 0 {
		results = []string{"No OAuth providers configured"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Providers: %d", len(resp.Providers))},
		ResultsHeading: "Providers",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("%s oauth login --provider google", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runLogin(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("oauth login")
	provider := fs.String("provider", "", "Provider name (google|github)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*provider) == "" {
		return fmt.Errorf("usage: oauth login --provider <google|github>")
	}

	query := support.BuildQuery(map[string]string{"provider": *provider})
	body, err := core.Get("/auth/oauth/login", query)
	if err != nil {
		return err
	}

	var resp support.OAuthLoginResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Authorization URL for %s", resp.Provider)},
		ResultsHeading: "URL",
		Results:        []string{resp.AuthURL},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
