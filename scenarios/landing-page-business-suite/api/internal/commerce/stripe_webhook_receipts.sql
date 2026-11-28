-- Stripe event inbox: retain only identifiers and normalized status, never
-- arbitrary provider payloads or customer PII. A matching event ID is idempotent;
-- reuse with a changed payload digest is rejected by the application.
CREATE TABLE IF NOT EXISTS stripe_webhook_events (
  stripe_event_id TEXT PRIMARY KEY,
  livemode BOOLEAN NOT NULL,
  event_type TEXT NOT NULL,
  stripe_object_id TEXT NOT NULL DEFAULT '',
  payload_sha256 CHAR(64) NOT NULL,
  py1_test_run_id TEXT,
  py1_test_outcome TEXT,
  status TEXT NOT NULL DEFAULT 'received' CHECK (status IN ('received','processing','processed','retryable_failed')),
  claim_token TEXT NOT NULL DEFAULT '',
  attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  first_seen_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_seen_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  processed_at TIMESTAMP,
  failure_code TEXT
);
CREATE INDEX IF NOT EXISTS idx_stripe_webhook_events_outcome
  ON stripe_webhook_events(py1_test_run_id, py1_test_outcome, status);
CREATE INDEX IF NOT EXISTS idx_stripe_webhook_events_object
  ON stripe_webhook_events(stripe_object_id, livemode, event_type);
