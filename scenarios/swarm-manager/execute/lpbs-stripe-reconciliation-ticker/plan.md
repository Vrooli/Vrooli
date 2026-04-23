# Plan: Stripe reconciliation ticker for paid-but-unmaterialized sessions

## 1. Purpose

Add a belt-and-suspenders background reconciliation loop to the LPBS API that, on a periodic tick, queries Stripe for recently-paid checkout sessions and flags any that do not have a matching local `subscriptions` row. Each detection writes a `payment_anomaly_log` row (type `checkout_subscription_missing`) and — per the item description — attempts re-materialization via the existing subscription-insert path.

This is the second rung of the payment-assurance ladder. Rung 1 (`fix/lpbs-checkout-webhook-atomicity`, completed) closes the transactional gap between `checkout_sessions.status='complete'` and `subscriptions` INSERT. Rung 2 (this item) catches every failure mode atomicity cannot: pre-fix historical data, webhook deliveries dropped while the API was down, deliberately deleted/replayed Stripe events, and any unknown-unknown that leaves Stripe and the local DB diverged.

This is greenfield work. Do not include compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables.

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement idempotency-replay-safety-hardening
```

Plus the research conclusion and sibling plans this item builds on:

```bash
swarm-manager backlog file-get --kind research --name lpbs-payment-guardrails-and-reconciliation-plan --path conclusion.md
swarm-manager backlog file-get --kind fix --name lpbs-checkout-webhook-atomicity --path plan.md
swarm-manager backlog file-get --kind execute --name lpbs-payment-anomaly-log-and-alerts --path plan.md
```

Authoritative source files (read before writing code):

- `scenarios/landing-page-business-suite/api/stripe_webhook_service.go` — `handleCheckoutCompleted`, `handleSubscriptionCompletion`, `createSubscriptionSchedule` (the re-materialization target)
- `scenarios/landing-page-business-suite/api/stripe_service.go` (or wherever the Stripe client is constructed) — the shared `*stripe.Client` / API key plumbing
- `scenarios/landing-page-business-suite/api/payment_anomaly_service.go` — `PaymentAnomalyService.Log(ctx, anomaly)` and the package-level `LogPaymentAnomaly(...)` helper
- `scenarios/landing-page-business-suite/api/payment_settings_service.go` — singleton `payment_settings` row, the push-on-PATCH refresh pattern
- `scenarios/landing-page-business-suite/api/main.go` — migration `stmts []string` slice, server/service wiring
- `scenarios/landing-page-business-suite/api/test_helpers_test.go` — testcontainers postgres harness
- `scenarios/landing-page-business-suite/docs/internal/SEAMS.md` — must be updated when a new seam is added

## 3. Problem Statement

Even after `fix/lpbs-checkout-webhook-atomicity` lands, divergence between Stripe and the local subscription state remains possible in at least three scenarios:

1. **Pre-fix historical data.** `checkout_sessions.status='complete'` rows written before the atomicity fix shipped can be in an "orphan complete" state — paid in Stripe, no local `subscriptions` row, no signal. The atomicity fix does not heal history.
2. **Webhook deliveries missed while the API is down.** Stripe retries `checkout.session.completed` up to ~3 days with exponential backoff, but a longer outage — or a retry that expires during maintenance — can exhaust retries with no successful delivery. Once retries stop, Stripe never tries again.
3. **Deliberately deleted or replayed Stripe events.** Operators occasionally delete or reroute webhook events during debugging; unknown-unknown failure modes can also produce silent gaps. The atomicity fix only protects inside a delivery; it does not detect missing deliveries.

None of these produce operator-visible signal today. An operator only learns about the divergence when a customer complains.

## 4. Scope

### In Scope

- A new `StripeReconciliationTicker` service that owns a goroutine started on server boot and cancelled on shutdown.
- A reconciliation function `ReconcileOnce(ctx)` that (a) queries Stripe for recently-paid checkout sessions since the last successful reconciliation, (b) cross-checks each against local `checkout_sessions` + `subscriptions`, (c) for any paid session with no matching local subscription, writes a `payment_anomaly_log` row with `anomaly_type='checkout_subscription_missing'` and attempts re-materialization through the existing subscription-insert path, (d) on any Stripe error, records a `payment_anomaly_log` row with `anomaly_type='reconciliation_stripe_error'` and skips the tick.
- A small `reconciliation_state` table (or new column on an existing singleton) holding the cursor — `last_reconciled_at` timestamp or `last_event_id` — so ticks resume from where the previous tick ended rather than re-scanning the same window.
- Integration with the existing `PaymentAnomalyService` (already shipped) for all recording and alerting. The ticker never writes to `payment_anomaly_log` directly — always via `LogPaymentAnomaly(...)`.
- Configuration surface for interval, enable flag, and lookback overlap (see §8 decision resolution). On boot, the ticker reads config; on PATCH, it hot-reloads via the same push-refresh pattern used by the anomaly dispatcher.
- Integration tests using testcontainers (real postgres) + a stubbed Stripe session-list endpoint covering: missing-subscription detection + re-materialization, Stripe-error anomaly path, idempotency under double-tick, cursor advancement, interval hot-reload, shutdown-ctx cancellation mid-tick.
- Docs: `docs/internal/SEAMS.md` updated with the new seam; `docs/reference/STRIPE_WEBHOOKS.md` updated with a short "Reconciliation" subsection.

### Out of Scope

- Reconciling credit wallets, supporter contributions, or download entitlements (separate items; this ticker is subscription-only per the item description).
- Detecting subscription _state_ drift (e.g., Stripe says cancelled but local says active) — this is listed as a possible future anomaly type but is deliberately deferred pending operator demand.
- Backfilling deep history beyond the configured lookback window. The ticker walks forward from boot; a separate one-shot CLI / admin endpoint can backfill pre-existing orphan-complete rows later if needed.
- Multi-replica leader election. LPBS runs single-instance today; the ticker assumes that and documents the assumption in SEAMS.md. If LPBS ever scales horizontally, a postgres advisory lock can be added at that time.
- Adding a new admin UI for viewing reconciliation state. The `payment_anomaly_log` rows it produces are visible through whatever surface `execute/lpbs-admin-payment-ops` exposes.

## 5. Current Technical Context

| Area | Where | Notes |
|---|---|---|
| Stripe client construction | `api/stripe_service.go` (TBD — confirm during Phase 1) | Shared `*stripe.Client`; the ticker will accept a `sessionLister` interface satisfied by the real client and a stub in tests |
| Webhook re-entry point | `api/stripe_webhook_service.go::handleCheckoutCompleted` / `handleSubscriptionCompletion` / `createSubscriptionSchedule` | The functions the ticker will call for re-materialization — exact signature (and whether they need a shared helper extracted) is a plan decision, see §8 |
| Anomaly logging | `api/payment_anomaly_service.go::PaymentAnomalyService.Log`, package-level `LogPaymentAnomaly` | Completed. Reuse unchanged. |
| Settings + push-refresh pattern | `api/payment_settings_service.go`, `api/payment_settings_handlers.go`; `handleUpdateStripeSettings` already calls `stripeService.RefreshConfig(ctx)` and `paymentAnomalyService.RefreshConfig(ctx)` | New service will add `ReconciliationTicker.RefreshConfig(ctx)` if config is source-of-truth from `payment_settings` |
| Checkout + subscription tables | `api/main.go:~603` (checkout_sessions), `api/main.go` (subscriptions) | `checkout_sessions(session_id UNIQUE, status, subscription_id, customer_id, customer_email, created_at, …)`; `subscriptions(subscription_id UNIQUE, plan_tier, price_id, …)` — both have upsert keys the re-materialization path relies on |
| Migration slice | `api/main.go` `stmts []string` around payment tables | New `CREATE TABLE IF NOT EXISTS reconciliation_state` + possibly `ALTER TABLE payment_settings ADD COLUMN IF NOT EXISTS reconcile_*` go here |
| Logging | `api/main.go::logStructured`, `logStructuredError` | Ticker uses these for per-tick operational lines (`reconciliation_tick_started`, `reconciliation_tick_completed`, etc.) |
| Test harness | `api/test_helpers_test.go::setupTestDB` | testcontainers postgres:15-alpine; real migrations; reconciliation tests will pre-seed `checkout_sessions` rows and stub the Stripe client |

## 6. Target End State

After this item ships:

1. On LPBS API boot, a `StripeReconciliationTicker` goroutine starts and ticks at the configured interval (default 5 minutes). It cancels cleanly on server shutdown via the shared shutdown context.
2. Each tick queries Stripe for recently-paid checkout sessions created since the cursor (minus a configured overlap window), walks the results, and for each paid session with no local `subscriptions` row writes a `checkout_subscription_missing` anomaly via `LogPaymentAnomaly(...)` and attempts re-materialization through the canonical subscription-insert path.
3. The cursor advances only when the tick completes without a fatal Stripe error. On fatal error, the tick records a `reconciliation_stripe_error` anomaly, leaves the cursor untouched, and returns — the next tick retries the same window (driving at-least-once re-scan).
4. Re-materialization is idempotent: a paid session that a concurrent webhook has just materialized is a no-op thanks to `ON CONFLICT (subscription_id) DO UPDATE` on `subscriptions`. Double-ticking the same window produces the same state regardless of interleavings.
5. `payment_anomaly_log` becomes the single durable record of any divergence the ticker ever saw — historical and current — and routes through the already-built webhook alert dispatcher when configured.
6. Integration tests prove: detection + re-materialization path, Stripe-error path, cursor advancement, idempotency under re-scan, config hot-reload, shutdown cancellation.
7. Downstream items (monetization dashboard, admin payment ops) can query `payment_anomaly_log WHERE anomaly_type='checkout_subscription_missing'` and `reconciliation_state` without touching this code again.

## 7. Implementation Strategy

> **Greenfield constraint:** Greenfield ticker, greenfield cursor table. No compatibility shims. If the existing subscription-insert helpers need to be extracted or parameterized to be callable from both the webhook path and the ticker path, do the refactor cleanly — do not introduce a second near-duplicate function "just for reconciliation".

### Phase 1 — Schema + cursor state (sequential, first)

1. Add `CREATE TABLE IF NOT EXISTS reconciliation_state (id INT PRIMARY KEY DEFAULT 1 CHECK (id=1), last_reconciled_at TIMESTAMPTZ, last_stripe_created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ DEFAULT NOW())` to the migration block in `api/main.go`. Singleton row pattern matches `payment_settings`.
2. If decision §8/d3 resolves to `payment_settings`-backed config, add `ALTER TABLE payment_settings ADD COLUMN IF NOT EXISTS reconcile_interval_seconds INT DEFAULT 300`, `reconcile_enabled BOOLEAN DEFAULT TRUE`, `reconcile_lookback_overlap_seconds INT DEFAULT 600`.
3. Seed an initial `reconciliation_state` row on first boot (`INSERT ... ON CONFLICT DO NOTHING`) with `last_stripe_created_at = NOW() - reconcile_lookback_overlap_seconds`, so the first tick scans only a bounded recent window rather than all-time history.

### Phase 2 — Reconciliation service skeleton (sequential)

1. Create `api/stripe_reconciliation_service.go` with:
   - `type sessionLister interface { ListPaidSessionsSince(ctx, cursor time.Time) ([]PaidSessionSummary, error) }` — the seam for swapping a stub in tests.
   - `type StripeReconciliationTicker struct { db *sql.DB; stripe sessionLister; anomaly *PaymentAnomalyService; subscriptionRematerializer subscriptionRematerializer; cfg atomic.Pointer[reconcileConfig]; shutdownCtx context.Context; tickerC <-chan time.Time }`
   - `type subscriptionRematerializer interface { InsertOrUpdateFromStripeSession(ctx, session PaidSessionSummary) error }` — interface over the existing `handleSubscriptionCompletion` path (see Phase 3 for how this is satisfied).
   - `func (t *StripeReconciliationTicker) Start() error` — launches the goroutine bound to `shutdownCtx`; returns immediately.
   - `func (t *StripeReconciliationTicker) ReconcileOnce(ctx context.Context) error` — the per-tick body, exposed for tests + possible future admin "reconcile now" endpoint.
   - `func (t *StripeReconciliationTicker) RefreshConfig(ctx context.Context) error` — reloads config from `payment_settings` via `atomic.Pointer[reconcileConfig].Store(...)`.
2. Wire `StripeReconciliationTicker` into `Server` construction. Pass the shutdown context and the shared Stripe client. Call `.Start()` after the rest of the server is up.
3. The goroutine body is a classic `for { select { case <-ctx.Done(): return; case <-ticker.C: _ = t.ReconcileOnce(ctx) } }`. The ticker interval is re-read from `cfg.Load()` each iteration so `RefreshConfig` takes effect on the next tick (within ≤ old interval).

### Phase 3 — Re-materialization seam (sequential, depends on Phase 2)

The item description says to "attempt re-materialization via the existing `insertOrUpdateSubscription()` path". The real symbol is `handleSubscriptionCompletion` (see research code-verification). We need a callable entry point that works both from the webhook path (which passes a `stripe.Event` plus parsed session) and from the ticker path (which has only a `PaidSessionSummary` fetched by list).

Two options are presented as a decision (see §8/d4). Whichever is chosen, this phase implements exactly one adapter — no parallel paths.

- If §8/d4 = A (direct helper call): extract a small `(tx *sql.Tx) subscribeFromSession(ctx, tx, parsedSession)` helper from the current `handleSubscriptionCompletion` body. The webhook path and the ticker path both call it via `WithTransaction(...)`. No interface needed in production; `subscriptionRematerializer` is test-only.
- If §8/d4 = B (event replay): fetch the corresponding `checkout.session.completed` event from Stripe by session id and invoke `handleWebhookEvent` with the captured event JSON. The adapter owns the fetch + dispatch; `handleWebhookEvent` is unchanged.

Either way: the ticker never bypasses `ON CONFLICT (subscription_id) DO UPDATE` on `subscriptions`, so a concurrent webhook materializing the same session is harmless.

### Phase 4 — Reconciliation tick body (sequential, depends on Phases 1–3)

1. `ReconcileOnce(ctx)`:
   1. Load cursor from `reconciliation_state` (singleton row).
   2. `sessions, err := t.stripe.ListPaidSessionsSince(ctx, cursor - overlap)`. On error, `LogPaymentAnomaly(...)` with `anomaly_type='reconciliation_stripe_error'`, return the error (cursor unchanged).
   3. For each session: fast-path lookup in local `subscriptions` by `subscription_id` (when present in the Stripe session) or by `checkout_sessions.session_id` → `subscription_id`. If present and plan_tier matches, skip.
   4. If no local subscription row: call `LogPaymentAnomaly(...)` with `anomaly_type='checkout_subscription_missing'`, `subject={kind:'checkout_session', id:session.id}`, `details={stripe_subscription_id, customer_id, price_id, amount}`. Then call `subscriptionRematerializer.InsertOrUpdateFromStripeSession(ctx, session)`.
   5. If re-materialization fails: `LogPaymentAnomaly(...)` with `anomaly_type='reconciliation_rematerialize_failed'`, `details={error: truncated}`. Do NOT propagate — keep processing the rest of the batch.
   6. After all sessions processed, UPDATE `reconciliation_state.last_stripe_created_at = max(session.created)` and `last_reconciled_at = NOW()` in a single statement.
2. Structured logging at tick start (`reconciliation_tick_started` with cursor), tick completed (`reconciliation_tick_completed` with `sessions_scanned`, `anomalies_recorded`, `rematerialized_count`, `duration_ms`), and on Stripe error (`reconciliation_tick_failed`).

### Phase 5 — Tests (sequential, depends on Phases 1–4)

All tests use testcontainers postgres + a stub `sessionLister` and stub `subscriptionRematerializer`. No `time.Sleep` — the ticker's `time.Ticker` is swapped for a test channel so tests drive ticks explicitly.

1. `stripe_reconciliation_service_test.go::TestReconcileOnce_DetectsMissingSubscription` — seed `checkout_sessions` with status=pending, stub Stripe to return one paid session, assert `payment_anomaly_log` gets a `checkout_subscription_missing` row and re-materializer is called once.
2. `TestReconcileOnce_SkipsMatchedSubscription` — seed both `checkout_sessions` and `subscriptions` with a matching row, stub Stripe to return the same paid session, assert no anomaly row is written and re-materializer is not called.
3. `TestReconcileOnce_StripeErrorRecordsAnomalyAndHoldsCursor` — stub returns a transport error; assert a `reconciliation_stripe_error` row is written and `reconciliation_state.last_stripe_created_at` is unchanged.
4. `TestReconcileOnce_RematerializeFailureRecordsAnomalyAndContinuesBatch` — stub re-materializer errors on the first session and succeeds on the second; assert first produces `reconciliation_rematerialize_failed`, second completes.
5. `TestReconcileOnce_IdempotentUnderDoubleScan` — run twice with the same stub output; assert anomaly_log rows are deduplicated (or accepted — decision note: a duplicate anomaly is recorded per-tick by design, but re-materialization is a no-op via `ON CONFLICT`).
6. `TestReconcileOnce_AdvancesCursorToMaxSessionCreated` — stub returns three sessions with staggered `created` times; assert `reconciliation_state.last_stripe_created_at` is the max after success.
7. `TestRefreshConfig_TakesEffectOnNextTick` — PATCH `reconcile_interval_seconds`; assert the next `cfg.Load()` returns the new value.
8. `TestStart_CancelsOnShutdown` — start the goroutine, cancel the shutdown ctx, assert the goroutine returns within 1s.
9. `TestReconcileOnce_MidTickCancellation` — cancel ctx while the stub is returning results; assert partial state is not persisted (cursor update is the last step).

### Phase 6 — Docs + cleanup (parallel with Phase 5)

1. `docs/internal/SEAMS.md` — add `StripeReconciliationTicker.Start`, `ReconcileOnce`, `RefreshConfig`, the `sessionLister` interface as the Stripe seam, and the `subscriptionRematerializer` interface as the webhook-path seam. Note single-instance assumption and how to re-evaluate for multi-replica.
2. `docs/reference/STRIPE_WEBHOOKS.md` — add a short "Reconciliation ticker" subsection pointing operators at the new settings fields, the `reconciliation_state` table, and the `checkout_subscription_missing` / `reconciliation_stripe_error` / `reconciliation_rematerialize_failed` anomaly types.
3. Run `gofumpt -w`, `go build ./...`, `go test ./... -timeout 600s`, `golangci-lint run` on the touched package. Fix all lint/type/test issues in modified files — including pre-existing ones.

### Phase 7 — Handoff

User manually restarts the scenario and confirms the ticker emits `reconciliation_tick_completed` log lines at the configured cadence. Claude does not run `vrooli scenario restart`.

## 8. Contract Decisions

Decisions listed below are **pending** for round 001 — each is presented as a decision item in `workshop/round-001.json`. The plan sections above reference these decisions by id; once resolved, this section is updated to record the selection.

- **d1 — Stripe API query strategy.** How to enumerate recently-paid sessions: `CheckoutSessions.List()` with `created[gte]` filter and client-side `payment_status=paid` filter, or `Events.List()` filtered by `type=checkout.session.completed`. `<!-- TBD -->`
- **d2 — Cursor advancement strategy.** Persisted cursor vs fixed sliding window; overlap size. `<!-- TBD -->`
- **d3 — Configuration source.** Env var vs `payment_settings` row vs hardcoded constant. `<!-- TBD -->`
- **d4 — Re-materialization mechanism.** Direct helper call into extracted `subscribeFromSession` vs event replay via `handleWebhookEvent`. `<!-- TBD -->`
- **d5 — Concurrency guard against in-flight webhook.** Rely on `ON CONFLICT` idempotency alone, or add a short `pg_try_advisory_lock` keyed by `session_id`. `<!-- TBD -->`

Additional standing contract points (not decisions — these hold regardless of d1-d5):

- The ticker never writes to `payment_anomaly_log` directly; always via `LogPaymentAnomaly(...)` so the dispatcher and rate limiter apply.
- Re-materialization is fire-and-complete within the tick; no background retry queue. A failed re-materialize produces an anomaly row and continues the batch.
- Cursor advances only on whole-tick success. On Stripe list error, cursor is held; on per-session re-materialize error, cursor still advances (the anomaly is the durable record).
- Default tick interval is 300 seconds (5 minutes) per the item description. Min 60s, max 3600s — enforced at config-load time.
- Default lookback overlap is 600 seconds (10 minutes). Larger than one tick interval so a tick that fails transiently does not leave a gap; small enough that repeated re-scans are cheap.

## 9. Testing Plan

Covered in Phase 5 above. Acceptance criteria:

- All nine listed tests pass deterministically across 5 consecutive local runs.
- No test uses `time.Sleep` for synchronization.
- The stubbed `sessionLister` is the sole Stripe interaction surface — no network calls in tests.
- Integration tests run inside the existing testcontainers harness; no new container images.

## 10. Rollout / Validation Checklist

- [ ] `go build ./...` clean
- [ ] `go test ./... -timeout 600s` passes (all new tests + pre-existing)
- [ ] `golangci-lint run` clean on the API package (fix pre-existing warnings in touched files)
- [ ] `gofumpt -l` reports no changes
- [ ] `docs/internal/SEAMS.md` updated
- [ ] `docs/reference/STRIPE_WEBHOOKS.md` updated
- [ ] User manually runs `vrooli scenario restart landing-page-business-suite` and confirms `reconciliation_tick_completed` log lines appear on the configured cadence
- [ ] User confirms `GET /api/v1/admin/stripe/settings` returns the new reconcile fields (if d3 = `payment_settings`)

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Ticker double-processes a session concurrently with an in-flight webhook, producing a race on `subscriptions` | M | L | `ON CONFLICT (subscription_id) DO UPDATE` keeps both paths convergent; d5 (advisory lock) considered if this proves insufficient |
| Stripe rate-limits the `sessions.list` (or `events.list`) call under heavy reconciliation load | L | M | Default 5-min interval + cursor + overlap bound list size; per-call timeout; on 429, record anomaly and hold cursor (next tick retries) |
| Cursor drifts — ticker thinks it's caught up but missed a session that was `created` before cursor and paid after | M | M | Overlap window (default 600s) re-scans a buffer around the cursor; if Stripe `created` and `payment` times diverge beyond 600s, increase overlap via config |
| Re-materialization path diverges from webhook path, leaving local state subtly different | L | H | d4 resolution forces a single shared helper (option A) or a single shared dispatcher (option B); tests assert re-materialized rows match webhook-produced rows column-for-column |
| Multi-replica deployment launches two tickers that duplicate work | L | L | Single-instance assumption documented in SEAMS.md; advisory lock on `reconciliation_state.id` added at multi-replica time (not this item) |
| Config hot-reload races with an in-flight tick | L | L | `atomic.Pointer[reconcileConfig].Load()` gives a single snapshot per tick; a PATCH mid-tick takes effect on the next tick boundary |
| Stub Stripe `sessionLister` in tests drifts from real Stripe client shape | L | M | `sessionLister` interface returns a minimal `PaidSessionSummary` struct (fields listed in §5) which the real client adapter populates explicitly; schema changes in the Stripe SDK break the adapter, not the interface |
| Pre-fix historical data contains hundreds of orphan-complete rows and first tick floods `payment_anomaly_log` | M | M | Initial cursor seeded at `NOW() - overlap` so first tick only sees recent activity; a separate one-shot backfill is deliberately deferred (§4 Out of Scope) |
| Ticker goroutine panics crash the API | L | H | Wrap tick body in `recover()`; on panic, log structured error and continue ticking |

## 12. Non-goals / Prohibited Patterns

- **No reconciliation of credit wallets, supporter contributions, or downloads.** Subscription-only per the item description.
- **No multi-replica leader election.** Single-instance assumption holds.
- **No background retry queue for failed re-materialization.** A failed re-materialize produces an anomaly row; the next tick retries the same session naturally via overlap.
- **No time.Sleep in tests.** Use a test-driven ticker channel for determinism.
- **No duplicate subscription-insert helper.** If the webhook path needs to be refactored to share a helper with the ticker path, refactor cleanly (d4=A) or replay events (d4=B) — do not create a parallel "reconciliation-only" insert path.
- **No direct writes to `payment_anomaly_log`.** Always go through `LogPaymentAnomaly(...)`.
- **No Claude-initiated `vrooli scenario restart`.** User runs it manually.
- **No `_unused` vars, `// removed` comments, or dead code.**
- **No test-only production code branches.** Stubs live in `_test.go` files; production code uses the real interface.

## 13. Definition of Done

1. `StripeReconciliationTicker` exists and starts on server boot, cancels on shutdown.
2. `ReconcileOnce(ctx)` produces the five documented log events and three anomaly types correctly.
3. Cursor in `reconciliation_state` advances on success, holds on Stripe list error.
4. Re-materialization runs through the canonical subscription-insert path (decision d4 resolved) and is idempotent under concurrent webhook delivery.
5. All configuration is hot-reloadable via the push-on-PATCH pattern (if d3 = `payment_settings`) or settable via env (if d3 = env).
6. All nine tests in Phase 5 pass deterministically.
7. `SEAMS.md` and `STRIPE_WEBHOOKS.md` updated.
8. `gofumpt`, `go build`, `golangci-lint`, and `go test` all clean (including pre-existing issues in modified files).
9. No backwards-compatibility shims, no dead code, no `_unused` variables.
10. User has run `vrooli scenario restart landing-page-business-suite` manually and confirmed `reconciliation_tick_completed` log lines.
