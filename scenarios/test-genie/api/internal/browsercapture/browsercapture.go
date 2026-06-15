package browsercapture

import (
	"context"
	"fmt"
	"time"

	"test-genie/internal/evidence"
	"test-genie/internal/playbooks/config"
	"test-genie/internal/playbooks/execution"
)

// Default capture parameters. They mirror the historical smoke contract.
const (
	// DefaultViewportWidth / DefaultViewportHeight size the browser viewport.
	DefaultViewportWidth  = 1280
	DefaultViewportHeight = 720
)

// DefaultHandshakeTimeout bounds how long the handshake assert waits for the
// iframe-bridge readiness marker before failing.
const DefaultHandshakeTimeout = 15 * time.Second

// WorkflowClient is the narrow slice of the BAS workflow engine the smoke
// capture needs. It is satisfied by *execution.Client (the shared BAS client
// used by playbooks); tests wire FakeWorkflowClient.
//
// seam: WorkflowClient is the BAS-workflow-engine capture seam. Production wires
// execution.NewClient(baseURL) (Connect-RPC over BAS apiconnect); tests wire
// browsercapture.FakeWorkflowClient (fake.go) returning a canned ParsedTimeline.
type WorkflowClient interface {
	// ExecuteWorkflowWithParams starts an ad-hoc workflow and returns its
	// execution id.
	ExecuteWorkflowWithParams(ctx context.Context, definition map[string]any, name, description string, params *execution.ExecutionParams) (string, error)
	// WaitForCompletionWithProgress blocks until the execution finishes.
	WaitForCompletionWithProgress(ctx context.Context, executionID string, callback execution.ProgressCallback) error
	// GetTimeline returns the parsed execution timeline.
	GetTimeline(ctx context.Context, executionID string) (parsedTimeline, error)
	// DownloadAsset fetches an artifact (e.g. the frame screenshot) by URL.
	DownloadAsset(ctx context.Context, assetURL string) ([]byte, error)
}

// parsedTimeline is the timeline shape browsercapture consumes. It is an alias
// for the playbooks execution.ParsedTimeline so the production client satisfies
// the seam directly while the package stays decoupled from the proto types.
type parsedTimeline = *execution.ParsedTimeline

// liveClient adapts an execution.Client to WorkflowClient. execution.Client's
// GetTimeline returns (proto, json, err); the capture only needs the parsed
// timeline, so this adapter parses it.
type liveClient struct {
	inner execution.Client
}

// NewLiveClient wraps a BAS execution client as a WorkflowClient. baseURL is the
// BAS "/api/v1" endpoint; cfg bounds the workflow timeouts.
func NewLiveClient(baseURL string, cfg execution.ClientConfig) WorkflowClient {
	return &liveClient{inner: execution.NewClientWithConfig(baseURL, cfg)}
}

// NewLiveClientFrom wraps an existing BAS HTTP client as a WorkflowClient so the
// single-page smoke capture and the all-pages Capture client can share ONE BAS
// connection.
func NewLiveClientFrom(client *execution.HTTPClient) WorkflowClient {
	return &liveClient{inner: client}
}

func (c *liveClient) ExecuteWorkflowWithParams(ctx context.Context, definition map[string]any, name, description string, params *execution.ExecutionParams) (string, error) {
	return c.inner.ExecuteWorkflowWithParams(ctx, definition, name, description, params)
}

func (c *liveClient) WaitForCompletionWithProgress(ctx context.Context, executionID string, callback execution.ProgressCallback) error {
	return c.inner.WaitForCompletionWithProgress(ctx, executionID, callback)
}

func (c *liveClient) GetTimeline(ctx context.Context, executionID string) (parsedTimeline, error) {
	proto, _, err := c.inner.GetTimeline(ctx, executionID)
	if err != nil {
		return nil, err
	}
	return execution.FromProtoTimeline(proto), nil
}

func (c *liveClient) DownloadAsset(ctx context.Context, assetURL string) ([]byte, error) {
	return c.inner.DownloadAsset(ctx, assetURL)
}

