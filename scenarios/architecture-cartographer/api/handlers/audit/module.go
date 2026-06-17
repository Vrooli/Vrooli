package audit

import (
	"architecture-cartographer/internal/audit"
	"architecture-cartographer/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/audit/audit_v1connect"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

// Module returns the audit domain's contribution to the API router.
func Module(svc audit.Service) module.Module {
	h := NewHandler(svc)
	auditPattern, auditHandler := audit_v1connect.NewAuditServiceHandler(h)
	validationPattern, validationHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(h)
	return module.Module{
		Name: "audit",
		Mount: func(r *mux.Router) {
			r.PathPrefix(auditPattern).Handler(auditHandler)
			r.PathPrefix(validationPattern).Handler(validationHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the audit domain's SQL contribution. The audit domain
// is stateless (pure orchestrator), so this is empty.
func Schema() string { return "" }
