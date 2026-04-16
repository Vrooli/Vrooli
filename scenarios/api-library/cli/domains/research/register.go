package research

import (
	"fmt"
	"os"
	"strings"

	"api-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `api-library research <capability>` as a flat command that
// POSTs to /request-research. Requirements (nested JSON) are accepted through
// --body-file to avoid hand-building JSON on the CLI surface.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Research",
		Commands: []cliapp.Command{
			{
				Name:        "research",
				Description: "Request research for a new API capability",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runResearch(core, args) },
			},
		},
	}
}

func runResearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("research")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with {capability, requirements}")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var body interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		body = raw
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: research <capability> | research --body-file PATH")
		}
		capability := strings.Join(fs.Args(), " ")
		body = map[string]interface{}{
			"capability":   capability,
			"requirements": map[string]interface{}{},
		}
	}

	raw, err := core.Request("POST", "/request-research", nil, body)
	if err != nil {
		return err
	}
	var resp support.ResearchResponse
	if err := support.Decode(raw, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Research ID: %s", resp.ResearchID),
			fmt.Sprintf("Status: %s", resp.Status),
			fmt.Sprintf("Estimated time: %d seconds", resp.EstimatedTime),
		},
		Changes: []string{"Queued research request"},
		NextCommand: []string{
			fmt.Sprintf("%s search <capability>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
