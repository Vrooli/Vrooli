package detection

import (
	"path/filepath"
	"sync"

	appconfig "scenario-dependency-analyzer/internal/config"
)

// catalog.go - Catalog management and caching
//
// This file handles lazy loading and caching of scenario and resource catalogs,
// which are used to validate detected dependencies.

// catalogManager handles thread-safe catalog loading and caching
type catalogManager struct {
	cfg            appconfig.Config
	mu             sync.RWMutex
	loaded         bool
	knownScenarios map[string]struct{}
	knownResources map[string]struct{}
}

// newCatalogManager creates a new catalog manager with the given configuration
func newCatalogManager(cfg appconfig.Config) *catalogManager {
	return &catalogManager{
		cfg: cfg,
	}
}

// ensureLoaded loads catalogs if they haven't been loaded yet (thread-safe)
func (c *catalogManager) ensureLoaded() {
	// Fast path: already loaded
	c.mu.RLock()
	if c.loaded {
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()

	// Slow path: need to load
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.loaded {
		return
	}

	scenariosDir := determineScenariosDir(c.cfg.ScenariosDir)
	c.knownScenarios = discoverAvailableScenarios(scenariosDir)

	resourcesDir := filepath.Join(filepath.Dir(scenariosDir), "resources")
	c.knownResources = discoverAvailableResources(resourcesDir)

	c.loaded = true
}

// refresh invalidates the cached catalogs, forcing a reload on next access
func (c *catalogManager) refresh() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.loaded = false
	c.knownScenarios = nil
	c.knownResources = nil
}

// isKnownScenario checks if a scenario exists in the catalog
func (c *catalogManager) isKnownScenario(name string) bool {
	c.ensureLoaded()

	c.mu.RLock()
	defer c.mu.RUnlock()

	// If catalog is empty, assume all scenarios are valid (permissive mode)
	if len(c.knownScenarios) == 0 {
		return true
	}

	_, ok := c.knownScenarios[normalizeName(name)]
	return ok
}

// isKnownResource checks if a resource exists in the catalog
func (c *catalogManager) isKnownResource(name string) bool {
	c.ensureLoaded()

	c.mu.RLock()
	defer c.mu.RUnlock()

	// If catalog is empty, assume all resources are valid (permissive mode)
	if len(c.knownResources) == 0 {
		return true
	}

	_, ok := c.knownResources[normalizeName(name)]
	return ok
}

// getScenarioCatalog returns a copy of the scenario catalog for inspection
func (c *catalogManager) getScenarioCatalog() map[string]struct{} {
	c.ensureLoaded()

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to prevent external modification
	snapshot := make(map[string]struct{}, len(c.knownScenarios))
	for k := range c.knownScenarios {
		snapshot[k] = struct{}{}
	}

	return snapshot
}

// getResourceCatalog returns a copy of the resource catalog for inspection
func (c *catalogManager) getResourceCatalog() map[string]struct{} {
	c.ensureLoaded()

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to prevent external modification
	snapshot := make(map[string]struct{}, len(c.knownResources))
	for k := range c.knownResources {
		snapshot[k] = struct{}{}
	}

	return snapshot
}
