package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"scenario-to-cloud/bundle"
	"scenario-to-cloud/closure"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/secrets"
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

// manifestRefresher implements ManifestRefresher. The dependency and bundle
// sections are the closure's projection and carry its digest.
type manifestRefresher struct {
	secretsFetcher secrets.Fetcher
	closureSource  ClosureSource
	portsFetcher   PortsFetcher
	logger         func(msg string, fields map[string]interface{})
}

// ClosureSource derives the deployment closure for a scenario. A failure is
// typed (closure_unavailable, closure_cycle, closure_conflict) and stops the
// refresh: a manifest must never be rebuilt from a stale dependency snapshot
// when the closure cannot be derived.
type ClosureSource interface {
	Resolve(ctx context.Context, scenarioID, environment string) (domain.Closure, error)
}

// PortsFetcher fetches scenario ports from service.json.
type PortsFetcher interface {
	FetchPorts(ctx context.Context, scenarioID string) (map[string]int, error)
}

// ManifestRefresherConfig holds configuration for creating a ManifestRefresher.
type ManifestRefresherConfig struct {
	SecretsFetcher secrets.Fetcher
	Closure        ClosureSource
	PortsFetcher   PortsFetcher
	Logger         func(msg string, fields map[string]interface{})
}

// NewManifestRefresher creates a new ManifestRefresher with the given dependencies.
func NewManifestRefresher(cfg ManifestRefresherConfig) ManifestRefresher {
	return &manifestRefresher{
		secretsFetcher: cfg.SecretsFetcher,
		closureSource:  cfg.Closure,
		portsFetcher:   cfg.PortsFetcher,
		logger:         cfg.Logger,
	}
}

// RefreshManifest regenerates the manifest from the current closure. It
// re-derives dependencies and bundle inclusions, refreshes ports, and clears
// secrets so they are re-fetched. Target, edge and the rest of the manifest
// are preserved from the base manifest.
func (r *manifestRefresher) RefreshManifest(ctx context.Context, base domain.CloudManifest) (domain.CloudManifest, error) {
	if base.Scenario.ID == "" {
		return base, fmt.Errorf("manifest has no scenario ID")
	}
	if r.closureSource == nil {
		return base, fmt.Errorf("manifest refresher has no closure source")
	}
	refreshed := base
	if err := r.refreshFromClosure(ctx, &refreshed); err != nil {
		return base, err
	}
	return r.refreshPortsAndSecrets(ctx, refreshed)
}

// refreshFromClosure replaces the dependency snapshot and bundle inclusions
// with the closure projection and binds the closure digest into the manifest.
func (r *manifestRefresher) refreshFromClosure(ctx context.Context, refreshed *domain.CloudManifest) error {
	environment := refreshed.Environment
	if environment == "" {
		environment = "production"
	}
	resolved, err := r.closureSource.Resolve(ctx, refreshed.Scenario.ID, environment)
	if err != nil {
		r.log("closure unavailable, refusing manifest refresh", map[string]interface{}{
			"scenario_id": refreshed.Scenario.ID,
			"error":       err.Error(),
		})
		return fmt.Errorf("derive closure for %s: %w", refreshed.Scenario.ID, err)
	}
	deps, bundleSpec := closure.ManifestDependencies(resolved)
	deps.ProgramBindingPeers = refreshed.Dependencies.ProgramBindingPeers
	deps.Analyzer.Fingerprint = refreshed.Dependencies.Analyzer.Fingerprint
	deps.Analyzer.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	refreshed.Dependencies = deps
	refreshed.Bundle.Scenarios = bundleSpec.Scenarios
	refreshed.Bundle.Resources = bundleSpec.Resources
	r.log("refreshed dependencies from closure", map[string]interface{}{
		"scenario_id":    refreshed.Scenario.ID,
		"closure_digest": resolved.Digest,
		"resources":      deps.Resources,
		"scenarios":      deps.Scenarios,
		"unsupported":    len(resolved.Unsupported),
	})
	return nil
}

func (r *manifestRefresher) refreshPortsAndSecrets(ctx context.Context, refreshed domain.CloudManifest) (domain.CloudManifest, error) {
	scenarioID := refreshed.Scenario.ID

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
