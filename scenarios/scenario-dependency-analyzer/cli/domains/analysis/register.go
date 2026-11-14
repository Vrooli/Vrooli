package analysis

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/analysis_operations"
	"scenario-dependency-analyzer/cli/internal/fleetoperations"
	"scenario-dependency-analyzer/cli/internal/support"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "analysis", Description: "Durable fleet analysis operations", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "start", Description: "Start or replay a fleet operation and return its receipt", NeedsAPI: true, Run: func(args []string) error { return start(core, args) }},
		{Name: "get", Description: "Read a bounded operation receipt", NeedsAPI: true, Run: func(args []string) error { return observe(core, args, false) }},
		{Name: "wait", Description: "Wait once for a server-owned operation", NeedsAPI: true, Run: func(args []string) error { return observe(core, args, true) }},
	}}
}

func start(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analysis start")
	var kind, callerID string
	var apply, jsonOutput bool
	fs.StringVar(&kind, "kind", "fleet-analysis", "fleet-analysis or graph-rebuild")
	fs.StringVar(&callerID, "client-operation-id", "", "Replay key; generated and printed when omitted")
	fs.BoolVar(&apply, "apply", false, "Publish rebuilt graph edges")
	fs.BoolVar(&jsonOutput, "json", false, "Output the operation receipt as JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("analysis start accepts flags only")
	}
	operationKind := pb.AnalysisOperationKind_ANALYSIS_OPERATION_KIND_FLEET_ANALYSIS
	switch kind {
	case "fleet-analysis", "ANALYSIS_OPERATION_KIND_FLEET_ANALYSIS":
		if apply {
			return fmt.Errorf("--apply requires --kind graph-rebuild")
		}
	case "graph-rebuild", "ANALYSIS_OPERATION_KIND_GRAPH_REBUILD":
		operationKind = pb.AnalysisOperationKind_ANALYSIS_OPERATION_KIND_GRAPH_REBUILD
	default:
		return fmt.Errorf("unsupported kind %q", kind)
	}
	return fleetoperations.StartAndObserve(core, operationKind, apply, callerID, false, jsonOutput)
}

func observe(core *cliapp.ScenarioApp, args []string, wait bool) error {
	fs := support.NewFlagSet("analysis get|wait")
	var jsonOutput bool
	fs.BoolVar(&jsonOutput, "json", false, "Output the operation receipt as JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: scenario-dependency-analyzer analysis get|wait <operation-id> [--json]")
	}
	return fleetoperations.Observe(core, fs.Args()[0], wait, jsonOutput)
}
