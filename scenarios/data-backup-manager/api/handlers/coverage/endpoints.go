package coverage

import (
	"data-backup-manager/internal/module"

	coverageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/coverage/coverage_v1connect"
)

// Endpoints is the machine-readable description of the coverage module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so adding or renaming an RPC in coverage.proto breaks this file at
// compile time. The global coverage gate asserts every RPC is either bound in
// cli/manifest.json or explicitly omitted there.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "coverage_report",
		Path:        coverageconnect.CoverageServiceGetCoverageReportProcedure,
		Method:      "POST",
		Summary:     "Get the backup coverage report",
		Description: "Derives, live, the first-real-backup readiness picture: registered targets annotated with planned/backed-up/verified state, the non-sensitive discovered targets recommended for default coverage but not yet registered, and the sensitive (credential/token) suggestions held for review. Composes the discovery, targets, plans, runs and restores seams; reads no file contents and persists nothing.",
		Category:    "coverage",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"report": "CoverageReport"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Suggestion scan or catalog read failure"},
		},
		Examples: []module.Example{
			{Name: "Get coverage report", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.coverage.CoverageService/GetCoverageReport -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "coverage_accept_defaults",
		Path:        coverageconnect.CoverageServiceAcceptDefaultTargetsProcedure,
		Method:      "POST",
		Summary:     "Accept default coverage targets",
		Description: "Bulk-registers every non-sensitive discovered durable target so default coverage protects all known regenerable-once state. Sensitive credential/token suggestions are skipped unless include_sensitive is set. dry_run registers nothing and reports what would be registered. Idempotent — already-registered targets are not re-suggested, so re-running accepts nothing new. Registration stores locators only; file contents are never read.",
		Category:    "coverage",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"include_sensitive": "bool (default false)",
				"dry_run":           "bool (default false)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"accepted":          "array<AcceptedTarget>",
				"skipped_sensitive": "array<SuggestedTarget>",
				"failed":            "array<AcceptError>",
				"dry_run":           "boolean",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Suggestion scan failure before any registration"},
		},
		Examples: []module.Example{
			{Name: "Preview default acceptance", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.coverage.CoverageService/AcceptDefaultTargets -H 'Content-Type: application/json' -d '{\"dryRun\":true}'"},
			{Name: "Register non-sensitive defaults", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.coverage.CoverageService/AcceptDefaultTargets -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
