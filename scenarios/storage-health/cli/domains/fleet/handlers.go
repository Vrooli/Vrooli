package fleet

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-health/v1/fleet"
	fleetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/storage-health/v1/fleet/fleet_v1connect"
)

// fleetTimeout is generous: a whole-fleet scan runs static storage validation
// over every discovered scenario, which can take a couple of minutes.
const fleetTimeout = 5 * time.Minute

type handlers struct {
	client fleetconnect.FleetServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, fleetTimeout)
	return &handlers{client: fleetconnect.NewFleetServiceClient(httpClient, baseURL)}
}

// scan runs a fresh fleet scan and renders the selected view.
func (h *handlers) scan(ctx cliapp.RunContext) error {
	resp, err := h.client.ScanFleet(context.Background(), connect.NewRequest(&fleetv1.ScanFleetRequest{
		Scenarios: ctx.FlagValues("scenario"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("scan fleet", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no fleet response")
	}
	return render(ctx, resp.Msg)
}

// inventory renders the latest persisted snapshot.
func (h *handlers) inventory(ctx cliapp.RunContext) error {
	resp, err := h.client.GetInventory(context.Background(), connect.NewRequest(&fleetv1.GetInventoryRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("read fleet inventory", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no inventory response")
	}
	if resp.Msg.GetScenarioCount() == 0 {
		return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
			Summary:        []string{"No persisted snapshot yet. Run `storage-health fleet scan` to build one."},
			ResultsHeading: "Fleet storage inventory",
			Results:        []string{"(empty)"},
		})
	}
	return render(ctx, resp.Msg)
}

// render projects a fleet response onto the selected --view.
func render(ctx cliapp.RunContext, msg *fleetv1.ScanFleetResponse) error {
	view := strings.ToLower(strings.TrimSpace(firstFlag(ctx.FlagValues("view"))))
	if view == "" {
		view = "all"
	}
	summary := []string{fmt.Sprintf(
		"%d scenario(s): %d isolation-unready, %d data-persisting without a backup target, %d over data-dir budget, %d total finding(s).",
		msg.GetScenarioCount(), msg.GetIsolationUnreadyCount(), msg.GetNoBackupCount(), msg.GetDataDirOverBudgetCount(), msg.GetFindingCount(),
	)}
	if msg.GetScannedAt() != "" {
		summary = append(summary, "Snapshot: "+msg.GetScannedAt())
	}

	var heading string
	var results []string
	switch view {
	case "isolation":
		heading = "Scenarios with unproven test-isolation (destructive-E2E risk)"
		for _, e := range msg.GetEntries() {
			if e.GetIsolationReady() {
				continue
			}
			results = append(results, fmt.Sprintf("%s — %s", e.GetScenario(), reasonOr(e.GetIsolationReason(), "isolation seams unproven")))
		}
	case "no-backup":
		heading = "Data-persisting scenarios with no declared backup target"
		for _, e := range msg.GetEntries() {
			if e.GetHasBackupTarget() || !dataPersisting(e.GetEngines()) {
				continue
			}
			results = append(results, fmt.Sprintf("%s — stage=%s engines=%s", e.GetScenario(), e.GetStorageStage(), strings.Join(e.GetEngines(), ",")))
		}
	case "engines":
		heading = "Engine adoption across the fleet"
		for _, d := range msg.GetEngineDistribution() {
			results = append(results, fmt.Sprintf("%s — %d scenario(s)", d.GetEngine(), d.GetScenarioCount()))
		}
	case "stages":
		heading = "Deploy-stage distribution"
		for _, d := range msg.GetStageDistribution() {
			results = append(results, fmt.Sprintf("%s — %d scenario(s)", d.GetStage(), d.GetScenarioCount()))
		}
	default: // "all"
		heading = "Fleet storage inventory"
		for _, e := range msg.GetEntries() {
			iso := "isolation-ready"
			if !e.GetIsolationReady() {
				iso = "ISOLATION-UNREADY"
			}
			backup := ""
			if dataPersisting(e.GetEngines()) && !e.GetHasBackupTarget() {
				backup = " no-backup"
			}
			dataBudget := ""
			if e.GetDataDirOverBudget() {
				dataBudget = fmt.Sprintf(" data-dir=%s %.1f%%", e.GetDataDirSeverity(), e.GetDataDirUtilization()*100)
			}
			results = append(results, fmt.Sprintf("%s — engines=%s stage=%s %s findings=%d%s%s",
				e.GetScenario(), strings.Join(e.GetEngines(), ","), e.GetStorageStage(), iso, e.GetFindingCount(), backup, dataBudget))
		}
	}
	if len(results) == 0 {
		results = append(results, "No scenarios match this view.")
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: heading,
		Results:        results,
	})
}

func dataPersisting(engines []string) bool {
	for _, e := range engines {
		switch e {
		case "sqlite", "postgres", "qdrant", "file":
			return true
		}
	}
	return false
}

func reasonOr(reason, fallback string) string {
	if strings.TrimSpace(reason) == "" {
		return fallback
	}
	return reason
}

func firstFlag(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
