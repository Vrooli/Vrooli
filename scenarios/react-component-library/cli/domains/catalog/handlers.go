package catalog

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	catalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog/catalog_v1connect"
)

type handlers struct {
	client catalogconnect.CatalogServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: catalogconnect.NewCatalogServiceClient(httpClient, baseURL)}
}

func (h *handlers) coverage(ctx cliapp.RunContext) error {
	resp, err := h.client.GetCoverage(context.Background(), connect.NewRequest(&catalogv1.GetCoverageRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get catalog coverage", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Report == nil {
		return fmt.Errorf("server returned no catalog coverage")
	}
	r := resp.Msg.Report
	summary := fmt.Sprintf("Catalog targets: %d; at/above target: %d.", r.Maturity.GetTotal(), r.Maturity.GetAtOrAboveTarget())
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{summary}, ResultsHeading: "Maturity distribution", Results: formatMaturity(r.Maturity.GetByRung())})
}

func (h *handlers) next(ctx cliapp.RunContext) error {
	resp, err := h.client.ListNextWork(context.Background(), connect.NewRequest(&catalogv1.ListNextWorkRequest{Limit: 10}))
	if err != nil {
		return cliapp.WrapAPIError("get catalog next work", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no catalog next work")
	}
	rows := make([]string, 0, len(resp.Msg.Rows))
	for _, row := range resp.Msg.Rows {
		rows = append(rows, fmt.Sprintf("%s [%s/%s] %s -> %s (blocks %d)", row.AssetId, row.Platform, row.Target, row.Achieved, row.Name, row.BlocksDownstream))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Next work: %d target(s).", len(rows))}, ResultsHeading: "Ranked next work", Results: rows})
}

func (h *handlers) gate(ctx cliapp.RunContext) error {
	gate := ctx.Positional("gate")
	resp, err := h.client.RunGate(context.Background(), connect.NewRequest(&catalogv1.RunGateRequest{Gate: gate}))
	if err != nil {
		return cliapp.WrapAPIError("run catalog gate", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no catalog gate result")
	}
	findings := make([]string, 0, len(resp.Msg.Findings))
	for _, finding := range resp.Msg.Findings {
		findings = append(findings, fmt.Sprintf("%s [%s] %s: %s", finding.AssetId, finding.Code, finding.Severity, finding.Message))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Gate %s: %d finding(s).", resp.Msg.Gate, len(findings))},
		ResultsHeading: "Findings",
		Results:        findings,
	})
}

func formatMaturity(values map[string]int32) []string {
	rows := make([]string, 0, len(values))
	for key, value := range values {
		rows = append(rows, fmt.Sprintf("%s: %d", key, value))
	}
	return rows
}
