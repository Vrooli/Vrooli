package conformance

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	conformancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/conformance"
	conformanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/conformance/conformance_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	client conformanceconnect.ConformanceServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: conformanceconnect.NewConformanceServiceClient(httpClient, baseURL)}
}

func (h *handlers) scan(ctx cliapp.RunContext) error {
	resp, err := h.client.ScanScenario(context.Background(), connect.NewRequest(&conformancev1.ScanScenarioRequest{
		Scenario: ctx.Flag("scenario"),
		Path:     ctx.Flag("path"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("scan AI conformance", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no conformance scan")
	}
	results := make([]string, 0, len(resp.Msg.GetFindings())+len(resp.Msg.GetRecommendations()))
	for _, finding := range resp.Msg.GetFindings() {
		results = append(results, fmt.Sprintf("%s %s %s: %s", finding.GetSeverity(), finding.GetRuleId(), finding.GetPath(), finding.GetMessage()))
	}
	for _, rec := range resp.Msg.GetRecommendations() {
		results = append(results, "recommendation: "+rec)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("AI conformance scan for %s: maturity=%s findings=%d.", resp.Msg.GetScenario(), resp.Msg.GetMaturityLevel(), len(resp.Msg.GetFindings()))},
		ResultsHeading: "Findings",
		Results:        results,
		RetrievalHints: []string{"`validation validate --scenario <scenario>` — run the shared Test Genie provider contract shape"},
	})
}
