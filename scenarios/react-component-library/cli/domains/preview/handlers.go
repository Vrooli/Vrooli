package preview

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"

	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"
	previewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview/preview_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client previewconnect.PreviewServiceClient
}

func (h *handlers) capture(ctx cliapp.RunContext) error {
	if strings.TrimSpace(ctx.Flag("asset")) == "" && !ctx.BoolFlag("all") {
		return fmt.Errorf("preview capture requires --asset <library-id> or --all")
	}
	root := cliutil.ResolveRepoRoot()
	args := []string{filepath.Join(root, "scenarios", "react-component-library", "tools", "capture-previews.py")}
	if asset := strings.TrimSpace(ctx.Flag("asset")); asset != "" {
		args = append(args, "--asset", asset)
	}
	if version := strings.TrimSpace(ctx.Flag("version")); version != "" {
		args = append(args, "--version", version)
	}
	if ctx.BoolFlag("all") {
		args = append(args, "--all")
	}
	if ctx.BoolFlag("refresh") {
		args = append(args, "--refresh")
	}
	cmd := exec.CommandContext(context.Background(), "python3", args...)
	cmd.Dir = root
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("capture previews: %w\n%s", err, strings.TrimSpace(string(output)))
	}
	var results []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(line) != "" {
			results = append(results, line)
		}
	}
	return ctx.RenderList(cliapp.ListReport{Summary: []string{"Preview capture completed."}, ResultsHeading: "Capture", Results: results})
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
func (h *handlers) bundleCall(ctx cliapp.OperationContext) (*previewv1.GetPreviewBundleResponse, error) {
	id := ctx.Positional("id")
	resp, err := h.client.GetPreviewBundle(context.Background(), connect.NewRequest(&previewv1.GetPreviewBundleRequest{Id: id}))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("bundle component %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no bundle response")
	}
	return resp.Msg, nil
}

func (h *handlers) bundleReport(_ cliapp.OperationContext, msg *previewv1.GetPreviewBundleResponse) cliapp.ListReport {
	summary := []string{fmt.Sprintf("Bundled %s (sha256=%s, %d byte(s)).",
		msg.SourcePath, msg.Sha256, len(msg.Js))}
	if len(msg.Warnings) > 0 {
		summary = append(summary, fmt.Sprintf("%d warning(s) reported.", len(msg.Warnings)))
	}
	results := []string{msg.Js}
	if len(msg.Warnings) > 0 {
		results = append(results, msg.Warnings...)
	}
	return cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Bundle",
		Results:        results,
	}
}
