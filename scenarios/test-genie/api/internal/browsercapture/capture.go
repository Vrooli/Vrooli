package browsercapture

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"test-genie/internal/evidence"
	"test-genie/internal/playbooks/execution"

	capturepb "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	"github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture/captureconnect"
)

// CaptureClient is the narrow slice of BAS CaptureService the all-pages visual
// capture needs. It is satisfied by the BAS captureconnect client; tests wire
// FakeCaptureClient.
//
// seam: CaptureClient is the BAS single-location capture seam. Production wires
// browsercapture.NewLiveCaptureClient (sharing the workflow client's connection
// via execution.HTTPClient.CaptureServiceClient); tests wire
// browsercapture.FakeCaptureClient (capture_fake.go) returning canned responses.
type CaptureClient interface {
	// Capture loads one URL and emits every requested artifact from one session.
	Capture(ctx context.Context, req *capturepb.CaptureRequest) (*capturepb.CaptureResponse, error)
}

// liveCaptureClient adapts a captureconnect client to CaptureClient.
type liveCaptureClient struct {
	inner captureconnect.CaptureServiceClient
}

// NewLiveCaptureClient builds a CaptureClient that shares the workflow client's
// HTTP connection and Connect base URL — no second BAS client is created.
func NewLiveCaptureClient(workflow *execution.HTTPClient) CaptureClient {
	return &liveCaptureClient{inner: workflow.CaptureServiceClient()}
}

