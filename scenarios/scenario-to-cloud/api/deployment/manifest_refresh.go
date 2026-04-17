package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"scenario-to-cloud/bundle"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/stringutil"
	"scenario-to-cloud/secrets"
	"sort"
	"time"
)

// ManifestRefresher regenerates manifest data from current scenario state.
// Used when ForceBundleBuild=true to ensure the deployed code reflects current state.
type ManifestRefresher interface {
	RefreshManifest(ctx context.Context, base domain.CloudManifest) (domain.CloudManifest, error)
}

// ManifestRefreshResult contains the refreshed manifest and metadata about changes.
type ManifestRefreshResult struct {
	Manifest            domain.CloudManifest
	DependenciesChanged bool
	PortsChanged        bool
	Source              string // "analyzer" or "service.json"
}

// manifestRefresher implements ManifestRefresher using scenario-dependency-analyzer
// with fallback to service.json.
type manifestRefresher struct {
	secretsFetcher secrets.Fetcher
	depsFetcher    DependenciesFetcher
	portsFetcher   PortsFetcher
	logger         func(msg string, fields map[string]interface{})
}

// DependenciesFetcher fetches scenario dependencies.
type DependenciesFetcher interface {
	FetchDependencies(ctx context.Context, scenarioID string) (resources, scenarios []string, source string, err error)
}

// PortsFetcher fetches scenario ports from service.json.
type PortsFetcher interface {
	FetchPorts(ctx context.Context, scenarioID string) (map[string]int, error)
}

// ManifestRefresherConfig holds configuration for creating a ManifestRefresher.
type ManifestRefresherConfig struct {
	SecretsFetcher secrets.Fetcher
	DepsFetcher    DependenciesFetcher
	PortsFetcher   PortsFetcher
	Logger         func(msg string, fields map[string]interface{})
}

// NewManifestRefresher creates a new ManifestRefresher with the given dependencies.
func NewManifestRefresher(cfg ManifestRefresherConfig) ManifestRefresher {
	return &manifestRefresher{
		secretsFetcher: cfg.SecretsFetcher,
		depsFetcher:    cfg.DepsFetcher,
		portsFetcher:   cfg.PortsFetcher,
		logger:         cfg.Logger,
	}
}

// RefreshManifest regenerates the manifest from current scenario state.
// It re-fetches dependencies, ports, and updates bundle inclusions.
// Target, edge, and secrets configuration are preserved from the base manifest.
func (r *manifestRefresher) RefreshManifest(ctx context.Context, base domain.CloudManifest) (domain.CloudManifest, error) {
	scenarioID := base.Scenario.ID
	if scenarioID == "" {
		return base, fmt.Errorf("manifest has no scenario ID")
	}

	// Start with a copy of the base manifest
	refreshed := base

	// Re-fetch dependencies from analyzer or service.json
	resources, scenarios, source, err := r.depsFetcher.FetchDependencies(ctx, scenarioID)
	if err != nil {
		r.log("failed to fetch dependencies, keeping original", map[string]interface{}{
			"scenario_id": scenarioID,
			"error":       err.Error(),
		})
		// Continue with original dependencies rather than failing
	} else {
		r.log("refreshed dependencies", map[string]interface{}{
			"scenario_id": scenarioID,
			"source":      source,
			"resources":   resources,
			"scenarios":   scenarios,
		})

		// Update dependencies - always include the target scenario itself
		// The validator requires dependencies.scenarios to include scenario.id
		refreshed.Dependencies.Resources = resources
		refreshed.Dependencies.Scenarios = stringutil.SortedUnique(append(scenarios, scenarioID))
		refreshed.Dependencies.Analyzer.GeneratedAt = time.Now().UTC().Format(time.RFC3339)

		// Update bundle inclusions - always include the target scenario itself
		// The validator requires bundle.scenarios to include scenario.id
		refreshed.Bundle.Resources = resources
		refreshed.Bundle.Scenarios = stringutil.SortedUnique(append(scenarios, scenarioID))
	}

	// Re-fetch ports from service.json
	ports, err := r.portsFetcher.FetchPorts(ctx, scenarioID)
	if err != nil {
		r.log("failed to fetch ports, keeping original", map[string]interface{}{
			"scenario_id": scenarioID,
			"error":       err.Error(),
		})
		// Continue with original ports rather than failing
	} else {
		r.log("refreshed ports", map[string]interface{}{
			"scenario_id": scenarioID,
			"ports":       ports,
		})
		refreshed.Ports = ports
	}

	// Clear secrets so they get re-fetched during deployment
	// This ensures any new secrets from updated dependencies are included
	refreshed.Secrets = nil

	return refreshed, nil
}

