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
		Description: "Spawns a PTY (ephemeral) or tmux pane (persistent). X-Idempotency-Key replays the cached response for 5 minutes.",
		Category:    "sessions",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"session": "Session"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Malformed body or invalid policy"},
			{Status: 429, Code: "resource_exhausted", Description: "Configured session limit reached"},
			{Status: 503, Code: "unavailable", Description: "Backend unavailable (e.g. tmux missing)"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console session create", Args: []string{"--body-file"}},
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
		CLIMapping: &module.CLIMapping{Command: "web-console session list"},
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
		CLIMapping: &module.CLIMapping{Command: "web-console session get", Args: []string{"--id"}},
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
		CLIMapping: &module.CLIMapping{Command: "web-console session delete", Args: []string{"--id"}},
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
		CLIMapping: &module.CLIMapping{Command: "web-console session list-recoverable"},
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
		CLIMapping: &module.CLIMapping{Command: "web-console session dismiss", Args: []string{"--id"}},
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
		CLIMapping: &module.CLIMapping{Command: "web-console session recover", Args: []string{"--id"}},
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
		CLIMapping: &module.CLIMapping{Command: "web-console session policy-get", Args: []string{"--id"}},
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
		CLIMapping: &module.CLIMapping{Command: "web-console session policy-set", Args: []string{"--id", "--body-file"}},
	},
}
