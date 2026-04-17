package cliutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

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
	client      *http.Client
	baseOptions APIBaseOptions
	token       string
	dryRun      bool
}

type HTTPClientOptions struct {
	Client      *http.Client
	BaseOptions APIBaseOptions
	Token       string
	Timeout     time.Duration
}

func NewHTTPClient(opts HTTPClientOptions) *HTTPClient {
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

func (h *HTTPClient) SetToken(token string) {
	h.token = token
}

// SetDryRun enables or disables dry-run mode. When enabled, requests include
// the X-Dry-Run header so APIs can skip mutations after validation.
func (h *HTTPClient) SetDryRun(enabled bool) {
	h.dryRun = enabled
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
	if h.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.token))
	}
	if h.dryRun {
		req.Header.Set("X-Dry-Run", "true")
	}

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
