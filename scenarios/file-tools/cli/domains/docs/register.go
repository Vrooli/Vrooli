package docs

import (
	"fmt"
	"os"

	"file-tools/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `file-tools docs` as a flat command. The API serves
// `/docs` at the root (outside `/api/v1`), so we use GetRoot.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Docs",
		Commands: []cliapp.Command{
			{
				Name:        "docs",
				Description: "Show API documentation index",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runDocs(core, args) },
			},
		},
	}
}

func runDocs(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("docs")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.GetRoot("/docs", nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	name, _ := data["name"].(string)
	version, _ := data["version"].(string)
	description, _ := data["description"].(string)

	summary := []string{}
	if name != "" {
		if version != "" {
			summary = append(summary, fmt.Sprintf("%s v%s", name, version))
		} else {
			summary = append(summary, name)
		}
	}
	if description != "" {
		summary = append(summary, description)
	}

	results := endpointRows(data["endpoints"])
	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Endpoints",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s health", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func endpointRows(endpoints interface{}) []string {
	list, ok := endpoints.([]interface{})
	if !ok || len(list) == 0 {
		return []string{"(no endpoints reported)"}
	}
	rows := make([]string, 0, len(list))
	for _, item := range list {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		method, _ := entry["method"].(string)
		path, _ := entry["path"].(string)
		desc, _ := entry["description"].(string)
		line := fmt.Sprintf("%-6s %s", method, path)
		if desc != "" {
			line += " -- " + desc
		}
		rows = append(rows, line)
	}
	if len(rows) == 0 {
		return []string{"(no endpoints reported)"}
	}
	return rows
}
