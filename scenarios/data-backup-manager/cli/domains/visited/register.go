package visited

import (
	"fmt"
	"os"

	"data-backup-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `visited` for integration with the visited-tracker scenario.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "visited",
		Description: "Record and fetch visited-tracker entries",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "record", Description: "Record a visit (payload from --body-file)", Run: func(args []string) error { return runRecord(core, args) }},
			{Name: "next", Description: "Fetch the next target to visit", Run: func(args []string) error { return runNext(core, args) }},
		},
	}
}

func runRecord(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("visited record")
	bodyFile := fs.String("body-file", "", "Path to a JSON file containing the visit record (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/visited/record", nil, payload)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = "Visit recorded"
	}

	mutation := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     support.MapRows(resp),
		NextCommand: []string{fmt.Sprintf("%s visited next", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, mutation)
	}
	return cliapp.RenderMutationReport(os.Stdout, mutation)
}

func runNext(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("visited next")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/visited/next", nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Next visit target"},
		ResultsHeading: "Details",
		Results:        support.MapRows(payload),
		RetrievalHints: []string{fmt.Sprintf("%s visited record --body-file ./visit.json", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
