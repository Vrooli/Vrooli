package checks

import (
	"time"

	"experience-manager/internal/attestation"
	"experience-manager/internal/reconcile"
)

// RegistryDeps wires runtime seams needed by checks.
type RegistryDeps struct {
	EvidenceRepository     reconcile.EvidenceRepository
	AttestationRepository  attestation.Repository
	AttestationCurrentTime func() time.Time
}

// Registry returns the registered experience checks in run order.
func Registry(deps ...RegistryDeps) []Check {
	var d RegistryDeps
	if len(deps) > 0 {
		d = deps[0]
	}
	return []Check{
		BASReferenceCheck{},
		StateCoverageCheck{},
		attestation.Check{Repository: d.AttestationRepository, Now: d.AttestationCurrentTime},
		reconcile.Check{Repository: d.EvidenceRepository},
	}
}
