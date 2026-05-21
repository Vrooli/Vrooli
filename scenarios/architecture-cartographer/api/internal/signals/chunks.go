package signals

import (
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/manifest"
)

// GraphContext bundles the inputs every Signal needs plus the per-run
// shared caches (community detection, glossary lookups). The aggregator
// builds the context once per scoring batch; signals receive a value
// copy with shared pointers to immutable caches.
//
// GraphContext is created via NewGraphContext so callers can't construct
// half-populated contexts.
type GraphContext struct {
	Scenario  string
	Snapshot  graph.GraphSnapshot
	Manifest  manifest.ManifestDefinition
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
func NewGraphContext(scenario string, snap graph.GraphSnapshot, m manifest.ManifestDefinition) GraphContext {
	return GraphContext{
		Scenario: scenario,
		Snapshot: snap,
		Manifest: m,
		Caches:   &Caches{Community: map[string]int{}},
	}
}
