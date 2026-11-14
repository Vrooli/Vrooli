// Package provider exposes Brand Manager's shared ScenarioValidationService
// contract directly for operators and Test Genie troubleshooting.
package provider

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"

	// Register brand-manager's native scan payload so protojson can expand the
	// Any field in ValidateScenarioResponse when `provider validate --json`
	// marshals it. Without this link the type_url is unresolvable client-side
	// and every --json call fails with "google.protobuf.Any: unable to resolve".
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/validation"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "provider"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ScenarioValidationService.ValidateScenario": h.validate,
		"ScenarioValidationService.PreviewFix":       func(ctx cliapp.RunContext) error { return h.fix(ctx, false) },
		"ScenarioValidationService.ApplyFix":         func(ctx cliapp.RunContext) error { return h.fix(ctx, true) },
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("provider: load from manifest: %w", err)
	}
	return group, nil
}

type handlers struct {
	client scenariovalidationconnect.ScenarioValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL)}
}

func (h *handlers) validate(ctx cliapp.RunContext) error {
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: ctx.Positional("scenario"),
		Path:     ctx.Flag("path"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("validate branding provider scenario", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no provider validation response")
	}
	summary, results := presentationReport(resp.Msg.GetAssessment().GetPresentation())
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s: %s", resp.Msg.GetScenario(), resp.Msg.GetStatus()), summary},
		ResultsHeading: "Provider maturity",
		Results:        results,
	})
}

// fix previews (or applies) the provider's deterministic fixers for one
// scenario, optionally restricted to named rules.
func (h *handlers) fix(ctx cliapp.RunContext, apply bool) error {
	scenario := ctx.Positional("scenario")
	req := connect.NewRequest(&scenariovalidationv1.FixRequest{
		Scenario: scenario,
		Path:     ctx.Flag("path"),
		RuleIds:  splitRuleIDs(ctx.FlagValues("rule")),
	})
	var (
		resp *connect.Response[scenariovalidationv1.FixResponse]
		err  error
		verb = "preview branding fixes"
	)
	if apply {
		verb = "apply branding fixes"
		resp, err = h.client.ApplyFix(context.Background(), req)
	} else {
		resp, err = h.client.PreviewFix(context.Background(), req)
	}
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("%s for %q", verb, scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no fix response")
	}
	changes := make([]string, 0, len(resp.Msg.GetCandidates()))
	for _, c := range resp.Msg.GetCandidates() {
		state := "preview"
		if c.GetApplied() {
			state = "applied"
		}
		changes = append(changes, fmt.Sprintf("[%s] %s → %s — %s", state, c.GetRuleId(), c.GetFilePath(), c.GetDescription()))
	}
	result := append([]string{fmt.Sprintf("%d branding fix candidate(s) for %s.", len(resp.Msg.GetCandidates()), resp.Msg.GetScenario())}, resp.Msg.GetMessages()...)
	next := []string{fmt.Sprintf("`brand-manager provider validate %s` - re-check branding findings", scenario)}
	if !apply && len(resp.Msg.GetCandidates()) > 0 {
		next = append([]string{fmt.Sprintf("`brand-manager provider fix-apply %s` - apply these fixes", scenario)}, next...)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: result, Changes: changes, NextCommand: next})
}

// splitRuleIDs accepts repeated and comma-separated --rule values.
func splitRuleIDs(values []string) []string {
	var out []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if part = strings.TrimSpace(part); part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

// presentationReport is intentionally a shallow adapter. It displays the
// provider-owned projection in its supplied capability order and never derives
// a phase story from the assessment when presentation is absent or historical.
func presentationReport(p *commonv1.PhasePresentation) (string, []string) {
	if p == nil {
		return "presentation: unavailable (provider returned no PhasePresentation)", []string{"Raw assessment and provider-native detail remain available."}
	}
	if p.GetContractVersion() != "v1" {
		return fmt.Sprintf("presentation: historical or unsupported contract %q", p.GetContractVersion()), []string{"Raw assessment and provider-native detail remain available; no phase story was synthesized."}
	}
	level := p.GetCurrentLevelLabel()
	if level == "" {
		level = p.GetCurrentLevel()
	}
	if level == "" {
		level = "unknown"
	}
	lines := make([]string, 0, len(p.GetCapabilities())+3)
	for _, capability := range p.GetCapabilities() {
		if capability == nil {
			continue
		}
		label := capability.GetLabel()
		if label == "" {
			label = capability.GetId()
		}
		current := capability.GetCurrentLevelLabel()
		if current == "" {
			current = capability.GetCurrentLevel()
		}
		line := fmt.Sprintf("%s: %s", label, current)
		if next := strings.TrimSpace(capability.GetNextUnlock()); next != "" {
			line += " — next: " + next
		}
		lines = append(lines, line)
	}
	if action := strings.TrimSpace(p.GetNextAction()); action != "" {
		lines = append(lines, "Next action: "+action)
	}
	if topics := p.GetDocumentationTopics(); len(topics) > 0 {
		lines = append(lines, "Documentation: "+strings.Join(topics, ", "))
	}
	if len(lines) == 0 {
		lines = append(lines, "No capability detail was supplied by the provider.")
	}
	return fmt.Sprintf("%s/%s: %s", p.GetProvider(), p.GetPhase(), level), lines
}
