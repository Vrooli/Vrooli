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
  current_event_id TEXT NOT NULL DEFAULT '',
  -- The answer form of the current revision: [AskQuestion] JSON, empty for none.
  questions_json TEXT NOT NULL DEFAULT '',
  accepts_context_requests INTEGER NOT NULL DEFAULT 0,
  -- The attachments of the current revision (see ask_attachments): how many,
  -- and a digest of what the producer attached, for the subject-key comparison.
  attachment_count INTEGER NOT NULL DEFAULT 0,
  attachments_digest TEXT NOT NULL DEFAULT '',
  -- The verified identity that sent the ask: run:<run id> or owner:<subject>.
  -- Empty for a caller with no identity and for every event-sourced ask.
  -- created_conversation is the sender run's verified conversation (shared by its
  -- recovered and child runs), and created_subject its verified owner account.
  -- Together they decide who reads the ask back: the run, its conversation, a run
  -- of the same owner that names the ask's subject_key, and the owner.
  created_by TEXT NOT NULL DEFAULT '',
  created_conversation TEXT NOT NULL DEFAULT '',
  created_subject TEXT NOT NULL DEFAULT '',
  -- Who the operator reads as asking: the sender the requester declared, else
  -- the verified service's name, else the kind of caller (ask_sender.go). It is
  -- display and filter text; it never decides who may read the ask.
  sender TEXT NOT NULL DEFAULT '',
  -- Absolute path of the decision's ready-to-run packet (the brief the agent
  -- that receives the answer starts a worker from); '' for none.
  packet TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL DEFAULT 'decision',
  template_name TEXT NOT NULL DEFAULT '',
  template_data_json TEXT NOT NULL DEFAULT '',
  decision_family TEXT NOT NULL DEFAULT '',
  decision_cause TEXT NOT NULL DEFAULT '',
  decision_kind TEXT NOT NULL DEFAULT ''
);
-- A recipient's explicit approval for one verified service to continue one
-- existing direct ask. This is an audit fact, never inferred from its key.
CREATE TABLE IF NOT EXISTS ask_source_attestations (
  ask_id TEXT NOT NULL REFERENCES asks(id) ON DELETE CASCADE,
  source TEXT NOT NULL,
  recipient TEXT NOT NULL,
  attested_at TEXT NOT NULL,
  PRIMARY KEY (ask_id, source, recipient)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_asks_notification ON asks(notification_id);
CREATE INDEX IF NOT EXISTS idx_asks_resolution_outbox ON asks(resolution_published_at, resolved_at, id);
-- The open asks, in attention order: the inbox page, the awaiting count and the
-- deadline sweep read only these rows, never the history behind them. The WHERE
-- must stay textually the one those queries use for SQLite to choose the index.
CREATE INDEX IF NOT EXISTS idx_asks_open_attention ON asks(deadline, created_at, id) WHERE state IN ('pending', 'escalated');
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
  late INTEGER NOT NULL DEFAULT 0,
  -- The operator's answers to the ask's questions: [AskQuestionResponse] JSON.
  responses_json TEXT NOT NULL DEFAULT '',
  -- Notes the operator pinned to the ask's attachments: [AskAttachmentNote] JSON.
  attachment_notes_json TEXT NOT NULL DEFAULT '',
  changes_json TEXT NOT NULL DEFAULT '',
  images_json TEXT NOT NULL DEFAULT '',
  attachments_json TEXT NOT NULL DEFAULT ''
);
-- Answer attribution is sourced only from the request authenticator. Keeping
-- it beside the durable Hub answer lets downstream decision verifiers fail
-- closed for legacy and recipient-fallback answers without a second authority.
CREATE TABLE IF NOT EXISTS answer_identity (
  ask_id TEXT PRIMARY KEY REFERENCES answers(ask_id) ON DELETE CASCADE,
  identity_kind TEXT NOT NULL DEFAULT 'unknown' CHECK (identity_kind IN ('verified_human', 'unknown')),
  identity_source TEXT NOT NULL DEFAULT '' CHECK (identity_source IN ('', 'cloudflare_access', 'personal_local'))
);
-- Additive evidence for password-authenticated root answers; existing answer
-- identity tables retain their original CHECK constraint and data.
CREATE TABLE IF NOT EXISTS root_password_answer_proofs (
  ask_id TEXT PRIMARY KEY REFERENCES answers(ask_id) ON DELETE CASCADE
);
-- A human approval authorizes at most one root rebaseline execution.
CREATE TABLE IF NOT EXISTS root_rebaseline_consumptions (
  ask_id TEXT PRIMARY KEY REFERENCES asks(id) ON DELETE CASCADE,
  consumed_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS answer_image_refs (
  ask_id TEXT NOT NULL REFERENCES answers(ask_id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  position INTEGER NOT NULL,
  sha256 TEXT NOT NULL REFERENCES attachment_blobs(sha256),
  PRIMARY KEY (ask_id, name)
);
CREATE INDEX IF NOT EXISTS idx_answer_image_refs_sha ON answer_image_refs(sha256);
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
-- "I need more context": the recipient's questions about an open ask. A
-- revision of the ask marks the open ones addressed and keeps the requester's
-- written answer (response) with them. published_at is the outbox
-- marker for notification_hub.ask.context_requested.v1.
CREATE TABLE IF NOT EXISTS ask_context_requests (
  id TEXT PRIMARY KEY,
  ask_id TEXT NOT NULL REFERENCES asks(id) ON DELETE CASCADE,
  question TEXT NOT NULL,
  requested_by TEXT NOT NULL,
  requested_at TEXT NOT NULL,
  addressed_at TEXT NOT NULL DEFAULT '',
  addressed_revision INTEGER NOT NULL DEFAULT 0,
  response TEXT NOT NULL DEFAULT '',
  published_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_ask_context_requests_ask ON ask_context_requests(ask_id, requested_at);
CREATE INDEX IF NOT EXISTS idx_ask_context_requests_outbox ON ask_context_requests(requested_at) WHERE published_at = '';
-- Decision-packet attachments. attachment_blobs is the inventory of the Hub's
-- own copies (the bytes live in the attachment blob store, keyed by sha256);
-- an ask's attachments reference blobs by hash. A blob nothing references is
-- swept 24 hours after last_touched_at (its last upload or reference).
CREATE TABLE IF NOT EXISTS attachment_blobs (
  sha256 TEXT PRIMARY KEY,
  size_bytes INTEGER NOT NULL,
  media_type TEXT NOT NULL DEFAULT '',
  stored_at TEXT NOT NULL,
  last_touched_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_attachment_blobs_touched ON attachment_blobs(last_touched_at);
-- One row per attachment of an ask's current revision; a revision that changes
-- attachments replaces them. file attachments carry sha256 and size_bytes;
-- folders carry their files in ask_attachment_entries.
CREATE TABLE IF NOT EXISTS ask_attachments (
  ask_id TEXT NOT NULL REFERENCES asks(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  position INTEGER NOT NULL,
  label TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL CHECK (kind IN ('file', 'folder')),
  role TEXT NOT NULL DEFAULT 'evidence' CHECK (role IN ('evidence', 'explainer')),
  source_path TEXT NOT NULL DEFAULT '',
  source_host TEXT NOT NULL DEFAULT '',
  sha256 TEXT NOT NULL DEFAULT '',
  size_bytes INTEGER NOT NULL DEFAULT 0,
  media_type TEXT NOT NULL DEFAULT '',
  preview_kind TEXT NOT NULL DEFAULT '',
  truncated INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (ask_id, name)
);
CREATE INDEX IF NOT EXISTS idx_ask_attachments_sha ON ask_attachments(sha256) WHERE sha256 <> '';
CREATE TABLE IF NOT EXISTS ask_attachment_entries (
  ask_id TEXT NOT NULL,
  attachment_name TEXT NOT NULL,
  position INTEGER NOT NULL,
  path TEXT NOT NULL,
  sha256 TEXT NOT NULL,
  size_bytes INTEGER NOT NULL DEFAULT 0,
  media_type TEXT NOT NULL DEFAULT '',
  preview_kind TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (ask_id, attachment_name, path),
  FOREIGN KEY (ask_id, attachment_name) REFERENCES ask_attachments(ask_id, name) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ask_attachment_entries_sha ON ask_attachment_entries(sha256);

-- Narrow proof association for root decision answers; never grants Hub access.
CREATE TABLE IF NOT EXISTS root_enrollments (
  recipient TEXT NOT NULL, cloudflare_subject TEXT NOT NULL, authenticator_subject TEXT NOT NULL,
  PRIMARY KEY (recipient, cloudflare_subject, authenticator_subject)
);

CREATE TABLE IF NOT EXISTS root_enrollment_generations (
 recipient TEXT NOT NULL, cloudflare_subject TEXT NOT NULL, generation INTEGER NOT NULL DEFAULT 0,
 PRIMARY KEY (recipient, cloudflare_subject)
);
