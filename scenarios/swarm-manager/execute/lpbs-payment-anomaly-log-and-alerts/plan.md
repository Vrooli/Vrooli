# Plan: Payment anomaly log + configurable webhook alert delivery

## 1. Purpose

Create a unified `payment_anomaly_log` table and an outbound alert dispatcher so that all guardrail-detected payment anomalies (stuck checkout sessions, missing subscriptions, repeated authorization-without-fulfillment, credit-spike anomalies, webhook processing failures) are recorded in one durable place and surfaced to operators in near-real-time via a configurable webhook (Slack/Discord/generic monitoring endpoint), subject to a rate limiter that prevents alert storms.

This plan is the substrate that downstream items (`execute/lpbs-stripe-reconciliation-ticker`, `execute/lpbs-download-delivery-confirmation`, `execute/lpbs-admin-payment-ops`) write into.

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Plus the research conclusion this item derives from:

```bash
swarm-manager backlog file-get --kind research --name lpbs-payment-guardrails-and-reconciliation-plan --path conclusion.md
```

Authoritative source files (read before writing code):

- `scenarios/landing-page-business-suite/api/main.go:1009-1020` — existing `intro_anomaly_log` schema (the pattern being generalized and migrated away from)
- `scenarios/landing-page-business-suite/api/stripe_coupon_service.go:585-620` — existing `logIntroAnomaly()` insert pattern + structured-log fallback on insert failure
- `scenarios/landing-page-business-suite/api/stripe_webhook_service.go:129-135` — webhook event dispatch (where new anomaly emits will be wired)
- `scenarios/landing-page-business-suite/api/payment_settings_service.go` — existing settings storage (table created at `main.go:830-837`); pattern for new alert-config fields
- `scenarios/landing-page-business-suite/api/payment_settings_handlers.go` — admin GET/UPDATE handler shape; secret-redaction + reveal pattern
- `scenarios/landing-page-business-suite/api/test_helpers_test.go` — testcontainers postgres setup used by all integration tests
- `scenarios/landing-page-business-suite/docs/internal/SEAMS.md` — must be updated when new seams are added

## 3. Problem Statement

Today, anomalies in the payment pipeline are written to one of three places:

1. `intro_anomaly_log` table (only intro-coupon fraud signals)
2. `logStructured*()` JSON lines on stdout (269+ events; never aggregated)
3. Nowhere at all — many guardrail signals (stuck sessions, missed subscriptions, fulfillment gaps) have no detection code at all because the downstream items will need this surface to write into

There is no operator notification path. An operator only learns about a problem when a customer complains. Once the reconciliation ticker (`execute/lpbs-stripe-reconciliation-ticker`) and download-fulfillment tracker (`execute/lpbs-download-delivery-confirmation`) start producing detections, they need a place to write them and a way to page a human.

## 4. Scope

### In Scope

- New `payment_anomaly_log` table with a generalized schema (covers all anomaly types from Findings 2/4/6/7 of the research conclusion)
- A `PaymentAnomalyService` with a `Log(ctx, anomaly)` method that records the row and triggers alert dispatch
- An `AnomalyAlertDispatcher` that POSTs a JSON payload to a configurable webhook URL when an anomaly is recorded, with:
  - A simple rate limiter (per anomaly_type) that suppresses alerts beyond a configured rate
  - HTTP retry with bounded backoff (max 3 attempts) and dead-letter behaviour (record dispatch failures back into the anomaly row, do not block the caller)
  - Caller-side fire-and-forget: `Log(...)` returns as soon as the row is committed; dispatch happens on a background goroutine
- New columns on `payment_settings` exposing: `anomaly_webhook_url`, `anomaly_webhook_enabled`, per-type rate-limit overrides (JSONB)
- Admin GET/UPDATE handlers extended to read/write the new settings (using the existing redact-on-read / reveal-via-dedicated-endpoint pattern)
- A `LogPaymentAnomaly()` helper exported to other services so future reconciliation/fulfillment items can call it without depending on the dispatcher internals
- One reference call site: replace the `logIntroAnomaly()` insert with a forwarding call to `LogPaymentAnomaly(...)` so the existing intro-coupon detector becomes the first producer of the new pipeline (proves the seam end-to-end)
- **One-time boot migration that copies all existing `intro_anomaly_log` rows into `payment_anomaly_log` (mapping in §8g) and then `DROP`s `intro_anomaly_log`** — single source of truth going forward
- Integration tests using testcontainers (real postgres) for table writes, end-to-end dispatch via a stub HTTP server, and the intro-anomaly migration path
- Docs: `docs/reference/api/admin.md` updated for the new settings fields; `docs/internal/SEAMS.md` updated with the new dispatcher seam

