package audit

import (
	"log"

	"performance-health/internal/capture"
	"performance-health/internal/module"
	"performance-health/internal/readiness"

	"github.com/gorilla/mux"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/audit/audit_v1connect"
)

// ProtoFile is the FileDescriptor backing the Connect-mounted AuditService.
var ProtoFile = auditv1.File_performance_health_v1_audit_audit_proto

// Module mounts the AuditService backed by the real capture conductor: a
// readiness-driven tier decision, a profile-mode build controller, and a Connect
// client to Browser Automation Studio's perf capture. Captures skip cleanly when
// impossible (no browser / no UI / headless / BAS unreachable).
func Module(logger *log.Logger, repoRoot string) module.Module {
	svc := capture.NewService(&capture.BASConnectClient{}, &capture.CLIBuildController{}).
		WithFlowResolver(&capture.FileFlowResolver{RepoRoot: repoRoot})
	tierer := readiness.NewService(readiness.NewCodeFactsClient(repoRoot))
	handler := NewHandler(svc, tierer, logger)
	path, connectHandler := auditconnect.NewAuditServiceHandler(handler)
	return module.Module{
		Name: "audit",
		Mount: func(r *mux.Router) {
			r.PathPrefix(path).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the empty schema: audit owns no database tables.
func Schema() string { return "" }

// Endpoints is the static endpoint metadata for codegen and the parity test.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "audit_run_audit",
		Path:        auditconnect.AuditServiceRunAuditProcedure,
		Method:      "POST",
		Summary:     "Orchestrate a profile-mode perf capture of a scenario",
		Description: "Restarts the scenario in profile build mode, drives a BAS perf capture on a chosen interaction, and restores the default build — cleanly skipping when capture is impossible (no browser / no UI / headless).",
		Category:    "audit",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "workflow": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "outcome": "AuditOutcome", "tier": "CaptureTier", "trace_artifact": "string", "web_vitals_artifact": "string", "reason": "string"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario is missing"}, {Status: 500, Code: "internal", Description: "Capture orchestration failure"}},
	},
}
