package staleness

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	stalenessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/staleness"
	stalenessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/staleness/staleness_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client stalenessconnect.StalenessServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: stalenessconnect.NewStalenessServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListStale(context.Background(), connect.NewRequest(&stalenessv1.ListStaleRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list stale", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no staleness response")
	}
	results := make([]string, 0, len(resp.Msg.Entries))
	for _, e := range resp.Msg.Entries {
		results = append(results, formatEntry(e))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d stale tuple(s).", len(resp.Msg.Entries))},
		ResultsHeading: "Stale tuples",
		Results:        results,
		RetrievalHints: []string{
			"`manifest clear-stale --skill <id> --golden <slug>` — suppress drift after deliberate re-pin",
		},
	})
}

func formatEntry(e *stalenessv1.StaleEntry) string {
	if e == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s/%s — kind=%s template=%s→%s skill=%s→%s",
		e.SkillId, e.GoldenSlug,
		kindLabel(e.Kind),
		e.ManifestTemplateVersionPinned, e.GoldenTemplateVersionCurrent,
		e.ManifestSkillVersionPinned, e.SkillVersionCurrent,
	)
}

func kindLabel(k stalenessv1.StaleKind) string {
	switch k {
	case stalenessv1.StaleKind_STALE_KIND_TEMPLATE_DRIFT:
		return "template_drift"
	case stalenessv1.StaleKind_STALE_KIND_SKILL_DRIFT:
		return "skill_drift"
	case stalenessv1.StaleKind_STALE_KIND_BOTH:
		return "both"
	default:
		return "unspecified"
	}
}
