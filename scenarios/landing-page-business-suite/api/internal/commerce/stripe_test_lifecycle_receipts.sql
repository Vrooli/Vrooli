-- Test-only lifecycle receipts minimize Stripe renewal, failed-renewal and
-- cancellation transitions. Paid invoices have an idempotent Money Ledger
-- outbox; non-payment lifecycle events are durably marked not_applicable.
CREATE TABLE IF NOT EXISTS py1_test_lifecycle_receipts (
  stripe_event_id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL,
  outcome TEXT NOT NULL,
  event_type TEXT NOT NULL,
  stripe_object_id TEXT NOT NULL,
  subscription_id TEXT NOT NULL,
  subscription_status TEXT NOT NULL,
  payment_status TEXT NOT NULL,
  billing_reason TEXT NOT NULL DEFAULT '',
  amount_minor BIGINT NOT NULL DEFAULT 0 CHECK (amount_minor >= 0),
  currency CHAR(3),
  money_ledger_external_id TEXT UNIQUE,
  money_ledger_status TEXT NOT NULL DEFAULT 'not_applicable' CHECK (money_ledger_status IN ('not_applicable','pending','posted','retryable_failed')),
  money_ledger_posting_id TEXT,
  money_ledger_receipt_id TEXT,
  money_ledger_attempts INTEGER NOT NULL DEFAULT 0 CHECK (money_ledger_attempts >= 0),
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  posted_at TIMESTAMP,
  CHECK ((money_ledger_status = 'not_applicable' AND money_ledger_external_id IS NULL) OR money_ledger_external_id IS NOT NULL)
);
CREATE INDEX IF NOT EXISTS idx_py1_test_lifecycle_outcome ON py1_test_lifecycle_receipts(run_id,outcome,created_at);
CREATE INDEX IF NOT EXISTS idx_py1_test_lifecycle_ledger ON py1_test_lifecycle_receipts(money_ledger_status,created_at);
