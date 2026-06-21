package validation

import (
	"context"
	"database/sql"
	"log"
	"path/filepath"

	"performance-health/internal/autofix"
	"performance-health/internal/budgets"
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
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
	},
	{
		ID:          "readiness_preview_readiness_fix",
		Path:        readinessconnect.ReadinessServicePreviewReadinessFixProcedure,
		Method:      "POST",
		Summary:     "Preview deterministic readiness fixes",
		Description: "Returns the format-preserving edits readiness could apply to move a scenario toward Tier 1, without writing.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "array<scenario_validation.v1.FixCandidate>", "messages": "array<string>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
	},
	{
		ID:          "readiness_apply_readiness_fix",
		Path:        readinessconnect.ReadinessServiceApplyReadinessFixProcedure,
		Method:      "POST",
		Summary:     "Apply deterministic readiness fixes",
		Description: "Applies the format-preserving edits to move a scenario toward Tier 1 and reports what changed.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "array<scenario_validation.v1.FixCandidate>", "messages": "array<string>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
	},
	{
		ID:          "scenario_validation_validate_scenario",
		Path:        scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate scenario performance readiness through the shared provider contract",
		Description: "Runs performance-health's readiness engine and returns the shared scenario-validation response; the native readiness response is packed into native_detail. This is the contract test-genie delegates axes ①/③ to.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "include_execution": "bool"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "status": "scenario_validation.v1.ValidationStatus", "assessment": "common.v1.MaturityAssessment", "native_detail": "google.protobuf.Any<performance_health.v1.readiness.ValidateReadinessResponse>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
	},
	{
		ID:          "scenario_validation_preview_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure,
		Method:      "POST",
		Summary:     "Preview deterministic readiness fixes through the shared provider contract",
		Description: "Returns the format-preserving readiness edits in the shared scenario-validation FixResponse shape, without writing.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "array<scenario_validation.v1.FixCandidate>", "messages": "array<string>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
	},
	{
		ID:          "scenario_validation_apply_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure,
		Method:      "POST",
		Summary:     "Apply deterministic readiness fixes through the shared provider contract",
		Description: "Applies the format-preserving readiness edits and reports what changed, in the shared scenario-validation FixResponse shape.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "array<scenario_validation.v1.FixCandidate>", "messages": "array<string>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
	},
}
