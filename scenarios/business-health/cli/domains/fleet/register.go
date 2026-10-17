// Package fleet is the CLI's fleet-sweep command surface, mirroring the
// API's FleetService.
package fleet

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/fleet"
	fleetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/fleet/fleet_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "fleet"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"FleetService.ScanFleet": h.scan,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("fleet: load from manifest: %w", err)
	}
	return group, nil
}

type handlers struct {
	core   *cliapp.ScenarioApp
	client fleetconnect.FleetServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: fleetconnect.NewFleetServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) scan(ctx cliapp.RunContext) error {
	var scenarios []string
	if raw := strings.TrimSpace(ctx.Flag("scenarios")); raw != "" {
		for _, p := range strings.Split(raw, ",") {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				scenarios = append(scenarios, trimmed)
			}
		}
	}
	resp, err := h.client.ScanFleet(context.Background(), connect.NewRequest(&fleetv1.ScanFleetRequest{
		Scenarios: scenarios,
	}))
	if err != nil {
		return cliapp.WrapAPIError("scan fleet", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no fleet response")
	}
	results := make([]string, 0, len(resp.Msg.Entries))
	for _, e := range resp.Msg.Entries {
		flag := "ok"
		if !e.Passed {
			flag = fmt.Sprintf("%d error(s)", e.ErrorCount)
		}
		results = append(results, fmt.Sprintf("%s — debt %d (%s)", e.Scenario, e.DebtScore, flag))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Scanned %d scenario(s); %d passing.", resp.Msg.ScenarioCount, resp.Msg.PassingCount)},
		ResultsHeading: "Fleet (worst first)",
		Results:        results,
	})
}
