package harness

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	harnessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/harness"
	harnessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/harness/harness_v1connect"
)

type handlers struct {
	client harnessconnect.HarnessServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	http, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: harnessconnect.NewHarnessServiceClient(http, base)}
}
func runtime(ctx cliapp.OperationContext) string {
	if v := ctx.Flag("runtime"); v != "" {
		return v
	}
	return "claude-code"
}
func (h *handlers) importCall(ctx cliapp.OperationContext) (*harnessv1.RunImportResponse, error) {
	resp, err := h.client.RunImport(context.Background(), connect.NewRequest(&harnessv1.RunImportRequest{Runtime: runtime(ctx)}))
	if err != nil {
		return nil, cliapp.WrapAPIError("start memory import", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) statusCall(ctx cliapp.OperationContext) (*harnessv1.GetImportStatusResponse, error) {
	resp, err := h.client.GetImportStatus(context.Background(), connect.NewRequest(&harnessv1.GetImportStatusRequest{RunId: ctx.Flag("run-id"), Runtime: runtime(ctx)}))
	if err != nil {
		return nil, cliapp.WrapAPIError("read import status", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) importReport(_ cliapp.OperationContext, msg *harnessv1.RunImportResponse) cliapp.MutationReport {
	if msg.Run == nil {
		return cliapp.MutationReport{Result: []string{"Import request accepted."}}
	}
	joined := "started"
	if msg.JoinedExistingRun {
		joined = "joined existing"
	}
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("%s import run %s (%d sources).", joined, msg.Run.Id, msg.Run.TotalSources)}, NextCommand: []string{fmt.Sprintf("`import-status --run-id %s` — view durable progress", msg.Run.Id)}}
}
func (h *handlers) statusReport(_ cliapp.OperationContext, msg *harnessv1.GetImportStatusResponse) cliapp.ListReport {
	if msg.Run == nil {
		return cliapp.ListReport{Summary: []string{"No import run found."}}
	}
	r := msg.Run
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Import %s: %s (%d/%d processed).", r.Id, r.Status, r.ProcessedSources, r.TotalSources)}, ResultsHeading: "Counters", Results: []string{fmt.Sprintf("imported: %d", r.ImportedCount), fmt.Sprintf("already present: %d", r.ExistingCount), fmt.Sprintf("failed: %d", r.FailedCount), fmt.Sprintf("checkpoint: %s", r.CurrentPath)}}
}
