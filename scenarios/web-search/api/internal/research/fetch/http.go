package fetch

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"web-search/internal/evidence"
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
	// AllowPrivateAddresses is intended only for hermetic tests and explicitly
	// authorized internal research. Production leaves it false.
	AllowPrivateAddresses bool
}

// HTTPValidators are response validators retained from an earlier source
// observation. Supplying them makes a conditional request; a 304 is still a
// new owner observation with its own retrieval timestamp.
type HTTPValidators struct {
	ETag         string
	LastModified string
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
		// Leave Doer nil so Fetch constructs the dialValidated transport. A
		// caller may inject a Doer for hermetic tests, but production requests
		// must recheck DNS at connection time as well as before the request.
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
	if err := validateDestination(ctx, url, f.AllowPrivateAddresses); err != nil {
		return "", err
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
		doer = &http.Client{
			Timeout: f.Timeout,
			Transport: &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return dialValidated(ctx, network, address, f.AllowPrivateAddresses)
			}},
		}
	}
	if client, ok := doer.(*http.Client); ok && !f.AllowPrivateAddresses {
		copy := *client
		copy.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
			return validateDestination(req.Context(), req.URL.String(), false)
		}
		doer = &copy
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
	// Read one byte beyond the cap so an exact-cap body remains valid while an
	// over-cap body becomes an explicit fetch failure. Returning a truncated
	// body as successful evidence would represent incomplete source content as
	// complete and would make the evidence receipt dishonest.
	// Bodies are treated as UTF-8: the dominant real-world encoding. Legacy
	// charset pages degrade to mojibake the extractor's whitespace collapsing
	// tolerates; not worth a charset-detection dependency.
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return "", fmt.Errorf("research: read fetch body: %w", err)
	}
	if int64(len(raw)) > maxBytes {
		return "", fmt.Errorf("research: fetch body exceeds %d-byte limit", maxBytes)
	}
	return internalresearch.ExtractReadableText(string(raw)), nil
}

func validateDestination(ctx context.Context, rawURL string, allowPrivate bool) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("research: fetch destination is not a public http(s) URL")
	}
	if allowPrivate {
		return nil
	}
	host := u.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		if privateAddress(ip) {
			return fmt.Errorf("research: fetch destination resolves to a private address")
		}
		return nil
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("research: resolve fetch destination: %w", err)
	}
	for _, ip := range ips {
		if privateAddress(ip) {
			return fmt.Errorf("research: fetch destination resolves to a private address")
		}
	}
	return nil
}

func privateAddress(ip net.IP) bool {
	_, cgnat, _ := net.ParseCIDR("100.64.0.0/10")
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || ip.IsMulticast() || cgnat.Contains(ip)
}

func dialValidated(ctx context.Context, network, address string, allowPrivate bool) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("research: split fetch destination: %w", err)
	}
	ips := []net.IP{net.ParseIP(host)}
	if ips[0] == nil {
		ips, err = net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("research: resolve fetch destination at dial: %w", err)
		}
	}
	if !allowPrivate {
		for _, ip := range ips {
			if privateAddress(ip) {
				return nil, fmt.Errorf("research: fetch destination resolves to a private address")
			}
		}
	}
	dialer := &net.Dialer{}
	var lastErr error
	for _, ip := range ips {
		conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		lastErr = dialErr
	}
	return nil, lastErr
}

func (f *HTTPFetcher) FetchObservation(ctx context.Context, url string) (evidence.NewObservation, error) {
	return f.FetchObservationWithValidators(ctx, url, HTTPValidators{})
}

// FetchObservationWithValidators performs one conditional HTTP observation.
// A 304 is represented as metadata-only content with FailureCode=not_modified;
// callers can persist it as a distinct validation receipt without pretending
// that a new body was retrieved.
func (f *HTTPFetcher) FetchObservationWithValidators(ctx context.Context, rawURL string, validators HTTPValidators) (evidence.NewObservation, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return evidence.NewObservation{}, fmt.Errorf("research: fetch: empty url")
	}
	if err := validateDestination(ctx, rawURL, f.AllowPrivateAddresses); err != nil {
		return evidence.NewObservation{}, err
	}
	if f.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, f.Timeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return evidence.NewObservation{}, fmt.Errorf("research: build fetch request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,text/plain;q=0.8,*/*;q=0.1")
	if validators.ETag != "" {
		req.Header.Set("If-None-Match", validators.ETag)
	}
	if validators.LastModified != "" {
		req.Header.Set("If-Modified-Since", validators.LastModified)
	}
	redirectURLs := make([]string, 0, 2)
	doer := f.Doer
	if doer == nil {
		doer = &http.Client{Timeout: f.Timeout, Transport: &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialValidated(ctx, network, address, f.AllowPrivateAddresses)
		}}}
	}
	if client, ok := doer.(*http.Client); ok {
		copy := *client
		copy.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
			redirectURLs = append(redirectURLs, req.URL.String())
			if f.AllowPrivateAddresses {
				return nil
			}
			return validateDestination(req.Context(), req.URL.String(), false)
		}
		doer = &copy
	}
	resp, err := doer.Do(req)
	if err != nil {
		return evidence.NewObservation{}, fmt.Errorf("research: fetch request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	finalURL := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	base := evidence.NewObservation{URL: rawURL, FinalURL: finalURL, RedirectURLs: redirectURLs, RetrievedAt: time.Now().UTC(), ETag: resp.Header.Get("ETag"), LastModified: resp.Header.Get("Last-Modified"), ExtractionRevision: "readable-text-v1", Retention: "metadata-and-extracted-content"}
	if resp.StatusCode == http.StatusNotModified {
		base.FailureCode = "not_modified"
		base.Retention = "metadata-only"
		base.Content = []byte{}
		return base, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return evidence.NewObservation{}, fmt.Errorf("research: fetch status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !allowedContentType(ct) {
		return evidence.NewObservation{}, fmt.Errorf("research: fetch content-type %q not readable", ct)
	}
	maxBytes := f.MaxBodyBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBodyBytes
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return evidence.NewObservation{}, fmt.Errorf("research: read fetch body: %w", err)
	}
	if int64(len(raw)) > maxBytes {
		return evidence.NewObservation{}, fmt.Errorf("research: fetch body exceeds %d-byte limit", maxBytes)
	}
	base.Content = []byte(internalresearch.ExtractReadableText(string(raw)))
	return base, nil
}

// Compile-time guarantee the leg satisfies the package seam.
var _ Fetcher = (*HTTPFetcher)(nil)
var _ internalresearch.ObservationFetcher = (*HTTPFetcher)(nil)