### Out of Scope

- Detecting any new anomaly type (the producers — stuck-session detector, fulfillment tracker, credit-spike detector — are separate backlog items that depend on this one)
- Building the admin UI for viewing anomalies (that is `execute/lpbs-admin-payment-ops`)
- Email or in-app alert delivery (research d2=B settled on webhook-only)
- Webhook signing (HMAC of payload) — deferred until a consumer requires it
- A monetization dashboard (separate item: `execute/lpbs-monetization-dashboard`)
- A background sweeper that re-dispatches rows stuck in `dispatch_status = 'pending'` (round-1 d5=A chose fire-and-forget; a sweeper can be added later if operationally needed)

## 5. Current Technical Context

| Area | Where | Notes |
|---|---|---|
| Existing anomaly table | `api/main.go:1009-1020` | `intro_anomaly_log`: id (SERIAL), email, customer_id, coupon_id, anomaly_type, details JSONB, created_at; indexes on email/type/created_at. Will be migrated and dropped. |
| Existing anomaly insert | `api/stripe_coupon_service.go:585-620` | `logIntroAnomaly()` writes to `intro_anomaly_log`; on insert failure, logs `intro_anomaly_log_insert_failed` via `logStructuredError()` and returns silently. Will be rewritten to forward to `LogPaymentAnomaly`. |
| Settings storage | `api/main.go:830-837`, `api/payment_settings_service.go` | Singleton row (`id = 1`) with `publishable_key`, `secret_key`, `webhook_secret`, `dashboard_url`; loaded into `StripeService.ConfigSnapshot()` |
| Settings admin surface | `api/payment_settings_handlers.go` | `handleGetStripeSettings` redacts secrets; `handleRevealStripeSecret` returns one secret on demand; `handleUpdateStripeSettings` validates prefix and persists |
| Webhook event dispatch | `api/stripe_webhook_service.go:129-135` | `switch eventType` in webhook handler — future producers (e.g. fix item for atomicity) will emit anomalies from inside this handler |
| Structured logging | `api/main.go:1077-1093` | `logStructured()` / `logStructuredError()` already wrap everything in `level/message/fields/timestamp` JSON; reuse for dispatcher diagnostics |
| Test infra | `api/test_helpers_test.go` | testcontainers postgres:15-alpine via `setupTestDB(t)`; all integration tests run real schema migration |
| Schema migration entrypoint | `api/main.go` (around line 700-1020) | Inline `stmts []string` slice executed at boot; new `CREATE TABLE IF NOT EXISTS payment_anomaly_log`, `ALTER TABLE payment_settings ADD COLUMN IF NOT EXISTS ...`, and the intro-anomaly data migration (DO $$...$$) all go here |

## 6. Target End State

After this item ships:

1. A new `payment_anomaly_log` table exists with columns capturing all anomaly dimensions any downstream guardrail will need.
2. A single Go function `LogPaymentAnomaly(ctx, anomaly)` is the only way payment guardrail code records anomalies. It writes the row and (if the dispatcher is configured + enabled + within rate limits) fires a background webhook POST.
3. The Stripe settings admin surface exposes `anomaly_webhook_url`, `anomaly_webhook_enabled`, and per-type rate limits; admin can flip these without a redeploy.
4. The intro-coupon anomaly detector (the first and currently only producer) routes through the new helper. `intro_anomaly_log` is **dropped** after a one-time boot migration copies its existing rows into `payment_anomaly_log`, so there is a single source of truth for all payment anomalies, historical and current.
5. Integration tests prove: row insert + dispatch + retry + rate-limit suppression + admin save/load + intro-anomaly-log migration (both fresh-DB and upgrade-path fixtures).
6. Downstream items (`execute/lpbs-stripe-reconciliation-ticker`, `execute/lpbs-download-delivery-confirmation`) can call `LogPaymentAnomaly(...)` with new `anomaly_type` values without touching this code again.

