// Package scenariomanifest exposes the canonical typed service manifest to
// scenario modules whose Go modules cannot import the root internal package.
package scenariomanifest

import scenariomodel "github.com/vrooli/vrooli/internal/scenario"

type ServiceManifest = scenariomodel.ServiceManifest

// Load reads the canonical manifest shape without applying lifecycle
// validation. Callers that only need declared fields should use this reader so
// validation remains owned by the control plane.
func Load(path string) (ServiceManifest, error) {
	return scenariomodel.LoadServiceManifest(path)
}
