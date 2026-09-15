package sessions

import (
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions/sessions_v1connect"

	"web-console/internal/module"
)

// Endpoints describes the sessions module's public surface. Connect-RPC
// method paths reference generated *Procedure constants so adding or
// renaming an RPC in sessions.proto breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "sessions_create",
		Path:        sessionsconnect.SessionsServiceCreateProcedure,
		Method:      "POST",
		Summary:     "Create a terminal session",
		Description: "Spawns a local PTY/tmux pane or routes creation to a dispatchable target catalog node. X-Idempotency-Key replays the cached response for 5 minutes.",
		Category:    "sessions",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"target_id": "string (optional target catalog ID)", "working_dir": "string (optional)",
		}},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"session": "Session"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Malformed body or invalid policy"},
			{Status: 404, Code: "not_found", Description: "Target catalog ID not found"},
			{Status: 412, Code: "failed_precondition", Description: "Target is not dispatchable"},
			{Status: 429, Code: "resource_exhausted", Description: "Configured session limit reached"},
			{Status: 503, Code: "unavailable", Description: "Backend unavailable (e.g. tmux missing)"},
		},
	},
	{
		ID:          "sessions_list",
		Path:        sessionsconnect.SessionsServiceListProcedure,
		Method:      "POST",
		Summary:     "List all active sessions",
		Description: "Returns every live session known to the server, including persistent panes adopted from a prior restart.",
		Category:    "sessions",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"sessions": "[]Session"},
		},
	},
	{
		ID:          "sessions_get",
		Path:        sessionsconnect.SessionsServiceGetProcedure,
		Method:      "POST",
		Summary:     "Get a session",
		Description: "Returns one session by id.",
		Category:    "sessions",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"session": "Session"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id"},
			{Status: 404, Code: "not_found", Description: "Session not found"},
		},
	},
	{
		ID:          "sessions_archived_list",
		Path:        sessionsconnect.SessionsServiceListArchivedProcedure,
		Method:      "POST",
		Summary:     "List archived sessions",
		Description: "Returns one newest entry per reopen lineage with pane identity, message count, and restore state.",
		Category:    "sessions",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"sessions": "[]ArchivedSession", "total": "integer"},
		},
	},
	{
		ID:          "sessions_delete",
		Path:        sessionsconnect.SessionsServiceDeleteProcedure,
		Method:      "POST",
		Summary:     "Terminate a session",
		Description: "Idempotent: succeeds whether or not the session existed.",
		Category:    "sessions",
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id"},
		},
	},
	{
		ID:          "sessions_archive",
		Path:        sessionsconnect.SessionsServiceArchiveProcedure,
		Method:      "POST",
		Summary:     "Archive a terminal session",
		Description: "Stops the live PTY while preserving the session row, workspace identity, transcript, and agent checkpoints.",
		Category:    "sessions",
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id"},
			{Status: 404, Code: "not_found", Description: "Session not found"},
		},
	},
	{
		ID:          "sessions_unarchive",
		Path:        sessionsconnect.SessionsServiceUnarchiveProcedure,
		Method:      "POST",
		Summary:     "Undo a session archive marker",
		Description: "Clears archived_at without starting a replacement process or resuming an agent.",
		Category:    "sessions",
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id"},
			{Status: 404, Code: "not_found", Description: "Session not found"},
		},
	},
	{
		ID:          "sessions_recoverable_list",
		Path:        sessionsconnect.SessionsServiceListRecoverableProcedure,
		Method:      "POST",
		Summary:     "List recoverable (orphaned) sessions",
		Description: "Returns awaiting_recovery rows from the session store, ordered by recency.",
		Category:    "sessions",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"sessions": "[]RecoverableSession"},
		},
	},
	{
		ID:          "sessions_recoverable_dismiss",
		Path:        sessionsconnect.SessionsServiceDismissRecoverableProcedure,
		Method:      "POST",
		Summary:     "Dismiss a recoverable session",
		Description: "Marks a row dismissed without recovery. On-disk state (CODEX_HOME) is preserved.",
		Category:    "sessions",
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id"},
			{Status: 404, Code: "not_found", Description: "No such session row"},
			{Status: 412, Code: "failed_precondition", Description: "Session is not in awaiting_recovery status"},
		},
	},
	{
		ID:          "sessions_recover",
		Path:        sessionsconnect.SessionsServiceRecoverProcedure,
		Method:      "POST",
		Summary:     "Recover an orphaned session",
		Description: "Spawns a fresh persistent pane, copies CODEX_HOME for codex agents, and pastes the resume command. Idempotent via X-Idempotency-Key.",
		Category:    "sessions",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"old_session_id":    "string",
				"new_session_id":    "string",
				"agent_type":        "string",
				"command_sent":      "string",
				"codex_home_copied": "bool",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id"},
			{Status: 404, Code: "not_found", Description: "No such session row"},
			{Status: 412, Code: "failed_precondition", Description: "Session is not awaiting_recovery, or no agent identity recorded"},
			{Status: 429, Code: "resource_exhausted", Description: "Session limit reached while spawning replacement"},
		},
	},
	{
		ID:          "sessions_archive_retention_get",
		Path:        sessionsconnect.SessionsServiceGetArchiveRetentionProcedure,
		Method:      "POST",
		Summary:     "Inspect archive retention",
		Description: "Returns the configured bounds and measured transcript/agent-history storage for explicitly archived sessions.",
		Category:    "sessions",
	},
	{
		ID:          "sessions_archive_prune",
		Path:        sessionsconnect.SessionsServicePruneArchiveProcedure,
		Method:      "POST",
		Summary:     "Preview or apply archive retention",
		Description: "Dry-run by default. Apply mode prunes agent history first and deletes only old message-less archived transcripts.",
		Category:    "sessions",
	},
	{
		ID:          "sessions_policy_get",
		Path:        sessionsconnect.SessionsServiceGetPolicyProcedure,
		Method:      "POST",
		Summary:     "Get a session's expiration policy",
		Description: "Returns the policy plus derived expires_at/ttl_seconds (present only when has_expiry = true).",
		Category:    "sessions",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"policy": "PolicyView"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id"},
			{Status: 404, Code: "not_found", Description: "Session not found"},
		},
	},
	{
		ID:          "sessions_policy_update",
		Path:        sessionsconnect.SessionsServiceUpdatePolicyProcedure,
		Method:      "POST",
		Summary:     "Update a session's expiration policy",
		Description: "Sets a new policy. Replaying the same mode+duration is a no-op for the audit event log.",
		Category:    "sessions",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"policy": "PolicyView"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id, missing policy, or invalid mode/duration"},
			{Status: 404, Code: "not_found", Description: "Session not found"},
		},
	},
}
