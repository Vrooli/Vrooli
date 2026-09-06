-- Read-only Web Console conversation-integrity audit.
-- Run with:
-- sqlite3 -readonly /home/matthalloran8/.vrooli/data/vrooli/web-console/web-console.db \
--   < /home/matthalloran8/Vrooli/scenarios/web-console/docs/internal/plans/artifacts/web-console-unlosable-conversations/integrity-audit.sql

.headers on
.mode column

SELECT
  (SELECT COUNT(*) FROM sessions) AS session_rows,
  (SELECT COUNT(*) FROM sessions WHERE archived_at <> '') AS archived_rows,
  (SELECT COUNT(*) FROM conversation_sessions) AS conversation_rows,
  (SELECT COUNT(*) FROM workspace_panes) AS workspace_rows,
  (SELECT COUNT(*) FROM agent_transcript_checkpoints) AS checkpoint_rows;

SELECT COUNT(*) AS orphan_conversation_sessions
FROM conversation_sessions AS conversation
LEFT JOIN sessions AS session ON session.id = conversation.session_id
WHERE session.id IS NULL;

SELECT COUNT(*) AS orphan_workspace_panes
FROM workspace_panes AS pane
LEFT JOIN sessions AS session ON session.id = pane.session_id
WHERE session.id IS NULL;

SELECT COUNT(*) AS orphan_checkpoints
FROM agent_transcript_checkpoints AS checkpoint
LEFT JOIN sessions AS session
  ON session.id = checkpoint.web_console_session_id
WHERE session.id IS NULL;

SELECT
  conversation.session_id,
  COUNT(event.id) AS messages,
  MAX(event.created_at) AS last_message,
  EXISTS(
    SELECT 1 FROM workspace_panes AS pane
    WHERE pane.session_id = conversation.session_id
  ) AS has_workspace,
  EXISTS(
    SELECT 1 FROM agent_transcript_checkpoints AS checkpoint
    WHERE checkpoint.web_console_session_id = conversation.session_id
  ) AS has_checkpoint
FROM conversation_sessions AS conversation
LEFT JOIN sessions AS session ON session.id = conversation.session_id
LEFT JOIN conversation_events AS event
  ON event.session_id = conversation.session_id
WHERE session.id IS NULL
GROUP BY conversation.session_id
ORDER BY last_message DESC;

