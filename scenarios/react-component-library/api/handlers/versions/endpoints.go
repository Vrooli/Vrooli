package versions

import (
	versionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions/versions_v1connect"

	"react-component-library/internal/module"
)

// Endpoints describes the versions module's public surface.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "versions_list",
		Path:        versionsconnect.VersionsServiceListVersionsProcedure,
		Method:      "POST",
		Summary:     "List recorded versions for a component",
		Description: "Returns version rows newest-first for the given component_id.",
		Category:    "versions",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"component_id": "string",
				"limit":        "int32 (max rows, 0 = server default)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"versions": "array<Version>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
	},
	{
		ID:          "versions_get",
		Path:        versionsconnect.VersionsServiceGetVersionProcedure,
		Method:      "POST",
		Summary:     "Get a specific recorded version",
		Description: "Returns the row matching (component_id, version). Set include_content=true to receive the full recorded body.",
		Category:    "versions",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"component_id":    "string",
				"version":         "string",
				"include_content": "bool",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"version": "Version",
				"content": "string (populated only when include_content was true)",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No version with that component_id/version pair"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
	},
	{
		ID:          "versions_diff",
		Path:        versionsconnect.VersionsServiceDiffVersionsProcedure,
		Method:      "POST",
		Summary:     "Diff two versions or a version and an adopted copy",
		Description: "Returns paired left/right diff cells (equal/add/remove/empty). `from` and `to` accept either a @version value or `adoption:<id>` to compare against an adopted copy on disk.",
		Category:    "versions",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"component_id": "string",
				"from":         "string (version or adoption:<id>)",
				"to":           "string (version or adoption:<id>)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"rows":       "array<DiffRow>",
				"additions":  "int32",
				"removals":   "int32",
				"from_label": "string",
				"to_label":   "string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing from/to or invalid adoption reference"},
			{Status: 404, Code: "not_found", Description: "Referenced version or adoption does not exist"},
			{Status: 500, Code: "internal", Description: "Repository or filesystem failure"},
		},
	},
}

func init() {
	Endpoints = append(Endpoints,
		module.EndpointDescriptor{ID: "versions_progression", Path: versionsconnect.VersionLifecycleServiceListVersionLedgerProcedure, Method: "POST", Summary: "List durable version progression ledger", Category: "versions"},
		module.EndpointDescriptor{ID: "versions_retire_candidates", Path: versionsconnect.VersionLifecycleServiceListRetireCandidatesProcedure, Method: "POST", Summary: "List safe version retirement candidates", Category: "versions"},
		module.EndpointDescriptor{ID: "versions_plan_cleanup", Path: versionsconnect.VersionLifecycleServicePlanCleanupProcedure, Method: "POST", Summary: "Plan safe batch version cleanup", Category: "versions"},
		module.EndpointDescriptor{ID: "versions_cleanup", Path: versionsconnect.VersionLifecycleServiceCleanupVersionsProcedure, Method: "POST", Summary: "Apply confirmed batch version cleanup", Category: "versions"},
		module.EndpointDescriptor{ID: "versions_cleanup_draft", Path: versionsconnect.VersionLifecycleServiceCleanupDraftProcedure, Method: "POST", Summary: "Discard a confirmed abandoned draft", Category: "versions"},
		module.EndpointDescriptor{ID: "versions_deprecate", Path: versionsconnect.VersionLifecycleServiceDeprecateVersionProcedure, Method: "POST", Summary: "Deprecate a version", Category: "versions"},
		module.EndpointDescriptor{ID: "versions_archive", Path: versionsconnect.VersionLifecycleServiceArchiveVersionProcedure, Method: "POST", Summary: "Archive a version", Category: "versions"},
		module.EndpointDescriptor{ID: "versions_retire", Path: versionsconnect.VersionLifecycleServiceRetireVersionProcedure, Method: "POST", Summary: "Retire an unreferenced version", Category: "versions"},
		module.EndpointDescriptor{ID: "versions_materialize", Path: versionsconnect.VersionLifecycleServiceMaterializeVersionProcedure, Method: "POST", Summary: "Materialize version bytes from the durable mirror", Category: "versions"},
		module.EndpointDescriptor{ID: "versions_reconcile_presence", Path: versionsconnect.VersionLifecycleServiceReconcilePresenceProcedure, Method: "POST", Summary: "Reconcile materialized and evicted version presence", Category: "versions"},
		module.EndpointDescriptor{ID: "versions_export_archive", Path: versionsconnect.VersionLifecycleServiceExportArchiveProcedure, Method: "POST", Summary: "Export the version ledger archive", Category: "versions"},
		module.EndpointDescriptor{ID: "versions_import_archive", Path: versionsconnect.VersionLifecycleServiceImportArchiveProcedure, Method: "POST", Summary: "Import the version ledger archive", Category: "versions"},
		module.EndpointDescriptor{ID: "versions_doctor", Path: versionsconnect.VersionLifecycleServiceDoctorProcedure, Method: "POST", Summary: "Check evicted version mirror integrity", Category: "versions"},
	)
}
