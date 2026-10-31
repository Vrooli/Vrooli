package renders

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image/png"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	shotsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots/shots_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client shotsconnect.RenderJobsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	hc, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: shotsconnect.NewRenderJobsServiceClient(hc, base)}
}

func (h *handlers) submitCall(ctx cliapp.OperationContext) (*shotsv1.SubmitRenderResponse, error) {
	rev, err := strconv.ParseInt(ctx.Flag("revision"), 10, 64)
	if err != nil || rev <= 0 {
		return nil, fmt.Errorf("revision must be a positive integer")
	}
	profile := shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PNG_V1
	mode := shotsv1.RenderMode_RENDER_MODE_STILL
	switch strings.ToUpper(strings.TrimSpace(ctx.Flag("render-mode"))) {
	case "", "STILL", "RENDER_MODE_STILL":
	case "VIDEO_SAMPLE_PNG", "RENDER_MODE_VIDEO_SAMPLE_PNG":
		mode = shotsv1.RenderMode_RENDER_MODE_VIDEO_SAMPLE_PNG
	case "VIDEO_FINAL_MP4", "RENDER_MODE_VIDEO_FINAL_MP4":
		mode, profile = shotsv1.RenderMode_RENDER_MODE_VIDEO_FINAL_MP4, shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_MP4_H264_SDR_V1
	case "PHYSICAL_STILL_PNG", "RENDER_MODE_PHYSICAL_STILL_PNG":
		mode = shotsv1.RenderMode_RENDER_MODE_PHYSICAL_STILL_PNG
	case "PHYSICAL_FINAL_MP4", "PHYSICAL_VIDEO_FINAL_MP4", "RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4":
		mode, profile = shotsv1.RenderMode_RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4, shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_MP4_H264_SDR_V1
	case "PHYSICAL_PREVIEW_PNG", "RENDER_MODE_PHYSICAL_PREVIEW_PNG":
		mode, profile = shotsv1.RenderMode_RENDER_MODE_PHYSICAL_PREVIEW_PNG, shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PHYSICAL_PREVIEW_PNG_V1
	case "PHYSICAL_DIAGNOSTIC_PNG", "RENDER_MODE_PHYSICAL_DIAGNOSTIC_PNG":
		mode, profile = shotsv1.RenderMode_RENDER_MODE_PHYSICAL_PREVIEW_PNG, shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PHYSICAL_DIAGNOSTIC_PNG_V1
	case "PHYSICAL_DIAGNOSTIC_MOTION_MP4", "RENDER_MODE_PHYSICAL_DIAGNOSTIC_MOTION_MP4":
		mode, profile = shotsv1.RenderMode_RENDER_MODE_PHYSICAL_DIAGNOSTIC_MOTION_MP4, shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PHYSICAL_DIAGNOSTIC_MOTION_MP4_V1
	default:
		return nil, fmt.Errorf("render-mode must be STILL, VIDEO_SAMPLE_PNG, VIDEO_FINAL_MP4, PHYSICAL_STILL_PNG, PHYSICAL_FINAL_MP4, PHYSICAL_PREVIEW_PNG, PHYSICAL_DIAGNOSTIC_PNG or PHYSICAL_DIAGNOSTIC_MOTION_MP4")
	}
	r, err := h.client.SubmitRender(context.Background(), connect.NewRequest(&shotsv1.SubmitRenderRequest{ProjectId: ctx.Flag("project-id"), ShotId: ctx.Flag("shot-id"), Revision: rev, IdempotencyKey: ctx.Flag("idempotency-key"), OutputProfile: profile, RenderMode: mode}))
	if err != nil {
		return nil, cliapp.WrapAPIError("submit render", err, nil)
	}
	if r.Msg.Job == nil {
		return nil, fmt.Errorf("server returned no render job")
	}
	return r.Msg, nil
}

