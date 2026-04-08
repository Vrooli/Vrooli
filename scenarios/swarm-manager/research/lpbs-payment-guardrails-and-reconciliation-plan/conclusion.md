# Research Conclusion: LPBS Payment Guardrails and Reconciliation Plan

## Research Question
What operational guardrails — anomaly detection, reconciliation checks, alerting, manual recovery tools, and dashboards — does LPBS need before scaling real revenue through its Stripe-backed payment and download fulfillment pipeline?

## Summary
<!-- TBD — will be refined as investigation progresses -->

## Methodology
Investigation of the LPBS API codebase (`scenarios/landing-page-business-suite/api/`) focusing on:
- Stripe integration layer (webhook handling, checkout, subscriptions, credits)
- Entitlement and download authorization flow
- Database schema (checkout_sessions, subscriptions, credit_wallets, credit_transactions, download_assets)
- Existing structured logging and error handling patterns
- Idempotency and consistency mechanisms already in place

## Findings

### Finding 1: Solid transactional foundations already exist
The payment pipeline has strong consistency primitives:
- **Webhook idempotency**: `stripe_event_id` unique constraint on `credit_transactions` prevents double-processing.
- **Row-level locking**: `FOR UPDATE` on credit wallet during consumption prevents TOCTOU races (`stripe_credit_service.go:65`).
- **Atomic email migration**: `handleCustomerUpdated()` updates all tables in a single transaction.
- **Email normalization**: All lookups use lowercase/trimmed emails.

These mean the core payment→entitlement pipeline is unlikely to silently lose data. Guardrails should focus on **detecting** problems (not preventing already-handled ones) and **recovering** from edge cases the automation can't handle.

### Finding 2: Structured logging exists but lacks aggregation and alerting
Key events are logged via `logStructured()` / `logStructuredError()`:
- `checkout_session_created`, `checkout_session_create_failed`
- `webhook_processing_failed`, `webhook_signature_missing`, `webhook_timestamp_out_of_range`
- `subscription_fetch_failed`, `entitlements_fetch_failed`
- `download_authorization_failed`
- `credits_consumed`, `credit_consumption_already_processed`

However, there is no evidence of:
- Log aggregation or search infrastructure
- Alerting rules triggered by these events
- Dashboards visualizing payment health
- Periodic reconciliation jobs

### Finding 3: Checkout-to-subscription gap is the highest-risk reconciliation target
The flow `checkout.session.completed` → `insertOrUpdateSubscription()` is the critical link between "user paid" and "user gets access." If a webhook delivery fails after Stripe retries are exhausted, the user has paid but has no subscription record. Currently there is no mechanism to detect this state.

The `checkout_sessions` table records sessions with status, but there is no periodic job that checks for sessions stuck in a non-complete state whose corresponding Stripe session has actually been paid.

### Finding 4: Download fulfillment has no delivery confirmation
`DownloadAuthorizer.Authorize()` returns a presigned S3 URL, but there is no tracking of whether the user actually downloaded the artifact. A failed download after authorization is invisible to the operator.

### Finding 5: No admin-facing payment operations exist
The API has `requireAdmin()` middleware but no admin endpoints for:
- Viewing payment/subscription status for a customer
- Manually granting or revoking entitlements
- Issuing refund-adjacent actions (credit adjustments, subscription overrides)
- Reviewing failed webhooks or stuck checkout sessions

## Limitations
- This is round 1 — investigation has not yet explored Stripe's built-in dashboard/alerting capabilities as a complement to custom guardrails.
- The exact log output format and destination (stdout, file, external service) has not been verified.
- Credit consumption patterns and volume are unknown, making it hard to set anomaly thresholds.

## Actions
<!-- TBD — actions will be defined as the research converges on specific guardrail recommendations -->
