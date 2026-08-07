package journal

import (
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
	"source-ledger/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "journal_append", Path: journalconnect.JournalServiceAppendEntryProcedure, Method: "POST", Summary: "Append an immutable memory entry", Category: "journal", Request: &module.Schema{Type: "AppendEntryRequest"}, Response: &module.Schema{Type: "AppendEntryResponse"}, Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Missing body"}, {Status: 500, Code: "internal", Description: "Persistence failure"}}},
	{ID: "journal_get", Path: journalconnect.JournalServiceGetEntryProcedure, Method: "POST", Summary: "Get an immutable memory entry", Category: "journal", Request: &module.Schema{Type: "GetEntryRequest"}, Response: &module.Schema{Type: "GetEntryResponse"}, Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Missing id"}, {Status: 404, Code: "not_found", Description: "Entry does not exist"}}},
	{ID: "journal_list", Path: journalconnect.JournalServiceListEntriesProcedure, Method: "POST", Summary: "List immutable memory entries", Category: "journal", Request: &module.Schema{Type: "ListEntriesRequest"}, Response: &module.Schema{Type: "ListEntriesResponse"}, Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Invalid limit"}, {Status: 500, Code: "internal", Description: "Persistence failure"}}},
	{ID: "journal_retry_classifications", Path: journalconnect.JournalServiceProcessClassificationRetriesProcedure, Method: "POST", Summary: "Replay queued classification failures", Category: "journal", Request: &module.Schema{Type: "ProcessClassificationRetriesRequest"}, Response: &module.Schema{Type: "ProcessClassificationRetriesResponse"}, Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Invalid limit"}, {Status: 500, Code: "internal", Description: "Retry persistence failure"}}},
	{ID: "journal_retry_embeddings", Path: journalconnect.JournalServiceProcessEmbeddingRetriesProcedure, Method: "POST", Summary: "Replay queued embedding failures", Category: "journal", Request: &module.Schema{Type: "ProcessEmbeddingRetriesRequest"}, Response: &module.Schema{Type: "ProcessEmbeddingRetriesResponse"}, Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Invalid limit"}, {Status: 500, Code: "internal", Description: "Retry persistence failure"}}},
}
