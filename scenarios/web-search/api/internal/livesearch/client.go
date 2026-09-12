// Package livesearch is the L0 live web-search + L1 snippet-synthesis domain.
//
// The read path is: governor (budget) -> cache (TTL) -> SearXNG client (L0
// meta-search) -> normalization -> optional synthesis (L1). Every external
// dependency sits behind an interface seam so the service is fully testable
// with no live network: SearxngClient, Cache (over a clock seam), Governor
// (over a clock seam), and Synthesizer. main.go wires the production impls
// from env; tests inject fakes.
//
// Synthesis is strictly additive: raw L0 results are never blocked by a
// synthesis failure, and synthesis is off unless the request asks for it.
package livesearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"web-search/internal/httpc"
)

// DefaultSearxngURL is the SearXNG base URL used when SEARXNG_URL is unset.
const DefaultSearxngURL = "http://localhost:8280"

// defaultClientTimeout bounds a single SearXNG round-trip.
const defaultClientTimeout = 10 * time.Second

// RawResult mirrors one element of the SearXNG JSON "results" array. Only the
// fields the domain consumes are decoded; unknown fields are ignored.
type RawResult struct {
	URL      string  `json:"url"`
	Title    string  `json:"title"`
	Content  string  `json:"content"`
	Engine   string  `json:"engine"`
	Score    float64 `json:"score"`
	Category string  `json:"category"`
}

// EngineIssue describes one upstream engine SearXNG reported unresponsive for
// a query (suspended, CAPTCHA'd, parse error, …). Surfacing these is what
// turns "the results look mysteriously bad" into "4 of 6 engines are down".
type EngineIssue struct {
	Engine string
	Reason string
}

// SearchPage is one SearXNG query's payload: the raw results plus the
// per-query engine-degradation signal that rides the same envelope.
type SearchPage struct {
	Results []RawResult
	// UnresponsiveEngines lists engines that did not contribute to THIS
	// query. Non-empty means the result set may be partial.
	UnresponsiveEngines []EngineIssue
}

// searxngEnvelope is the top-level SearXNG JSON response shape.
type searxngEnvelope struct {
	Results []RawResult `json:"results"`
	// Entries are [engine, reason] pairs (a third "suspended" element may be
	// present on some versions).
	UnresponsiveEngines [][]json.RawMessage `json:"unresponsive_engines"`
}

// SearxngClient is the L0 meta-search seam. The production impl hits SearXNG
// over HTTP; tests inject a fake to pin request shape and stub results.
type SearxngClient interface {
	// Search returns up to limit raw results for query plus the engine
	// degradation reported alongside them. A non-nil error means the upstream
	// call failed (the service surfaces this as degraded).
	Search(ctx context.Context, query string, limit int) (SearchPage, error)
}

// HTTPSearxngClient is the production SearxngClient: GET {BaseURL}/search with
// format=json over the injected httpc.Doer seam.
type HTTPSearxngClient struct {
	BaseURL string
	Doer    httpc.Doer
	Timeout time.Duration
}

// NewHTTPSearxngClient constructs an HTTP SearXNG client. An empty baseURL
// falls back to DefaultSearxngURL; a nil doer falls back to a timeout-bounded
// *http.Client.
func NewHTTPSearxngClient(baseURL string, doer httpc.Doer) *HTTPSearxngClient {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = DefaultSearxngURL
	}
	if doer == nil {
		doer = &http.Client{Timeout: defaultClientTimeout}
	}
	return &HTTPSearxngClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Doer:    doer,
		Timeout: defaultClientTimeout,
	}
}

// Search performs the live SearXNG query and decodes the JSON envelope.
func (c *HTTPSearxngClient) Search(ctx context.Context, query string, limit int) (SearchPage, error) {
	if c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}

	q := url.Values{}
	q.Set("q", query)
	q.Set("format", "json")
	endpoint := c.BaseURL + "/search?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return SearchPage{}, fmt.Errorf("livesearch: build searxng request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.Doer.Do(req)
	if err != nil {
		return SearchPage{}, fmt.Errorf("livesearch: searxng request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return SearchPage{}, fmt.Errorf("livesearch: searxng status %s", strconv.Itoa(resp.StatusCode))
	}

	var env searxngEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return SearchPage{}, fmt.Errorf("livesearch: decode searxng response: %w", err)
	}

	results := env.Results
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return SearchPage{
		Results:             results,
		UnresponsiveEngines: decodeUnresponsiveEngines(env.UnresponsiveEngines),
	}, nil
}

// decodeUnresponsiveEngines maps the loose [engine, reason] tuples onto typed
// issues, tolerating malformed entries.
func decodeUnresponsiveEngines(entries [][]json.RawMessage) []EngineIssue {
	if len(entries) == 0 {
		return nil
	}
	issues := make([]EngineIssue, 0, len(entries))
	for _, entry := range entries {
		issue := EngineIssue{}
		if len(entry) > 0 {
			_ = json.Unmarshal(entry[0], &issue.Engine)
		}
		if len(entry) > 1 {
			_ = json.Unmarshal(entry[1], &issue.Reason)
		}
		if issue.Engine != "" {
			issues = append(issues, issue)
		}
	}
	if len(issues) == 0 {
		return nil
	}
	return issues
}

// Compile-time guarantee that the production client satisfies the seam.
var _ SearxngClient = (*HTTPSearxngClient)(nil)
