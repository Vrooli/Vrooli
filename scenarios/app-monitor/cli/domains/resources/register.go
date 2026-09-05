package resources

import (
	"fmt"
	"os"

	"app-monitor/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `resource` subcommand group for Vrooli resource lifecycle.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "resource",
		Description: "Inspect and control shared local resources",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List resources", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one resource", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "status", Description: "Show resource status", Run: func(args []string) error { return runStatus(core, args) }},
			{Name: "start", Description: "Start a resource", Run: func(args []string) error { return runAction(core, args, "start", "Start") }},
			{Name: "stop", Description: "Stop a resource", Run: func(args []string) error { return runAction(core, args, "stop", "Stop") }},
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
	var resources []map[string]interface{}
	if err := support.Decode(body, &resources); err != nil {
		return err
	}

	rows := make([]string, 0, len(resources))
	for _, r := range resources {
		id, _ := r["id"].(string)
		name, _ := r["name"].(string)
		rtype, _ := r["type"].(string)
		status, _ := r["status"].(string)
		rows = append(rows, fmt.Sprintf("%s | %s | type=%s | status=%s", id, name, rtype, status))
	}
	if len(rows) == 0 {
		rows = []string{"(no resources)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Resources: %d", len(resources))},
		ResultsHeading: "Resources",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s resource get <resource-id>", support.CLIName)},
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
		RetrievalHints: []string{fmt.Sprintf("%s resource status %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("resource status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: resource status <resource-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/resources/"+id+"/status", nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status:    []string{fmt.Sprintf("Resource %s status", id)},
		Triage:    []cliapp.TriageGroup{{Heading: "Fields", Items: support.MapRows(data)}},
		NextSteps: []string{fmt.Sprintf("%s resource get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runAction(core *cliapp.ScenarioApp, args []string, verb, display string) error {
	fs := support.NewFlagSet("resource " + verb)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: resource %s <resource-id>", verb)
	}
	id := fs.Arg(0)

	body, err := core.Request("POST", "/resources/"+id+"/"+verb, nil, nil)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("%s issued for resource %s", display, id)
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{fmt.Sprintf("Resource %s: %s", id, verb)},
		NextCommand: []string{fmt.Sprintf("%s resource status %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
