package audit

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"tunnel-manager/internal/clock"
	"tunnel-manager/internal/module"

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
// (the RoutesReader seam) and the scenarios filesystem tree through the
// scenarios root resolved by resolveScenariosRoot.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	return ModuleWithService(internalaudit.NewService(nil, resolveScenariosRoot(logger)), logger)
}

func ModuleWithRoutes(routes internalaudit.RoutesReader, logger *log.Logger) module.Module {
	svc := internalaudit.NewService(routes, resolveScenariosRoot(logger))
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

// resolveScenariosRoot determines the directory that holds scenario folders
// (each with a .vrooli/service.json). Resolution order:
//
//  1. The VROOLI_SCENARIOS_ROOT env var, when set — the explicit override the
//     lifecycle system can inject.
//  2. Otherwise, walk up from the current working directory looking for a
//     `scenarios` directory (the repo layout puts this scenario under
//     <repo>/scenarios/tunnel-manager/api). The found `scenarios` dir is the
//     root every scenario hangs off.
//  3. As a last resort, "scenarios" relative to the cwd, so audit findings
//     degrade to missing_scenario rather than panicking when the layout is
//     unexpected.
func resolveScenariosRoot(logger *log.Logger) string {
	if root := strings.TrimSpace(os.Getenv("VROOLI_SCENARIOS_ROOT")); root != "" {
		return root
	}
	if cwd, err := os.Getwd(); err == nil {
		for dir := cwd; ; {
			candidate := filepath.Join(dir, "scenarios")
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	if logger != nil {
		logger.Printf("audit.resolveScenariosRoot: could not locate a scenarios directory; falling back to ./scenarios (set VROOLI_SCENARIOS_ROOT to override)")
	}
	return "scenarios"
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
