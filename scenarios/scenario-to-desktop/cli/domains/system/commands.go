// Package system provides CLI commands for system operations.
package system

import (
	"context"
	"fmt"
	"strings"

	"scenario-to-desktop/cli/internal/support"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	"google.golang.org/protobuf/types/known/emptypb"
)

const appName = "scenario-to-desktop"

type Commands struct {
	deps       support.Dependencies
	records    recordsRPC
	system     systemRPC
	operations operationsRPC
}

type systemRPC interface {
	ListTemplates(context.Context, *connect.Request[domainv1.ListTemplatesRequest]) (*connect.Response[domainv1.ListTemplatesResponse], error)
	GetTemplate(context.Context, *connect.Request[domainv1.GetTemplateRequest]) (*connect.Response[domainv1.TemplateConfigResponse], error)
	CheckWine(context.Context, *connect.Request[domainv1.CheckWineRequest]) (*connect.Response[domainv1.WineCheckResponse], error)
	InstallWine(context.Context, *connect.Request[domainv1.InstallWineRequest]) (*connect.Response[domainv1.WineInstallResponse], error)
	GetWineInstallStatus(context.Context, *connect.Request[domainv1.GetWineInstallStatusRequest]) (*connect.Response[domainv1.WineInstallStatusResponse], error)
}

type operationsRPC interface {
	ListDesktopScenarioStatus(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[domainv1.DesktopScenarioStatusResponse], error)
}

type recordsRPC interface {
	ListDesktopRecords(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[domainv1.DesktopRecordsResponse], error)
	MoveDesktopRecord(context.Context, *connect.Request[domainv1.MoveDesktopRecordRequest]) (*connect.Response[domainv1.MoveDesktopRecordResponse], error)
	DeleteDesktopScenario(context.Context, *connect.Request[domainv1.DeleteDesktopScenarioRequest]) (*connect.Response[domainv1.DeleteDesktopScenarioResponse], error)
}

func New(deps support.Dependencies) *Commands {
	app := deps.Core()
	httpClient, baseURL := cliapp.NewConnectHTTPClient(app)
	return &Commands{deps: deps, records: domainconnect.NewDesktopRecordsServiceClient(httpClient, baseURL), system: domainconnect.NewSystemServiceClient(httpClient, baseURL), operations: domainconnect.NewOperationsServiceClient(httpClient, baseURL)}
}

func CommandGroups(deps support.Dependencies) []cliapp.CommandGroup {
	cmds := New(deps)
	return []cliapp.CommandGroup{
		{Title: "Templates", Commands: []cliapp.Command{
			(cliapp.Command{Name: "templates", NeedsAPI: true, Description: "List available desktop templates"}).WithPrimitive(cmds.templatesListPrimitive()),
			(cliapp.Command{Name: "template", NeedsAPI: true, Description: "Get template details: template <type>", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "type", Required: true, Description: "Template type"}}}}).WithPrimitive(cmds.templateGetPrimitive()),
		}},
		{Title: "Records", Commands: []cliapp.Command{
			(cliapp.Command{Name: "records", NeedsAPI: true, Description: "List desktop generation records"}).WithPrimitive(cmds.recordsListPrimitive()),
			(cliapp.Command{Name: "records-move", NeedsAPI: true, Description: "Move desktop wrapper: records-move <id> [--target <path>]", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "Desktop record ID"}}, Flags: []cliapp.Flag{{Name: "target", Default: "destination", Description: "Move target"}, {Name: "path", Description: "Custom destination path"}}}}).WithPrimitive(cmds.recordsMovePrimitive()),
			(cliapp.Command{Name: "records-delete", NeedsAPI: true, Description: "Delete desktop app: records-delete <scenario>", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true, Description: "Scenario name"}}}}).WithPrimitive(cmds.recordsDeletePrimitive()),
		}},
		{Title: "Download", Commands: []cliapp.Command{
			(cliapp.Command{Name: "download", NeedsAPI: true, Description: "Download built package: download <scenario> <platform> [--output <path>]", Args: cliapp.ArgSchema{
				Positionals: []cliapp.Positional{{Name: "scenario", Required: true, Description: "Scenario name"}, {Name: "platform", Required: true, Description: "Target platform (win, mac, or linux)"}},
				Flags:       []cliapp.Flag{{Name: "output", Description: "Output file path (defaults to the current directory)"}},
			}}).WithPrimitive(cmds.downloadPrimitive()),
		}},
		{Title: "Scenarios", Commands: []cliapp.Command{
			(cliapp.Command{Name: "desktop-status", NeedsAPI: true, Description: "List desktop build status and artifacts", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "name", Description: "Filter by scenario name"}}}}).WithPrimitive(cmds.desktopStatusPrimitive()),
		}},
	}
}

