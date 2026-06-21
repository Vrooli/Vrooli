package capture

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	captureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture/captureconnect"
)

const basScenarioID = "browser-automation-studio"

// componentMark is the prefix BAS's perf trace carries for React component
// commits (Tier 1). Its presence in the trace means ⚛ marks rode along.
const componentMark = "⚛"

// BASConnectClient is the production BASClient: it calls Browser Automation
// Studio's CaptureService with CAPTURE_TYPE_PERFORMANCE to produce a CDP trace +
// web-vitals (Tier 0), then inspects the returned trace for ⚛ component marks
// (Tier 1). BAS stays a dumb mechanism — tier meaning is derived here.
//
// A capture that returns no usable trace (no browser available / unavailable
// artifact) yields empty Artifacts with a nil error, which the orchestrator
// treats as a clean skip.
type BASConnectClient struct {
	// Resolve maps the BAS scenario slug → its API base URL; nil uses the
	// discovery resolver.
	Resolve func(ctx context.Context) (string, error)

	// HTTPClient is the Connect transport; nil uses a default client.
	HTTPClient connect.HTTPClient

	// ReadTrace reads a trace artifact's bytes for ⚛ detection; nil reads the
	// server-filesystem path directly (BAS and performance-health are co-hosted).
	ReadTrace func(path string) ([]byte, error)
}

var _ BASClient = (*BASConnectClient)(nil)

// CapturePerf drives a BAS perf capture for url. workflow is reserved for a BAS
// recorded-workflow slug; empty uses BAS's default navigate+settle interaction.
func (c *BASConnectClient) CapturePerf(ctx context.Context, url, workflow string) (Artifacts, error) {
	if strings.TrimSpace(url) == "" {
		return Artifacts{}, errors.New("capture: url is required")
	}

	baseURL, err := c.resolve(ctx)
	if err != nil {
		// BAS unreachable → clean skip (no browser/mechanism available).
		return Artifacts{}, nil
	}

	client := captureconnect.NewCaptureServiceClient(c.httpClient(), baseURL)
	resp, err := client.Capture(ctx, connect.NewRequest(&capturev1.CaptureRequest{
		Url:      url,
		Captures: []capturev1.CaptureType{capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE},
		Label:    "performance-health audit",
	}))
	if err != nil {
		// A transport/exec failure (e.g. no browser in the env) is a clean skip,
		// not a hard error — performance-health degrades gracefully headless.
		return Artifacts{}, nil
	}
	if resp == nil || resp.Msg == nil {
		return Artifacts{}, nil
	}

	return c.artifactsFromResponse(resp.Msg), nil
}

// artifactsFromResponse extracts the trace + web-vitals paths from a BAS
// capture response and decides whether ⚛ component marks were present.
func (c *BASConnectClient) artifactsFromResponse(msg *capturev1.CaptureResponse) Artifacts {
	var art Artifacts
	for _, a := range msg.GetArtifacts() {
		if a.GetType() != capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE {
			continue
		}
		if a.GetMetadata()["unavailable"] == "true" {
			// Driver could not produce a trace (no browser): skip.
			continue
		}
		switch a.GetMetadata()["artifact"] {
		case "cdp-trace":
			art.TraceArtifact = a.GetPath()
		case "web-vitals":
			art.WebVitalsArtifact = a.GetPath()
		default:
			// Older shape: first non-vitals perf artifact is the trace.
			if art.TraceArtifact == "" {
				art.TraceArtifact = a.GetPath()
			}
		}
	}
	if art.TraceArtifact != "" {
		art.HasComponentMarks = c.traceHasComponentMarks(art.TraceArtifact)
	}
	return art
}

// traceHasComponentMarks reports whether the trace contains React ⚛ component
// marks (Tier 1). On any read failure it returns false — capture never fails for
// lack of instrumentation; the result simply downgrades to Tier 0.
func (c *BASConnectClient) traceHasComponentMarks(path string) bool {
	read := c.ReadTrace
	if read == nil {
		read = os.ReadFile
	}
	raw, err := read(path)
	if err != nil {
		return false
	}
	return bytes.Contains(raw, []byte(componentMark))
}

func (c *BASConnectClient) resolve(ctx context.Context) (string, error) {
	if c.Resolve != nil {
		return c.Resolve(ctx)
	}
	return discovery.NewResolver(discovery.ResolverConfig{}).ResolveScenarioURLDefault(ctx, basScenarioID)
}

func (c *BASConnectClient) httpClient() connect.HTTPClient {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 120 * time.Second}
}
