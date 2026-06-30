// Package imagetools is brand-manager's production adapter onto the image-tools
// scenario — the concrete implementation of the generation.ImageBackend seam.
//
// It is the only place brand-manager talks to image-tools, and it does so over
// image-tools' public HTTP/Connect surface (never by importing image-tools' Go
// packages or shelling out to its CLI):
//
//   - Model-backed ops (text_to_image / edit_instruct / background_removal) use
//     the REST multipart submit edge (POST /api/v1/ai/{operation}); the op runs
//     as a durable job, so the client blocks ONCE on JobsService.WaitJob and then
//     downloads the result blob (GET /api/v1/blobs/{ref}). image-tools job ids
//     never escape this package.
//   - Deterministic resize uses the synchronous ops edge
//     (POST /api/v1/ops/resize?output=bytes), which streams the result bytes.
//   - Flattening a transparent mark onto a solid background is a same-size alpha
//     composite done with the Go image stdlib after an image-tools resize — the
//     ops `canvas` op pastes rather than alpha-blends, so it cannot flatten.
//   - Readiness uses ModelsService.ExplainResolution per operation.
//
// image-tools' base URL is resolved at call time via api-core service discovery,
// so the two scenarios stay decoupled from each other's ports.
package imagetools

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"brand-manager/internal/generation"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"google.golang.org/protobuf/encoding/protojson"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	jobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs/jobs_v1connect"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
	modelsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models/models_v1connect"
)

// scenarioName is the image-tools scenario id used for service discovery.
const scenarioName = "image-tools"

// image-tools operation names brand-manager drives.
const (
	opTextToImage      = "text_to_image"
	opEditInstruct     = "edit_instruct"
	opBackgroundRemove = "background_removal"
)

// brand operation labels reported by Status (brand-manager vocabulary).
const (
	brandOpGenerate         = "generate"
	brandOpEdit             = "edit"
	brandOpRemoveBackground = "remove_background"
)

// Client implements generation.ImageBackend over image-tools' HTTP surface.
type Client struct {
	httpClient *http.Client
	resolve    func(ctx context.Context) (string, error)
}

var _ generation.ImageBackend = (*Client)(nil)

// NewClient builds the production image-tools client. AI ops can take a while on
// CPU, so the HTTP timeout is generous; the durable job itself is unaffected by
// a client-side timeout once submitted (WaitJob is a fresh request).
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Minute},
		resolve: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, scenarioName)
		},
	}
}

// NewClientWithResolver injects a base-URL resolver + HTTP client (for tests).
func NewClientWithResolver(resolve func(ctx context.Context) (string, error), httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Minute}
	}
	return &Client{httpClient: httpClient, resolve: resolve}
}

// Generate runs text_to_image.
func (c *Client) Generate(ctx context.Context, req generation.ImageGenerateRequest) (generation.ImageOutput, error) {
	params := &aiv1.AIParams{
		Prompt:         req.Prompt,
		NegativePrompt: req.NegativePrompt,
		Width:          int32(req.Width),
		Height:         int32(req.Height),
		Seed:           req.Seed,
		ModelOverride:  req.ModelOverride,
		AllowByok:      req.AllowBYOK,
		QualityPolicy:  req.QualityPolicy,
		FallbackPolicy: req.FallbackPolicy,
		Priority:       req.Priority,
		AllowReclaim:   &req.AllowReclaim,
	}
	return c.submitAndWait(ctx, opTextToImage, nil, params)
}

// Edit runs edit_instruct over the source image.
func (c *Client) Edit(ctx context.Context, req generation.ImageEditRequest) (generation.ImageOutput, error) {
	params := &aiv1.AIParams{
		Prompt:         req.Instruction,
		Seed:           req.Seed,
		ModelOverride:  req.ModelOverride,
		AllowByok:      req.AllowBYOK,
		QualityPolicy:  req.QualityPolicy,
		FallbackPolicy: req.FallbackPolicy,
		Priority:       req.Priority,
		AllowReclaim:   &req.AllowReclaim,
	}
	return c.submitAndWait(ctx, opEditInstruct, req.Source, params)
}

