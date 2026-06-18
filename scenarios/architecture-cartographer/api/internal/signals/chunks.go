package signals

import (
	"sync"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

// GraphContext bundles the inputs every Signal needs plus the per-run
// shared caches (community detection, glossary lookups). The aggregator
// builds the context once per scoring batch; signals receive a value
// copy with shared pointers to immutable caches.
//
// GraphContext is created via NewGraphContext so callers can't construct
// half-populated contexts. The embedded *Caches is goroutine-safe so the
// same context can be passed to a worker pool batch-scoring many chunks
// concurrently.
type GraphContext struct {
	Scenario string
	Snapshot graph.GraphSnapshot
	// DomainMap is the derived domain map for the scenario (replaces the
	// deleted per-scenario architecture manifest). Signals read declared
	// domains, owned paths, and glossary vocabulary from it.
	DomainMap domains.DerivedDomainMap
	Caches    *Caches
}

// Caches is the shared cache surface for one scoring batch. Access is
// goroutine-safe: ScoreBatch shares one *Caches across workers.
type Caches struct {
	mu sync.RWMutex
	// community is the per-package Louvain community id used by the
	// import-cluster signal. Written once (idempotently) and read by every
	// subsequent score.
	community map[string]int
	// domainPackages maps package_id -> domain_name for the derived domain
	// map. Several signals need this same index, so it is computed once per
	// scoring batch and cached here.
	domainPackages map[string]string
}

// CommunitySnapshot returns the current community map, or nil if the
// cache has not been populated yet. Returns a defensive copy so callers
// can iterate without holding the lock.
func (c *Caches) CommunitySnapshot() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.community) == 0 {
		return nil
	}
	out := make(map[string]int, len(c.community))
	for k, v := range c.community {
		out[k] = v
	}
	return out
}

// SetCommunity replaces the community cache atomically. Subsequent
// calls are no-ops if the cache is already populated (the value is
// idempotent — same graph in, same clusters out).
func (c *Caches) SetCommunity(in map[string]int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.community) > 0 {
		return
	}
	c.community = make(map[string]int, len(in))
	for k, v := range in {
		c.community[k] = v
	}
}

// DomainPackagesSnapshot returns the current package-to-domain index, or nil
// if the cache has not been populated yet. Returns a defensive copy.
func (c *Caches) DomainPackagesSnapshot() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.domainPackages) == 0 {
		return nil
	}
	out := make(map[string]string, len(c.domainPackages))
	for k, v := range c.domainPackages {
		out[k] = v
	}
	return out
}

// SetDomainPackages replaces the package-to-domain cache atomically.
// Subsequent calls are no-ops if the cache is already populated.
func (c *Caches) SetDomainPackages(in map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.domainPackages) > 0 {
		return
	}
	c.domainPackages = make(map[string]string, len(in))
	for k, v := range in {
		c.domainPackages[k] = v
	}
}

// NewGraphContext constructs a fresh context with an empty Caches.
func NewGraphContext(scenario string, snap graph.GraphSnapshot, m domains.DerivedDomainMap) GraphContext {
	return GraphContext{
		Scenario:  scenario,
		Snapshot:  snap,
		DomainMap: m,
		Caches:    &Caches{},
	}
}
