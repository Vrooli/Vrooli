package conversation

import (
	conversationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation/conversation_v1connect"

	"web-console/internal/module"
)

// Endpoints describes the conversation module's public surface. Connect-RPC
// method paths reference generated *Procedure constants so adding or renaming
// an RPC in conversation.proto breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "conversation_get",
		Path:        conversationconnect.ConversationServiceGetProcedure,
		Method:      "POST",
		Summary:     "Get a session's conversation history",
		Description: "Returns events and cursor for a session; since_sequence > 0 returns only events after that sequence.",
		Category:    "conversation",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"session_id":     "string",
				"since_sequence": "int64",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"session_id": "string",
				"events":     "[]ConversationEvent",
				"cursor":     "ConversationCursor",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing session_id"},
		},
	},
	{
		ID:          "conversation_cursor_update",
		Path:        conversationconnect.ConversationServiceUpdateCursorProcedure,
		Method:      "POST",
		Summary:     "Update a session's read/listened cursor",
		Description: "Partial update; only fields with has_* = true are applied.",
		Category:    "conversation",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"cursor": "ConversationCursor"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing session_id"},
		},
	},
	{
		ID:          "conversation_event_summarize",
		Path:        conversationconnect.ConversationServiceSummarizeEventProcedure,
		Method:      "POST",
		Summary:     "Summarize one assistant event for TTS",
		Description: "Re-runs the Ollama-backed summarizer on a single event and updates its speech paragraphs.",
		Category:    "conversation",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"summarized":        "bool",
				"speech_paragraphs": "[]string",
				"error":             "string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing ids or event is not assistant-authored"},
			{Status: 404, Code: "not_found", Description: "Session or event not found"},
		},
	},
	{
		ID:          "conversation_file_resolve",
		Path:        conversationconnect.ConversationServiceResolveFileReferenceProcedure,
		Method:      "POST",
		Summary:     "Resolve a file reference for a session",
		Description: "Translates a path string (with optional :line suffix or file:// scheme) into an absolute path that lives under an allowed root.",
		Category:    "conversation",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"input_path":       "string",
				"resolved_path":    "string",
				"line":             "int32",
				"has_line":         "bool",
				"exists":           "bool",
				"resolution_basis": "string",
				"category":         "string",
				"can_preview":      "bool",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing or malformed path"},
			{Status: 403, Code: "permission_denied", Description: "Path resolved outside allowed roots"},
			{Status: 404, Code: "not_found", Description: "Session or file not found"},
		},
	},
	{
		ID:          "conversation_file_content",
		Path:        conversationconnect.ConversationServiceGetFileReferenceContentProcedure,
		Method:      "POST",
		Summary:     "Read a previewable file referenced by a session",
		Description: "Returns up to 256 KiB of UTF-8 text content for a path that resolved under an allowed root.",
		Category:    "conversation",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"path":         "string",
				"line":         "int32",
				"has_line":     "bool",
				"category":     "string",
				"content_type": "string",
				"content":      "string",
				"truncated":    "bool",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing path"},
			{Status: 403, Code: "permission_denied", Description: "Path resolved outside allowed roots"},
			{Status: 404, Code: "not_found", Description: "Session or file not found"},
			{Status: 412, Code: "failed_precondition", Description: "File type not previewable or too large"},
		},
	},
}
