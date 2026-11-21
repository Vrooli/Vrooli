package assets

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"

	assetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assets"
	assetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assets/assets_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each RunCtx-func has
// typed access to the API client.
type handlers struct {
	core   *cliapp.ScenarioApp
	client assetsconnect.AssetsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: assetsconnect.NewAssetsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListAssets(context.Background(), connect.NewRequest(&assetsv1.ListAssetsRequest{
		BrandId: ctx.Flag("brand-id"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list assets", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no assets response")
	}
	results := make([]string, 0, len(resp.Msg.Assets))
	for _, a := range resp.Msg.Assets {
		results = append(results, formatAsset(a))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d asset(s).", len(resp.Msg.Assets))},
		ResultsHeading: "Assets",
		Results:        results,
		RetrievalHints: []string{
			"`assets download <id> --out <path>` — save an asset's bytes",
			"`assets upload --brand-id <id> --file <path>` — upload a new asset",
		},
	})
}

func (h *handlers) upload(ctx cliapp.RunContext) error {
	path := ctx.Flag("file")
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("--file is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %q: %w", path, err)
	}
	filename := ctx.Flag("filename")
	if strings.TrimSpace(filename) == "" {
		filename = filepath.Base(path)
	}

	resp, err := h.client.UploadAsset(context.Background(), connect.NewRequest(&assetsv1.UploadAssetRequest{
		BrandId:  ctx.Flag("brand-id"),
		Filename: filename,
		MimeType: ctx.Flag("mime-type"),
		Content:  data,
	}))
	if err != nil {
		return cliapp.WrapAPIError("upload asset", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Asset == nil {
		return fmt.Errorf("server returned no asset")
	}
	a := resp.Msg.Asset
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Uploaded %s to brand %s (%d bytes).", a.Filename, a.BrandId, a.Size)},
		Changes: []string{formatAsset(a)},
		NextCommand: []string{
			fmt.Sprintf("`assets download %s --out %s` — fetch the bytes back", a.Id, a.Filename),
			fmt.Sprintf("`assets list --brand-id %s` — show this brand's assets", a.BrandId),
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetAsset(context.Background(), connect.NewRequest(&assetsv1.GetAssetRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get asset %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Asset == nil {
		return fmt.Errorf("server returned no asset")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched asset %s.", resp.Msg.Asset.Id)},
		ResultsHeading: "Asset",
		Results:        []string{formatAsset(resp.Msg.Asset)},
	})
}

func (h *handlers) download(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.DownloadAsset(context.Background(), connect.NewRequest(&assetsv1.DownloadAssetRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("download asset %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no download response")
	}
	out := strings.TrimSpace(ctx.Flag("out"))
	if out == "" {
		out = resp.Msg.Filename
	}
	if out == "" {
		return fmt.Errorf("no output path: pass --out (the asset has no stored filename)")
	}
	if err := os.WriteFile(out, resp.Msg.Content, 0o640); err != nil {
		return fmt.Errorf("write %q: %w", out, err)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Saved %d bytes to %s (%s).", len(resp.Msg.Content), out, resp.Msg.MimeType)},
		NextCommand: []string{fmt.Sprintf("`assets get %s` — show the catalog entry", id)},
	})
}

func (h *handlers) delete(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	_, err := h.client.DeleteAsset(context.Background(), connect.NewRequest(&assetsv1.DeleteAssetRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("delete asset %q", id), err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, &assetsv1.DeleteAssetResponse{}, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted asset %s (idempotent).", id)},
		NextCommand: []string{"`assets list` — show remaining assets"},
	})
}

func formatAsset(a *assetsv1.Asset) string {
	if a == nil {
		return "(nil)"
	}
	created := ""
	if a.CreatedAt != nil {
		created = a.CreatedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — %s [brand=%s type=%s size=%dB created=%s]", a.Id, a.Filename, a.BrandId, a.MimeType, a.Size, created)
}