// RemoveBackground runs background_removal over the source image.
func (c *Client) RemoveBackground(ctx context.Context, req generation.ImageRemoveBackgroundRequest) (generation.ImageOutput, error) {
	params := &aiv1.AIParams{
		ModelOverride: req.ModelOverride,
		AllowByok:     req.AllowBYOK,
	}
	return c.submitAndWait(ctx, opBackgroundRemove, req.Source, params)
}

// Resize runs the deterministic resize op to width×height.
func (c *Client) Resize(ctx context.Context, src []byte, width, height int) (generation.ImageOutput, error) {
	data, err := c.runOp(ctx, "resize", src, fmt.Sprintf(`{"resize":{"width":%d,"height":%d}}`, width, height))
	if err != nil {
		return generation.ImageOutput{}, err
	}
	return generation.ImageOutput{Data: data, MimeType: "image/png", Tier: "deterministic"}, nil
}

// Flatten resizes the source to width×height (via image-tools) and composites it
// over a solid background of that size. The same-size alpha composite is done
// with the image stdlib because the ops `canvas` op pastes rather than blends.
func (c *Client) Flatten(ctx context.Context, src []byte, width, height int, background string) (generation.ImageOutput, error) {
	resized, err := c.runOp(ctx, "resize", src, fmt.Sprintf(`{"resize":{"width":%d,"height":%d}}`, width, height))
	if err != nil {
		return generation.ImageOutput{}, err
	}
	flattened, err := flattenOnto(resized, width, height, background)
	if err != nil {
		return generation.ImageOutput{}, generation.ErrImageJobFailed{Operation: "flatten", Detail: err.Error()}
	}
	return generation.ImageOutput{Data: flattened, MimeType: "image/png", Tier: "deterministic"}, nil
}

// Status reports image-tools reachability + per-operation readiness via
// ModelsService.ExplainResolution. It never errors.
func (c *Client) Status(ctx context.Context) generation.ImageBackendStatus {
	baseURL, err := c.resolve(ctx)
	if err != nil {
		return generation.ImageBackendStatus{Available: false, Detail: "image-tools is not reachable: " + err.Error()}
	}
	client := modelsconnect.NewModelsServiceClient(c.httpClient, baseURL)

	ops := []struct{ brand, image string }{
		{brandOpGenerate, opTextToImage},
		{brandOpEdit, opEditInstruct},
		{brandOpRemoveBackground, opBackgroundRemove},
	}
	statuses := make([]generation.ImageOperationStatus, 0, len(ops))
	reachable := false
	for _, op := range ops {
		st := generation.ImageOperationStatus{Operation: op.brand}
		resp, err := client.ExplainResolution(ctx, connect.NewRequest(&modelsv1.ExplainResolutionRequest{
			Operation:     op.image,
			AllowByok:     true,
			QualityPolicy: "quality",
		}))
		if err != nil {
			st.Ready = false
			st.Hint = explainErrorHint(err)
			statuses = append(statuses, st)
			continue
		}
		reachable = true
		res := resp.Msg.GetResolution()
		st.ModelID = res.GetModelId()
		st.Tier = res.GetTier()
		st.Ready = res.GetModelId() != ""
		st.Hint = res.GetCaveat()
		st.Warnings = append([]string(nil), res.GetWarnings()...)
		if st.Hint == "" && len(st.Warnings) > 0 {
			st.Hint = st.Warnings[0]
		}
		statuses = append(statuses, st)
	}
	if !reachable {
		return generation.ImageBackendStatus{Available: false, Detail: "image-tools did not resolve any image operation", Operations: statuses}
	}
	return generation.ImageBackendStatus{Available: true, Operations: statuses}
}

