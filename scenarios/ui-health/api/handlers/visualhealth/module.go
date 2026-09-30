package visualhealth

import (
	"log"

	"ui-health/internal/module"
	vhdomain "ui-health/internal/visualhealth"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
	visualconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth/visualhealth_v1connect"
)

var ProtoFile = visualpb.File_ui_health_v1_visualhealth_visualhealth_proto

func Module(logger *log.Logger) module.Module {
	_ = logger
	connectPath, connectHandler := visualconnect.NewVisualHealthServiceHandler(NewConnectHandler(vhdomain.DefaultAnalyzer()))
	return module.Module{
		Name: "visualhealth",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "visualhealth_analyze_artifacts",
		Path:        visualconnect.VisualHealthServiceAnalyzeArtifactsProcedure,
		Method:      "POST",
		Summary:     "Analyze browser UI artifacts for generic visual health",
		Description: "Runs ui-health's visual-health analyzer over screenshots, DOM, layout, console, network, and page-error artifacts.",
		Category:    "visualhealth",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string", "run_id": "string", "steps": "VisualStepArtifact[]"},
		},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"verdict": "string", "findings": "VisualFinding[]"}},
		Examples: []module.Example{
			{Name: "Analyze artifacts", Curl: "curl http://localhost:${API_PORT}/vrooli.ui_health.v1.visualhealth.VisualHealthService/AnalyzeArtifacts -H 'Content-Type: application/json' -d '{\"scenario\":\"demo\"}'"},
		},
	},
	{
		ID:          "visualhealth_compare_artifacts",
		Path:        visualconnect.VisualHealthServiceCompareArtifactsProcedure,
		Method:      "POST",
		Summary:     "Compare two sets of visual artifacts",
		Description: "Compares baseline and current screenshot artifacts and returns neutral per-page visual deltas.",
		Category:    "visualhealth",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string", "base_run_id": "string", "current_run_id": "string", "base": "CompareArtifact[]", "current": "CompareArtifact[]"},
		},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"deltas": "VisualDelta[]"}},
		Examples: []module.Example{
			{Name: "Compare artifacts", Curl: "curl http://localhost:${API_PORT}/vrooli.ui_health.v1.visualhealth.VisualHealthService/CompareArtifacts -H 'Content-Type: application/json' -d '{\"scenario\":\"demo\"}'"},
		},
	},
	{
		ID:          "visualhealth_list_rules",
		Path:        visualconnect.VisualHealthServiceListRulesProcedure,
		Method:      "POST",
		Summary:     "List visual-health analyzer rules",
		Description: "Returns the generic visual-health rules owned by ui-health.",
		Category:    "visualhealth",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"rules": "VisualRule[]"}},
		Examples: []module.Example{
			{Name: "List rules", Curl: "curl http://localhost:${API_PORT}/vrooli.ui_health.v1.visualhealth.VisualHealthService/ListRules -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