## 7. Implementation Strategy

> **Greenfield constraint:** This is greenfield work for the `payment_anomaly_log` surface and dispatcher. Do not add compatibility shims or rename-and-keep wrappers around `logIntroAnomaly`. Replace its body to forward to `LogPaymentAnomaly` and delete any code paths that become unreachable. The old `intro_anomaly_log` table is migrated and dropped — do not leave a dual-write or a union view.

### Phase 1 — Schema + data migration (sequential, no dependencies)

1. Add `payment_anomaly_log` `CREATE TABLE IF NOT EXISTS` and indexes to the migration block in `api/main.go` (alongside `intro_anomaly_log`).
2. Add `ALTER TABLE payment_settings ADD COLUMN IF NOT EXISTS anomaly_webhook_url TEXT`, `anomaly_webhook_enabled BOOLEAN DEFAULT FALSE`, `anomaly_rate_limits JSONB DEFAULT '{}'::jsonb`.
3. Append the one-time data migration (see §8g) in the same migration block: within a single `DO $$` block, copy all `intro_anomaly_log` rows into `payment_anomaly_log` with the defined column mapping, then `DROP TABLE intro_anomaly_log`. The migration is guarded by an `information_schema` check so it no-ops on already-migrated DBs.
4. Run builds + integration tests on **two** fixtures: a clean container (fresh DB) and a container pre-seeded with `intro_anomaly_log` rows (upgrade path). Both must converge on the same post-migration schema.

### Phase 2 — Anomaly service + dispatcher (sequential)

1. Create `api/payment_anomaly_service.go` with:
   - `type PaymentAnomaly struct { ... }` (fields finalized in §8b)
   - `type PaymentAnomalyService struct { db *sql.DB; dispatcher *AnomalyAlertDispatcher; cfg atomic.Pointer[anomalyConfig] }`
   - `func (s *PaymentAnomalyService) Log(ctx, anomaly) error` — inserts row, then fires `s.dispatcher.Dispatch(ctx, row)` on a goroutine if enabled.
2. Create `api/anomaly_alert_dispatcher.go` with:
   - Per-type token-bucket rate limiter (in-process, single instance — multi-replica deployments are not in scope for LPBS today); bucket state guarded by `sync.Mutex`
   - HTTP POST with retry (3 attempts, exponential backoff: 1s, 2s, 4s capped) with 5s per-attempt timeout
   - On dispatch failure after retries, `UPDATE payment_anomaly_log SET dispatch_status='failed', dispatch_attempts=3, dispatch_error=<truncated>` on the same row (do not insert a new row — the existing one owns its own dispatch state)
3. Wire `PaymentAnomalyService` into `Server` in `api/server.go` (or wherever services are constructed) and load configured settings from `payment_settings`.
4. Add `func LogPaymentAnomaly(ctx context.Context, s *Server, anomaly PaymentAnomaly)` — package-level convenience for callers that already have the server reference.

### Phase 3 — Migrate the one existing producer (sequential, depends on Phase 2)

1. Replace the body of `logIntroAnomaly` in `api/stripe_coupon_service.go` to construct a `PaymentAnomaly` (with `SubjectKind = "intro_coupon"`, `SubjectID = couponID`) and call `LogPaymentAnomaly(...)`. Since Phase 1 drops `intro_anomaly_log`, there is no second write path.
2. Delete any helpers, struct fields, imports, or constants that become unreachable after the rewrite. No `_unused` locals, no `// removed` comments, no dead code.
3. Update any `stripe_coupon_*_test.go` assertions that query `intro_anomaly_log` directly to query `payment_anomaly_log WHERE subject_kind = 'intro_coupon'` instead.

### Phase 4 — Admin settings surface (sequential, depends on Phase 1)

1. Extend `PaymentSettingsService.GetStripeSettings` / `SaveStripeSettings` to read/write the three new columns.
2. Extend `handleGetStripeSettings` to return `anomaly_webhook_url` (treated as a secret — redacted in GET, available via `handleRevealStripeSecret` with a new allowed field `anomaly_webhook_url`).
3. Extend `handleUpdateStripeSettings` to accept and validate the new fields. Validate URL via existing `ValidateURL()`. Reject if `anomaly_webhook_enabled = true` and URL is empty.
4. After save, call `stripeService.RefreshConfig(ctx)` AND `paymentAnomalyService.RefreshConfig(ctx)` so the dispatcher picks up the change without restart.

