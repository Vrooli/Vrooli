package system

import (
	"context"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"google.golang.org/protobuf/types/known/emptypb"
)

func stringPointer(value string) *string { return &value }

type downloadResult struct {
	ScenarioName string `json:"scenario_name"`
	Platform     string `json:"platform"`
	OutputPath   string `json:"output_path"`
	Bytes        int    `json:"bytes"`
}

func (c *Commands) downloadPrimitive() cliapp.PrimitiveHandler {
	return cliapp.Action(func(ctx cliapp.OperationContext) (downloadResult, error) {
		scenario, platform := strings.TrimSpace(ctx.Positional("scenario")), strings.TrimSpace(ctx.Positional("platform"))
		body, err := c.deps.Get("/desktop/download/"+scenario+"/"+platform, nil)
		if err != nil {
			return downloadResult{}, err
		}
		outputPath := strings.TrimSpace(ctx.Flag("output"))
		if outputPath == "" {
			extensions := map[string]string{"win": ".exe", "mac": ".zip", "linux": ".AppImage"}
			outputPath = scenario + "-" + platform + extensions[platform]
		}
		if err := os.WriteFile(outputPath, body, 0o755); err != nil {
			return downloadResult{}, fmt.Errorf("write desktop package: %w", err)
		}
		return downloadResult{ScenarioName: scenario, Platform: platform, OutputPath: outputPath, Bytes: len(body)}, nil
	}, func(_ cliapp.OperationContext, result downloadResult) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Downloaded %s package to %s (%d bytes)", result.Platform, result.OutputPath, result.Bytes)}}
	})
}

func (c *Commands) templatesListPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(cliapp.OperationContext) (*domainv1.ListTemplatesResponse, error) {
		response, err := c.system.ListTemplates(context.Background(), connect.NewRequest(&domainv1.ListTemplatesRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list templates", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.ListTemplatesResponse) cliapp.ListReport {
		report := cliapp.ListReport{Summary: []string{fmt.Sprintf("Available desktop templates: %d", len(response.GetTemplates()))}, ResultsHeading: "Templates", RetrievalHints: []string{"Use `scenario-to-desktop template <type>` for full template details."}}
		for _, item := range response.GetTemplates() {
			report.Results = append(report.Results, fmt.Sprintf("%s - %s [%s]: %s", item.GetType(), item.GetName(), item.GetComplexity(), item.GetDescription()))
		}
		return report
	})
}

func (c *Commands) templateGetPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*domainv1.TemplateConfigResponse, error) {
		response, err := c.system.GetTemplate(context.Background(), connect.NewRequest(&domainv1.GetTemplateRequest{Type: ctx.Positional("type")}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get template", err, nil)
		}
		return response.Msg, nil
	}, func(ctx cliapp.OperationContext, _ *domainv1.TemplateConfigResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Template: " + ctx.Positional("type")}}
	})
}

func (c *Commands) recordsListPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(cliapp.OperationContext) (*domainv1.DesktopRecordsResponse, error) {
		response, err := c.records.ListDesktopRecords(context.Background(), connect.NewRequest(&emptypb.Empty{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list desktop records", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.DesktopRecordsResponse) cliapp.ListReport {
		report := cliapp.ListReport{Summary: []string{fmt.Sprintf("Desktop records: %d", len(response.GetRecords()))}, ResultsHeading: "Records", RetrievalHints: []string{"Use `scenario-to-desktop records-move <id>` to relocate an existing wrapper."}}
		if len(response.GetRecords()) == 0 {
			report.RetrievalHints = []string{"Run `scenario-to-desktop pipeline list` to inspect active desktop pipelines."}
		}
		for _, item := range response.GetRecords() {
			report.Results = append(report.Results, fmt.Sprintf("%s | %s | %s", item.GetRecord().GetId(), item.GetRecord().GetScenarioName(), item.GetBuildState()))
		}
		return report
	})
}

func (c *Commands) recordsMovePrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*domainv1.MoveDesktopRecordResponse, error) {
		req := &domainv1.MoveDesktopRecordRequest{RecordId: ctx.Positional("id"), Target: stringPointer(ctx.Flag("target"))}
		if ctx.FlagProvided("path") {
			req.DestinationPath = stringPointer(ctx.Flag("path"))
		}
		response, err := c.records.MoveDesktopRecord(context.Background(), connect.NewRequest(req))
		if err != nil {
			return nil, cliapp.WrapAPIError("move desktop record", err, nil)
		}
		return response.Msg, nil
	}, func(ctx cliapp.OperationContext, _ *domainv1.MoveDesktopRecordResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Desktop record %s moved successfully.", ctx.Positional("id"))}, Changes: []string{"Target: " + ctx.Flag("target")}, NextCommand: []string{"scenario-to-desktop records"}}
	})
}

