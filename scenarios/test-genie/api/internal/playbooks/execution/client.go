package execution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	playbooksconfig "test-genie/internal/playbooks/config"

	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	"github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture/captureconnect"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	commonpb "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

const (
	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 15 * time.Second
	// HealthCheckTimeout is the timeout for health check requests.
	HealthCheckTimeout = 5 * time.Second
	// HealthCheckWaitTimeout is how long to wait for BAS to become healthy.
	HealthCheckWaitTimeout = 45 * time.Second
	// WorkflowExecutionTimeout is how long to wait for a workflow to complete.
	WorkflowExecutionTimeout = 3 * time.Minute
)

var (
	// protoJSONUnmarshal accepts BAS responses with the strictness that
	// catches contract drift — unknown fields fail loudly.
	protoJSONUnmarshal = protojson.UnmarshalOptions{DiscardUnknown: false}
	// protoJSONMarshal serializes proto responses back to JSON for artifact
	// persistence (timeline blobs written to disk).
	protoJSONMarshal = protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false}
	// flowDefinitionUnmarshal accepts the playbook author's workflow JSON.
	// We allow unknown fields here so test-genie keeps working when BAS
	// adds optional knobs to the workflow proto — the playbook itself is
	// out-of-band content and should not break on additive schema changes.
	flowDefinitionUnmarshal = protojson.UnmarshalOptions{DiscardUnknown: true}
)

// ProgressCallback is called periodically during workflow execution with status updates.
// It receives the current status and elapsed time. Return an error to abort waiting.
// Status is the proto Execution message returned by BAS.
type ProgressCallback func(status *basexecution.Execution, elapsed time.Duration) error

// ExecutionParams contains namespace-aware parameters for workflow execution.
// These support the ${@namespace/path} variable interpolation system in BAS.
type ExecutionParams struct {
	// ProjectRoot is the absolute path to the project root for workflowPath resolution.
	// Example: "/home/user/Vrooli/scenarios/my-scenario/bas"
	ProjectRoot string `json:"project_root,omitempty"`
	// InitialParams are read-only input parameters (@params/ namespace).
	// Subflows inherit parent's params unless explicitly overridden.
	InitialParams map[string]any `json:"initial_params,omitempty"`
	// InitialStore is pre-seeded mutable runtime state (@store/ namespace).
	// Modified via setVariable steps and storeResult params.
	InitialStore map[string]any `json:"initial_store,omitempty"`
	// Env contains project/user configuration (@env/ namespace).
	// Read-only, inherited by all subflows unchanged.
	Env map[string]any `json:"env,omitempty"`
	// ExtraHeaders, when non-empty, are attached to every HTTP request the
	// browser context makes during workflow execution (Playwright's
	// extraHTTPHeaders). Used by test-genie to inject the test-mode header
	// so the scenario's RoutedDB serves the test pool for the run.
	ExtraHeaders map[string]string `json:"-"`
	// Diagnostics selects which rich artifacts BAS captures for this workflow.
	// It maps onto ExecuteWorkflowOptions (video/trace/HAR) and
	// ArtifactCollectionConfig (console/network/DOM).
	Diagnostics playbooksconfig.DiagnosticsConfig `json:"-"`
}

