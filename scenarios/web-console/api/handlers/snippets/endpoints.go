package snippets

import (
	snippetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/snippets/snippets_v1connect"

	"web-console/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "snippets_list", Path: snippetsconnect.SnippetsServiceListSnippetsProcedure, Method: "POST", Summary: "List message snippets", Description: "Returns snippets in pinned and recent-use order.", Category: "snippets", Response: &module.Schema{Type: "object", Properties: map[string]string{"snippets": "[]Snippet"}}},
	{ID: "snippets_upsert", Path: snippetsconnect.SnippetsServiceUpsertSnippetProcedure, Method: "POST", Summary: "Create or update a message snippet", Description: "Upserts sender-owned reusable text by id and rejects a blank name.", Category: "snippets", Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Blank snippet name"}}},
	{ID: "snippets_delete", Path: snippetsconnect.SnippetsServiceDeleteSnippetProcedure, Method: "POST", Summary: "Delete a message snippet", Description: "Idempotently removes an ordinary snippet row.", Category: "snippets"},
	{ID: "snippets_touch", Path: snippetsconnect.SnippetsServiceTouchSnippetProcedure, Method: "POST", Summary: "Record a snippet use", Description: "Atomically increments use count and records the latest use time without rewriting content.", Category: "snippets", Errors: []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "Snippet id does not exist"}}},
	{ID: "snippets_promote", Path: snippetsconnect.SnippetsServicePromoteSnippetProcedure, Method: "POST", Summary: "Promote a snippet to a governed skill", Description: "Copies the snippet name and body one way into prompt-manager without linking the records.", Category: "snippets", Errors: []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "Snippet id does not exist"}, {Status: 503, Code: "unavailable", Description: "prompt-manager binary is not available"}}},
}
