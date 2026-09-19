package audit

import (
	"log"

	internalaudit "vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/audit/audit_v1connect"
)

// Module returns the audit domain's contribution to the API: the generated
// Connect-RPC AuditService read handler. The append-only store is constructed in
// main.go and shared as a Sink with the dispatch handler (write side) and as a
// Reader here (read side).
func Module(reader internalaudit.Reader, logger *log.Logger) module.Module {
	path, handler := auditconnect.NewAuditServiceHandler(NewConnectHandler(Deps{
		Reader: reader,
		Logger: logger,
	}))
	return module.Module{
		Name: "audit",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the audit domain's SQL contribution so the modules registry
// collects endpoint descriptors and schema from one handler package.
func Schema() string { return internalaudit.Schema() }
