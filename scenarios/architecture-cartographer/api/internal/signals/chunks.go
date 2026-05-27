package signals

import (
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

// GraphContext bundles the inputs every Signal needs plus the per-run
// shared caches (community detection, glossary lookups). The aggregator
// builds the context once per scoring batch; signals receive a value
// copy with shared pointers to immutable caches.
//
// GraphContext is created via NewGraphContext so callers can't construct
// half-populated contexts.
type GraphContext struct {
	Scenario string
	Snapshot graph.GraphSnapshot
	// DomainMap is the derived domain map for the scenario (replaces the
	// deleted per-scenario architecture manifest). Signals read declared
	// domains, owned paths, and glossary vocabulary from it.
	DomainMap domains.DerivedDomainMap
	Caches    *Caches
}

// Caches is the shared cache surface. Empty in Phase 2; Phase 3 wires
// the community-detection + glossary caches.
type Caches struct {
	// Community is the per-package Louvain modularity cluster id. Empty
	// until Phase 5 wires the production community detection.
	Community map[string]int
}

// NewGraphContext constructs a fresh context with an empty Caches.
func NewGraphContext(scenario string, snap graph.GraphSnapshot, m domains.DerivedDomainMap) GraphContext {
	return GraphContext{
		Scenario:  scenario,
		Snapshot:  snap,
		DomainMap: m,
		Caches:    &Caches{Community: map[string]int{}},
	}
}
