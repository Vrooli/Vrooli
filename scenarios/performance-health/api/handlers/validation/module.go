package validation

import (
	"context"
	"database/sql"
	"log"
	"path/filepath"

	"performance-health/internal/autofix"
	internalbench "performance-health/internal/benchmark"
	"performance-health/internal/budgets"
	internallh "performance-health/internal/lighthouse"
	"performance-health/internal/module"
	"performance-health/internal/readiness"
	"performance-health/internal/trend"

	"github.com/gorilla/mux"
	"github.com/vrooli/maturity-go/assessment"
	vroolicli "github.com/vrooli/vrooli-cli-go"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/readiness"
	readinessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/readiness/readiness_v1connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

// ProtoFile and ScenarioValidationProtoFile are the FileDescriptors backing the
// two Connect-mounted services; the global parity test walks them against the
// Endpoints slice.
var (
	ProtoFile                   = readinessv1.File_performance_health_v1_readiness_readiness_proto
	ScenarioValidationProtoFile = scenariovalidationv1.File_scenario_validation_v1_validation_proto
)

// Module mounts both the native ReadinessService and the shared
// ScenarioValidationService (dual-mount), backed by one readiness engine.
//
// The readiness engine reads surfaces + UI framework through the Code Facts
// intake seam (CodeFactsClient), which degrades to a filesystem scan when Code
// Facts is unavailable.
func Module(logger *log.Logger, repoRoot string, db *sql.DB) module.Module {
	readinessSvc := readiness.NewService(readiness.NewCodeFactsClient(repoRoot))
	autofixSvc := autofix.NewService()

	// The budget gate folds a perf-budget breach into the validation assessment
	// as an ERROR finding, so a regression fails baseline-diff like any other
	// health regression. It reads the declarative budget config and evaluates the
	// newest persisted trend sample.
	budgetOpts := []budgets.Option{}
	if db != nil {
		budgetOpts = append(budgetOpts, budgets.WithMeasurementSource(budgets.NewTrendMeasurementSource(trend.NewStore(db))))
	}
	budgetSvc := budgets.NewService(budgets.NewConfigStore(repoRoot, nil), budgetOpts...)

	// Execution-mode orchestrator: runs the deterministic producers (build +
	// bundle benchmark, Lighthouse-if-UI), persists a fresh trend sample, and
	// folds threshold breaches into the assessment when a caller sends
	// include_execution=true. Startup is intentionally NOT wired here: restarting
	// the scenario-under-test mid test-run collides with the harness lifecycle, so
	// the startup axis stays standalone-fed (see Contract Decisions in PROGRESS.md).
	// Needs the trend store, so it is only wired when a DB is present.
	var execution ExecutionRunner
	if db != nil {
		execution = NewExecutionOrchestrator(ExecutionDeps{
			Benchmark:  internalbench.NewService(&internalbench.CLIRunner{RepoRoot: repoRoot}),
			Lighthouse: internallh.NewService(&internallh.CLIRunner{RepoRoot: repoRoot}),
			Trend:      trend.NewStore(db),
			Logger:     logger,
		})
	}

	spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", "performance-health"))
	if err != nil && logger != nil {
		logger.Printf("validation: maturity assessment unavailable: %v", err)
	}

	environment, envErr := vroolicli.New().HostCaptureEnvironment(context.Background())
	if envErr != nil {
		if logger != nil {
			logger.Printf("validation: host inventory unavailable, metrics environment limited to stdlib baseline: %v", envErr)
		}
		environment = nil
	}

	handler := NewHandlerWithDeps(Deps{
		Readiness:    readinessSvc,
		Autofix:      autofixSvc,
		Budgets:      budgetSvc,
		Logger:       logger,
		MaturitySpec: spec,
		RepoRoot:     repoRoot,
		Environment:  environment,
		Execution:    execution,
	})
	connectPath, connectHandler := readinessconnect.NewReadinessServiceHandler(handler)
	sharedPath, sharedHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(NewSharedHandler(handler))
	return module.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			r.PathPrefix(connectPath).Handler(connectHandler)
			r.PathPrefix(sharedPath).Handler(sharedHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the empty schema: validation owns no database tables.
func Schema() string { return "" }

// scenarioPathErrors is the invalid-argument error every validation endpoint
// returns when the scenario/path cannot be resolved.
func scenarioPathErrors() []module.ErrorDesc {
	return []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}}
}

// fixEndpoint builds the descriptor for a deterministic-fix endpoint (preview or
// apply). All four fix endpoints — native ReadinessService and the shared
// ScenarioValidationService — take the same {scenario, path, rule_ids} request
// and return the same FixResponse shape, so only the id/path/summary/description
// vary.
func fixEndpoint(id, path, summary, description string) module.EndpointDescriptor {
	return module.EndpointDescriptor{
		ID:          id,
		Path:        path,
		Method:      "POST",
		Summary:     summary,
		Description: description,
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "array<scenario_validation.v1.FixCandidate>", "messages": "array<string>"}},
		Errors:      scenarioPathErrors(),
	}
}

// Endpoints is the static endpoint metadata for codegen and the parity test.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "readiness_validate_readiness",
		Path:        readinessconnect.ReadinessServiceValidateReadinessProcedure,
		Method:      "POST",
		Summary:     "Validate a scenario's capture-tier readiness",
		Description: "Decides the reachable capture tier from Code Facts and detects the Tier-1 perf-build infra, emitting a finding per missing piece with a shared maturity assessment.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"status": "string", "tier": "CaptureTier", "ui_framework": "string", "surfaces": "array<string>", "assessment": "common.v1.MaturityAssessment", "autofixable_count": "int32"}},
		Errors:      scenarioPathErrors(),
	},
	fixEndpoint(
		"readiness_preview_readiness_fix",
		readinessconnect.ReadinessServicePreviewReadinessFixProcedure,
		"Preview deterministic readiness fixes",
		"Returns the format-preserving edits readiness could apply to move a scenario toward Tier 1, without writing.",
	),
	fixEndpoint(
		"readiness_apply_readiness_fix",
		readinessconnect.ReadinessServiceApplyReadinessFixProcedure,
		"Apply deterministic readiness fixes",
		"Applies the format-preserving edits to move a scenario toward Tier 1 and reports what changed.",
	),
	{
		ID:          "scenario_validation_validate_scenario",
		Path:        scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate scenario performance readiness through the shared provider contract",
		Description: "Runs performance-health's readiness engine and returns the shared scenario-validation response; the native readiness response is packed into native_detail. This is the contract test-genie delegates axes ①/③ to.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "include_execution": "bool"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "status": "scenario_validation.v1.ValidationStatus", "assessment": "common.v1.MaturityAssessment", "native_detail": "google.protobuf.Any<performance_health.v1.readiness.ValidateReadinessResponse>"}},
		Errors:      scenarioPathErrors(),
	},
	fixEndpoint(
		"scenario_validation_preview_fix",
		scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure,
		"Preview deterministic readiness fixes through the shared provider contract",
		"Returns the format-preserving readiness edits in the shared scenario-validation FixResponse shape, without writing.",
	),
	fixEndpoint(
		"scenario_validation_apply_fix",
		scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure,
		"Apply deterministic readiness fixes through the shared provider contract",
		"Applies the format-preserving readiness edits and reports what changed, in the shared scenario-validation FixResponse shape.",
	),
}
