package domains

import (
	"context"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	integrationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/command-center/v1/integrations/integrations_v1connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// CommandGroups aggregates flat command groups from domain packages.
//
// Keep app.go focused on CLI metadata and cli-core wiring. As the scenario
// grows, add domains like domains/tasks or domains/projects and append their
// registrations here. For greenfield scenarios, domain packages are the
// default architecture; do not treat flat command files as the long-term plan.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	integrations := integrationconnect.NewIntegrationsServiceClient(httpClient, baseURL)
	writeProtoJSON := func(msg proto.Message) error {
		body, err := protojson.Marshal(msg)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(append(body, '\n'))
		return err
	}
	read := func(path string) func([]string) error {
		return func(args []string) error {
			body, err := core.Get(path, nil)
			if err != nil {
				return err
			}
			if hasJSON(args) {
				_, err = os.Stdout.Write(body)
				if err == nil {
					_, err = os.Stdout.Write([]byte("\n"))
				}
				return err
			}
			fmt.Println(string(body))
			return nil
		}
	}
	room := func(args []string) error {
		if len(args) == 0 || strings.HasPrefix(args[0], "-") {
			return fmt.Errorf("room requires an id")
		}
		path := "/rooms/" + args[0]
		if len(args) > 1 && args[1] == "--samples" && len(args) > 2 {
			path += "?samples=" + args[2]
		}
		return read(path)(args)
	}
	integrationRefresh := func(args []string) error {
		resp, err := integrations.Refresh(context.Background(), connect.NewRequest(&commonv1.RefreshIntegrationsRequest{}))
		if err != nil {
			return cliapp.WrapAPIError("refresh integrations", err, nil)
		}
		if hasJSON(args) {
			return writeProtoJSON(resp.Msg)
		}
		fmt.Printf("Refreshed %d integration(s).\n", len(resp.Msg.GetIntegrations()))
		return nil
	}
	integrationList := func(args []string) error {
		resp, err := integrations.List(context.Background(), connect.NewRequest(&commonv1.ListIntegrationsRequest{}))
		if err != nil {
			return cliapp.WrapAPIError("list integrations", err, nil)
		}
		if hasJSON(args) {
			return writeProtoJSON(resp.Msg)
		}
		for _, integration := range resp.Msg.GetIntegrations() {
			status := "unknown"
			if integration.GetLifecycle() != nil {
				status = integration.GetLifecycle().GetStatus().String()
			}
			fmt.Printf("%s\t%s\t%s\n", integration.GetId(), status, integration.GetOrigin())
		}
		return nil
	}
	integrationAction := func(args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("integration-action requires an id and action")
		}
		confirmed := false
		for _, arg := range args[2:] {
			if arg == "--confirm" {
				confirmed = true
			}
		}
		if !confirmed {
			return fmt.Errorf("integration-action requires --confirm")
		}
		action, err := parseActionKind(args[1])
		if err != nil {
			return err
		}
		resp, err := integrations.RunAction(context.Background(), connect.NewRequest(&commonv1.RunIntegrationActionRequest{IntegrationId: args[0], Action: action, Confirmed: true}))
		if err != nil {
			return cliapp.WrapAPIError("run integration action", err, nil)
		}
		if hasJSON(args) {
			return writeProtoJSON(resp.Msg)
		}
		fmt.Printf("%s: %s\n", resp.Msg.GetStatus(), resp.Msg.GetMessage())
		return nil
	}
	return []cliapp.CommandGroup{{Title: "Instrument reads", Commands: []cliapp.Command{{Name: "board", Description: "Show board shape", NeedsAPI: true, Run: read("/board")}, {Name: "room", Description: "Show one composed room", NeedsAPI: true, Run: room}, {Name: "focus", Description: "Show ranked findings", NeedsAPI: true, Run: read("/focus")}, {Name: "open-loop", Description: "Show dated open holes", NeedsAPI: true, Run: read("/open-loop")}, {Name: "describe", Description: "Describe the sensor space", NeedsAPI: true, Run: read("/capabilities/describe")}, {Name: "gaps", Description: "Show compatibility gaps", NeedsAPI: true, Run: read("/gaps")}, {Name: "integrations", Description: "Show lifecycle and feature state for declared integrations", NeedsAPI: true, Run: integrationList}, {Name: "integrations-refresh", Description: "Refresh declared integration state", NeedsAPI: true, Run: integrationRefresh}, {Name: "integration-action", Description: "Run a confirmed allowlisted integration action", NeedsAPI: true, Run: integrationAction}}}}
}

func parseActionKind(raw string) (commonv1.ActionKind, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "owner_guidance":
		return commonv1.ActionKind_ACTION_KIND_OWNER_GUIDANCE, nil
	case "scenario_start":
		return commonv1.ActionKind_ACTION_KIND_SCENARIO_START, nil
	case "scenario_restart":
		return commonv1.ActionKind_ACTION_KIND_SCENARIO_RESTART, nil
	case "operator_command":
		return commonv1.ActionKind_ACTION_KIND_OPERATOR_COMMAND, nil
	default:
		return commonv1.ActionKind_ACTION_KIND_UNSPECIFIED, fmt.Errorf("unsupported integration action %q", raw)
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
//
// Prefer domain packages as the default growth path:
//
//	cli/domains/tasks/register.go
//	cli/domains/projects/register.go
//
// For API-backed commands:
//   - set NeedsAPI: true so stale-check + --auto-start preflight works
//   - call core.Get(...) / core.Request(...) for versioned /api/v1 routes
//   - use cliapp.RenderOperationalReport / RenderListReport /
//     RenderMutationReport for default human output contracts
//   - use cliapp.PrintReportJSON(...) when a --json mode should mirror the
//     same structured report
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	_ = core
	return nil
}

func hasJSON(args []string) bool {
	for _, a := range args {
		if a == "--json" {
			return true
		}
	}
	return false
}
