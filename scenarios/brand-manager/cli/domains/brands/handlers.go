package brands

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	brandsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/brands"
	brandsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/brands/brands_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each RunCtx-func has
// typed access to the API client.
type handlers struct {
	core   *cliapp.ScenarioApp
	client brandsconnect.BrandsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: brandsconnect.NewBrandsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListBrands(context.Background(), connect.NewRequest(&brandsv1.ListBrandsRequest{
		NameContains: ctx.Flag("name"),
		Limit:        atoiOrZero(ctx.Flag("limit")),
		Offset:       atoiOrZero(ctx.Flag("offset")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list brands", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no brands response")
	}
	results := make([]string, 0, len(resp.Msg.Brands))
	for _, b := range resp.Msg.Brands {
		results = append(results, formatBrand(b))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d brand(s).", len(resp.Msg.Brands))},
		ResultsHeading: "Brands",
		Results:        results,
		RetrievalHints: []string{
			"`brands get <id>` — show a single brand",
			"`brands create --name <name>` — create a new brand",
			"`brands versions <brand-id>` — show a brand's version history",
		},
	})
}

func (h *handlers) create(ctx cliapp.RunContext) error {
	resp, err := h.client.CreateBrand(context.Background(), connect.NewRequest(&brandsv1.CreateBrandRequest{
		Name:        ctx.Flag("name"),
		Description: ctx.Flag("description"),
		Notes:       ctx.Flag("notes"),
		Identity:    identityFromFlags(ctx),
		Colors:      colorsFromFlags(ctx),
	}))
	if err != nil {
		return cliapp.WrapAPIError("create brand", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Brand == nil {
		return fmt.Errorf("server returned no brand")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created brand %s.", resp.Msg.Brand.Id)},
		Changes: []string{formatBrand(resp.Msg.Brand)},
		NextCommand: []string{
			fmt.Sprintf("`brands get %s` — show this brand", resp.Msg.Brand.Id),
			"`brands list` — show all brands",
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetBrand(context.Background(), connect.NewRequest(&brandsv1.GetBrandRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get brand %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Brand == nil {
		return fmt.Errorf("server returned no brand")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched brand %s.", resp.Msg.Brand.Id)},
		ResultsHeading: "Brand",
		Results:        []string{formatBrand(resp.Msg.Brand)},
	})
}

func (h *handlers) update(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.UpdateBrand(context.Background(), connect.NewRequest(&brandsv1.UpdateBrandRequest{
		Id:              id,
		Name:            ctx.Flag("name"),
		Description:     ctx.Flag("description"),
		Notes:           ctx.Flag("notes"),
		Identity:        identityFromFlags(ctx),
		Colors:          colorsFromFlags(ctx),
		ExpectedVersion: atoiOrZero(ctx.Flag("expected-version")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("update brand %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Brand == nil {
		return fmt.Errorf("server returned no brand")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated brand %s (now version %d).", resp.Msg.Brand.Id, resp.Msg.Brand.Version)},
		Changes:     []string{formatBrand(resp.Msg.Brand)},
		NextCommand: []string{fmt.Sprintf("`brands versions %s` — show the version history", resp.Msg.Brand.Id)},
	})
}

func (h *handlers) delete(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	_, err := h.client.DeleteBrand(context.Background(), connect.NewRequest(&brandsv1.DeleteBrandRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("delete brand %q", id), err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, &brandsv1.DeleteBrandResponse{}, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted brand %s (idempotent).", id)},
		NextCommand: []string{"`brands list` — show remaining brands"},
	})
}

func (h *handlers) versions(ctx cliapp.RunContext) error {
	brandID := ctx.Positional("brand-id")
	resp, err := h.client.ListBrandVersions(context.Background(), connect.NewRequest(&brandsv1.ListBrandVersionsRequest{BrandId: brandID}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("list versions for brand %q", brandID), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no versions response")
	}
	results := make([]string, 0, len(resp.Msg.Versions))
	for _, v := range resp.Msg.Versions {
		results = append(results, formatVersion(v))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d version(s) for brand %s.", len(resp.Msg.Versions), brandID)},
		ResultsHeading: "Versions",
		Results:        results,
	})
}

func (h *handlers) tokens(ctx cliapp.RunContext) error {
	brandID := ctx.Positional("brand-id")
	resp, err := h.client.GetTokens(context.Background(), connect.NewRequest(&brandsv1.GetTokensRequest{BrandId: brandID}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get tokens for brand %q", brandID), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no tokens response")
	}
	results := make([]string, 0, len(resp.Msg.Tokens))
	for _, token := range resp.Msg.Tokens {
		results = append(results, fmt.Sprintf("%s=%s", token.Name, token.Value))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d token(s) for brand %s.", len(resp.Msg.Tokens), brandID)},
		ResultsHeading: "Design tokens",
		Results:        results,
	})
}

// identityFromFlags builds an Identity message from the display-name/tagline
// flags. Returns nil when neither is set so the partial-update merge leaves the
// facet untouched.
func identityFromFlags(ctx cliapp.RunContext) *brandsv1.Identity {
	displayName := ctx.Flag("display-name")
	tagline := ctx.Flag("tagline")
	if displayName == "" && tagline == "" {
		return nil
	}
	return &brandsv1.Identity{DisplayName: displayName, Tagline: tagline}
}

// colorsFromFlags builds a Colors message from the color flags. Returns nil when
// none are set.
func colorsFromFlags(ctx cliapp.RunContext) *brandsv1.Colors {
	c := &brandsv1.Colors{
		Primary:    ctx.Flag("primary"),
		Secondary:  ctx.Flag("secondary"),
		Accent:     ctx.Flag("accent"),
		Background: ctx.Flag("background"),
		Surface:    ctx.Flag("surface"),
		Text:       ctx.Flag("text"),
	}
	if c.Primary == "" && c.Secondary == "" && c.Accent == "" && c.Background == "" && c.Surface == "" && c.Text == "" {
		return nil
	}
	return c
}

// atoiOrZero parses s as a 32-bit int, returning 0 for empty, unparseable, or
// out-of-range input. ParseInt with bitSize 32 bounds the result so the int32
// conversion can't overflow (gosec G109). The server clamps/validates ranges,
// so the CLI stays a thin pass-through.
func atoiOrZero(s string) int32 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 32)
	if err != nil {
		return 0
	}
	return int32(n)
}

func formatBrand(b *brandsv1.Brand) string {
	if b == nil {
		return "(nil)"
	}
	updated := ""
	if b.UpdatedAt != nil {
		updated = b.UpdatedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — %s [v%d updated=%s]", b.Id, b.Name, b.Version, updated)
}

func formatVersion(v *brandsv1.BrandVersion) string {
	if v == nil {
		return "(nil)"
	}
	created := ""
	if v.CreatedAt != nil {
		created = v.CreatedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("v%d — %s [created=%s]", v.Version, v.Id, created)
}
