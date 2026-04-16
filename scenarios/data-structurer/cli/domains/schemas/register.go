package schemas

import (
	"fmt"
	"os"

	"data-structurer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `schemas` subcommand group covering the /api/v1/schemas CRUD
// surface plus the from-template creation endpoint.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "schemas",
		Description: "Manage data schemas",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List active schemas", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one schema", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", Description: "Create a schema from a schema-definition JSON file", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", Description: "Update a schema from a JSON payload (--body-file PATH)", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Soft-delete a schema", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "create-from-template", Description: "Create a schema from a template ID", Run: func(args []string) error { return runCreateFromTemplate(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("schemas list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/schemas", nil)
	if err != nil {
		return err
	}
	var resp support.SchemaListResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Active schemas: %d", resp.Count)},
		ResultsHeading: "Schemas",
		Results:        schemaRows(resp.Schemas),
		RetrievalHints: []string{
			fmt.Sprintf("%s schemas get <schema-id>", support.CLIName),
			fmt.Sprintf("%s process run <schema-id> <input>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("schemas get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: schemas get <schema-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/schemas/"+id, nil)
	if err != nil {
		return err
	}
	var schema support.Schema
	if err := support.Decode(body, &schema); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", schema.ID),
		fmt.Sprintf("Name: %s", schema.Name),
	}
	if schema.Description != "" {
		results = append(results, fmt.Sprintf("Description: %s", schema.Description))
	}
	results = append(results,
		fmt.Sprintf("Version: %d", schema.Version),
		fmt.Sprintf("Active: %t", schema.IsActive),
		fmt.Sprintf("Usage count: %d", schema.UsageCount),
		fmt.Sprintf("Avg confidence: %.2f", schema.AvgConfidence),
		fmt.Sprintf("Created: %s", support.FormatTimePtr(schema.CreatedAt)),
		fmt.Sprintf("Updated: %s", support.FormatTimePtr(schema.UpdatedAt)),
	)
	if schema.CreatedBy != "" {
		results = append(results, fmt.Sprintf("Created by: %s", schema.CreatedBy))
	}
	if len(schema.SchemaDefinition) > 0 {
		results = append(results, "Schema definition:")
		results = append(results, support.MapRows(schema.SchemaDefinition)...)
	}
	if len(schema.ExampleData) > 0 {
		results = append(results, "Example data:")
		results = append(results, support.MapRows(schema.ExampleData)...)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Schema: %s (v%d)", schema.Name, schema.Version)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s process run %s <input>", support.CLIName, schema.ID),
			fmt.Sprintf("%s data %s", support.CLIName, schema.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("schemas create")
	description := fs.String("description", "", "Optional description for the schema")
	bodyFile := fs.String("body-file", "", "Path to a JSON file containing the full create payload; overrides --description and the positional args")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}

	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if fs.NArg() < 2 {
			return fmt.Errorf("usage: schemas create <name> <schema-definition.json> [--description TEXT] | --body-file PATH")
		}
		name := fs.Arg(0)
		schemaPath := fs.Arg(1)
		schemaDef, err := support.ReadJSONFile(schemaPath, true)
		if err != nil {
			return err
		}
		// The API accepts {name, description, schema_definition, example_data?}.
		// example_data is only set via --body-file (requires nested JSON).
		payload = map[string]interface{}{
			"name":              name,
			"description":       *description,
			"schema_definition": schemaDef,
		}
	}

	body, err := core.Request("POST", "/schemas", nil, payload)
	if err != nil {
		return err
	}
	var resp support.SchemaMutationResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	result := []string{
		fmt.Sprintf("Schema created: %s", resp.ID),
	}
	if resp.Name != "" {
		result = append(result, fmt.Sprintf("Name: %s", resp.Name))
	}
	if resp.CreatedAt != nil {
		result = append(result, fmt.Sprintf("Created: %s", support.FormatTimePtr(resp.CreatedAt)))
	}

	report := cliapp.MutationReport{
		Result:  result,
		Changes: []string{fmt.Sprintf("Created schema %s", resp.ID)},
		NextCommand: []string{
			fmt.Sprintf("%s schemas get %s", support.CLIName, resp.ID),
			fmt.Sprintf("%s process run %s <input>", support.CLIName, resp.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("schemas update")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with fields to update (description, schema_definition, example_data, is_active)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: schemas update <schema-id> --body-file PATH")
	}
	id := fs.Arg(0)

	// The API accepts any subset of {description, schema_definition, example_data, is_active};
	// building arbitrary nested JSON from flags is error-prone, so we require --body-file.
	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("PUT", "/schemas/"+id, nil, payload)
	if err != nil {
		return err
	}
	var resp support.SchemaMutationResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	status := resp.Status
	if status == "" {
		status = "updated"
	}

	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Schema %s: %s", id, status)},
		Changes: []string{fmt.Sprintf("Applied update to schema %s", id)},
		NextCommand: []string{
			fmt.Sprintf("%s schemas get %s", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("schemas delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: schemas delete <schema-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("DELETE", "/schemas/"+id, nil, nil)
	if err != nil {
		return err
	}
	var resp support.SchemaMutationResponse
	_ = support.Decode(body, &resp)

	status := resp.Status
	if status == "" {
		status = "deleted"
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Schema %s: %s", id, status)},
		Changes:     []string{fmt.Sprintf("Soft-deleted schema %s (is_active=false)", id)},
		NextCommand: []string{fmt.Sprintf("%s schemas list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runCreateFromTemplate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("schemas create-from-template")
	description := fs.String("description", "", "Optional description for the new schema")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full create payload; overrides --description and positional args")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var (
		templateID string
		payload    interface{}
	)

	if *bodyFile != "" {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: schemas create-from-template <template-id> --body-file PATH")
		}
		templateID = fs.Arg(0)
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if fs.NArg() < 2 {
			return fmt.Errorf("usage: schemas create-from-template <template-id> <name> [--description TEXT]")
		}
		templateID = fs.Arg(0)
		payload = map[string]interface{}{
			"name":        fs.Arg(1),
			"description": *description,
		}
	}

	body, err := core.Request("POST", "/schemas/from-template/"+templateID, nil, payload)
	if err != nil {
		return err
	}
	var resp support.SchemaMutationResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	result := []string{
		fmt.Sprintf("Schema created from template %s", templateID),
		fmt.Sprintf("Schema ID: %s", resp.ID),
	}
	if resp.Name != "" {
		result = append(result, fmt.Sprintf("Name: %s", resp.Name))
	}
	if resp.CreatedAt != nil {
		result = append(result, fmt.Sprintf("Created: %s", support.FormatTimePtr(resp.CreatedAt)))
	}

	report := cliapp.MutationReport{
		Result:  result,
		Changes: []string{fmt.Sprintf("Created schema %s from template %s", resp.ID, templateID)},
		NextCommand: []string{
			fmt.Sprintf("%s schemas get %s", support.CLIName, resp.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func schemaRows(schemas []support.Schema) []string {
	if len(schemas) == 0 {
		return []string{"No schemas registered"}
	}
	rows := make([]string, 0, len(schemas))
	for _, s := range schemas {
		rows = append(rows, fmt.Sprintf("%s | %s | v%d | %d uses | avg-confidence=%.2f",
			support.ShortID(s.ID), s.Name, s.Version, s.UsageCount, s.AvgConfidence))
	}
	return rows
}