func (h *handlers) submitReport(_ cliapp.OperationContext, r *shotsv1.SubmitRenderResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Render %s is %s at version %d.", r.Job.Id, r.Job.Status, r.Job.Version)}, Changes: []string{formatJob(r.Job)}, NextCommand: []string{fmt.Sprintf("`renders get %s --project-id %s` — inspect progress", r.Job.Id, r.Job.ProjectId)}}
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*shotsv1.GetRenderResponse, error) {
	r, err := h.client.GetRender(context.Background(), connect.NewRequest(&shotsv1.GetRenderRequest{ProjectId: ctx.Flag("project-id"), RenderId: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get render", err, nil)
	}
	if r.Msg.Job == nil {
		return nil, fmt.Errorf("server returned no render job")
	}
	return r.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, r *shotsv1.GetRenderResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Render job status."}, ResultsHeading: "Render", Results: []string{formatJob(r.Job)}}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*shotsv1.ListRendersResponse, error) {
	size := int32(50)
	if raw := strings.TrimSpace(ctx.Flag("page-size")); raw != "" {
		n, e := strconv.ParseInt(raw, 10, 32)
		if e != nil {
			return nil, fmt.Errorf("invalid page-size: %w", e)
		}
		size = int32(n)
	}
	r, err := h.client.ListRenders(context.Background(), connect.NewRequest(&shotsv1.ListRendersRequest{ProjectId: ctx.Flag("project-id"), PageSize: size, PageToken: ctx.Flag("page-token")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list renders", err, nil)
	}
	return r.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, r *shotsv1.ListRendersResponse) cliapp.ListReport {
	rows := make([]string, 0, len(r.Jobs))
	for _, j := range r.Jobs {
		rows = append(rows, formatJob(j))
	}
	hints := []string{}
	if r.NextPageToken != "" {
		hints = append(hints, "`renders list --page-token "+r.NextPageToken+"` — next page")
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d render job(s).", len(rows))}, ResultsHeading: "Renders", Results: rows, RetrievalHints: hints}
}

func (h *handlers) cancelCall(ctx cliapp.OperationContext) (*shotsv1.CancelRenderResponse, error) {
	version, err := strconv.ParseInt(ctx.Flag("expected-version"), 10, 64)
	if err != nil || version <= 0 {
		return nil, fmt.Errorf("expected-version must be a positive integer")
	}
	r, err := h.client.CancelRender(context.Background(), connect.NewRequest(&shotsv1.CancelRenderRequest{ProjectId: ctx.Flag("project-id"), RenderId: ctx.Positional("id"), ExpectedVersion: version}))
	if err != nil {
		return nil, cliapp.WrapAPIError("cancel render", err, nil)
	}
	if r.Msg.Job == nil {
		return nil, fmt.Errorf("server returned no render job")
	}
	return r.Msg, nil
}

func (h *handlers) cancelReport(_ cliapp.OperationContext, r *shotsv1.CancelRenderResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Render %s is now %s (version %d).", r.Job.Id, r.Job.Status, r.Job.Version)}, Changes: []string{formatJob(r.Job)}}
}

func (h *handlers) exportCall(ctx cliapp.OperationContext) (*shotsv1.GetRenderResponse, error) {
	projectID, id, path := ctx.Flag("project-id"), ctx.Positional("id"), strings.TrimSpace(ctx.Flag("out"))
	if projectID == "" || id == "" || path == "" {
		return nil, fmt.Errorf("project-id, render id, and out path are required")
	}
	metadata, err := h.client.GetRender(context.Background(), connect.NewRequest(&shotsv1.GetRenderRequest{ProjectId: projectID, RenderId: id}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get render metadata", err, nil)
	}
	job := metadata.Msg.Job
	if job == nil || job.Status != shotsv1.RenderJobStatus_RENDER_JOB_STATUS_SUCCEEDED || job.Output == nil {
		return nil, fmt.Errorf("render has no verified output yet")
	}
	if job.Id != id || job.ProjectId != projectID {
		return nil, fmt.Errorf("render metadata belongs to a different project or job")
	}
	if err = h.downloadOutput(context.Background(), projectID, id, path, job.Output); err != nil {
		return nil, err
	}
	return metadata.Msg, nil
}

func (h *handlers) exportReport(ctx cliapp.OperationContext, r *shotsv1.GetRenderResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Verified %s exported to %s.", r.Job.Output.MediaType, ctx.Flag("out"))}, Changes: []string{formatJob(r.Job)}}
}

// downloadOutput retains only a bounded stream and publishes a verified file
// without replacing any existing destination, including one created mid-download.
func (h *handlers) downloadOutput(ctx context.Context, projectID, id, path string, output *shotsv1.RenderOutput) error {
	if output == nil || output.SizeBytes <= 0 || output.SizeBytes > 160<<20 || output.WidthPixels == 0 || output.HeightPixels == 0 || output.WidthPixels > 16384 || output.HeightPixels > 16384 || uint64(output.WidthPixels)*uint64(output.HeightPixels) > 16_000_000 {
		return fmt.Errorf("render output metadata exceeds supported export bounds")
	}
	expectedHash, err := hex.DecodeString(output.Sha256)
	if err != nil || len(expectedHash) != sha256.Size || output.Sha256 != strings.ToLower(output.Sha256) || (output.MediaType != "image/png" && output.MediaType != "video/mp4") {
		return fmt.Errorf("render output digest or media type is invalid")
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		if err == nil {
			return fmt.Errorf("output path already exists: %s", path)
		}
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	client, _ := cliapp.NewConnectHTTPClientWithTimeout(h.core, 2*time.Minute)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/projects/"+url.PathEscape(projectID)+"/renders/"+url.PathEscape(id)+"/content", nil)
	if err != nil {
		return err
	}
	response, err := client.Do(req)
	if err != nil {
		return cliapp.WrapAPIError("download render output", err, nil)
	}
	defer response.Body.Close()
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if response.StatusCode != http.StatusOK || err != nil || mediaType != output.MediaType || (response.ContentLength >= 0 && response.ContentLength != output.SizeBytes) {
		return fmt.Errorf("render download status, type or length differs from verified metadata (HTTP %d)", response.StatusCode)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".device-studio-export-*")
	if err != nil {
		return fmt.Errorf("create export temporary file: %w", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()
	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, output.SizeBytes+1))
	if err != nil {
		return fmt.Errorf("download render output: %w", err)
	}
	if n != output.SizeBytes || hex.EncodeToString(hash.Sum(nil)) != output.Sha256 {
		return fmt.Errorf("downloaded output size or digest differs from verified metadata")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if output.MediaType == "image/png" {
		cfg, err := png.DecodeConfig(file)
		if err != nil || cfg.Width != int(output.WidthPixels) || cfg.Height != int(output.HeightPixels) {
			return fmt.Errorf("downloaded PNG dimensions differ from verified metadata")
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if _, err := png.Decode(file); err != nil {
			return fmt.Errorf("downloaded output is not a complete PNG: %w", err)
		}
	} else {
		var prefix [8]byte
		if _, err := io.ReadFull(file, prefix[:]); err != nil || string(prefix[4:]) != "ftyp" {
			return fmt.Errorf("downloaded output is not an MP4 container")
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	// A same-directory hard link gives atomic no-replace publication on the
	// supported desktop filesystems. Unsupported filesystems fail explicitly.
	if err := os.Link(file.Name(), path); err != nil {
		return fmt.Errorf("publish verified export without overwriting: %w", err)
	}
	return nil
}

func (h *handlers) sampleBindingCall(ctx cliapp.OperationContext) (*shotsv1.GetVideoSampleBindingResponse, error) {
	r, err := h.client.GetVideoSampleBinding(context.Background(), connect.NewRequest(&shotsv1.GetVideoSampleBindingRequest{ProjectId: ctx.Flag("project-id"), ShotId: ctx.Flag("shot-id"), RenderId: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get video sample binding", err, nil)
	}
	if r.Msg.Binding == nil {
		return nil, fmt.Errorf("server returned no video sample binding")
	}
	return r.Msg, nil
}

func (h *handlers) sampleBindingReport(_ cliapp.OperationContext, r *shotsv1.GetVideoSampleBindingResponse) cliapp.ListReport {
	b := r.Binding
	return cliapp.ListReport{Summary: []string{"Verified sample values for an explicit new shot revision."}, ResultsHeading: "Sample", Results: []string{fmt.Sprintf("job=%s base_revision=%d source_frame=%d source_pts=%d pixels=%s", b.JobId, b.BaseShotRevision, b.SourceOrdinal, b.SourcePts, b.ComposedFrameSha256)}, RetrievalHints: []string{"Use --json to retrieve the complete binding for video_timing.sample_binding in a new shot snapshot."}}
}

func formatJob(j *shotsv1.RenderJob) string {
	if j == nil {
		return "(nil)"
	}
	s := fmt.Sprintf("%s project=%s shot=%s revision=%d status=%s version=%d", j.Id, j.ProjectId, j.ShotId, j.ShotRevision, j.Status, j.Version)
	if j.Issue != nil {
		s += " issue=" + j.Issue.Summary
	}
	if j.Output != nil {
		s += fmt.Sprintf(" output=%s %dx%d %d bytes sha256=%s", j.Output.MediaType, j.Output.WidthPixels, j.Output.HeightPixels, j.Output.SizeBytes, j.Output.Sha256)
	}
	return s
}
