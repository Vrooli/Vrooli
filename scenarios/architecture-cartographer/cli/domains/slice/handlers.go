package slice

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph/graph_v1connect"

	"architecture-cartographer/cli/internal/attestrender"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client graphconnect.GraphServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: graphconnect.NewGraphServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) show(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	domain := ctx.Positional("domain")
	resp, err := h.client.GetSlice(context.Background(), connect.NewRequest(&graphv1.GetSliceRequest{
		Scenario:   scenario,
		Domain:     domain,
		SnapshotId: ctx.Flag("snapshot-id"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("show slice for %q/%q", scenario, domain), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetSlice() == nil {
		return fmt.Errorf("server returned no slice")
	}
	s := resp.Msg.GetSlice()
	results := make([]string, 0, len(s.GetRungs()))
	for _, rung := range s.GetRungs() {
		status := "missing"
		if rung.GetPresent() {
			status = "present"
		}
		results = append(results, fmt.Sprintf("%s: %s (%d evidence, %d file(s), %d symbol(s))",
			rung.GetName(), status, len(rung.GetEvidence()), len(rung.GetFiles()), len(rung.GetSymbols())))
	}
	summary := []string{
		fmt.Sprintf("Slice for %q in %q.", s.GetDomain(), s.GetScenario()),
		fmt.Sprintf("Snapshot: %s", s.GetSnapshotId()),
		fmt.Sprintf("Surfaces: %s", formatSurfaces(s.GetSurfaces())),
		fmt.Sprintf("Layer edges: %s", formatEdges(s.GetLayerEdges())),
	}
	if att := s.GetAttestation(); att != nil {
		summary = append(summary, fmt.Sprintf("Slice basis: %s (%s)", attestrender.Basis(att.GetBasis()), attestrender.Sufficiency(att.GetSufficiency())))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Rungs",
		Results:        results,
		RetrievalHints: []string{"Use --json for per-rung files, exported symbols, typed evidence, and layer edges."},
	})
}

func formatEdges(edges []*graphv1.SliceEdge) string {
	if len(edges) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(edges))
	for _, e := range edges {
		parts = append(parts, e.GetFromRung()+"->"+e.GetToRung())
	}
	return strings.Join(parts, ", ")
}

func formatSurfaces(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	out := values[0]
	for _, value := range values[1:] {
		out += ", " + value
	}
	return out
}
