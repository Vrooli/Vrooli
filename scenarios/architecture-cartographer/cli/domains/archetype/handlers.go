package archetype

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	domainsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/domains"
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

func (h *handlers) infer(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.InferArchetype(context.Background(), connect.NewRequest(&graphv1.InferArchetypeRequest{
		Scenario:   scenario,
		Domain:     ctx.Flag("domain"),
		SnapshotId: ctx.Flag("snapshot-id"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("infer archetypes for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no archetype report")
	}
	reports := resp.Msg.GetReports()
	results := make([]string, 0, len(reports))
	drift := 0
	for _, r := range reports {
		line := fmt.Sprintf("%s: %s", r.GetDomain(), archetypeSummary(r.GetArchetypes()))
		if r.GetConvergenceDrift() {
			drift++
			line += " — DRIFT (declared != inferred)"
		}
		if att := r.GetAttestation(); att != nil {
			line += fmt.Sprintf(" [%s]", attestrender.Basis(att.GetBasis()))
		}
		results = append(results, line)
	}
	summary := []string{
		fmt.Sprintf("Inferred archetypes for %d domain(s) in %q.", len(reports), scenario),
		fmt.Sprintf("Convergence drift: %d domain(s)", drift),
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Archetypes",
		Results:        results,
		RetrievalHints: []string{"Use --json for per-archetype confidence, evidence, and the full attestation."},
	})
}

// archetypeSummary renders the declared and inferred archetypes for one domain
// as "declared=<a,b> inferred=<c,d>".
func archetypeSummary(archetypes []*domainsv1.DomainArchetype) string {
	var declared, inferred []string
	for _, a := range archetypes {
		label := archetypeLabel(a)
		if label == "" {
			continue
		}
		switch a.GetSource() {
		case domainsv1.ArchetypeSource_ARCHETYPE_SOURCE_DECLARED:
			declared = append(declared, label)
		case domainsv1.ArchetypeSource_ARCHETYPE_SOURCE_INFERRED:
			inferred = append(inferred, fmt.Sprintf("%s(%.2f)", label, a.GetConfidence()))
		}
	}
	return fmt.Sprintf("declared=[%s] inferred=[%s]", strings.Join(declared, ", "), strings.Join(inferred, ", "))
}

func archetypeLabel(a *domainsv1.DomainArchetype) string {
	switch a.GetArchetype() {
	case domainsv1.Archetype_ARCHETYPE_REPORTING:
		return "reporting"
	case domainsv1.Archetype_ARCHETYPE_SERVICE:
		return "service"
	case domainsv1.Archetype_ARCHETYPE_MUTATION:
		return "mutation"
	case domainsv1.Archetype_ARCHETYPE_CLASSIFICATION:
		return "classification"
	case domainsv1.Archetype_ARCHETYPE_ORCHESTRATION:
		return "orchestration"
	case domainsv1.Archetype_ARCHETYPE_SCORING:
		return "scoring"
	case domainsv1.Archetype_ARCHETYPE_QUERY:
		return "query"
	default:
		return a.GetDeclaredLabel()
	}
}
