package audit

import (
	"log"

	internalaudit "quality-health/internal/audit"
	"quality-health/internal/module"

	"github.com/gorilla/mux"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit/audit_v1connect"
)

var ProtoFile = auditv1.File_quality_health_v1_audit_audit_proto

func Module(logger *log.Logger) module.Module {
	svc := internalaudit.New(nil)
	connectPath, connectHandler := auditconnect.NewAuditServiceHandler(NewHandler(svc, logger))
	return module.Module{
		Name: "audit",
		Mount: func(r *mux.Router) {
			r.PathPrefix(connectPath).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "audit_quality",
		Path:        auditconnect.AuditServiceAuditQualityProcedure,
		Method:      "POST",
		Summary:     "Run static quality audit",
		Description: "Discovers scenario surfaces through Code Facts, evaluates static-quality contracts, optionally runs lint/type commands, and returns normalized findings and maturity.",
		Category:    "audit",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>", "surfaces": "array<string>"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"status": "string", "surfaces": "array<QualitySurface>", "findings": "array<QualityFinding>", "maturity": "MaturitySummary"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
		CLIMapping:  &module.CLIMapping{Command: "quality-health audit run", Args: []string{"<scenario>", "--json"}},
	},
	{
		ID:         "audit_contracts_list",
		Path:       auditconnect.AuditServiceListContractsProcedure,
		Method:     "POST",
		Summary:    "List quality contracts",
		Category:   "contracts",
		Request:    &module.Schema{Type: "object", Properties: map[string]string{"language": "string", "framework": "string", "surface_kind": "string", "rule_ids": "array<string>"}},
		Response:   &module.Schema{Type: "object", Properties: map[string]string{"contracts": "array<QualityContract>"}},
		CLIMapping: &module.CLIMapping{Command: "quality-health contracts list", Args: []string{"--json"}},
	},
	{
		ID:         "audit_explain_finding",
		Path:       auditconnect.AuditServiceExplainFindingProcedure,
		Method:     "POST",
		Summary:    "Explain a quality finding",
		Category:   "explain",
		Request:    &module.Schema{Type: "object", Properties: map[string]string{"finding_id": "string", "rule_id": "string", "scenario": "string"}},
		Response:   &module.Schema{Type: "object", Properties: map[string]string{"finding": "QualityFinding", "contract": "QualityContract", "next_steps": "array<string>"}},
		CLIMapping: &module.CLIMapping{Command: "quality-health explain finding", Args: []string{"<finding-id>", "--scenario", "<scenario>"}},
	},
	{
		ID:         "audit_preview_fix_config",
		Path:       auditconnect.AuditServicePreviewFixConfigProcedure,
		Method:     "POST",
		Summary:    "Preview static quality config fixes",
		Category:   "autofix",
		Request:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>"}},
		Response:   &module.Schema{Type: "object", Properties: map[string]string{"candidates": "array<AutofixCandidate>", "messages": "array<string>"}},
		CLIMapping: &module.CLIMapping{Command: "quality-health fix-config run", Args: []string{"<scenario>", "--dry-run"}},
	},
	{
		ID:         "audit_apply_fix_config",
		Path:       auditconnect.AuditServiceApplyFixConfigProcedure,
		Method:     "POST",
		Summary:    "Apply static quality config fixes",
		Category:   "autofix",
		Request:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>", "apply": "bool"}},
		Response:   &module.Schema{Type: "object", Properties: map[string]string{"applied": "bool", "candidates": "array<AutofixCandidate>", "messages": "array<string>"}},
		CLIMapping: &module.CLIMapping{Command: "quality-health fix-config apply", Args: []string{"<scenario>"}},
	},
}
