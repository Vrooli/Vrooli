package discovery

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/discovery"
	discoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/discovery/discovery_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each RunCtx-func has
// typed access to the API client.
type handlers struct {
	core   *cliapp.ScenarioApp
	client discoveryconnect.DiscoveryServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: discoveryconnect.NewDiscoveryServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) scan(ctx cliapp.RunContext) error {
	resp, err := h.client.DiscoverScenario(context.Background(), connect.NewRequest(&discoveryv1.DiscoverScenarioRequest{
		ScenarioName: ctx.Flag("scenario"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("discover scenario", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no discovery response")
	}
	m := resp.Msg
	hints := suggestionLines(m.Suggestions)
	if m.DraftBrand != nil {
		hints = append(hints, fmt.Sprintf("`discovery import --scenario %s` — create a brand from this state", m.Scenario))
	}
	return cliapp.RenderProtoList(ctx, m, cliapp.ListReport{
		Summary: []string{fmt.Sprintf(
			"Discovered %d source(s) in %s (confidence %.0f%%).",
			len(m.Sources), m.Scenario, m.Confidence*100,
		)},
		ResultsHeading: "Sources",
		Results:        sourceLines(m.Sources),
		RetrievalHints: hints,
	})
}

func (h *handlers) importBrand(ctx cliapp.RunContext) error {
	resp, err := h.client.ImportBrand(context.Background(), connect.NewRequest(&discoveryv1.ImportBrandRequest{
		ScenarioName: ctx.Flag("scenario"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("import brand", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no import response")
	}
	m := resp.Msg
	return cliapp.RenderProtoMutation(ctx, m, cliapp.MutationReport{
		Result: []string{fmt.Sprintf(
			"Imported brand %s (%q, v%d) from %d discovered source(s) (confidence %.0f%%).",
			m.BrandId, m.BrandName, m.BrandVersion, len(m.Sources), m.Confidence*100,
		)},
		Changes: sourceLines(m.Sources),
		NextCommand: []string{
			fmt.Sprintf("`brands get --id %s` — inspect the imported brand", m.BrandId),
		},
	})
}

// sourceLines renders each discovered source as a human line.
func sourceLines(sources []*discoveryv1.DiscoverySource) []string {
	out := make([]string, 0, len(sources))
	for _, s := range sources {
		out = append(out, fmt.Sprintf("%s — %s (%d field(s), %.0f%% confidence)", s.File, s.Type, s.Fields, s.Confidence*100))
	}
	return out
}

// suggestionLines renders each suggestion as a hint line.
func suggestionLines(suggestions []string) []string {
	out := make([]string, 0, len(suggestions))
	out = append(out, suggestions...)
	return out
}
