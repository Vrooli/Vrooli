-- Minimized economic receipt/outbox for an explicitly correlated PY1 Stripe
-- test Checkout payment. It contains provider identities and amounts only.
CREATE TABLE IF NOT EXISTS py1_test_economic_receipts (
  stripe_event_id TEXT PRIMARY KEY,
  checkout_session_id TEXT NOT NULL UNIQUE,
  payment_intent_id TEXT,
  subscription_id TEXT,
  run_id TEXT NOT NULL,
  outcome TEXT NOT NULL,
  amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
  currency CHAR(3) NOT NULL,
  payment_status TEXT NOT NULL CHECK (payment_status = 'paid'),
  money_ledger_external_id TEXT NOT NULL UNIQUE,
  money_ledger_status TEXT NOT NULL DEFAULT 'pending' CHECK (money_ledger_status IN ('pending','posted','retryable_failed')),
  money_ledger_posting_id TEXT,
  money_ledger_receipt_id TEXT,
  money_ledger_attempts INTEGER NOT NULL DEFAULT 0 CHECK (money_ledger_attempts >= 0),
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  posted_at TIMESTAMP,
  CHECK (payment_intent_id IS NOT NULL OR subscription_id IS NOT NULL)
);
CREATE INDEX IF NOT EXISTS idx_py1_test_economic_receipts_outcome ON py1_test_economic_receipts(run_id,outcome);
CREATE INDEX IF NOT EXISTS idx_py1_test_economic_receipts_ledger ON py1_test_economic_receipts(money_ledger_status,created_at);
