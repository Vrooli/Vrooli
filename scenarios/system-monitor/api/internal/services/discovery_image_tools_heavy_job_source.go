package services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// ScenarioURLResolver is the shared target-addressing contract needed by this
// data source. api-core owns local peer/registry/CLI discovery and freshness.
type ScenarioURLResolver interface {
	ResolveScenarioURL(ctx context.Context, scenarioSlug, portKey string) (string, error)
}

// DiscoveryImageToolsHeavyJobSource reads image-tools telemetry through the
// canonical scenario resolver so peer restarts and port drift follow the same
// freshness contract as other Go scenario consumers.
type DiscoveryImageToolsHeavyJobSource struct {
	resolver ScenarioURLResolver
	client   *http.Client
}

func NewDiscoveryImageToolsHeavyJobSource(resolver ScenarioURLResolver, client *http.Client) *DiscoveryImageToolsHeavyJobSource {
	if resolver == nil {
		resolver = discovery.DefaultResolver()
	}
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	return &DiscoveryImageToolsHeavyJobSource{resolver: resolver, client: client}
}

func (s *DiscoveryImageToolsHeavyJobSource) GetHeavyJobs(ctx context.Context) (HeavyJobSnapshot, error) {
	baseURL, err := s.resolveBaseURL(ctx)
	if err != nil {
		return HeavyJobSnapshot{}, err
	}
	source, err := NewHTTPHeavyJobSource(baseURL, s.client)
	if err != nil {
		return HeavyJobSnapshot{}, err
	}
	return source.GetHeavyJobs(ctx)
}

func (s *DiscoveryImageToolsHeavyJobSource) resolveBaseURL(ctx context.Context) (string, error) {
	if s == nil || s.resolver == nil {
		return "", fmt.Errorf("image-tools scenario resolver is not configured")
	}
	baseURL, err := s.resolver.ResolveScenarioURL(ctx, "image-tools", "API_PORT")
	if err != nil {
		return "", fmt.Errorf("resolve image-tools API URL: %w", err)
	}
	return baseURL, nil
}
