package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unit-health/internal/testquality"

	apiDiscovery "github.com/vrooli/api-core/discovery"
)

// RequirementDeclaration is an owner-provided declaration, not execution proof.
type RequirementDeclaration = testquality.RequirementDeclaration
type RequirementResponsibility = testquality.RequirementResponsibility
type RequirementRegistry = testquality.RequirementRegistry
type RequirementRegistryReader interface {
	Read(context.Context, string) (RequirementRegistry, error)
}

// testGenieRegistryReader consumes current declarations from their owner. It
// never scans a private copy of the registry or consumes cached live statuses.
type testGenieRegistryReader struct {
	Resolver interface {
		ResolveScenarioURLDefault(context.Context, string) (string, error)
	}
	HTTPClient *http.Client
}

func (r testGenieRegistryReader) Read(ctx context.Context, scenario string) (RequirementRegistry, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if scenario == "" {
		return RequirementRegistry{}, fmt.Errorf("scenario required")
	}
	resolver := r.Resolver
	if resolver == nil {
		resolver = apiDiscovery.NewResolver(apiDiscovery.ResolverConfig{})
	}
	base, err := resolver.ResolveScenarioURLDefault(ctx, "test-genie")
	if err != nil {
		return RequirementRegistry{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/scenarios/"+url.PathEscape(scenario)+"/requirements?view=registry", nil)
	if err != nil {
		return RequirementRegistry{}, err
	}
	client := r.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return RequirementRegistry{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return RequirementRegistry{}, fmt.Errorf("requirement registry returned HTTP %d", response.StatusCode)
	}
	const maxBytes = 8 << 20
	data, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return RequirementRegistry{}, err
	}
	if len(data) > maxBytes {
		return RequirementRegistry{}, fmt.Errorf("requirement registry exceeds byte limit")
	}
	var registry RequirementRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return RequirementRegistry{}, err
	}
	if registry.SchemaVersion != "requirement-registry/v1" || registry.Requirements == nil || len(registry.Requirements) > 10000 {
		return RequirementRegistry{}, fmt.Errorf("unsupported or incomplete requirement registry")
	}
	seen := map[string]bool{}
	for _, requirement := range registry.Requirements {
		if requirement.ID == "" || strings.TrimSpace(requirement.ID) != requirement.ID || seen[requirement.ID] {
			return RequirementRegistry{}, fmt.Errorf("invalid or duplicate requirement identity")
		}
		seen[requirement.ID] = true
	}
	return registry, nil
}
