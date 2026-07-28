package validation

import (
	"context"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/maturity-go/assessment"
	vroolicli "github.com/vrooli/vrooli-cli-go"

	"proto-health/internal/codefacts"
	"proto-health/internal/module"
	"proto-health/internal/protosurface"
	internal "proto-health/internal/validation"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/validation/validation_v1connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

var (
	ProtoFile                   = validationv1.File_proto_health_v1_validation_validation_proto
	ScenarioValidationProtoFile = scenariovalidationv1.File_scenario_validation_v1_validation_proto
)

func Module(logger *log.Logger, repoRoot string) module.Module {
	loader, err := protosurface.NewDescriptorLoaderFromFile(
		repoRoot,
		filepath.Join(repoRoot, "packages", "proto", "gen", "descriptor", "image.binpb"),
	)
	if err != nil {
		logger.Fatalf("proto descriptor loader: %v", err)
	}
	// DescribeProvider answers readiness from this provider's own descriptor,
	// so a readiness probe no longer costs a full target analysis. A load
	// failure yields the zero Describer, which reports Unimplemented and makes
	// consumers fall back to the legacy probe.
	describer, _ := assessment.LoadDescriber(filepath.Join(repoRoot, "scenarios", "proto-health"))
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", "proto-health"))
	if err != nil {
		logger.Fatalf("validation: load maturity spec: %v", err)
	}
	// Capture host facts once; they do not change during the process lifetime.
	// A failure (CLI unavailable) is non-fatal — the metrics collector backfills
	// os/arch/num_cpu from the stdlib, leaving richer facts unset.
	environment, err := vroolicli.New().HostCaptureEnvironment(context.Background())
	if err != nil {
		logger.Printf("validation: host inventory unavailable, metrics environment limited to stdlib baseline: %v", err)
		environment = nil
	}
	catalog, err := internal.NewFindingCatalog(spec)
	if err != nil {
		logger.Fatalf("validation: build finding catalog: %v", err)
	}
	validator := internal.New(internal.Deps{
		Loader:         loader,
		GenSyncChecker: internal.NewManifestVerifier(repoRoot),
		RepoRoot:       repoRoot,
		CodeFacts:      codefacts.NewClient(nil, http.DefaultClient),
		Catalog:        catalog,
	})
	handler := NewConnectHandler(Deps{
		Logger:       logger,
		Validator:    validator,
		MaturitySpec: spec,
		Environment:  environment,
	})
	protoPath, protoHandler := validationconnect.NewProtoHealthServiceHandler(handler)
	validationPath, validationHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(assessment.Serve(handler, describer))
	return module.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(
				r,
				connectx.ServiceMount{Path: validationPath, Handler: validationHandler},
				connectx.ServiceMount{Path: protoPath, Handler: protoHandler},
			)
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "validation_validate_scenario",
		Path:        scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate a scenario's proto contracts",
		Description: "Validates one scenario's Protocol Buffer structure, annotations, declared transport world, REST exception payload declarations, and conservative unused-message hints without computing fleet dependency graphs.",
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
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario id or no proto files found for the scenario"},
		},
		Examples: []module.Example{
			{Name: "Validate proto-health", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/ValidateScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"proto-health\"}'"},
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
		Summary:     "Preview proto validation fixes",
		Description: "Returns unimplemented because Proto Health currently has no deterministic fixer.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "finding_code": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"status": "string", "message": "string"}},
		Errors:      []module.ErrorDesc{{Status: 501, Code: "unimplemented", Description: "Proto Health has no deterministic fixer"}},
	},
	{
		ID:          "validation_apply_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure,
		Method:      "POST",
		Summary:     "Apply proto validation fixes",
		Description: "Returns unimplemented because Proto Health currently has no deterministic fixer.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "finding_code": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"status": "string", "message": "string"}},
		Errors:      []module.ErrorDesc{{Status: 501, Code: "unimplemented", Description: "Proto Health has no deterministic fixer"}},
	},
	{
		ID:          "validation_describe_scenario_protos",
		Path:        validationconnect.ProtoHealthServiceDescribeScenarioProtosProcedure,
		Method:      "POST",
		Summary:     "Describe a scenario's proto surface",
		Description: "Returns the scenario-scoped proto surface fact: files, services, RPCs, messages, imports, annotations, REST exception payload declarations, and transport-world summary.",
		Category:    "validation",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string (required, scenario id)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"surface": "ProtoSurface"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario id or no proto files found for the scenario"},
		},
		Examples: []module.Example{
			{Name: "Describe proto-health", Curl: "curl http://localhost:${API_PORT}/vrooli.proto_health.v1.validation.ProtoHealthService/DescribeScenarioProtos -H 'Content-Type: application/json' -d '{\"scenario\":\"proto-health\"}'"},
		},
	},
	{
		ID:          "validation_describe_scenarios_protos",
		Path:        validationconnect.ProtoHealthServiceDescribeScenariosProtosProcedure,
		Method:      "POST",
		Summary:     "Describe multiple scenarios' proto surfaces",
		Description: "Returns independent proto-surface fact results for many or all scenarios so downstream fleet graph consumers can avoid N per-scenario calls.",
		Category:    "validation",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenarios":        "array<string> (optional, empty means all descriptor scenarios)",
				"limit":            "integer (optional, 0 means no limit, max 500)",
				"stability_filter": "string (optional, filters returned surface files by @stability)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"results": "array<ProtoSurfaceResult>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Invalid limit or descriptor scenario discovery failure"},
		},
		Examples: []module.Example{
			{Name: "Describe selected scenarios", Curl: "curl http://localhost:${API_PORT}/vrooli.proto_health.v1.validation.ProtoHealthService/DescribeScenariosProtos -H 'Content-Type: application/json' -d '{\"scenarios\":[\"proto-health\",\"code-facts\"]}'"},
		},
	},
}
