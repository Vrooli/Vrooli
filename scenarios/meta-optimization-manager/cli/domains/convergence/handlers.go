package convergence

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	convergencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/convergence"
	convergenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/convergence/convergence_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client convergenceconnect.ConvergenceServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: convergenceconnect.NewConvergenceServiceClient(httpClient, baseURL)}
}

func (h *handlers) status(ctx cliapp.RunContext) error {
	resp, err := h.client.GetConvergenceStatus(context.Background(), connect.NewRequest(&convergencev1.GetConvergenceStatusRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("convergence status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no status response")
	}
	results := make([]string, 0, len(resp.Msg.Templates)+len(resp.Msg.References))
	for _, tf := range resp.Msg.Templates {
		results = append(results, formatFitness(tf))
	}
	for _, rh := range resp.Msg.References {
		results = append(results, formatReference(rh))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d template(s), %d reference(s).", len(resp.Msg.Templates), len(resp.Msg.References))},
		ResultsHeading: "Convergence",
		Results:        results,
		RetrievalHints: []string{
			"`convergence fitness --template <t>` — four-lens detail for one template",
			"`convergence trend --template <t>` — the compounding proof over time",
		},
	})
}

func (h *handlers) fitness(ctx cliapp.RunContext) error {
	resp, err := h.client.GetTemplateFitness(context.Background(), connect.NewRequest(&convergencev1.GetTemplateFitnessRequest{
		Template: strings.TrimSpace(ctx.Flag("template")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("convergence fitness", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no fitness response")
	}
	results := make([]string, 0, len(resp.Msg.Templates))
	for _, tf := range resp.Msg.Templates {
		results = append(results, formatFitness(tf))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d template(s).", len(resp.Msg.Templates))},
		ResultsHeading: "Template fitness",
		Results:        results,
	})
}

func (h *handlers) references(ctx cliapp.RunContext) error {
	eligibility, err := eligibilityFlag(ctx)
	if err != nil {
		return err
	}
	resp, err := h.client.ListReferences(context.Background(), connect.NewRequest(&convergencev1.ListReferencesRequest{Eligibility: eligibility}))
	if err != nil {
		return cliapp.WrapAPIError("convergence references", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no references response")
	}
	results := make([]string, 0, len(resp.Msg.References))
	for _, rh := range resp.Msg.References {
		results = append(results, formatReference(rh))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d reference(s).", len(resp.Msg.References))},
		ResultsHeading: "Reference health",
		Results:        results,
	})
}

func (h *handlers) trend(ctx cliapp.RunContext) error {
	resp, err := h.client.GetConvergenceTrend(context.Background(), connect.NewRequest(&convergencev1.GetConvergenceTrendRequest{
		Template: strings.TrimSpace(ctx.Flag("template")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("convergence trend", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no trend response")
	}
	results := make([]string, 0, len(resp.Msg.Points))
	for _, p := range resp.Msg.Points {
		results = append(results, fmt.Sprintf("%s @ %s — per_replica_cost=%d coordinated_edits=%d",
			p.GetTemplate(), p.GetAt().AsTime().Format("2006-01-02"), p.GetPerReplicaCost(), p.GetCoordinatedEditCount()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d trend point(s).", len(resp.Msg.Points))},
		ResultsHeading: "Convergence trend",
		Results:        results,
	})
}

func eligibilityFlag(ctx cliapp.RunContext) (convergencev1.ReferenceEligibility, error) {
	switch strings.ToLower(strings.TrimSpace(ctx.Flag("eligibility"))) {
	case "":
		return convergencev1.ReferenceEligibility_REFERENCE_ELIGIBILITY_UNSPECIFIED, nil
	case "eligible":
		return convergencev1.ReferenceEligibility_REFERENCE_ELIGIBILITY_ELIGIBLE, nil
	case "candidate":
		return convergencev1.ReferenceEligibility_REFERENCE_ELIGIBILITY_CANDIDATE, nil
	case "ineligible":
		return convergencev1.ReferenceEligibility_REFERENCE_ELIGIBILITY_INELIGIBLE, nil
	default:
		return 0, fmt.Errorf("unknown eligibility %q (use eligible|candidate|ineligible)", ctx.Flag("eligibility"))
	}
}

func formatFitness(tf *convergencev1.TemplateFitness) string {
	return fmt.Sprintf("%s [%s] per_replica=%d drift=%d comment_contracts=%d coordinated_edits=%d",
		tf.GetTemplate(), tierLabel(tf.GetTier()), tf.GetPerReplicaCost(), tf.GetDriftSurfaceCount(),
		tf.GetCommentOnlyContractCount(), tf.GetCoordinatedEditCount())
}

func formatReference(rh *convergencev1.ReferenceHealth) string {
	clean := "dirty"
	if rh.GetCleanOnAllTools() {
		clean = "clean"
	}
	stale := ""
	if rh.GetStaleFromTemplate() {
		stale = " stale-from-template"
	}
	return fmt.Sprintf("%s [%s] %s stability=%dd breadth=%d%s",
		rh.GetScenario(), eligibilityLabel(rh.GetEligibility()), clean, rh.GetStabilityDays(), rh.GetBreadth(), stale)
}

func tierLabel(t convergencev1.FitnessTier) string {
	switch t {
	case convergencev1.FitnessTier_FITNESS_TIER_STRONG:
		return "strong"
	case convergencev1.FitnessTier_FITNESS_TIER_FAIR:
		return "fair"
	case convergencev1.FitnessTier_FITNESS_TIER_WEAK:
		return "weak"
	default:
		return "?"
	}
}

func eligibilityLabel(e convergencev1.ReferenceEligibility) string {
	switch e {
	case convergencev1.ReferenceEligibility_REFERENCE_ELIGIBILITY_ELIGIBLE:
		return "eligible"
	case convergencev1.ReferenceEligibility_REFERENCE_ELIGIBILITY_CANDIDATE:
		return "candidate"
	case convergencev1.ReferenceEligibility_REFERENCE_ELIGIBILITY_INELIGIBLE:
		return "ineligible"
	default:
		return "?"
	}
}
