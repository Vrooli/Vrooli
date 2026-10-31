package themes

import (
	"fmt"
	"os"
	"strings"

	"bedtime-story-generator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `bedtime-story-generator themes` as a flat command because
// the themes surface is a single read-only list (GET /api/v1/themes).
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Themes",
		Commands: []cliapp.Command{
			{
				Name:        "themes",
				Description: "List available story themes",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runList(core, args) },
			},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("themes")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/themes", nil)
	if err != nil {
		return err
	}
	var themes []support.Theme
	if err := support.Decode(body, &themes); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Themes: %d", len(themes))},
		ResultsHeading: "Available themes",
		Results:        themeRows(themes),
		RetrievalHints: []string{
			fmt.Sprintf("%s stories generate --theme <theme>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func themeRows(themes []support.Theme) []string {
	if len(themes) == 0 {
		return []string{"(no themes configured)"}
	}
	rows := make([]string, 0, len(themes))
	for _, t := range themes {
		line := t.Name
		if t.Emoji != "" {
			line = t.Emoji + " " + line
		}
		if t.Description != "" {
			line += " — " + strings.TrimSpace(t.Description)
		}
		rows = append(rows, line)
	}
	return rows
}
