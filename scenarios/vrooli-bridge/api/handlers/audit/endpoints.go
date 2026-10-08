package audit

import (
	"vrooli-bridge/internal/module"

	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/audit/audit_v1connect"
)

// Endpoints is the machine-readable description of the audit module's public
// surface. The Connect-RPC method path references the generated *Procedure
// constant, so renaming the RPC in audit.proto breaks this at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "audit_list_records",
		Path:        auditconnect.AuditServiceListAuditRecordsProcedure,
		Method:      "POST",
		Summary:     "List audit records",
		Description: "Returns the append-only accountability trail newest-first — every dispatch and provisioning operation (actor/node/verb/args/outcome) — optionally filtered by node or run. Read-only; records are appended internally by the operations they record. Owner-gated.",
		Category:    "audit",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"node_id": "string",
			"run_id":  "string",
			"limit":   "int32",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"records": "array<AuditRecord>"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List audit records", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.audit.AuditService/ListAuditRecords -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"limit\":50}'"},
		},
	},
}
