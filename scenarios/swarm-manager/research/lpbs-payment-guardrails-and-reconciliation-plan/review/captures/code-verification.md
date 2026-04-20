# Code Verification Evidence — LPBS Payment Guardrails Research

Cross-checked the 8 findings in `conclusion.md` against the LPBS API source tree
at `scenarios/landing-page-business-suite/api/`. Each claim below is verified
with a file:line reference captured on 2026-04-17.

## Finding 1 — Transactional primitives exist (SUPPORTED)

- `stripe_event_id` unique constraint on `credit_transactions`:
  `main.go:826` defines `stripe_event_id VARCHAR(255) UNIQUE`; index at `main.go:978`.
- Row-level locking on credit wallet consumption:
  `stripe_credit_service.go:66` uses `SELECT ... FROM credit_wallets WHERE customer_email = $1 FOR UPDATE` inside the `ConsumeCreditsIdempotent` transaction. The conclusion cites line 65; the actual line is 66 (trivial off-by-one).
- Atomic email migration:
  `stripe_webhook_service.go:635-663` (`handleCustomerUpdated`) wraps BEGIN/COMMIT across subscriptions, users, credit_wallets.

## Finding 2 — Structured logging exists (SUPPORTED)

- `logStructured()` at `main.go:1077`; `logStructuredError()` at `main.go:1086`.
- Sample events confirmed in-tree:
  - `credits_consumed` — `stripe_credit_service.go:106`
  - `credit_consumption_already_processed` — `stripe_credit_service.go:54`
  - `checkout.session.completed` handler path — `stripe_webhook_service.go:185`
  - `webhook_processing_failed` — `stripe_webhook_service.go:71`

## Finding 3 — Checkout→subscription flow (SUPPORTED with naming nuance)

- `handleCheckoutCompleted` (`stripe_webhook_service.go:154-228`) delegates to
  `handleSubscriptionCompletion` (line 230+), which performs
  `INSERT INTO subscriptions ... ON CONFLICT DO UPDATE` at `stripe_webhook_service.go:245-249`.
- `checkout_sessions.status VARCHAR(50) NOT NULL` at `main.go:603`; status set
  to `'complete'` at `stripe_webhook_service.go:191-195`.
- **Naming nuance**: conclusion references `insertOrUpdateSubscription()` by
  name, but the actual function is `handleSubscriptionCompletion`. Behaviour
  (insert-or-update) is accurate; only the symbol name differs. Execute-item
  plans should refer to the real function name.

## Finding 4 — Download fulfillment tracking (SUPPORTED with nuance)

- `DownloadAuthorizer.Authorize()` at `download_service.go:718-756` performs
  entitlement checks and returns a `*DownloadAsset`. No delivery-tracking call
  inside the method.
- No table in schema tracks download completion or bytes-served confirmation.
- **Nuance**: the conclusion describes `Authorize()` as "returning a presigned
  S3 URL". The method returns a `DownloadAsset`; presigned URL generation
  happens elsewhere in the download pipeline. Does not change the recommendation
  (no fulfillment tracking exists either way), but Action 4's plan should locate
  the actual URL-signing seam.

## Finding 5 — Admin payment operations missing (SUPPORTED)

- Only Stripe admin surface found is `/api/v1/admin/settings/stripe` (GET/PUT)
  for configuration. No admin endpoints for customer lookup, entitlement
  grant/revoke, credit adjustment, stuck-session viewer, or webhook retry.

## Finding 6 — Webhook atomicity gap (SUPPORTED — CRITICAL)

This is the highest-priority finding and is confirmed verbatim:

- `stripe_webhook_service.go:191-195` — direct `s.db.Exec()` to set
  `checkout_sessions.status = 'complete'` (no tx).
- `stripe_webhook_service.go:226` — subsequent call to
  `handleSubscriptionCompletion()` performs its INSERT/ON CONFLICT via a
  separate connection (`stripe_webhook_service.go:245`).
- The two writes are not wrapped in a shared transaction, so a failure between
  them leaves the session marked `complete` with no subscription row — exactly
  the silent partial-failure mode the research names.

## Finding 7 — intro_anomaly_log table (SUPPORTED)

- Defined at `main.go:1009-1020`. Columns: `id`, `email`, `customer_id`,
  `coupon_id`, `anomaly_type`, `details JSONB`, `created_at`. Three indexes on
  `email`, `anomaly_type`, `created_at`. Matches the conclusion's description;
  it correctly notes this is the right foundation to generalize.
- Minor: conclusion cites lines `1007-1018`, actual is `1009-1020`. Off by two.

## Finding 8 — Credit consumption vs reversal (SUPPORTED)

- `ConsumeCreditsIdempotent` (`stripe_credit_service.go:38-104`) uses
  BEGIN/COMMIT and `FOR UPDATE`.
- No generalized refund endpoint. Only `AdjustUsage()` at
  `usage_service.go:1203` provides the AI-gateway estimate-refund path.
  `stripe_credit_service.go` has no corresponding refund method. Matches the
  gap the research flags as a precondition for Action 5's credit-adjustment
  endpoint.

## Follow-up item materialization

All 6 actions listed in the conclusion have corresponding backlog items:

| Action | Backlog item |
|--------|--------------|
| 1 — Webhook atomicity fix | `fix/lpbs-checkout-webhook-atomicity` |
| 2 — Reconciliation ticker | `execute/lpbs-stripe-reconciliation-ticker` |
| 3 — Payment anomaly log + alerts | `execute/lpbs-payment-anomaly-log-and-alerts` |
| 4 — Download delivery confirmation | `execute/lpbs-download-delivery-confirmation` |
| 5 — Admin payment ops | `execute/lpbs-admin-payment-ops` |
| 6 — Monetization dashboard | `execute/lpbs-monetization-dashboard` |

## Workshop decision alignment

Round 001 decisions drove the research:
- `d1 = A` (full breadth across all 5 guardrail areas) — matches the six-action breadth.
- `d2 = A` (periodic reconciliation job vs. webhook dead-letter) — matches Action 2 as an in-process ticker.
- `d3 = A` (in-app alerting) — matches Action 3 webhook-dispatcher design.

## Summary

Research conclusions are faithful to the code. Minor inaccuracies are naming
(`insertOrUpdateSubscription` vs `handleSubscriptionCompletion`), line-number
drift (1-2 lines), and one simplification (Authorize return type). None of
these invalidate the recommendations. Critical Finding 6 is directly
reproducible at the cited location.
