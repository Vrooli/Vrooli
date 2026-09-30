package folder

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"notes/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `folder` subcommand group covering CRUD on /api/folders.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "folder",
		Description: "Create, list, update, and delete folders",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List folders", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", Aliases: []string{"new", "add"}, Description: "Create a folder", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", Aliases: []string{"edit"}, Description: "Update a folder", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", Aliases: []string{"rm", "remove"}, Description: "Delete a folder", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("folder list")
	userID := fs.String("user-id", "", "Filter by user ID (server-side)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"user_id": *userID,
	})
	body, err := core.Get("/folders", query)
	if err != nil {
		return err
	}

	var folders []support.Folder
	if err := support.Decode(body, &folders); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Folders: %d", len(folders))},
		ResultsHeading: "Folders",
		Results:        folderRows(folders),
		RetrievalHints: []string{
			fmt.Sprintf("%s folder create --name \"...\"", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("folder create")
	name := fs.String("name", "", "Folder name (required)")
	icon := fs.String("icon", "", "Icon (emoji)")
	color := fs.String("color", "", "Hex color")
	parentID := fs.String("parent-id", "", "Parent folder ID")
	userID := fs.String("user-id", "", "Owner user ID")
	position := fs.Int("position", 0, "Sort position")
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
			"name":     *name,
			"position": *position,
		}
		if strings.TrimSpace(*icon) != "" {
			body["icon"] = *icon
		}
		if strings.TrimSpace(*color) != "" {
			body["color"] = *color
		}
		if strings.TrimSpace(*parentID) != "" {
			body["parent_id"] = *parentID
		}
		if strings.TrimSpace(*userID) != "" {
			body["user_id"] = *userID
		}
		payload = body
	}

	respBody, err := core.Request("POST", "/folders", nil, payload)
	if err != nil {
		return err
	}
	var created support.Folder
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
		Result:      []string{fmt.Sprintf("Created folder %q", created.Name)},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s folder list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("folder update")
	name := fs.String("name", "", "New name")
	icon := fs.String("icon", "", "New icon")
	color := fs.String("color", "", "New color")
	position := fs.String("position", "", "New sort position (integer)")
	bodyFile := fs.String("body-file", "", "Path to JSON file with the full update payload (overrides flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: folder update <folder-id> [--name ...] [--icon ...] [--color ...] [--position N] [--body-file PATH]")
	}
	id := fs.Arg(0)

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		body := map[string]interface{}{}
		if strings.TrimSpace(*name) != "" {
			body["name"] = *name
		}
		if strings.TrimSpace(*icon) != "" {
			body["icon"] = *icon
		}
		if strings.TrimSpace(*color) != "" {
			body["color"] = *color
		}
		if strings.TrimSpace(*position) != "" {
			n, err := strconv.Atoi(strings.TrimSpace(*position))
			if err != nil {
				return fmt.Errorf("--position must be an integer: %w", err)
			}
			body["position"] = n
		}
		if len(body) == 0 {
			return fmt.Errorf("no fields to update — pass at least one of --name, --icon, --color, --position, or --body-file")
		}
		payload = body
	}

	if _, err := core.Request("PUT", "/folders/"+id, nil, payload); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated folder %s", id)},
		NextCommand: []string{fmt.Sprintf("%s folder list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("folder delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: folder delete <folder-id>")
	}
	id := fs.Arg(0)

	if _, err := core.Request("DELETE", "/folders/"+id, nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted folder %s", id)},
		NextCommand: []string{fmt.Sprintf("%s folder list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func folderRows(folders []support.Folder) []string {
	if len(folders) == 0 {
		return []string{"No folders found"}
	}
	rows := make([]string, 0, len(folders))
	for _, f := range folders {
		icon := f.Icon
		if icon == "" {
			icon = "-"
		}
		color := f.Color
		if color == "" {
			color = "-"
		}
		rows = append(rows, fmt.Sprintf("%s | %s %s [%s] pos=%d",
			support.ShortID(f.ID), icon, f.Name, color, f.Position))
	}
	return rows
}
