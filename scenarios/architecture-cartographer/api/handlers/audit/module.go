package audit

import (
	"context"
	"log"
	"path/filepath"

	"architecture-cartographer/internal/audit"
	"architecture-cartographer/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/maturity-go/assessment"
	vroolicli "github.com/vrooli/vrooli-cli-go"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/audit/audit_v1connect"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

// Module returns the audit domain's contribution to the API router.
func Module(svc audit.Service, repoRoot string, logger *log.Logger) module.Module {
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", "architecture-cartographer"))
	if err != nil && logger != nil {
		logger.Printf("audit: maturity assessment disabled: %v", err)
	}
	// Capture host facts once; they do not change during the process lifetime.
	// A failure (CLI unavailable) is non-fatal — the metrics collector backfills
	// os/arch/num_cpu from the stdlib, leaving richer facts unset.
	environment, envErr := vroolicli.New().HostCaptureEnvironment(context.Background())
	if envErr != nil {
		if logger != nil {
			logger.Printf("audit: host inventory unavailable, metrics environment limited to stdlib baseline: %v", envErr)
		}
		environment = nil
	}
	h := NewHandler(HandlerDeps{Svc: svc, MaturitySpec: spec, Environment: environment})
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
