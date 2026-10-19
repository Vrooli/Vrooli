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
  resolution_published_at TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_asks_notification ON asks(notification_id);
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
