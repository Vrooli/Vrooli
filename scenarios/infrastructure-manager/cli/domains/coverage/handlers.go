package coverage

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/coverage"
	coverageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/coverage/coverage_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/shared"
)

type handlers struct {
	client coverageconnect.CoverageServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: coverageconnect.NewCoverageServiceClient(httpClient, baseURL)}
}

func (h *handlers) statusCall(ctx cliapp.OperationContext) (*coveragev1.GetCoverageResponse, error) {
	projection, err := projectionFlag(ctx.Flag("projection"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.GetCoverage(context.Background(), connect.NewRequest(&coveragev1.GetCoverageRequest{Projections: projectionList(projection)}))
	if err != nil {
		return nil, cliapp.WrapAPIError("coverage status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no coverage response")
	}
	return resp.Msg, nil
}

func (h *handlers) statusReport(_ cliapp.OperationContext, msg *coveragev1.GetCoverageResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Projections))
	for _, p := range msg.Projections {
		results = append(results, formatProjection(p))
	}
	for _, f := range msg.IntegrityFindings {
		results = append(results, fmt.Sprintf("[%s] %s — %s", f.Severity, f.Code, f.Message))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Coverage across %d projection(s).", len(msg.Projections))}, ResultsHeading: "Projections", Results: results, RetrievalHints: []string{"`coverage cells --projection availability`", "`coverage open-loop`"}}
}

func (h *handlers) cellsCall(ctx cliapp.OperationContext) (*coveragev1.ListCellsResponse, error) {
	projection, err := projectionFlag(ctx.Flag("projection"))
	if err != nil {
		return nil, err
	}
	status, err := statusFlag(ctx.Flag("status"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.ListCells(context.Background(), connect.NewRequest(&coveragev1.ListCellsRequest{Projection: projection, Status: status}))
	if err != nil {
		return nil, cliapp.WrapAPIError("coverage cells", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no cells response")
	}
	return resp.Msg, nil
}

func (h *handlers) openLoopCall(ctx cliapp.OperationContext) (*coveragev1.ListCellsResponse, error) {
	projection, err := optionalProjection(ctx.Flag("projection"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.ListOpenLoopCells(context.Background(), connect.NewRequest(&coveragev1.ListCellsRequest{Projection: projection}))
	if err != nil {
		return nil, cliapp.WrapAPIError("coverage open-loop", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no open-loop response")
	}
	return resp.Msg, nil
}

func (h *handlers) cellsReport(_ cliapp.OperationContext, msg *coveragev1.ListCellsResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Cells))
	for _, c := range msg.Cells {
		results = append(results, formatCell(c))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d cell(s).", len(msg.Cells))}, ResultsHeading: "Cells", Results: results}
}

func (h *handlers) showCall(ctx cliapp.OperationContext) (*coveragev1.GetProjectionResponse, error) {
	projection, err := projectionFlag(ctx.Positional("projection"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.GetProjection(context.Background(), connect.NewRequest(&coveragev1.GetProjectionRequest{Projection: projection}))
	if err != nil {
		return nil, cliapp.WrapAPIError("coverage show", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no projection response")
	}
	return resp.Msg, nil
}

func (h *handlers) showReport(_ cliapp.OperationContext, msg *coveragev1.GetProjectionResponse) cliapp.ListReport {
	results := []string{formatProjection(msg.Coverage)}
	for _, c := range msg.Cells {
		results = append(results, formatCell(c))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Projection %s.", msg.Projection.String())}, ResultsHeading: "Projection", Results: results}
}

func (h *handlers) validateCall(_ cliapp.OperationContext) (*coveragev1.ValidateSetpointResponse, error) {
	resp, err := h.client.ValidateSetpoint(context.Background(), connect.NewRequest(&coveragev1.ValidateSetpointRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("coverage validate", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no validation response")
	}
	return resp.Msg, nil
}

func (h *handlers) validateReport(_ cliapp.OperationContext, msg *coveragev1.ValidateSetpointResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Findings))
	for _, f := range msg.Findings {
		results = append(results, fmt.Sprintf("[%s] %s — %s", f.Severity, f.Code, f.Message))
	}
	verdict := "valid"
	if !msg.Ok {
		verdict = "INVALID"
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Setpoint: %s (%d finding(s)).", verdict, len(results))}, ResultsHeading: "Integrity findings", Results: results}
}

func (h *handlers) driftCall(ctx cliapp.OperationContext) (*coveragev1.GetDriftResponse, error) {
	projection, err := optionalProjection(ctx.Flag("projection"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.GetDrift(context.Background(), connect.NewRequest(&coveragev1.GetDriftRequest{Projection: projection}))
	if err != nil {
		return nil, cliapp.WrapAPIError("coverage drift", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no drift response")
	}
	return resp.Msg, nil
}

func (h *handlers) driftReport(_ cliapp.OperationContext, msg *coveragev1.GetDriftResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Findings))
	for _, f := range msg.Findings {
		results = append(results, fmt.Sprintf("[%s] %s — %s", f.Code, f.CellRef, f.Message))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d drift finding(s).", len(results))}, ResultsHeading: "Drift", Results: results}
}

func projectionList(p coveragev1.Projection) []coveragev1.Projection {
	if p == coveragev1.Projection_PROJECTION_UNSPECIFIED {
		return nil
	}
	return []coveragev1.Projection{p}
}

func optionalProjection(raw string) (coveragev1.Projection, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	return projectionFlag(raw)
}

func projectionFlag(raw string) (coveragev1.Projection, error) {
	name := strings.ToLower(strings.TrimSpace(raw))
	name = strings.ReplaceAll(name, "-", "_")
	if name == "" {
		return 0, nil
	}
	value, ok := coveragev1.Projection_value["PROJECTION_"+strings.ToUpper(name)]
	if !ok || value == 0 {
		return 0, fmt.Errorf("unknown projection %q", raw)
	}
	return coveragev1.Projection(value), nil
}

func statusFlag(raw string) (coveragev1.CellStatus, error) {
	name := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(raw), "-", "_"))
	if name == "" {
		return 0, nil
	}
	value, ok := coveragev1.CellStatus_value["CELL_STATUS_"+strings.ToUpper(name)]
	if !ok || value == 0 {
		return 0, fmt.Errorf("unknown status %q", raw)
	}
	return coveragev1.CellStatus(value), nil
}

func formatProjection(p *coveragev1.ProjectionCoverage) string {
	if p == nil {
		return "(nil projection)"
	}
	ratio := p.Ratio
	if ratio == nil {
		return p.Projection.String() + ": unavailable"
	}
	return fmt.Sprintf("%s: %.0f%% NOW (%d/%d), in_reach=%d, missing=%d, confidence=%s", strings.ToLower(strings.TrimPrefix(p.Projection.String(), "PROJECTION_")), ratio.Value*100, p.NowCount, p.TotalCells, p.InReachCount, p.MissingCount, p.Confidence.Level.String())
}

func formatCell(c *sharedv1.Cell) string {
	return fmt.Sprintf("%s [%s] %s — %s", c.Id, strings.ToLower(strings.TrimPrefix(c.Status.String(), "CELL_STATUS_")), c.Question, c.Owner)
}