// submitAndWait posts the multipart AI op, blocks once on the durable job, and
// downloads the result blob. file may be nil (text_to_image needs no input).
func (c *Client) submitAndWait(ctx context.Context, operation string, file []byte, params *aiv1.AIParams) (generation.ImageOutput, error) {
	baseURL, err := c.resolve(ctx)
	if err != nil {
		return generation.ImageOutput{}, generation.ErrImageBackendUnavailable{Detail: err.Error()}
	}

	submit, err := c.submitAI(ctx, baseURL, operation, file, params)
	if err != nil {
		return generation.ImageOutput{}, err
	}

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
		return generation.ImageOutput{}, generation.ErrImageJobFailed{Operation: operation, Detail: detail}
	}
	ref := job.GetResultRef()
	if ref == "" {
		return generation.ImageOutput{}, generation.ErrImageJobFailed{Operation: operation, Detail: "job produced no result"}
	}
	data, mime, err := c.downloadBlob(ctx, baseURL, ref)
	if err != nil {
		return generation.ImageOutput{}, generation.ErrImageJobFailed{Operation: operation, Detail: "download result: " + err.Error()}
	}
	return generation.ImageOutput{
		Data:     data,
		MimeType: mime,
		ModelID:  submit.GetModelId(),
		Tier:     submit.GetTier(),
		Warnings: append([]string(nil), submit.GetWarnings()...),
	}, nil
}

