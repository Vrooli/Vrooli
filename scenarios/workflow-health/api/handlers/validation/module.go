package validation

import (
	"context"
	"log"
	"path/filepath"
	"time"

	"workflow-health/internal/module"
	internalvalidation "workflow-health/internal/validation"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	corevalidationrun "github.com/vrooli/api-core/validationrun"
	workflowrun "workflow-health/internal/validationrun"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

var ProtoFile = scenariovalidationv1.File_scenario_validation_v1_validation_proto

func Module(logger *log.Logger, repoRoot string, db *database.RoutedDB) module.Module {
	spec, err := internalvalidation.LoadSpec(filepath.Join(repoRoot, "scenarios", "workflow-health"))
	if err != nil && logger != nil {
		logger.Printf("validation: maturity assessment disabled: %v", err)
	}
	handler := NewConnectHandler(Deps{
		Logger:       logger,
		Engine:       internalvalidation.NewEngine(),
		MaturitySpec: spec,
		RepoRoot:     repoRoot,
		Ledger:       workflowrun.Repository{DB: db},
	})
	connectPath, connectHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(handler)
	durablePath, durableHandler := scenariovalidationconnect.NewDurableValidationRunServiceHandler(handler)
	return module.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler}, connectx.ServiceMount{Path: durablePath, Handler: durableHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

// RecoverInterrupted marks unfinished provider work explicitly terminal after a
// server restart. BAS execution is never replayed implicitly; operators can
// inspect the durable record and deliberately start a new idempotent request.
func RecoverInterrupted(ctx context.Context, db *database.RoutedDB) error {
	ledger := workflowrun.Repository{DB: db}
	runs, err := ledger.ListInterrupted(ctx)
	if err != nil {
		return err
	}
	for _, record := range runs {
		next, err := corevalidationrun.Transition(record.Run, corevalidationrun.EventRecoveryFailed, time.Now().UTC())
		if err != nil {
			return err
		}
		next.Version = record.Run.Version + 1
		record.Run = next
		record.ErrorCode, record.Error = string(corevalidationrun.ErrorRecoveryFailed), "workflow-health restarted before durable validation execution completed"
		if err := ledger.Update(ctx, record, next.Version-1); err != nil {
			return err
		}
	}
	return nil
}

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "validation_validate_scenario",
		Path:        scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate scenario workflow health",
		Description: "Scans scenario-owned BAS cases, flows, actions, seeds, and registry metadata, optionally executes validation cases through BAS, and returns the shared scenario-validation response.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "status": "scenario_validation.v1.ValidationStatus", "assessment": "common.v1.MaturityAssessment", "native_detail": "google.protobuf.Any<google.protobuf.Struct>", "metrics": "common.v1.ExecutionMetrics"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing, cannot be resolved, or workflow execution cannot reach BAS"}},
		Examples: []module.Example{
			{Name: "Validate workflow assets", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/ValidateScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"workflow-health\"}'"},
		},
	},
	{
		ID:          "validation_start_run",
		Path:        scenariovalidationconnect.DurableValidationRunServiceStartValidationRunProcedure,
		Method:      "POST",
		Summary:     "Start a durable workflow validation run",
		Description: "Persists a provider-owned execution run and returns promptly with static evidence and a reattachable handle.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "idempotency_key": "string", "parent_run_id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"run": "scenario_validation.v1.ValidationRun"}},
	},
	{
		ID:          "validation_get_run",
		Path:        scenariovalidationconnect.DurableValidationRunServiceGetValidationRunProcedure,
		Method:      "POST",
		Summary:     "Get a durable workflow validation run",
		Description: "Reads persisted provider-owned run state for reattachment or recovery.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"run_id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"run": "scenario_validation.v1.ValidationRun"}},
	},
	{
		ID:          "validation_wait_run",
		Path:        scenariovalidationconnect.DurableValidationRunServiceWaitValidationRunProcedure,
		Method:      "POST",
		Summary:     "Wait for a durable workflow validation run",
		Description: "Blocks on provider-owned lifecycle notifications without cancelling provider work on caller timeout or disconnect.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"run_id": "string", "timeout": "google.protobuf.Duration"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"run": "scenario_validation.v1.ValidationRun"}},
	},
	{
		ID:          "validation_abort_run",
		Path:        scenariovalidationconnect.DurableValidationRunServiceAbortValidationRunProcedure,
		Method:      "POST",
		Summary:     "Abort a durable workflow validation run",
		Description: "Records an explicit cancellation request and asks the provider worker to stop.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"run_id": "string", "reason": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"run": "scenario_validation.v1.ValidationRun"}},
	},
	{
		ID:          "validation_preview_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure,
		Method:      "POST",
		Summary:     "Preview deterministic workflow fixes",
		Description: "Returns candidate edits for mechanical workflow fixes such as registry rebuilds, metadata stubs, execution-mode normalization, and legacy reset normalization.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "[]string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "[]scenario_validation.v1.FixCandidate", "messages": "[]string"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
	},
	{
		ID:          "validation_apply_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure,
		Method:      "POST",
		Summary:     "Apply deterministic workflow fixes",
		Description: "Applies workflow-health's deterministic mechanical fixes and reports the edits written to disk.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "[]string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "[]scenario_validation.v1.FixCandidate", "messages": "[]string"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
	},
}
