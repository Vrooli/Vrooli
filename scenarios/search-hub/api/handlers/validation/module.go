package validation

import (
	"log"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"

	evalH "search-hub/handlers/eval"
	"search-hub/internal/clock"
	internaleval "search-hub/internal/eval"
	"search-hub/internal/module"
	internalregistry "search-hub/internal/registry"
	internalvalidation "search-hub/internal/validation"
)

var ProtoFile = scenariovalidationv1.File_scenario_validation_v1_validation_proto

func Module(logger *log.Logger, repoRoot string, db *database.RoutedDB, clk clock.Clock) module.Module {
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", "search-hub"))
	if err != nil && logger != nil {
		logger.Printf("validation: maturity assessment disabled: %v", err)
	}
	validator := internalvalidation.New(repoRoot)
	if db != nil {
		resolver := internalregistry.NewSQLiteStore(db, clk)
		validator.EvalStore = internaleval.NewSQLiteStore(db, clk)
		validator.EvalValidator = internaleval.NewValidator(resolver, evalH.NewDefaultProviderClient())
	}
	connectPath, connectHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(NewConnectHandler(Deps{
		Logger:       logger,
		Validator:    validator,
		MaturitySpec: spec,
	}))
	return module.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "validation_validate_scenario",
		Path:        scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate scenario search maturity",
		Description: "Validates a target scenario's .vrooli/search.json provider descriptors, readiness declarations, tuning budget posture, and eval corpus shape, then returns the shared scenario-validation response.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "include_execution": "bool"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "status": "scenario_validation.v1.ValidationStatus", "assessment": "common.v1.MaturityAssessment", "native_detail": "google.protobuf.Any<google.protobuf.Struct>", "metrics": "common.v1.ExecutionMetrics"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
		Examples: []module.Example{
			{Name: "Validate search maturity", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/ValidateScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"cli-health\"}'"},
		},
	},
	{
		ID:          "validation_preview_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure,
		Method:      "POST",
		Summary:     "Preview deterministic search maturity fixes",
		Description: "Reserved for future mechanical .vrooli/search.json repairs.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "array<scenario_validation.v1.FixCandidate>", "messages": "array<string>"}},
		Errors:      []module.ErrorDesc{{Status: 501, Code: "unimplemented", Description: "Search maturity fixes are not implemented"}},
	},
	{
		ID:          "validation_apply_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure,
		Method:      "POST",
		Summary:     "Apply deterministic search maturity fixes",
		Description: "Reserved for future mechanical .vrooli/search.json repairs.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "array<scenario_validation.v1.FixCandidate>", "messages": "array<string>"}},
		Errors:      []module.ErrorDesc{{Status: 501, Code: "unimplemented", Description: "Search maturity fixes are not implemented"}},
	},
}
