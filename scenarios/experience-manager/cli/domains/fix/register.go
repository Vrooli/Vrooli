// Package fix is the CLI's deterministic-remediation command surface.
package fix

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "fix"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ScenarioValidationService.PreviewFix": h.preview,
		"ScenarioValidationService.ApplyFix":   h.apply,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("fix: load from manifest: %w", err)
	}
	return group, nil
}

type handlers struct {
	core   *cliapp.ScenarioApp
	client scenariovalidationconnect.ScenarioValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL),
	}
}

func ruleIDs(ctx cliapp.RunContext) []string {
	raw := strings.TrimSpace(ctx.Flag("rules"))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func (h *handlers) preview(ctx cliapp.RunContext) error {
	resp, err := h.client.PreviewFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{
		Scenario: ctx.Positional("scenario"),
		RuleIds:  ruleIDs(ctx),
	}))
	if err != nil {
		return cliapp.WrapAPIError("preview fixes", err, nil)
	}
	return renderFix(ctx, resp.Msg, false)
}

func (h *handlers) apply(ctx cliapp.RunContext) error {
	resp, err := h.client.ApplyFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{
		Scenario: ctx.Positional("scenario"),
		RuleIds:  ruleIDs(ctx),
	}))
	if err != nil {
		return cliapp.WrapAPIError("apply fixes", err, nil)
	}
	return renderFix(ctx, resp.Msg, true)
}

func renderFix(ctx cliapp.RunContext, msg *scenariovalidationv1.FixResponse, applied bool) error {
	if msg == nil {
		return fmt.Errorf("server returned no fix response")
	}
	verb := "Previewed"
	if applied {
		verb = "Applied"
	}
	results := make([]string, 0, len(msg.Candidates))
	for _, c := range msg.Candidates {
		results = append(results, fmt.Sprintf("%s -> %s: %s", c.RuleId, c.FilePath, c.Description))
	}
	results = append(results, msg.Messages...)
	if len(results) == 0 {
		results = append(results, "No fix candidates.")
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s %d fix candidate(s) for %s.", verb, len(msg.Candidates), msg.Scenario)},
		ResultsHeading: "Candidates",
		Results:        results,
	})
}
