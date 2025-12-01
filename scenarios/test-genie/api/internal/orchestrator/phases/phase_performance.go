package phases

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"test-genie/internal/orchestrator/workspace"
)

var performanceMaxDuration = 90 * time.Second

// runPerformancePhase ensures the Go API builds within acceptable time limits so suites catch regressions early.
func runPerformancePhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if err := ctx.Err(); err != nil {
		return RunReport{Err: err, FailureClassification: FailureClassSystem}
	}

	if err := EnsureCommandAvailable("go"); err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMissingDependency,
			Remediation:           "Install the Go toolchain so API builds can be benchmarked.",
		}
	}

	apiDir := filepath.Join(env.ScenarioDir, "api")
	if err := ensureDir(apiDir); err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Restore the api/ directory before running performance benchmarks.",
		}
	}

	tmpFile, err := os.CreateTemp("", "test-genie-perf-*")
	if err != nil {
		return RunReport{
			Err:                   fmt.Errorf("failed to create temp binary: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Verify the filesystem is writable for performance artifacts.",
		}
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	logPhaseStep(logWriter, "building Go API binary in %s", apiDir)
	start := time.Now()
	if err := phaseCommandExecutor(ctx, apiDir, logWriter, "go", "build", "-o", tmpPath, "./..."); err != nil {
		return RunReport{
			Err:                   fmt.Errorf("go build failed: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Fix compilation errors before re-running the performance phase.",
		}
	}
	duration := time.Since(start)
	seconds := int(duration.Round(time.Second) / time.Second)
	logPhaseStep(logWriter, "go build completed in %ds", seconds)
	observations := []string{fmt.Sprintf("go build duration: %ds", seconds)}

	if duration > performanceMaxDuration {
		return RunReport{
			Err:                   fmt.Errorf("go build exceeded %s (took %s)", performanceMaxDuration, duration),
			FailureClassification: FailureClassSystem,
			Remediation:           "Investigate slow dependencies or remove unnecessary modules before building.",
			Observations:          observations,
		}
	}

	logPhaseStep(logWriter, "performance validation complete")
	return RunReport{Observations: observations}
}
