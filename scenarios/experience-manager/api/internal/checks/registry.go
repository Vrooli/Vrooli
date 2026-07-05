package checks

import "experience-manager/internal/reconcile"

// RegistryDeps wires runtime seams needed by checks.
type RegistryDeps struct {
	EvidenceRepository reconcile.EvidenceRepository
}

// Registry returns the registered experience checks in run order.
func Registry(deps ...RegistryDeps) []Check {
	var d RegistryDeps
	if len(deps) > 0 {
		d = deps[0]
	}
	return []Check{reconcile.Check{Repository: d.EvidenceRepository}}
}
