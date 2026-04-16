package oauth

import (
	"fmt"
	"os"

	"social-media-scheduler/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register wraps /api/v1/oauth/:platform/*. The full interactive OAuth flow
// (redirect + browser callback) is finished in the UI; the CLI surfaces the
// connect URL and the disconnect action so scripts can drive them.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "oauth",
		Description: "Manage social platform OAuth connections",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "connect", Description: "Retrieve the OAuth redirect URL for a platform", Run: func(args []string) error { return runConnect(core, args) }},
			{Name: "disconnect", Description: "Disconnect the account for a platform", Run: func(args []string) error { return runDisconnect(core, args) }},
		},
	}
}

func runConnect(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("oauth connect")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: oauth connect <platform>")
	}
	platform := fs.Arg(0)

	body, err := core.Get("/oauth/"+platform+"/connect", nil)
	if err != nil {
		return err
	}
	var generic map[string]interface{}
	_ = support.Decode(body, &generic)

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("OAuth connect payload for %s", platform)},
		ResultsHeading: "Payload",
		Results:        support.MapRows(generic),
		RetrievalHints: []string{fmt.Sprintf("%s oauth disconnect %s", support.CLIName, platform)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runDisconnect(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("oauth disconnect")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: oauth disconnect <platform>")
	}
	platform := fs.Arg(0)

	body, err := core.Request("DELETE", "/oauth/"+platform+"/disconnect", nil, nil)
	if err != nil {
		return err
	}
	msg := support.EnvelopeMessage(body)
	if msg == "" {
		msg = fmt.Sprintf("Disconnected %s", platform)
	}
	report := cliapp.MutationReport{
		Result:      []string{msg},
		Changes:     []string{fmt.Sprintf("Removed OAuth connection: %s", platform)},
		NextCommand: []string{fmt.Sprintf("%s accounts", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
