package apply

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	applyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/apply"
	applyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/apply/apply_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each RunCtx-func has
// typed access to the API client.
type handlers struct {
	core   *cliapp.ScenarioApp
	client applyconnect.ApplyServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: applyconnect.NewApplyServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) preview(ctx cliapp.RunContext) error {
	resp, err := h.client.PreviewApply(context.Background(), connect.NewRequest(&applyv1.PreviewApplyRequest{
		BrandId:      ctx.Flag("brand-id"),
		ScenarioName: ctx.Flag("scenario"),
		Elements:     splitElements(ctx.Flag("elements")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("preview apply", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no preview response")
	}
	m := resp.Msg
	return cliapp.RenderProtoList(ctx, m, cliapp.ListReport{
		Summary: []string{fmt.Sprintf(
			"Preview: applying brand %s (v%d) to %s would write %d file(s), skip %d.",
			m.BrandId, m.BrandVersion, m.Scenario, len(m.Applied), len(m.Skipped),
		)},
		ResultsHeading: "Would write",
		Results:        appliedLines(m.Applied),
		RetrievalHints: append(
			skippedHints(m.Skipped),
			fmt.Sprintf("`apply run --brand-id %s --scenario %s` — write these changes", m.BrandId, m.Scenario),
		),
	})
}

func (h *handlers) run(ctx cliapp.RunContext) error {
	resp, err := h.client.ApplyBrand(context.Background(), connect.NewRequest(&applyv1.ApplyBrandRequest{
		BrandId:      ctx.Flag("brand-id"),
		ScenarioName: ctx.Flag("scenario"),
		Elements:     splitElements(ctx.Flag("elements")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("apply brand", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no apply response")
	}
	m := resp.Msg
	return cliapp.RenderProtoMutation(ctx, m, cliapp.MutationReport{
		Result: []string{fmt.Sprintf(
			"Applied brand %s (v%d) to %s: wrote %d file(s), skipped %d.",
			m.BrandId, m.BrandVersion, m.Scenario, len(m.Applied), len(m.Skipped),
		)},
		Changes: appliedLines(m.Applied),
		NextCommand: []string{
			fmt.Sprintf("`assignments status --scenario %s` — confirm the recorded assignment", m.Scenario),
		},
	})
}

// appliedLines renders each written action as a human line.
func appliedLines(actions []*applyv1.ApplyAction) []string {
	out := make([]string, 0, len(actions))
	for _, a := range actions {
		out = append(out, fmt.Sprintf("%s — %s (%s)", a.File, a.Element, a.Type))
	}
	return out
}

// skippedHints renders each skipped element as a hint line, so the user sees why
// an element produced no change.
func skippedHints(skips []*applyv1.SkipReason) []string {
	out := make([]string, 0, len(skips))
	for _, s := range skips {
		out = append(out, fmt.Sprintf("skipped %s — %s", s.Element, s.Reason))
	}
	return out
}

// splitElements parses a comma-separated --elements value into a trimmed,
// non-empty slice (nil when empty so the server applies all elements).
func splitElements(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