// Client defines the interface for BAS API operations.
type Client interface {
	// Health checks if the BAS API is healthy.
	Health(ctx context.Context) error
	// WaitForHealth waits until BAS becomes healthy or timeout.
	WaitForHealth(ctx context.Context) error
	// ValidateResolved validates a resolved workflow before execution.
	// Returns validation issues or nil if the workflow is valid.
	ValidateResolved(ctx context.Context, definition map[string]any) (*ValidationResult, error)
	// ExecuteWorkflow starts a workflow execution and returns the execution ID.
	ExecuteWorkflow(ctx context.Context, definition map[string]any, name, description string) (string, error)
	// ExecuteWorkflowWithParams starts a workflow execution with namespace-aware parameters.
	// This is the preferred method for callers that support the new variable interpolation system.
	ExecuteWorkflowWithParams(ctx context.Context, definition map[string]any, name, description string, params *ExecutionParams) (string, error)
	// GetStatus retrieves the status of an execution.
	GetStatus(ctx context.Context, executionID string) (*basexecution.Execution, error)
	// WaitForCompletion waits for a workflow to complete.
	WaitForCompletion(ctx context.Context, executionID string) error
	// WaitForCompletionWithProgress waits for completion and calls the progress callback periodically.
	WaitForCompletionWithProgress(ctx context.Context, executionID string, callback ProgressCallback) error
	// GetTimeline retrieves the timeline data for an execution.
	GetTimeline(ctx context.Context, executionID string) (*bastimeline.ExecutionTimeline, []byte, error)
	// GetScreenshots retrieves screenshot metadata for an execution.
	GetScreenshots(ctx context.Context, executionID string) ([]Screenshot, error)
	// GetRecordedVideos lists recorded videos for an execution.
	GetRecordedVideos(ctx context.Context, executionID string) ([]RecordedArtifact, error)
	// GetRecordedTraces lists recorded Playwright traces for an execution.
	GetRecordedTraces(ctx context.Context, executionID string) ([]RecordedArtifact, error)
	// GetRecordedHar lists recorded HAR archives for an execution.
	GetRecordedHar(ctx context.Context, executionID string) ([]RecordedArtifact, error)
	// StreamAsset downloads one asset directly to destination within maxBytes.
	// Callers must persist it before requesting the next rich artifact.
	StreamAsset(ctx context.Context, assetURL string, destination io.Writer, maxBytes int64) (int64, error)
	// BaseURL returns the base URL of the BAS API (for constructing asset URLs).
	BaseURL() string
}

// ValidationResult represents the result of BAS workflow validation.
type ValidationResult struct {
	Valid         bool              `json:"valid"`
	Errors        []ValidationIssue `json:"errors,omitempty"`
	Warnings      []ValidationIssue `json:"warnings,omitempty"`
	SchemaVersion string            `json:"schema_version,omitempty"`
}

// ValidationIssue represents a single validation issue.
type ValidationIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	NodeID   string `json:"node_id,omitempty"`
	NodeType string `json:"node_type,omitempty"`
	Field    string `json:"field,omitempty"`
	Pointer  string `json:"pointer,omitempty"`
	Hint     string `json:"hint,omitempty"`
}

