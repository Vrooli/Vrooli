// Package research is the L2/L3 deep-research domain that sits above the L0/L1
// live-search path.
//
// L2 (l2_pipeline.go) is a synchronous single-pass pipeline: take a query, ask
// the live-search service for the top-N candidate URLs, fetch each page via the
// Fetcher seam (browserless in production), extract readable text, and run a
// single always-cited synthesis over the fetched content. Auto-capture is
// opt-in for L2 — when requested the distilled claims are written to the
// findings store (FINDING_SOURCE_L2).
//
// L3 (l3_agent.go) hands the harder iterative research-and-reconcile loop to an
// agent-manager run and returns a run handle the caller polls.
//
// Every external dependency sits behind an interface seam so the pipeline is
// fully testable with no live network: Fetcher (browserless), Searcher
// (live-search), Synthesizer (LLM), and the agent-manager Client. main.go wires
// the production impls from env; tests inject fakes.
package research

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"web-search/internal/httpc"
)

// DefaultBrowserlessURL is the browserless base URL used when BROWSERLESS_URL is
// unset.
const DefaultBrowserlessURL = "http://localhost:4110"

// defaultFetchTimeout bounds a single page fetch.
const defaultFetchTimeout = 20 * time.Second

// Fetcher is the L2 page-fetch seam: it returns the readable body text of a
// URL. The production impl talks to the browserless resource; tests inject a
// fake to pin behavior with no live browser.
type Fetcher interface {
	// Fetch returns the readable text of url. A non-nil error means the fetch or
	// extraction failed (the pipeline skips that page and continues).
	Fetch(ctx context.Context, url string) (text string, err error)
}

// BrowserlessFetcher is the production Fetcher: it POSTs to
// {BaseURL}/chrome/content, receives the page HTML, and extracts readable body
// text by stripping tags and known boilerplate.
type BrowserlessFetcher struct {
	BaseURL string
	Doer    httpc.Doer
	Timeout time.Duration
}

// NewBrowserlessFetcher constructs a browserless-backed fetcher. An empty
// baseURL falls back to DefaultBrowserlessURL; a nil doer falls back to a
// timeout-bounded *http.Client.
func NewBrowserlessFetcher(baseURL string, doer httpc.Doer) *BrowserlessFetcher {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = DefaultBrowserlessURL
	}
	if doer == nil {
		doer = &http.Client{Timeout: defaultFetchTimeout}
	}
	return &BrowserlessFetcher{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Doer:    doer,
		Timeout: defaultFetchTimeout,
	}
}

// contentRequest is the browserless /chrome/content request body.
type contentRequest struct {
	URL string `json:"url"`
}

// Fetch retrieves url's HTML via browserless and extracts readable text.
func (f *BrowserlessFetcher) Fetch(ctx context.Context, url string) (string, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return "", fmt.Errorf("research: fetch: empty url")
	}
	if f.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, f.Timeout)
		defer cancel()
	}

	payload, err := json.Marshal(contentRequest{URL: url})
	if err != nil {
		return "", fmt.Errorf("research: marshal content request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.BaseURL+"/chrome/content", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("research: build content request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.Doer.Do(req)
	if err != nil {
		return "", fmt.Errorf("research: content request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("research: content status %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("research: read content body: %w", err)
	}
	return ExtractReadableText(string(raw)), nil
}

// Compile-time guarantee that the production fetcher satisfies the seam.
var _ Fetcher = (*BrowserlessFetcher)(nil)
