// Package generation provides desktop application generation services.
// This domain handles analyzing scenarios, creating desktop configurations,
// and orchestrating the Electron wrapper template generation process.
package generation

// Service orchestrates desktop generation operations.
type Service interface {
	// Generate generates a desktop application from a config.
	Generate(buildID string, config *DesktopConfig)

	// QueueBuild queues a desktop build and returns the initial status.
	QueueBuild(config *DesktopConfig, metadata *ScenarioMetadata, includeMetadata bool) *BuildStatus
}

// ScenarioAnalyzer analyzes scenarios and extracts metadata.
type ScenarioAnalyzer interface {
	// AnalyzeScenario analyzes a scenario and extracts all relevant metadata.
	AnalyzeScenario(scenarioName string) (*ScenarioMetadata, error)

	// ValidateScenarioForDesktop checks if a scenario is ready for desktop generation.
	ValidateScenarioForDesktop(scenarioName string) error

	// CreateDesktopConfigFromMetadata generates a DesktopConfig from analyzed metadata.
	CreateDesktopConfigFromMetadata(metadata *ScenarioMetadata, templateType string) (*DesktopConfig, error)
}

// IconSyncer syncs scenario icons into desktop assets.
type IconSyncer interface {
	// SyncIcons copies the best-available scenario icon into the Electron assets.
	SyncIcons(scenario, outputDir string, logf func(string, map[string]interface{})) error
}

// TemplateExecutor executes the template generator.
type TemplateExecutor interface {
	// Execute runs the template generator with the given config.
	Execute(configPath string) (output string, err error)
}

// BuildStore manages build statuses.
type BuildStore interface {
	// Create creates a new build status.
	Create(buildID string) *BuildStatus

	// Get retrieves a build status by ID.
	Get(buildID string) (*BuildStatus, bool)

	// Update updates a build status.
	Update(buildID string, fn func(status *BuildStatus))
}

// ConfigPersister persists and loads desktop configurations.
type ConfigPersister interface {
	// Save saves a desktop config for a scenario.
	Save(scenarioPath string, config *DesktopConfig) error

	// Load loads a desktop config for a scenario.
	Load(scenarioPath string) (*ConnectionConfig, error)
}

// RecordDeleter deletes records by scenario name.
type RecordDeleter interface {
	// DeleteByScenario removes all records for a scenario and returns the count.
	DeleteByScenario(scenarioName string) int
}
