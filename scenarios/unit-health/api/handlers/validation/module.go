package validation

import (
	"context"
	"log"
	"path/filepath"

	"unit-health/internal/discovery"
	"unit-health/internal/module"
	"unit-health/internal/runhistory"
	internalvalidation "unit-health/internal/validation"

	"github.com/gorilla/mux"
	"github.com/vrooli/maturity-go/assessment"
	vroolicli "github.com/vrooli/vrooli-cli-go"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation/validation_v1connect"
)

// ProtoFile is the FileDescriptor backing this Connect-mounted module; the
// global parity test walks it against the Endpoints slice.
var (
	ProtoFile                   = validationv1.File_unit_health_v1_validation_validation_proto
	ScenarioValidationProtoFile = scenariovalidationv1.File_scenario_validation_v1_validation_proto
)

// Module mounts the ValidationService Connect handler. history persists run
// timing/status for cross-run diagnostics; pass nil to disable persistence.
func Module(logger *log.Logger, repoRoot string, history runhistory.Store) module.Module {
	svc := internalvalidation.New()
	// DescribeProvider answers readiness from this provider's own descriptor,
	// so a readiness probe no longer costs a full target analysis. A load
	// failure yields the zero Describer, which reports Unimplemented and makes
	// consumers fall back to the legacy probe.
	describer, _ := assessment.LoadDescriber(filepath.Join(repoRoot, "scenarios", "unit-health"))
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", "unit-health"))
	if err != nil && logger != nil {
		logger.Printf("validation: maturity assessment unavailable: %v", err)
	}
	svc.Spec = spec
	svc.Locator = discovery.DefaultLocator{RepoRoot: repoRoot}
	svc.History = history
	// Capture host facts once; they do not change during the process lifetime.
	// A failure (CLI unavailable) is non-fatal — the metrics collector backfills
	// os/arch/num_cpu from the stdlib, leaving richer facts unset.
	environment, envErr := vroolicli.New().HostCaptureEnvironment(context.Background())
	if envErr != nil {
		if logger != nil {
			logger.Printf("validation: host inventory unavailable, metrics environment limited to stdlib baseline: %v", envErr)
		}
		environment = nil
	}
	handler := NewHandlerWithDeps(Deps{
		Service:      svc,
		Logger:       logger,
		MaturitySpec: spec,
		Environment:  environment,
	})
	connectPath, connectHandler := validationconnect.NewValidationServiceHandler(handler)
	sharedPath, sharedHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(assessment.Serve(NewSharedHandler(handler), describer))
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
		ID:          "validation_validate_scenario",
		Path:        validationconnect.ValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate scenario test maturity",
		Description: "Discovers test surfaces through Code Facts, plans and optionally runs the canonical test commands, analyzes coverage/architecture/quality, and returns normalized findings plus a shared maturity assessment.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "workspaces": "array<string>", "include_execution": "bool", "use_cache": "bool"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"status": "string", "surfaces": "array<TestSurface>", "workspaces": "array<TestWorkspace>", "findings": "array<ValidationFinding>", "coverage": "array<CoverageTarget>", "projection_checks": "array<ProjectionCheck>", "maturity": "MaturitySummary", "assessment": "common.v1.MaturityAssessment"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
	},
	{
		ID:          "scenario_validation_validate_scenario",
		Path:        scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate scenario test maturity through the shared provider contract",
		Description: "Runs Unit Health's validation engine and returns the shared scenario-validation response; the native unit-health ValidateScenarioResponse is packed into native_detail.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "include_execution": "bool"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "status": "scenario_validation.v1.ValidationStatus", "assessment": "common.v1.MaturityAssessment", "native_detail": "google.protobuf.Any<unit_health.v1.validation.ValidateScenarioResponse>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
		// Same user-facing capability as validation_validate_scenario, exposed
		// through the shared scenario-validation provider contract. The
		// `validate scenario` CLI command covers it; map both endpoints to it.

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
		ID:          "scenario_validation_preview_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure,
		Method:      "POST",
		Summary:     "Preview unit validation fixes",
		Description: "Previews deterministic low-risk Unit Health config/projection fixes without writing files.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "string[]"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "scenario_validation.v1.FixCandidate[]", "messages": "string[]"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
	},
	{
		ID:          "scenario_validation_apply_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure,
		Method:      "POST",
		Summary:     "Apply unit validation fixes",
		Description: "Applies deterministic low-risk Unit Health config/projection fixes with before-write drift checks.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "string[]"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "scenario_validation.v1.FixCandidate[]", "messages": "string[]"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}, {Status: 412, Code: "failed_precondition", Description: "A target file changed before the fix could be applied"}},
	},
}
