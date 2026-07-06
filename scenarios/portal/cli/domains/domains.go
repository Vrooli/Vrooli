package domains

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	integrationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/integrations"
	integrationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/integrations/integrations_v1connect"
)

// CommandGroups aggregates flat command groups from domain packages.
//
// Keep app.go focused on CLI metadata and cli-core wiring. As the scenario
// grows, add domains like domains/tasks or domains/projects and append their
// registrations here. For greenfield scenarios, domain packages are the
// default architecture; do not treat flat command files as the long-term plan.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
//
// Each domain package owns a Register(core, manifest) function returning a
// SubcommandGroup built from the scenario's cli/manifest.json. The aggregator
// passes the embedded manifest bytes through unchanged; per-domain Register
// implementations call cliapp.LoadFromManifest with the relevant group name.
//
// This is the CLI side of the domain-module pattern; the API side uses
// the same one-liner-per-domain shape via server.New(deps, modules...).
// See docs/concepts/ARCHITECTURE.md "Domain modules" for the canonical
// pattern when swapping the example domain for your scenario's first
// domain.
//
// For API-backed commands the manifest carries the declarative surface
// (governance, flags, positionals, RPC binding). Handlers stay in
// handlers.go and are wired via the bindings map; refer to
// templates/scenarios/react-vite/docs/internal/SEAMS.md (manifest ↔
// handlers bindings seam) for the contract.
func SubcommandGroups(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.SubcommandGroup, error) {
	integrations, err := cliapp.LoadFromManifest(manifest, "integrations", map[string]func(cliapp.RunContext) error{
		"IntegrationsService.Status": runIntegrationsStatus,
	})
	if err != nil {
		return nil, err
	}
	return []cliapp.SubcommandGroup{integrations}, nil
}

func runIntegrationsStatus(ctx cliapp.RunContext) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(ctx.Core())
	client := integrationsconnect.NewIntegrationsServiceClient(httpClient, baseURL)
	resp, err := client.Status(context.Background(), connect.NewRequest(&integrationsv1.StatusRequest{}))
	if err != nil {
		return err
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("mode: %s", resp.Msg.GetActiveMode().String()),
			fmt.Sprintf("reason: %s", resp.Msg.GetReason()),
		},
	})
}
