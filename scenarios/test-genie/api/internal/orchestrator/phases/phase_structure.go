package phases

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/structure"
	"test-genie/internal/structure/existence"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// runStructurePhase validates the essential scenario layout using the structure package.
// This includes existence checks (directories, files, CLI), content validation (JSON, manifest),
// and UI smoke testing.
func runStructurePhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return RunNativePhase(ctx, env, logWriter, Structure,
		structure.LoadExpectations,
		func(expectations *structure.Expectations) (StandardRunResult, error) {
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
		WithNativePhaseReportHook(func(report *RunReport, _ StandardRunResult) {
			report.Findings = structureArchFindings(env.ScenarioName, report.Observations)
		}),
	)
}

// structureArchFindings maps the structure phase's error/warning
// observations into the shared ArchitectureFinding contract
// (source=STRUCTURE). The structure runner produces text-only
// observations (no rule codes/locations), so the observation MESSAGE is
// the finding identity: it serves as both `code` (stable-ID input) and
// `message`. Distinct messages yield distinct IDs; identical messages
// collapse (correct dedup). When the structure runner gains coded
// findings, switch `code` to the rule id. Success/info/section/skip
// observations are not findings and are skipped.
func structureArchFindings(scenario string, obs []Observation) []*architecturev1.ArchitectureFinding {
	var out []*architecturev1.ArchitectureFinding
	for _, o := range obs {
		var sev string
		switch o.Prefix {
		case "ERROR":
			sev = "error"
		case "WARNING":
			sev = "warning"
		default:
			continue
		}
		text := strings.TrimSpace(o.Text)
		if text == "" {
			continue
		}
		out = append(out, newFinding(
			scenario,
			architecturev1.FindingSource_FINDING_SOURCE_STRUCTURE,
			text, sev, text, "",
			nil, nil,
		))
	}
	return out
}

// GetCLIApproach returns the CLI approach for the given scenario.
// This is exposed for other phases (like integration) that may need to know.
func GetCLIApproach(scenarioDir, scenarioName string) existence.CLIApproach {
	manifest, err := existence.LoadServiceManifest(scenarioDir)
	if err != nil {
		return existence.CLIApproachUnknown
	}
	return existence.DetectCLIApproach(manifest)
}
