package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	bas_execution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bas_workflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"

	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
)

// basWorkflowsAPI is the subset of the generated WorkflowsServiceClient
// surface that BrowserAutomationClient calls. Stating it locally makes the
// client trivially mockable in tests (no httptest + Connect handler needed).
type basWorkflowsAPI interface {
	ExecuteAdhocWorkflow(ctx context.Context, req *connect.Request[bas_execution.ExecuteAdhocRequest]) (*connect.Response[bas_execution.ExecuteAdhocResponse], error)
}

// basExecutionsAPI is the subset of the generated ExecutionsServiceClient
// surface that BrowserAutomationClient calls.
type basExecutionsAPI interface {
	GetExecution(ctx context.Context, req *connect.Request[basapi.GetExecutionRequest]) (*connect.Response[basapi.GetExecutionResponse], error)
	GetExecutionScreenshots(ctx context.Context, req *connect.Request[basapi.GetExecutionScreenshotsRequest]) (*connect.Response[bas_execution.GetScreenshotsResponse], error)
	GetExecutionRecordedVideos(ctx context.Context, req *connect.Request[basapi.GetExecutionArtifactsRequest]) (*connect.Response[basapi.GetExecutionVideosResponse], error)
}

// BrowserAutomationClient is a Connect-RPC client for browser-automation-studio.
//
// The four workflow/execution RPCs use Connect; only the binary artifact
// downloads (screenshot PNGs, video bytes) stay on raw HTTP because they
// fetch large opaque blobs over plain URLs.
type BrowserAutomationClient struct {
	BaseClient

	// workflowsFactory and executionsFactory build the typed Connect clients
	// once a base URL has been resolved. They are package-level seams in
	// tests so we can inject stubs without spinning up Connect handlers.
	workflowsFactory  func(baseURL string) basWorkflowsAPI
	executionsFactory func(baseURL string) basExecutionsAPI
}

// NewBrowserAutomationClient creates a new BAS client with the given timeout.
func NewBrowserAutomationClient(timeout time.Duration) *BrowserAutomationClient {
	base := NewBaseClient("browser-automation-studio", timeout)
	httpClient := &http.Client{Timeout: timeout}
	return &BrowserAutomationClient{
		BaseClient: base,
		workflowsFactory: func(baseURL string) basWorkflowsAPI {
			return apiconnect.NewWorkflowsServiceClient(httpClient, baseURL)
		},
		executionsFactory: func(baseURL string) basExecutionsAPI {
			return apiconnect.NewExecutionsServiceClient(httpClient, baseURL)
		},
	}
}

func (c *BrowserAutomationClient) workflows(ctx context.Context) (basWorkflowsAPI, error) {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve BAS url: %w", err)
	}
	factory := c.workflowsFactory
	if factory == nil {
		factory = defaultWorkflowsFactory(c.httpClient)
	}
	return factory(baseURL), nil
}

func (c *BrowserAutomationClient) executions(ctx context.Context) (basExecutionsAPI, error) {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve BAS url: %w", err)
	}
	factory := c.executionsFactory
	if factory == nil {
		factory = defaultExecutionsFactory(c.httpClient)
	}
	return factory(baseURL), nil
}

func defaultWorkflowsFactory(httpClient *http.Client) func(baseURL string) basWorkflowsAPI {
	return func(baseURL string) basWorkflowsAPI {
		return apiconnect.NewWorkflowsServiceClient(httpClient, baseURL)
	}
}

func defaultExecutionsFactory(httpClient *http.Client) func(baseURL string) basExecutionsAPI {
	return func(baseURL string) basExecutionsAPI {
		return apiconnect.NewExecutionsServiceClient(httpClient, baseURL)
	}
}

// GetScreenshotData fetches raw screenshot bytes and content-type using the
// artifact's URL. The artifact URL is whatever BAS reports in
// TimelineScreenshot.url; it's a plain HTTP path served by the BAS asset
// proxy, not a Connect-RPC.
func (c *BrowserAutomationClient) GetScreenshotData(ctx context.Context, screenshotURL string) ([]byte, string, error) {
	return c.doRaw(ctx, screenshotURL)
}

