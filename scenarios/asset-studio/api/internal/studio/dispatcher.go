package studio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

// RenderDispatcher is Asset Studio's only production seam for generating a
// still asset. It accepts resolved creative intent and returns producer-owned
// references plus receipt truth; it has no provider credential or model-slug
// input. Image Tools remains responsible for model selection, byte storage,
// and its own durable job lifecycle.
type RenderDispatcher interface {
	Dispatch(context.Context, RenderDispatchRequest) (RenderDispatchResult, error)
}

// AdvisoryAnalyzer obtains a producer-owned automated conformance signal. The
// signal is intentionally separate from RenderDispatcher and contains no
// release decision: Asset Studio retains it solely as operator context.
type AdvisoryAnalyzer interface {
	Analyze(context.Context, string) (AdvisoryResult, error)
}

type AdvisoryResult struct {
	Source string
	Score  float64
	Notes  []string
}

type UnavailableAdvisoryAnalyzer struct{ Reason string }

func (a UnavailableAdvisoryAnalyzer) Analyze(context.Context, string) (AdvisoryResult, error) {
	reason := strings.TrimSpace(a.Reason)
	if reason == "" {
		reason = "Image Tools advisory analyzer is not configured"
	}
	return AdvisoryResult{}, errors.New(reason)
}

type RenderDispatchRequest struct {
	RenderID               string
	Prompt                 string
	CandidateCount         int
	Producer               ProducerKind
	FrameCount             int
	ParentAssetID          string
	ParentReference        string
	CaptureURL             string
	ConditioningReferences []ConditioningReference
}

// BrowserCaptureDispatcher calls Browser Automation Studio's single-location
// CaptureService. It asks for a screenshot and records BAS's opaque artifact
// reference, never the remote server's filesystem path.
type BrowserCaptureDispatcher struct {
	BaseURL string
	Client  *http.Client
}

