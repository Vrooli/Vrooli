package tag

import (
	"fmt"
	"os"
	"strings"

	"notes/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `tag` subcommand group covering /api/tags endpoints.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "tag",
		Description: "List and create tags",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List tags", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", Aliases: []string{"new", "add"}, Description: "Create a tag", Run: func(args []string) error { return runCreate(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tag list")
	userID := fs.String("user-id", "", "Filter by user ID (server-side)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"user_id": *userID,
	})
	body, err := core.Get("/tags", query)
	if err != nil {
		return err
	}

	var tags []support.Tag
	if err := support.Decode(body, &tags); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Tags: %d", len(tags))},
		ResultsHeading: "Tags",
		Results:        tagRows(tags),
		RetrievalHints: []string{
			fmt.Sprintf("%s tag create --name \"...\"", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tag create")
	name := fs.String("name", "", "Tag name (required)")
	color := fs.String("color", "", "Hex color")
	userID := fs.String("user-id", "", "Owner user ID")
	bodyFile := fs.String("body-file", "", "Path to JSON file with the full create payload (overrides flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if strings.TrimSpace(*name) == "" {
			return fmt.Errorf("--name is required (or use --body-file)")
		}
		body := map[string]interface{}{
			"name": *name,
		}
		if strings.TrimSpace(*color) != "" {
			body["color"] = *color
		}
		if strings.TrimSpace(*userID) != "" {
			body["user_id"] = *userID
		}
		payload = body
	}

	respBody, err := core.Request("POST", "/tags", nil, payload)
	if err != nil {
		return err
	}
	var created support.Tag
	if err := support.Decode(respBody, &created); err != nil {
		return err
	}

	changes := []string{}
	if created.Name != "" {
		changes = append(changes, fmt.Sprintf("Name: %s", created.Name))
	}
	if created.ID != "" {
		changes = append(changes, fmt.Sprintf("ID: %s", created.ID))
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Created tag %q", created.Name)},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s tag list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func tagRows(tags []support.Tag) []string {
	if len(tags) == 0 {
		return []string{"No tags found"}
	}
	rows := make([]string, 0, len(tags))
	for _, t := range tags {
		color := t.Color
		if color == "" {
			color = "-"
		}
		rows = append(rows, fmt.Sprintf("%s | %s (%d uses) [%s]",
			support.ShortID(t.ID), t.Name, t.UsageCount, color))
	}
	return rows
}
