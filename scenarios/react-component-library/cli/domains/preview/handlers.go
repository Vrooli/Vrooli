package preview

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"
	previewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview/preview_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client previewconnect.PreviewServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: previewconnect.NewPreviewServiceClient(httpClient, baseURL),
	}
}

// bundle calls PreviewService.GetPreviewBundle and renders the JS body
// as the single result line. --json emits the proto wire shape (with
// js, sourcePath, sha256, warnings) for piping.
func (h *handlers) bundle(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetPreviewBundle(context.Background(), connect.NewRequest(&previewv1.GetPreviewBundleRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("bundle component %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no bundle response")
	}
	summary := []string{fmt.Sprintf("Bundled %s (sha256=%s, %d byte(s)).",
		resp.Msg.SourcePath, resp.Msg.Sha256, len(resp.Msg.Js))}
	if len(resp.Msg.Warnings) > 0 {
		summary = append(summary, fmt.Sprintf("%d warning(s) reported.", len(resp.Msg.Warnings)))
	}
	results := []string{resp.Msg.Js}
	if len(resp.Msg.Warnings) > 0 {
		results = append(results, resp.Msg.Warnings...)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Bundle",
		Results:        results,
	})
}
