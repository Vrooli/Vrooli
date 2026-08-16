package file_preview

import (
	"context"
	"fmt"
	"math"
	"strconv"
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

func (h *handlers) list(ctx cliapp.RunContext) error {
	session := ctx.Flag("session")
	previewID := ctx.Flag("preview-id")
	if session == "" || previewID == "" {
		return fmt.Errorf("--session and --preview-id are required")
	}

	sort, err := parseSortFlag(ctx.Flag("sort"))
	if err != nil {
		return err
	}
	pageSize, err := parsePageSizeFlag(ctx.Flag("page-size"))
	if err != nil {
		return err
	}

	resp, err := h.client.ListDirectory(context.Background(), connect.NewRequest(&filepreviewv1.ListDirectoryRequest{
		SessionId:  session,
		PreviewId:  previewID,
		Sort:       sort,
		ShowHidden: ctx.BoolFlag("show-hidden"),
		PageSize:   pageSize,
		PageToken:  ctx.Flag("page-token"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("file-preview ls", err, nil)
	}

	m := resp.Msg
	rows := make([]string, 0, len(m.GetEntries())+len(m.GetWarnings())+2)
	for _, e := range m.GetEntries() {
		rows = append(rows, formatEntry(e))
	}
	if len(rows) == 0 {
		rows = append(rows, "(empty)")
	}
	for _, w := range m.GetWarnings() {
		rows = append(rows, "warning:  "+w)
	}
	if token := m.GetNextPageToken(); token != "" {
		rows = append(rows, fmt.Sprintf("next:     web-console file-preview ls --session %s --preview-id %s --page-token %s",
			session, previewID, token))
	}

	summary := []string{
		fmt.Sprintf("%s — %d entries, sorted by %s",
			m.GetResolvedPath(), m.GetTotalEntries(), sortLabel(m.GetEffectiveSort())),
	}
	if m.GetTruncated() {
		summary = append(summary, "listing truncated at the server scan ceiling")
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Entries",
		Results:        rows,
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

// formatEntry renders one directory entry as an ls-like row: mode, size,
// name, and the detail that matters for that entry type.
func formatEntry(e *filepreviewv1.DirectoryEntry) string {
	name := e.GetName()
	switch e.GetEntryType() {
	case filepreviewv1.EntryType_ENTRY_TYPE_DIRECTORY:
		name += "/"
	case filepreviewv1.EntryType_ENTRY_TYPE_SYMLINK:
		if target := e.GetSymlinkTarget(); target != "" {
			name += " -> " + target
		}
		if e.GetSymlinkBroken() {
			name += " (broken)"
		}
	}

	size := fmt.Sprintf("%d", e.GetSizeBytes())
	if e.GetEntryType() == filepreviewv1.EntryType_ENTRY_TYPE_DIRECTORY {
		if n := e.GetChildCount(); n >= 0 {
			size = fmt.Sprintf("%d items", n)
		} else {
			size = "-"
		}
	}

	mode := e.GetMode()
	if mode == "" {
		mode = "?"
	}
	return fmt.Sprintf("%-11s %10s  %s", mode, size, name)
}

// parseSortFlag maps the --sort flag onto the proto enum. An empty flag lets
// the server pick its default.
func parseSortFlag(raw string) (filepreviewv1.DirectorySort, error) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "":
		return filepreviewv1.DirectorySort_DIRECTORY_SORT_UNSPECIFIED, nil
	case "dirs_first_name":
		return filepreviewv1.DirectorySort_DIRECTORY_SORT_DIRS_FIRST_NAME, nil
	case "name":
		return filepreviewv1.DirectorySort_DIRECTORY_SORT_NAME, nil
	case "size_desc":
		return filepreviewv1.DirectorySort_DIRECTORY_SORT_SIZE_DESC, nil
	case "mtime_desc":
		return filepreviewv1.DirectorySort_DIRECTORY_SORT_MTIME_DESC, nil
	default:
		return 0, fmt.Errorf("--sort must be one of: dirs_first_name, name, size_desc, mtime_desc")
	}
}

func parsePageSizeFlag(raw string) (int32, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(trimmed)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("--page-size must be a non-negative integer")
	}
	// int is 64-bit on the platforms we ship, so bound the value before
	// narrowing it to the wire's int32. The server clamps to its own maximum
	// anyway; this only keeps the conversion provably lossless.
	if n > math.MaxInt32 {
		n = math.MaxInt32
	}
	return int32(n), nil
}

// kindLabel renders a PreviewKind enum as a lower-case short label
// (e.g. "markdown") for display.
func kindLabel(k filepreviewv1.PreviewKind) string {
	return strings.ToLower(strings.TrimPrefix(k.String(), "PREVIEW_KIND_"))
}

// sortLabel renders a DirectorySort enum as a lower-case short label.
func sortLabel(s filepreviewv1.DirectorySort) string {
	return strings.ToLower(strings.TrimPrefix(s.String(), "DIRECTORY_SORT_"))
}
