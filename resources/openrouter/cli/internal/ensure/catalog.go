package ensure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"resource-openrouter/cli/internal/auth"
	"resource-openrouter/cli/internal/config"
	"strings"
	"time"

	resourceenv "resource-openrouter/cli/internal/env"
)

// liveCatalogChecker probes the live OpenRouter catalog. It hits /models for
// chat-family endpoints and /images/models for the images endpoint, collecting
// every visible model id. It is best-effort: missing key or any HTTP error is
// returned to Run, which degrades to a warning rather than failing.
type liveCatalogChecker struct {
	httpClient *http.Client
	runtime    resourceenv.Runtime
	resolveKey func(ctx context.Context) (auth.Credentials, error)
}

// NewCatalogChecker builds the production checker, or returns nil when no usable
// API key is configured (so ensure skips the live check cleanly).
func NewCatalogChecker(ctx context.Context) CatalogChecker {
	runtime := resourceenv.Load()
	resolver := auth.NewResolver()
	creds, _ := resolver.Resolve(ctx)
	if !creds.Valid() {
		return nil
	}
	return &liveCatalogChecker{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		runtime:    runtime,
		resolveKey: func(ctx context.Context) (auth.Credentials, error) {
			r := auth.NewResolver()
			return r.Resolve(ctx)
		},
	}
}

func (c *liveCatalogChecker) Present(ctx context.Context, endpoints, models []string) (map[string]bool, error) {
	creds, err := c.resolveKey(ctx)
	if err != nil || !creds.Valid() {
		return nil, fmt.Errorf("no usable OpenRouter API key")
	}
	want := map[string]bool{}
	for _, m := range models {
		want[m] = false
	}

	urls := map[string]struct{}{}
	for _, ep := range endpoints {
		if ep == "images" {
			urls[config.ImagesModelsEndpoint(c.runtime.APIBaseURL)] = struct{}{}
		} else {
			urls[config.ModelsEndpoint(c.runtime.APIBaseURL)] = struct{}{}
		}
	}
	for url := range urls {
		ids, err := c.fetchIDs(ctx, url, creds.APIKey)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			if _, tracked := want[id]; tracked {
				want[id] = true
			}
		}
	}
	return want, nil
}

func (c *liveCatalogChecker) fetchIDs(ctx context.Context, url, apiKey string) ([]string, error) {
	if strings.TrimSpace(url) == "" {
		return nil, fmt.Errorf("empty catalog endpoint")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("catalog fetch %s returned %d", url, resp.StatusCode)
	}
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode catalog %s: %w", url, err)
	}
	ids := make([]string, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		ids = append(ids, strings.TrimSpace(m.ID))
	}
	return ids, nil
}
