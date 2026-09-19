package conformance

import (
	"log"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/maturity-go/assessment"
	conformancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/conformance"
	conformanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/conformance/conformance_v1connect"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"

	"ai-gateway/internal/module"
)

var ProtoFile = conformancev1.File_ai_gateway_v1_conformance_conformance_proto

func Module(logger *log.Logger, repoRoot string, deps ...Deps) module.Module {
	var d Deps
	if len(deps) > 0 {
		d = deps[0]
	}
	// DescribeProvider answers readiness from this provider's own descriptor,
	// so a readiness probe no longer costs a full target analysis. A load
	// failure yields the zero Describer, which reports Unimplemented and makes
	// consumers fall back to the legacy probe.
	var describer assessment.Describer
	if repoRoot != "" {
		describer, _ = assessment.LoadDescriber(filepath.Join(repoRoot, "scenarios", "ai-gateway"))
	}
	if d.MaturitySpec == nil && repoRoot != "" {
		spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", "ai-gateway"))
		if err != nil && logger != nil {
			logger.Printf("conformance: maturity assessment disabled: %v", err)
		}
		d.MaturitySpec = spec
	}
	handler := NewConnectHandler(d)
	connectPath, connectHandler := conformanceconnect.NewConformanceServiceHandler(NewConnectHandler(d))
	sharedPath, sharedHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(assessment.Serve(handler, describer))
	return module.Module{
		Name: "conformance",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
			connectx.RegisterServices(r, connectx.ServiceMount{Path: sharedPath, Handler: sharedHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
