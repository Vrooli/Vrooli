// Package journal exposes append-only event reads at the transport edge.
package journal

import (
	"token-economy/internal/journal"
	"token-economy/internal/module"

	"github.com/gorilla/mux"
	accessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access/accessv1connect"
)

// Endpoints remains empty because journal mutation is an internal Appender
// seam. Holder-scoped read RPCs land with authenticated holder isolation.
var Endpoints = []module.EndpointDescriptor{
	{ID: "journal_events_list", Path: accessconnect.MinterServiceListJournalEventsProcedure, Method: "POST", Summary: "List journal events", Description: "Lists append-only events for one holder and token type.", Category: "journal"},
	{ID: "journal_balance_show", Path: accessconnect.MinterServiceShowBalanceProcedure, Method: "POST", Summary: "Show projected balance", Description: "Projects the balance from journal events.", Category: "journal"},
	{ID: "journal_export", Path: accessconnect.MinterServiceExportJournalProcedure, Method: "POST", Summary: "Export journal", Description: "Returns the stable generated journal response for export.", Category: "journal"},
	{ID: "journal_reverse", Path: accessconnect.MinterServiceReverseEventProcedure, Method: "POST", Summary: "Reverse journal event", Description: "Appends one idempotent compensating event while preserving the original.", Category: "journal"},
}

// Module registers the domain without exposing unauthenticated journal reads.
func Module() module.Module {
	return module.Module{Name: "journal", Mount: func(*mux.Router) {}, Endpoints: Endpoints}
}

// Schema re-exports the domain-owned schema for the central boot registry.
func Schema() string { return journal.Schema() }