func (d *BrowserCaptureDispatcher) Dispatch(ctx context.Context, req RenderDispatchRequest) (RenderDispatchResult, error) {
	if d == nil || strings.TrimSpace(d.BaseURL) == "" {
		return RenderDispatchResult{}, errors.New("Browser Automation Studio capture dispatcher URL is not configured")
	}
	if req.Producer != ProducerCapture {
		return RenderDispatchResult{}, fmt.Errorf("Browser Automation Studio dispatcher does not support %s producer", req.Producer)
	}
	if strings.TrimSpace(req.CaptureURL) == "" {
		return RenderDispatchResult{}, errors.New("capture URL is required")
	}
	client := d.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Minute}
	}
	payload, _ := json.Marshal(map[string]any{"url": req.CaptureURL, "captures": []string{"CAPTURE_TYPE_SCREENSHOT"}, "label": "asset-studio-" + req.RenderID})
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(d.BaseURL, "/")+"/browser_automation_studio.v1.capture.CaptureService/Capture", bytes.NewReader(payload))
	if err != nil {
		return RenderDispatchResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil {
		return RenderDispatchResult{}, fmt.Errorf("submit BAS capture: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return RenderDispatchResult{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return RenderDispatchResult{}, fmt.Errorf("BAS capture returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var captured basCaptureResponse
	if err := json.Unmarshal(body, &captured); err != nil {
		return RenderDispatchResult{}, fmt.Errorf("decode BAS capture receipt: %w", err)
	}
	for _, artifact := range captured.Artifacts {
		if artifact.Type != "CAPTURE_TYPE_SCREENSHOT" || strings.TrimSpace(artifact.Reference) == "" {
			continue
		}
		width, _ := strconv.Atoi(artifact.Metadata["width"])
		height, _ := strconv.Atoi(artifact.Metadata["height"])
		return RenderDispatchResult{Backend: "browser-automation-studio", Model: "browser-capture", Parameters: "capture=screenshot", RouteReceipt: captured.ExecutionID, ActualCost: 0, CostRecorded: true, Outputs: []RenderOutput{{Reference: artifact.Reference, MediaType: "image/png", Width: width, Height: height}}}, nil
	}
	return RenderDispatchResult{}, errors.New("BAS capture receipt did not include a screenshot artifact reference")
}

type basCaptureResponse struct {
	ExecutionID string `json:"executionId"`
	Artifacts   []struct {
		Type      string            `json:"type"`
		Reference string            `json:"reference"`
		Metadata  map[string]string `json:"metadata"`
	} `json:"artifacts"`
}

type RenderDispatchResult struct {
	Backend      string
	Model        string
	Seed         string
	Parameters   string
	RouteReceipt string
	ActualCost   float64
	CostRecorded bool
	Outputs      []RenderOutput
}

type RenderOutput struct {
	Reference string
	MediaType string
	Width     int
	Height    int
}

// UnavailableRenderDispatcher is the safe default. It prevents any runtime
// configuration mistake from recreating the old synthetic-success behavior.
type UnavailableRenderDispatcher struct{ Reason string }

func (d UnavailableRenderDispatcher) Dispatch(context.Context, RenderDispatchRequest) (RenderDispatchResult, error) {
	reason := strings.TrimSpace(d.Reason)
	if reason == "" {
		reason = "Image Tools dispatcher is not configured"
	}
	return RenderDispatchResult{}, errors.New(reason)
}

// ProducerDispatchers routes producer-neutral media intents to a dedicated
// scenario capability. Unsupported producers fail explicitly, preserving a
// recoverable receipt rather than falling back to an unrelated generator.
type ProducerDispatchers map[ProducerKind]RenderDispatcher

func (d ProducerDispatchers) Dispatch(ctx context.Context, req RenderDispatchRequest) (RenderDispatchResult, error) {
	producer := req.Producer
	if producer == "" {
		producer = ProducerImage
	}
	dispatcher := d[producer]
	if dispatcher == nil {
		return RenderDispatchResult{}, fmt.Errorf("%s producer is not configured", producer)
	}
	return dispatcher.Dispatch(ctx, req)
}

// ImageToolsDispatcher composes the public Image Tools submit edge and durable
// JobsService wait edge. It never reads, copies, or stores output bytes: its
// output reference identifies the Image Tools producer store.
type ImageToolsDispatcher struct {
	BaseURL string
	Client  *http.Client
}

// ImageToolsAdvisoryAnalyzer uses Image Tools' pure-Go quality assessment on
// an Image Tools-owned reference. The bytes never transit Asset Studio.
type ImageToolsAdvisoryAnalyzer struct {
	BaseURL string
	Client  *http.Client
}

func (a *ImageToolsAdvisoryAnalyzer) Analyze(ctx context.Context, reference string) (AdvisoryResult, error) {
	if a == nil || strings.TrimSpace(a.BaseURL) == "" {
		return AdvisoryResult{}, errors.New("Image Tools advisory analyzer URL is not configured")
	}
	inputRef := strings.TrimPrefix(reference, "image-tools://")
	if inputRef == "" || inputRef == reference {
		return AdvisoryResult{}, errors.New("advisory analysis requires an Image Tools-owned artifact reference")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("input_ref", inputRef); err != nil {
		return AdvisoryResult{}, err
	}
	if err := writer.Close(); err != nil {
		return AdvisoryResult{}, err
	}
	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(a.BaseURL, "/")+"/api/v1/analysis/quality_assessment", &body)
	if err != nil {
		return AdvisoryResult{}, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		return AdvisoryResult{}, fmt.Errorf("request Image Tools advisory analysis: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return AdvisoryResult{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return AdvisoryResult{}, fmt.Errorf("Image Tools advisory analysis returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var payload struct {
		Quality struct {
			OverallScore float64  `json:"overallScore"`
			Notes        []string `json:"notes"`
		} `json:"quality"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return AdvisoryResult{}, fmt.Errorf("decode Image Tools advisory analysis: %w", err)
	}
	if payload.Quality.OverallScore < 0 || payload.Quality.OverallScore > 1 {
		return AdvisoryResult{}, errors.New("Image Tools advisory analysis returned invalid score")
	}
	return AdvisoryResult{Source: "image-tools/quality_assessment", Score: payload.Quality.OverallScore, Notes: payload.Quality.Notes}, nil
}

// GatewayVideoDispatcher uses the provider-neutral AI Gateway media contract
// for video. Asset Studio supplies creative intent and waits once on a durable
// receipt; it never receives provider credentials, URLs, or model selectors.
type GatewayVideoDispatcher struct {
	BaseURL string
	Client  *http.Client
}

func (d *GatewayVideoDispatcher) Dispatch(ctx context.Context, req RenderDispatchRequest) (RenderDispatchResult, error) {
	if d == nil || strings.TrimSpace(d.BaseURL) == "" {
		return RenderDispatchResult{}, errors.New("AI Gateway video dispatcher URL is not configured")
	}
	if req.Producer != ProducerVideo {
		return RenderDispatchResult{}, fmt.Errorf("AI Gateway video dispatcher does not support %s producer", req.Producer)
	}
	if strings.TrimSpace(req.Prompt) == "" || req.CandidateCount < 1 {
		return RenderDispatchResult{}, errors.New("video prompt and candidate count are required")
	}
	client := d.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Minute}
	}
	submission := map[string]any{
		"request": map[string]any{
			"kind": "REQUEST_KIND_VIDEO_GENERATION", "role": "video.generate.default", "profile": "PROFILE_QUALITY_FIRST", "privacyClass": "PRIVACY_CLASS_INTERNAL", "scenario": "asset-studio", "operation": "video_render", "requestId": req.RenderID,
		},
		"prompt": req.Prompt, "outputCount": req.CandidateCount, "idempotencyKey": "asset-studio-video-" + req.RenderID,
	}
	data, err := json.Marshal(submission)
	if err != nil {
		return RenderDispatchResult{}, err
	}
	base := strings.TrimRight(d.BaseURL, "/")
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/vrooli.ai_gateway.v1.routing.RoutingService/SubmitMedia", bytes.NewReader(data))
	if err != nil {
		return RenderDispatchResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil {
		return RenderDispatchResult{}, fmt.Errorf("submit Gateway video: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return RenderDispatchResult{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return RenderDispatchResult{}, fmt.Errorf("Gateway video submit returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var submitted gatewayMediaResponse
	if err := json.Unmarshal(body, &submitted); err != nil || submitted.Execution.ExecutionID == "" {
		return RenderDispatchResult{}, fmt.Errorf("decode Gateway video submission: %w", err)
	}
	return d.wait(ctx, client, base, submitted.Execution.ExecutionID)
}

type gatewayMediaResponse struct {
	Execution gatewayMediaExecution `json:"execution"`
}
type gatewayMediaExecution struct {
	ExecutionID   string               `json:"executionId"`
	Status        string               `json:"status"`
	ActualCostUSD float64              `json:"actualCostUsd"`
	ResolvedModel string               `json:"resolvedModel"`
	Seed          string               `json:"seed"`
	ErrorCode     string               `json:"errorCode"`
	ErrorMessage  string               `json:"errorMessage"`
	Outputs       []gatewayMediaOutput `json:"outputs"`
}
type gatewayMediaOutput struct {
	Reference string `json:"reference"`
	MediaType string `json:"mediaType"`
}

func (d *GatewayVideoDispatcher) wait(ctx context.Context, client *http.Client, base, executionID string) (RenderDispatchResult, error) {
	body, _ := json.Marshal(map[string]string{"executionId": executionID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/vrooli.ai_gateway.v1.routing.RoutingService/WaitMediaExecution", bytes.NewReader(body))
	if err != nil {
		return RenderDispatchResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return RenderDispatchResult{}, fmt.Errorf("wait for Gateway video: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return RenderDispatchResult{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return RenderDispatchResult{}, fmt.Errorf("Gateway video wait returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var waited gatewayMediaResponse
	if err := json.Unmarshal(data, &waited); err != nil {
		return RenderDispatchResult{}, fmt.Errorf("decode Gateway video receipt: %w", err)
	}
	if waited.Execution.Status != "MEDIA_EXECUTION_STATUS_SUCCEEDED" {
		return RenderDispatchResult{}, fmt.Errorf("Gateway video %s failed (%s): %s", executionID, waited.Execution.ErrorCode, waited.Execution.ErrorMessage)
	}
	if waited.Execution.ResolvedModel == "" || len(waited.Execution.Outputs) == 0 {
		return RenderDispatchResult{}, errors.New("Gateway video receipt lacks model or outputs")
	}
	result := RenderDispatchResult{Backend: "ai-gateway", Model: waited.Execution.ResolvedModel, Seed: waited.Execution.Seed, Parameters: "producer=video", RouteReceipt: executionID, ActualCost: waited.Execution.ActualCostUSD, CostRecorded: true}
	for _, output := range waited.Execution.Outputs {
		if strings.TrimSpace(output.Reference) == "" || strings.TrimSpace(output.MediaType) == "" {
			return RenderDispatchResult{}, errors.New("Gateway video receipt contains incomplete output metadata")
		}
		result.Outputs = append(result.Outputs, RenderOutput{Reference: output.Reference, MediaType: output.MediaType})
	}
	return result, nil
}

func (d *ImageToolsDispatcher) Dispatch(ctx context.Context, req RenderDispatchRequest) (RenderDispatchResult, error) {
	if req.Producer != "" && req.Producer != ProducerImage && req.Producer != ProducerRefine {
		return RenderDispatchResult{}, fmt.Errorf("Image Tools dispatcher does not support %s producer", req.Producer)
	}
	if d == nil || strings.TrimSpace(d.BaseURL) == "" {
		return RenderDispatchResult{}, errors.New("Image Tools dispatcher URL is not configured")
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return RenderDispatchResult{}, errors.New("resolved render prompt is required")
	}
	if req.CandidateCount < 1 {
		return RenderDispatchResult{}, errors.New("candidate count must be positive")
	}
	client := d.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Minute}
	}
	op := "text_to_image"
	inputRef := ""
	if req.Producer == ProducerRefine {
		op = "edit_instruct"
		inputRef = strings.TrimPrefix(req.ParentReference, "image-tools://")
		if strings.TrimSpace(inputRef) == "" || inputRef == req.ParentReference {
			return RenderDispatchResult{}, errors.New("refinement requires an Image Tools-owned parent artifact reference")
		}
	}
	result := RenderDispatchResult{Backend: "image-tools", Parameters: "operation=" + op, CostRecorded: true}
	for i := 0; i < req.CandidateCount; i++ {
		submitted, err := d.submit(ctx, client, op, req.Prompt, inputRef, req.ConditioningReferences)
		if err != nil {
			return RenderDispatchResult{}, err
		}
		if submitted.Tier == "byok-cloud" {
			// Image Tools currently does not expose an actual provider charge on
			// its job receipt. A successful cloud image with an invented zero
			// cost would make Asset Studio provenance dishonest.
			return RenderDispatchResult{}, errors.New("Image Tools cloud receipt did not report actual cost")
		}
		job, err := d.wait(ctx, client, submitted.JobID)
		if err != nil {
			return RenderDispatchResult{}, err
		}
		if job.Job.State != "JOB_STATE_SUCCEEDED" {
			return RenderDispatchResult{}, fmt.Errorf("Image Tools job %s ended in %s: %s", submitted.JobID, job.Job.State, job.Job.Error)
		}
		mediaType, err := mediaTypeForReference(job.Job.ResultRef)
		if err != nil {
			return RenderDispatchResult{}, err
		}
		result.Model = submitted.ModelID
		result.RouteReceipt = submitted.JobID
		result.Outputs = append(result.Outputs, RenderOutput{Reference: "image-tools://" + job.Job.ResultRef, MediaType: mediaType})
	}
	return result, nil
}

type imageToolsSubmitResponse struct {
	JobID   string `json:"jobId"`
	ModelID string `json:"modelId"`
	Tier    string `json:"tier"`
}

type imageToolsWaitResponse struct {
	Job struct {
		State     string `json:"state"`
		ResultRef string `json:"resultRef"`
		Error     string `json:"error"`
	} `json:"job"`
}

func (d *ImageToolsDispatcher) submit(ctx context.Context, client *http.Client, operation, prompt, inputRef string, conditioning []ConditioningReference) (imageToolsSubmitResponse, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	paramsBody := map[string]any{"prompt": prompt, "variations": 1, "allowByok": true}
	for _, reference := range conditioning {
		if reference.Kind != "adapter" {
			continue
		}
		if strings.TrimSpace(reference.ID) == "" {
			return imageToolsSubmitResponse{}, errors.New("adapter conditioning reference requires an id")
		}
		paramsBody["adapters"] = appendAdapterParam(paramsBody["adapters"], map[string]any{"adapterId": reference.ID})
	}
	params, _ := json.Marshal(paramsBody)
	if err := writer.WriteField("params", string(params)); err != nil {
		return imageToolsSubmitResponse{}, err
	}
	if strings.TrimSpace(inputRef) != "" {
		if err := writer.WriteField("input_ref", inputRef); err != nil {
			return imageToolsSubmitResponse{}, err
		}
	}
	if err := writer.Close(); err != nil {
		return imageToolsSubmitResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(d.BaseURL, "/")+"/api/v1/ai/"+operation, &body)
	if err != nil {
		return imageToolsSubmitResponse{}, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		return imageToolsSubmitResponse{}, fmt.Errorf("submit Image Tools render: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return imageToolsSubmitResponse{}, err
	}
	if resp.StatusCode != http.StatusAccepted {
		return imageToolsSubmitResponse{}, fmt.Errorf("Image Tools submit returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var submitted imageToolsSubmitResponse
	if err := json.Unmarshal(data, &submitted); err != nil || strings.TrimSpace(submitted.JobID) == "" {
		return imageToolsSubmitResponse{}, fmt.Errorf("decode Image Tools submit receipt: %w", err)
	}
	return submitted, nil
}

func appendAdapterParam(existing any, adapter map[string]any) []map[string]any {
	if adapters, ok := existing.([]map[string]any); ok {
		return append(adapters, adapter)
	}
	return []map[string]any{adapter}
}

func (d *ImageToolsDispatcher) wait(ctx context.Context, client *http.Client, jobID string) (imageToolsWaitResponse, error) {
	body, _ := json.Marshal(map[string]string{"id": jobID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(d.BaseURL, "/")+"/vrooli.image_tools.v1.jobs.JobsService/WaitJob", bytes.NewReader(body))
	if err != nil {
		return imageToolsWaitResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return imageToolsWaitResponse{}, fmt.Errorf("wait for Image Tools job: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return imageToolsWaitResponse{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return imageToolsWaitResponse{}, fmt.Errorf("Image Tools wait returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var waited imageToolsWaitResponse
	if err := json.Unmarshal(data, &waited); err != nil {
		return imageToolsWaitResponse{}, fmt.Errorf("decode Image Tools job receipt: %w", err)
	}
	return waited, nil
}

func mediaTypeForReference(reference string) (string, error) {
	switch strings.ToLower(path.Ext(strings.TrimSpace(reference))) {
	case ".png":
		return "image/png", nil
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	case ".webp":
		return "image/webp", nil
	default:
		return "", fmt.Errorf("Image Tools output %q has no supported media type", reference)
	}
}