### Phase 5 — Tests (sequential, depends on Phases 1-4)

1. `payment_anomaly_service_test.go` — testcontainers postgres; covers: insert returns nil, row visible, dispatcher invoked, dispatcher skipped when disabled, dispatcher skipped when over rate limit, retry-then-fail records `dispatch_status='failed'`, and `TestMigration_IntroAnomalyLog` verifies a pre-seeded source table is fully migrated.
2. `anomaly_alert_dispatcher_test.go` — `httptest.Server` stub; covers: POST body shape, retry on 5xx, no retry on 4xx, header set, timeout enforced.
3. `payment_settings_service_test.go` — extend existing test file to cover load/save of the three new fields including the JSONB rate-limit overrides.
4. `payment_settings_handlers_test.go` — extend to cover redaction, reveal, validation rejection (enabled=true with empty URL), and `RefreshConfig` propagation.
5. `stripe_coupon_test.go` — confirm intro coupon anomalies now appear in `payment_anomaly_log` with `subject_kind = 'intro_coupon'`.

### Phase 6 — Docs (parallel with Phase 5)

1. `docs/reference/api/admin.md` — document the three new fields on the Stripe settings endpoints.
2. `docs/internal/SEAMS.md` — add `PaymentAnomalyService.Log` and `AnomalyAlertDispatcher.Dispatch` as new seams; describe the in-process rate limiter as a known single-instance constraint; note `intro_anomaly_log` has been removed.
3. `docs/reference/STRIPE_WEBHOOKS.md` — add a short "Anomaly alerts" subsection pointing operators at the new settings.

### Phase 7 — Cleanup & Verification (sequential, final)

1. `cd scenarios/landing-page-business-suite/api && gofumpt -w .`
2. `cd scenarios/landing-page-business-suite/api && go build ./...` — fix all errors, **including pre-existing**.
3. `cd scenarios/landing-page-business-suite/api && golangci-lint run` — fix all warnings in modified files, **including pre-existing**.
4. `cd scenarios/landing-page-business-suite/api && go test ./... -timeout 600s` — fix all failures, **including pre-existing**.
5. **Do not restart the running scenario.** Write code only — the user will restart manually after review.

## 8. Contract Decisions

### 8a. `payment_anomaly_log` schema

```sql
CREATE TABLE IF NOT EXISTS payment_anomaly_log (
    id BIGSERIAL PRIMARY KEY,
    anomaly_type VARCHAR(64) NOT NULL,
    severity VARCHAR(16) NOT NULL DEFAULT 'warn',  -- info | warn | error | critical
    email VARCHAR(255),
    customer_id VARCHAR(255),
    subject_id VARCHAR(255),                       -- generic subject (session_id, asset_id, subscription_id, coupon_id, etc.)
    subject_kind VARCHAR(64),                      -- 'checkout_session' | 'subscription' | 'download_asset' | 'intro_coupon' | ...
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    dispatch_status VARCHAR(16) NOT NULL DEFAULT 'pending',  -- pending | sent | skipped | failed
    dispatch_attempts INTEGER NOT NULL DEFAULT 0,
    dispatched_at TIMESTAMP,
    dispatch_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_payment_anomaly_log_type ON payment_anomaly_log(anomaly_type);
CREATE INDEX IF NOT EXISTS idx_payment_anomaly_log_email ON payment_anomaly_log(email);
CREATE INDEX IF NOT EXISTS idx_payment_anomaly_log_subject ON payment_anomaly_log(subject_kind, subject_id);
CREATE INDEX IF NOT EXISTS idx_payment_anomaly_log_created ON payment_anomaly_log(created_at);
CREATE INDEX IF NOT EXISTS idx_payment_anomaly_log_dispatch_pending ON payment_anomaly_log(created_at) WHERE dispatch_status = 'pending';
```

### 8b. `PaymentAnomaly` Go struct

