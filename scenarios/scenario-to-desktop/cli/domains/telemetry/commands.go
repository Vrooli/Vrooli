// Package telemetry provides CLI commands for telemetry management.
package telemetry

import (
	"context"
	"scenario-to-desktop/cli/internal/support"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
)

// Commands provides telemetry CLI commands.
type Commands struct {
	deps support.Dependencies
	rpc  telemetryRPC
}

// New creates a new telemetry Commands instance.
func New(deps support.Dependencies) *Commands {
	return &Commands{deps: deps, rpc: newTelemetryRPC(deps.ScenarioApp())}
}

type telemetryRPC interface {
	IngestTelemetry(context.Context, *connect.Request[domainv1.IngestTelemetryRequest]) (*connect.Response[domainv1.IngestTelemetryResponse], error)
	GetTelemetrySummary(context.Context, *connect.Request[domainv1.TelemetryScenarioRequest]) (*connect.Response[domainv1.TelemetryPayloadResponse], error)
	GetTelemetryInsights(context.Context, *connect.Request[domainv1.TelemetryScenarioRequest]) (*connect.Response[domainv1.TelemetryPayloadResponse], error)
	GetTelemetryTail(context.Context, *connect.Request[domainv1.TelemetryTailRequest]) (*connect.Response[domainv1.TelemetryPayloadResponse], error)
	DeleteTelemetry(context.Context, *connect.Request[domainv1.TelemetryScenarioRequest]) (*connect.Response[domainv1.TelemetryDeleteResponse], error)
}

func newTelemetryRPC(app *cliapp.ScenarioApp) telemetryRPC {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(app)
	return domainconnect.NewTelemetryServiceClient(httpClient, baseURL)
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	cmds := New(deps)
	return cliapp.SubcommandGroup{
		Name:        "telemetry",
		Description: "Deployment telemetry (run 'telemetry help' for details)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			(cliapp.Command{Name: "ingest", Description: "Ingest telemetry from file: ingest <scenario> --file <path>", Args: telemetryScenarioArgs("file", "source")}).WithPrimitive(cmds.ingestPrimitive()),
			(cliapp.Command{Name: "summary", Description: "Get telemetry summary: summary <scenario>", Args: telemetryScenarioArgs()}).WithPrimitive(cmds.summaryPrimitive()),
			(cliapp.Command{Name: "insights", Description: "Get AI-generated insights: insights <scenario>", Args: telemetryScenarioArgs()}).WithPrimitive(cmds.insightsPrimitive()),
			(cliapp.Command{Name: "tail", Description: "Get recent telemetry: tail <scenario> [--limit N]", Args: telemetryScenarioArgs("limit")}).WithPrimitive(cmds.tailPrimitive()),
			(cliapp.Command{Name: "download", Description: "Download telemetry file: download <scenario> [--output <path>]", Args: telemetryScenarioArgs("output")}).WithPrimitive(cmds.downloadPrimitive()),
			(cliapp.Command{Name: "delete", Description: "Delete telemetry: delete <scenario>", Args: telemetryScenarioArgs()}).WithPrimitive(cmds.deletePrimitive()),
		},
	}
}

func telemetryScenarioArgs(flags ...string) cliapp.ArgSchema {
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true, Description: "Scenario name"}}}
	for _, name := range flags {
		flag := cliapp.Flag{Name: name}
		switch name {
		case "file":
			flag.Required = true
			flag.Description = "Path to telemetry JSONL file"
		case "source":
			flag.Default = "cli"
			flag.Description = "Telemetry source identifier"
		case "limit":
			flag.Default = "200"
			flag.Description = "Maximum entries to return"
		case "output":
			flag.Description = "Write raw JSONL to this file instead of reporting it"
		}
		schema.Flags = append(schema.Flags, flag)
	}
	return schema
}
