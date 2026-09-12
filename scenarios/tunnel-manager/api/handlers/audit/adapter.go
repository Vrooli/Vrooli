package audit

import (
	internalaudit "tunnel-manager/internal/audit"

	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/audit"
)

// domainToProto converts an internal audit.PortAuditResult into the wire shape
// the audit proto declares. Lives in the handler package by intent — the
// conversion is mechanical and only used at the transport edge.
func domainToProto(r internalaudit.PortAuditResult) *auditv1.PortAuditResult {
	return &auditv1.PortAuditResult{
		Subdomain:    r.Subdomain,
		Scenario:     r.Scenario,
		ExpectedPort: int32(r.ExpectedPort),
		ActualPort:   int32(r.ActualPort),
		Status:       statusToProto(r.Status),
		Detail:       r.Detail,
	}
}

// statusToProto maps the domain status enum to the proto enum. An unknown
// status maps to AUDIT_STATUS_UNSPECIFIED.
func statusToProto(s internalaudit.AuditStatus) auditv1.AuditStatus {
	switch s {
	case internalaudit.StatusCompliant:
		return auditv1.AuditStatus_AUDIT_STATUS_COMPLIANT
	case internalaudit.StatusMismatch:
		return auditv1.AuditStatus_AUDIT_STATUS_MISMATCH
	case internalaudit.StatusMissingScenario:
		return auditv1.AuditStatus_AUDIT_STATUS_MISSING_SCENARIO
	case internalaudit.StatusMissingPort:
		return auditv1.AuditStatus_AUDIT_STATUS_MISSING_PORT
	default:
		return auditv1.AuditStatus_AUDIT_STATUS_UNSPECIFIED
	}
}
