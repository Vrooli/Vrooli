---
title: "Invariants"
description: "Properties that must hold across every state transition"
category: "internal"
order: 103
audience: ["developers"]
internal: true
---

# Invariants

Properties that must hold **at all times** across the data + behavior of the scenario. If any of these can be observed broken, that is a bug — not a config issue.

## Identity & sessions

1. **Exactly one seeded admin row exists.** `admin_users.id = 1` is the seeded admin; `seedDefaultData` deletes any other admin with the same email and upserts the row. No code path may introduce a second admin without a deliberate migration.
2. **Admin sessions and user sessions are disjoint.** A row in `admin_sessions` cannot satisfy `requireUserAuth`, and a row in `user_sessions` cannot satisfy `requireAdmin`.
3. **End-user JWT subject = `users.id`.** A token verified by `requireUserAuth` always resolves to a single live row in `users`. If the user is deleted, all their tokens MUST be invalidated by row-level cascade (`auth_tokens.user_id … ON DELETE CASCADE`).

## Pricing & money

4. **A `free` plan always costs $0.** `PlanService` rejects free-tier price updates that would set a non-zero amount.
5. **`credits` and `donation` plan tiers are one-time.** `bundle_prices.billing_interval = 'one_time'` for these tiers; subscription-style intervals are forbidden.
6. **No bundle price can reference a Stripe product from a different bundle.** Plan updates and Stripe imports enforce bundle↔product alignment to prevent cross-product contamination.
7. **Stripe webhook delivery is idempotent.** Re-delivery of a `stripe_event_id` we have already processed is a no-op (matches Stripe's retry contract).

## Credits & usage

8. **A finalized credit reservation is immutable.** `credit_reservations.status` transitions are `pending → finalized | released | expired`. No row may go back to `pending`.
9. **Usage cannot exceed a non-`-1` tier limit.** A `usage_records.usage_amount > limit_value` row, where `limit_value != -1`, indicates a metering bug — usage reporting must reject the increment instead of clamping it silently.
10. **Bonus credits are spent before subscription credits.** `credit_wallets.bonus_credits` decrements first, then `balance_credits`.
11. **Credit ledger mutations are replay-safe.** `CreditWalletService` records a top-up and its wallet update in one transaction; a repeated non-empty Stripe event ID is a successful no-op. Consumption locks the wallet row and uses its caller-provided idempotency key to make a retried debit a successful no-op after the first commit. Distinct keys remain independent debits.

## Configuration

12. **`config/variants/*.json` and `config/branding.json` are the source of truth for landing config.** The runtime `ConfigStore` reflects them; a database row is never authoritative for variant or branding fields.
13. **`.vrooli/plans.json` is the source of truth for pricing.** Database `bundle_prices` may exist for in-flight data but does not override the file.
14. **`metrics_events.event_type` is constrained.** Only the values listed in the `CHECK` constraint (`page_view`, `scroll_depth`, `click`, `form_submit`, `conversion`, `download`) are accepted; ingestion rejects others with `400`.

## Anomalies & alerts

15. **Every detected payment anomaly produces exactly one `payment_anomaly_log` row.** The dispatcher may retry sending the alert N times; that is reflected in `dispatch_attempts`, never in duplicate rows.
16. **Anomaly detection is best-effort and never fails the originating request.** A failure in `payment_anomaly` writes a structured log line but does not propagate to the caller.

## Routing

17. **Every public-facing route is also reachable when coming-soon mode is on.** The UI's `PublicRouteGuard` wraps public routes — when `branding.coming_soon_enabled = true`, those routes render `ComingSoonPage` *instead of* their normal content but are still mounted.
18. **Admin and user-auth routes are never gated by coming-soon mode.** `/admin/*`, `/admin/login`, `/auth/login`, and `/auth/verify` always render their real content.

## Tests

19. **No test mocks the database.** Integration tests run against a real Postgres (matches our prod migration discipline).
