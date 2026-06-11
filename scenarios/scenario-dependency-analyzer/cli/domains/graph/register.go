package graph

import (
	"fmt"
	"os"
	"strings"

	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Visualization",
		Commands: []cliapp.Command{
			{
				Name:        "graph",
				Description: "Generate dependency graph output",
				NeedsAPI:    true,
				Run: func(args []string) error {
					return run(core, args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	if len(args) > 0 && strings.EqualFold(strings.TrimSpace(args[0]), "centrality") {
		return runCentrality(core, args[1:])
	}

	fs := support.NewFlagSet("graph")
	var graphType string
	var format string
	var outputPath string
	var jsonOutput bool
	fs.StringVar(&graphType, "type", "combined", "Graph type")
	fs.StringVar(&format, "format", "json", "Output format: json, dot, mermaid")
	fs.StringVar(&outputPath, "output", "", "Write output to a file")
	fs.StringVar(&outputPath, "o", "", "Write output to a file")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) > 1 {
		return fmt.Errorf("usage: %s graph [type] [--format json|dot|mermaid] [--output file] [--json]", support.AppName)
	}
	if len(positionals) == 1 {
		graphType = positionals[0]
	}
	resolvedType, err := support.GraphType(graphType)
	if err != nil {
		return err
	}
	body, err := core.Get("/graph/"+resolvedType, nil)
	if err != nil {
		return err
	}
	if jsonOutput {
		return support.PrintAPIJSON(body)
	}
	rendered, err := convert(body, format)
	if err != nil {
		return err
	}
	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(rendered), 0o644); err != nil {
			return fmt.Errorf("write graph output: %w", err)
		}
		report := cliapp.MutationReport{
			Result:  []string{fmt.Sprintf("Exported %s graph.", resolvedType)},
			Changes: []string{fmt.Sprintf("Wrote %s output to %s", format, outputPath)},
			NextCommand: []string{
				fmt.Sprintf("%s graph %s --format %s", support.AppName, resolvedType, format),
			},
		}
		return support.PrintMutation(false, report, nil)
	}
	if format != "json" {
		fmt.Fprintln(os.Stdout, rendered)
		return nil
	}

	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	nodes := support.Maps(resp["nodes"])
	edges := support.Maps(resp["edges"])
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Graph type: %s", resolvedType),
			fmt.Sprintf("Nodes: %d", len(nodes)),
			fmt.Sprintf("Edges: %d", len(edges)),
		},
		ResultsHeading: "Sample Nodes",
		Results:        sampleNodes(nodes),
		RetrievalHints: []string{
			fmt.Sprintf("%s graph %s --json", support.AppName, resolvedType),
			fmt.Sprintf("%s graph %s --format dot --output deps.dot", support.AppName, resolvedType),
			fmt.Sprintf("%s cycles --type %s", support.AppName, resolvedType),
		},
	}
	return support.PrintList(false, report, nil)
}

func runCentrality(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("graph centrality")
	var scenario string
	var jsonOutput bool
	fs.StringVar(&scenario, "scenario", "", "Limit centrality to one scenario")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) > 1 {
		return fmt.Errorf("usage: %s graph centrality [scenario] [--scenario name] [--json]", support.AppName)
	}
	if len(positionals) == 1 {
		if strings.TrimSpace(scenario) != "" {
			return fmt.Errorf("provide scenario either positionally or with --scenario, not both")
		}
		scenario = positionals[0]
	}

	query := support.BuildQuery(map[string]string{"scenario": scenario})
	body, err := core.Get("/graph/centrality", query)
	if err != nil {
		return err
	}
	if jsonOutput {
		return support.PrintAPIJSON(body)
	}

	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	nodes := support.Maps(resp["nodes"])
	summary := []string{
		fmt.Sprintf("Centrality rows: %d", len(nodes)),
		fmt.Sprintf("Graph type: %s", support.String(resp["graph_type"])),
	}
	if scenario := support.String(resp["scenario"]); scenario != "" {
		summary = append(summary, fmt.Sprintf("Scenario filter: %s", scenario))
	}
	hints := []string{fmt.Sprintf("%s graph centrality --json", support.AppName)}
	if strings.TrimSpace(scenario) != "" {
		hints = append(hints, fmt.Sprintf("%s graph centrality --scenario %s --json", support.AppName, strings.TrimSpace(scenario)))
	}
	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Centrality",
		Results:        centralityRows(nodes),
		RetrievalHints: hints,
	}
	return support.PrintList(false, report, nil)
}

func centralityRows(nodes []map[string]interface{}) []string {
	limit := 10
	if len(nodes) < limit {
		limit = len(nodes)
	}
	out := make([]string, 0, limit)
	for _, node := range nodes[:limit] {
		nearest := support.String(node["nearest_core_seed"])
		distance := support.Int(node["distance_to_core_seed"])
		coreText := "unreachable from core seed"
		if distance >= 0 && nearest != "" {
			coreText = fmt.Sprintf("core distance %d via %s", distance, nearest)
		}
		out = append(out, fmt.Sprintf(
			"%s: reverse=%d transitive=%d required=%d weighted=%.1f, %s",
			support.String(node["scenario"]),
			support.Int(node["direct_reverse_dependency_count"]),
			support.Int(node["transitive_reverse_dependency_count"]),
			support.Int(node["required_reverse_dependency_count"]),
			support.Float(node["required_edge_weighted_score"]),
			coreText,
		))
	}
	return out
}

func convert(body []byte, format string) (string, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "json"
	}
	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return "", err
	}
	nodes := support.Maps(resp["nodes"])
	edges := support.Maps(resp["edges"])

	switch format {
	case "json":
		return string(body), nil
	case "dot":
		lines := []string{"digraph Dependencies {", "  rankdir=LR;", "  node [shape=box];"}
		for _, node := range nodes {
			lines = append(lines, fmt.Sprintf("  %s [label=\"%s\"];", sanitizeID(support.String(node["id"])), support.String(node["label"])))
		}
		for _, edge := range edges {
			lines = append(lines, fmt.Sprintf("  %s -> %s [label=\"%s\"];", sanitizeID(support.String(edge["source"])), sanitizeID(support.String(edge["target"])), support.String(edge["label"])))
		}
		lines = append(lines, "}")
		return strings.Join(lines, "\n"), nil
	case "mermaid":
		lines := []string{"graph TD"}
		for _, edge := range edges {
			lines = append(lines, fmt.Sprintf("  %s[%s] --> %s[%s]", sanitizeID(support.String(edge["source"])), support.String(edge["source"]), sanitizeID(support.String(edge["target"])), support.String(edge["target"])))
		}
		return strings.Join(lines, "\n"), nil
	default:
		return "", fmt.Errorf("unsupported graph format %q; valid formats: json, dot, mermaid", format)
	}
}

func sanitizeID(value string) string {
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, ".", "_")
	return value
}

func sampleNodes(nodes []map[string]interface{}) []string {
	limit := 8
	if len(nodes) < limit {
		limit = len(nodes)
	}
	out := make([]string, 0, limit)
	for _, node := range nodes[:limit] {
		out = append(out, fmt.Sprintf("%s (%s)", support.String(node["label"]), support.String(node["type"])))
	}
	return out
}
