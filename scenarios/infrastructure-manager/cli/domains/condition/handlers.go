package condition

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	conditionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/condition"
	conditionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/condition/condition_v1connect"
	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/coverage"
)

type handlers struct {
	client conditionconnect.ConditionServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: conditionconnect.NewConditionServiceClient(httpClient, baseURL)}
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*conditionv1.GetConditionResponse, error) {
	projection, err := projectionFlag(ctx.Flag("projection"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.GetCondition(context.Background(), connect.NewRequest(&conditionv1.GetConditionRequest{Projection: projection, CellRef: ctx.Flag("cell")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("condition status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no condition response")
	}
	return resp.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, msg *conditionv1.GetConditionResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Readings))
	for _, reading := range msg.Readings {
		results = append(results, formatReading(reading))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d condition reading(s).", len(results))}, ResultsHeading: "Readings", Results: results}
}

func (h *handlers) trustCall(_ cliapp.OperationContext) (*conditionv1.GetTrustDistributionResponse, error) {
	resp, err := h.client.GetTrustDistribution(context.Background(), connect.NewRequest(&conditionv1.GetTrustDistributionRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("condition trust", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no trust response")
	}
	return resp.Msg, nil
}

func (h *handlers) trustReport(_ cliapp.OperationContext, msg *conditionv1.GetTrustDistributionResponse) cliapp.ListReport {
	if msg.Trust == nil {
		return cliapp.ListReport{Summary: []string{"Trust distribution unavailable."}}
	}
	results := make([]string, 0, len(msg.Trust.Distribution))
	for _, item := range msg.Trust.Distribution {
		results = append(results, fmt.Sprintf("%s: %d", item.Verdict.String(), item.Count))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Trust checked=%d/%d.", msg.Trust.CheckedDenominator, msg.Trust.Total)}, ResultsHeading: "Trust distribution", Results: results}
}

func (h *handlers) explainCall(ctx cliapp.OperationContext) (*conditionv1.ExplainCellResponse, error) {
	cell := ctx.Positional("cell")
	resp, err := h.client.ExplainCell(context.Background(), connect.NewRequest(&conditionv1.ExplainCellRequest{CellRef: cell}))
	if err != nil {
		return nil, cliapp.WrapAPIError("condition explain", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no explanation response")
	}
	return resp.Msg, nil
}

func (h *handlers) explainReport(_ cliapp.OperationContext, msg *conditionv1.ExplainCellResponse) cliapp.ListReport {
	results := []string{}
	if msg.Cell != nil {
		results = append(results, fmt.Sprintf("%s: %s (%s)", msg.Cell.Id, msg.Cell.Question, msg.Cell.Owner))
	}
	if msg.Reading != nil {
		results = append(results, formatReading(msg.Reading))
	}
	return cliapp.ListReport{Summary: []string{"Condition cell explanation."}, ResultsHeading: "Explanation", Results: results}
}

func (h *handlers) historyCall(ctx cliapp.OperationContext) (*conditionv1.GetHistoryResponse, error) {
	limit := int32(0)
	if raw := strings.TrimSpace(ctx.Flag("limit")); raw != "" {
		if _, err := fmt.Sscan(raw, &limit); err != nil || limit < 0 {
			return nil, fmt.Errorf("invalid limit %q", raw)
		}
	}
	resp, err := h.client.GetHistory(context.Background(), connect.NewRequest(&conditionv1.GetHistoryRequest{CellRef: ctx.Positional("cell"), Limit: limit}))
	if err != nil {
		return nil, cliapp.WrapAPIError("condition history", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no history response")
	}
	return resp.Msg, nil
}

func (h *handlers) historyReport(_ cliapp.OperationContext, msg *conditionv1.GetHistoryResponse) cliapp.ListReport {
	if !msg.Measurable {
		return cliapp.ListReport{Summary: []string{"History unmeasurable: " + msg.UnmeasurableReason}}
	}
	results := make([]string, 0, len(msg.Readings))
	for _, reading := range msg.Readings {
		results = append(results, formatReading(reading))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d historical reading(s).", len(results))}, ResultsHeading: "History", Results: results}
}

func projectionFlag(raw string) (coveragev1.Projection, error) {
	name := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(raw), "-", "_"))
	if name == "" {
		// Unspecified means every projection with a live reader, not one
		// hard-coded default. Defaulting to availability hid the substrate
		// readings from anyone who did not already know to ask for them.
		return coveragev1.Projection_PROJECTION_UNSPECIFIED, nil
	}
	value, ok := coveragev1.Projection_value["PROJECTION_"+name]
	if !ok || value == 0 {
		return 0, fmt.Errorf("unknown projection %q", raw)
	}
	return coveragev1.Projection(value), nil
}

func formatReading(reading *conditionv1.Reading) string {
	if reading == nil {
		return "(nil reading)"
	}
	return fmt.Sprintf("%s [%s/%s] %.3f %s", reading.CellRef, reading.TrustVerdict.String(), reading.BandVerdict.String(), reading.Value, reading.Unit)
}