```go
type PaymentAnomaly struct {
    Type        string                 // e.g. "checkout_subscription_missing"
    Severity    string                 // info | warn | error | critical (default warn)
    Email       string                 // optional
    CustomerID  string                 // optional
    SubjectID   string                 // optional opaque id
    SubjectKind string                 // optional, only when SubjectID is set
    Details     map[string]interface{} // optional, marshalled to JSONB
}
```

### 8c. Outbound webhook payload (POST body)

```json
{
  "id": 1234,
  "type": "checkout_subscription_missing",
  "severity": "error",
  "email": "user@example.com",
  "customer_id": "cus_...",
  "subject": { "kind": "checkout_session", "id": "cs_..." },
  "details": { "stripe_event_id": "evt_..." },
  "created_at": "2026-04-18T12:34:56Z",
  "scenario": "landing-page-business-suite"
}
```

Header: `Content-Type: application/json`, `User-Agent: lpbs-anomaly-dispatcher/1`. No HMAC signing (deferred). Generic JSON only — Slack/Discord rendering is the integrator's problem (round-1 d2=A).

### 8d. Rate limiter

- **Algorithm:** in-process token bucket per `anomaly_type`.
- **Defaults:** `burst = 5`, `refill = 1 token / 60s` (i.e., max one alert per minute per type, with burst headroom).
- **Per-type overrides:** read from `payment_settings.anomaly_rate_limits` JSONB shaped as `{"<type>": {"burst": N, "refill_seconds": M}}`.
- **When suppressed:** the row is still inserted with `dispatch_status = 'skipped'` and `dispatch_error = 'rate_limited'`. The row remains visible to operators so suppression is auditable.
- **Thread safety:** bucket state is guarded by `sync.Mutex`; concurrent `Log()` calls serialize only on the bucket lookup + token check, not on the HTTP POST.

### 8e. Dispatch failure handling

- HTTP timeout: 5 seconds per attempt.
- Retry on 5xx and on transport errors (connect, dns, tls). No retry on 2xx (success) or 4xx (caller misconfigured the webhook URL — log and stop).
- Backoff: 1s, 2s, 4s.
- On final failure, update the row: `dispatch_status = 'failed'`, `dispatch_attempts = 3`, `dispatch_error = '<truncated body, max 512 chars>'`. Do NOT insert a second anomaly row for the dispatch failure (the existing row owns its own dispatch state — this avoids alert-storm-on-broken-webhook).

### 8f. Admin API surface

- `GET /api/v1/admin/stripe/settings` — adds `anomaly_webhook_url` (redacted: empty string when set, omitted when unset) and `anomaly_webhook_enabled`, `anomaly_rate_limits` (shown plain) to the response.
- `GET /api/v1/admin/stripe/settings/reveal?field=anomaly_webhook_url` — returns the unredacted URL.
- `PATCH /api/v1/admin/stripe/settings` — accepts `anomaly_webhook_url`, `anomaly_webhook_enabled`, `anomaly_rate_limits`. Validation: if `anomaly_webhook_enabled = true`, `anomaly_webhook_url` must be a valid HTTPS URL.

### 8g. `intro_anomaly_log` → `payment_anomaly_log` migration mapping

Executed once as part of the Phase 1 migration block. Idempotent: no-ops if the source table no longer exists.

```sql
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'intro_anomaly_log') THEN
        INSERT INTO payment_anomaly_log
            (anomaly_type, severity, email, customer_id, subject_id, subject_kind, details, dispatch_status, created_at)
        SELECT
            anomaly_type,
            'warn'          AS severity,
            email,
            customer_id,
            coupon_id       AS subject_id,
            'intro_coupon'  AS subject_kind,
            COALESCE(details, '{}'::jsonb) AS details,
            'skipped'       AS dispatch_status,
            created_at
        FROM intro_anomaly_log;
        DROP TABLE intro_anomaly_log;
    END IF;
END $$;
```

Column mapping:

