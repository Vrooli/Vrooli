package schemes

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"stream-of-consciousness-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "stream-of-consciousness-analyzer"

type scheme struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type schemeExport struct {
	Scheme struct {
		Name string `json:"name"`
	} `json:"scheme"`
	Information []json.RawMessage `json:"information"`
	Thoughts    []json.RawMessage `json:"thoughts"`
	Edges       []json.RawMessage `json:"edges"`
	Format      string            `json:"export_format"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "scheme",
		Description: "Manage thought schemes",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List schemes", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", NeedsAPI: true, Description: "Get a scheme", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", NeedsAPI: true, Description: "Create a scheme", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", NeedsAPI: true, Description: "Update a scheme", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", NeedsAPI: true, Description: "Delete a scheme", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "export", NeedsAPI: true, Description: "Export a scheme graph", Run: func(args []string) error { return runExport(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs, jsonOut, err := support.ParseFlags("scheme list", args)
	if err != nil {
		return err
	}
	_ = fs

	body, err := core.Get("/schemes", nil)
	if err != nil {
		return err
	}

	var items []scheme
	if err := support.Unmarshal(body, &items); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{fmt.Sprintf("Total schemes: %d", len(items))},
		Results: renderList(items),
		RetrievalHints: []string{
			cliName + " scheme get <scheme-id>",
			cliName + " scheme create --name \"Design review\"",
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs, jsonOut, err := support.ParseFlags("scheme get", args)
	if err != nil {
		return err
	}
	if err := support.RequireArg(fs, "scheme get <id> [--json]"); err != nil {
		return err
	}

	body, err := core.Get("/schemes/"+fs.Arg(0), nil)
	if err != nil {
		return err
	}

	var item scheme
	if err := support.Unmarshal(body, &item); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Scheme loaded", "Scheme ID: " + item.ID},
		ResultsHeading: "Details",
		Results: []string{
			"Name: " + item.Name,
			"Created: " + item.CreatedAt,
		},
		RetrievalHints: []string{
			cliName + " scheme export " + item.ID,
			cliName + " thought list --scheme " + item.ID,
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("scheme create", flag.ContinueOnError)
	name := fs.String("name", "", "Scheme name (required)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *name == "" {
		if fs.NArg() > 0 {
			*name = fs.Arg(0)
		} else {
			return fmt.Errorf("usage: scheme create <name> or --name NAME")
		}
	}

	body, err := core.Request("POST", "/schemes", nil, map[string]string{"name": *name})
	if err != nil {
		return err
	}

	var item scheme
	if err := support.Unmarshal(body, &item); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"Scheme created", "Scheme ID: " + item.ID},
		Changes: []string{
			"Name: " + item.Name,
		},
		NextCommand: []string{
			cliName + " scheme get " + item.ID,
			cliName + " info create --scheme " + item.ID + " --content \"...\"",
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("scheme update", flag.ContinueOnError)
	name := fs.String("name", "", "New scheme name (required)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := support.RequireArg(fs, "scheme update <id> --name NAME [--json]"); err != nil {
		return err
	}
	if *name == "" {
		return fmt.Errorf("--name is required")
	}

	body, err := core.Request("PUT", "/schemes/"+fs.Arg(0), nil, map[string]string{"name": *name})
	if err != nil {
		return err
	}

	var item scheme
	if err := support.Unmarshal(body, &item); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"Scheme updated", "Scheme ID: " + item.ID},
		Changes: []string{
			"Name: " + item.Name,
		},
		NextCommand: []string{
			cliName + " scheme get " + item.ID,
			cliName + " scheme export " + item.ID,
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs, jsonOut, err := support.ParseFlags("scheme delete", args)
	if err != nil {
		return err
	}
	if err := support.RequireArg(fs, "scheme delete <id> [--json]"); err != nil {
		return err
	}

	if _, err := core.Request("DELETE", "/schemes/"+fs.Arg(0), nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Scheme deleted", "Scheme ID: " + fs.Arg(0)},
		Changes:     []string{"Removed scheme graph and associated references."},
		NextCommand: []string{cliName + " scheme list"},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runExport(core *cliapp.ScenarioApp, args []string) error {
	fs, jsonOut, err := support.ParseFlags("scheme export", args)
	if err != nil {
		return err
	}
	if err := support.RequireArg(fs, "scheme export <id> [--json]"); err != nil {
		return err
	}

	body, err := core.Get("/schemes/"+fs.Arg(0)+"/export", nil)
	if err != nil {
		return err
	}

	var export schemeExport
	if err := support.Unmarshal(body, &export); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			"Scheme export ready",
			"Scheme: " + export.Scheme.Name,
		},
		ResultsHeading: "Export Contents",
		Results: []string{
			fmt.Sprintf("Format: %s", export.Format),
			fmt.Sprintf("Information items: %d", len(export.Information)),
			fmt.Sprintf("Thoughts: %d", len(export.Thoughts)),
			fmt.Sprintf("Edges: %d", len(export.Edges)),
		},
		RetrievalHints: []string{
			cliName + " scheme get " + fs.Arg(0),
			cliName + " suggestion generate " + fs.Arg(0),
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func renderList(items []scheme) []string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("%s  %s", item.ID, item.Name))
	}
	return lines
}
