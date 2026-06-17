package validation

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"unit-health/internal/discovery"
	"unit-health/internal/module"
	"unit-health/internal/runhistory"
	internalvalidation "unit-health/internal/validation"

	"github.com/gorilla/mux"
	"github.com/vrooli/maturity-go/assessment"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation/validation_v1connect"
)

// ProtoFile is the FileDescriptor backing this Connect-mounted module; the
// global parity test walks it against the Endpoints slice.
var ProtoFile = validationv1.File_unit_health_v1_validation_validation_proto

// Module mounts the ValidationService Connect handler. history persists run
// timing/status for cross-run diagnostics; pass nil to disable persistence.
func Module(logger *log.Logger, repoRoot string, history runhistory.Store) module.Module {
	svc := internalvalidation.New()
	spec, err := loadMaturitySpec(repoRoot)
	if err != nil && logger != nil {
		logger.Printf("validation: maturity assessment unavailable: %v", err)
	}
	svc.Spec = spec
	svc.Locator = discovery.DefaultLocator{RepoRoot: repoRoot}
	svc.History = history
	connectPath, connectHandler := validationconnect.NewValidationServiceHandler(NewHandlerWithDeps(Deps{
		Service:      svc,
		Logger:       logger,
		MaturitySpec: spec,
	}))
	return module.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			r.PathPrefix(connectPath).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

func loadMaturitySpec(repoRoot string) (*assessment.Spec, error) {
	path := filepath.Join(repoRoot, "scenarios", "unit-health", ".vrooli", "maturity.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return assessment.ParseSpec(raw)
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
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"status": "string", "surfaces": "array<TestSurface>", "workspaces": "array<TestWorkspace>", "findings": "array<ValidationFinding>", "coverage": "array<CoverageTarget>", "maturity": "MaturitySummary", "assessment": "common.v1.MaturityAssessment"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
		CLIMapping:  &module.CLIMapping{Command: "unit-health validate scenario", Args: []string{"<scenario>", "--json"}},
	},
}
