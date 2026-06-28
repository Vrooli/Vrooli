package generation

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	generationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/generation"
	generationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/generation/generation_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each RunCtx-func has
// typed access to the API client.
type handlers struct {
	core   *cliapp.ScenarioApp
	client generationconnect.GenerationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: generationconnect.NewGenerationServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) status(ctx cliapp.RunContext) error {
	resp, err := h.client.GetProviderStatus(context.Background(), connect.NewRequest(&generationv1.GetProviderStatusRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get provider status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no status response")
	}
	results := make([]string, 0, len(resp.Msg.Providers))
	for _, p := range resp.Msg.Providers {
		results = append(results, fmt.Sprintf("%s — %s", p.Name, availability(p.Available)))
	}
	summary := "No AI providers are currently available."
	if resp.Msg.Available {
		summary = "At least one AI provider is available."
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Providers",
		Results:        results,
		RetrievalHints: []string{
			"`generation elements --brand-id <id> --elements colors,typography,voice` — generate text facets",
			"`generation image --brand-id <id> --type logo` — generate a logo",
		},
	})
}

func (h *handlers) elements(ctx cliapp.RunContext) error {
	resp, err := h.client.GenerateBrandElements(context.Background(), connect.NewRequest(&generationv1.GenerateBrandElementsRequest{
		BrandId:  ctx.Flag("brand-id"),
		Elements: splitElements(ctx.Flag("elements")),
		Model:    ctx.Flag("model"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("generate brand elements", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no generation response")
	}
	changes := make([]string, 0, len(resp.Msg.Results))
	for _, r := range resp.Msg.Results {
		line := fmt.Sprintf("%s — %s", r.Element, r.Status)
		if r.Detail != "" {
			line += fmt.Sprintf(" (%s)", r.Detail)
		}
		changes = append(changes, line)
	}
	provider := resp.Msg.Provider
	if provider == "" {
		provider = "none"
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf(
			"Applied %d of %d element(s) via %s; brand now at v%d.",
			len(resp.Msg.Applied), len(resp.Msg.Results), provider, resp.Msg.BrandVersion,
		)},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("`brands get %s` — see the updated brand", ctx.Flag("brand-id")),
		},
	})
}

func (h *handlers) image(ctx cliapp.RunContext) error {
	resp, err := h.client.GenerateBrandImage(context.Background(), connect.NewRequest(&generationv1.GenerateBrandImageRequest{
		BrandId: ctx.Flag("brand-id"),
		Type:    ctx.Flag("type"),
		Model:   ctx.Flag("model"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("generate brand image", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no image response")
	}
	m := resp.Msg
	return cliapp.RenderProtoMutation(ctx, m, cliapp.MutationReport{
		Result: []string{fmt.Sprintf(
			"Generated %s (%s, %d bytes) via %s and stored it as asset %s.",
			m.Type, m.MimeType, m.Size, m.Provider, m.AssetId,
		)},
		Changes: []string{fmt.Sprintf("%s — %s [brand=%s type=%s]", m.AssetId, m.Filename, m.BrandId, m.Type)},
		NextCommand: []string{
			fmt.Sprintf("`assets download %s --out %s` — fetch the generated image", m.AssetId, m.Filename),
		},
	})
}

// availability renders a provider's boolean readiness as a human label.
func availability(ok bool) string {
	if ok {
		return "available"
	}
	return "unavailable"
}

// splitElements parses a comma-separated --elements value into a trimmed,
// non-empty slice (nil when empty).
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
