package audit

import (
	"log"

	"tunnel-manager/internal/module"
	"tunnel-manager/internal/scenarioroot"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/audit/audit_v1connect"

	internalaudit "tunnel-manager/internal/audit"
)

// Module returns the audit domain's contribution to the API: the generated
// Connect-RPC AuditService handler. The audit domain owns NO table, so there
// is intentionally no Schema() here — it must not be registered in
// modules.AllSchemas. The center (server.New) does not change; adding this
// domain is one Module() call in main.go plus one row each in
// modules.AllEndpoints and modules.AllProtoFiles (but NOT AllSchemas).
//
// The audit service reads the routes manifest through internalroutes.Service
// (the RoutesReader seam) and resolves service manifests through the shared
// repository-contract path seam.
func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger) module.Module {
	return ModuleWithService(internalaudit.NewServiceWithScenarioFiles(nil, resolveScenarioFiles(logger)), logger)
}

func ModuleWithRoutes(routes internalaudit.RoutesReader, logger *log.Logger) module.Module {
	svc := internalaudit.NewServiceWithScenarioFiles(routes, resolveScenarioFiles(logger))
	return ModuleWithService(svc, logger)
}

func ModuleWithService(svc internalaudit.Service, logger *log.Logger) module.Module {
	connectPath, connectHandler := auditconnect.NewAuditServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "audit",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func resolveScenarioFiles(logger *log.Logger) *scenarioroot.Resolver {
	resolver := scenarioroot.New()
	if _, err := resolver.ScenariosRoot(); err != nil && logger != nil {
		logger.Printf("audit scenario path resolver unavailable: %v", err)
	}
	return resolver
}

// Endpoints is the machine-readable description of the audit module's public
// surface. The Connect-RPC method path references the generated *Procedure
// constant from auditconnect, so adding or renaming an RPC in audit.proto
// breaks this file at compile time. TestProtoConnectParity
// (api/internal/modules/registry_test.go) asserts every rpc has exactly one
// entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "audit_run",
		Path:        auditconnect.AuditServiceRunAuditProcedure,
		Method:      "POST",
		Summary:     "Run a port-compliance audit",
		Description: "Compares each enabled manifest route's expected local_port against the UI port declared in the scenario's service.json, returning one finding per route plus a count of violations (status != compliant).",
		Category:    "audit",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"results":         "array<PortAuditResult>",
				"violation_count": "int32 (count of results with status != compliant)",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Manifest read failure"},
		},
		Examples: []module.Example{
			{Name: "Run audit", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.audit.AuditService/RunAudit -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
