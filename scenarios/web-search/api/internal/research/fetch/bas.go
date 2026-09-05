package fetch

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	captureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture/captureconnect"

	internalresearch "web-search/internal/research"
)

// DefaultBrowserTimeout bounds one browser-leg fetch. A BAS capture is a real
// navigation (2–10s) plus artifact export, so this is deliberately generous.
const DefaultBrowserTimeout = 60 * time.Second

// CaptureClient is the narrow BAS seam: the generated Connect client
// satisfies it in production, tests fake it.
type CaptureClient interface {
	Capture(ctx context.Context, req *connect.Request[capturev1.CaptureRequest]) (*connect.Response[capturev1.CaptureResponse], error)
}

// ClientResolver lazily produces the Capture client. Production resolves
// browser-automation-studio's base URL through scenario discovery on first
// use (BAS may boot after web-search); tests inject a canned client.
type ClientResolver func(ctx context.Context) (CaptureClient, error)

// BASFetcher is the browser escalation leg: one CaptureService.Capture call
// with inline DOM return, then the same readable-text extraction as the HTTP
// leg. No artifact paths are read — inline_dom keeps the contract
// host-independent.
type BASFetcher struct {
	// Resolve produces the Capture client (nil = discovery-based default).
	Resolve ClientResolver
	// Timeout bounds one fetch when > 0 (default DefaultBrowserTimeout).
	Timeout time.Duration

	mu     sync.Mutex
	client CaptureClient
}

// NewBASFetcher builds the production browser leg with discovery-based
// resolution of the BAS Connect endpoint.
func NewBASFetcher() *BASFetcher {
	httpClient := &http.Client{Timeout: DefaultBrowserTimeout}
	return &BASFetcher{
		Resolve: func(ctx context.Context) (CaptureClient, error) {
			baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "browser-automation-studio")
			if err != nil {
				return nil, fmt.Errorf("research: browser-automation-studio not available: %w", err)
			}
			return captureconnect.NewCaptureServiceClient(httpClient, baseURL), nil
		},
		Timeout: DefaultBrowserTimeout,
	}
}

func (f *BASFetcher) captureClient(ctx context.Context) (CaptureClient, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.client != nil {
		return f.client, nil
	}
	if f.Resolve == nil {
		return nil, fmt.Errorf("research: BAS fetcher has no client resolver")
	}
	client, err := f.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	f.client = client
	return client, nil
}

// Fetch implements the research.Fetcher contract for the browser leg.
func (f *BASFetcher) Fetch(ctx context.Context, url string) (string, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return "", fmt.Errorf("research: fetch: empty url")
	}
	timeout := f.Timeout
	if timeout <= 0 {
		timeout = DefaultBrowserTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := f.captureClient(ctx)
	if err != nil {
		return "", err
	}

	resp, err := client.Capture(ctx, connect.NewRequest(&capturev1.CaptureRequest{
		Url:       url,
		Captures:  []capturev1.CaptureType{capturev1.CaptureType_CAPTURE_TYPE_DOM},
		WaitFor:   &capturev1.WaitFor{Spec: &capturev1.WaitFor_Networkidle{Networkidle: true}},
		InlineDom: true,
		Label:     "web-search L2 fetch",
	}))
	if err != nil {
		// Drop the cached client so a BAS restart (new port) re-resolves.
		f.mu.Lock()
		f.client = nil
		f.mu.Unlock()
		return "", fmt.Errorf("research: browser capture: %w", err)
	}

	dom := resp.Msg.GetDomHtml()
	if strings.TrimSpace(dom) == "" {
		return "", fmt.Errorf("research: browser capture returned no DOM for %q", url)
	}
	return internalresearch.ExtractReadableText(dom), nil
}

// Compile-time guarantee the leg satisfies the package seam.
var _ Fetcher = (*BASFetcher)(nil)
