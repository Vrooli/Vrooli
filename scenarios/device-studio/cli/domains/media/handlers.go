package media

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/vrooli/cli-core/cliapp"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
	projectsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects/projects_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client projectsconnect.ProjectsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	hc, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: projectsconnect.NewProjectsServiceClient(hc, base)}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*projectsv1.ListSourceMediaResponse, error) {
	size := int32(50)
	if raw := ctx.Flag("page-size"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid page-size: %w", err)
		}
		size = int32(n)
	}
	r, err := h.client.ListSourceMedia(context.Background(), connect.NewRequest(&projectsv1.ListSourceMediaRequest{ProjectId: ctx.Positional("project-id"), PageSize: size, PageToken: ctx.Flag("page-token")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list source media", err, nil)
	}
	return r.Msg, nil
}
func (h *handlers) listReport(_ cliapp.OperationContext, r *projectsv1.ListSourceMediaResponse) cliapp.ListReport {
	rows := make([]string, 0, len(r.SourceMedia))
	for _, m := range r.SourceMedia {
		rows = append(rows, format(m))
	}
	hints := []string{}
	if r.NextPageToken != "" {
		hints = append(hints, "`media list <project-id> --page-token "+r.NextPageToken+"` — next page")
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d source image(s).", len(rows))}, ResultsHeading: "Source media", Results: rows, RetrievalHints: hints}
}
func (h *handlers) getCall(ctx cliapp.OperationContext) (*projectsv1.GetSourceMediaResponse, error) {
	r, err := h.client.GetSourceMedia(context.Background(), connect.NewRequest(&projectsv1.GetSourceMediaRequest{ProjectId: ctx.Positional("project-id"), Id: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get source media", err, nil)
	}
	if r.Msg.SourceMedia == nil {
		return nil, fmt.Errorf("server returned no source media")
	}
	return r.Msg, nil
}
func (h *handlers) getReport(_ cliapp.OperationContext, r *projectsv1.GetSourceMediaResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Source image metadata."}, ResultsHeading: "Source media", Results: []string{format(r.SourceMedia)}}
}
func (h *handlers) importCall(ctx cliapp.OperationContext) (*projectsv1.UploadSourceMediaResponse, error) {
	projectID := ctx.Positional("project-id")
	path := ctx.Flag("file")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open source image: %w", err)
	}
	defer f.Close()
	body, err := cliapp.UploadFile(h.core, "/projects/"+url.PathEscape(projectID)+"/media", nil, cliapp.UploadedFile{Name: filepath.Base(path), Reader: f})
	if err != nil {
		return nil, cliapp.WrapAPIError("import source image", err, nil)
	}
	r, err := cliapp.DecodeUploadResponse[*projectsv1.UploadSourceMediaResponse](body)
	if err != nil {
		return nil, err
	}
	if r.SourceMedia == nil {
		return nil, fmt.Errorf("server returned no source media")
	}
	return r, nil
}
func (h *handlers) importReport(_ cliapp.OperationContext, r *projectsv1.UploadSourceMediaResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Imported source image %s.", r.SourceMedia.Id)}, Changes: []string{format(r.SourceMedia)}, NextCommand: []string{fmt.Sprintf("`media get %s %s` — inspect metadata", r.SourceMedia.ProjectId, r.SourceMedia.Id)}}
}
func format(m *projectsv1.SourceMedia) string {
	if m == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — %s, %dx%d, %s, %d bytes, sha256=%s", m.Id, m.OriginalFilename, m.Width, m.Height, m.MediaType, m.SizeBytes, m.Sha256)
}

var _ = os.ErrNotExist
