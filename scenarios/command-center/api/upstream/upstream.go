// Package upstream provides read-only clients for the three data sources
// command-center aggregates: Swarm Manager, Vrooli Core, and LPBS.
package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrNotAvailable is returned when an upstream is not configured or cannot
// be reached. Callers should fall back to gap-mode data.
var ErrNotAvailable = errors.New("upstream not available")

// Client is the shared read interface for every upstream source.
type Client interface {
	// Name returns the source identifier (e.g. "swarm", "vrooli", "lpbs").
	Name() string
	// Fetch performs a GET against the upstream and returns the raw body.
	Fetch(ctx context.Context, path string) (json.RawMessage, error)
}

// baseClient is the shared HTTP plumbing used by every upstream.
type baseClient struct {
	name    string
	baseURL string
	http    *http.Client
	authFn  func(req *http.Request)
}

func newBase(name, baseURL string) *baseClient {
	return &baseClient{
		name:    name,
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *baseClient) Name() string { return c.name }

func (c *baseClient) Fetch(ctx context.Context, path string) (json.RawMessage, error) {
	if c.baseURL == "" {
		return nil, ErrNotAvailable
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.authFn != nil {
		c.authFn(req)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", c.name, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s read body: %w", c.name, err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotAvailable
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s http %d: %s", c.name, resp.StatusCode, truncate(string(body), 200))
	}
	return json.RawMessage(body), nil
}

// NewSwarm returns a client for the Swarm Manager scenario's REST API.
func NewSwarm(baseURL string) Client {
	return newBase("swarm", baseURL)
}

// NewVrooli returns a client for the Vrooli core's REST API.
func NewVrooli(baseURL string) Client {
	return newBase("vrooli", baseURL)
}

// NewLPBS returns a client for the LPBS scenario's admin dashboard API.
// The bearer token is attached to every request when non-empty so that
// /api/v1/admin/dashboard/* is accessible once LPBS ships those handlers.
// On 404 (endpoints not yet implemented) the client returns ErrNotAvailable
// so the caller can fall through to gap-mode data from the registry.
func NewLPBS(baseURL, bearerToken string) Client {
	c := newBase("lpbs", baseURL)
	if bearerToken != "" {
		c.authFn = func(req *http.Request) {
			req.Header.Set("Authorization", "Bearer "+bearerToken)
		}
	}
	return c
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
