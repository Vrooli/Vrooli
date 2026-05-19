package resources

import (
	"fmt"
	"io"
	"path/filepath"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/structure/types"
)

// ExpectationsLoader loads resource expectations from service.json.
type ExpectationsLoader interface {
	// Load reads the service manifest and returns required resources.
	Load() ([]string, error)
}

// loader is the default implementation of ExpectationsLoader.
type loader struct {
	scenarioDir string
	logWriter   io.Writer

	// manifestLoader is injectable for testing.
	manifestLoader ManifestLoader
}

// ManifestLoader abstracts manifest loading for testing.
type ManifestLoader interface {
	Load(path string) (*workspace.ServiceManifest, error)
}

// New creates a new expectations loader.
func NewLoader(scenarioDir string, logWriter io.Writer, opts ...LoaderOption) ExpectationsLoader {
	l := &loader{
		scenarioDir:    scenarioDir,
		logWriter:      logWriter,
		manifestLoader: &defaultManifestLoader{},
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// LoaderOption configures a loader.
type LoaderOption func(*loader)

// WithManifestLoader sets a custom manifest loader (for testing).
func WithManifestLoader(ml ManifestLoader) LoaderOption {
	return func(l *loader) {
		l.manifestLoader = ml
	}
}

// Load implements ExpectationsLoader.
func (l *loader) Load() ([]string, error) {
	manifestPath := filepath.Join(l.scenarioDir, ".vrooli", "service.json")
	manifest, err := l.manifestLoader.Load(manifestPath)
	if err != nil {
		return nil, err
	}

	required := manifest.RequiredResources()
	if len(required) == 0 {
		l.logWarn("no required resources declared in %s", manifestPath)
		return nil, nil
	}

	for _, resource := range required {
		l.logStep("resource requirement detected: %s", resource)
	}

	return required, nil
}

// logStep writes a step message to the log.
func (l *loader) logStep(format string, args ...interface{}) {
	if l.logWriter == nil {
		return
	}
	fmt.Fprintf(l.logWriter, format+"\n", args...)
}

// logWarn writes a warning message to the log.
func (l *loader) logWarn(format string, args ...interface{}) {
	if l.logWriter == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.logWriter, "[WARNING] %s\n", msg)
}

// defaultManifestLoader uses workspace.LoadServiceManifest.
type defaultManifestLoader struct{}

func (d *defaultManifestLoader) Load(path string) (*workspace.ServiceManifest, error) {
	return workspace.LoadServiceManifest(path)
}

// LoadResult represents the outcome of loading resource expectations.
type LoadResult struct {
	// Success indicates whether loading succeeded.
	Success bool

	// Error contains the loading error, if any.
	Error error

	// FailureClass categorizes the type of failure.
	FailureClass types.FailureClass

	// Remediation provides guidance on how to fix the issue.
	Remediation string

	// Observations contains detailed observations.
	Observations []types.Observation

	// Resources lists the required resources.
	Resources []string
}

// LoadExpectations loads and returns resource expectations as a result.
func LoadExpectations(scenarioDir string, logWriter io.Writer) LoadResult {
	loader := NewLoader(scenarioDir, logWriter)
	resources, err := loader.Load()
	if err != nil {
		return LoadResult{
			Success:      false,
			Error:        err,
			FailureClass: types.FailureClassMisconfiguration,
			Remediation:  "Fix .vrooli/service.json so required resources can be read.",
		}
	}

	var observations []types.Observation
	if len(resources) == 0 {
		observations = append(observations, types.NewInfoObservation("manifest declares no required resources"))
	} else {
		for _, resource := range resources {
			observations = append(observations, types.NewInfoObservation(
				fmt.Sprintf("requires resource: %s", resource),
			))
		}
	}

	return LoadResult{
		Success:      true,
		Observations: observations,
		Resources:    resources,
	}
}
