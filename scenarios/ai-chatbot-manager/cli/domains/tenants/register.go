package tenants

import (
	"fmt"
	"os"

	"ai-chatbot-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register wires `ai-chatbot-manager tenant ...` covering the multi-tenant
// endpoints (/api/v1/tenants). Create accepts simple flags or --body-file.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "tenant",
		Description: "Manage tenants",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "create", Description: "Create a tenant", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show tenant details + usage", Run: func(args []string) error { return runGet(core, args) }},
		},
	}
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tenant create")
	name := fs.String("name", "", "Tenant name")
	slug := fs.String("slug", "", "Tenant slug")
	description := fs.String("description", "", "Tenant description")
	plan := fs.String("plan", "", "Plan tier (e.g. free, pro)")
	bodyFile := fs.String("body-file", "", "Path to tenant create JSON (overrides flags)")
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
		if *name == "" || *slug == "" {
			return fmt.Errorf("usage: tenant create --name NAME --slug SLUG [--description TEXT] [--plan PLAN] | --body-file PATH")
		}
		req := map[string]interface{}{
			"name": *name,
			"slug": *slug,
		}
		if *description != "" {
			req["description"] = *description
		}
		if *plan != "" {
			req["plan"] = *plan
		}
		payload = req
	}

	body, err := core.Request("POST", "/tenants", nil, payload)
	if err != nil {
		return err
	}

	var resp struct {
		Tenant  map[string]interface{} `json:"tenant"`
		Message string                 `json:"message"`
	}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	message := resp.Message
	if message == "" {
		message = "Tenant created"
	}
	result := []string{message}
	result = append(result, support.MapRows(resp.Tenant)...)

	tenantID, _ := resp.Tenant["id"].(string)

	report := cliapp.MutationReport{
		Result:      result,
		Changes:     []string{message},
		NextCommand: []string{fmt.Sprintf("%s tenant get %s", support.CLIName, tenantID)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tenant get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: tenant get <tenant-id-or-slug>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/tenants/"+id, nil)
	if err != nil {
		return err
	}

	var resp struct {
		Tenant map[string]interface{} `json:"tenant"`
		Usage  map[string]interface{} `json:"usage"`
	}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := []string{"=== Tenant ==="}
	results = append(results, support.MapRows(resp.Tenant)...)
	if len(resp.Usage) > 0 {
		results = append(results, "=== Usage ===")
		results = append(results, support.MapRows(resp.Usage)...)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Tenant: %s", id)},
		ResultsHeading: "Details",
		Results:        results,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