func (c *Commands) recordsDeletePrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*domainv1.DeleteDesktopScenarioResponse, error) {
		response, err := c.records.DeleteDesktopScenario(context.Background(), connect.NewRequest(&domainv1.DeleteDesktopScenarioRequest{ScenarioName: ctx.Positional("scenario")}))
		if err != nil {
			return nil, cliapp.WrapAPIError("delete desktop scenario", err, nil)
		}
		return response.Msg, nil
	}, func(ctx cliapp.OperationContext, _ *domainv1.DeleteDesktopScenarioResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Desktop app deleted for " + ctx.Positional("scenario") + "."}, Changes: []string{"Desktop wrapper and generated artifacts were removed."}, NextCommand: []string{"scenario-to-desktop desktop-status"}}
	})
}

func (c *Commands) desktopStatusPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(cliapp.OperationContext) (*domainv1.DesktopScenarioStatusResponse, error) {
		response, err := c.operations.ListDesktopScenarioStatus(context.Background(), connect.NewRequest(&emptypb.Empty{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list desktop scenario status", err, nil)
		}
		return response.Msg, nil
	}, func(ctx cliapp.OperationContext, response *domainv1.DesktopScenarioStatusResponse) cliapp.ListReport {
		value := desktopStatusResponseFromProto(response)
		if name := strings.TrimSpace(ctx.Flag("name")); name != "" {
			value.Scenarios = filterScenariosByName(value.Scenarios, name)
		}
		report := cliapp.ListReport{Summary: []string{fmt.Sprintf("Scenarios: %d total", value.Stats.Total), fmt.Sprintf("With desktop: %d", value.Stats.WithDesktop), fmt.Sprintf("Built: %d", value.Stats.Built), fmt.Sprintf("Web-only: %d", value.Stats.WebOnly)}, ResultsHeading: "Scenarios", RetrievalHints: []string{"Use `scenario-to-desktop download <scenario> <platform>` once a scenario is built."}}
		if len(value.Scenarios) == 0 {
			report.Summary = []string{"Desktop scenarios: 0"}
			report.RetrievalHints = []string{"Run `scenario-to-desktop pipeline run <scenario>` to start a desktop build."}
		}
		for _, item := range value.Scenarios {
			report.Results = append(report.Results, desktopScenarioLine(item))
		}
		return report
	})
}

func (c *Commands) wineCheckPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoOperational(func(cliapp.OperationContext) (*domainv1.WineCheckResponse, error) {
		response, err := c.system.CheckWine(context.Background(), connect.NewRequest(&domainv1.CheckWineRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("check wine", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.WineCheckResponse) cliapp.OperationalReport {
		report := cliapp.OperationalReport{NextSteps: []string{"scenario-to-desktop wine install --method <flatpak|flatpak-auto|appimage>"}}
		if response.GetInstalled() {
			version := response.GetVersion()
			if version == "" {
				version = "unknown version"
			}
			report.Status = []string{fmt.Sprintf("Wine installed (%s)", version)}
		} else {
			report.Status = []string{"Wine not installed"}
			if len(response.GetInstallMethods()) > 0 {
				methods := make([]string, 0, len(response.GetInstallMethods()))
				for _, item := range response.GetInstallMethods() {
					methods = append(methods, item.GetId())
				}
				report.Triage = []cliapp.TriageGroup{{Heading: "Install Methods", Items: []string{strings.Join(methods, ", ")}}}
			}
		}
		return report
	})
}

func (c *Commands) wineInstallPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*domainv1.WineInstallResponse, error) {
		response, err := c.system.InstallWine(context.Background(), connect.NewRequest(&domainv1.InstallWineRequest{Method: ctx.Flag("method")}))
		if err != nil {
			return nil, cliapp.WrapAPIError("install wine", err, nil)
		}
		return response.Msg, nil
	}, func(ctx cliapp.OperationContext, response *domainv1.WineInstallResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Wine installation started: " + response.GetInstallId()}, Changes: []string{"Method: " + ctx.Flag("method")}, NextCommand: []string{fmt.Sprintf("%s wine status %s", appName, response.GetInstallId())}}
	})
}

func (c *Commands) wineStatusPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*domainv1.WineInstallStatusResponse, error) {
		response, err := c.system.GetWineInstallStatus(context.Background(), connect.NewRequest(&domainv1.GetWineInstallStatusRequest{InstallId: ctx.Positional("install_id")}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get wine installation status", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.WineInstallStatusResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{fmt.Sprintf("Wine installation %s: %s", response.GetInstallId(), response.GetStatus())}}
	})
}
