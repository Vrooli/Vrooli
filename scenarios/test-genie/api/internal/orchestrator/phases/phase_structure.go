package phases

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/structure"
	"test-genie/internal/structure/existence"
)

// runStructurePhase validates the essential scenario layout using the structure package.
// This includes existence checks (directories, files, CLI), content validation (JSON, manifest),
// and UI smoke testing.
func runStructurePhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	var summary string

	report := RunPhaseWithExpectations(ctx, env, logWriter, "structure",
		structure.LoadExpectations,
		func(expectations *structure.Expectations) (*structure.RunResult, error) {
			// Determine schemas directory
			schemasDir := filepath.Join(env.AppRoot, "scenarios", "test-genie", "schemas")
			if info, err := os.Stat(schemasDir); err != nil || !info.IsDir() {
				schemasDir = ""
			}

			runner := structure.New(structure.Config{
				ScenarioDir:  env.ScenarioDir,
				ScenarioName: env.ScenarioName,
				SchemasDir:   schemasDir,
				Expectations: expectations,
			}, structure.WithLogger(logWriter))
			return runner.Run(ctx), nil
		},
		func(r *structure.RunResult) PhaseResult[structure.Observation] {
			if r != nil {
				summary = r.Summary.String()
			}
			return ExtractWithSummary(
				r.Success,
				r.Error,
				r.FailureClass,
				r.Remediation,
				r.Observations,
				"✅",
				fmt.Sprintf("Structure validation completed (%s)", r.Summary.String()),
			)
		},
	)

	writePhasePointer(env, "structure", report, map[string]any{"summary": summary}, logWriter)
	return report
}

// GetCLIApproach returns the CLI approach for the given scenario.
// This is exposed for other phases (like integration) that may need to know.
func GetCLIApproach(scenarioDir, scenarioName string) existence.CLIApproach {
	return existence.DetectCLIApproach(scenarioDir, scenarioName)
}
