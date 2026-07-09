package scenariovalidation

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

var ProtoFile protoreflect.FileDescriptor = scenariovalidationv1.File_scenario_validation_v1_validation_proto

func Module(repo catalog.Repository, logger *log.Logger) module.Module {
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil && logger != nil {
		logger.Printf("scenario validation: repo root unavailable: %v", err)
	}
	spec := templatevalidation.MaturitySpec()
	if loaded, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", templatevalidation.Provider)); err == nil {
		spec = loaded
	}
	connectPath, connectHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(NewConnectHandler(Deps{
		Logger:       logger,
		Validator:    templatevalidation.NewValidator(repoRoot, repo),
		Fixers:       templatevalidation.NewFixRegistry(repoRoot),
		MaturitySpec: spec,
	}))
	return module.Module{
		Name: "scenario-validation",
		Mount: func(r *mux.Router) {
			r.PathPrefix(strings.TrimRight(connectPath, "/")).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
