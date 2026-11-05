package focus

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	focusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/focus"
	focusconnect "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/focus/focus_v1connect"
)

type handlers struct {
	client focusconnect.FocusServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: focusconnect.NewFocusServiceClient(httpClient, baseURL)}
}

func (h *handlers) nextCall(ctx cliapp.OperationContext) (*focusv1.GetNextResponse, error) {
	limit := int32(0)
	if raw := strings.TrimSpace(ctx.Flag("limit")); raw != "" {
		if _, err := fmt.Sscan(raw, &limit); err != nil || limit < 0 {
			return nil, fmt.Errorf("invalid limit %q", raw)
		}
	}
	resp, err := h.client.GetNext(context.Background(), connect.NewRequest(&focusv1.GetNextRequest{Limit: limit}))
	if err != nil {
		return nil, cliapp.WrapAPIError("focus next", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no focus response")
	}
	return resp.Msg, nil
}

func (h *handlers) nextReport(_ cliapp.OperationContext, msg *focusv1.GetNextResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Findings))
	for _, finding := range msg.Findings {
		results = append(results, formatFinding(finding))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d ranked finding(s); all sources unavailable=%t.", len(results), msg.AllSourcesUnavailable)}, ResultsHeading: "Next findings", Results: results}
}

func (h *handlers) findingCall(ctx cliapp.OperationContext) (*focusv1.GetFindingResponse, error) {
	resp, err := h.client.GetFinding(context.Background(), connect.NewRequest(&focusv1.GetFindingRequest{FindingId: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("focus finding", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no finding response")
	}
	return resp.Msg, nil
}

func (h *handlers) findingReport(_ cliapp.OperationContext, msg *focusv1.GetFindingResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Finding."}, ResultsHeading: "Finding", Results: []string{formatFinding(msg.Finding)}}
}

func (h *handlers) sourcesCall(_ cliapp.OperationContext) (*focusv1.ListSourcesResponse, error) {
	resp, err := h.client.ListSources(context.Background(), connect.NewRequest(&focusv1.ListSourcesRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("focus sources", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no source response")
	}
	return resp.Msg, nil
}

func (h *handlers) sourcesReport(_ cliapp.OperationContext, msg *focusv1.ListSourcesResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Sources))
	for _, source := range msg.Sources {
		results = append(results, fmt.Sprintf("%s available=%t findings=%d — %s", source.Id, source.Available, source.FindingCount, source.Reason))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d focus source(s).", len(results))}, ResultsHeading: "Sources", Results: results}
}

func (h *handlers) efficacyCall(ctx cliapp.OperationContext) (*focusv1.GetEfficacyResponse, error) {
	resp, err := h.client.GetEfficacy(context.Background(), connect.NewRequest(&focusv1.GetEfficacyRequest{FindingId: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("focus efficacy", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no efficacy response")
	}
	return resp.Msg, nil
}

func (h *handlers) efficacyReport(_ cliapp.OperationContext, msg *focusv1.GetEfficacyResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Records))
	for _, record := range msg.Records {
		results = append(results, fmt.Sprintf("%s: %s — %s", record.FindingId, record.Verdict.String(), record.ObservedReturn))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d efficacy record(s).", len(results))}, ResultsHeading: "Efficacy", Results: results}
}

func formatFinding(finding *focusv1.Finding) string {
	if finding == nil {
		return "(nil finding)"
	}
	rank := int32(0)
	stage := ""
	if finding.Rationale != nil {
		rank = finding.Rationale.Rank
		stage = finding.Rationale.CascadeStage
	}
	return fmt.Sprintf("#%d %s [%s] %s — %s", rank, finding.Id, stage, finding.Title, finding.Message)
}
