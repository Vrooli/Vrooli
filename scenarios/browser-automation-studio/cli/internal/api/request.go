package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"browser-automation-studio/cli/internal/appctx"
)

func Do(ctx *appctx.Context, method, path string, query url.Values, body []byte, headers map[string]string) (int, []byte, error) {
	if ctx == nil || ctx.Core == nil {
		return 0, nil, fmt.Errorf("missing app context")
	}
	base := strings.TrimRight(strings.TrimSpace(ctx.APIRoot()), "/")
	if base == "" {
		return 0, nil, fmt.Errorf("api base URL is empty")
	}

	endpoint := base + path
	if query != nil && len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		return 0, nil, fmt.Errorf("create request: %w", err)
	}

	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if token := strings.TrimSpace(ctx.Token()); token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	timeout := ctx.Core.HTTPClient.Timeout()
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read response: %w", err)
	}

	return resp.StatusCode, data, nil
}