func (c *liveCaptureClient) Capture(ctx context.Context, req *capturepb.CaptureRequest) (*capturepb.CaptureResponse, error) {
	resp, err := c.inner.Capture(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// PageRequest describes one page visual capture.
type PageRequest struct {
	// ScenarioSlug is the scenario whose UI is captured. Combined with Path it
	// forms the cross-scenario URL shorthand "scenario=<slug>,path=<path>".
	ScenarioSlug string
	// Path is the route to capture (e.g. "/", "/backlog").
	Path string
	// Label names the surface in the artifact bundle (the page label).
	Label string
	// IncludeVideo requests a VIDEO artifact in addition to screenshot/console/
	// network (set under the "full" capture profile).
	IncludeVideo bool
}

// PageResult is the outcome of one page visual capture: the analyzed evidence
// plus the BAS server-filesystem artifact paths (screenshot, optional video).
// On the Tier-1 single-host stack these paths are directly readable; the smoke
// writer copies them into the run artifact tree. (Multi-host artifact transport
// is an explicit non-goal — see plan §5.)
type PageResult struct {
	Evidence       evidence.Evidence
	ScreenshotPath string
	VideoPath      string
}

// MultiCapturer issues one CaptureService.Capture per page for plain multi-page
// visual screenshots (no host-iframe handshake — that path stays on the workflow
// engine). It maps each response into engine-agnostic evidence for the shared
// analyzer; the caller persists the artifact paths it returns.
type MultiCapturer struct {
	client CaptureClient
}

// NewMultiCapturer builds a MultiCapturer over the given capture client.
func NewMultiCapturer(client CaptureClient) *MultiCapturer {
	return &MultiCapturer{client: client}
}

// CapturePage captures one page and maps the response into evidence. A non-nil
// error means the capture call itself failed (transport/engine); the returned
// evidence still has Loaded=false so evidence.Analyze yields a failure verdict.
func (m *MultiCapturer) CapturePage(ctx context.Context, req PageRequest) (PageResult, error) {
	captures := []capturepb.CaptureType{
		capturepb.CaptureType_CAPTURE_TYPE_SCREENSHOT,
		capturepb.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS,
		capturepb.CaptureType_CAPTURE_TYPE_NETWORK,
	}
	if req.IncludeVideo {
		captures = append(captures, capturepb.CaptureType_CAPTURE_TYPE_VIDEO)
	}

	url := fmt.Sprintf("scenario=%s,path=%s", req.ScenarioSlug, req.Path)
	pbReq := &capturepb.CaptureRequest{
		Url:        url,
		Captures:   captures,
		Dimensions: &capturepb.Dimensions{Preset: capturepb.DimensionsPreset_DIMENSIONS_PRESET_DESKTOP},
		WaitFor:    &capturepb.WaitFor{Spec: &capturepb.WaitFor_Networkidle{Networkidle: true}},
		Label:      labelOrPath(req),
	}

	resp, err := m.client.Capture(ctx, pbReq)
	if err != nil {
		return PageResult{Evidence: notLoadedPage(req, fmt.Sprintf("capture failed: %v", err))}, err
	}
	return captureToResult(req, resp), nil
}

// captureToResult maps a CaptureResponse into a PageResult. A successful capture
// is treated as a loaded surface; console errors and network failures are
// extracted from artifact metadata so the shared analyzer can judge them.
//
// Capture does not embed the page inside the iframe-bridge host shell, so there
// is no handshake to evaluate; the evidence's Handshake is marked Signaled so the
// analyzer's handshake gate (which is a smoke-specific concern) does not fail a
// plain visual capture. The visual capture's verdict therefore rests on
// load success + network failures + page errors.
func captureToResult(req PageRequest, resp *capturepb.CaptureResponse) PageResult {
	ev := evidence.Evidence{
		URL:    req.Path,
		Label:  labelOrPath(req),
		Loaded: true,
		// Plain visual capture has no host-iframe handshake; mark it satisfied so
		// the smoke-specific handshake gate does not apply here.
		Handshake: evidence.Handshake{Signaled: true},
	}

	res := PageResult{}
	for _, art := range resp.GetArtifacts() {
		switch art.GetType() {
		case capturepb.CaptureType_CAPTURE_TYPE_SCREENSHOT:
			ev.ScreenshotRef = art.GetPath()
			res.ScreenshotPath = art.GetPath()
		case capturepb.CaptureType_CAPTURE_TYPE_NETWORK:
			ev.Network = networkFailuresFromMetadata(art.GetMetadata())
		case capturepb.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS:
			ev.Console = consoleErrorsFromMetadata(art.GetMetadata())
		case capturepb.CaptureType_CAPTURE_TYPE_VIDEO:
			res.VideoPath = art.GetPath()
		}
	}

	res.Evidence = ev
	return res
}

// networkFailuresFromMetadata reads the failed-request count BAS reports in the
// NETWORK artifact metadata ("failure_count") and synthesizes that many opaque
// failure entries so the analyzer's network-failure gate fires. BAS reports
// counts, not per-request detail, in the capture metadata; the count is the
// signal the metadata-level visual diff needs.
func networkFailuresFromMetadata(meta map[string]string) []evidence.NetworkEntry {
	n := intMeta(meta, "failure_count")
	if n <= 0 {
		return nil
	}
	out := make([]evidence.NetworkEntry, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, evidence.NetworkEntry{ErrorText: "network request failed"})
	}
	return out
}

// consoleErrorsFromMetadata reads the console error count BAS reports in the
// CONSOLE_LOGS artifact metadata ("error_count") and synthesizes that many
// error entries so the analyzer's console-error count is accurate (console
// errors are counted, not fatal).
func consoleErrorsFromMetadata(meta map[string]string) []evidence.ConsoleEntry {
	n := intMeta(meta, "error_count")
	if n <= 0 {
		return nil
	}
	out := make([]evidence.ConsoleEntry, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, evidence.ConsoleEntry{Level: "error", Message: "console error"})
	}
	return out
}

func intMeta(meta map[string]string, key string) int {
	if meta == nil {
		return 0
	}
	v, ok := meta[key]
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0
	}
	return n
}

func labelOrPath(req PageRequest) string {
	if strings.TrimSpace(req.Label) != "" {
		return req.Label
	}
	return req.Path
}

func notLoadedPage(req PageRequest, reason string) evidence.Evidence {
	return evidence.Evidence{
		URL:       req.Path,
		Label:     labelOrPath(req),
		Loaded:    false,
		LoadError: reason,
	}
}