func (r *manifestRefresher) log(msg string, fields map[string]interface{}) {
	if r.logger != nil {
		r.logger(msg, fields)
	}
}

// DefaultDependenciesFetcher implements DependenciesFetcher using analyzer with service.json fallback.
type DefaultDependenciesFetcher struct {
	AnalyzerFetcher    func(ctx context.Context, scenarioID string) (resources, scenarios []string, err error)
	ServiceJSONFetcher func(scenarioID string) (resources, scenarios []string, err error)
}

// FetchDependencies fetches dependencies from analyzer with service.json fallback.
func (f *DefaultDependenciesFetcher) FetchDependencies(ctx context.Context, scenarioID string) (resources, scenarios []string, source string, err error) {
	// Try analyzer first
	if f.AnalyzerFetcher != nil {
		resources, scenarios, err = f.AnalyzerFetcher(ctx, scenarioID)
		if err == nil {
			// Merge with service.json to ensure declared dependencies aren't dropped
			if f.ServiceJSONFetcher != nil {
				sjResources, sjScenarios, sjErr := f.ServiceJSONFetcher(scenarioID)
				if sjErr == nil {
					resources = stringutil.SortedUnique(append(resources, sjResources...))
					scenarios = stringutil.SortedUnique(append(scenarios, sjScenarios...))
				}
			}
			return resources, scenarios, "analyzer", nil
		}
	}

	// Fallback to service.json
	if f.ServiceJSONFetcher != nil {
		resources, scenarios, err = f.ServiceJSONFetcher(scenarioID)
		if err != nil {
			return nil, nil, "", fmt.Errorf("both analyzer and service.json failed: %w", err)
		}
		return resources, scenarios, "service.json", nil
	}

	return nil, nil, "", fmt.Errorf("no dependency fetcher available")
}

// DefaultPortsFetcher implements PortsFetcher using service.json.
type DefaultPortsFetcher struct{}

// FetchPorts reads ports from the scenario's service.json file.
func (f *DefaultPortsFetcher) FetchPorts(ctx context.Context, scenarioID string) (map[string]int, error) {
	repoRoot, err := bundle.FindRepoRootFromCWD()
	if err != nil {
		return nil, fmt.Errorf("find repo root: %w", err)
	}

	serviceJSONPath, err := bundle.ResolveScenarioFile(repoRoot, scenarioID, "service")
	if err != nil {
		return nil, fmt.Errorf("resolve service.json path: %w", err)
	}
	data, err := os.ReadFile(serviceJSONPath)
	if err != nil {
		return nil, fmt.Errorf("read service.json: %w", err)
	}

	var svc struct {
		Ports map[string]struct {
			Port int `json:"port"`
		} `json:"ports"`
	}
	if err := json.Unmarshal(data, &svc); err != nil {
		return nil, fmt.Errorf("parse service.json: %w", err)
	}

	// Extract port numbers
	ports := make(map[string]int, len(svc.Ports))
	for name, cfg := range svc.Ports {
		if cfg.Port > 0 {
			ports[name] = cfg.Port
		}
	}

	return ports, nil
}

// ServiceJSONDependenciesFetcher extracts dependencies from service.json.
func ServiceJSONDependenciesFetcher(scenarioID string) (resources, scenarios []string, err error) {
	repoRoot, err := bundle.FindRepoRootFromCWD()
	if err != nil {
		return nil, nil, err
	}

	serviceJSONPath, err := bundle.ResolveScenarioFile(repoRoot, scenarioID, "service")
	if err != nil {
		return nil, nil, err
	}
	data, err := os.ReadFile(serviceJSONPath)
	if err != nil {
		return nil, nil, err
	}

	var svc struct {
		Dependencies struct {
			Resources map[string]struct {
				Enabled  bool `json:"enabled"`
				Required bool `json:"required"`
			} `json:"resources"`
			Scenarios map[string]struct {
				Enabled  bool `json:"enabled"`
				Required bool `json:"required"`
			} `json:"scenarios"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &svc); err != nil {
		return nil, nil, fmt.Errorf("parse service.json: %w", err)
	}

	// Extract enabled/required resources
	for name, dep := range svc.Dependencies.Resources {
		if dep.Enabled || dep.Required {
			resources = append(resources, name)
		}
	}
	sort.Strings(resources)

	// Extract enabled/required scenarios
	for name, dep := range svc.Dependencies.Scenarios {
		if dep.Enabled || dep.Required {
			scenarios = append(scenarios, name)
		}
	}
	sort.Strings(scenarios)

	return resources, scenarios, nil
}