// submitAI posts the multipart submit edge and parses SubmitAIResponse, mapping
// image-tools' status codes onto the typed brand-level readiness errors.
func (c *Client) submitAI(ctx context.Context, baseURL, operation string, file []byte, params *aiv1.AIParams) (*aiv1.SubmitAIResponse, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if len(file) > 0 {
		fw, err := mw.CreateFormFile("file", "source.png")
		if err != nil {
			return nil, generation.ErrImageJobFailed{Operation: operation, Detail: err.Error()}
		}
		if _, err := fw.Write(file); err != nil {
			return nil, generation.ErrImageJobFailed{Operation: operation, Detail: err.Error()}
		}
	}
	if params != nil {
		raw, err := protojson.Marshal(params)
		if err != nil {
			return nil, generation.ErrImageJobFailed{Operation: operation, Detail: "marshal params: " + err.Error()}
		}
		if err := mw.WriteField("params", string(raw)); err != nil {
			return nil, generation.ErrImageJobFailed{Operation: operation, Detail: err.Error()}
		}
	}
	if err := mw.Close(); err != nil {
		return nil, generation.ErrImageJobFailed{Operation: operation, Detail: err.Error()}
	}

	url := strings.TrimRight(baseURL, "/") + "/api/v1/ai/" + operation
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return nil, generation.ErrImageJobFailed{Operation: operation, Detail: err.Error()}
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, generation.ErrImageBackendUnavailable{Detail: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	out, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusAccepted {
		return nil, submitStatusError(operation, resp.StatusCode, strings.TrimSpace(string(out)))
	}
	parsed := &aiv1.SubmitAIResponse{}
	if err := protojson.Unmarshal(out, parsed); err != nil {
		return nil, generation.ErrImageJobFailed{Operation: operation, Detail: "decode submit response: " + err.Error()}
	}
	return parsed, nil
}

// submitStatusError maps the submit edge's HTTP status onto a typed error.
// 409 (model not installed) / 422 (not runnable / override invalid) / 503 (no
// backend / no provider) are readiness problems with actionable hints; a refused
// connection is unavailability; anything else is a job failure.
func submitStatusError(operation string, status int, body string) error {
	switch status {
	case http.StatusConflict, http.StatusUnprocessableEntity, http.StatusServiceUnavailable:
		return generation.ErrImageBackendNotReady{Operation: operation, Hint: body}
	case http.StatusTooManyRequests:
		return generation.ErrImageBackendNotReady{Operation: operation, Hint: "image-tools is rate limited; retry shortly"}
	default:
		return generation.ErrImageJobFailed{Operation: operation, Detail: fmt.Sprintf("submit returned %d: %s", status, body)}
	}
}

// runOp posts the synchronous deterministic ops edge and returns the result
// bytes. paramsJSON is the OpParams oneof as protojson (e.g. `{"resize":{...}}`).
func (c *Client) runOp(ctx context.Context, operation string, src []byte, paramsJSON string) ([]byte, error) {
	baseURL, err := c.resolve(ctx)
	if err != nil {
		return nil, generation.ErrImageBackendUnavailable{Detail: err.Error()}
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", "source.png")
	if err != nil {
		return nil, generation.ErrImageJobFailed{Operation: operation, Detail: err.Error()}
	}
	if _, err := fw.Write(src); err != nil {
		return nil, generation.ErrImageJobFailed{Operation: operation, Detail: err.Error()}
	}
	if err := mw.WriteField("params", paramsJSON); err != nil {
		return nil, generation.ErrImageJobFailed{Operation: operation, Detail: err.Error()}
	}
	if err := mw.Close(); err != nil {
		return nil, generation.ErrImageJobFailed{Operation: operation, Detail: err.Error()}
	}

	url := strings.TrimRight(baseURL, "/") + "/api/v1/ops/" + operation + "?output=bytes"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return nil, generation.ErrImageJobFailed{Operation: operation, Detail: err.Error()}
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, generation.ErrImageBackendUnavailable{Detail: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, generation.ErrImageJobFailed{Operation: operation, Detail: fmt.Sprintf("ops %s returned %d: %s", operation, resp.StatusCode, strings.TrimSpace(string(out)))}
	}
	return out, nil
}

// downloadBlob fetches the job's result blob bytes + mime.
func (c *Client) downloadBlob(ctx context.Context, baseURL, ref string) ([]byte, string, error) {
	url := strings.TrimRight(baseURL, "/") + "/api/v1/blobs/" + strings.TrimLeft(ref, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("blob returned %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	mime := resp.Header.Get("Content-Type")
	if mime == "" {
		mime = "image/png"
	}
	return data, mime, nil
}

// flattenOnto composites a (transparent) PNG over a solid-color opaque canvas of
// width×height, returning an opaque PNG. Pure image stdlib — no scaling here, the
// input is already the target size.
func flattenOnto(srcPNG []byte, width, height int, background string) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(srcPNG))
	if err != nil {
		return nil, fmt.Errorf("decode source: %w", err)
	}
	bg := parseHexColor(background)
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
	// Center the source if it is not exactly the canvas size (defensive).
	sb := src.Bounds()
	offset := image.Pt((width-sb.Dx())/2, (height-sb.Dy())/2)
	draw.Draw(canvas, sb.Add(offset), src, sb.Min, draw.Over)

	var out bytes.Buffer
	if err := png.Encode(&out, canvas); err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}
	return out.Bytes(), nil
}

// parseHexColor parses "#rrggbb" into an opaque color, defaulting to white.
func parseHexColor(s string) color.Color {
	s = strings.TrimSpace(s)
	if len(s) == 7 && s[0] == '#' {
		r, rok := hexByte(s[1:3])
		g, gok := hexByte(s[3:5])
		b, bok := hexByte(s[5:7])
		if rok && gok && bok {
			return color.RGBA{R: r, G: g, B: b, A: 0xff}
		}
	}
	return color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
}

func hexByte(s string) (uint8, bool) {
	var v uint8
	for i := 0; i < len(s); i++ {
		c := s[i]
		var d uint8
		switch {
		case c >= '0' && c <= '9':
			d = c - '0'
		case c >= 'a' && c <= 'f':
			d = c - 'a' + 10
		case c >= 'A' && c <= 'F':
			d = c - 'A' + 10
		default:
			return 0, false
		}
		v = v<<4 | d
	}
	return v, true
}

// explainErrorHint turns an ExplainResolution error into a short readiness hint.
func explainErrorHint(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if connect.CodeOf(err) == connect.CodeUnavailable {
		return "image-tools is not reachable"
	}
	return msg
}
