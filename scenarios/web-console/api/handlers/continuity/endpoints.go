package continuity

import (
	continuityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/continuity/continuity_v1connect"
	"web-console/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "continuity_integrity", Path: continuityconnect.ContinuityServiceIntegrityProcedure, Method: "POST", Summary: "Audit conversation continuity", Description: "Read-only counts and a deterministic generation for local session, transcript, checkpoint, and workspace evidence.", Category: "continuity"},
	{ID: "continuity_reconcile", Path: continuityconnect.ContinuityServiceReconcileProcedure, Method: "POST", Summary: "Plan or apply catalog reconciliation", Description: "Dry-run by default; apply=true writes only the Web Console catalog projection.", Category: "continuity"},
	{ID: "continuity_catalog_list", Path: continuityconnect.ContinuityServiceListCatalogProcedure, Method: "POST", Summary: "List canonical conversation records", Description: "Exports stable Web Console source identities and aliases for local recovery and Agent Manager import.", Category: "continuity"},
	{ID: "continuity_search", Path: continuityconnect.ContinuityServiceSearchProcedure, Method: "POST", Summary: "Search all local conversation states", Description: "Deterministic exact-text search across live, archived, recoverable, and orphaned retained events.", Category: "continuity"},
	{ID: "continuity_receipt_get", Path: continuityconnect.ContinuityServiceGetReceiptProcedure, Method: "POST", Summary: "Inspect a lifecycle receipt", Description: "Returns durable status for a lifecycle or reconciliation operation.", Category: "continuity"},
	{ID: "continuity_rollback", Path: continuityconnect.ContinuityServiceRollbackProcedure, Method: "POST", Summary: "Rollback a catalog reconciliation", Description: "Restores the catalog projection preimage captured by a manifest; durable conversation evidence is untouched.", Category: "continuity"},
	{ID: "continuity_publish", Path: continuityconnect.ContinuityServicePublishProcedure, Method: "POST", Summary: "Publish queued continuity records", Description: "Drains a bounded retry-safe publication queue to Agent Manager; local recovery is independent of downstream availability.", Category: "continuity"},
}
