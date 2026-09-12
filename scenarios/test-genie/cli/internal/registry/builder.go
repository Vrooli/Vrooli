package registry

import apiRegistry "test-genie/cli/internal/playbookregistry"

const RegistryFileName = apiRegistry.RegistryFileName

type (
	Builder     = apiRegistry.Builder
	BuildResult = apiRegistry.BuildResult
)

// NewBuilder reuses the API registry builder so CLI and API share one registry
// schema and one normalization pipeline.
func NewBuilder(scenarioDir string) *Builder {
	return apiRegistry.NewBuilder(scenarioDir)
}
