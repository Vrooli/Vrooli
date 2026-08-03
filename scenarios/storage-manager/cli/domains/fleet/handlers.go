package fleet

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/fleet"
	fleetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/fleet/fleet_v1connect"
)

type handlers struct {
	client fleetconnect.FleetServiceClient
}

const fleetTimeout = 20 * time.Minute

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, fleetTimeout)
	return &handlers{client: fleetconnect.NewFleetServiceClient(httpClient, baseURL)}
}

func (h *handlers) scan(ctx cliapp.RunContext) error {
	resp, err := h.client.ScanFleet(context.Background(), connect.NewRequest(&fleetv1.ScanFleetRequest{Scenarios: ctx.FlagValues("scenario")}))
	if err != nil {
		return cliapp.WrapAPIError("scan storage fleet", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no fleet response")
	}
	return render(ctx, resp.Msg)
}

func (h *handlers) inventory(ctx cliapp.RunContext) error {
	resp, err := h.client.GetInventory(context.Background(), connect.NewRequest(&fleetv1.GetInventoryRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("read storage fleet", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no fleet response")
	}
	return render(ctx, resp.Msg)
}

func render(ctx cliapp.RunContext, msg *fleetv1.ScanFleetResponse) error {
	results := make([]string, 0, len(msg.GetEntries()))
	for _, entry := range msg.GetEntries() {
		results = append(results, fmt.Sprintf("%s — engines=%s stage=%s isolation=%t findings=%d", entry.GetScenario(), strings.Join(entry.GetEngines(), ","), entry.GetStorageStage(), entry.GetIsolationReady(), entry.GetFindingCount()))
	}
	if len(results) == 0 {
		results = append(results, "No fleet entries.")
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Scanned %d scenario(s), %d finding(s).", msg.GetScenarioCount(), msg.GetFindingCount())}, ResultsHeading: "Storage fleet", Results: results})
}
