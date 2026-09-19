package drills

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	drillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/drills"
	drillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/drills/drills_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client drillsconnect.RecoveryDrillsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: drillsconnect.NewRecoveryDrillsServiceClient(httpClient, baseURL)}
}

func (h *handlers) preview(ctx cliapp.RunContext) error {
	r, err := h.client.PreviewDrill(context.Background(), connect.NewRequest(&drillsv1.PreviewDrillRequest{PlanId: ctx.Flag("plan"), TargetId: ctx.Flag("target"), DestinationId: ctx.Flag("destination")}))
	if err != nil {
		return cliapp.WrapAPIError("preview recovery drill", err, nil)
	}
	if r == nil || r.Msg == nil {
		return fmt.Errorf("server returned no drill preview")
	}
	return cliapp.RenderProtoList(ctx, r.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Recovery drill eligible: %v.", r.Msg.Eligible)}, ResultsHeading: "Recovery drill preview", Results: []string{fmt.Sprintf("plan=%s target=%s destination=%s snapshot=%s reason=%s warnings=%s", r.Msg.PlanId, r.Msg.TargetId, r.Msg.DestinationId, r.Msg.SnapshotId, r.Msg.Reason, strings.Join(r.Msg.Warnings, "; "))}})
}

func (h *handlers) run(ctx cliapp.RunContext) error {
	r, err := h.client.RunDrill(context.Background(), connect.NewRequest(&drillsv1.RunDrillRequest{PlanId: ctx.Flag("plan"), TargetId: ctx.Flag("target"), DestinationId: ctx.Flag("destination"), IdempotencyKey: ctx.Flag("idempotency-key")}))
	if err != nil {
		return cliapp.WrapAPIError("run recovery drill", err, nil)
	}
	if r == nil || r.Msg == nil || r.Msg.Drill == nil {
		return fmt.Errorf("server returned no recovery drill")
	}
	return cliapp.RenderProtoMutation(ctx, r.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Recovery drill %s: %s.", statusLabel(r.Msg.Drill.Status), r.Msg.Drill.Id)}, Changes: []string{format(r.Msg.Drill)}, NextCommand: []string{fmt.Sprintf("`drills get %s` — inspect persisted evidence", r.Msg.Drill.Id)}})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	r, err := h.client.GetDrill(context.Background(), connect.NewRequest(&drillsv1.GetDrillRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get drill %q", id), err, nil)
	}
	if r == nil || r.Msg == nil || r.Msg.Drill == nil {
		return fmt.Errorf("server returned no recovery drill")
	}
	return cliapp.RenderProtoList(ctx, r.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Fetched drill %s.", id)}, ResultsHeading: "Recovery drill", Results: []string{format(r.Msg.Drill)}})
}
func (h *handlers) list(ctx cliapp.RunContext) error {
	r, err := h.client.ListDrills(context.Background(), connect.NewRequest(&drillsv1.ListDrillsRequest{PlanId: ctx.Flag("plan"), TargetId: ctx.Flag("target")}))
	if err != nil {
		return cliapp.WrapAPIError("list recovery drills", err, nil)
	}
	if r == nil || r.Msg == nil {
		return fmt.Errorf("server returned no recovery drills")
	}
	out := make([]string, 0, len(r.Msg.Drills))
	for _, d := range r.Msg.Drills {
		out = append(out, format(d))
	}
	return cliapp.RenderProtoList(ctx, r.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d recovery drill(s).", len(out))}, ResultsHeading: "Recovery drills", Results: out})
}

func statusLabel(s drillsv1.DrillStatus) string {
	switch s {
	case drillsv1.DrillStatus_DRILL_STATUS_REQUESTED:
		return "requested"
	case drillsv1.DrillStatus_DRILL_STATUS_RUNNING:
		return "running"
	case drillsv1.DrillStatus_DRILL_STATUS_VERIFIED:
		return "verified"
	case drillsv1.DrillStatus_DRILL_STATUS_FAILED:
		return "failed"
	default:
		return "unspecified"
	}
}
func format(d *drillsv1.RecoveryDrill) string {
	if d == nil {
		return "(nil)"
	}
	at := ""
	if d.RequestedAt != nil {
		at = d.RequestedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — plan=%s target=%s destination=%s snapshot=%s restore=%s status=%s scheduled=%v requested=%s error=%q next=%q", d.Id, d.PlanId, d.TargetId, d.DestinationId, d.SnapshotId, d.RestoreId, statusLabel(d.Status), d.Scheduled, at, d.Error, d.NextAction)
}
