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

var (
	performanceMaxDuration = 90 * time.Second
	uiBuildMaxDuration     = 180 * time.Second
)

// runPerformancePhase ensures core build artifacts complete within acceptable time limits.
func runPerformancePhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if err := ctx.Err(); err != nil {
		return RunReport{Err: err, FailureClassification: FailureClassSystem}
	}

	var observations []Observation
	goObs, goFailure := benchmarkGoBuild(ctx, env, logWriter)
	if goFailure != nil {
		goFailure.Observations = append(goFailure.Observations, observations...)
		return *goFailure
	}
	observations = append(observations, goObs...)

	if hasNodeWorkspace(env) {
		uiObs, uiFailure := benchmarkUIBuild(ctx, env, logWriter)
		if uiFailure != nil {
			uiFailure.Observations = append(uiFailure.Observations, observations...)
			return *uiFailure
		}
		observations = append(observations, uiObs...)
	} else {
		observations = append(observations, NewObservation("ui workspace not detected"))
	}

	logPhaseStep(logWriter, "performance validation complete")
	return RunReport{Observations: observations}
}

func benchmarkGoBuild(ctx context.Context, env workspace.Environment, logWriter io.Writer) ([]Observation, *RunReport) {
	if err := EnsureCommandAvailable("go"); err != nil {
		return nil, &RunReport{
			Err:                   err,
			FailureClassification: FailureClassMissingDependency,
			Remediation:           "Install the Go toolchain so API builds can be benchmarked.",
		}
	}

	apiDir := filepath.Join(env.ScenarioDir, "api")
	if err := ensureDir(apiDir); err != nil {
		return nil, &RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Restore the api/ directory before running performance benchmarks.",
		}
	}

	tmpFile, err := os.CreateTemp("", "test-genie-perf-*")
	if err != nil {
		return nil, &RunReport{
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
		return nil, &RunReport{
			Err:                   fmt.Errorf("go build failed: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Fix compilation errors before re-running the performance phase.",
		}
	}
	duration := time.Since(start)
	seconds := int(duration.Round(time.Second) / time.Second)
	logPhaseStep(logWriter, "go build completed in %ds", seconds)
	observations := []Observation{NewSuccessObservation(fmt.Sprintf("go build duration: %ds", seconds))}

	if duration > performanceMaxDuration {
		return nil, &RunReport{
			Err:                   fmt.Errorf("go build exceeded %s (took %s)", performanceMaxDuration, duration),
			FailureClassification: FailureClassSystem,
			Remediation:           "Investigate slow dependencies or remove unnecessary modules before building.",
			Observations:          observations,
		}
	}
	return observations, nil
}

func benchmarkUIBuild(ctx context.Context, env workspace.Environment, logWriter io.Writer) ([]Observation, *RunReport) {
	nodeDir := detectNodeWorkspaceDir(env.ScenarioDir)
	if nodeDir == "" {
		return []Observation{NewObservation("ui workspace not detected")}, nil
	}

	manifest, err := loadPackageManifest(filepath.Join(nodeDir, "package.json"))
	if err != nil {
		return nil, &RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Fix package.json so the Node workspace can be parsed.",
		}
	}
	buildScript := ""
	if manifest != nil {
		buildScript = manifest.Scripts["build"]
	}
	if buildScript == "" {
		return []Observation{NewObservation("ui workspace lacks build script")}, nil
	}

	manager := detectPackageManager(manifest, nodeDir)
	if err := EnsureCommandAvailable(manager); err != nil {
		return nil, &RunReport{
			Err:                   err,
			FailureClassification: FailureClassMissingDependency,
			Remediation:           fmt.Sprintf("Install %s to run UI build benchmarks.", manager),
		}
	}
	logPhaseStep(logWriter, "running UI build via %s", manager)
	if _, err := os.Stat(filepath.Join(nodeDir, "node_modules")); os.IsNotExist(err) {
		if installErr := installNodeDependencies(ctx, nodeDir, manager, logWriter); installErr != nil {
			return nil, &RunReport{
				Err:                   fmt.Errorf("%s install failed: %w", manager, installErr),
				FailureClassification: FailureClassSystem,
				Remediation:           "Resolve dependency installation issues before benchmarking the UI build.",
			}
		}
	}
	start := time.Now()
	if err := phaseCommandExecutor(ctx, nodeDir, logWriter, manager, "run", "build"); err != nil {
		return nil, &RunReport{
			Err:                   fmt.Errorf("ui build failed: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Inspect the UI build output above, fix failures, and rerun the suite.",
		}
	}
	duration := time.Since(start)
	seconds := int(duration.Round(time.Second) / time.Second)
	logPhaseStep(logWriter, "ui build completed in %ds", seconds)
	observations := []Observation{NewSuccessObservation(fmt.Sprintf("ui build duration: %ds", seconds))}
	if duration > uiBuildMaxDuration {
		return nil, &RunReport{
			Err:                   fmt.Errorf("ui build exceeded %s (took %s)", uiBuildMaxDuration, duration),
			FailureClassification: FailureClassSystem,
			Remediation:           "Investigate slow front-end builds or trim unused dependencies.",
			Observations:          observations,
		}
	}
	return observations, nil
}
