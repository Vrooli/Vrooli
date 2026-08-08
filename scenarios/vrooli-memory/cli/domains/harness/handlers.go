package harness

import (
	"context"
	"fmt"
	"time"

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

func projectRuntime(ctx cliapp.OperationContext) string {
	if v := ctx.Flag("harness"); v != "" {
		return v
	}
	return runtime(ctx)
}

func (h *handlers) importCall(ctx cliapp.OperationContext) (*harnessv1.RunImportResponse, error) {
	resp, err := h.client.RunImport(context.Background(), connect.NewRequest(&harnessv1.RunImportRequest{Runtime: runtime(ctx), DryRun: ctx.BoolFlag("dry-run")}))
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

func (h *handlers) projectCall(ctx cliapp.OperationContext) (*harnessv1.RefreshProjectionResponse, error) {
	resp, err := h.client.RefreshProjection(context.Background(), connect.NewRequest(&harnessv1.RefreshProjectionRequest{Runtime: projectRuntime(ctx), DryRun: ctx.BoolFlag("dry-run")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("project unified memory", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) captureCall(ctx cliapp.OperationContext) (*harnessv1.CaptureWriteResponse, error) {
	resp, err := h.client.CaptureWrite(context.Background(), connect.NewRequest(&harnessv1.CaptureWriteRequest{Runtime: runtime(ctx), SourcePath: ctx.Flag("source-path"), Content: ctx.Flag("content")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("capture native memory", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) promptCall(ctx cliapp.OperationContext) (*harnessv1.InstallPromptBlockResponse, error) {
	resp, err := h.client.InstallPromptBlock(context.Background(), connect.NewRequest(&harnessv1.InstallPromptBlockRequest{Runtime: runtime(ctx)}))
	if err != nil {
		return nil, cliapp.WrapAPIError("install memory prompt", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) maintenanceCall(ctx cliapp.OperationContext) (*harnessv1.GetMaintenanceStatusResponse, error) {
	resp, err := h.client.GetMaintenanceStatus(context.Background(), connect.NewRequest(&harnessv1.GetMaintenanceStatusRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("read maintenance status", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) importReport(_ cliapp.OperationContext, msg *harnessv1.RunImportResponse) cliapp.MutationReport {
	if msg.DryRun {
		if msg.Observation != "" && msg.ImportedCount == 0 {
			return cliapp.MutationReport{Result: []string{fmt.Sprintf("Dry run found no importable memory sources; no journal entries were written. Observation: %s", msg.Observation)}}
		}
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Dry run validated %d importable memory source(s); no journal entries were written.", msg.ImportedCount)}}
	}
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
	results := []string{fmt.Sprintf("imported: %d", r.ImportedCount), fmt.Sprintf("already present: %d", r.ExistingCount), fmt.Sprintf("failed: %d", r.FailedCount), fmt.Sprintf("checkpoint: %s", r.CurrentPath)}
	if rate, eta, ok := importProgressEstimate(r, time.Now()); ok {
		results = append(results, fmt.Sprintf("throughput: %.1f sources/min", rate), fmt.Sprintf("estimated remaining: %s", eta.Round(time.Second)))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Import %s: %s (%d/%d processed).", r.Id, r.Status, r.ProcessedSources, r.TotalSources)}, ResultsHeading: "Counters", Results: results}
}

// importProgressEstimate makes a long-running, durable import observable
// without guessing from polling intervals. It derives rate from the run's own
// persisted start time and only projects an ETA for an in-progress run with a
// positive observed rate.
func importProgressEstimate(run *harnessv1.ImportRun, now time.Time) (float64, time.Duration, bool) {
	if run.GetStatus() != "running" || run.GetProcessedSources() <= 0 || run.GetTotalSources() <= run.GetProcessedSources() {
		return 0, 0, false
	}
	started, err := time.Parse(time.RFC3339Nano, run.GetStartedAt())
	if err != nil || !now.After(started) {
		return 0, 0, false
	}
	elapsed := now.Sub(started)
	ratePerSecond := float64(run.GetProcessedSources()) / elapsed.Seconds()
	if ratePerSecond <= 0 {
		return 0, 0, false
	}
	return ratePerSecond * 60, time.Duration(float64(run.GetTotalSources()-run.GetProcessedSources())/ratePerSecond) * time.Second, true
}

func (h *handlers) projectReport(_ cliapp.OperationContext, msg *harnessv1.RefreshProjectionResponse) cliapp.MutationReport {
	if msg.DryRun {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Dry run rendered %d lines / %d bytes for %s (ceilings %d lines / %d bytes, overflow=%t).", msg.SizeLines, msg.SizeBytes, msg.Path, msg.LineCap, msg.ByteCap, msg.Overflow), msg.RenderedContent}}
	}
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Projected %d lines / %d bytes to %s (ceilings %d lines / %d bytes, overflow=%t).", msg.SizeLines, msg.SizeBytes, msg.Path, msg.LineCap, msg.ByteCap, msg.Overflow)}}
}

func (h *handlers) captureReport(_ cliapp.OperationContext, msg *harnessv1.CaptureWriteResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Captured native memory as %s.", msg.EntryId)}}
}

func (h *handlers) promptReport(_ cliapp.OperationContext, msg *harnessv1.InstallPromptBlockResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Memory prompt installed: %t.", msg.Installed)}}
}

func (h *handlers) maintenanceReport(_ cliapp.OperationContext, msg *harnessv1.GetMaintenanceStatusResponse) cliapp.ListReport {
	if msg.Run == nil {
		return cliapp.ListReport{Summary: []string{"No maintenance run has completed."}}
	}
	results := make([]string, 0, len(msg.Run.Outcomes))
	for _, outcome := range msg.Run.Outcomes {
		results = append(results, fmt.Sprintf("%s: import=%s projection=%s usage=%d/%d lines, %d/%d bytes", outcome.Runtime, outcome.ImportStatus, outcome.ProjectionStatus, outcome.ProjectionSizeLines, outcome.ProjectionLineCap, outcome.ProjectionSizeBytes, outcome.ProjectionByteCap))
		if outcome.ImportError != "" {
			results = append(results, "  import error: "+outcome.ImportError)
		}
		if outcome.ProjectionError != "" {
			results = append(results, "  projection error: "+outcome.ProjectionError)
		}
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Maintenance %s: started %s, completed %s.", msg.Run.Id, msg.Run.StartedAt, msg.Run.CompletedAt)}, ResultsHeading: "Runtime outcomes", Results: results}
}
