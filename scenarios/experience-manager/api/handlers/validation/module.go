package validation

import (
	"log"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	apidb "github.com/vrooli/api-core/database"

	"experience-manager/internal/attestation"
	localautofix "experience-manager/internal/autofix"
	"experience-manager/internal/checks"
	"experience-manager/internal/module"
	"experience-manager/internal/reconcile"

	maturity "github.com/vrooli/maturity-go/assessment"

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
	evidenceRepository    reconcile.EvidenceRepository
	attestationRepository attestation.Repository
}

// Option configures the validation module.
type Option func(*moduleConfig)

// WithDatabase wires stateful validation dependencies through the routed DB.
func WithDatabase(db *apidb.RoutedDB) Option {
	return func(cfg *moduleConfig) {
		if db != nil {
			cfg.evidenceRepository = reconcile.NewSQLiteRepository(db)
			cfg.attestationRepository = attestation.NewSQLiteRepository(db)
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
		engine = NewEngine(repoRoot, checks.RegistryDeps{
			EvidenceRepository:    cfg.evidenceRepository,
			AttestationRepository: cfg.attestationRepository,
		})
	}
	core := NewConnectHandler(Deps{
		Logger:                logger,
		Engine:                engine,
		Fixers:                localautofix.NewRegistry(),
		RepoRoot:              repoRoot,
		AttestationRepository: cfg.attestationRepository,
	})
	// DescribeProvider answers readiness from this provider's own descriptor,
	// so a readiness probe no longer costs a full target analysis. A load
	// failure yields the zero Describer, which reports Unimplemented and makes
	// consumers fall back to the legacy probe.
	describer, _ := maturity.LoadDescriber(filepath.Join(repoRoot, "scenarios", "experience-manager"))
	sharedPath, sharedHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(maturity.Serve(core, describer))
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

func scenarioRequestSchema() *module.Schema {
	return &module.Schema{
		Type:       "object",
		Properties: map[string]string{"scenario": "string (required, scenario slug)", "path": "string (optional resolved path)"},
	}
}

func fleetRequestSchema() *module.Schema {
	return &module.Schema{Type: "object", Properties: map[string]string{"repo_root": "string (optional repo root override)"}}
}

func attestationRequestSchema() *module.Schema {
	return &module.Schema{Type: "object", Properties: map[string]string{
		"scenario":   "string",
		"page":       "string",
		"claim":      "string",
		"author":     "string",
		"rationale":  "string",
		"expires_at": "string (RFC3339)",
	}}
}

func scaffoldCasesRequestSchema() *module.Schema {
	return &module.Schema{Type: "object", Properties: map[string]string{
		"scenario": "string",
		"path":     "string",
		"dry_run":  "boolean",
	}}
}

func scaffoldCasesResponseSchema() *module.Schema {
	return &module.Schema{Type: "object", Properties: map[string]string{
		"scenario": "string",
		"applied":  "boolean",
		"diffs":    "array<FileDiff>",
		"messages": "array<string>",
	}}
}

func fixRequestSchema() *module.Schema {
	return &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string> (optional filter)"}}
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
		ID:          "validation_describe_provider",
		Path:        scenariovalidationconnect.ScenarioValidationServiceDescribeProviderProcedure,
		Method:      "POST",
		Summary:     "Describe this provider's identity and contract",
		Description: "Reports provider identity, backed phase, maturity spec version, contract, build provenance, and capabilities. Inspects no target, so readiness consumers can confirm this provider is live and current without paying for a full validation run.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"provider":     "string",
			"phase":        "string",
			"spec_version": "string",
			"contract":     "string",
			"build":        "scenario_validation.v1.ProviderBuild",
			"capabilities": "scenario_validation.v1.ProviderCapabilities",
		}},
		Examples: []module.Example{{Name: "Describe provider", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/DescribeProvider -H 'Content-Type: application/json' -d '{}'"}},
	},
	{
		ID:          "validation_preview_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure,
		Method:      "POST",
		Summary:     "Preview deterministic experience-contract fixes",
		Description: "Previews deterministic spec, BAS scaffold, and documentation fixes without writing.",
		Category:    "validation",
		Request:     fixRequestSchema(),
		Response:    fixResponseSchema(),
	},
	{
		ID:          "validation_apply_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure,
		Method:      "POST",
		Summary:     "Apply deterministic experience-contract fixes",
		Description: "Applies deterministic spec, BAS scaffold, and documentation fixes with sequential re-preview semantics.",
		Category:    "validation",
		Request:     fixRequestSchema(),
		Response:    fixResponseSchema(),
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
	{
		ID:          "contract_list_fleet",
		Path:        contractconnect.ContractServiceListFleetProcedure,
		Method:      "POST",
		Summary:     "List fleet-wide experience depth and debt",
		Description: "Computes experience/ presence, page depth, parser findings, and debt score across scenarios without persisted fleet state.",
		Category:    "validation",
		Request:     fleetRequestSchema(),
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"scenarios":             "array<FleetScenario>",
			"scenario_count":        "int32",
			"with_experience_count": "int32",
			"total_pages":           "int32",
		}},
		Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Repo root cannot be swept."}},
	},
	{
		ID:          "contract_append_attestation",
		Path:        contractconnect.ContractServiceAppendAttestationProcedure,
		Method:      "POST",
		Summary:     "Append manual experience attestation",
		Description: "Records one append-only manual attestation for a manual-tier claim with author, rationale, and expiry.",
		Category:    "validation",
		Request:     attestationRequestSchema(),
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"attestation": "ManualAttestation"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Attestation fields are incomplete or malformed."},
			{Status: 412, Code: "failed_precondition", Description: "Attestation repository is not configured."},
		},
	},
	{
		ID:          "contract_scaffold_cases",
		Path:        contractconnect.ContractServiceScaffoldCasesProcedure,
		Method:      "POST",
		Summary:     "Scaffold BAS cases from experience specs",
		Description: "Derives workflow-health-governed BAS case stubs from active experience page specs through the deterministic case-scaffold fixer.",
		Category:    "validation",
		Request:     scaffoldCasesRequestSchema(),
		Response:    scaffoldCasesResponseSchema(),
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario path cannot be resolved."}},
	},
	{
		ID:          "contract_get_readiness_profile",
		Path:        contractconnect.ContractServiceGetReadinessProfileProcedure,
		Method:      "POST",
		Summary:     "Compile the experience readiness profile",
		Description: "Compiles the authored experience contract into the single route/region readiness projection consumed by automation clients.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"profile": "experience_manager.v1.contract.ReadinessProfile"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario path cannot be resolved."}},
	},
}
