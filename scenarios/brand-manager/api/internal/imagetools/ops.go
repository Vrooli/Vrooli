package imagetools

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"brand-manager/internal/generation"

	"connectrpc.com/connect"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	jobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs/jobs_v1connect"
	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
	"google.golang.org/protobuf/encoding/protojson"
)

// VectorizeOptions are the image-tools vectorize parameters brand-manager exposes.
type VectorizeOptions struct {
	Colors                     int
	KeepColors                 []string
	DropBackgroundLayers       bool
	ClipToLargestRoundedRegion bool
	InsetPx                    float64
	TolerancePx                float64
	Smoothing                  bool
	MinAreaPx                  float64
}

// Vectorize runs the deterministic vectorize op over a raster and returns SVG.
func (c *Client) Vectorize(ctx context.Context, src []byte, opts VectorizeOptions) ([]byte, error) {
	msg := &opsv1.OpParams{Op: &opsv1.OpParams_Vectorize{Vectorize: &opsv1.VectorizeParams{
		Colors:                     int32(opts.Colors),
		KeepColors:                 opts.KeepColors,
		DropBackgroundLayers:       opts.DropBackgroundLayers,
		ClipToLargestRoundedRegion: opts.ClipToLargestRoundedRegion,
		InsetPx:                    opts.InsetPx,
		TolerancePx:                opts.TolerancePx,
		Smoothing:                  opts.Smoothing,
		MinAreaPx:                  opts.MinAreaPx,
	}}}
	return c.runOpJSON(ctx, "vectorize", src, msg)
}

// Rasterize renders an SVG at an exact size, optionally onto a background.
func (c *Client) Rasterize(ctx context.Context, svg []byte, width, height int, background string) ([]byte, error) {
	msg := &opsv1.OpParams{Op: &opsv1.OpParams_Rasterize{Rasterize: &opsv1.RasterizeParams{
		Width: int32(width), Height: int32(height), Background: background,
	}}}
	return c.runOpJSON(ctx, "rasterize", svg, msg)
}

// IconContainer packs PNG renders into an ICO or ICNS.
func (c *Client) IconContainer(ctx context.Context, src []byte, format string, sizes []int) ([]byte, error) {
	pbSizes := make([]int32, 0, len(sizes))
	for _, s := range sizes {
		pbSizes = append(pbSizes, int32(s))
	}
	msg := &opsv1.OpParams{Op: &opsv1.OpParams_IconContainer{IconContainer: &opsv1.IconContainerParams{
		Format: format, Sizes: pbSizes,
	}}}
	return c.runOpJSON(ctx, "icon_container", src, msg)
}

// runOpJSON marshals a typed OpParams and runs the sync op.
func (c *Client) runOpJSON(ctx context.Context, operation string, src []byte, msg *opsv1.OpParams) ([]byte, error) {
	raw, err := protojson.Marshal(msg)
	if err != nil {
		return nil, generation.ErrImageJobFailed{Operation: operation, Detail: "marshal params: " + err.Error()}
	}
	return c.runOp(ctx, operation, src, string(raw))
}

// RemoveObject runs object_removal with a mask and returns the result. The
// grow/halo knobs are the mask_fill provider defaults; image-tools reads them
// from its own op parameters, so brand-manager does not duplicate them.
func (c *Client) RemoveObject(ctx context.Context, src, mask []byte) (generation.ImageOutput, error) {
	baseURL, err := c.resolve(ctx)
	if err != nil {
		return generation.ImageOutput{}, generation.ErrImageBackendUnavailable{Detail: err.Error()}
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", "source.png")
	if err != nil {
		return generation.ImageOutput{}, generation.ErrImageJobFailed{Operation: "object_removal", Detail: err.Error()}
	}
	if _, err := fw.Write(src); err != nil {
		return generation.ImageOutput{}, generation.ErrImageJobFailed{Operation: "object_removal", Detail: err.Error()}
	}
	maskWriter, err := mw.CreateFormFile("mask", "mask.png")
	if err != nil {
		return generation.ImageOutput{}, generation.ErrImageJobFailed{Operation: "object_removal", Detail: err.Error()}
	}
	if _, err := maskWriter.Write(mask); err != nil {
		return generation.ImageOutput{}, generation.ErrImageJobFailed{Operation: "object_removal", Detail: err.Error()}
	}
	if err := mw.Close(); err != nil {
		return generation.ImageOutput{}, generation.ErrImageJobFailed{Operation: "object_removal", Detail: err.Error()}
	}

	url := strings.TrimRight(baseURL, "/") + "/api/v1/ai/object_removal"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return generation.ImageOutput{}, generation.ErrImageJobFailed{Operation: "object_removal", Detail: err.Error()}
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return generation.ImageOutput{}, generation.ErrImageBackendUnavailable{Detail: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusAccepted {
		return generation.ImageOutput{}, submitStatusError("object_removal", resp.StatusCode, strings.TrimSpace(string(out)))
	}
	submit := &aiv1.SubmitAIResponse{}
	if err := protojson.Unmarshal(out, submit); err != nil {
		return generation.ImageOutput{}, generation.ErrImageJobFailed{Operation: "object_removal", Detail: "decode submit response: " + err.Error()}
	}
	return c.waitAndDownload(ctx, baseURL, submit)
}

// waitAndDownload blocks once on a submitted AI job and downloads its result.
func (c *Client) waitAndDownload(ctx context.Context, baseURL string, submit *aiv1.SubmitAIResponse) (generation.ImageOutput, error) {
	jobsClient := jobsconnect.NewJobsServiceClient(c.httpClient, baseURL)
	waitResp, err := jobsClient.WaitJob(ctx, connect.NewRequest(&jobsv1.WaitJobRequest{Id: submit.GetJobId()}))
	if err != nil {
		return generation.ImageOutput{}, generation.ErrImageBackendUnavailable{Detail: "wait job: " + err.Error()}
	}
	job := waitResp.Msg.GetJob()
	if job.GetState() != jobsv1.JobState_JOB_STATE_SUCCEEDED {
		detail := job.GetError()
		if detail == "" {
			detail = "job ended in state " + job.GetState().String()
		}
		return generation.ImageOutput{}, generation.ErrImageJobFailed{Operation: job.GetOperation(), Detail: detail}
	}
	ref := job.GetResultRef()
	if ref == "" {
		return generation.ImageOutput{}, generation.ErrImageJobFailed{Operation: job.GetOperation(), Detail: "job produced no result"}
	}
	data, mime, err := c.downloadBlob(ctx, baseURL, ref)
	if err != nil {
		return generation.ImageOutput{}, generation.ErrImageJobFailed{Operation: job.GetOperation(), Detail: "download result: " + err.Error()}
	}
	return generation.ImageOutput{Data: data, MimeType: mime, ModelID: submit.GetModelId(), Tier: submit.GetTier(), Warnings: append([]string(nil), submit.GetWarnings()...)}, nil
}