| `payment_anomaly_log` column | Source | Notes |
|---|---|---|
| `anomaly_type` | `intro_anomaly_log.anomaly_type` | copied verbatim |
| `severity` | constant `'warn'` | historical rows have no severity; `warn` is the service default |
| `email` | `intro_anomaly_log.email` | copied |
| `customer_id` | `intro_anomaly_log.customer_id` | copied |
| `subject_id` | `intro_anomaly_log.coupon_id` | the coupon is the subject |
| `subject_kind` | constant `'intro_coupon'` | all source rows are coupon-related |
| `details` | `COALESCE(intro_anomaly_log.details, '{}'::jsonb)` | preserve JSONB; default if null |
| `dispatch_status` | constant `'skipped'` | historical rows are not re-dispatched |
| `created_at` | `intro_anomaly_log.created_at` | preserve original timestamp |
| `dispatch_attempts`, `dispatched_at`, `dispatch_error` | unset | defaults (0, null, null) apply |
| `id` | new `BIGSERIAL` | old `id` is discarded — rows are internally renumbered |

## 9. Testing Plan

All tests are automated; no manual checklist.

| Layer | File | What it proves |
|---|---|---|
| Schema | `payment_anomaly_service_test.go::TestSchemaApplied` | The `CREATE TABLE` and indexes exist after `setupTestDB` |
| Migration (fresh DB) | `payment_anomaly_service_test.go::TestMigration_FreshDB` | Fresh container starts with no `intro_anomaly_log` and no historical rows |
| Migration (upgrade path) | `payment_anomaly_service_test.go::TestMigration_IntroAnomalyLog` | A container pre-seeded with `intro_anomaly_log` rows has those rows copied to `payment_anomaly_log` with the correct mapping, and `intro_anomaly_log` is dropped |
| Migration idempotency | `payment_anomaly_service_test.go::TestMigration_ReRun` | Running migrations twice does not duplicate rows and does not error |
| Insert | `payment_anomaly_service_test.go::TestLog_InsertsRow` | `Log()` writes a row with all fields correctly |
| Dispatch enabled | `payment_anomaly_service_test.go::TestLog_DispatchesWhenEnabled` | With a stub HTTP server, POST is received with the expected JSON body |
| Dispatch disabled | `payment_anomaly_service_test.go::TestLog_NoDispatchWhenDisabled` | With `anomaly_webhook_enabled = false`, no HTTP call is made; row marked `skipped` |
| Rate limit | `payment_anomaly_service_test.go::TestLog_RateLimited` | Burst+1 alerts of the same type produce `burst` dispatches and 1 row marked `skipped/rate_limited` |
| Per-type override | `payment_anomaly_service_test.go::TestLog_PerTypeRateLimit` | A type with override `{burst:1, refill_seconds:3600}` is suppressed faster |
| Retry on 5xx | `anomaly_alert_dispatcher_test.go::TestDispatch_RetriesOn5xx` | Stub returns 503, then 200 — exactly 2 attempts, success recorded |
| No retry on 4xx | `anomaly_alert_dispatcher_test.go::TestDispatch_NoRetryOn4xx` | Stub returns 400 — exactly 1 attempt, `failed` recorded |
| Retry exhaustion | `anomaly_alert_dispatcher_test.go::TestDispatch_RetryExhausted` | Stub returns 503 three times — row updated to `failed` with truncated error |
| Settings load | `payment_settings_service_test.go::TestGetStripeSettings_AnomalyFields` | New columns round-trip through GetStripeSettings |
| Settings save validation | `payment_settings_handlers_test.go::TestUpdate_RejectsEnabledWithoutURL` | PATCH with enabled=true and empty URL returns 400 |
| Settings reveal | `payment_settings_handlers_test.go::TestRevealAnomalyWebhookURL` | Reveal endpoint returns plaintext URL |
| Refresh config | `payment_settings_handlers_test.go::TestUpdate_RefreshesAnomalyConfig` | After PATCH, `paymentAnomalyService` reflects new URL/enabled/limits |
| Migration of existing producer | `stripe_coupon_test.go` (extend existing) | Intro coupon anomaly now appears in `payment_anomaly_log` with `subject_kind='intro_coupon'` |

Run command: `cd scenarios/landing-page-business-suite/api && go test ./... -timeout 600s`

## 10. Rollout / Validation Checklist

