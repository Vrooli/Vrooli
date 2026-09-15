package resource

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"data-tools/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `resource` subcommand group for the CRUD surface on
// /api/v1/resources. The bash CLI exposed these as top-level `list/get/create/
// update/delete` verbs; consolidating them under `resource` matches the
// scenario-CLI conventions and the REST path shape.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "resource",
		Description: "Manage data-tools resources",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List resources", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one resource", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", Aliases: []string{"add"}, Description: "Create a resource", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", Aliases: []string{"edit"}, Description: "Update a resource", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", Aliases: []string{"remove", "rm"}, Description: "Delete a resource", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("resource list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/resources", nil)
	if err != nil {
		return err
	}
	var resources []support.Resource
	if err := support.Decode(body, &resources); err != nil {
		return err
	}

	rows := make([]string, 0, len(resources))
	for _, r := range resources {
		desc := r.Description
		if desc == "" {
			desc = "(no description)"
		}
		rows = append(rows, fmt.Sprintf("%s | %s | %s", support.ShortID(r.ID), r.Name, desc))
	}
	if len(rows) == 0 {
		rows = []string{"No resources found"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Resources: %d", len(resources))},
		ResultsHeading: "Resources",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s resource get <resource-id>", support.CLIName),
			fmt.Sprintf("%s resource create --name <name>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("resource get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: resource get <resource-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/resources/"+id, nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Resource: %s", id)},
		ResultsHeading: "Details",
		Results:        support.MapRows(data),
		RetrievalHints: []string{
			fmt.Sprintf("%s resource update %s --name <name>", support.CLIName, id),
			fmt.Sprintf("%s resource delete %s", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("resource create")
	name := fs.String("name", "", "Resource name (required)")
	description := fs.String("description", "", "Resource description")
	configFile := fs.String("config-file", "", "Path to JSON config file")
	bodyFile := fs.String("body-file", "", "Path to JSON file containing the full request body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := buildResourcePayload(*name, *description, *configFile, *bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/resources", nil, payload)
	if err != nil {
		return err
	}
	var created map[string]interface{}
	if err := support.Decode(body, &created); err != nil {
		return err
	}

	id, _ := created["id"].(string)
	display := id
	if display == "" {
		display = "<unknown>"
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Created resource: %s", display)},
		Changes:     support.MapRows(created),
		NextCommand: []string{fmt.Sprintf("%s resource get %s", support.CLIName, display)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("resource update")
	name := fs.String("name", "", "New resource name")
	description := fs.String("description", "", "New resource description")
	configFile := fs.String("config-file", "", "Path to JSON config file")
	bodyFile := fs.String("body-file", "", "Path to JSON file containing the full request body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: resource update <resource-id> [--name ...] [--description ...] [--config-file PATH | --body-file PATH]")
	}
	id := fs.Arg(0)

	payload, err := buildResourcePayload(*name, *description, *configFile, *bodyFile, false)
	if err != nil {
		return err
	}
	if len(payload) == 0 {
		return fmt.Errorf("no fields to update — provide --name, --description, --config-file, or --body-file")
	}

	if _, err := core.Request("PUT", "/resources/"+id, nil, payload); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated resource: %s", id)},
		Changes:     mutationFields(payload),
		NextCommand: []string{fmt.Sprintf("%s resource get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("resource delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: resource delete <resource-id>")
	}
	id := fs.Arg(0)

	if _, err := core.Request("DELETE", "/resources/"+id, nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted resource: %s", id)},
		Changes:     []string{fmt.Sprintf("Resource %s removed", id)},
		NextCommand: []string{fmt.Sprintf("%s resource list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

// buildResourcePayload produces the request body. When --body-file is provided
// it is used verbatim (must decode to a JSON object). Otherwise the payload is
// built from the individual flags; for create, --name is required.
func buildResourcePayload(name, description, configFile, bodyFile string, nameRequired bool) (map[string]interface{}, error) {
	if strings.TrimSpace(bodyFile) != "" {
		raw, err := support.ReadJSONFile(bodyFile, true)
		if err != nil {
			return nil, err
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, fmt.Errorf("--body-file must contain a JSON object: %w", err)
		}
		return parsed, nil
	}

	payload := map[string]interface{}{}
	if strings.TrimSpace(name) != "" {
		payload["name"] = name
	} else if nameRequired {
		return nil, fmt.Errorf("--name is required (or provide --body-file)")
	}
	if strings.TrimSpace(description) != "" {
		payload["description"] = description
	}
	if strings.TrimSpace(configFile) != "" {
		config, err := support.ReadJSONFile(configFile, true)
		if err != nil {
			return nil, err
		}
		payload["config"] = config
	}
	return payload, nil
}

func mutationFields(payload map[string]interface{}) []string {
	if len(payload) == 0 {
		return []string{"(no fields changed)"}
	}
	rows := make([]string, 0, len(payload))
	for k, v := range payload {
		rows = append(rows, fmt.Sprintf("%s: %s", k, support.RenderValue(v)))
	}
	return rows
}
