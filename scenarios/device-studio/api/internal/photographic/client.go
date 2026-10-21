package photographic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/apihttp"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	jobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs/jobs_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const MaxPNGBytes = 12 << 20

type ResolveOwner func(context.Context, string) (string, error)

type Operation struct {
	JobID      string            `json:"job_id"`
	Operation  string            `json:"operation"`
	ModelID    string            `json:"model_id,omitempty"`
	Tier       string            `json:"tier,omitempty"`
	State      string            `json:"state"`
	Progress   int               `json:"progress"`
	Message    string            `json:"message,omitempty"`
	Error      string            `json:"error,omitempty"`
	ResultRef  string            `json:"result_ref,omitempty"`
	ResultMeta map[string]string `json:"result_meta,omitempty"`
}

type Client struct {
	resolve ResolveOwner
	http    *http.Client
}

type OwnerHTTPError struct{ Status int }

func (e OwnerHTTPError) Error() string {
	return fmt.Sprintf("Image Tools owner returned HTTP %d", e.Status)
}

func NewClient(resolve ResolveOwner, client *http.Client) *Client {
	if client == nil {
		client = &http.Client{Timeout: 35 * time.Minute}
	}
	return &Client{resolve: resolve, http: apihttp.NewTestModeClient(client)}
}

func validateRequest(prompt string, width, height int) error {
	if strings.TrimSpace(prompt) == "" || len([]byte(prompt)) > 800 || strings.ContainsRune(prompt, '\x00') {
		return fmt.Errorf("prompt must contain 1 to 800 UTF-8 bytes")
	}
	if !((width == 768 && height == 512) || (width == 512 && height == 768)) {
		return fmt.Errorf("unsupported photographic output dimensions")
	}
	return nil
}

func (c *Client) base(ctx context.Context) (string, error) {
	if c == nil || c.resolve == nil {
		return "", fmt.Errorf("Image Tools owner discovery is unavailable")
	}
	base, err := c.resolve(ctx, "image-tools")
	if err != nil {
		return "", err
	}
	u, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("Image Tools owner URL is invalid")
	}
	return strings.TrimRight(base, "/"), nil
}

func (c *Client) Submit(ctx context.Context, operationID, prompt string, width, height int) (Operation, int, error) {
	if err := validateRequest(prompt, width, height); err != nil {
		return Operation{}, 0, err
	}
	if operationID == "" {
		return Operation{}, 0, fmt.Errorf("operation id is required")
	}
	base, err := c.base(ctx)
	if err != nil {
		return Operation{}, 0, err
	}
	params, err := protojson.Marshal(&aiv1.AIParams{Prompt: prompt, ModelOverride: "sd-1.5", Variations: 1, Width: int32(width), Height: int32(height), AllowByok: false, FallbackPolicy: "local_only", AllowReclaim: proto.Bool(false)})
	if err != nil {
		return Operation{}, 0, err
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("params", string(params)); err != nil {
		return Operation{}, 0, err
	}
	if err := mw.Close(); err != nil {
		return Operation{}, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/ai/text_to_image", &body)
	if err != nil {
		return Operation{}, 0, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Idempotency-Key", operationID)
	resp, err := c.http.Do(req)
	if err != nil {
		return Operation{}, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		return Operation{}, resp.StatusCode, fmt.Errorf("Image Tools rejected local generation (%s)", resp.Status)
	}
	var result aiv1.SubmitAIResponse
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return Operation{}, 0, err
	}
	if err := protojson.Unmarshal(data, &result); err != nil || result.GetJobId() == "" {
		return Operation{}, 0, fmt.Errorf("Image Tools returned an invalid job receipt")
	}
	return Operation{JobID: result.GetJobId(), Operation: "text_to_image", ModelID: result.GetModelId(), Tier: result.GetTier(), State: "queued", Message: "Accepted by local Image Tools"}, resp.StatusCode, nil
}

func (c *Client) Lookup(ctx context.Context, operationID string) (Operation, error) {
	base, err := c.base(ctx)
	if err != nil {
		return Operation{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/ai/operations/"+url.PathEscape(operationID), nil)
	if err != nil {
		return Operation{}, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Operation{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Operation{}, OwnerHTTPError{Status: resp.StatusCode}
	}
	var op Operation
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&op); err != nil || op.JobID == "" {
		return Operation{}, fmt.Errorf("Image Tools returned an invalid operation record")
	}
	return op, nil
}

func (c *Client) jobs(ctx context.Context) (jobsconnect.JobsServiceClient, error) {
	base, err := c.base(ctx)
	if err != nil {
		return nil, err
	}
	return jobsconnect.NewJobsServiceClient(c.http, base), nil
}

func (c *Client) Wait(ctx context.Context, id string) (Operation, error) {
	client, err := c.jobs(ctx)
	if err != nil {
		return Operation{}, err
	}
	resp, err := client.WaitJob(ctx, connect.NewRequest(&jobsv1.WaitJobRequest{Id: id}))
	if err != nil {
		return Operation{}, err
	}
	return fromJob(resp.Msg.GetJob()), nil
}

func (c *Client) Cancel(ctx context.Context, id string) (Operation, error) {
	client, err := c.jobs(ctx)
	if err != nil {
		return Operation{}, err
	}
	resp, err := client.CancelJob(ctx, connect.NewRequest(&jobsv1.CancelJobRequest{Id: id}))
	if err != nil {
		return Operation{}, err
	}
	return fromJob(resp.Msg.GetJob()), nil
}

func fromJob(job *jobsv1.Job) Operation {
	if job == nil {
		return Operation{}
	}
	var model, tier string
	if request := job.GetRequest(); request != nil {
		model, tier = request.GetModelId(), request.GetTier()
	}
	return Operation{JobID: job.GetId(), Operation: job.GetOperation(), ModelID: model, Tier: tier, State: strings.TrimPrefix(strings.ToLower(job.GetState().String()), "job_state_"), Progress: int(job.GetProgress()), Message: job.GetMessage(), Error: job.GetError(), ResultRef: job.GetResultRef(), ResultMeta: job.GetResultMeta()}
}

func (c *Client) PNG(ctx context.Context, ref string, wantWidth, wantHeight int) ([]byte, error) {
	if !((wantWidth == 768 && wantHeight == 512) || (wantWidth == 512 && wantHeight == 768)) {
		return nil, fmt.Errorf("requested image dimensions are unsupported")
	}
	clean := path.Clean(ref)
	if !strings.HasPrefix(clean, "out/") || clean != ref || strings.Contains(ref, "\\") || strings.Contains(ref, "..") {
		return nil, fmt.Errorf("owner result reference is not a managed output")
	}
	base, err := c.base(ctx)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/blobs/"+ref, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Image Tools result content returned %s", resp.Status)
	}
	if resp.ContentLength > MaxPNGBytes {
		return nil, fmt.Errorf("generated image exceeds 12 MiB")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxPNGBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > MaxPNGBytes || !strings.EqualFold(strings.Split(resp.Header.Get("Content-Type"), ";")[0], "image/png") {
		return nil, fmt.Errorf("generated result is not a bounded PNG")
	}
	img, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || img.Width != wantWidth || img.Height != wantHeight {
		return nil, fmt.Errorf("generated PNG dimensions do not match the requested output")
	}
	return data, nil
}
