package cliutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPClient wraps an http.Client with base URL resolution, token injection,
// and JSON request helpers.
type HTTPClient struct {
	client      *http.Client
	baseOptions APIBaseOptions
	token       string
}

type HTTPClientOptions struct {
	Client      *http.Client
	BaseOptions APIBaseOptions
	Token       string
}

func NewHTTPClient(opts HTTPClientOptions) *HTTPClient {
	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
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

func (h *HTTPClient) SetBaseOptions(opts APIBaseOptions) {
	h.baseOptions = opts
}

func (h *HTTPClient) BaseURL() string {
	return DetermineAPIBase(h.baseOptions)
}

// Do performs an HTTP request with JSON encoding and standard error handling.
func (h *HTTPClient) Do(method, path string, query url.Values, body interface{}) ([]byte, error) {
	base := strings.TrimSpace(h.BaseURL())
	if base == "" {
		return nil, fmt.Errorf("api base URL is empty; configure an API base or set an API port")
	}
	if parsed, err := url.Parse(base); err != nil || parsed.Scheme == "" {
		return nil, fmt.Errorf("invalid api base URL %q", base)
	}
	endpoint := strings.TrimRight(base, "/") + path
	if query != nil && len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode payload: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if h.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.token))
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
		return nil, fmt.Errorf("api error (%d): %s", resp.StatusCode, ExtractErrorMessage(data))
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
