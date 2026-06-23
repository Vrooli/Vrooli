package snapshot

import (
	"fmt"
	"net/http"

	"github.com/vrooli/cli-core/cliapp"
	snapshotv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/snapshot"
	snapshotconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/snapshot/snapshot_v1connect"
)

type handlers struct{ core *cliapp.ScenarioApp }

func (h handlers) run(ctx cliapp.RunContext) error {
	resp, err := cliapp.Call[*snapshotv1.RunSnapshotRequest, *snapshotv1.RunSnapshotResponse](h.core, http.MethodPost, snapshotconnect.SnapshotServiceRunSnapshotProcedure, &snapshotv1.RunSnapshotRequest{Profile: ctx.Flag("profile"), DryRun: ctx.BoolFlag("dry-run")})
	if err != nil {
		return cliapp.WrapAPIError("run snapshot", err, nil)
	}
	return renderSnapshot(ctx, resp)
}

func (h handlers) list(ctx cliapp.RunContext) error {
	resp, err := cliapp.Call[*snapshotv1.ListSnapshotsRequest, *snapshotv1.ListSnapshotsResponse](h.core, http.MethodPost, snapshotconnect.SnapshotServiceListSnapshotsProcedure, &snapshotv1.ListSnapshotsRequest{})
	if err != nil {
		return cliapp.WrapAPIError("list snapshots", err, nil)
	}
	lines := make([]string, 0, len(resp.GetSnapshots()))
	for _, s := range resp.GetSnapshots() {
		lines = append(lines, formatSnapshot(s))
	}
	return cliapp.RenderProtoList(ctx, resp, cliapp.ListReport{Summary: []string{"Fetched network snapshots."}, ResultsHeading: "Snapshots", Results: lines, RetrievalHints: []string{"`snapshot run` — collect a new read-only snapshot"}})
}

func (h handlers) get(ctx cliapp.RunContext) error {
	resp, err := cliapp.Call[*snapshotv1.GetSnapshotRequest, *snapshotv1.GetSnapshotResponse](h.core, http.MethodPost, snapshotconnect.SnapshotServiceGetSnapshotProcedure, &snapshotv1.GetSnapshotRequest{Id: ctx.Positional("id")})
	if err != nil {
		return cliapp.WrapAPIError("get snapshot", err, nil)
	}
	return renderSnapshot(ctx, &snapshotv1.RunSnapshotResponse{Snapshot: resp.GetSnapshot()})
}

func (h handlers) export(ctx cliapp.RunContext) error {
	resp, err := cliapp.Call[*snapshotv1.ExportSnapshotReportRequest, *snapshotv1.ExportSnapshotReportResponse](h.core, http.MethodPost, snapshotconnect.SnapshotServiceExportSnapshotReportProcedure, &snapshotv1.ExportSnapshotReportRequest{Id: ctx.Positional("id"), Format: ctx.Flag("format")})
	if err != nil {
		return cliapp.WrapAPIError("export snapshot", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp, cliapp.ListReport{Summary: []string{fmt.Sprintf("Exported snapshot %s as %s.", resp.GetId(), resp.GetFormat())}, ResultsHeading: "Report", Results: []string{resp.GetReport()}})
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
