package discovery

import (
	"context"
	"errors"
	"fmt"
	"io"

	reqdiscovery "test-genie/internal/requirements/discovery"
)

// Validator discovers requirement modules and returns validation results.
type Validator interface {
	// Discover finds all requirement modules in the scenario.
	// requireModules controls whether empty results should be an error.
	Discover(ctx context.Context, requireModules bool) DiscoveryResult
}

// validator wraps requirements/discovery with validation result handling.
type validator struct {
	scenarioDir string
	discoverer  reqdiscovery.Discoverer
	logWriter   io.Writer
}

// New creates a new discovery validator for the given scenario directory.
func New(scenarioDir string, logWriter io.Writer) Validator {
	return &validator{
		scenarioDir: scenarioDir,
		discoverer:  reqdiscovery.NewDefault(),
		logWriter:   logWriter,
	}
}

// NewWithDiscoverer creates a validator with a custom discoverer (for testing).
func NewWithDiscoverer(scenarioDir string, discoverer reqdiscovery.Discoverer, logWriter io.Writer) Validator {
	return &validator{
		scenarioDir: scenarioDir,
		discoverer:  discoverer,
		logWriter:   logWriter,
	}
}

// Discover implements Validator.
func (v *validator) Discover(ctx context.Context, requireModules bool) DiscoveryResult {
	logInfo(v.logWriter, "Discovering requirement files...")

	files, err := v.discoverer.Discover(ctx, v.scenarioDir)
	if err != nil {
		if errors.Is(err, reqdiscovery.ErrNoRequirementsDir) {
			return DiscoveryResult{
				Result: FailMisconfiguration(
					err,
					"Run `vrooli scenario requirements init` to scaffold P0/P1 modules for this scenario.",
				),
			}
		}
		return DiscoveryResult{
			Result: FailSystem(
				err,
				"Ensure Test Genie can read module files under requirements/.",
			),
		}
	}

	// Convert discovered files to our type
	converted := make([]DiscoveredFile, len(files))
	for i, f := range files {
		converted[i] = DiscoveredFile{
			AbsolutePath: f.AbsolutePath,
			RelativePath: f.RelativePath,
			IsIndex:      f.IsIndex,
			ModuleDir:    f.ModuleDir,
		}
	}

	// Filter to only module files (not index files)
	moduleCount := 0
	for _, f := range converted {
		if !f.IsIndex {
			moduleCount++
		}
	}

	if moduleCount == 0 && requireModules {
		return DiscoveryResult{
			Result: FailMisconfiguration(
				fmt.Errorf("no requirement modules found"),
				"Run `vrooli scenario requirements init` to scaffold P0/P1 modules for this scenario.",
			),
			Files: converted,
		}
	}

	logSuccess(v.logWriter, "Discovered %d module file(s)", moduleCount)

	return DiscoveryResult{
		Result: OK().WithObservations(
			NewSuccessObservation(fmt.Sprintf("discovered %d module(s)", moduleCount)),
		),
		ModuleCount: moduleCount,
		Files:       converted,
	}
}

// Logging helpers

func logInfo(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "🔍 %s\n", msg)
}

func logSuccess(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[SUCCESS] ✅ %s\n", msg)
}
