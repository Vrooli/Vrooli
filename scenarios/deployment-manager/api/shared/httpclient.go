package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// HTTPClient defines the interface for making HTTP requests.
// This seam allows tests to substitute real HTTP calls with mocks.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// httpClientKey is the context key for injecting an HTTP client.
type httpClientKey struct{}

// WithHTTPClient returns a new context with the given HTTP client.
func WithHTTPClient(ctx context.Context, client HTTPClient) context.Context {
	return context.WithValue(ctx, httpClientKey{}, client)
}

// GetHTTPClient retrieves the HTTP client from context, falling back to http.DefaultClient.
func GetHTTPClient(ctx context.Context) HTTPClient {
	if client, ok := ctx.Value(httpClientKey{}).(HTTPClient); ok && client != nil {
		return client
	}
	return http.DefaultClient
}

// GetScenarioDependencies fetches the resource dependencies for a scenario from the analyzer.
func GetScenarioDependencies(ctx context.Context, scenario string) ([]string, error) {
	baseURL, err := GetConfigResolver().ResolveAnalyzerURL()
	if err != nil {
		return nil, fmt.Errorf("resolve analyzer URL: %w", err)
	}

	target, err := url.Parse(fmt.Sprintf("%s/api/v1/analyze/%s", baseURL, url.PathEscape(scenario)))
	if err != nil {
		return nil, fmt.Errorf("build analyzer URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	res, err := GetHTTPClient(ctx).Do(req)
	if err != nil {
		return nil, fmt.Errorf("call analyzer: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<15))
		return nil, fmt.Errorf("analyzer returned %d: %s", res.StatusCode, string(body))
	}

	// Parse the response to extract resource names
	var response struct {
		Resources []struct {
			DependencyName string `json:"dependency_name"`
		} `json:"resources"`
	}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode analyzer response: %w", err)
	}

	deps := make([]string, 0, len(response.Resources))
	for _, r := range response.Resources {
		deps = append(deps, r.DependencyName)
	}
	return deps, nil
}
