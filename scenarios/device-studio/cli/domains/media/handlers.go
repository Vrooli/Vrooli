package media

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"connectrpc.com/connect"

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

func (h *handlers) clockCall(ctx cliapp.OperationContext) (*projectsv1.GetSourceMediaClockPageResponse, error) {
	size := int64(100)
	var err error
	if raw := ctx.Flag("page-size"); raw != "" {
		size, err = strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid page-size: %w", err)
		}
	}
	r, err := h.client.GetSourceMediaClockPage(context.Background(), connect.NewRequest(&projectsv1.GetSourceMediaClockPageRequest{ProjectId: ctx.Positional("project-id"), MediaId: ctx.Positional("id"), ExpectedClockSha256: ctx.Flag("expected-clock-sha256"), PageSize: int32(size), PageToken: ctx.Flag("page-token")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get source frame clock", err, nil)
	}
	return r.Msg, nil
}

func (h *handlers) clockReport(_ cliapp.OperationContext, r *projectsv1.GetSourceMediaClockPageResponse) cliapp.ListReport {
	rows := make([]string, 0, len(r.Points))
	for _, point := range r.Points {
		rows = append(rows, fmt.Sprintf("frame=%d pts=%d", point.Ordinal, point.Pts))
	}
	hints := []string{}
	if r.NextPageToken != "" {
		hints = append(hints, "Continue with --page-token "+r.NextPageToken+" and the same expected-clock-sha256.")
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Clock %s; time base %d/%d.", r.ClockSha256, r.TimeBaseNumerator, r.TimeBaseDenominator)}, ResultsHeading: "Source frames", Results: rows, RetrievalHints: hints}
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
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d source media item(s).", len(rows))}, ResultsHeading: "Source media", Results: rows, RetrievalHints: hints}
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
	return cliapp.ListReport{Summary: []string{"Source media metadata and inspection status."}, ResultsHeading: "Source media", Results: []string{format(r.SourceMedia)}}
}

func (h *handlers) importCall(ctx cliapp.OperationContext) (*projectsv1.UploadSourceMediaResponse, error) {
	projectID := ctx.Positional("project-id")
	path := ctx.Flag("file")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open source media: %w", err)
	}
	defer f.Close()
	body, err := cliapp.UploadFile(h.core, "/projects/"+url.PathEscape(projectID)+"/media", nil, cliapp.UploadedFile{Name: filepath.Base(path), Reader: f})
	if err != nil {
		return nil, cliapp.WrapAPIError("import source media", err, nil)
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
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Imported source media %s.", r.SourceMedia.Id)}, Changes: []string{format(r.SourceMedia)}, NextCommand: []string{fmt.Sprintf("`media get %s %s` — inspect metadata and status", r.SourceMedia.ProjectId, r.SourceMedia.Id)}}
}

func format(m *projectsv1.SourceMedia) string {
	if m == nil {
		return "(nil)"
	}
	dimensions := "dimensions unknown"
	if m.Width != nil && m.Height != nil {
		dimensions = fmt.Sprintf("%dx%d", m.GetWidth(), m.GetHeight())
	}
	readiness := strings.ToLower(strings.TrimPrefix(m.Readiness.String(), "MEDIA_READINESS_"))
	out := fmt.Sprintf("%s — %s, %s, %s, %d bytes, readiness=%s, sha256=%s", m.Id, m.OriginalFilename, dimensions, m.MediaType, m.SizeBytes, readiness, m.Sha256)
	if m.InspectionJob != nil {
		out += fmt.Sprintf("; inspection=%s state=%s version=%d attempt=%s", m.InspectionJob.Id, m.InspectionJob.State, m.InspectionJob.Version, m.InspectionJob.CurrentAttemptId)
	}
	if m.InspectionIssueCode != "" {
		out += fmt.Sprintf("; issue=%s: %s", m.InspectionIssueCode, m.InspectionIssueMessage)
	}
	return out
}

func (h *handlers) inspectCall(ctx cliapp.OperationContext) (*projectsv1.EnsureSourceMediaInspectionResponse, error) {
	r, err := h.client.EnsureSourceMediaInspection(context.Background(), connect.NewRequest(&projectsv1.EnsureSourceMediaInspectionRequest{ProjectId: ctx.Positional("project-id"), MediaId: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("ensure local source inspection", err, nil)
	}
	if r.Msg.SourceMedia == nil || r.Msg.Job == nil {
		return nil, fmt.Errorf("server returned incomplete inspection status")
	}
	return r.Msg, nil
}

func inspectionVersion(ctx cliapp.OperationContext) (int64, error) {
	v, err := strconv.ParseInt(ctx.Flag("expected-version"), 10, 64)
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("expected-version must be a positive integer from media get")
	}
	return v, nil
}

func (h *handlers) retryInspectionCall(ctx cliapp.OperationContext) (*projectsv1.RetrySourceMediaInspectionResponse, error) {
	v, err := inspectionVersion(ctx)
	if err != nil {
		return nil, err
	}
	r, err := h.client.RetrySourceMediaInspection(context.Background(), connect.NewRequest(&projectsv1.RetrySourceMediaInspectionRequest{ProjectId: ctx.Positional("project-id"), MediaId: ctx.Positional("id"), ExpectedJobVersion: v}))
	if err != nil {
		return nil, cliapp.WrapAPIError("retry source inspection", err, nil)
	}
	if r.Msg.SourceMedia == nil || r.Msg.Job == nil {
		return nil, fmt.Errorf("server returned incomplete inspection status")
	}
	return r.Msg, nil
}

func (h *handlers) cancelInspectionCall(ctx cliapp.OperationContext) (*projectsv1.CancelSourceMediaInspectionResponse, error) {
	v, err := inspectionVersion(ctx)
	if err != nil {
		return nil, err
	}
	r, err := h.client.CancelSourceMediaInspection(context.Background(), connect.NewRequest(&projectsv1.CancelSourceMediaInspectionRequest{ProjectId: ctx.Positional("project-id"), MediaId: ctx.Positional("id"), ExpectedJobVersion: v}))
	if err != nil {
		return nil, cliapp.WrapAPIError("cancel source inspection", err, nil)
	}
	if r.Msg.SourceMedia == nil || r.Msg.Job == nil {
		return nil, fmt.Errorf("server returned incomplete inspection status")
	}
	return r.Msg, nil
}

func inspectionReport(m *projectsv1.SourceMedia, job *projectsv1.InspectionJobStatus) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Inspection %s is %s at version %d.", job.Id, job.State, job.Version)}, Changes: []string{format(m)}, NextCommand: []string{fmt.Sprintf("`media get %s %s` — read current status", m.ProjectId, m.Id)}}
}

func (h *handlers) inspectReport(_ cliapp.OperationContext, r *projectsv1.EnsureSourceMediaInspectionResponse) cliapp.MutationReport {
	return inspectionReport(r.SourceMedia, r.Job)
}

func (h *handlers) retryInspectionReport(_ cliapp.OperationContext, r *projectsv1.RetrySourceMediaInspectionResponse) cliapp.MutationReport {
	return inspectionReport(r.SourceMedia, r.Job)
}

func (h *handlers) cancelInspectionReport(_ cliapp.OperationContext, r *projectsv1.CancelSourceMediaInspectionResponse) cliapp.MutationReport {
	return inspectionReport(r.SourceMedia, r.Job)
}
