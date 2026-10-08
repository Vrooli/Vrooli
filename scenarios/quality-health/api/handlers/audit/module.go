package audit

import (
	"fmt"
	"log"
	"path/filepath"

	internalaudit "quality-health/internal/audit"
	"quality-health/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/maturity-go/assessment"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit/audit_v1connect"
)

var ProtoFile = auditv1.File_quality_health_v1_audit_audit_proto

func Module(logger *log.Logger, repoRoot string) module.Module {
	svc := internalaudit.New(nil)
	spec, err := loadMaturitySpec(repoRoot)
	if err != nil && logger != nil {
		logger.Printf("audit: maturity assessment unavailable: %v", err)
	}
	connectPath, connectHandler := auditconnect.NewAuditServiceHandler(NewHandlerWithDeps(Deps{
		Service:      svc,
		Logger:       logger,
		MaturitySpec: spec,
	}))
	return module.Module{
		Name: "audit",
		Mount: func(r *mux.Router) {
			r.PathPrefix(connectPath).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

func loadMaturitySpec(repoRoot string) (*assessment.Spec, error) {
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", "quality-health"))
	if err != nil {
		return nil, fmt.Errorf("load quality-health descriptor maturity: %w", err)
	}
	return spec, nil
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
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"status": "string", "surfaces": "array<QualitySurface>", "findings": "array<QualityFinding>", "maturity": "MaturitySummary", "assessment": "common.v1.MaturityAssessment"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
	},
	{
		ID:       "audit_contracts_list",
		Path:     auditconnect.AuditServiceListContractsProcedure,
		Method:   "POST",
		Summary:  "List quality contracts",
		Category: "contracts",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"language": "string", "framework": "string", "surface_kind": "string", "rule_ids": "array<string>"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"contracts": "array<QualityContract>"}},
	},
	{
		ID:       "audit_explain_finding",
		Path:     auditconnect.AuditServiceExplainFindingProcedure,
		Method:   "POST",
		Summary:  "Explain a quality finding",
		Category: "explain",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"finding_id": "string", "rule_id": "string", "scenario": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"finding": "QualityFinding", "contract": "QualityContract", "next_steps": "array<string>"}},
	},
	{
		ID:       "audit_preview_fix_config",
		Path:     auditconnect.AuditServicePreviewFixConfigProcedure,
		Method:   "POST",
		Summary:  "Preview static quality config fixes",
		Category: "autofix",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"candidates": "array<AutofixCandidate>", "messages": "array<string>"}},
	},
	{
		ID:       "audit_apply_fix_config",
		Path:     auditconnect.AuditServiceApplyFixConfigProcedure,
		Method:   "POST",
		Summary:  "Apply static quality config fixes",
		Category: "autofix",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>", "apply": "bool"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"applied": "bool", "candidates": "array<AutofixCandidate>", "messages": "array<string>"}},
	},
}
