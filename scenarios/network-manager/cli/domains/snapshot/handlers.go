package snapshot

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	snapshotv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/snapshot"
	snapshotconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/snapshot/snapshot_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client snapshotconnect.SnapshotServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return handlers{
		core:   core,
		client: snapshotconnect.NewSnapshotServiceClient(httpClient, baseURL),
	}
}

func (h handlers) run(ctx cliapp.RunContext) error {
	resp, err := h.client.RunSnapshot(context.Background(), connect.NewRequest(&snapshotv1.RunSnapshotRequest{Profile: ctx.Flag("profile"), DryRun: ctx.BoolFlag("dry-run")}))
	if err != nil {
		return cliapp.WrapAPIError("run snapshot", err, nil)
	}
	return renderSnapshot(ctx, resp.Msg)
}

func (h handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListSnapshots(context.Background(), connect.NewRequest(&snapshotv1.ListSnapshotsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list snapshots", err, nil)
	}
	lines := make([]string, 0, len(resp.Msg.GetSnapshots()))
	for _, s := range resp.Msg.GetSnapshots() {
		lines = append(lines, formatSnapshot(s))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{"Fetched network snapshots."}, ResultsHeading: "Snapshots", Results: lines, RetrievalHints: []string{"`snapshot run` — collect a new read-only snapshot"}})
}

func (h handlers) get(ctx cliapp.RunContext) error {
	resp, err := h.client.GetSnapshot(context.Background(), connect.NewRequest(&snapshotv1.GetSnapshotRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("get snapshot", err, nil)
	}
	return renderSnapshot(ctx, &snapshotv1.RunSnapshotResponse{Snapshot: resp.Msg.GetSnapshot()})
}

func (h handlers) export(ctx cliapp.RunContext) error {
	resp, err := h.client.ExportSnapshotReport(context.Background(), connect.NewRequest(&snapshotv1.ExportSnapshotReportRequest{Id: ctx.Positional("id"), Format: ctx.Flag("format")}))
	if err != nil {
		return cliapp.WrapAPIError("export snapshot", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Exported snapshot %s as %s.", resp.Msg.GetId(), resp.Msg.GetFormat())}, ResultsHeading: "Report", Results: []string{resp.Msg.GetReport()}})
}

func renderSnapshot(ctx cliapp.RunContext, resp *snapshotv1.RunSnapshotResponse) error {
	return cliapp.RenderProtoList(ctx, resp, cliapp.ListReport{Summary: []string{"Snapshot ready."}, ResultsHeading: "Snapshot", Results: []string{formatSnapshot(resp.GetSnapshot())}, RetrievalHints: []string{"`adapters capabilities` — see what can be measured or changed"}})
}

func formatSnapshot(s *snapshotv1.Snapshot) string {
	if s == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s status=%s profile=%s summary=%s", s.GetId(), s.GetStatus(), s.GetProfile(), s.GetSummary())
}
