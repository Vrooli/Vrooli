package file_preview

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	filepreviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/file_preview"
	filepreviewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/file_preview/file_preview_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client filepreviewconnect.FilePreviewServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: filepreviewconnect.NewFilePreviewServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) resolve(ctx cliapp.RunContext) error {
	session := ctx.Flag("session")
	path := ctx.Flag("path")
	if session == "" || path == "" {
		return fmt.Errorf("--session and --path are required")
	}

	resp, err := h.client.Resolve(context.Background(), connect.NewRequest(&filepreviewv1.ResolveRequest{
		SessionId:     session,
		Path:          path,
		SourceContext: filepreviewv1.SourceContext_SOURCE_CONTEXT_CLI,
	}))
	if err != nil {
		return cliapp.WrapAPIError("file-preview resolve", err, nil)
	}

	m := resp.Msg
	rows := []string{
		fmt.Sprintf("input:      %s", m.GetInputPath()),
		fmt.Sprintf("resolved:   %s", m.GetResolvedPath()),
		fmt.Sprintf("kind:       %s", kindLabel(m.GetPreviewKind())),
		fmt.Sprintf("mime:       %s", m.GetMimeType()),
		fmt.Sprintf("size:       %d bytes", m.GetSizeBytes()),
		fmt.Sprintf("basis:      %s", m.GetResolutionBasis()),
		fmt.Sprintf("preview:    %t | download=%t | range=%t | text=%t",
			m.GetCanPreview(), m.GetCanDownload(), m.GetSupportsRange(), m.GetTextContentAvailable()),
		fmt.Sprintf("preview_id: %s", m.GetPreviewId()),
		fmt.Sprintf("blob_url:   %s", m.GetBlobUrl()),
	}
	if m.GetHasLine() {
		rows = append(rows, fmt.Sprintf("line:       %d", m.GetLine()))
	}
	for _, w := range m.GetWarnings() {
		rows = append(rows, "warning:    "+w)
	}
	if m.GetTextContentAvailable() {
		rows = append(rows, fmt.Sprintf("next:       web-console file-preview text --session %s --preview-id %s", session, m.GetPreviewId()))
	}
	report := cliapp.ListReport{
		Summary:        []string{"Preview target"},
		ResultsHeading: "Resolution",
		Results:        rows,
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

func (h *handlers) text(ctx cliapp.RunContext) error {
	session := ctx.Flag("session")
	previewID := ctx.Flag("preview-id")
	if session == "" || previewID == "" {
		return fmt.Errorf("--session and --preview-id are required")
	}

	resp, err := h.client.GetTextContent(context.Background(), connect.NewRequest(&filepreviewv1.GetTextContentRequest{
		SessionId: session,
		PreviewId: previewID,
	}))
	if err != nil {
		return cliapp.WrapAPIError("file-preview text", err, nil)
	}

	m := resp.Msg
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("%s (%s)", m.GetResolvedPath(), kindLabel(m.GetPreviewKind()))},
			ResultsHeading: "Content",
			Results:        []string{m.GetContent()},
		})
	}
	fmt.Fprintf(ctx.Stdout(), "%s (%s, %s)\n", m.GetResolvedPath(), kindLabel(m.GetPreviewKind()), m.GetMimeType())
	fmt.Fprintln(ctx.Stdout(), m.GetContent())
	return nil
}

// kindLabel renders a PreviewKind enum as a lower-case short label
// (e.g. "markdown") for display.
func kindLabel(k filepreviewv1.PreviewKind) string {
	return strings.ToLower(strings.TrimPrefix(k.String(), "PREVIEW_KIND_"))
}
