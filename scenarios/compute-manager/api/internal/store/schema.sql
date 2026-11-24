CREATE TABLE IF NOT EXISTS instance_intents (
  id TEXT PRIMARY KEY,
  idempotency_key TEXT NOT NULL UNIQUE,
  requested_by TEXT NOT NULL DEFAULT '',
  provider TEXT NOT NULL,
  spec_json TEXT NOT NULL DEFAULT '{}',
  reservation_id TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL DEFAULT 'open',
  instance_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  resolved_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_instance_intents_state_created ON instance_intents(state, created_at);

CREATE TABLE IF NOT EXISTS instances (
  id TEXT PRIMARY KEY,
  provider TEXT NOT NULL,
  provider_instance_id TEXT NOT NULL,
  state TEXT NOT NULL,
  region TEXT NOT NULL DEFAULT '',
  size TEXT NOT NULL DEFAULT '',
  image TEXT NOT NULL DEFAULT '',
  address TEXT NOT NULL DEFAULT '',
  ssh_user TEXT NOT NULL DEFAULT '',
  ssh_port INTEGER NOT NULL DEFAULT 0,
  bridge_machine_id TEXT NOT NULL DEFAULT '',
  tenant TEXT NOT NULL DEFAULT '',
  reservation_id TEXT NOT NULL DEFAULT '',
  spend_idempotency_key TEXT NOT NULL DEFAULT '',
  tags_json TEXT NOT NULL DEFAULT '{}',
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  running_at TEXT NOT NULL DEFAULT '',
  destroyed_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_instances_state_expires ON instances(state, expires_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_instances_provider_identity ON instances(provider, provider_instance_id);

-- Deployment Manager staged lifecycle. The acquire row binds the stable
-- request key to one immutable request digest and the exact CM instance.
CREATE TABLE IF NOT EXISTS managed_acquisitions (
  idempotency_key TEXT PRIMARY KEY,
  request_digest TEXT NOT NULL,
  instance_id TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS managed_enrollments (
  idempotency_key TEXT PRIMARY KEY,
  request_digest TEXT NOT NULL,
  acquisition_idempotency_key TEXT NOT NULL,
  instance_id TEXT NOT NULL,
  state TEXT NOT NULL,
  bridge_machine_id TEXT NOT NULL DEFAULT '',
  bridge_operation_id TEXT NOT NULL DEFAULT '',
  reason TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS adopted_vps_instances (
  instance_id TEXT PRIMARY KEY,
  display_name TEXT NOT NULL DEFAULT '',
  adoption_state TEXT NOT NULL,
  credential_step_ref TEXT NOT NULL DEFAULT '',
  evidence_verified BOOLEAN NOT NULL DEFAULT FALSE,
  spend_idempotency_key TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

-- Request/evidence identity only; the first-touch credential is never stored.
CREATE TABLE IF NOT EXISTS adopted_vps_first_touch (
  instance_id TEXT PRIMARY KEY,
  idempotency_key TEXT NOT NULL DEFAULT '',
  bridge_onboarding_op_id TEXT NOT NULL DEFAULT '',
  evidence_status TEXT NOT NULL DEFAULT '',
  evidence_observed_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS provider_receipts (
  id TEXT PRIMARY KEY,
  intent_id TEXT NOT NULL,
  provider TEXT NOT NULL,
  operation TEXT NOT NULL,
  provider_instance_id TEXT NOT NULL DEFAULT '',
  response_status TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS reservations (
  id TEXT PRIMARY KEY,
  intent_id TEXT NOT NULL,
  instance_id TEXT NOT NULL DEFAULT '',
  supersedes TEXT NOT NULL DEFAULT '',
  meter_key TEXT NOT NULL,
  state TEXT NOT NULL,
  held_at TEXT NOT NULL,
  settled_at TEXT NOT NULL DEFAULT '',
  quantity INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_reservations_instance_state ON reservations(instance_id, state);

CREATE TABLE IF NOT EXISTS spend_quotes (
  id TEXT PRIMARY KEY,
  idempotency_key TEXT NOT NULL UNIQUE,
  provider TEXT NOT NULL,
  offer_id TEXT NOT NULL DEFAULT '',
  region TEXT NOT NULL,
  size TEXT NOT NULL,
  ipv4 BOOLEAN NOT NULL,
  term TEXT NOT NULL,
  currency TEXT NOT NULL,
  server_minor BIGINT NOT NULL DEFAULT 0,
  ipv4_minor BIGINT NOT NULL DEFAULT 0,
  monthly_minor BIGINT NOT NULL,
  tax_basis TEXT NOT NULL,
  source_url TEXT NOT NULL DEFAULT '',
  source_version TEXT NOT NULL,
  observed_at TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  state TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  quote_json TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_spend_quotes_created ON spend_quotes(created_at);

CREATE TABLE IF NOT EXISTS spend_admissions (
  idempotency_key TEXT PRIMARY KEY,
  quote_id TEXT NOT NULL,
  cap_currency TEXT NOT NULL,
  cap_monthly_minor BIGINT NOT NULL,
  cap_tax_basis TEXT NOT NULL,
  decision TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tenant_ceiling_holds (
  id TEXT PRIMARY KEY,
  tenant TEXT NOT NULL,
  quantity INTEGER NOT NULL,
  expires_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tenant_ceiling_holds_active ON tenant_ceiling_holds(tenant, expires_at);

CREATE TABLE IF NOT EXISTS usage_records (
  id TEXT PRIMARY KEY,
  instance_id TEXT NOT NULL,
  tenant TEXT NOT NULL DEFAULT '',
  quantity INTEGER NOT NULL DEFAULT 0,
  started_at TEXT NOT NULL,
  ended_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_usage_records_instance_started ON usage_records(instance_id, started_at);

CREATE TABLE IF NOT EXISTS reconcile_findings (
  id TEXT PRIMARY KEY,
  observed_at TEXT NOT NULL,
  kind TEXT NOT NULL,
  provider TEXT NOT NULL,
  provider_instance_id TEXT NOT NULL DEFAULT '',
  instance_id TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open',
  detail_json TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_reconcile_findings_status_observed ON reconcile_findings(status, observed_at);

-- Provider invoice lines are immutable source facts. The stable external ID is
-- the provider's invoice + line identity; corrections append a separate receipt.
CREATE TABLE IF NOT EXISTS invoice_reconciliation_receipts (
  external_id TEXT PRIMARY KEY,
  provider TEXT NOT NULL,
  invoice_id TEXT NOT NULL,
  line_id TEXT NOT NULL,
  provider_instance_id TEXT NOT NULL,
  instance_id TEXT NOT NULL,
  reservation_id TEXT NOT NULL,
  quote_id TEXT NOT NULL,
  ledger_book_id TEXT NOT NULL,
  ledger_account_id TEXT NOT NULL,
  amount_minor BIGINT NOT NULL,
  currency TEXT NOT NULL,
  occurred_at TEXT NOT NULL,
  provenance TEXT NOT NULL,
  state TEXT NOT NULL,
  invoice_digest TEXT NOT NULL,
  confirmation_event_id TEXT NOT NULL DEFAULT '',
  confirmation_ask_id TEXT NOT NULL DEFAULT '',
  confirmation_actor TEXT NOT NULL DEFAULT '',
  confirmation_identity_source TEXT NOT NULL DEFAULT '',
  confirmation_digest TEXT NOT NULL DEFAULT '',
  confirmed_at TEXT NOT NULL DEFAULT '',
  ledger_posting_id TEXT NOT NULL DEFAULT '',
  reversal_posting_id TEXT NOT NULL DEFAULT '',
  reverses_external_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_invoice_receipts_instance ON invoice_reconciliation_receipts(instance_id, occurred_at);

-- Immutable evidence that Notification Hub read back one verified-human
-- answer for one invoice digest. The invoice row retains its current state;
-- this row is the durable authorization receipt used to audit/replay it.
CREATE TABLE IF NOT EXISTS invoice_confirmation_receipts (
  ask_id TEXT PRIMARY KEY,
  external_id TEXT NOT NULL UNIQUE,
  requester_ref TEXT NOT NULL,
  invoice_digest TEXT NOT NULL,
  answer TEXT NOT NULL CHECK (answer IN ('approve','decline')),
  answered_by TEXT NOT NULL,
  identity_source TEXT NOT NULL,
  verified_at TEXT NOT NULL,
  recorded_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS enrollment_queue (
  id TEXT PRIMARY KEY,
  instance_id TEXT NOT NULL,
  host TEXT NOT NULL,
  user_name TEXT NOT NULL DEFAULT 'root',
  machine_id TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL DEFAULT 'pending',
  last_error TEXT NOT NULL DEFAULT '',
  attempts INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_enrollment_queue_state_updated ON enrollment_queue(state, updated_at);

CREATE TABLE IF NOT EXISTS bridge_key_cache (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  public_key TEXT NOT NULL,
  fingerprint TEXT NOT NULL DEFAULT '',
  key_type TEXT NOT NULL DEFAULT '',
  refreshed_at TEXT NOT NULL
);
