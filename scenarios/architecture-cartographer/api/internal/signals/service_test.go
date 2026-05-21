package signals_test

import (
	"architecture-cartographer/internal/manifest"
)

// manifestEmpty returns an empty manifest used by aggregator tests
// that don't care about manifest overlay.
func manifestEmpty() manifest.ManifestDefinition {
	return manifest.ManifestDefinition{Scenario: "demo"}
}