// Request describes one smoke capture against a running scenario UI.
type Request struct {
	// ScenarioURL is the UI URL embedded inside the host iframe shell.
	ScenarioURL string
	// Label optionally names the surface (a page path) for multi-surface runs.
	Label string
	// HandshakeSignals overrides the window-property readiness signals; the
	// defaults are used when empty.
	HandshakeSignals []string
	// HandshakeTimeout bounds the handshake assert; DefaultHandshakeTimeout
	// when zero.
	HandshakeTimeout time.Duration
	// ViewportWidth / ViewportHeight size the viewport; package defaults when
	// zero.
	ViewportWidth  int
	ViewportHeight int
}

// Capturer drives the smoke workflow on the BAS engine and returns
// engine-agnostic evidence ready for evidence.Analyze.
type Capturer struct {
	client WorkflowClient
}

// New returns a Capturer over the given workflow client.
func New(client WorkflowClient) *Capturer {
	return &Capturer{client: client}
}

// Result is the outcome of one smoke capture: the engine-agnostic evidence plus
// the raw screenshot bytes (when a frame screenshot was captured) for artifact
// persistence.
type Result struct {
	Evidence   evidence.Evidence
	Screenshot []byte
}

// Capture runs the smoke workflow for one surface and maps the resulting BAS
// timeline into evidence. The returned evidence carries the handshake outcome
// (from the handshake assert step), console/network observations (from the
// timeline), and the frame screenshot reference; the screenshot bytes are
// downloaded for persistence. A non-nil error means the capture itself could
// not execute (an engine/transport failure) — the caller should surface that as
// a failed, not skipped, smoke result; the resulting evidence still has
// Loaded=false so evidence.Analyze yields a failure verdict.
func (c *Capturer) Capture(ctx context.Context, req Request) (Result, error) {
	def := buildWorkflow(workflowParams{
		ScenarioURL:        req.ScenarioURL,
		HandshakeSignals:   req.HandshakeSignals,
		HandshakeTimeoutMs: handshakeTimeoutMs(req.HandshakeTimeout),
		ViewportWidth:      req.ViewportWidth,
		ViewportHeight:     req.ViewportHeight,
	})

	params := &execution.ExecutionParams{
		Diagnostics: config.DiagnosticsConfig{Console: true, Network: true},
	}

	execID, err := c.client.ExecuteWorkflowWithParams(ctx, def, "ui-smoke", "test-genie UI smoke", params)
	if err != nil {
		return Result{Evidence: notLoaded(req, fmt.Sprintf("failed to start smoke workflow: %v", err))}, err
	}

	// A handshake timeout is an expected smoke *outcome*, not a transport
	// error: the handshake assert fails and the workflow reports failure. We
	// therefore do not treat WaitForCompletion's error as fatal — the timeline
	// is the source of truth for the verdict. Only a missing timeline is fatal.
	_ = c.client.WaitForCompletionWithProgress(ctx, execID, nil)

	timeline, tlErr := c.client.GetTimeline(ctx, execID)
	if tlErr != nil {
		return Result{Evidence: notLoaded(req, fmt.Sprintf("failed to fetch smoke timeline: %v", tlErr))}, tlErr
	}

	ev := timelineToEvidence(req, timeline)

	var screenshot []byte
	if ev.ScreenshotRef != "" {
		if bytes, dErr := c.client.DownloadAsset(ctx, ev.ScreenshotRef); dErr == nil {
			screenshot = bytes
		}
	}

	return Result{Evidence: ev, Screenshot: screenshot}, nil
}

// handshakeTimeoutMs resolves the configured handshake timeout to milliseconds.
func handshakeTimeoutMs(d time.Duration) int64 {
	if d <= 0 {
		return DefaultHandshakeTimeout.Milliseconds()
	}
	return d.Milliseconds()
}

// notLoaded builds evidence for a capture that never executed.
func notLoaded(req Request, reason string) evidence.Evidence {
	return evidence.Evidence{
		URL:       req.ScenarioURL,
		Label:     req.Label,
		Loaded:    false,
		LoadError: reason,
	}
}
