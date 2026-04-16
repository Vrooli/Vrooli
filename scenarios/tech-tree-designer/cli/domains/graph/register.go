package graph

import (
	"fmt"
	"os"
	"strings"

	"tech-tree-designer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "graph",
		Description: "Inspect exported graph views, dependencies, and connections",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "export", Description: "Export the graph as DOT", Run: func(args []string) error { return runExport(deps, args) }},
			{Name: "dependencies", Description: "List stage dependencies", Run: func(args []string) error { return runDependencies(deps, args) }},
			{Name: "connections", Description: "List cross-sector connections", Run: func(args []string) error { return runConnections(deps, args) }},
		},
	}
}

func runExport(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("graph export")
	format := fs.String("format", "dot", "Export format")
	output := fs.String("output", "", "Output file")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *format != "dot" {
		return fmt.Errorf("only DOT export is supported right now")
	}
	body, err := deps.Get("/tech-tree/graph/dot", nil)
	if err != nil {
		return err
	}
	if strings.TrimSpace(*output) != "" {
		if err := os.WriteFile(*output, body, 0o644); err != nil {
			return err
		}
	}
	report := cliapp.MutationReport{
		Result: []string{"Exported the graph as DOT."},
		Changes: []string{
			support.TreeScopeLine(deps.Selector),
			fmt.Sprintf("Bytes: %d", len(body)),
		},
		NextCommand: []string{
			"tech-tree-designer graph dependencies",
			"tech-tree-designer graph connections",
		},
	}
	if strings.TrimSpace(*output) != "" {
		report.Changes = append(report.Changes, "Output file: "+*output)
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, map[string]interface{}{
			"format": "dot",
			"bytes":  len(body),
			"output": *output,
			"data":   string(body),
		})
	}
	if strings.TrimSpace(*output) == "" {
		if _, err := os.Stdout.Write(body); err != nil {
			return err
		}
		if len(body) == 0 || body[len(body)-1] != '\n' {
			_, _ = fmt.Fprintln(os.Stdout)
		}
		return nil
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDependencies(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("graph dependencies")
	bottlenecks := fs.Bool("bottlenecks", false, "Only show strong dependencies")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := deps.Get("/dependencies", nil)
	if err != nil {
		return err
	}
	var response support.DependenciesResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	results := dependencyRows(response.Dependencies, *bottlenecks)
	report := cliapp.ListReport{
		Summary: []string{
			support.TreeScopeLine(deps.Selector),
			fmt.Sprintf("Dependencies returned: %d", len(results)),
		},
		ResultsHeading: "Dependencies",
		Results:        results,
		RetrievalHints: []string{
			"tech-tree-designer graph dependencies --bottlenecks",
			"tech-tree-designer stages get <stage-id>",
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runConnections(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("graph connections")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := deps.Get("/connections", nil)
	if err != nil {
		return err
	}
	var response support.ConnectionsResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary: []string{
			support.TreeScopeLine(deps.Selector),
			fmt.Sprintf("Connections returned: %d", len(response.Connections)),
		},
		ResultsHeading: "Cross-sector connections",
		Results:        connectionRows(response.Connections),
		RetrievalHints: []string{
			"tech-tree-designer overview --verbose",
			"tech-tree-designer graph export --output tech-tree.dot",
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func dependencyRows(items []support.DependencyEntry, bottlenecks bool) []string {
	rows := make([]string, 0, len(items))
	for _, item := range items {
		if bottlenecks && item.Dependency.DependencyStrength < 0.8 {
			continue
		}
		rows = append(rows, fmt.Sprintf("%s <- %s | %s | strength %.0f%%", item.DependentName, item.PrerequisiteName, item.Dependency.DependencyType, item.Dependency.DependencyStrength*100))
	}
	if len(rows) == 0 {
		return []string{"No dependencies matched the current filter."}
	}
	return rows
}

func connectionRows(items []support.ConnectionEntry) []string {
	if len(items) == 0 {
		return []string{"No cross-sector connections found."}
	}
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, fmt.Sprintf("%s -> %s | %s | strength %.0f%%", item.SourceName, item.TargetName, item.Connection.ConnectionType, item.Connection.Strength*100))
	}
	return rows
}
