package measures

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DefaultExecutePath is the path the serve helper (Registry.Handler) mounts the
// execute endpoint at, relative to the mount prefix. The HTTPExecutor appends it
// to the resolved measures base URL.
const DefaultExecutePath = "/execute"

// maxExecuteResponseBytes caps how much of a measure-execute response the proxy
// reads, so a misbehaving scenario cannot exhaust memory.
const maxExecuteResponseBytes = 4 << 20 // 4 MiB

// HTTPDoer is the minimal HTTP client seam (satisfied by *http.Client). Declared
// here so the executor never hard-binds a transport.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// BaseURLResolver turns an owning scenario id into the live base URL of its
// measures serve endpoint (scheme://host:port + mount prefix). The hosting
// scenario supplies it — in search-hub this wraps the same cross-scenario
// URLResolver the router already uses (never client-computed). The returned URL
// is the prefix the Registry.Handler is mounted at; DefaultExecutePath is
// appended to it.
type BaseURLResolver interface {
	ResolveMeasuresBaseURL(ctx context.Context, scenario string) (baseURL string, err error)
}

// BaseURLResolverFunc adapts a function to BaseURLResolver.
type BaseURLResolverFunc func(ctx context.Context, scenario string) (string, error)

// ResolveMeasuresBaseURL calls the underlying function.
func (f BaseURLResolverFunc) ResolveMeasuresBaseURL(ctx context.Context, scenario string) (string, error) {
	return f(ctx, scenario)
}

// HTTPExecutor is the production execution-proxy: it resolves the owning
// scenario's measures base URL and POSTs a MeasureRequest to <base><path>,
// decoding the uniform MeasureResult. It is the Executor counterpart to
// Registry.Handler — the two ends of the measures serve contract.
type HTTPExecutor struct {
	// Doer performs the HTTP round-trip. Defaults to http.DefaultClient.
	Doer HTTPDoer
	// Resolver maps scenario → measures base URL. Required.
	Resolver BaseURLResolver
	// Path is appended to the resolved base URL. Defaults to DefaultExecutePath.
	Path string
}

// NewHTTPExecutor constructs an HTTPExecutor over a resolver, defaulting the
// client and path.
func NewHTTPExecutor(resolver BaseURLResolver) *HTTPExecutor {
	return &HTTPExecutor{Doer: http.DefaultClient, Resolver: resolver, Path: DefaultExecutePath}
}

// Execute resolves the owning scenario and POSTs the measure request, returning
// the decoded result. The engine only calls this after the gate authorizes
// execution, so a write/destructive measure never reaches here.
func (x *HTTPExecutor) Execute(ctx context.Context, decl MeasureDeclaration, params map[string]string) (MeasureResult, error) {
	if x.Resolver == nil {
		return MeasureResult{}, fmt.Errorf("measures: HTTPExecutor has no resolver")
	}
	doer := x.Doer
	if doer == nil {
		doer = http.DefaultClient
	}
	path := x.Path
	if path == "" {
		path = DefaultExecutePath
	}

	base, err := x.Resolver.ResolveMeasuresBaseURL(ctx, decl.Scenario)
	if err != nil {
		return MeasureResult{}, fmt.Errorf("measures: resolve scenario %q: %w", decl.Scenario, err)
	}

	body, err := json.Marshal(MeasureRequest{Measure: decl.Name, Params: params})
	if err != nil {
		return MeasureResult{}, fmt.Errorf("measures: marshal request: %w", err)
	}

	url := strings.TrimRight(base, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return MeasureResult{}, fmt.Errorf("measures: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := doer.Do(req)
	if err != nil {
		return MeasureResult{}, fmt.Errorf("measures: execute %q: %w", decl.Name, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxExecuteResponseBytes))
	if err != nil {
		return MeasureResult{}, fmt.Errorf("measures: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return MeasureResult{}, fmt.Errorf("measures: scenario %q returned HTTP %d: %s", decl.Scenario, resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out MeasureResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return MeasureResult{}, fmt.Errorf("measures: decode result: %w", err)
	}
	return out, nil
}
