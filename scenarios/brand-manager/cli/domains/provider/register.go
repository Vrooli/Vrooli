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

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "provider"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ScenarioValidationService.ValidateScenario": h.validate,
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
