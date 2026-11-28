-- Deterministic copy of the production continuity incident for routed BAS.
-- Keep the missing sessions row intentional: the catalog, pane, checkpoint,
-- and 88-event transcript are the recoverable drift shape under test.
BEGIN;

INSERT OR IGNORE INTO conversation_sessions
  (session_id, last_sequence, last_seen_sequence, last_listened_sequence, created_at, updated_at)
VALUES
  ('a7e71c3c-e422-4c89-916a-03f92906fb89', 88, 88, 88,
   '2026-09-03T23:17:07Z', '2026-09-03T23:17:07Z');

INSERT OR IGNORE INTO workspace_panes
  (session_id, name, header_color, theme_id, font_size, sort_order, is_active,
   supports_messages_view, manually_unread, created_at, updated_at)
VALUES
  ('a7e71c3c-e422-4c89-916a-03f92906fb89', 'Codex', 'transparent', 'default', 14, 0, 0, 1, 0,
   '2026-09-03T23:17:07Z', '2026-09-03T23:17:07Z');

INSERT OR IGNORE INTO agent_transcript_checkpoints
  (source, source_key, web_console_session_id, cursor, updated_at)
VALUES
  ('codex', '01a06a6b-88da-7422-b391-bb0e1b686232',
   'a7e71c3c-e422-4c89-916a-03f92906fb89', '88', '2026-09-03T23:17:07Z');

INSERT OR IGNORE INTO conversation_catalog
  (session_id, lifecycle_state, lifecycle_version, backend, agent_type, agent_session_id,
   original_title, current_title, topic_summary, cwd, created_at, last_activity_at,
   archived_at, source_fingerprint)
VALUES
  ('a7e71c3c-e422-4c89-916a-03f92906fb89', 'recoverable', 1, 'codex', 'codex',
   '01a06a6b-88da-7422-b391-bb0e1b686232', 'Web Console continuity incident',
   'Web Console continuity incident', 'validation-intent receipt contract', '/tmp/bas-continuity',
   '2026-09-03T23:17:07Z', '2026-09-03T23:17:07Z', '2026-09-04T00:00:00Z',
   'sha256:bas-continuity-incident');

INSERT OR IGNORE INTO conversation_aliases
  (session_id, alias_kind, alias_value, observed_at)
VALUES
  ('a7e71c3c-e422-4c89-916a-03f92906fb89', 'agent_session',
   '01a06a6b-88da-7422-b391-bb0e1b686232', '2026-09-03T23:17:07Z'),
  ('a7e71c3c-e422-4c89-916a-03f92906fb89', 'pane',
   'a7e71c3c-e422-4c89-916a-03f92906fb89', '2026-09-03T23:17:07Z');

WITH RECURSIVE sequence(number) AS (
  SELECT 1
  UNION ALL
  SELECT number + 1 FROM sequence WHERE number < 88
)
INSERT OR IGNORE INTO conversation_events
  (id, session_id, source, role, text, speech_paragraphs, original_speech_paragraphs,
   summarized, created_at, sequence, delivery_state, tts_state, consumption_state)
SELECT
  printf('bas-continuity-event-%03d', number),
  'a7e71c3c-e422-4c89-916a-03f92906fb89',
  'codex_rollout',
  CASE WHEN number % 2 = 0 THEN 'assistant' ELSE 'user' END,
  printf('validation-intent receipt contract incident event %d', number),
  '[]', NULL, 0, '2026-09-03T23:17:07Z', number, 'received', 'idle', 'seen'
FROM sequence;

-- The migration is applied before the scenario restarts, so the API's normal
-- FTS startup hook may not have installed triggers or rebuilt an already
-- completed index yet. Make the fixture searchable in either a fresh or an
-- existing routed database.
CREATE VIRTUAL TABLE IF NOT EXISTS conversation_events_fts
  USING fts5(text, content='conversation_events', content_rowid='rowid', tokenize='unicode61');
INSERT INTO conversation_events_fts(conversation_events_fts) VALUES ('rebuild');

COMMIT;
