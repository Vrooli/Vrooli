package validation

import (
	"context"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	maturity "github.com/vrooli/maturity-go/assessment"
	vroolicli "github.com/vrooli/vrooli-cli-go"

	"business-health/internal/assessment"
	localautofix "business-health/internal/autofix"
	"business-health/internal/checks"
	"business-health/internal/extraction"
	"business-health/internal/module"

	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/contract"
	contractconnect "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/contract/contract_v1connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

var (
	ProtoFile                   = contractv1.File_business_health_v1_contract_contract_proto
	ScenarioValidationProtoFile = scenariovalidationv1.File_scenario_validation_v1_validation_proto
)

// NewEngine builds the check engine (shared with the wizard module's
// post-apply validation).
func NewEngine(repoRoot string) *checks.Engine {
	return checks.New(repoRoot, extraction.NewFileExtractor(), checks.Registry()...)
}

// Module wires the validation domain: the shared ScenarioValidationService
// (test-genie's delegated-provider contract) and the native ContractService
// (rich detail for business-health's own CLI and UI), both answering from
// one engine.
func Module(logger *log.Logger, repoRoot, scenarioDir string, engine *checks.Engine) module.Module {
	spec, err := maturity.LoadSpecFromScenario(scenarioDir)
	if err != nil {
		logger.Fatalf("validation: load maturity spec: %v", err)
	}
	builder, err := assessment.NewBuilder(spec)
	if err != nil {
		logger.Fatalf("validation: %v", err)
	}
	// Capture host facts once; they do not change during the process
	// lifetime. Failure is non-fatal — the metrics collector backfills
	// os/arch/num_cpu from the stdlib.
	environment, err := vroolicli.New().HostCaptureEnvironment(context.Background())
	if err != nil {
		logger.Printf("validation: host inventory unavailable, metrics environment limited to stdlib baseline: %v", err)
		environment = nil
	}
	extractor := extraction.NewFileExtractor()
	if engine == nil {
		engine = checks.New(repoRoot, extractor, checks.Registry()...)
	}
	core := NewConnectHandler(Deps{
		Logger:      logger,
		Engine:      engine,
		Builder:     builder,
		Fixers:      localautofix.NewRegistry(),
		Extractor:   extractor,
		RepoRoot:    repoRoot,
		Environment: environment,
	})
	sharedPath, sharedHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(core)
	contractPath, contractHandler := contractconnect.NewContractServiceHandler(newContractService(core))
	return module.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(
				r,
				connectx.ServiceMount{Path: sharedPath, Handler: sharedHandler},
				connectx.ServiceMount{Path: contractPath, Handler: contractHandler},
			)
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
		Summary:     "Validate a scenario's business contract (shared mount)",
		Description: "Validates one scenario's PRD.md + requirements/ (template conformance, registry structure, intent linkage, evidence traceability) and returns the shared maturity assessment with execution metrics.",
		Category:    "validation",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string (required, scenario slug)", "path": "string (optional resolved path)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":      "string",
				"status":        "scenario_validation.v1.ValidationStatus",
				"assessment":    "common.v1.MaturityAssessment",
				"native_detail": "google.protobuf.Any (business_health.v1.contract.BusinessContractReport)",
				"metrics":       "common.v1.ExecutionMetrics",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario slug or unresolvable target"},
		},
		Examples: []module.Example{
			{Name: "Validate a scenario", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/ValidateScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"image-tools\"}'"},
		},
	},
	{
		ID:          "validation_preview_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure,
		Method:      "POST",
		Summary:     "Preview deterministic business-contract fixes",
		Description: "Dry-run of the deterministic fixers (template-section scaffold, registry creation, status normalization, prd_ref stubs). Fixers register in plan phase 6; until then the registry answers with no candidates.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "rule_ids": "array<string> (optional filter)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "boolean (always false)", "candidates": "array<FixCandidate>", "messages": "array<string>"}},
	},
	{
		ID:          "validation_apply_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure,
		Method:      "POST",
		Summary:     "Apply deterministic business-contract fixes",
		Description: "Applies the previewed deterministic fixes. Fixers register in plan phase 6; until then the registry answers with no candidates.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "rule_ids": "array<string> (optional filter)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "boolean", "candidates": "array<FixCandidate>", "messages": "array<string>"}},
	},
	{
		ID:          "contract_validate_scenario",
		Path:        contractconnect.ContractServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate a scenario's business contract (native detail)",
		Description: "Same validation as the shared mount, answering with the rich native BusinessContractReport (capability rollups, matrix rows, drift entries) beside the shared assessment.",
		Category:    "validation",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string (required, scenario slug)", "path": "string (optional resolved path)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":   "string",
				"status":     "string",
				"report":     "BusinessContractReport",
				"assessment": "common.v1.MaturityAssessment",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario slug or unresolvable target"},
		},
		Examples: []module.Example{
			{Name: "Native validate", Curl: "curl http://localhost:${API_PORT}/vrooli.business_health.v1.contract.ContractService/ValidateScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"image-tools\"}'"},
		},
	},
	{
		ID:          "contract_get_matrix",
		Path:        contractconnect.ContractServiceGetMatrixProcedure,
		Method:      "POST",
		Summary:     "Traceability matrix for one scenario",
		Description: "OT × requirement × validation × evidence join. Lands with the evidence domain (plan phase 4); answers Unimplemented until then.",
		Category:    "traceability",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"matrix": "array<MatrixRow>", "registry": "RegistrySummary"}},
		Errors:      []module.ErrorDesc{{Status: 501, Code: "unimplemented", Description: "Matrix lands in plan phase 4"}},
	},
	{
		ID:          "contract_get_drift",
		Path:        contractconnect.ContractServiceGetDriftProcedure,
		Method:      "POST",
		Summary:     "Evidence drift for one scenario",
		Description: "Stale snapshots, expired manual attestations, unearned statuses, unproven claims. Lands with the evidence domain (plan phase 4); answers Unimplemented until then.",
		Category:    "traceability",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"drift": "array<DriftEntry>"}},
		Errors:      []module.ErrorDesc{{Status: 501, Code: "unimplemented", Description: "Drift lands in plan phase 4"}},
	},
	{
		ID:          "contract_log_manual_validation",
		Path:        contractconnect.ContractServiceLogManualValidationProcedure,
		Method:      "POST",
		Summary:     "Append a manual-validation attestation",
		Description: "Appends to the manual-validations ledger (the one evidence artifact business-health owns). Lands with the evidence domain (plan phase 4); answers Unimplemented until then.",
		Category:    "traceability",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "requirement_id": "string", "attested_by": "string", "notes": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"attestation": "ManualAttestation", "ledger_path": "string"}},
		Errors:      []module.ErrorDesc{{Status: 501, Code: "unimplemented", Description: "Ledger lands in plan phase 4"}},
	},
}
