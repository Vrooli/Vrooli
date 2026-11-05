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

8. **Credit authority is account-owned and source-attributed.** Funding grants require a business account, customer, positive amount, funding kind, provider source, external event, and explicit live/test mode. Historical email-wallet balances and bonuses never become spendable lots. Grant, ledger link, and funding lot commit atomically; exact event replays succeed and conflicting grant data fails.
9. **Meter kind is independent of app scope.** `cost_based` uses account funding; `count_based` uses the authenticated principal's non-credit quota and custom reservation window. Each owner rejects the other kind. The exact legacy `app_specific/ai_credits` money mapping is retained only with valid conversion and app scope; ambiguous rows fail closed. Global and applicable app caps must both hold before money admission.
10. **Credit admission precedes dispatch.** The public AI door authenticates only LPBS-issued device/session keys and rechecks account membership, parent revocation and expiry. Keys bind actor, account, app, optional harness/model and cumulative credit caps. Caller billing metadata and outside-funded markers cannot bypass the wallet. Global account limits, device/session holds and spendable lots precede private Gateway execution. App-specific cap binding remains unavailable to the legacy inference adapters, which fail closed.
11. **Credit settlement is immutable and capped.** Account reservation state follows SM2 in [funding flows](../concepts/FLOWS.md). Exact replay cannot allocate twice; conflicting replay fails. Measured provider cost may exceed the reserved ceiling, but customer charge and lot allocation cannot. Ambiguous dispatched outcomes retain a hold for reconciliation. A disconnect or timeout alone does not authorize release. Count quota remains separate and never touches funding lots or credit events.

## Configuration

12. **`config/variants/*.json` is the seed source for landing variants; `site_settings` is the authoritative source for mutable branding and mail settings.** The runtime `ConfigStore` imports branding once when the singleton row is empty and thereafter reads and writes the durable database row.
13. **`.vrooli/plans.json` is the source of truth for pricing.** Database `bundle_prices` may exist for in-flight data but does not override the file.
14. **`metrics_events.event_type` is constrained.** Only the values listed in the `CHECK` constraint (`page_view`, `scroll_depth`, `click`, `form_submit`, `conversion`, `download`) are accepted; ingestion rejects others with `400`.

## Anomalies & alerts

15. **Every detected payment anomaly produces exactly one `payment_anomaly_log` row.** The dispatcher may retry sending the alert N times; that is reflected in `dispatch_attempts`, never in duplicate rows.
16. **Anomaly detection is best-effort and never fails the originating request.** A failure in `payment_anomaly` writes a structured log line but does not propagate to the caller.

## Routing

17. **Every public-facing route is also reachable when coming-soon mode is on.** The UI's `PublicRouteGuard` wraps public routes — when `branding.coming_soon_enabled = true`, those routes render `ComingSoonPage` *instead of* their normal content but are still mounted.
18. **Admin and user-auth routes are never gated by coming-soon mode.** `/admin/*`, `/admin/login`, `/auth/login`, and `/auth/verify` always render their real content.

## Tests

19. **Evidence matches its boundary.** SQL mocks prove narrow query and rejection behavior; SQLite exercises deterministic domain transactions; isolated real Postgres proves funding order, attribution, and concurrent SQL behavior. None alone proves authenticated production admission, app binding, provider receipts, or live payment behavior.

## AI door and recovery

`internal/aidoor` owns `/api/v1/ai/chat/completions`, `/responses`, device/session keys and receipt reconciliation. AI Gateway is a required private dependency. It owns provider rules, dispatch and durable measured-cost receipts. LPBS has no production OpenRouter client composition. Legacy role adapters await caller migration and remain unavailable for paid execution.

A dispatch retains its wallet hold beyond ordinary expiry. LPBS restart reconciles pending operations from Gateway receipts; repeated requests never repeat provider dispatch. A verified pre-dispatch rejection releases the hold; ambiguous outcomes retain it. Streaming handlers clear the ordinary HTTP write deadline per response. Journey tests use real PostgreSQL and real HTTP handlers/processes, with only Stripe, upstream inference and explicitly labelled approval data as fixtures.

## Replay and funding reversals

Signed Stripe dispute/refund/early-warning events match an attributed, mode-matched payment lot. Unknown payments return an error for retry. The event payload, lot mutation, payment state, loss rollup and Money Ledger outbox commit under the same account lock used by reservation settlement. Identical event replay is a no-op; changed payload under the same event identity fails. Cumulative refund amounts cannot decrease. Delayed inquiry snapshots cannot demote formal chargebacks or release their funding. A genuine late win after a lost dispute corrects returned funds without clearing a separate purchase block or another active dispute.

At the metered Door boundary, `Idempotency-Key` is scoped by the authenticated key ID and mapped to a stable operation ID. Repeating the same request finds the existing reservation, reads the Gateway receipt, and applies exact settlement replay through the existing idempotent settlement owner; it does not dispatch a second provider request. The Door returns `409` for that duplicate, including when the receipt confirms an already settled operation; if the receipt cannot be read it returns `409` while reconciliation remains pending. The duplicate response does not include `X-Operation-ID`, so callers retain the ID from the first successful response. `journeytest/composed_money_door_test.go` verifies the observed conflict status, the original reservation identity, stable Gateway receipt semantics, and unchanged request/account/global money snapshots. This is a selected isolated account/request observation, not a claim of arbitrary account equivalence or full SM1/SM2 coverage.

A freeze raced against an already-admitted reservation still settles measured usage once, with unavailable funding attributed as loss to its original lot. New requests cannot spend frozen/refunded credit; remaining credit never becomes negative. Refund and dispute overlap share one derived loss row, avoiding duplicate provider cost. Immutable Stripe balance-transaction IDs own gross principal movements and fees, including partial reinstatements while a dispute remains lost; older snapshots cannot erase returned funds or replace an existing transaction's money facts. Lost disputes block new checkout and grants. Source-policy age/hold/cap values remain operator-owned data; no production trust values are seeded.

Production main starts the reservation-cleanup worker; server shutdown cancels and joins it before closing the database. The worker dispatches committed reversal deltas through Money Ledger's typed ingestion API. Separate live/test sinks select books and accounts. An uncertain acknowledgement retries the same immutable external identity. USD provider cost is rounded cumulatively to currency minor units before delta calculation; refund/fee currency follows the lot. Corrections append journal entries. `journeytest/fraud_test.go` and `funding_ledger_test.go` prove these boundaries against real Postgres, Gateway, and Money Ledger modules/database; Stripe test-card operation remains unverified without operator keys.
