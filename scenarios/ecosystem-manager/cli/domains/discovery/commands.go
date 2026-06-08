// Package discovery is the CLI's discovery-domain command surface. It mirrors
// the API's Connect-RPC DiscoveryService and the UI's api/discovery.ts client,
// consuming the generated Connect client over the shared cli-core HTTP client.
// This is the reference shape for migrated-domain CLI commands (see
// docs/internal/MIGRATION-GUIDE.md).
package discovery

import (
	"context"
	"flag"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ecosystem-manager/v1/discovery"
	discoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ecosystem-manager/v1/discovery/discovery_v1connect"
)

// Commands returns the discovery command group.
func Commands(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Discovery",
		Commands: []cliapp.Command{
			{
				Name:        "discovery",
				NeedsAPI:    true,
				Description: "Discover resources and scenarios (resources|scenarios)",
				Run: func(args []string) error {
					return route(core, args)
				},
			},
		},
	}
}

func route(core *cliapp.ScenarioApp, args []string) error {
	if len(args) == 0 || args[0] == "help" {
		fmt.Println(usageText())
		return nil
	}
	switch args[0] {
	case "resources":
		return cmdResources(core, args[1:])
	case "scenarios":
		return cmdScenarios(core, args[1:])
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", args[0], usageText())
	}
}

func usageText() string {
	return "Usage: ecosystem-manager discovery <resources|scenarios> [--refresh]"
}

func client(core *cliapp.ScenarioApp) discoveryconnect.DiscoveryServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return discoveryconnect.NewDiscoveryServiceClient(httpClient, baseURL)
}

func cmdResources(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("resources", flag.ContinueOnError)
	refresh := fs.Bool("refresh", false, "bypass the discovery cache")
	if err := fs.Parse(args); err != nil {
		return err
	}
	resp, err := client(core).ListResources(
		context.Background(),
		connect.NewRequest(&discoveryv1.ListResourcesRequest{Refresh: *refresh}),
	)
	if err != nil {
		return err
	}
	fmt.Printf("%d resource(s):\n", resp.Msg.GetCount())
	for _, r := range resp.Msg.GetResources() {
		health := "down"
		if r.GetHealthy() {
			health = "ok"
		}
		fmt.Printf("  %-28s [%s] %s %s\n", r.GetName(), health, r.GetCategory(), r.GetStatus())
	}
	return nil
}

func cmdScenarios(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("scenarios", flag.ContinueOnError)
	refresh := fs.Bool("refresh", false, "bypass the discovery cache")
	if err := fs.Parse(args); err != nil {
		return err
	}
	resp, err := client(core).ListScenarios(
		context.Background(),
		connect.NewRequest(&discoveryv1.ListScenariosRequest{Refresh: *refresh}),
	)
	if err != nil {
		return err
	}
	fmt.Printf("%d scenario(s):\n", resp.Msg.GetCount())
	for _, s := range resp.Msg.GetScenarios() {
		fmt.Printf("  %-28s %s %s\n", s.GetName(), s.GetCategory(), s.GetStatus())
	}
	return nil
}
