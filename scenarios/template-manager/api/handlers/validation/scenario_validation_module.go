package validation

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/module"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatevalidation"

	"github.com/gorilla/mux"
	"github.com/vrooli/maturity-go/assessment"
	repocontract "github.com/vrooli/repo-contract-go"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var ScenarioValidationProtoFile protoreflect.FileDescriptor = scenariovalidationv1.File_scenario_validation_v1_validation_proto

func ScenarioValidationModule(repo catalog.Repository, logger *log.Logger) module.Module {
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil && logger != nil {
		logger.Printf("scenario validation: repo root unavailable: %v", err)
	}
	spec := templatevalidation.MaturitySpec()
	// DescribeProvider answers readiness from this provider's own descriptor,
	// so a readiness probe no longer costs a full target analysis. A load
	// failure yields the zero Describer, which reports Unimplemented and makes
	// consumers fall back to the legacy probe.
	describer, _ := assessment.LoadDescriber(filepath.Join(repoRoot, "scenarios", templatevalidation.Provider))
	if loaded, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", templatevalidation.Provider)); err == nil {
		spec = loaded
	}
	connectPath, connectHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(assessment.Serve(NewScenarioValidationHandler(ScenarioValidationDeps{
		Logger:       logger,
		Validator:    templatevalidation.NewValidator(repoRoot, repo),
		Fixers:       templatevalidation.NewFixRegistry(repoRoot),
		MaturitySpec: spec,
	}), describer))
	return module.Module{
		Name: "scenario-validation",
		Mount: func(r *mux.Router) {
			r.PathPrefix(strings.TrimRight(connectPath, "/")).Handler(connectHandler)
		},
		Endpoints: ScenarioValidationEndpoints,
	}
}
