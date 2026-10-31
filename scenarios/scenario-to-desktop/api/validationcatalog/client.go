package validationcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
)

type URLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

type discoveryResolver struct{}

func (discoveryResolver) ResolveScenarioURLDefault(ctx context.Context, scenario string) (string, error) {
	return discovery.ResolveScenarioURLDefault(ctx, scenario)
}

type Client struct {
	resolver URLResolver
	http     *http.Client
}

func NewWorkflowHealthResolver() *Client {
	return NewClient(discoveryResolver{}, &http.Client{Timeout: 10 * time.Second})
}

func NewClient(resolver URLResolver, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{resolver: resolver, http: httpClient}
}

type response struct {
	Journeys []validationmatrix.JourneySelection `json:"journeys"`
}

func (c *Client) Resolve(ctx context.Context, scenario string) (validationmatrix.CatalogSnapshot, error) {
	if c == nil || c.resolver == nil || c.http == nil {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("workflow validation catalog client is unavailable")
	}
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("scenario is required")
	}
	base, err := c.resolver.ResolveScenarioURLDefault(ctx, "workflow-health")
	if err != nil {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("resolve workflow-health URL: %w", err)
	}
	endpoint := strings.TrimRight(base, "/") + "/api/v1/validation/catalog?scenario=" + url.QueryEscape(scenario)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("build catalog request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("read workflow-health catalog: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("workflow-health catalog returned HTTP %d", resp.StatusCode)
	}
	var payload response
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("decode workflow-health catalog: %w", err)
	}
	return validationmatrix.CatalogSnapshot{Journeys: payload.Journeys}, nil
}