func WineRegister(deps support.Dependencies) cliapp.SubcommandGroup {
	cmds := New(deps)
	return cliapp.SubcommandGroup{Name: "wine", Description: "Wine for Windows builds on Linux (run 'wine help' for details)", NeedsAPI: true, Subcommands: []cliapp.Command{
		(cliapp.Command{Name: "check", Description: "Check Wine installation status"}).WithPrimitive(cmds.wineCheckPrimitive()),
		(cliapp.Command{Name: "install", Description: "Install Wine: install --method <flatpak|appimage>", Args: cliapp.ArgSchema{
			Flags: []cliapp.Flag{{Name: "method", Required: true, Description: "Installation method", Values: []string{"flatpak", "flatpak-auto", "appimage"}}},
		}}).WithPrimitive(cmds.wineInstallPrimitive()),
		(cliapp.Command{Name: "status", Description: "Get Wine install status: status <id>", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "install_id", Required: true, Description: "Installation ID"}}}}).WithPrimitive(cmds.wineStatusPrimitive()),
	}}
}

type desktopBuildArtifact struct {
	Platform, FileName         string
	SizeBytes                  int64
	RelativePath, AbsolutePath string
}
type desktopScenarioStatus struct {
	Name, DisplayName, Version string
	Built                      bool
	Platforms                  []string
	BuildArtifacts             []desktopBuildArtifact
}
type desktopStatusResponse struct {
	Scenarios []desktopScenarioStatus
	Stats     struct{ Total, WithDesktop, Built, WebOnly int }
}

func desktopStatusResponseFromProto(value *domainv1.DesktopScenarioStatusResponse) desktopStatusResponse {
	result := desktopStatusResponse{}
	if value == nil {
		return result
	}
	if stats := value.GetStats(); stats != nil {
		result.Stats.Total, result.Stats.WithDesktop, result.Stats.Built, result.Stats.WebOnly = int(stats.GetTotal()), int(stats.GetWithDesktop()), int(stats.GetBuilt()), int(stats.GetWebOnly())
	}
	for _, item := range value.GetScenarios() {
		scenario := desktopScenarioStatus{Name: item.GetName(), DisplayName: item.GetDisplayName(), Version: item.GetVersion(), Built: item.GetBuilt(), Platforms: item.GetPlatforms()}
		for _, artifact := range item.GetBuildArtifacts() {
			scenario.BuildArtifacts = append(scenario.BuildArtifacts, desktopBuildArtifact{Platform: artifact.GetPlatform(), FileName: artifact.GetFileName(), SizeBytes: artifact.GetSizeBytes(), RelativePath: artifact.GetRelativePath(), AbsolutePath: artifact.GetAbsolutePath()})
		}
		result.Scenarios = append(result.Scenarios, scenario)
	}
	return result
}

func filterScenariosByName(scenarios []desktopScenarioStatus, name string) []desktopScenarioStatus {
	filtered := make([]desktopScenarioStatus, 0)
	for _, scenario := range scenarios {
		if scenario.Name == name {
			filtered = append(filtered, scenario)
		}
	}
	return filtered
}

func desktopScenarioLine(scenario desktopScenarioStatus) string {
	name := scenario.Name
	if strings.TrimSpace(scenario.DisplayName) != "" {
		name = fmt.Sprintf("%s (%s)", name, scenario.DisplayName)
	}
	version := scenario.Version
	if strings.TrimSpace(version) == "" {
		version = "unknown"
	}
	status := "not built"
	if scenario.Built {
		status = "built"
	}
	parts := []string{fmt.Sprintf("%s v%s [%s]", name, version, status)}
	if len(scenario.Platforms) > 0 {
		parts = append(parts, fmt.Sprintf("platforms=%s", strings.Join(scenario.Platforms, ", ")))
	}
	if len(scenario.BuildArtifacts) > 0 {
		artifacts := make([]string, 0, len(scenario.BuildArtifacts))
		for _, artifact := range scenario.BuildArtifacts {
			fileName := artifact.FileName
			if fileName == "" {
				fileName = artifact.RelativePath
			}
			artifacts = append(artifacts, fmt.Sprintf("%s=%s (%d bytes)", artifact.Platform, fileName, artifact.SizeBytes))
		}
		parts = append(parts, "artifacts="+strings.Join(artifacts, "; "))
	}
	return strings.Join(parts, " | ")
}
