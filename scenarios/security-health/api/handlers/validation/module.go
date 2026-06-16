package validation

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"security-health/internal/module"
	"security-health/internal/validation"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/maturity-go/assessment"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/validation/validation_v1connect"
)

// ProtoFile exposes the validation domain's proto FileDescriptor so the global
// parity test (api/internal/modules/registry_test.go) can walk it without
// importing the gen/go package directly.
var ProtoFile = validationv1.File_security_health_v1_validation_validation_proto

// Module returns the validation domain's contribution to the API: a single
// Connect-RPC service handler mounted at the generated procedure path. The
// validator is constructed with the real exec/scanner seams rooted at repoRoot.
func Module(logger *log.Logger, repoRoot string) module.Module {
	validator := validation.New(validation.Deps{RepoRoot: repoRoot, Logger: logger})
	spec, err := loadMaturitySpec(repoRoot)
	if err != nil && logger != nil {
		logger.Printf("validation: maturity assessment disabled: %v", err)
	}
	connectPath, connectHandler := validationconnect.NewValidationServiceHandler(NewConnectHandler(Deps{
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

func loadMaturitySpec(repoRoot string) (*assessment.Spec, error) {
	path := filepath.Join(repoRoot, "scenarios", "security-health", ".vrooli", "maturity.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return assessment.ParseSpec(raw)
}

// Schema returns "" — validation is stateless (no tables). The registry
// re-exports it anyway so the per-domain shape stays uniform.
func Schema() string { return "" }

// Endpoints is the machine-readable description of the validation module's
// public surface. References the generated *Procedure constant so renaming the
// RPC in validation.proto breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "validation_validate_scenario",
		Path:        validationconnect.ValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate a scenario's security posture",
		Description: "Detects the target scenario's substrates and runs the applicable security scanners (secrets, Go SAST, Go vuln-DB, JS deps), returning normalized severity-correct findings. critical/high → ERROR (gates ecosystem-manager R1), medium → WARNING, low/info → INFO.",
		Category:    "validation",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string (required, scenario id)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":         "string",
				"passed":           "boolean",
				"findings":         "array<Finding>",
				"summary":          "Summary",
				"skipped_scanners": "array<string>",
				"assessment":       "common.v1.MaturityAssessment",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario id or scenario not found under scenarios/"},
		},
		Examples: []module.Example{
			{Name: "Validate scenario", Curl: "curl http://localhost:${API_PORT}/vrooli.security_health.v1.validation.ValidationService/ValidateScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"security-health\"}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "security-health validate scenario",
			Args:    []string{"<name>"},
		},
	},
}
