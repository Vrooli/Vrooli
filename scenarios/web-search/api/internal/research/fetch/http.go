package fetch

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"web-search/internal/httpc"
	internalresearch "web-search/internal/research"
)

// DefaultHTTPTimeout bounds one HTTP-leg page fetch.
const DefaultHTTPTimeout = 15 * time.Second

// DefaultMaxBodyBytes caps how much of a page body is read. 2 MiB covers
// real article HTML comfortably; anything larger is media or an attack.
const DefaultMaxBodyBytes = 2 << 20

// userAgent is honest about who is fetching: a metasearch synthesis agent,
// not a spoofed browser.
const userAgent = "vrooli-web-search/1.0 (+https://github.com/Vrooli/Vrooli)"

// HTTPFetcher is the browserless-free default leg: GET the page, gate on a
// text-ish content type, cap the body, extract readable text.
type HTTPFetcher struct {
	// Doer is the outbound HTTP seam (nil = timeout-bounded http.Client).
	Doer httpc.Doer
	// Timeout bounds a single fetch when > 0 (default DefaultHTTPTimeout).
	Timeout time.Duration
	// MaxBodyBytes caps the body read when > 0 (default DefaultMaxBodyBytes).
	MaxBodyBytes int64
}

// NewHTTPFetcher builds the default HTTP leg with compiled defaults; pass
// zero values to keep them.
func NewHTTPFetcher(timeout time.Duration, maxBodyBytes int64) *HTTPFetcher {
	if timeout <= 0 {
		timeout = DefaultHTTPTimeout
	}
	if maxBodyBytes <= 0 {
		maxBodyBytes = DefaultMaxBodyBytes
	}
	return &HTTPFetcher{
		Doer:         &http.Client{Timeout: timeout},
		Timeout:      timeout,
		MaxBodyBytes: maxBodyBytes,
	}
}

// allowedContentType gates what the HTTP leg will read. Everything else
// (PDFs, images, video, JSON APIs) is not synthesizable page text.
func allowedContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		// No/garbled Content-Type: most such pages are still HTML; let the
		// extractor decide.
		return contentType == ""
	}
	switch mediaType {
	case "text/html", "application/xhtml+xml", "text/plain":
		return true
	default:
		return false
	}
}

// Fetch implements the research.Fetcher contract for the HTTP leg.
func (f *HTTPFetcher) Fetch(ctx context.Context, url string) (string, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return "", fmt.Errorf("research: fetch: empty url")
	}
	if f.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, f.Timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("research: build fetch request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,text/plain;q=0.8,*/*;q=0.1")

	doer := f.Doer
	if doer == nil {
		doer = &http.Client{Timeout: DefaultHTTPTimeout}
	}
	resp, err := doer.Do(req)
	if err != nil {
		return "", fmt.Errorf("research: fetch request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("research: fetch status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !allowedContentType(ct) {
		return "", fmt.Errorf("research: fetch content-type %q not readable", ct)
	}

	maxBytes := f.MaxBodyBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBodyBytes
	}
	// Bodies are read as bytes and treated as UTF-8: the dominant real-world
	// encoding. Legacy-charset pages degrade to mojibake the extractor's
	// whitespace collapsing tolerates; not worth a charset-detection dep.
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return "", fmt.Errorf("research: read fetch body: %w", err)
	}
	return internalresearch.ExtractReadableText(string(raw)), nil
}

// Compile-time guarantee the leg satisfies the package seam.
var _ Fetcher = (*HTTPFetcher)(nil)