- [ ] All Phase 7 cleanup commands pass without errors
- [ ] All new tests pass
- [ ] All pre-existing tests still pass
- [ ] `docs/internal/SEAMS.md` updated
- [ ] `docs/reference/api/admin.md` updated
- [ ] `docs/reference/STRIPE_WEBHOOKS.md` updated
- [ ] No new `_unused` variables, `// removed` comments, or compatibility wrappers
- [ ] `intro_anomaly_log` has been dropped after its rows were migrated into `payment_anomaly_log`
- [ ] User restarts scenario manually and confirms `/api/v1/admin/stripe/settings` returns the three new fields

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Outbound webhook target is down or slow → blocks API request thread | M | H | Dispatch on goroutine; 5s HTTP timeout per attempt; bounded retry |
| Misconfigured webhook URL produces alert storm via retries | L | H | No retry on 4xx; cap at 3 attempts per row; rate limiter is per-type so a flood of one anomaly cannot exhaust the dispatcher for others |
| In-process rate limiter is not shared across replicas | L | M | LPBS is single-instance today; documented as a known constraint in SEAMS.md. If multi-instance is needed later, swap for a postgres-backed limiter |
| JSONB `anomaly_rate_limits` shape drift | M | L | Validate shape at config load time; reject invalid shapes with a clear error; default to in-code defaults if column is null/empty |
| Boot-time migration fails partway (INSERT succeeds but DROP fails, or vice versa) | L | H | Wrap INSERT + DROP in a single `DO $$` block inside one transaction; `information_schema` guard + `DROP TABLE IF EXISTS` make re-runs no-op; integration test covers both fresh-DB and upgrade-path fixtures |
| Dispatcher goroutine leaks on shutdown | L | L | Use a context with the server's shutdown context; bounded retry means goroutines exit within ~7s of fire; goroutine count bounded by rate limiter (burst × type count) |
| Race between `RefreshConfig` and an in-flight dispatch reads stale settings | L | L | Acceptable — settings are eventually consistent; `atomic.Pointer[anomalyConfig]` for the config pointer avoids torn reads; an in-flight POST uses the URL captured at dispatch start |

## 12. Non-goals / Prohibited Patterns

- **No new anomaly types.** This item only builds the substrate. The reconciliation ticker, fulfillment tracker, and credit-spike detectors are separate items.
- **No HMAC payload signing.** Add only when a consumer requires it.
- **No email/SMS/PagerDuty integration.** Webhook-only per research d2=B.
- **No background worker that polls `payment_anomaly_log` for un-dispatched rows.** Dispatch is fire-and-forget at insert time (round-1 d5=A). A retry sweep can be added later if operationally needed.
- **No compatibility wrapper around `logIntroAnomaly`.** Replace its body to forward; do not keep a thin "kept for back-compat" function.
- **No dual-write to `intro_anomaly_log` nor a union view.** The old table is migrated and dropped. All historical rows live in `payment_anomaly_log` after migration.
- **No `_unused` vars or `// removed` comments.** Greenfield: delete what becomes unreachable.
- **No restart of the running scenario.** Write code; user restarts manually.

## 13. Definition of Done

The plan is done when **all** of the following are true:

1. `payment_anomaly_log` table is created on fresh-container boot via the existing migration block.
2. `payment_settings` has three new columns and the admin GET/PATCH/reveal endpoints expose them with the same secret-handling pattern as existing fields.
3. `PaymentAnomalyService.Log(ctx, anomaly)` is the sole code path for recording payment anomalies, and `stripe_coupon_service.go::logIntroAnomaly` forwards through it. `intro_anomaly_log` has been dropped after its rows were migrated into `payment_anomaly_log`.
4. End-to-end integration test (`TestLog_DispatchesWhenEnabled`) passes against testcontainers postgres + httptest stub.
5. Rate limiter test (`TestLog_RateLimited`) proves that burst+1 dispatches produce `burst` sends and one `skipped/rate_limited` row.
6. Retry tests prove correct 4xx/5xx behaviour and final-failure recording on the row.
7. Migration tests (`TestMigration_IntroAnomalyLog`, `TestMigration_ReRun`) prove upgrade-path correctness and idempotency.
8. `gofumpt`, `go build`, `golangci-lint`, and `go test` all pass cleanly with no pre-existing issues remaining in modified files.
9. `docs/internal/SEAMS.md`, `docs/reference/api/admin.md`, and `docs/reference/STRIPE_WEBHOOKS.md` are updated.
10. No backwards-compatibility shims, no dead code, no `_unused` variables.
11. The user has restarted the scenario manually and confirmed the new admin settings surface responds.
