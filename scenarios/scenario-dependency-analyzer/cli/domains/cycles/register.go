package cycles

import (
	"fmt"
	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Verification",
		Commands: []cliapp.Command{
			{
				Name:        "cycles",
				Description: "Detect circular dependencies",
				NeedsAPI:    true,
				Run: func(args []string) error {
					return run(core, args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("cycles")
	var graphType string
	var jsonOutput bool
	fs.StringVar(&graphType, "type", "combined", "Graph type")
	fs.BoolVar(&jsonOutput, "json", false, "Output JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) > 1 {
		return fmt.Errorf("usage: %s cycles [graph-type] [--type combined|resource|scenario] [--json]", support.AppName)
	}
	if len(positionals) == 1 {
		graphType = positionals[0]
	}
	resolvedType, err := support.GraphType(graphType)
	if err != nil {
		return err
	}
	body, err := core.Get("/graph/"+resolvedType+"/cycles", nil)
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
	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Graph type: %s", resolvedType),
			fmt.Sprintf("Cycles detected: %t", support.Bool(resp["has_cycles"])),
			fmt.Sprintf("Severity: %s", support.String(resp["severity"])),
			support.String(resp["message"]),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Cycles", Items: cycleDescriptions(support.Maps(resp["cycles"]))},
			{Heading: "Affected dependencies", Items: support.Strings(resp["affected_dependencies"])},
		},
		NextSteps: []string{
			fmt.Sprintf("%s graph %s", support.AppName, resolvedType),
			fmt.Sprintf("%s analyze all --verbose", support.AppName),
		},
	}
	return support.PrintOperational(false, report, nil)
}

func cycleDescriptions(cycles []map[string]interface{}) []string {
	out := make([]string, 0, len(cycles))
	for _, cycle := range cycles {
		line := support.String(cycle["description"])
		if line == "" {
			line = fmt.Sprintf("%s cycle with %d hops", support.String(cycle["cycle_type"]), support.Int(cycle["length"]))
		}
		if support.Bool(cycle["required"]) {
			line += " (all required)"
		}
		out = append(out, line)
	}
	return out
}
