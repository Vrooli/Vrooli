// Package contract is the CLI's experience-contract command surface.
package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"connectrpc.com/connect"
	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract"
	contractconnect "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract/contract_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "spec"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ContractService.ValidateScenario":     h.validateScenario,
		"ContractService.ListFleet":            h.listFleet,
		"ContractService.AppendAttestation":    h.appendAttestation,
		"ContractService.ScaffoldCases":        h.scaffoldCases,
		"StudioSessionService.ListSpec":        h.listSpec,
		"StudioSessionService.ShowSpec":        h.showSpec,
		"StudioSessionService.ListEvidence":    h.listEvidence,
		"StudioSessionService.SuggestBindings": h.suggestBindings,
		"StudioSessionService.RenderSpec":      h.renderSpec,
		"StudioSessionService.CompareVariants": h.compareVariants,
		"StudioSessionService.PromoteVariant":  h.promoteVariant,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("contract: load from manifest: %w", err)
	}
	return group, nil
}

type handlers struct {
	core         *cliapp.ScenarioApp
	client       contractconnect.ContractServiceClient
	studioClient contractconnect.StudioSessionServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:         core,
		client:       contractconnect.NewContractServiceClient(httpClient, baseURL),
		studioClient: contractconnect.NewStudioSessionServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) listFleet(ctx cliapp.RunContext) error {
	resp, err := h.client.ListFleet(context.Background(), connect.NewRequest(&contractv1.ListFleetRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list experience fleet", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetScenarios()))
	for _, row := range resp.Msg.GetScenarios() {
		results = append(results, fmt.Sprintf("%s %s pages=%d debt=%d status=%s", row.GetScenario(), row.GetMaxDepth(), row.GetPageCount(), row.GetDebtScore(), row.GetStatus()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{fmt.Sprintf("%d scenarios, %d with experience/, %d pages",
			resp.Msg.GetScenarioCount(), resp.Msg.GetWithExperienceCount(), resp.Msg.GetTotalPages())},
		ResultsHeading: "Experience Fleet",
		Results:        results,
	})
}

func (h *handlers) appendAttestation(ctx cliapp.RunContext) error {
	resp, err := h.client.AppendAttestation(context.Background(), connect.NewRequest(&contractv1.AppendAttestationRequest{
		Scenario:  ctx.Positional("scenario"),
		Page:      ctx.Positional("page"),
		Claim:     ctx.Positional("claim"),
		Author:    ctx.Flag("author"),
		Rationale: ctx.Flag("rationale"),
		ExpiresAt: ctx.Flag("expires-at"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("append experience attestation", err, nil)
	}
	a := resp.Msg.GetAttestation()
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s/%s/%s attested until %s", a.GetScenario(), a.GetPage(), a.GetClaim(), a.GetExpiresAt())},
		ResultsHeading: "Attestation",
		Results:        []string{fmt.Sprintf("%s by %s: %s", a.GetId(), a.GetAuthor(), a.GetRationale())},
	})
}

func (h *handlers) scaffoldCases(ctx cliapp.RunContext) error {
	resp, err := h.client.ScaffoldCases(context.Background(), connect.NewRequest(&contractv1.ScaffoldCasesRequest{
		Scenario: ctx.Positional("scenario"),
		Path:     ctx.Flag("path"),
		DryRun:   ctx.BoolFlag("dry-run"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("scaffold experience BAS cases", err, nil)
	}
	verb := "Applied"
	if !resp.Msg.GetApplied() {
		verb = "Previewed"
	}
	results := make([]string, 0, len(resp.Msg.GetDiffs())+len(resp.Msg.GetMessages()))
	for _, diff := range resp.Msg.GetDiffs() {
		results = append(results, fmt.Sprintf("%s %s", diff.GetAction(), diff.GetPath()))
	}
	results = append(results, resp.Msg.GetMessages()...)
	if len(results) == 0 {
		results = append(results, "No BAS case scaffolds needed.")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s %d BAS scaffold change(s) for %s.", verb, len(resp.Msg.GetDiffs()), resp.Msg.GetScenario())},
		ResultsHeading: "Scaffolds",
		Results:        results,
	})
}

func (h *handlers) validateScenario(ctx cliapp.RunContext) error {
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&contractv1.ValidateScenarioRequest{
		Scenario: ctx.Positional("scenario"),
		Path:     ctx.Flag("path"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("validate experience scenario", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validation response")
	}
	msg := resp.Msg
	results := []string{"No experience-contract findings."}
	if msg.Report != nil && len(msg.Report.Findings) > 0 {
		results = results[:0]
		for _, f := range msg.Report.Findings {
			results = append(results, fmt.Sprintf("[%s] %s - %s (%s)", f.Severity, f.Code, f.Message, f.Location))
		}
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s: %s", msg.Scenario, msg.Status)},
		ResultsHeading: "Findings",
		Results:        results,
	})
}

func (h *handlers) listSpec(ctx cliapp.RunContext) error {
	resp, err := h.studioClient.ListSpec(context.Background(), connect.NewRequest(&contractv1.ListSpecRequest{
		Scenario: ctx.Positional("scenario"),
		Path:     ctx.Flag("path"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list experience spec", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetPages())+len(resp.Msg.GetJourneys()))
	for _, page := range resp.Msg.GetPages() {
		results = append(results, fmt.Sprintf("page %s %s %s", page.GetId(), page.GetStatus(), page.GetPath()))
	}
	for _, journey := range resp.Msg.GetJourneys() {
		results = append(results, fmt.Sprintf("journey %s %s %s", journey.GetId(), journey.GetStatus(), journey.GetPath()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s: %d pages, %d journeys", resp.Msg.GetScenario(), len(resp.Msg.GetPages()), len(resp.Msg.GetJourneys()))},
		ResultsHeading: "Documents",
		Results:        results,
	})
}

func (h *handlers) showSpec(ctx cliapp.RunContext) error {
	resp, err := h.studioClient.ShowSpec(context.Background(), connect.NewRequest(&contractv1.ShowSpecRequest{
		Scenario: ctx.Positional("scenario"),
		Page:     ctx.Positional("page"),
		Path:     ctx.Flag("path"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("show experience spec", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{fmt.Sprintf("%s/%s", resp.Msg.GetScenario(), resp.Msg.GetPage())},
		Results: []string{resp.Msg.GetJson()},
	})
}

func (h *handlers) listEvidence(ctx cliapp.RunContext) error {
	resp, err := h.studioClient.ListEvidence(context.Background(), connect.NewRequest(&contractv1.ListEvidenceRequest{
		Scenario: ctx.Positional("scenario"),
		Page:     ctx.Positional("page"),
		Claim:    ctx.Flag("claim"),
		Path:     ctx.Flag("path"),
		Limit:    int32(parseLimit(ctx.Flag("limit"))),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list experience evidence", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetEvidence()))
	for _, ev := range resp.Msg.GetEvidence() {
		results = append(results, fmt.Sprintf("%s %s/%s %s viewport=%s capture=%s checked=%s", ev.GetVerdict(), ev.GetPage(), ev.GetClaim(), ev.GetState(), ev.GetViewport(), ev.GetCaptureRef(), ev.GetCheckedAt()))
	}
	if len(results) == 0 {
		results = append(results, "No reconciliation evidence rows found.")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s/%s: %d evidence rows", resp.Msg.GetScenario(), resp.Msg.GetPage(), len(resp.Msg.GetEvidence()))},
		ResultsHeading: "Evidence",
		Results:        results,
	})
}

func (h *handlers) suggestBindings(ctx cliapp.RunContext) error {
	resp, err := h.studioClient.SuggestBindings(context.Background(), connect.NewRequest(&contractv1.SuggestBindingsRequest{
		Scenario: ctx.Positional("scenario"),
		Page:     ctx.Positional("page"),
		Path:     ctx.Flag("path"),
		Limit:    int32(parseLimit(ctx.Flag("limit"))),
	}))
	if err != nil {
		return cliapp.WrapAPIError("suggest experience bindings", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetSuggestions()))
	for _, s := range resp.Msg.GetSuggestions() {
		results = append(results, fmt.Sprintf("%s testid=%s role=%s name=%s source=%s", s.GetElementId(), s.GetTestid(), s.GetRole(), s.GetAccessibleName(), s.GetSource()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s/%s: %d suggestions", resp.Msg.GetScenario(), resp.Msg.GetPage(), len(resp.Msg.GetSuggestions()))},
		ResultsHeading: "Suggestions",
		Results:        results,
	})
}

func (h *handlers) renderSpec(ctx cliapp.RunContext) error {
	resp, err := h.studioClient.RenderSpec(context.Background(), connect.NewRequest(&contractv1.RenderSpecRequest{
		Scenario: ctx.Positional("scenario"),
		Page:     ctx.Positional("page"),
		Path:     ctx.Flag("path"),
		Mode:     ctx.Flag("mode"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("render experience spec", err, nil)
	}
	results := []string{"artifact: " + resp.Msg.GetArtifactPath()}
	if resp.Msg.GetDegradedReason() != "" {
		results = append(results, "degraded: "+resp.Msg.GetDegradedReason())
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s/%s rendered as %s", resp.Msg.GetScenario(), resp.Msg.GetPage(), resp.Msg.GetMode())},
		ResultsHeading: "Render",
		Results:        results,
	})
}

func (h *handlers) compareVariants(ctx cliapp.RunContext) error {
	variants, err := variantListFromFile(ctx.Flag("file"))
	if err != nil {
		return err
	}
	resp, err := h.studioClient.CompareVariants(context.Background(), connect.NewRequest(&contractv1.CompareVariantsRequest{
		Scenario: ctx.Positional("scenario"),
		Page:     ctx.Positional("page"),
		Path:     ctx.Flag("path"),
		Mode:     ctx.Flag("mode"),
		Variants: variants,
	}))
	if err != nil {
		return cliapp.WrapAPIError("compare experience variants", err, nil)
	}
	results := []string{"artifact: " + resp.Msg.GetArtifactPath()}
	for _, variant := range resp.Msg.GetVariants() {
		results = append(results, fmt.Sprintf("variant %s %s", variant.GetId(), variant.GetTitle()))
	}
	if resp.Msg.GetDegradedReason() != "" {
		results = append(results, "degraded: "+resp.Msg.GetDegradedReason())
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s/%s compared %d variants as %s", resp.Msg.GetScenario(), resp.Msg.GetPage(), len(resp.Msg.GetVariants()), resp.Msg.GetMode())},
		ResultsHeading: "Variants",
		Results:        results,
	})
}

func (h *handlers) promoteVariant(ctx cliapp.RunContext) error {
	variant, err := variantFromFile(ctx.Flag("file"))
	if err != nil {
		return err
	}
	resp, err := h.studioClient.PromoteVariant(context.Background(), connect.NewRequest(&contractv1.PromoteVariantRequest{
		Scenario: ctx.Positional("scenario"),
		Page:     ctx.Positional("page"),
		Path:     ctx.Flag("path"),
		Variant:  variant,
	}))
	if err != nil {
		return cliapp.WrapAPIError("promote experience variant", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetDiffs())+1)
	results = append(results, fmt.Sprintf("variant %s %s", resp.Msg.GetVariant().GetId(), resp.Msg.GetVariant().GetTitle()))
	for _, diff := range resp.Msg.GetDiffs() {
		results = append(results, fmt.Sprintf("%s %s", diff.GetAction(), diff.GetPath()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s/%s promoted: %s", resp.Msg.GetScenario(), resp.Msg.GetPage(), resp.Msg.GetValidation().GetStatus())},
		ResultsHeading: "Changes",
		Results:        results,
	})
}

func parseLimit(value string) int {
	var out int
	_, _ = fmt.Sscanf(value, "%d", &out)
	return out
}

func variantListFromFile(path string) ([]*contractv1.SpecVariant, error) {
	data, err := readRequiredJSON(path, "variant list")
	if err != nil {
		return nil, err
	}
	var variants []*contractv1.SpecVariant
	if err := json.Unmarshal(data, &variants); err == nil {
		return variants, nil
	}
	var wrapped struct {
		Variants []*contractv1.SpecVariant `json:"variants"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return nil, fmt.Errorf("parse variant list %q: %w", path, err)
	}
	return wrapped.Variants, nil
}

func variantFromFile(path string) (*contractv1.SpecVariant, error) {
	data, err := readRequiredJSON(path, "variant")
	if err != nil {
		return nil, err
	}
	var variant contractv1.SpecVariant
	if err := json.Unmarshal(data, &variant); err != nil {
		return nil, fmt.Errorf("parse variant %q: %w", path, err)
	}
	return &variant, nil
}

func readRequiredJSON(path, label string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("%s --file is required", label)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s %q: %w", label, path, err)
	}
	return data, nil
}
