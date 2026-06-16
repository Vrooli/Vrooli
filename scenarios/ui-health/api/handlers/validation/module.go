package validation

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"ui-health/internal/module"
	"ui-health/internal/services/manifestvalidation"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/maturity-go/assessment"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/validation/validation_v1connect"
)

// ProtoFile exposes the validation domain's proto FileDescriptor so the
// global parity test (api/internal/modules/registry_test.go) can walk it.
var ProtoFile = validationv1.File_ui_health_v1_validation_validation_proto

// Module returns the validation domain's contribution to the API.
func Module(logger *log.Logger, repoRoot string) module.Module {
	validator := manifestvalidation.New(repoRoot, logger)
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
	path := filepath.Join(repoRoot, "scenarios", "ui-health", ".vrooli", "maturity.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return assessment.ParseSpec(raw)
}

func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "validation_validate_scenario",
		Path:        validationconnect.ValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate a scenario's UI manifest, slot directories, and overlay rules",
		Description: "Runs ui-health validators against a scenario and returns a structured Finding list.",
		Category:    "validation",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string (required, scenario id)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":   "string",
				"passed":     "boolean",
				"findings":   "array<Finding>",
				"summary":    "Summary",
				"assessment": "common.v1.MaturityAssessment",
			},
		},
		Examples: []module.Example{
			{Name: "Validate scenario", Curl: "curl http://localhost:${API_PORT}/vrooli.ui_health.v1.validation.ValidationService/ValidateScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"ui-health\"}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "ui-health validate scenario",
			Args:    []string{"<name>"},
		},
	},
}
