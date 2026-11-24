// Package readinessprofile reads the compiled Experience Manager profile
// through Browser Automation Studio's shared readiness boundary.
package readinessprofile

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/readiness"
	readinessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/readiness/readinessconnect"
)

// Client reads the one BAS-owned profile projection shared by UX evaluators.
type Client struct {
	resolve func(context.Context) (string, error)
	http    *http.Client
}

// NewClient constructs a client using scenario discovery.
func NewClient() *Client {
	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	return NewClientWithResolver(func(ctx context.Context) (string, error) {
		return resolver.ResolveScenarioURLDefault(ctx, "browser-automation-studio")
	}, &http.Client{Timeout: 15 * time.Second})
}

// NewClientWithResolver creates a client with an explicit resolver and HTTP
// client. It is useful to callers with an existing endpoint policy and tests.
func NewClientWithResolver(resolve func(context.Context) (string, error), client *http.Client) *Client {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{resolve: resolve, http: client}
}

// GetReadinessProfile returns the shared typed projection for a scenario and
// optional path. The BAS response remains the provenance boundary; consumers
// no longer own duplicate Connect and JSON decoding adapters.
func (c *Client) GetReadinessProfile(ctx context.Context, scenario, path string) (*Profile, error) {
	if c == nil || c.resolve == nil {
		return nil, fmt.Errorf("readiness profile endpoint resolver is unavailable")
	}
	base, err := c.resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve browser-automation-studio readiness endpoint: %w", err)
	}
	if strings.TrimSpace(base) == "" {
		return nil, fmt.Errorf("resolve browser-automation-studio readiness endpoint: empty base URL")
	}
	httpClient := c.http
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := readinessconnect.NewReadinessServiceClient(httpClient, base).GetReadinessProfile(ctx, connect.NewRequest(&readinessv1.GetReadinessProfileRequest{Scenario: scenario, Path: path}))
	if err != nil {
		return nil, fmt.Errorf("read browser-automation-studio readiness profile: %w", err)
	}
	var profile Profile
	if err := json.Unmarshal([]byte(resp.Msg.GetProfileJson()), &profile); err != nil {
		return nil, fmt.Errorf("decode readiness profile: %w", err)
	}
	profile.Scenario = resp.Msg.GetScenario()
	if profile.Version == "" {
		profile.Version = resp.Msg.GetProfileVersion()
	}
	return &profile, nil
}

// Profile is the compiled Experience Manager readiness projection.
type Profile struct {
	Scenario string `json:"scenario,omitempty"`
	Version  string `json:"version"`
	Pages    []Page `json:"pages"`
}

// Page describes routes and their readiness surfaces.
type Page struct {
	ID              string   `json:"id"`
	Routes          []string `json:"routes"`
	RuntimeRoutes   []string `json:"runtimeRoutes,omitempty"`
	GeometryOptOuts []string `json:"geometryOptOuts,omitempty"`
	Regions         []Region `json:"regions"`
}

// Region describes one required or optional runtime surface.
type Region struct {
	ID        string          `json:"id"`
	Required  bool            `json:"required"`
	Binding   Binding         `json:"binding"`
	Component json.RawMessage `json:"component"`
	Lifecycle Lifecycle       `json:"lifecycle"`
}

// Binding identifies the rendered surface.
type Binding struct {
	TestID   string `json:"testid"`
	Selector string `json:"selector"`
	Note     string `json:"note"`
}

// Lifecycle declares allowed runtime states for a surface.
type Lifecycle struct {
	Kind   string   `json:"kind"`
	States []string `json:"states"`
}
