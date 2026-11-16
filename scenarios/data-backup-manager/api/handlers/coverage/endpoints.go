package coverage

import (
	"data-backup-manager/internal/modulecontract"

	coverageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/coverage/coverage_v1connect"
)

// Endpoints is the machine-readable description of the coverage module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so adding or renaming an RPC in coverage.proto breaks this file at
// compile time. The global coverage gate asserts every RPC is either bound in
// cli/manifest.json or explicitly omitted there.
var Endpoints = []modulecontract.EndpointDescriptor{
	{
		ID:          "coverage_report",
		Path:        coverageconnect.CoverageServiceGetCoverageReportProcedure,
		Method:      "POST",
		Summary:     "Get the backup coverage report",
		Description: "Derives, live, the first-real-backup readiness picture: registered targets annotated with enabled destination bindings, per-copy backup and scratch-restore receipts, and observed physical-independence warnings; the non-sensitive discovered targets recommended for default coverage; and sensitive suggestions held for review. Copy count alone does not imply off-host placement. Composes discovery, targets, plans, runs and restores; reads no file contents and persists nothing.",
		Category:    "coverage",
		Request: &modulecontract.Schema{
			Type:       "object",
			Properties: map[string]string{},
		},
		Response: &modulecontract.Schema{
			Type:       "object",
			Properties: map[string]string{"report": "CoverageReport"},
		},
		Errors: []modulecontract.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Suggestion scan or catalog read failure"},
		},
		Examples: []modulecontract.Example{
			{Name: "Get coverage report", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.coverage.CoverageService/GetCoverageReport -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "coverage_accept_defaults",
		Path:        coverageconnect.CoverageServiceAcceptDefaultTargetsProcedure,
		Method:      "POST",
		Summary:     "Accept default coverage targets",
		Description: "Bulk-registers non-sensitive discovered durable targets. Sensitive credential/token suggestions always require the verified Notification Hub review; setting include_sensitive is rejected. dry_run registers nothing and reports what would be registered. Idempotent — already-registered targets are not re-suggested, so re-running accepts nothing new. Registration stores locators only; file contents are never read.",
		Category:    "coverage",
		Request: &modulecontract.Schema{
			Type: "object",
			Properties: map[string]string{
				"include_sensitive": "bool (default false)",
				"dry_run":           "bool (default false)",
			},
		},
		Response: &modulecontract.Schema{
			Type: "object",
			Properties: map[string]string{
				"accepted":          "array<AcceptedTarget>",
				"skipped_sensitive": "array<SuggestedTarget>",
				"failed":            "array<AcceptError>",
				"dry_run":           "boolean",
			},
		},
		Errors: []modulecontract.ErrorDesc{
			{Status: 412, Code: "failed_precondition", Description: "Sensitive targets require the verified Notification Hub review"},
			{Status: 500, Code: "internal", Description: "Suggestion scan failure before any registration"},
		},
		Examples: []modulecontract.Example{
			{Name: "Preview default acceptance", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.coverage.CoverageService/AcceptDefaultTargets -H 'Content-Type: application/json' -d '{\"dryRun\":true}'"},
			{Name: "Register non-sensitive defaults", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.coverage.CoverageService/AcceptDefaultTargets -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "coverage_request_sensitive_review",
		Path:        coverageconnect.CoverageServiceRequestSensitiveReviewProcedure,
		Method:      "POST",
		Summary:     "Request one verified review of sensitive coverage suggestions",
		Description: "Raises one Notification Hub decision for the current sensitive suggestions only when the signed verified-human callback is configured. Answers are revalidated against current candidate identity before any protected target is registered; every terminal result is recorded and notified.",
		Category:    "coverage",
		Request:     &modulecontract.Schema{Type: "object", Properties: map[string]string{}},
		Response:    &modulecontract.Schema{Type: "object", Properties: map[string]string{"ask_id": "string", "candidate_count": "int32", "pending": "boolean"}},
		Errors: []modulecontract.ErrorDesc{
			{Status: 503, Code: "unavailable", Description: "Signed verified-human callback is not configured"},
			{Status: 412, Code: "failed_precondition", Description: "No current sensitive suggestions are available"},
			{Status: 500, Code: "internal", Description: "Coverage scan or decision publication failure"},
		},
		Examples: []modulecontract.Example{
			{Name: "Request sensitive coverage review", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.coverage.CoverageService/RequestSensitiveReview -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "coverage_get_sensitive_policy",
		Path:        coverageconnect.CoverageServiceGetSensitivePolicyProcedure,
		Method:      "POST",
		Summary:     "Show the standing rule for places that hold secrets",
		Description: "Reads the persisted standing rule: whether it is on, which verified answer consented (who, through which identity, when, and the Hub decision), what it added since the last weekly note, what it keeps out, and its history.",
		Category:    "coverage",
		Request:     &modulecontract.Schema{Type: "object", Properties: map[string]string{}},
		Response:    &modulecontract.Schema{Type: "object", Properties: map[string]string{"policy": "SensitivePolicy"}},
		Errors: []modulecontract.ErrorDesc{
			{Status: 503, Code: "unavailable", Description: "The policy store is not configured"},
			{Status: 500, Code: "internal", Description: "Policy store read failure"},
		},
		Examples: []modulecontract.Example{
			{Name: "Show the standing rule", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.coverage.CoverageService/GetSensitivePolicy -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "coverage_revoke_sensitive_policy",
		Path:        coverageconnect.CoverageServiceRevokeSensitivePolicyProcedure,
		Method:      "POST",
		Summary:     "Turn off the standing rule for places that hold secrets",
		Description: "Ends the standing rule. Items it already added stay registered; new places that hold secrets wait for a verified decision again. Revoking only reduces what is added, so it needs no Hub round trip. The revocation is recorded with who and when.",
		Category:    "coverage",
		Request:     &modulecontract.Schema{Type: "object", Properties: map[string]string{"revoked_by": "string (optional)"}},
		Response:    &modulecontract.Schema{Type: "object", Properties: map[string]string{"policy": "SensitivePolicy"}},
		Errors: []modulecontract.ErrorDesc{
			{Status: 412, Code: "failed_precondition", Description: "There is no standing rule to revoke"},
			{Status: 503, Code: "unavailable", Description: "The policy store is not configured"},
		},
		Examples: []modulecontract.Example{
			{Name: "Turn the standing rule off", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.coverage.CoverageService/RevokeSensitivePolicy -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "coverage_exclude_sensitive_target",
		Path:        coverageconnect.CoverageServiceExcludeSensitiveTargetProcedure,
		Method:      "POST",
		Summary:     "Keep one item out of the standing rule",
		Description: "Records that the standing rule must never add the named owner/name item. Excluding only reduces what is added. An item that is already registered stays registered until its target is removed.",
		Category:    "coverage",
		Request:     &modulecontract.Schema{Type: "object", Properties: map[string]string{"owner": "string", "name": "string", "reason": "string", "excluded_by": "string (optional)"}},
		Response:    &modulecontract.Schema{Type: "object", Properties: map[string]string{"policy": "SensitivePolicy"}},
		Errors: []modulecontract.ErrorDesc{
			{Status: 412, Code: "failed_precondition", Description: "Owner, name or reason is missing"},
			{Status: 503, Code: "unavailable", Description: "The policy store is not configured"},
		},
		Examples: []modulecontract.Example{
			{Name: "Exclude one item", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.coverage.CoverageService/ExcludeSensitiveTarget -H 'Content-Type: application/json' -d '{\"owner\":\"grok\",\"name\":\"home\",\"reason\":\"Too large\"}'"},
		},
	},
	{
		ID:          "coverage_adopt_sensitive_consent",
		Path:        coverageconnect.CoverageServiceAdoptSensitiveConsentProcedure,
		Method:      "POST",
		Summary:     "Record the operator's free-text answer as the standing consent",
		Description: "Reads the named Hub ask back as this scenario's ask-reader principal. It must be this scenario's sensitive coverage decision, answered by a verified human in their own words (Other). The standing rule is then recorded with the Hub ask id, the answer time, the verified identity and the operator's text verbatim, applied to the current candidates, and the stale local review is closed. Nothing is raised at the Hub. A menu answer, an unanswered decision or another scenario's ask is refused.",
		Category:    "coverage",
		Request:     &modulecontract.Schema{Type: "object", Properties: map[string]string{"hub_ask_id": "string"}},
		Response:    &modulecontract.Schema{Type: "object", Properties: map[string]string{"policy": "SensitivePolicy", "registered_count": "int32", "failed_count": "int32"}},
		Errors: []modulecontract.ErrorDesc{
			{Status: 412, Code: "failed_precondition", Description: "The answer cannot stand as the consent"},
			{Status: 503, Code: "unavailable", Description: "The policy store or the Hub read-back is not available"},
		},
		Examples: []modulecontract.Example{
			{Name: "Adopt an answer", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.coverage.CoverageService/AdoptSensitiveConsent -H 'Content-Type: application/json' -d '{\"hubAskId\":\"ASK_ID\"}'"},
		},
	},
}