// Screenshot represents screenshot metadata from BAS.
type Screenshot struct {
	ID           string `json:"id"`
	ExecutionID  string `json:"execution_id"`
	StepName     string `json:"step_name"`
	StepIndex    int    `json:"step_index,omitempty"`
	Timestamp    string `json:"timestamp"`
	StorageURL   string `json:"storage_url"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	SizeBytes    int64  `json:"size_bytes,omitempty"`
}

// ClientConfig holds configurable timeouts for the BAS client.
type ClientConfig struct {
	// Timeout is the HTTP client timeout for individual requests.
	Timeout time.Duration
	// HealthCheckWaitTimeout is how long to wait for BAS to become healthy.
	HealthCheckWaitTimeout time.Duration
	// WorkflowExecutionTimeout is how long to wait for a workflow to complete.
	WorkflowExecutionTimeout time.Duration
}

// DefaultClientConfig returns a ClientConfig with default values.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Timeout:                  DefaultTimeout,
		HealthCheckWaitTimeout:   HealthCheckWaitTimeout,
		WorkflowExecutionTimeout: WorkflowExecutionTimeout,
	}
}

// HTTPClient talks to BAS using Connect-RPC for the proto-typed surface
// (workflows, executions) and plain HTTP for the two endpoints that are
// not part of the Connect schema: the /api/v1/health probe and asset
// downloads. The two surfaces share one *http.Client so timeouts and
// transport settings stay in lockstep.
type HTTPClient struct {
	// baseURL keeps the caller-facing "http://host:port/api/v1" form so
	// downstream code that constructs asset URLs against it continues to
	// work unchanged.
	baseURL string
	// connectBaseURL is baseURL with any trailing "/api/v1" stripped.
	// BAS mounts Connect handlers at the bare proto path under the root
	// (e.g. /browser_automation_studio.v1.WorkflowsService/...), not under
	// /api/v1.
	connectBaseURL string
	httpClient     *http.Client
	workflows      apiconnect.WorkflowsServiceClient
	executions     apiconnect.ExecutionsServiceClient
	config         ClientConfig
}

// NewClient creates a new BAS HTTP client with default timeouts.
func NewClient(baseURL string) *HTTPClient {
	return NewClientWithConfig(baseURL, DefaultClientConfig())
}

// NewClientWithConfig creates a new BAS HTTP client with custom timeouts.
func NewClientWithConfig(baseURL string, cfg ClientConfig) *HTTPClient {
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.HealthCheckWaitTimeout <= 0 {
		cfg.HealthCheckWaitTimeout = HealthCheckWaitTimeout
	}
	if cfg.WorkflowExecutionTimeout <= 0 {
		cfg.WorkflowExecutionTimeout = WorkflowExecutionTimeout
	}

	httpClient := &http.Client{Timeout: cfg.Timeout}
	connectBase := strings.TrimRight(baseURL, "/")
	connectBase = strings.TrimSuffix(connectBase, "/api/v1")

	c := &HTTPClient{
		baseURL:        baseURL,
		connectBaseURL: connectBase,
		httpClient:     httpClient,
		config:         cfg,
	}
	c.workflows = apiconnect.NewWorkflowsServiceClient(httpClient, connectBase)
	c.executions = apiconnect.NewExecutionsServiceClient(httpClient, connectBase)
	return c
}

// WithHTTPClient sets a custom HTTP client (for testing). Reconstructs the
// Connect clients so they share the override.
func (c *HTTPClient) WithHTTPClient(client *http.Client) *HTTPClient {
	c.httpClient = client
	c.workflows = apiconnect.NewWorkflowsServiceClient(client, c.connectBaseURL)
	c.executions = apiconnect.NewExecutionsServiceClient(client, c.connectBaseURL)
	return c
}

// Health checks if the BAS API is healthy. This stays REST because the
// health probe is intentionally outside the Connect schema — it is meant
// to be reachable without proto plumbing.
func (c *HTTPClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("health check failed: status %s", resp.Status)
	}
	return nil
}

// WaitForHealth waits until BAS becomes healthy or timeout.
func (c *HTTPClient) WaitForHealth(ctx context.Context) error {
	if err := c.Health(ctx); err == nil {
		return nil
	}

	deadline := time.Now().Add(c.config.HealthCheckWaitTimeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("health check timeout after %s", c.config.HealthCheckWaitTimeout)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := c.Health(ctx); err == nil {
				return nil
			}
		}
	}
}

// ValidateResolved validates a resolved workflow before execution.
// This is the pre-flight check that catches unresolved tokens and schema errors.
func (c *HTTPClient) ValidateResolved(ctx context.Context, definition map[string]any) (*ValidationResult, error) {
	def, err := definitionToProto(definition)
	if err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	resp, err := c.workflows.ValidateResolvedWorkflow(ctx, connect.NewRequest(&basapi.ValidateWorkflowRequest{
		Workflow: def,
	}))
	if err != nil {
		return nil, fmt.Errorf("validation request failed: %w", err)
	}

	result := resp.Msg.GetResult()
	if result == nil {
		return &ValidationResult{Valid: true}, nil
	}

	out := &ValidationResult{
		Valid:         result.GetValid(),
		SchemaVersion: result.GetSchemaVersion(),
	}
	for _, issue := range result.GetErrors() {
		out.Errors = append(out.Errors, protoIssueToLocal(issue))
	}
	for _, issue := range result.GetWarnings() {
		out.Warnings = append(out.Warnings, protoIssueToLocal(issue))
	}
	return out, nil
}

// ExecuteWorkflow starts a workflow execution and returns the execution ID.
// Convenience wrapper that calls ExecuteWorkflowWithParams with nil params.
func (c *HTTPClient) ExecuteWorkflow(ctx context.Context, definition map[string]any, name, description string) (string, error) {
	return c.ExecuteWorkflowWithParams(ctx, definition, name, description, nil)
}

// ExecuteWorkflowWithParams starts a workflow execution with namespace-aware parameters.
func (c *HTTPClient) ExecuteWorkflowWithParams(ctx context.Context, definition map[string]any, name, description string, params *ExecutionParams) (string, error) {
	def, err := definitionToProto(definition)
	if err != nil {
		return "", fmt.Errorf("execute: %w", err)
	}

	req := &basexecution.ExecuteAdhocRequest{
		FlowDefinition: def,
	}

	if strings.TrimSpace(name) != "" || strings.TrimSpace(description) != "" {
		req.Metadata = &basexecution.ExecutionMetadata{
			Name:        strings.TrimSpace(name),
			Description: strings.TrimSpace(description),
		}
	}

	if params != nil {
		execParams := &basexecution.ExecutionParameters{}
		populated := false
		if params.ProjectRoot != "" {
			pr := params.ProjectRoot
			execParams.ProjectRoot = &pr
			populated = true
		}
		if len(params.InitialParams) > 0 {
			execParams.InitialParams = anyMapToJsonValueMap(params.InitialParams)
			populated = true
		}
		if len(params.InitialStore) > 0 {
			execParams.InitialStore = anyMapToJsonValueMap(params.InitialStore)
			populated = true
		}
		if len(params.Env) > 0 {
			execParams.Env = anyMapToJsonValueMap(params.Env)
			populated = true
		}
		if len(params.ExtraHeaders) > 0 {
			execParams.BrowserProfile = &basbase.BrowserProfile{
				ExtraHeaders: params.ExtraHeaders,
			}
			populated = true
		}

		// Diagnostics → execution-time options (video/trace/HAR are recorder
		// flags) and per-execution artifact-collection toggles (console/network/DOM).
		diag := params.Diagnostics
		if diag.Video || diag.Trace || diag.HAR {
			req.Options = &basexecution.ExecuteWorkflowOptions{
				RequiresVideo: diag.Video,
				RequiresTrace: diag.Trace,
				RequiresHar:   diag.HAR,
			}
		}
		if diag.Console || diag.Network || diag.DOM {
			console, network, dom := diag.Console, diag.Network, diag.DOM
			execParams.ArtifactConfig = &basexecution.ArtifactCollectionConfig{
				CollectConsoleLogs:   &console,
				CollectNetworkEvents: &network,
				CollectDomSnapshots:  &dom,
			}
			populated = true
		}

		if populated {
			req.Parameters = execParams
		}
	}

	resp, err := c.workflows.ExecuteAdhocWorkflow(ctx, connect.NewRequest(req))
	if err != nil {
		return "", fmt.Errorf("workflow execution failed: %w", err)
	}

	executionID := strings.TrimSpace(resp.Msg.GetExecutionId())
	if executionID == "" {
		return "", errors.New("execution_id missing in response")
	}
	return executionID, nil
}

// GetStatus retrieves the status of an execution.
func (c *HTTPClient) GetStatus(ctx context.Context, executionID string) (*basexecution.Execution, error) {
	resp, err := c.executions.GetExecution(ctx, connect.NewRequest(&basapi.GetExecutionRequest{
		ExecutionId: executionID,
	}))
	if err != nil {
		return nil, fmt.Errorf("status lookup failed: %w", err)
	}
	return resp.Msg.GetExecution(), nil
}

// WaitForCompletion waits for a workflow to complete.
func (c *HTTPClient) WaitForCompletion(ctx context.Context, executionID string) error {
	return c.WaitForCompletionWithProgress(ctx, executionID, nil)
}

// WaitForCompletionWithProgress waits for completion and calls the progress callback periodically.
// The callback is invoked approximately every 5 seconds with the current status.
func (c *HTTPClient) WaitForCompletionWithProgress(ctx context.Context, executionID string, callback ProgressCallback) error {
	start := time.Now()

	checkStatus := func() (done bool, status *basexecution.Execution, err error) {
		status, err = c.GetStatus(ctx, executionID)
		if err != nil {
			return true, status, err
		}

		execStatus := status.GetStatus()
		switch execStatus {
		case basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED:
			return true, status, nil
		case basbase.ExecutionStatus_EXECUTION_STATUS_FAILED:
			if status != nil && status.Error != nil {
				if msg := strings.TrimSpace(status.GetError()); msg != "" {
					return true, status, fmt.Errorf("workflow failed: %s", msg)
				}
			}
			return true, status, fmt.Errorf("workflow failed with status %s", execStatus.String())
		case basbase.ExecutionStatus_EXECUTION_STATUS_CANCELLED:
			return true, status, fmt.Errorf("workflow cancelled")
		}
		return false, status, nil
	}

	if done, status, err := checkStatus(); done {
		if callback != nil {
			_ = callback(status, time.Since(start))
		}
		return err
	}

	deadline := time.Now().Add(c.config.WorkflowExecutionTimeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastProgressReport := time.Now()
	progressInterval := 5 * time.Second

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("workflow execution timed out after %s", c.config.WorkflowExecutionTimeout)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			done, status, err := checkStatus()

			if callback != nil && time.Since(lastProgressReport) >= progressInterval {
				if callbackErr := callback(status, time.Since(start)); callbackErr != nil {
					return fmt.Errorf("progress callback aborted: %w", callbackErr)
				}
				lastProgressReport = time.Now()
			}

			if done {
				if callback != nil {
					_ = callback(status, time.Since(start))
				}
				return err
			}
		}
	}
}

// GetTimeline retrieves the timeline data for an execution. Returns the
// parsed proto alongside a JSON re-encoding so callers that need to persist
// the timeline as an artifact can do so without a second round trip.
func (c *HTTPClient) GetTimeline(ctx context.Context, executionID string) (*bastimeline.ExecutionTimeline, []byte, error) {
	resp, err := c.executions.GetExecutionTimeline(ctx, connect.NewRequest(&basapi.GetExecutionTimelineRequest{
		ExecutionId: executionID,
	}))
	if err != nil {
		return nil, nil, fmt.Errorf("timeline fetch failed: %w", err)
	}

	timeline := resp.Msg
	data, marshalErr := protoJSONMarshal.Marshal(timeline)
	if marshalErr != nil {
		return timeline, nil, fmt.Errorf("re-encode timeline: %w", marshalErr)
	}
	return timeline, data, nil
}

// TimelineSummary contains parsed timeline statistics.
type TimelineSummary struct {
	TotalSteps    int
	TotalAsserts  int
	AssertsPassed int
}

// String returns a human-readable summary string.
func (s TimelineSummary) String() string {
	if s.TotalAsserts > 0 {
		return fmt.Sprintf(" (%d steps, %d/%d assertions passed)", s.TotalSteps, s.AssertsPassed, s.TotalAsserts)
	}
	if s.TotalSteps > 0 {
		return fmt.Sprintf(" (%d steps)", s.TotalSteps)
	}
	return ""
}

// TimelineParseError indicates timeline data could not be parsed.
type TimelineParseError struct {
	RawData []byte
	Cause   error
}

func (e *TimelineParseError) Error() string {
	return fmt.Sprintf("failed to parse timeline data (%d bytes): %v (check BAS timeline proto contract)", len(e.RawData), e.Cause)
}

func (e *TimelineParseError) Unwrap() error {
	return e.Cause
}

// SummarizeTimeline extracts a summary from timeline data.
// Deprecated: Use ParseTimeline for better error handling.
func SummarizeTimeline(data []byte) string {
	summary, _ := ParseTimeline(data)
	return summary.String()
}

// ParseTimeline parses timeline data and returns a summary using the proto contract.
// Returns a TimelineParseError if the data cannot be parsed, allowing callers
// to save the raw data for debugging.
func ParseTimeline(data []byte) (TimelineSummary, error) {
	parsed, err := ParseFullTimeline(data)
	if err != nil {
		return TimelineSummary{}, err
	}
	if parsed == nil {
		return TimelineSummary{}, nil
	}
	return parsed.Summary, nil
}

// BaseURL returns the base URL of the BAS API.
func (c *HTTPClient) BaseURL() string {
	return c.baseURL
}

// CaptureServiceClient returns a BAS CaptureService client bound to this client's
// shared HTTP client and Connect base URL. Reusing the workflow client's
// connection/discovery plumbing keeps test-genie on exactly one BAS client — the
// single-location capture verb and the workflow engine share transport and
// timeouts.
func (c *HTTPClient) CaptureServiceClient() captureconnect.CaptureServiceClient {
	return captureconnect.NewCaptureServiceClient(c.httpClient, c.connectBaseURL)
}

// GetScreenshots retrieves screenshot metadata for an execution.
func (c *HTTPClient) GetScreenshots(ctx context.Context, executionID string) ([]Screenshot, error) {
	resp, err := c.executions.GetExecutionScreenshots(ctx, connect.NewRequest(&basapi.GetExecutionScreenshotsRequest{
		ExecutionId: executionID,
	}))
	if err != nil {
		return nil, fmt.Errorf("screenshots fetch failed: %w", err)
	}

	msg := resp.Msg
	out := make([]Screenshot, 0, len(msg.GetScreenshots()))
	for _, s := range msg.GetScreenshots() {
		shot := s.GetScreenshot()
		entry := Screenshot{
			StepName:    s.GetStepLabel(),
			StepIndex:   int(s.GetStepIndex()),
			ExecutionID: msg.GetExecutionId(),
		}
		if shot != nil {
			entry.ID = shot.GetArtifactId()
			entry.StorageURL = shot.GetUrl()
			entry.ThumbnailURL = shot.GetThumbnailUrl()
			entry.Width = int(shot.GetWidth())
			entry.Height = int(shot.GetHeight())
			entry.SizeBytes = shot.GetSizeBytes()
		}
		if ts := s.GetTimestamp(); ts != nil {
			entry.Timestamp = ts.AsTime().UTC().Format(time.RFC3339Nano)
		}
		out = append(out, entry)
	}
	return out, nil
}

// RecordedArtifact references a recorded diagnostic file (video, trace, or HAR)
// produced when the corresponding ExecuteWorkflowOptions flag was set.
type RecordedArtifact struct {
	Filename    string
	StorageURL  string
	ContentType string
}

func toRecordedArtifacts(files []*basapi.ExecutionFileArtifact) []RecordedArtifact {
	out := make([]RecordedArtifact, 0, len(files))
	for _, f := range files {
		if f == nil {
			continue
		}
		out = append(out, RecordedArtifact{
			Filename:    f.GetArtifactId(),
			StorageURL:  f.GetStorageUrl(),
			ContentType: f.GetContentType(),
		})
	}
	return out
}

// GetRecordedVideos lists the recorded videos for an execution (requires
// ExecuteWorkflowOptions.RequiresVideo to have been set at launch).
func (c *HTTPClient) GetRecordedVideos(ctx context.Context, executionID string) ([]RecordedArtifact, error) {
	resp, err := c.executions.GetExecutionRecordedVideos(ctx, connect.NewRequest(&basapi.GetExecutionArtifactsRequest{ExecutionId: executionID}))
	if err != nil {
		return nil, fmt.Errorf("videos fetch failed: %w", err)
	}
	return toRecordedArtifacts(resp.Msg.GetVideos()), nil
}

// GetRecordedTraces lists the recorded Playwright traces for an execution.
func (c *HTTPClient) GetRecordedTraces(ctx context.Context, executionID string) ([]RecordedArtifact, error) {
	resp, err := c.executions.GetExecutionRecordedTraces(ctx, connect.NewRequest(&basapi.GetExecutionArtifactsRequest{ExecutionId: executionID}))
	if err != nil {
		return nil, fmt.Errorf("traces fetch failed: %w", err)
	}
	return toRecordedArtifacts(resp.Msg.GetTraces()), nil
}

// GetRecordedHar lists the recorded HAR archives for an execution.
func (c *HTTPClient) GetRecordedHar(ctx context.Context, executionID string) ([]RecordedArtifact, error) {
	resp, err := c.executions.GetExecutionRecordedHar(ctx, connect.NewRequest(&basapi.GetExecutionArtifactsRequest{ExecutionId: executionID}))
	if err != nil {
		return nil, fmt.Errorf("har fetch failed: %w", err)
	}
	return toRecordedArtifacts(resp.Msg.GetHarFiles()), nil
}

// StreamAsset downloads an asset by URL without materializing its bytes in
// Test Genie memory. The URL can be absolute or relative to the BAS API.
func (c *HTTPClient) StreamAsset(ctx context.Context, assetURL string, destination io.Writer, maxBytes int64) (int64, error) {
	if destination == nil || maxBytes < 1 {
		return 0, fmt.Errorf("asset destination and positive size limit are required")
	}
	fullURL := assetURL
	if !strings.HasPrefix(assetURL, "http://") && !strings.HasPrefix(assetURL, "https://") {
		// Asset URLs from BAS are absolute paths rooted at the host
		// (e.g. /api/v1/storage/...), so prepend the bare host.
		baseForAssets := c.connectBaseURL
		if strings.HasPrefix(assetURL, "/") {
			fullURL = baseForAssets + assetURL
		} else {
			fullURL = baseForAssets + "/" + assetURL
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return 0, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return 0, fmt.Errorf("asset download failed: status=%s url=%s", resp.Status, fullURL)
	}
	if resp.ContentLength > maxBytes {
		return 0, fmt.Errorf("asset exceeds configured budget: %d > %d bytes", resp.ContentLength, maxBytes)
	}
	written, err := io.Copy(destination, io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return written, err
	}
	if written > maxBytes {
		return written, fmt.Errorf("asset exceeds configured budget: %d > %d bytes", written, maxBytes)
	}
	return written, nil
}

// definitionToProto converts a workflow definition expressed as a JSON
// map into the BAS workflow proto. The map shape is the proto-JSON
// representation a playbook author writes — we round-trip it through
// JSON so protojson can drive the type checking.
func definitionToProto(definition map[string]any) (*basworkflows.WorkflowDefinitionV2, error) {
	if definition == nil {
		return nil, errors.New("workflow definition is nil")
	}
	raw, err := json.Marshal(definition)
	if err != nil {
		return nil, fmt.Errorf("marshal workflow definition: %w", err)
	}
	out := &basworkflows.WorkflowDefinitionV2{}
	if err := flowDefinitionUnmarshal.Unmarshal(raw, out); err != nil {
		return nil, fmt.Errorf("decode workflow definition into proto: %w", err)
	}
	return out, nil
}

// protoIssueToLocal converts a proto WorkflowValidationIssue into the
// local representation used by playbook callers. We keep a local struct
// rather than exposing the proto type so callers do not have to depend
// on the proto enum for severity.
func protoIssueToLocal(issue *basapi.WorkflowValidationIssue) ValidationIssue {
	if issue == nil {
		return ValidationIssue{}
	}
	sev := strings.ToLower(strings.TrimPrefix(issue.GetSeverity().String(), "VALIDATION_SEVERITY_"))
	return ValidationIssue{
		Severity: sev,
		Code:     issue.GetCode(),
		Message:  issue.GetMessage(),
		NodeID:   issue.GetNodeId(),
		NodeType: strings.ToLower(strings.TrimPrefix(issue.GetNodeType().String(), "ACTION_TYPE_")),
		Field:    issue.GetField(),
		Pointer:  issue.GetPointer(),
		Hint:     issue.GetHint(),
	}
}

// anyMapToJsonValueMap wraps each value in a JsonValue proto via the
// structpb bridge — protojson knows how to serialize structpb.Value into
// the JsonValue oneof, so this preserves nested structure faithfully.
func anyMapToJsonValueMap(m map[string]any) map[string]*commonpb.JsonValue {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]*commonpb.JsonValue, len(m))
	for k, v := range m {
		jv := anyToJsonValue(v)
		if jv == nil {
			continue
		}
		out[k] = jv
	}
	return out
}

// anyToJsonValue converts an arbitrary Go value (produced by json.Unmarshal
// or a hand-built map literal) into a commonpb.JsonValue. We handle the
// shapes test-genie actually passes — primitives, maps, slices, nil —
// without pulling in the full BAS typeconv package.
func anyToJsonValue(v any) *commonpb.JsonValue {
	if v == nil {
		return &commonpb.JsonValue{Kind: &commonpb.JsonValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}}
	}
	switch val := v.(type) {
	case bool:
		return &commonpb.JsonValue{Kind: &commonpb.JsonValue_BoolValue{BoolValue: val}}
	case int:
		return &commonpb.JsonValue{Kind: &commonpb.JsonValue_IntValue{IntValue: int64(val)}}
	case int32:
		return &commonpb.JsonValue{Kind: &commonpb.JsonValue_IntValue{IntValue: int64(val)}}
	case int64:
		return &commonpb.JsonValue{Kind: &commonpb.JsonValue_IntValue{IntValue: val}}
	case float32:
		return &commonpb.JsonValue{Kind: &commonpb.JsonValue_DoubleValue{DoubleValue: float64(val)}}
	case float64:
		// json.Unmarshal produces float64 for numeric literals; treat
		// whole-number floats as int so the receiving side sees the
		// expected proto IntValue.
		if val == float64(int64(val)) {
			return &commonpb.JsonValue{Kind: &commonpb.JsonValue_IntValue{IntValue: int64(val)}}
		}
		return &commonpb.JsonValue{Kind: &commonpb.JsonValue_DoubleValue{DoubleValue: val}}
	case string:
		return &commonpb.JsonValue{Kind: &commonpb.JsonValue_StringValue{StringValue: val}}
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return &commonpb.JsonValue{Kind: &commonpb.JsonValue_IntValue{IntValue: i}}
		}
		if f, err := val.Float64(); err == nil {
			return &commonpb.JsonValue{Kind: &commonpb.JsonValue_DoubleValue{DoubleValue: f}}
		}
		return nil
	case []byte:
		return &commonpb.JsonValue{Kind: &commonpb.JsonValue_BytesValue{BytesValue: val}}
	case map[string]any:
		obj := &commonpb.JsonObject{Fields: make(map[string]*commonpb.JsonValue, len(val))}
		for k, item := range val {
			if jv := anyToJsonValue(item); jv != nil {
				obj.Fields[k] = jv
			}
		}
		return &commonpb.JsonValue{Kind: &commonpb.JsonValue_ObjectValue{ObjectValue: obj}}
	case []any:
		list := &commonpb.JsonList{Values: make([]*commonpb.JsonValue, 0, len(val))}
		for _, item := range val {
			if jv := anyToJsonValue(item); jv != nil {
				list.Values = append(list.Values, jv)
			}
		}
		return &commonpb.JsonValue{Kind: &commonpb.JsonValue_ListValue{ListValue: list}}
	default:
		// Fallback: round-trip through JSON to canonicalize.
		raw, err := json.Marshal(val)
		if err != nil {
			return nil
		}
		var tmp any
		if err := json.Unmarshal(raw, &tmp); err != nil {
			return nil
		}
		return anyToJsonValue(tmp)
	}
}
