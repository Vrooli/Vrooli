package chat

import (
	chatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/chat/chat_v1connect"

	"portal/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "chat_list", Path: chatconnect.ChatServiceListChatsProcedure, Method: "POST", Summary: "List chats", Description: "Lists conversations with optional group and text filters.", Category: "chat"},
	{ID: "chat_create", Path: chatconnect.ChatServiceCreateChatProcedure, Method: "POST", Summary: "Create chat", Description: "Creates a new LLM or agent conversation.", Category: "chat"},
	{ID: "chat_get", Path: chatconnect.ChatServiceGetChatProcedure, Method: "POST", Summary: "Get chat", Description: "Returns one conversation by id.", Category: "chat"},
	{ID: "chat_update", Path: chatconnect.ChatServiceUpdateChatProcedure, Method: "POST", Summary: "Update chat", Description: "Updates conversation metadata and active leaf selection.", Category: "chat"},
	{ID: "chat_delete", Path: chatconnect.ChatServiceDeleteChatProcedure, Method: "POST", Summary: "Delete chat", Description: "Deletes one conversation.", Category: "chat"},
	{ID: "chat_groups_list", Path: chatconnect.ChatServiceListGroupsProcedure, Method: "POST", Summary: "List chat groups", Description: "Lists user-defined sidebar groups.", Category: "chat"},
	{ID: "chat_group_create", Path: chatconnect.ChatServiceCreateGroupProcedure, Method: "POST", Summary: "Create chat group", Description: "Creates a colored sidebar group.", Category: "chat"},
	{ID: "chat_group_update", Path: chatconnect.ChatServiceUpdateGroupProcedure, Method: "POST", Summary: "Update chat group", Description: "Updates group name, color, order, or collapsed state.", Category: "chat"},
	{ID: "chat_group_delete", Path: chatconnect.ChatServiceDeleteGroupProcedure, Method: "POST", Summary: "Delete chat group", Description: "Deletes a sidebar group without deleting its chats.", Category: "chat"},
}
