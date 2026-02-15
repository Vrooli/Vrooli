package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) resolveV1Endpoint(endpointPath string) string {
	endpointPath = strings.TrimSpace(endpointPath)
	if endpointPath == "" {
		return ""
	}
	if !strings.HasPrefix(endpointPath, "/") {
		endpointPath = "/" + endpointPath
	}
	base := strings.TrimRight(strings.TrimSpace(a.core.HTTPClient.BaseURL()), "/")
	if strings.HasSuffix(base, "/api/v1") {
		return endpointPath
	}
	return "/api/v1" + endpointPath
}

func (a *App) getV1(path string, query url.Values) ([]byte, error) {
	return a.core.APIClient.Get(a.resolveV1Endpoint(path), query)
}

func (a *App) requestV1(method, path string, query url.Values, payload any) ([]byte, error) {
	return a.core.APIClient.Request(method, a.resolveV1Endpoint(path), query, payload)
}

func (a *App) requestMultipartV1(method, path string, payload []byte, contentType string) ([]byte, error) {
	base := strings.TrimRight(strings.TrimSpace(a.core.APIClient.BaseURL()), "/")
	if base == "" {
		return nil, fmt.Errorf("api base URL is empty; configure an API base or set an API port")
	}
	endpoint := base + a.resolveV1Endpoint(path)

	req, err := http.NewRequest(method, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if strings.TrimSpace(contentType) != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for key, value := range a.core.APIClient.AuthHeaders() {
		req.Header.Set(key, value)
	}

	timeout := a.core.HTTPClient.Timeout()
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, cliutil.ParseAPIError(resp.StatusCode, body)
	}
	return body, nil
}

func printJSONIfRequested(enabled bool, body []byte) bool {
	if !enabled {
		return false
	}
	cliutil.PrintJSON(body)
	return true
}

func decodeResponse[T any](body []byte) (T, error) {
	var response T
	if err := json.Unmarshal(body, &response); err != nil {
		return response, fmt.Errorf("failed to parse response: %w", err)
	}
	return response, nil
}

func parseJSONArg(args []string) (json.RawMessage, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("JSON payload is required")
	}
	raw := strings.TrimSpace(strings.Join(args, " "))
	if raw == "" {
		return nil, fmt.Errorf("JSON payload is required")
	}

	if strings.HasPrefix(raw, "@") {
		path := strings.TrimSpace(strings.TrimPrefix(raw, "@"))
		if path == "" {
			return nil, fmt.Errorf("JSON file path is required after @")
		}
		content, err := cliutil.ReadFileString(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		raw = strings.TrimSpace(content)
	}

	var parsed json.RawMessage
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return json.RawMessage(raw), nil
}

func printTree[T any](items []T, childFn func(T) []T, lineFn func(T) string, level int) {
	for _, item := range items {
		fmt.Printf("%s%s\n", strings.Repeat("  ", level), lineFn(item))
		children := childFn(item)
		if len(children) > 0 {
			printTree(children, childFn, lineFn, level+1)
		}
	}
}

func cliCommand(parts ...string) string {
	segments := make([]string, 0, len(parts)+1)
	segments = append(segments, appName)
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		segments = append(segments, trimmed)
	}
	return strings.Join(segments, " ")
}

func printSection(title string) {
	fmt.Printf("%s:\n", title)
}

func printCommandListSection(title string, commands []string) {
	filtered := make([]string, 0, len(commands))
	for _, command := range commands {
		if strings.TrimSpace(command) == "" {
			continue
		}
		filtered = append(filtered, command)
	}
	if len(filtered) == 0 {
		return
	}
	fmt.Printf("\n%s:\n", title)
	for _, command := range filtered {
		fmt.Printf("  %s\n", command)
	}
}
