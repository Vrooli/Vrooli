package phases

import "sync"

// The default catalog is the single source of truth for descriptor identity,
// ordering, presets, and validation. Callers (orchestrator, CLI, tests) derive
// their phase knowledge from these helpers instead of restating phase literals.

var (
	defaultCatalogOnce sync.Once
	defaultCatalogInst *Catalog
)

// DefaultCatalog returns the process-wide descriptor catalog, lazily built
// from provider-owned descriptors in NewDefaultCatalog.
func DefaultCatalog() *Catalog {
	defaultCatalogOnce.Do(func() {
		defaultCatalogInst = NewDefaultCatalog(DefaultTimeout)
	})
	return defaultCatalogInst
}

// AllPhases returns descriptor names in catalog order.
func AllPhases() []Name {
	specs := DefaultCatalog().All()
	names := make([]Name, 0, len(specs))
	for _, spec := range specs {
		names = append(names, spec.Name)
	}
	return names
}

// ValidPhaseNames returns descriptor names as strings in catalog order.
func ValidPhaseNames() []string {
	names := AllPhases()
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, n.String())
	}
	return out
}

// IsValidPhase reports whether raw resolves to a registered descriptor
// (case-insensitive).
func IsValidPhase(raw string) bool {
	_, ok := DefaultCatalog().Lookup(raw)
	return ok
}
