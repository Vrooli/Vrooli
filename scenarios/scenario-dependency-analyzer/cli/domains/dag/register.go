package dag

import (
	"fmt"
	"os"
	"strings"

	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "dag",
		Description: "Export recursive dependency DAGs",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "export",
				Description: "Export a scenario dependency DAG",
				Run: func(args []string) error {
					return runExport(core, args)
				},
			},
		},
	}
}

func runExport(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("dag export")
	var recursive bool
	var jsonOutput bool
	var outputPath string
	var refresh bool
	fs.BoolVar(&recursive, "recursive", true, "Include recursive dependencies")
	fs.BoolVar(&jsonOutput, "json", false, "Output JSON")
	fs.StringVar(&outputPath, "output", "", "Write output to a file")
	fs.StringVar(&outputPath, "o", "", "Write output to a file")
	fs.BoolVar(&refresh, "refresh", false, "Refresh deployment report first")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 {
		return fmt.Errorf("usage: %s dag export <scenario> [--recursive=false] [--output file] [--json]", support.AppName)
	}
	scenario := positionals[0]
	query := support.BuildQuery(map[string]string{
		"recursive": support.BoolWord(recursive, "true", "false"),
		"format":    "json",
		"refresh":   support.BoolWord(refresh, "true", ""),
	})
	body, err := core.Get("/scenarios/"+scenario+"/dag/export", query)
	if err != nil {
		return err
	}

	if outputPath != "" {
		if err := os.WriteFile(outputPath, body, 0o644); err != nil {
			return fmt.Errorf("write dag output: %w", err)
		}
		report := cliapp.MutationReport{
			Result:  []string{fmt.Sprintf("Exported DAG for %s.", scenario)},
			Changes: []string{fmt.Sprintf("Wrote %s", outputPath)},
			NextCommand: []string{
				fmt.Sprintf("jq '.' %s", outputPath),
			},
		}
		return support.PrintMutation(false, report, nil)
	}
	if jsonOutput {
		return support.PrintAPIJSON(body)
	}

	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	dag := support.Maps(resp["dag"])
	lines := make([]string, 0, len(dag))
	for _, node := range dag {
		lines = append(lines, renderNode(node, ""))
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenario: %s", scenario),
			fmt.Sprintf("Recursive export: %t", recursive),
			fmt.Sprintf("Root dependency count: %d", len(dag)),
		},
		ResultsHeading: "Dependency Tree",
		Results:        lines,
		RetrievalHints: []string{
			fmt.Sprintf("%s dag export %s --json", support.AppName, scenario),
			fmt.Sprintf("%s deployment %s", support.AppName, scenario),
		},
	}
	if gaps := support.Map(resp["metadata_gaps"]); gaps != nil && support.Int(gaps["total_gaps"]) > 0 {
		report.RetrievalHints = append(report.RetrievalHints, fmt.Sprintf("Metadata gaps: %d", support.Int(gaps["total_gaps"])))
	}
	return support.PrintList(false, report, nil)
}

func renderNode(node map[string]interface{}, indent string) string {
	name := support.String(node["name"])
	kind := support.String(node["type"])
	line := strings.TrimSpace(indent + "- " + kind + ": " + name)
	children := support.Maps(node["children"])
	if len(children) == 0 {
		return line
	}
	childLines := []string{line}
	for _, child := range children {
		childLines = append(childLines, renderNode(child, indent+"  "))
	}
	return strings.Join(childLines, "\n")
}
