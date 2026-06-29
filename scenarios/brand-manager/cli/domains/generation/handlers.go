package generation

import (
	"context"
	"fmt"
	"strconv"
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
	summary := "No text AI providers are currently available."
	if resp.Msg.Available {
		summary = "At least one text AI provider is available."
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Text providers (colors/typography/voice)",
		Results:        results,
		RetrievalHints: []string{
			"`generation image-status` — check image-tools readiness for logos/icons",
			"`generation elements --brand-id <id> --elements colors,typography,voice` — generate text facets",
		},
	})
}

func (h *handlers) imageStatus(ctx cliapp.RunContext) error {
	resp, err := h.client.GetImageBackendStatus(context.Background(), connect.NewRequest(&generationv1.GetImageBackendStatusRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get image backend status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no image status response")
	}
	results := make([]string, 0, len(resp.Msg.Operations))
	for _, op := range resp.Msg.Operations {
		line := fmt.Sprintf("%s — %s", op.Operation, availabilityReady(op.Ready))
		if op.ModelId != "" {
			line += fmt.Sprintf(" [%s/%s]", op.ModelId, op.Tier)
		}
		if op.Hint != "" {
			line += fmt.Sprintf(" — %s", op.Hint)
		}
		results = append(results, line)
	}
	summary := "image-tools is not reachable."
	if resp.Msg.Available {
		summary = "image-tools is reachable."
	} else if resp.Msg.Detail != "" {
		summary = resp.Msg.Detail
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Image operations",
		Results:        results,
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
		BrandId:        ctx.Flag("brand-id"),
		Type:           ctx.Flag("type"),
		ModelOverride:  ctx.Flag("model"),
		AllowByok:      ctx.BoolFlag("byok"),
		QualityPolicy:  ctx.Flag("quality-policy"),
		FallbackPolicy: ctx.Flag("fallback-policy"),
		Priority:       ctx.Flag("priority"),
		AllowReclaim:   optionalReclaim(ctx),
		Seed:           parseSeed(ctx.Flag("seed")),
		SetCanonical:   ctx.BoolFlag("set-canonical"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("generate brand image", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no image response")
	}
	return h.renderImageAsset(ctx, resp.Msg, "Generated")
}

func (h *handlers) editImage(ctx cliapp.RunContext) error {
	resp, err := h.client.EditBrandImage(context.Background(), connect.NewRequest(&generationv1.EditBrandImageRequest{
		BrandId:        ctx.Flag("brand-id"),
		SourceAssetId:  ctx.Flag("source-asset-id"),
		Instruction:    ctx.Flag("prompt"),
		ModelOverride:  ctx.Flag("model"),
		AllowByok:      ctx.BoolFlag("byok"),
		QualityPolicy:  ctx.Flag("quality-policy"),
		FallbackPolicy: ctx.Flag("fallback-policy"),
		Priority:       ctx.Flag("priority"),
		AllowReclaim:   optionalReclaim(ctx),
		Seed:           parseSeed(ctx.Flag("seed")),
		SetCanonical:   ctx.BoolFlag("set-canonical"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("edit brand image", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no image response")
	}
	return h.renderImageAsset(ctx, resp.Msg, "Edited")
}

func (h *handlers) removeBackground(ctx cliapp.RunContext) error {
	resp, err := h.client.RemoveBrandImageBackground(context.Background(), connect.NewRequest(&generationv1.RemoveBrandImageBackgroundRequest{
		BrandId:       ctx.Flag("brand-id"),
		SourceAssetId: ctx.Flag("source-asset-id"),
		ModelOverride: ctx.Flag("model"),
		AllowByok:     ctx.BoolFlag("byok"),
		SetCanonical:  ctx.BoolFlag("set-canonical"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("remove brand image background", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no image response")
	}
	return h.renderImageAsset(ctx, resp.Msg, "Cut out")
}

func (h *handlers) deriveIcons(ctx cliapp.RunContext) error {
	resp, err := h.client.DeriveBrandIcons(context.Background(), connect.NewRequest(&generationv1.DeriveBrandIconsRequest{
		BrandId:           ctx.Flag("brand-id"),
		SourceAssetId:     ctx.Flag("source-asset-id"),
		IncludeMaskable:   ctx.BoolFlag("maskable"),
		IncludeAppleTouch: ctx.BoolFlag("apple-touch"),
		IncludeFavicon:    ctx.BoolFlag("favicon"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("derive brand icons", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no icons response")
	}
	changes := make([]string, 0, len(resp.Msg.Icons))
	for _, ic := range resp.Msg.Icons {
		changes = append(changes, fmt.Sprintf("%s — %s (%d bytes)", ic.Kind, ic.Filename, ic.Size))
	}
	results := append([]string(nil), changes...)
	for _, w := range resp.Msg.Warnings {
		results = append(results, "warning: "+w)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Derived %d icon variant(s).", len(resp.Msg.Icons))},
		Changes: results,
		NextCommand: []string{
			fmt.Sprintf("`apply preview --brand-id %s --scenario <name>` — preview applying the icon set", ctx.Flag("brand-id")),
		},
	})
}

// renderImageAsset renders a single BrandImageAsset mutation result.
func (h *handlers) renderImageAsset(ctx cliapp.RunContext, m *generationv1.BrandImageAsset, verb string) error {
	model := m.ModelId
	if model == "" {
		model = m.Tier
	}
	canonical := ""
	if m.Canonical {
		canonical = " (canonical)"
	}
	changes := []string{fmt.Sprintf("%s — %s [brand=%s kind=%s]%s", m.AssetId, m.Filename, m.BrandId, m.Kind, canonical)}
	for _, w := range m.Warnings {
		changes = append(changes, "warning: "+w)
	}
	return cliapp.RenderProtoMutation(ctx, m, cliapp.MutationReport{
		Result: []string{fmt.Sprintf(
			"%s %s (%s, %d bytes) via %s and stored it as asset %s.",
			verb, m.Kind, m.MimeType, m.Size, model, m.AssetId,
		)},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("`assets download %s --out %s` — fetch the image", m.AssetId, m.Filename),
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

func availabilityReady(ok bool) string {
	if ok {
		return "ready"
	}
	return "not ready"
}

// parseSeed parses an optional --seed value; non-numeric/empty means 0 (random).
func parseSeed(raw string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	return n
}

func optionalReclaim(ctx cliapp.RunContext) *bool {
	if ctx.FlagDeclared("no-reclaim") && ctx.BoolFlag("no-reclaim") {
		v := false
		return &v
	}
	return nil
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
