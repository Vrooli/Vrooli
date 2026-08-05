package validation

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"security-health/internal/module"
	"security-health/internal/validation"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/maturity-go/assessment"
	vroolicli "github.com/vrooli/vrooli-cli-go"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

// ProtoFile exposes the shared validation proto FileDescriptor so the global
// parity test (api/internal/modules/registry_test.go) can walk it.
var ProtoFile = scenariovalidationv1.File_scenario_validation_v1_validation_proto

// Module returns the validation domain's contribution to the API: a single
// Connect-RPC service handler mounted at the generated procedure path. The
// validator is constructed with the real exec/scanner seams rooted at repoRoot.
func Module(logger *log.Logger, repoRoot string) module.Module {
	policy := validation.RolloutProfile(strings.ToLower(strings.TrimSpace(os.Getenv("SECURITY_HEALTH_POLICY_MODE"))))
	if policy != validation.RolloutAdvisory && policy != validation.RolloutGuided && policy != validation.RolloutGuarded && policy != validation.RolloutEnforcing {
		policy = validation.RolloutAdvisory
	}
	validator := validation.New(validation.Deps{RepoRoot: repoRoot, Logger: logger, PolicyMode: policy})
	scenarioDir := filepath.Join(repoRoot, "scenarios", "security-health")
	spec, err := assessment.LoadSpecFromScenario(scenarioDir)
	if err != nil && logger != nil {
		logger.Printf("validation: maturity assessment disabled: %v", err)
	}
	// DescribeProvider answers readiness from this descriptor alone. security-health
	// has no cheap inspection mode, so before this existed a readiness probe had to
	// run the full security scan over the target to read two identity strings.
	// A load failure is non-fatal: the zero Describer reports Unimplemented and
	// consumers fall back to the legacy probe.
	describer, describerErr := assessment.LoadDescriber(scenarioDir)
	if describerErr != nil && logger != nil {
		logger.Printf("validation: DescribeProvider disabled, readiness will fall back to full validation: %v", describerErr)
	}
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
	connectPath, connectHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(assessment.Serve(NewConnectHandler(Deps{
		Logger:       logger,
		Validator:    validator,
		MaturitySpec: spec,
		Environment:  environment,
	}), describer.WithFixes(true)))
	return module.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — validation is stateless (no tables). The registry
// re-exports it anyway so the per-domain shape stays uniform.
func Schema() string { return "" }

// Endpoints is the machine-readable description of the validation module's
// public surface. References the generated *Procedure constant so renaming the
// RPC in validation.proto breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{ID: "validation_validate_target", Path: scenariovalidationconnect.ScenarioValidationServiceValidateTargetProcedure, Method: "POST", Summary: "Validate a first-class repository target", Category: "validation"},
	{
		ID:          "validation_validate_scenario",
		Path:        scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate a scenario's security posture",
		Description: "Detects the target scenario's substrates and runs applicable security scanners, returning the shared scenario-validation response with findings in the maturity assessment.",
		Category:    "validation",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string (required, scenario id)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":      "string",
				"status":        "scenario_validation.v1.ValidationStatus",
				"assessment":    "common.v1.MaturityAssessment",
				"native_detail": "google.protobuf.Any",
				"metrics":       "common.v1.ExecutionMetrics",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario id or scenario not found under scenarios/"},
		},
		Examples: []module.Example{
			{Name: "Validate scenario", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/ValidateScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"security-health\"}'"},
		},
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
		Summary:     "Preview deterministic security fixes",
		Description: "Returns safe deterministic security remediations, currently limited to generated-Go API security headers middleware edits, without writing files.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "array<FixCandidate>", "messages": "array<string>"}},
		Examples:    []module.Example{{Name: "Preview fixes", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/PreviewFix -H 'Content-Type: application/json' -d '{\"scenario\":\"security-health\",\"rule_ids\":[\"security-health.security-headers-missing\"]}'"}},
	},
	{
		ID:          "validation_apply_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure,
		Method:      "POST",
		Summary:     "Apply deterministic security fixes",
		Description: "Applies safe deterministic security remediations selected by rule id and reports the file edits written.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "array<FixCandidate>", "messages": "array<string>"}},
		Examples:    []module.Example{{Name: "Apply fixes", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/ApplyFix -H 'Content-Type: application/json' -d '{\"scenario\":\"security-health\",\"rule_ids\":[\"security-health.security-headers-missing\"]}'"}},
	},
}
