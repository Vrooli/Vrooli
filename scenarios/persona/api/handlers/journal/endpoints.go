package journal

import (
	"persona/internal/module"

	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/journal/journal_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "journal_append", Path: journalconnect.JournalServiceAppendProcedure, Method: "POST", Summary: "Append an attributable journal entry", Category: "journal"},
	{ID: "journal_list", Path: journalconnect.JournalServiceListProcedure, Method: "POST", Summary: "List journal entries", Category: "journal"},
}
