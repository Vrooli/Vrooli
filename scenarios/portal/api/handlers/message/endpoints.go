package message

import (
	messageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/message/message_v1connect"

	"portal/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "message_tree_get", Path: messageconnect.MessageServiceGetTreeProcedure, Method: "POST", Summary: "Get message tree", Description: "Returns the persisted message tree and active leaf for a chat.", Category: "message"},
	{ID: "message_send", Path: messageconnect.MessageServiceSendMessageProcedure, Method: "POST", Summary: "Send message", Description: "Persists a user message under the selected parent.", Category: "message"},
	{ID: "message_edit", Path: messageconnect.MessageServiceEditMessageProcedure, Method: "POST", Summary: "Edit message", Description: "Edits a message by creating the next branch version.", Category: "message"},
	{ID: "message_regenerate", Path: messageconnect.MessageServiceRegenerateProcedure, Method: "POST", Summary: "Regenerate completion", Description: "Creates a sibling assistant branch from an existing message.", Category: "message"},
	{ID: "message_completion_stream", Path: messageconnect.MessageServiceStreamCompletionProcedure, Method: "POST", Summary: "Stream completion", Description: "Streams completion status, tokens, search attachments, agent activity, usage, and errors.", Category: "message"},
}
