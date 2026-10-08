package workflows

import (
	"log"

	"workflow-health/internal/module"
	internalsearch "workflow-health/internal/search"
	internalvalidation "workflow-health/internal/validation"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/workflow-health/v1/workflows"
	workflowsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/workflow-health/v1/workflows/workflows_v1connect"
)

var ProtoFile = workflowsv1.File_workflow_health_v1_workflows_workflows_proto

func Module(logger *log.Logger) module.Module {
	connectPath, connectHandler := workflowsconnect.NewWorkflowSearchServiceHandler(NewConnectHandler(Deps{
		Logger: logger,
		Engine: internalvalidation.NewEngine(),
	}))
	return module.Module{
		Name: "workflows",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "workflows_search",
		Path:        workflowsconnect.WorkflowSearchServiceSearchWorkflowsProcedure,
		Method:      "POST",
		Summary:     "Search scenario workflow assets",
		Description: "Scans scenario-owned BAS workflow assets and returns typed workflow.flow, workflow.test, and workflow.fragment leaves with deterministic ranking and safety guardrails.",
		Category:    "workflows",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "query": "string", "types": "[]string", "include_fragments": "bool", "limit": "int32"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "query": "string", "results": "[]WorkflowSearchResult", "total": "int32"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
		Examples: []module.Example{
			{Name: "Search safe flows", Curl: "curl http://localhost:${API_PORT}/vrooli.workflow_health.v1.workflows.WorkflowSearchService/SearchWorkflows -H 'Content-Type: application/json' -d '{\"scenario\":\"workflow-health\",\"query\":\"run workflow\",\"types\":[\"" + internalsearch.LeafTypeFlow + "\"]}'"},
		},
	},
}
