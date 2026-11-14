package runs

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/validation/validation_v1connect"
)

type handlers struct {
	client validationconnect.ValidationRunServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: validationconnect.NewValidationRunServiceClient(httpClient, baseURL)}
}

func (h *handlers) runCall(ctx cliapp.OperationContext) (*validationv1.RunTemplateValidationResponse, error) {
	resp, err := h.client.RunTemplateValidation(context.Background(), connect.NewRequest(&validationv1.RunTemplateValidationRequest{
		TemplateId: ctx.Flag("template"),
		Mode:       parseMode(ctx.Flag("mode")),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("run template validation", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) runReport(_ cliapp.OperationContext, msg *validationv1.RunTemplateValidationResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Recorded validation run %s.", msg.Run.Id)},
		Changes: []string{
			formatRun(msg.Run),
		},
		NextCommand: []string{
			fmt.Sprintf("`template-manager runs show %s` - inspect the stored run", msg.Run.Id),
			"`template-manager debt list --template " + msg.Run.TemplateId + "` - inspect debt entries",
		},
	}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*validationv1.ListValidationRunsResponse, error) {
	resp, err := h.client.ListValidationRuns(context.Background(), connect.NewRequest(&validationv1.ListValidationRunsRequest{TemplateId: ctx.Flag("template")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list validation runs", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, msg *validationv1.ListValidationRunsResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Runs))
	for _, run := range msg.Runs {
		results = append(results, formatRun(run))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d validation run(s).", len(msg.Runs))}, ResultsHeading: "Validation runs", Results: results}
}

func (h *handlers) showCall(ctx cliapp.OperationContext) (*validationv1.GetValidationRunResponse, error) {
	id := ctx.Positional("id")
	resp, err := h.client.GetValidationRun(context.Background(), connect.NewRequest(&validationv1.GetValidationRunRequest{Id: id}))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("show validation run %q", id), err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) showReport(_ cliapp.OperationContext, msg *validationv1.GetValidationRunResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Fetched validation run %s.", msg.Run.Id)}, ResultsHeading: "Validation run", Results: []string{formatRun(msg.Run)}}
}

func (h *handlers) driftRecordCall(_ cliapp.OperationContext) (*validationv1.RecordFleetDriftResponse, error) {
	resp, err := h.client.RecordFleetDrift(context.Background(), connect.NewRequest(&validationv1.RecordFleetDriftRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("record fleet drift", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) driftRecordReport(_ cliapp.OperationContext, msg *validationv1.RecordFleetDriftResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Recorded drift snapshot %s.", msg.Snapshot.Id)},
		Changes: []string{
			fmt.Sprintf("template=%s target=%s status=%s drift=%d", msg.Snapshot.TemplateId, msg.Snapshot.Target, msg.Snapshot.Status, msg.Snapshot.DriftCount),
		},
		NextCommand: []string{
			"`template-manager runs drift --template react-vite` - list drift snapshots",
			"`template-manager debt list --template react-vite` - inspect drift debt",
		},
	}
}

func (h *handlers) driftCall(ctx cliapp.OperationContext) (*validationv1.ListDriftSnapshotsResponse, error) {
	resp, err := h.client.ListDriftSnapshots(context.Background(), connect.NewRequest(&validationv1.ListDriftSnapshotsRequest{TemplateId: ctx.Flag("template")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list drift snapshots", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) driftReport(_ cliapp.OperationContext, msg *validationv1.ListDriftSnapshotsResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Snapshots))
	for _, snapshot := range msg.Snapshots {
		results = append(results, fmt.Sprintf("%s template=%s target=%s status=%s drift=%d", snapshot.Id, snapshot.TemplateId, snapshot.Target, snapshot.Status, snapshot.DriftCount))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d drift snapshot(s).", len(msg.Snapshots))}, ResultsHeading: "Drift snapshots", Results: results}
}

func formatRun(run *validationv1.ValidationRun) string {
	if run == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s template=%s mode=%s status=%s trigger=%s findings=%d", run.Id, run.TemplateId, run.Mode.String(), run.Status, run.Trigger, len(run.Findings))
}

func parseMode(value string) validationv1.ValidationMode {
	switch value {
	case "deep":
		return validationv1.ValidationMode_VALIDATION_MODE_DEEP
	case "drift":
		return validationv1.ValidationMode_VALIDATION_MODE_DRIFT
	default:
		return validationv1.ValidationMode_VALIDATION_MODE_SHALLOW
	}
}
