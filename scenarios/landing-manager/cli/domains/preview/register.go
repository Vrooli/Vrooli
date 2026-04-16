package preview

import (
	"fmt"
	"os"
	"strings"

	"landing-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `preview` as a flat command backed by
// `GET /api/v1/preview/{scenario_id}`.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Preview",
		Commands: []cliapp.Command{
			{
				Name:        "preview",
				Description: "Fetch preview links for a generated scenario",
				NeedsAPI:    true,
				Run:         func(args []string) error { return run(core, args) },
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("preview")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: preview <scenario-id>")
	}
	scenarioID := fs.Arg(0)

	body, err := core.Get("/preview/"+scenarioID, nil)
	if err != nil {
		return err
	}
	var links support.PreviewLinks
	if err := support.Decode(body, &links); err != nil {
		return err
	}

	results := []string{}
	if links.Path != "" {
		results = append(results, "Path: "+links.Path)
	}
	if links.BaseURL != "" {
		results = append(results, "Base URL: "+links.BaseURL)
	}
	if len(links.Links) > 0 {
		keys := make([]string, 0, len(links.Links))
		for k := range links.Links {
			keys = append(keys, k)
		}
		// sorted for stable rendering
		sortStrings(keys)
		results = append(results, "Links:")
		for _, k := range keys {
			results = append(results, fmt.Sprintf("  %s: %s", k, links.Links[k]))
		}
	}
	if len(links.Instructions) > 0 {
		results = append(results, "Instructions:")
		for _, line := range links.Instructions {
			results = append(results, "  - "+line)
		}
	}
	if links.Notes != "" {
		results = append(results, "Notes: "+strings.TrimSpace(links.Notes))
	}
	if len(results) == 0 {
		results = []string{"(no preview data returned)"}
	}

	heading := scenarioID
	if links.ScenarioID != "" {
		heading = links.ScenarioID
	}

	report := cliapp.ListReport{
		Summary:        []string{"Preview for " + heading},
		ResultsHeading: "Preview",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s lifecycle status %s", support.CLIName, scenarioID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func sortStrings(values []string) {
	// simple insertion sort to avoid importing sort just for a handful of keys
	for i := 1; i < len(values); i++ {
		j := i
		for j > 0 && values[j-1] > values[j] {
			values[j-1], values[j] = values[j], values[j-1]
			j--
		}
	}
}
