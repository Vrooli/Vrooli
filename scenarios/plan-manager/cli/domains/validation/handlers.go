package validation

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/validation/validation_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client validationconnect.ValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: validationconnect.NewValidationServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) references(ctx cliapp.RunContext) error {
	resp, err := h.client.ResolveReferences(context.Background(), connect.NewRequest(&validationv1.ResolveReferencesRequest{
		PlanId: ctx.Positional("plan"), PhaseId: ctx.Flag("phase"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("resolve references", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.References))
	for _, r := range resp.Msg.References {
		results = append(results, formatReference(r))
	}
	summary := fmt.Sprintf("Resolved %d reference(s).", len(resp.Msg.References))
	if resp.Msg.Degraded {
		summary += " (degraded: code-facts unavailable)"
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{summary}, ResultsHeading: "References", Results: results,
	})
}

func (h *handlers) staleness(ctx cliapp.RunContext) error {
	resp, err := h.client.ComputeStaleness(context.Background(), connect.NewRequest(&validationv1.ComputeStalenessRequest{
		PlanId: ctx.Positional("plan"), PhaseId: ctx.Flag("phase"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("compute staleness", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.References))
	for _, r := range resp.Msg.References {
		results = append(results, formatReference(r))
	}
	summary := fmt.Sprintf("Overall staleness: %s.", stalenessLabel(resp.Msg.Overall))
	if resp.Msg.Degraded {
		summary += " (degraded)"
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{summary}, ResultsHeading: "References", Results: results,
	})
}

func (h *handlers) baselineScope(ctx cliapp.RunContext) error {
	resp, err := h.client.DeriveBaselineScope(context.Background(), connect.NewRequest(&validationv1.DeriveBaselineScopeRequest{
		PlanId: ctx.Positional("plan"), PhaseId: ctx.Flag("phase"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("derive baseline scope", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Derived %d command(s) across %d location(s).", len(resp.Msg.Commands), len(resp.Msg.Locations))},
		ResultsHeading: "Baseline commands",
		Results:        resp.Msg.Commands,
		RetrievalHints: []string{"`validate run <plan>` — run this command set and report the verdict"},
	})
}

func (h *handlers) run(ctx cliapp.RunContext) error {
	resp, err := h.client.RunValidation(context.Background(), connect.NewRequest(&validationv1.RunValidationRequest{
		PlanId: ctx.Positional("plan"), PhaseId: ctx.Flag("phase"),
	}))
	if err != nil {
		return wrapRunValidationError(err)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Verdict: %s (staleness %s).", verdictLabel(resp.Msg.Result.GetVerdict()), stalenessLabel(resp.Msg.Result.GetStaleness()))},
		Changes: []string{resp.Msg.Result.GetDetail()},
	})
}

func wrapRunValidationError(err error) error {
	wrapped := cliapp.WrapAPIError("run validation", err, nil)
	if wrapped == nil {
		return nil
	}
	return fmt.Errorf("%w; draft authoring sessions are validated with `plan-manager author validate <session>` before finalize", wrapped)
}

func (h *handlers) verifyDoD(ctx cliapp.RunContext) error {
	resp, err := h.client.VerifyDefinitionOfDone(context.Background(), connect.NewRequest(&validationv1.VerifyDefinitionOfDoneRequest{
		PlanId: ctx.Positional("plan"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("verify definition of done", err, nil)
	}
	met := "NOT met"
	if resp.Msg.DodMet {
		met = "met"
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Definition of Done %s (verdict %s).", met, verdictLabel(resp.Msg.Result.GetVerdict()))},
		Changes: []string{resp.Msg.Result.GetDetail()},
	})
}

func formatReference(r *sharedv1.Reference) string {
	marker := "CODE"
	switch r.Kind {
	case sharedv1.ReferenceKind_REFERENCE_KIND_REQ:
		marker = "REQ"
	case sharedv1.ReferenceKind_REFERENCE_KIND_DOC:
		marker = "DOC"
	}
	line := fmt.Sprintf("[%s: %s] resolution=%s staleness=%s", marker, r.Target, resolutionLabel(r.Resolution), stalenessLabel(r.Staleness))
	if r.Note != "" {
		line += " (" + r.Note + ")"
	}
	return line
}

func verdictLabel(v sharedv1.ValidationVerdict) string {
	switch v {
	case sharedv1.ValidationVerdict_VALIDATION_VERDICT_PASS:
		return "pass"
	case sharedv1.ValidationVerdict_VALIDATION_VERDICT_FAIL:
		return "fail"
	case sharedv1.ValidationVerdict_VALIDATION_VERDICT_UNKNOWN:
		return "unknown"
	default:
		return "unspecified"
	}
}

func stalenessLabel(s sharedv1.StalenessTier) string {
	switch s {
	case sharedv1.StalenessTier_STALENESS_TIER_FRESH:
		return "fresh"
	case sharedv1.StalenessTier_STALENESS_TIER_LIGHTLY_STALE:
		return "lightly_stale"
	case sharedv1.StalenessTier_STALENESS_TIER_DEFINITELY_STALE:
		return "definitely_stale"
	default:
		return "unknown"
	}
}

func resolutionLabel(r sharedv1.ReferenceResolution) string {
	switch r {
	case sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_RESOLVED:
		return "resolved"
	case sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNRESOLVED:
		return "unresolved"
	case sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_FUTURE:
		return "future"
	case sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_MISSING:
		return "missing"
	default:
		return "unspecified"
	}
}
