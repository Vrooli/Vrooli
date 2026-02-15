package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

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

func requireArgs(args []string, min int, usage string) error {
	if len(args) < min {
		return fmt.Errorf("usage: %s", usage)
	}
	return nil
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
