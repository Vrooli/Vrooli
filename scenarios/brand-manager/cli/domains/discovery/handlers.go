package discovery

import (
	"context"
	"fmt"
	"strings"

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
	summary := []string{fmt.Sprintf(
		"Discovered %d source(s) in %s (confidence %.0f%%).",
		len(m.Sources), m.Scenario, m.Confidence*100,
	)}
	summary = append(summary, draftLines(m.DraftBrand)...)
	return cliapp.RenderProtoList(ctx, m, cliapp.ListReport{
		Summary:        summary,
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
			fmt.Sprintf("`brands get %s` — inspect the imported brand", m.BrandId),
		},
	})
}

// draftLines renders the facets a scan actually inferred, so the human output
// shows the values an import would persist rather than only which files matched.
func draftLines(d *discoveryv1.DraftBrand) []string {
	if d == nil {
		return nil
	}
	var out []string
	if id := d.Identity; id != nil && id.DisplayName != "" {
		out = append(out, fmt.Sprintf("Identity: %s", id.DisplayName))
	}
	if c := d.Colors; c != nil {
		var pairs []string
		for _, e := range []struct{ label, value string }{
			{"primary", c.Primary},
			{"secondary", c.Secondary},
			{"accent", c.Accent},
			{"background", c.Background},
			{"surface", c.Surface},
			{"text", c.Text},
			{"error", c.Error},
		} {
			if e.value != "" {
				pairs = append(pairs, fmt.Sprintf("%s=%s", e.label, e.value))
			}
		}
		if len(pairs) > 0 {
			out = append(out, "Colors: "+strings.Join(pairs, " "))
		}
	}
	return out
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
