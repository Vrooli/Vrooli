package conversation

import (
	"web-console/internal/module"

	conversationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation/conversation_v1connect"
)

// Endpoints describes the conversation module's public surface. Connect-RPC
// method paths reference generated *Procedure constants so adding or renaming
// an RPC in conversation.proto breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "conversation_fleet_archive_search",
		Path:        conversationconnect.ConversationServiceSearchNodeArchiveProcedure,
		Method:      "POST",
		Summary:     "Search one selected fleet node's Web Console archive",
		Description: "Sends the query over Bridge's authenticated target-aware Connect route; unavailable or unauthorized nodes degrade without changing local archive data.",
		Category:    "conversation",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"node_id": "string", "query": "string", "harness": "string", "role": "string", "limit": "int32",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"node_id": "string", "available": "bool", "search": "SearchArchivedResponse", "degradation": "string",
		}},
	},
	{
		ID:          "conversation_optional_semantic_archive_search",
		Path:        conversationconnect.ConversationServiceSearchSemanticProcedure,
		Method:      "POST",
		Summary:     "Search own archives with optional Agent Manager semantic results",
		Description: "Returns Web Console own-store lexical results even when Agent Manager is stopped; when available, adds semantic hits with coverage and degradation evidence.",
		Category:    "conversation",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"query": "string", "harnesses": "[]string", "limit": "int32",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"local": "SearchArchivedResponse", "semantic_hits": "[]SemanticSearchHit", "coverage": "SemanticSearchCoverage",
			"degradations": "[]SemanticSearchDegradation", "semantic_took_ms": "uint64", "semantic_available": "bool", "semantic_error": "string",
		}},
	},
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
		ID:          "conversation_archived_search",
		Path:        conversationconnect.ConversationServiceSearchArchivedProcedure,
		Method:      "POST",
		Summary:     "Search archived conversation messages",
		Description: "Uses the FTS5 index across deliberate archives, dismissed recovery rows, and crash orphans; live sessions are excluded.",
		Category:    "conversation",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"matches": "[]ArchivedSearchMatch", "total_matches": "int64", "distinct_sessions": "int64"},
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
}