// ExecuteAdhocWorkflow runs a workflow definition without persisting it.
//
// The workflow JSON is parsed into a typed WorkflowDefinitionV2 proto via
// protojson — BAS no longer accepts the legacy untyped JSON shape over the
// wire. requiresVideo maps to ExecuteWorkflowOptions.requires_video.
func (c *BrowserAutomationClient) ExecuteAdhocWorkflow(ctx context.Context, req BASExecuteAdhocRequest, requiresVideo bool) (*BASExecuteResponse, error) {
	wf := &bas_workflows.WorkflowDefinitionV2{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(req.FlowDefinition, wf); err != nil {
		return nil, fmt.Errorf("parse workflow definition: %w", err)
	}

	params := &bas_execution.ExecutionParameters{}
	if projectRoot, ok := req.Parameters["project_root"].(string); ok && projectRoot != "" {
		params.ProjectRoot = &projectRoot
	}

	pbReq := &bas_execution.ExecuteAdhocRequest{
		FlowDefinition: wf,
		Parameters:     params,
		Options:        &bas_execution.ExecuteWorkflowOptions{RequiresVideo: requiresVideo},
	}
	if len(req.Metadata) > 0 {
		meta := &bas_execution.ExecutionMetadata{
			Name:        req.Metadata["name"],
			Description: req.Metadata["description"],
		}
		pbReq.Metadata = meta
	}

	client, err := c.workflows(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.ExecuteAdhocWorkflow(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("browser-automation-studio ExecuteAdhocWorkflow: %w", err)
	}
	return executeAdhocResponseToLegacy(resp.Msg), nil
}

// GetExecutionStatus reports the current status of a BAS execution. Returns
// the proto status enum name (e.g. "EXECUTION_STATUS_COMPLETED") so existing
// callers' string comparisons continue to work.
func (c *BrowserAutomationClient) GetExecutionStatus(ctx context.Context, executionID string) (*BASExecutionDetail, error) {
	client, err := c.executions(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetExecution(ctx, connect.NewRequest(&basapi.GetExecutionRequest{ExecutionId: executionID}))
	if err != nil {
		return nil, fmt.Errorf("browser-automation-studio GetExecution: %w", err)
	}
	return executionToLegacyDetail(resp.Msg.GetExecution()), nil
}

// PollExecutionCompletion polls BAS until the execution reaches a terminal
// status (completed, failed, cancelled) or the context is cancelled.
func (c *BrowserAutomationClient) PollExecutionCompletion(ctx context.Context, executionID string, pollInterval time.Duration) (*BASExecutionDetail, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			detail, err := c.GetExecutionStatus(ctx, executionID)
			if err != nil {
				return nil, fmt.Errorf("poll execution %s: %w", executionID, err)
			}
			if isTerminalBASStatus(detail.Status) {
				return detail, nil
			}
		}
	}
}

// isTerminalBASStatus returns true for BAS execution statuses that indicate completion.
func isTerminalBASStatus(status string) bool {
	switch status {
	case "EXECUTION_STATUS_COMPLETED", "EXECUTION_STATUS_FAILED", "EXECUTION_STATUS_CANCELLED":
		return true
	default:
		return false
	}
}

// GetScreenshots lists screenshots captured during an execution.
func (c *BrowserAutomationClient) GetScreenshots(ctx context.Context, executionID string) (*BASScreenshotsResponse, error) {
	client, err := c.executions(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetExecutionScreenshots(ctx, connect.NewRequest(&basapi.GetExecutionScreenshotsRequest{ExecutionId: executionID}))
	if err != nil {
		return nil, fmt.Errorf("browser-automation-studio GetExecutionScreenshots: %w", err)
	}
	return screenshotsResponseToLegacy(resp.Msg), nil
}

// GetRecordedVideos lists recorded video artifacts for an execution.
func (c *BrowserAutomationClient) GetRecordedVideos(ctx context.Context, executionID string) (*BASRecordedVideosResponse, error) {
	client, err := c.executions(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetExecutionRecordedVideos(ctx, connect.NewRequest(&basapi.GetExecutionArtifactsRequest{ExecutionId: executionID}))
	if err != nil {
		return nil, fmt.Errorf("browser-automation-studio GetExecutionRecordedVideos: %w", err)
	}
	return videosResponseToLegacy(resp.Msg), nil
}

// GetVideoData fetches raw video bytes and content-type using the artifact's storage URL.
func (c *BrowserAutomationClient) GetVideoData(ctx context.Context, storageURL string) ([]byte, string, error) {
	return c.doRaw(ctx, storageURL)
}

func (c *BrowserAutomationClient) doRaw(ctx context.Context, path string) ([]byte, string, error) {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("resolve BAS url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return nil, "", fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("BAS request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", c.parseError(resp)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read response body: %w", err)
	}
	return data, resp.Header.Get("Content-Type"), nil
}

// -----------------------------------------------------------------------------
// Proto → legacy conversions.
//
// The legacy struct shapes (BASExecuteResponse, BASExecutionDetail,
// BASScreenshotsResponse, BASRecordedVideosResponse) are preserved so the
// visual-capture (screenshots) caller doesn't have to migrate alongside this
// client.
// -----------------------------------------------------------------------------

func executeAdhocResponseToLegacy(msg *bas_execution.ExecuteAdhocResponse) *BASExecuteResponse {
	if msg == nil {
		return &BASExecuteResponse{}
	}
	return &BASExecuteResponse{
		ExecutionID: msg.GetExecutionId(),
		Status:      msg.GetStatus().String(),
		Error:       msg.GetError(),
	}
}

func executionToLegacyDetail(exec *bas_execution.Execution) *BASExecutionDetail {
	if exec == nil {
		return &BASExecutionDetail{}
	}
	return &BASExecutionDetail{
		ExecutionID: exec.GetExecutionId(),
		Status:      exec.GetStatus().String(),
		Error:       exec.GetError(),
	}
}

func screenshotsResponseToLegacy(msg *bas_execution.GetScreenshotsResponse) *BASScreenshotsResponse {
	if msg == nil {
		return &BASScreenshotsResponse{}
	}
	out := &BASScreenshotsResponse{
		Total: int(msg.GetTotal()),
	}
	for _, s := range msg.GetScreenshots() {
		entry := BASExecutionScreenshot{
			StepIndex: int(s.GetStepIndex()),
			StepLabel: s.GetStepLabel(),
		}
		if ts := s.GetTimestamp(); ts != nil {
			entry.Timestamp = ts.AsTime().Format(time.RFC3339Nano)
		}
		if shot := s.GetScreenshot(); shot != nil {
			entry.Screenshot.ArtifactID = shot.GetArtifactId()
			entry.Screenshot.Url = shot.GetUrl()
			entry.Screenshot.ThumbnailUrl = shot.GetThumbnailUrl()
			entry.Screenshot.ContentType = shot.GetContentType()
			entry.Screenshot.Width = int(shot.GetWidth())
			entry.Screenshot.Height = int(shot.GetHeight())
		}
		out.Screenshots = append(out.Screenshots, entry)
	}
	return out
}

func videosResponseToLegacy(msg *basapi.GetExecutionVideosResponse) *BASRecordedVideosResponse {
	if msg == nil {
		return &BASRecordedVideosResponse{}
	}
	out := &BASRecordedVideosResponse{
		ExecutionID: msg.GetExecutionId(),
	}
	for _, v := range msg.GetVideos() {
		out.Videos = append(out.Videos, BASVideoArtifact{
			ArtifactID:  v.GetArtifactId(),
			StorageURL:  v.GetStorageUrl(),
			ContentType: v.GetContentType(),
			SizeBytes:   v.GetSizeBytes(),
		})
	}
	return out
}
