package zones

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph/graph_v1connect"

	"architecture-cartographer/cli/internal/attestrender"

	"github.com/vrooli/cli-core/cliapp"
)

func authorityConfidenceName(c graphv1.AuthorityConfidence) string {
	switch c {
	case graphv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_HIGH:
		return "high (ARCHITECTURE.md Zone Map declared)"
	case graphv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_LOW:
		return "low (derived-only — no ARCHITECTURE.md Zone Map)"
	default:
		return "unspecified"
	}
}

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
	resp, err := h.client.GetZoneMap(context.Background(), connect.NewRequest(&graphv1.GetZoneMapRequest{
		Scenario:   scenario,
		SnapshotId: ctx.Flag("snapshot-id"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("show zone map for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetZoneMap() == nil {
		return fmt.Errorf("server returned no zone map")
	}
	zoneMap := resp.Msg.GetZoneMap()
	results := make([]string, 0, len(zoneMap.GetPackages()))
	for _, pkg := range zoneMap.GetPackages() {
		domain := pkg.GetDomain()
		if domain == "" {
			domain = "-"
		}
		zone := pkg.GetZone()
		if zone == "" {
			zone = "unknown"
		}
		line := fmt.Sprintf("%s -> %s (domain: %s, confidence: %.2f)", pkg.GetRepoPath(), zone, domain, pkg.GetConfidence())
		if pkg.GetDrift() {
			line += fmt.Sprintf(" — DRIFT: declared %q", pkg.GetDeclaredLayer())
		}
		results = append(results, line)
	}
	summary := []string{
		fmt.Sprintf("Classified %d package(s) for %q.", len(zoneMap.GetPackages()), zoneMap.GetScenario()),
		fmt.Sprintf("Snapshot: %s", zoneMap.GetSnapshotId()),
		fmt.Sprintf("Layering violations: %d", len(zoneMap.GetViolations())),
		fmt.Sprintf("Authority confidence: %s", authorityConfidenceName(zoneMap.GetAuthorityConfidence())),
	}
	if att := zoneMap.GetAttestation(); att != nil {
		summary = append(summary, fmt.Sprintf("Zone map basis: %s (%s)", attestrender.Basis(att.GetBasis()), attestrender.Sufficiency(att.GetSufficiency())))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Packages",
		Results:        results,
		RetrievalHints: []string{"Use --json for the package zone map and layering cross-check violations."},
	})
}
