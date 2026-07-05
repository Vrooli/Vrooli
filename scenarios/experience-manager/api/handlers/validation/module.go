package validation

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	apidb "github.com/vrooli/api-core/database"

	localautofix "experience-manager/internal/autofix"
	"experience-manager/internal/checks"
	"experience-manager/internal/module"
	"experience-manager/internal/reconcile"

	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract"
	contractconnect "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract/contract_v1connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

var (
	ProtoFile                   = contractv1.File_experience_manager_v1_contract_contract_proto
	ScenarioValidationProtoFile = scenariovalidationv1.File_scenario_validation_v1_validation_proto
)

type moduleConfig struct {
	evidenceRepository reconcile.EvidenceRepository
}

// Option configures the validation module.
type Option func(*moduleConfig)

// WithDatabase wires stateful validation dependencies through the routed DB.
func WithDatabase(db *apidb.RoutedDB) Option {
	return func(cfg *moduleConfig) {
		if db != nil {
			cfg.evidenceRepository = reconcile.NewSQLiteRepository(db)
		}
	}
}

// NewEngine builds the validation engine seam.
func NewEngine(repoRoot string, deps ...checks.RegistryDeps) *checks.Engine {
	return checks.New(repoRoot, checks.Registry(deps...)...)
}

// Module wires the validation domain. Native and shared service surfaces use
// one parser-backed pipeline wrapper.
func Module(logger *log.Logger, repoRoot string, engine *checks.Engine, opts ...Option) module.Module {
	var cfg moduleConfig
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if engine == nil {
		engine = NewEngine(repoRoot, checks.RegistryDeps{EvidenceRepository: cfg.evidenceRepository})
	}
	core := NewConnectHandler(Deps{
		Logger:   logger,
		Engine:   engine,
		Fixers:   localautofix.NewRegistry(),
		RepoRoot: repoRoot,
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

func unimplementedError(description string) []module.ErrorDesc {
	return []module.ErrorDesc{{Status: 501, Code: "unimplemented", Description: description}}
}

func scenarioRequestSchema() *module.Schema {
	return &module.Schema{
		Type:       "object",
		Properties: map[string]string{"scenario": "string (required, scenario slug)", "path": "string (optional resolved path)"},
	}
}

func fixRequestSchema() *module.Schema {
	return &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "rule_ids": "array<string> (optional filter)"}}
}

func fixResponseSchema() *module.Schema {
	return &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "boolean", "candidates": "array<FixCandidate>", "messages": "array<string>"}}
}

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "validation_validate_scenario",
		Path:        scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate a scenario's experience contract (shared mount)",
		Description: "Parser-backed Test Genie ScenarioValidationService mount for scenario-experience-spec/v1 contracts.",
		Category:    "validation",
		Request:     scenarioRequestSchema(),
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":      "string",
				"status":        "scenario_validation.v1.ValidationStatus",
				"assessment":    "common.v1.MaturityAssessment",
				"native_detail": "google.protobuf.Any (experience_manager.v1.contract.ExperienceContractReport)",
			},
		},
		Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario path or experience contract cannot be parsed."}},
	},
	{
		ID:          "validation_preview_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure,
		Method:      "POST",
		Summary:     "Preview deterministic experience-contract fixes",
		Description: "Phase 1 mount for deterministic fix previews. Returns Unimplemented until fixers land.",
		Category:    "validation",
		Request:     fixRequestSchema(),
		Response:    fixResponseSchema(),
		Errors:      unimplementedError("Autofix lands in a later phase"),
	},
	{
		ID:          "validation_apply_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure,
		Method:      "POST",
		Summary:     "Apply deterministic experience-contract fixes",
		Description: "Phase 1 mount for deterministic fixes. Returns Unimplemented until fixers land.",
		Category:    "validation",
		Request:     fixRequestSchema(),
		Response:    fixResponseSchema(),
		Errors:      unimplementedError("Autofix lands in a later phase"),
	},
	{
		ID:          "contract_validate_scenario",
		Path:        contractconnect.ContractServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate a scenario's experience contract (native detail)",
		Description: "Native experience contract validation for scenario-experience-spec/v1 contracts.",
		Category:    "validation",
		Request:     scenarioRequestSchema(),
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":   "string",
				"status":     "string",
				"report":     "ExperienceContractReport",
				"assessment": "common.v1.MaturityAssessment",
			},
		},
		Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario path or experience contract cannot be parsed."}},
	},
}
