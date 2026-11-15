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
}
