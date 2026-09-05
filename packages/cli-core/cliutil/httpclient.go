package cliutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var installIdentityForwardingTransport sync.Once

// APIError represents a structured error from the API with rich recovery information.
type APIError struct {
	StatusCode   int
	Message      string
	Code         string
	Category     string
	Details      map[string]interface{}
	Recovery     string
	RecoveryHint string
	AutoFix      *AutoFixInfo
	ManualSteps  []string
	RawResponse  []byte
}

// AutoFixInfo describes an automatic fix command if available.
type AutoFixInfo struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Safe        bool   `json:"safe"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf("api error (%d): %s", e.StatusCode, e.Message)
}

// IsStructured returns true if the error has rich recovery information.
func (e *APIError) IsStructured() bool {
	return e.Code != "" || e.RecoveryHint != "" || e.AutoFix != nil || len(e.ManualSteps) > 0
}

// FormatConcise returns a human-readable error with recovery information.
func (e *APIError) FormatConcise() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Error: %s\n", e.Message))
	if e.Code != "" {
		b.WriteString(fmt.Sprintf("Code: %s\n", e.Code))
	}
	if e.RecoveryHint != "" {
		b.WriteString(fmt.Sprintf("\nRecovery: %s\n", e.RecoveryHint))
	}
	if e.AutoFix != nil && e.AutoFix.Command != "" {
		safe := ""
		if e.AutoFix.Safe {
			safe = " (safe)"
		}
		b.WriteString(fmt.Sprintf("\nAuto-fix%s:\n  %s\n", safe, e.AutoFix.Command))
	}
	if len(e.ManualSteps) > 0 {
		b.WriteString("\nManual steps:\n")
		for i, step := range e.ManualSteps {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, step))
		}
	}
	return b.String()
}

// ParseAPIError parses a structured API error response.
func ParseAPIError(statusCode int, data []byte) *APIError {
	apiErr := &APIError{StatusCode: statusCode, RawResponse: data}

	var parsed struct {
		Error        string                 `json:"error"`
		Code         string                 `json:"code"`
		Category     string                 `json:"category"`
		Details      map[string]interface{} `json:"details"`
		Recovery     string                 `json:"recovery"`
		RecoveryHint string                 `json:"recovery_hint"`
		AutoFix      *AutoFixInfo           `json:"auto_fix"`
		ManualSteps  []string               `json:"manual_steps"`
	}

	if err := json.Unmarshal(data, &parsed); err == nil {
		apiErr.Message = parsed.Error
		apiErr.Code = parsed.Code
		apiErr.Category = parsed.Category
		apiErr.Details = parsed.Details
		apiErr.Recovery = parsed.Recovery
		apiErr.RecoveryHint = parsed.RecoveryHint
		apiErr.AutoFix = parsed.AutoFix
		apiErr.ManualSteps = parsed.ManualSteps
	} else {
		apiErr.Message = strings.TrimSpace(string(data))
	}

	if apiErr.Message == "" {
		apiErr.Message = fmt.Sprintf("HTTP %d", statusCode)
	}
	return apiErr
}

// HTTPClient wraps an http.Client with base URL resolution, token injection,
// and JSON request helpers.
type HTTPClient struct {
	client       *http.Client
	baseOptions  APIBaseOptions
	token        string
	dryRun       bool
	headerSource func() map[string]string
	// invocationHeaderSource is owned by cliapp's per-command preflight. It
	// must remain distinct from an application header source: scenarios may
	// require their own transport contract (for example, runtime attribution)
	// in addition to common invocation provenance.
	invocationHeaderSource func() map[string]string
}

type HTTPClientOptions struct {
	Client      *http.Client
	BaseOptions APIBaseOptions
	Token       string
	Timeout     time.Duration
}

func NewHTTPClient(opts HTTPClientOptions) *HTTPClient {
	installIdentityForwardingTransport.Do(func() {
		http.DefaultTransport = identityForwardingTransport{next: http.DefaultTransport}
	})
	client := opts.Client
	if client == nil {
		timeout := opts.Timeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	} else if opts.Timeout > 0 {
		client.Timeout = opts.Timeout
	}
	return &HTTPClient{
		client:      client,
		baseOptions: opts.BaseOptions,
		token:       opts.Token,
	}
}

// identityForwardingTransport keeps verified Agent Manager identity attached to
// raw HTTP and Connect clients that use http.DefaultTransport. Scenario CLIs
// normally use HTTPClient.ApplyRequestHeaders, but durable streaming clients
// intentionally create their own no-timeout http.Client. Reading the token at
// request time makes both paths carry the same provenance without each
// scenario needing a special case.
type identityForwardingTransport struct{ next http.RoundTripper }

func (t identityForwardingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return t.roundTripper().RoundTrip(req)
	}
	if strings.TrimSpace(req.Header.Get(HeaderAgentIdentityToken)) != "" || strings.TrimSpace(os.Getenv(EnvIdentityToken)) == "" {
		return t.roundTripper().RoundTrip(req)
	}
	clone := req.Clone(req.Context())
	clone.Header = req.Header.Clone()
	if clone.Header == nil {
		clone.Header = make(http.Header)
	}
	clone.Header.Set(HeaderAgentIdentityToken, os.Getenv(EnvIdentityToken))
	return t.roundTripper().RoundTrip(clone)
}

func (t identityForwardingTransport) roundTripper() http.RoundTripper {
	if t.next == nil {
		return http.DefaultTransport
	}
	return t.next
}

func (h *HTTPClient) SetToken(token string) {
	h.token = token
}

// SetDryRun enables or disables dry-run mode. When enabled, requests include
// the X-Dry-Run header so APIs can skip mutations after validation.
func (h *HTTPClient) SetDryRun(enabled bool) {
	h.dryRun = enabled
}

// SetHeaderSource installs a callback invoked once per request to produce
// extra HTTP headers added to every outgoing call. Use this for headers
// whose value is computed lazily (e.g. read from environment variables
// that may be set after client construction). Returning nil or an empty
// map skips extra-header injection. Empty values in the returned map are
// dropped at request time so callers can express "skip this header" via
// an empty string without conditional wrapping.
//
// Repeated calls replace the previous source. Pass nil to clear.
func (h *HTTPClient) SetHeaderSource(fn func() map[string]string) {
	h.headerSource = fn
}

// SetInvocationHeaderSource installs the per-command provenance source used by
// cliapp. It composes with SetHeaderSource instead of replacing application
// headers that are required by a scenario API contract. Repeated calls replace
// only the prior invocation source.
func (h *HTTPClient) SetInvocationHeaderSource(fn func() map[string]string) {
	h.invocationHeaderSource = fn
}

func (h *HTTPClient) SetBaseOptions(opts APIBaseOptions) {
	h.baseOptions = opts
}

func (h *HTTPClient) BaseURL() string {
	return DetermineAPIBase(h.baseOptions)
}

// Timeout returns the configured HTTP timeout, or zero when no client exists.
func (h *HTTPClient) Timeout() time.Duration {
	if h == nil || h.client == nil {
		return 0
	}
	return h.client.Timeout
}

// CloneWithTimeout returns a copy of the client that uses timeout instead of the
// default. Every other field is carried over — base options, token, dry-run, and
// both header sources — so a long-running maintenance call keeps the same
// transport contract and provenance headers as ordinary commands. The
// underlying *http.Client is replaced rather than mutated, because it is shared
// with any other client built from the same options.
func (h *HTTPClient) CloneWithTimeout(timeout time.Duration) *HTTPClient {
	if h == nil || timeout <= 0 {
		return h
	}
	clone := *h
	clone.client = &http.Client{Timeout: timeout}
	if h.client != nil {
		transport := *h.client
		transport.Timeout = timeout
		clone.client = &transport
	}
	return &clone
}

// Do performs an HTTP request with JSON encoding and standard error handling.
func (h *HTTPClient) Do(method, path string, query url.Values, body interface{}) ([]byte, error) {
	return h.DoWithContext(context.Background(), method, path, query, body)
}

// DoWithContext performs an HTTP request with a provided context.
func (h *HTTPClient) DoWithContext(ctx context.Context, method, path string, query url.Values, body interface{}) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	base := strings.TrimSpace(h.BaseURL())
	if base == "" {
		return nil, fmt.Errorf("api base URL is empty; configure an API base or set an API port")
	}
	if parsed, err := url.Parse(base); err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid api base URL %q", base)
	}
	endpoint := strings.TrimRight(base, "/") + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		switch payload := body.(type) {
		case json.RawMessage:
			reader = bytes.NewReader(payload)
		case []byte:
			reader = bytes.NewReader(payload)
		case string:
			reader = strings.NewReader(payload)
		default:
			encoded, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("encode payload: %w", err)
			}
			reader = bytes.NewReader(encoded)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	h.ApplyRequestHeaders(req)

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, ParseAPIError(resp.StatusCode, data)
	}
	return data, nil
}

// ApplyRequestHeaders applies the configured authentication, dry-run, and
// custom headers to req. It lets non-JSON transports reuse the same request
// decoration as DoWithContext without inheriting JSON body handling.
func (h *HTTPClient) ApplyRequestHeaders(req *http.Request) {
	if h == nil || req == nil {
		return
	}
	if h.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.token))
	}
	if h.dryRun {
		req.Header.Set("X-Dry-Run", "true")
	}
	applyHeaderSource(req, h.headerSource)
	applyHeaderSource(req, h.invocationHeaderSource)
}

func applyHeaderSource(req *http.Request, source func() map[string]string) {
	if source == nil {
		return
	}
	for k, v := range source() {
		if v == "" {
			continue
		}
		req.Header.Set(k, v)
	}
}

// ExtractErrorMessage pulls a human-readable error string from a JSON error
// response; falls back to the raw body when parsing fails.
func ExtractErrorMessage(data []byte) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err == nil {
		if errObj, ok := parsed["error"].(map[string]interface{}); ok {
			if msg, ok := errObj["message"].(string); ok {
				return msg
			}
		}
	}
	return strings.TrimSpace(string(data))
}
