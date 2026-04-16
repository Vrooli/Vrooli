package bulk

import (
	"fmt"
	"os"

	"social-media-scheduler/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register wraps the /api/v1/bulk/* endpoints. All three endpoints accept
// deeply nested payloads; hand-building them in Go would duplicate server-side
// validation, so every subcommand requires --body-file.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "bulk",
		Description: "Bulk post operations (all take --body-file PATH)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "schedule", Description: "Schedule many posts at once (--body-file PATH)", Run: func(args []string) error {
				return runBodyFile(core, args, "bulk schedule", "POST", "/bulk/schedule", "Bulk schedule submitted")
			}},
			{Name: "import", Description: "Import posts from an external source (--body-file PATH)", Run: func(args []string) error {
				return runBodyFile(core, args, "bulk import", "POST", "/bulk/import", "Bulk import submitted")
			}},
			{Name: "reschedule", Description: "Reschedule many posts at once (--body-file PATH)", Run: func(args []string) error {
				return runBodyFile(core, args, "bulk reschedule", "PUT", "/bulk/reschedule", "Bulk reschedule submitted")
			}},
		},
	}
}

func runBodyFile(core *cliapp.ScenarioApp, args []string, cmdName, method, path, fallback string) error {
	fs := support.NewFlagSet(cmdName)
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the request payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request(method, path, nil, payload)
	if err != nil {
		return err
	}
	msg := support.EnvelopeMessage(body)
	if msg == "" {
		msg = fallback
	}
	report := cliapp.MutationReport{
		Result:      []string{msg},
		Changes:     []string{fmt.Sprintf("%s %s", method, path)},
		NextCommand: []string{fmt.Sprintf("%s post list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
