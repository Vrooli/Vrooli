package golden

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	goldenv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/golden"
	goldenconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/golden/golden_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client goldenconnect.GoldenServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: goldenconnect.NewGoldenServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListGoldens(context.Background(), connect.NewRequest(&goldenv1.ListGoldensRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list goldens", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no goldens response")
	}
	results := make([]string, 0, len(resp.Msg.Goldens))
	for _, g := range resp.Msg.Goldens {
		results = append(results, formatGolden(g))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d golden(s).", len(resp.Msg.Goldens))},
		ResultsHeading: "Goldens",
		Results:        results,
		RetrievalHints: []string{
			"`goldens get <slug>` — show a single golden",
			"`goldens register --slug <slug> --template <id> --version <v>` — register a generated golden",
			"`goldens regenerate <slug> --yes` — refresh generated-golden metadata from its pinned template",
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	slug := ctx.Positional("slug")
	resp, err := h.client.GetGolden(context.Background(), connect.NewRequest(&goldenv1.GetGoldenRequest{Slug: slug}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get golden %q", slug), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Golden == nil {
		return fmt.Errorf("server returned no golden")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched golden %s.", resp.Msg.Golden.Slug)},
		ResultsHeading: "Golden",
		Results:        []string{formatGolden(resp.Msg.Golden)},
	})
}

func (h *handlers) register(ctx cliapp.RunContext) error {
	resp, err := h.client.RegisterGolden(context.Background(), connect.NewRequest(&goldenv1.RegisterGoldenRequest{
		Slug:            ctx.Flag("slug"),
		TemplateId:      ctx.Flag("template"),
		TemplateVersion: ctx.Flag("version"),
		Path:            ctx.Flag("path"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("register golden", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Golden == nil {
		return fmt.Errorf("server returned no golden")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Registered golden %s.", resp.Msg.Golden.Slug)},
		Changes: []string{formatGolden(resp.Msg.Golden)},
		NextCommand: []string{
			fmt.Sprintf("`goldens get %s` — show this golden", resp.Msg.Golden.Slug),
			"`goldens list` — show all goldens",
		},
	})
}

func (h *handlers) update(ctx cliapp.RunContext) error {
	slug := ctx.Positional("slug")
	resp, err := h.client.UpdateGolden(context.Background(), connect.NewRequest(&goldenv1.UpdateGoldenRequest{
		Slug:            slug,
		Path:            ctx.Flag("path"),
		TemplateVersion: ctx.Flag("version"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("update golden %q", slug), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Golden == nil {
		return fmt.Errorf("server returned no golden")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Updated golden %s.", resp.Msg.Golden.Slug)},
		Changes: []string{formatGolden(resp.Msg.Golden)},
	})
}

func (h *handlers) delete(ctx cliapp.RunContext) error {
	slug := ctx.Positional("slug")
	if !ctx.BoolFlag("yes") {
		return fmt.Errorf("refusing to delete golden %q without --yes confirmation", slug)
	}
	_, err := h.client.DeleteGolden(context.Background(), connect.NewRequest(&goldenv1.DeleteGoldenRequest{Slug: slug}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("delete golden %q", slug), err, nil)
	}
	return ctx.RenderMutation(cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Deleted golden %s.", slug)},
	})
}

func (h *handlers) regenerate(ctx cliapp.RunContext) error {
	slug := ctx.Positional("slug")
	if !ctx.BoolFlag("yes") {
		return fmt.Errorf("refusing to regenerate golden %q without --yes confirmation", slug)
	}
	resp, err := h.client.RegenerateGolden(context.Background(), connect.NewRequest(&goldenv1.RegenerateGoldenRequest{Slug: slug}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("regenerate golden %q", slug), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Golden == nil {
		return fmt.Errorf("server returned no golden")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Regenerated golden %s (template_version=%s).", resp.Msg.Golden.Slug, resp.Msg.Golden.TemplateVersionPinned)},
		Changes: []string{formatGolden(resp.Msg.Golden)},
	})
}

func formatGolden(g *goldenv1.Golden) string {
	if g == nil {
		return "(nil)"
	}
	last := ""
	if g.LastRegeneratedAt != nil {
		last = g.LastRegeneratedAt.AsTime().Format(time.RFC3339)
	}
	logicalRoot := g.LogicalRoot
	if logicalRoot == "" {
		logicalRoot = g.Path
	}
	return fmt.Sprintf("%s — template=%s@%s logical_root=%s last_regenerated=%s",
		g.Slug, g.TemplateId, g.TemplateVersionPinned, logicalRoot, last)
}
