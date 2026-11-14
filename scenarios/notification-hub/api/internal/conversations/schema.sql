CREATE TABLE IF NOT EXISTS asks (
  id TEXT PRIMARY KEY,
  notification_id TEXT NOT NULL,
  question TEXT NOT NULL,
  allowed_answers TEXT NOT NULL,
  deadline TEXT NOT NULL,
  state TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  -- options_json carries [{key,label}]; allowed_answers stays the key list
  -- so asks written before labels existed keep validating answers.
  options_json TEXT NOT NULL DEFAULT '',
  recommended TEXT NOT NULL DEFAULT '',
  recommendation_reason TEXT NOT NULL DEFAULT '',
  default_answer TEXT NOT NULL DEFAULT '',
  reversible INTEGER NOT NULL DEFAULT 0,
  urgency TEXT NOT NULL DEFAULT '',
  context_url TEXT NOT NULL DEFAULT '',
  big_picture TEXT NOT NULL DEFAULT '',
  technical_details TEXT NOT NULL DEFAULT '',
  -- source + requester_ref identify the requester (event source scenario and
  -- event id, or the owner subject and idempotency key); one ask per pair.
  source TEXT NOT NULL DEFAULT '',
  source_event_type TEXT NOT NULL DEFAULT '',
  requester_ref TEXT NOT NULL DEFAULT '',
  correlation_json TEXT NOT NULL DEFAULT '',
  default_applied_at TEXT NOT NULL DEFAULT '',
  -- resolved_at is set on every terminal transition; resolution_published_at
  -- is the outbox marker for the ask-resolved event.
  resolved_at TEXT NOT NULL DEFAULT '',
  resolution_published_at TEXT NOT NULL DEFAULT '',
  -- Decision subjects: at most one open ask per (source, source_event_type,
  -- subject_key). The unique index that enforces it is created by
  -- Service.ensureAdditiveColumns AFTER the ALTERs, because on an existing
  -- database these columns do not exist when this file runs.
  subject_key TEXT NOT NULL DEFAULT '',
  last_asserted_at TEXT NOT NULL DEFAULT '',
  reassert_within_seconds INTEGER NOT NULL DEFAULT 0,
  revision_count INTEGER NOT NULL DEFAULT 0,
  superseded_by_ask_id TEXT NOT NULL DEFAULT '',
  -- Event id of the revision the ask currently shows (requester_ref is the
  -- first revision's id and never changes).
  current_event_id TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_asks_notification ON asks(notification_id);
CREATE INDEX IF NOT EXISTS idx_asks_resolution_outbox ON asks(resolution_published_at, resolved_at, id);
-- Continue a bounded outbox sweep even when older resolutions keep failing.
CREATE TABLE IF NOT EXISTS resolution_outbox_cursor (
  singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
  resolved_at TEXT NOT NULL,
  ask_id TEXT NOT NULL
);
-- Empty requester_ref values predate event identity and remain outside this
-- constraint. Nonempty identities must be owned by exactly one Ask, even when
-- concurrent producers choose different notification recipients/keys.
CREATE UNIQUE INDEX IF NOT EXISTS idx_asks_requester_identity
  ON asks(source, requester_ref) WHERE requester_ref <> '';
-- Request and withdrawal handlers serialize on this exact key. A withdrawal
-- received first leaves a durable state that blocks delayed request delivery.
-- These are producer lifecycle facts, not operator answers.
CREATE TABLE IF NOT EXISTS decision_request_lifecycle (
  source TEXT NOT NULL,
  requester_ref TEXT NOT NULL,
  source_event_type TEXT NOT NULL,
  state TEXT NOT NULL,
  reason TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (source, requester_ref)
);
CREATE TABLE IF NOT EXISTS answers (
  id TEXT PRIMARY KEY,
  ask_id TEXT NOT NULL UNIQUE REFERENCES asks(id) ON DELETE CASCADE,
  answer TEXT NOT NULL,
  answered_by TEXT NOT NULL,
  answered_at TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  late INTEGER NOT NULL DEFAULT 0
);
-- Answer attribution is sourced only from the request authenticator. Keeping
-- it beside the durable Hub answer lets downstream decision verifiers fail
-- closed for legacy and recipient-fallback answers without a second authority.
CREATE TABLE IF NOT EXISTS answer_identity (
  ask_id TEXT PRIMARY KEY REFERENCES answers(ask_id) ON DELETE CASCADE,
  identity_kind TEXT NOT NULL DEFAULT 'unknown' CHECK (identity_kind IN ('verified_human', 'unknown')),
  identity_source TEXT NOT NULL DEFAULT '' CHECK (identity_source IN ('', 'cloudflare_access', 'personal_local'))
);
CREATE TABLE IF NOT EXISTS escalation_steps (
  id TEXT PRIMARY KEY,
  ask_id TEXT NOT NULL REFERENCES asks(id) ON DELETE CASCADE,
  ordinal INTEGER NOT NULL,
  channel TEXT NOT NULL,
  outcome TEXT NOT NULL,
  reason TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(ask_id, ordinal)
);
-- Each in-place update keeps the revision it replaced, so the operator can see
-- what the decision used to say and producers can close it by any event id.
CREATE TABLE IF NOT EXISTS ask_revisions (
  ask_id TEXT NOT NULL REFERENCES asks(id) ON DELETE CASCADE,
  revision INTEGER NOT NULL,
  event_id TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  question TEXT NOT NULL,
  recommendation_reason TEXT NOT NULL DEFAULT '',
  context_url TEXT NOT NULL DEFAULT '',
  big_picture TEXT NOT NULL DEFAULT '',
  technical_details TEXT NOT NULL DEFAULT '',
  options_json TEXT NOT NULL DEFAULT '',
  correlation_json TEXT NOT NULL DEFAULT '',
  deadline TEXT NOT NULL DEFAULT '',
  revised_at TEXT NOT NULL,
  PRIMARY KEY (ask_id, revision)
);
CREATE INDEX IF NOT EXISTS idx_ask_revisions_event ON ask_revisions(event_id) WHERE event_id <> '';
-- Event ids an ask used to carry as requester_ref. requester_ref always names
-- the latest revision's event (producers bind answers to their current event
-- id); an earlier id still finds the ask for redeliveries and withdrawals.
CREATE TABLE IF NOT EXISTS ask_event_ids (
  ask_id TEXT NOT NULL REFERENCES asks(id) ON DELETE CASCADE,
  event_id TEXT NOT NULL,
  retired_at TEXT NOT NULL,
  PRIMARY KEY (ask_id, event_id)
);
CREATE INDEX IF NOT EXISTS idx_ask_event_ids_event ON ask_event_ids(event_id);
-- The last withdrawal a producer asserted for a subject. A request asserted
-- before it is stale: it describes a situation the producer already closed.
CREATE TABLE IF NOT EXISTS decision_subject_withdrawals (
  source TEXT NOT NULL,
  source_event_type TEXT NOT NULL,
  subject_key TEXT NOT NULL,
  asserted_at TEXT NOT NULL,
  reason TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (source, source_event_type, subject_key)
);
