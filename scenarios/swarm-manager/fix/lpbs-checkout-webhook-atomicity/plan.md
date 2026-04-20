# Plan: Wrap checkout_sessions status update + subscription insert in a single transaction

## 1. Purpose

Close the silent partial-failure mode in the LPBS Stripe `checkout.session.completed` webhook handler where `checkout_sessions.status = 'complete'` can commit successfully while the subscription insert that should follow it fails — leaving a user who has paid with no subscription and no detection signal.

## 2. Required Reading

```bash
prompt-manager skill read scientific-debugging idempotency-replay-safety-hardening seam-discovery-and-enforcement
```

## 3. Problem Statement

**Observed behavior (Finding 6 of research `lpbs-payment-guardrails-and-reconciliation-plan`):**

In `scenarios/landing-page-business-suite/api/stripe_webhook_service.go`, `handleCheckoutCompleted()` (lines 154–228) performs two DB writes on a successful subscription checkout:

1. `UPDATE checkout_sessions SET status='complete', subscription_id=…, …` via bare `s.db.Exec()` (lines 191–197)
2. Branch into `handleSubscriptionCompletion()` (line 226 → definition 230–285) which `INSERT … ON CONFLICT` into `subscriptions` via bare `s.db.Exec()` (lines 245–250)
3. On plan tiers with scheduled downgrades, `createSubscriptionSchedule()` (line 256) inserts into `subscription_schedules` via bare `s.db.Exec()` (~line 308)

No `sql.Tx` wraps these writes. If step 2 or step 3 fails after step 1 commits, the checkout session is marked complete but the user has no subscription row. Stripe's retry of the same webhook event will short-circuit at the `if session.Status == "complete" { return nil }` check at lines 184–189 because that flag already flipped, so the retry will not heal the state. The existing `credit_transactions.stripe_event_id` idempotency does not cover this branch.

**Why this is a silent failure:** no `payment_anomaly_log` exists today, `checkout_sessions.status='complete'` without a matching `subscriptions` row produces no operator-visible signal, and Stripe dashboards show the payment as successful.

**Canonical pattern already in the codebase:** `handleCustomerUpdated()` at `stripe_webhook_service.go:587–712` wraps its multi-table email migration in `s.db.Begin()` / `tx.Rollback()` / `tx.Commit()`. `addCredits()` at `stripe_credit_service.go:140–214` does the same for credit topups. A generic helper `WithTransaction(ctx, db, opts, fn)` exists at `dbhelpers.go:210–259`.

## 4. Scope

**In scope:**
- Wrap the subscription-completion branch of `handleCheckoutCompleted()` (checkout_sessions UPDATE + subscription INSERT + optional subscription_schedules INSERT) in a single DB transaction.
- Pass a `*sql.Tx` through `handleSubscriptionCompletion()` and `createSubscriptionSchedule()` so their SQL runs inside the outer transaction.
- Add an integration test that injects a failure in the subscription insert and asserts the checkout_sessions row does NOT advance to `status='complete'`.
- Add an integration test that confirms a replay of the same webhook after a rolled-back failure succeeds and leaves state consistent.

**Out of scope:**
- Adding a `stripe_event_id` idempotency column on `checkout_sessions` or `subscriptions` (Action 3 / `execute/lpbs-payment-anomaly-log-and-alerts` will add idempotency + anomaly log infrastructure broadly).
- A periodic Stripe reconciliation ticker (Action 2 / `execute/lpbs-stripe-reconciliation-ticker`).
- Wrapping the credit-topup branch (already atomic inside `addCredits()`).
- Wrapping the supporter-contribution branch (separate code path, not named in the research finding; will be addressed if Action 2's reconciliation finds gaps).
- Any schema migration to add FK between `checkout_sessions` and `subscriptions`.

## 5. Current Technical Context

Key files and symbols:

| File | Symbol | Lines | Role |
|------|--------|-------|------|
| `scenarios/landing-page-business-suite/api/stripe_webhook_service.go` | `handleCheckoutCompleted` | 154–228 | Top-level handler; does bare `db.Exec` UPDATE on checkout_sessions and branches |
| `scenarios/landing-page-business-suite/api/stripe_webhook_service.go` | `handleSubscriptionCompletion` | 230–285 | Bare `db.Exec` INSERT into subscriptions |
| `scenarios/landing-page-business-suite/api/stripe_webhook_service.go` | `createSubscriptionSchedule` | ~290–320 | Bare `db.Exec` INSERT into subscription_schedules |
| `scenarios/landing-page-business-suite/api/stripe_webhook_service.go` | `handleCustomerUpdated` | 587–712 | Reference pattern: multi-table tx with `s.db.Begin()` / `Commit()` |
| `scenarios/landing-page-business-suite/api/dbhelpers.go` | `WithTransaction` | 210–259 | Canonical transaction helper `(ctx, *sql.DB, *sql.TxOptions, func(*sql.Tx) error) error` |
| `scenarios/landing-page-business-suite/api/dbhelpers.go` | `WithSerializableTransaction`, `WithReadCommittedTransaction` | 261–291 | Isolation-level variants |
| `scenarios/landing-page-business-suite/api/stripe_credit_service.go` | `addCredits` | 140–214 | Reference pattern: idempotent insert in tx |
| `scenarios/landing-page-business-suite/api/stripe_handlers_test.go` | `TestHandleCheckoutCreateAndWebhookEndToEnd`, `TestHandleStripeWebhookCreditTopup` | 71–221 | Existing end-to-end webhook tests (testcontainers) |
| `scenarios/landing-page-business-suite/api/stripe_integration_test.go` | `TestFlow_MultipleCheckouts_SameUser`, `TestFlow_WebhookRetry_Idempotent` | 583–780+ | Existing integration tests — extend for atomicity |

Schema (from test setup in `stripe_integration_test.go`):
- `checkout_sessions(session_id UNIQUE, status, subscription_id, customer_id, customer_email, …)` — no `stripe_event_id`
- `subscriptions(subscription_id UNIQUE, status, plan_tier, price_id, bundle_key, …)` — no `stripe_event_id`, no FK to checkout_sessions
- `subscription_schedules(schedule_id UNIQUE, subscription_id, …)` — written by `createSubscriptionSchedule()`

## 6. Target End State

- `handleCheckoutCompleted()`'s subscription branch opens a single `*sql.Tx` (via `WithTransaction`) that encloses: the `UPDATE checkout_sessions` statement, the `INSERT … ON CONFLICT DO UPDATE` into `subscriptions`, and (when applicable) the `INSERT` into `subscription_schedules`.
- Any error inside the transaction rolls all three writes back. The handler returns the error so Stripe retries.
- `handleSubscriptionCompletion()` and `createSubscriptionSchedule()` accept a `*sql.Tx` (or an `Execer` interface) and use it instead of `s.db`.
- `checkout_sessions.status='complete'` is never observed without a matching `subscriptions` row (absent direct DB manipulation).
- A rolled-back handler invocation leaves state unchanged; a subsequent Stripe retry re-enters the branch and succeeds (both table writes have `ON CONFLICT` upserts, so retry is safe even if the first attempt partially wrote before rollback).

## 7. Implementation Strategy (phased)

**Phase 1 — Thread tx through the subscription helpers**
- Change `handleSubscriptionCompletion` signature from `(subscriptionID, customerID, customerEmail string, plan *PlanOption, session *checkoutSessionRecord, amountCents int64) error` to accept a `*sql.Tx` parameter.
- Change `createSubscriptionSchedule` the same way.
- Replace `s.db.Exec(...)` inside these functions with `tx.Exec(...)`.
- No public API change; both are unexported methods on `*StripeService`.

**Phase 2 — Wrap the subscription branch in a transaction**
- In `handleCheckoutCompleted()`, move the `UPDATE checkout_sessions` statement from lines 191–197 so that, on the subscription branch, it runs inside `WithTransaction(ctx, s.db, nil, func(tx *sql.Tx) error {...})`.
- Inside the closure: run the UPDATE, call `handleSubscriptionCompletion(tx, …)`, call `createSubscriptionSchedule(tx, …)` if applicable.
- The credit-topup and supporter branches continue to use `s.db` directly (out of scope). Refactor so the UPDATE is *not* duplicated across branches — the cleanest shape is to branch first and have each branch own its own UPDATE semantics.

**Phase 3 — Preserve pre-branch duplicate detection**
- The existing `if session.Status == "complete" { return nil }` short-circuit at lines 184–189 must stay. It reads the session status before opening the transaction. Confirm this read is safe (read-committed default is fine for a short-circuit).

**Phase 4 — Tests**
- Add `TestHandleCheckoutCompleted_SubscriptionInsertFailure_RollsBackStatus` in `stripe_webhook_service_test.go` (or the existing `stripe_integration_test.go`): inject a failure by, e.g., pre-inserting a conflicting `subscriptions` row with an incompatible value that causes the UPSERT to fail, or by stubbing the `PlanOption` with a value that fails the INSERT, and assert the `checkout_sessions.status` remains `pending` and no `subscriptions` row exists.
- Add `TestHandleCheckoutCompleted_SubscriptionInsertRetry_Succeeds`: trigger a rollback, then replay the same webhook event, assert both rows end up correct.
- Ensure `TestFlow_WebhookRetry_Idempotent` (line 668+) still passes — retries of a successful completion must remain no-ops via the pre-branch status check.

**Phase 5 — Verification**
```bash
cd scenarios/landing-page-business-suite/api
gofumpt -w stripe_webhook_service.go stripe_webhook_service_test.go
go build ./...
go test ./... -run 'TestHandleCheckoutCompleted|TestFlow_WebhookRetry|TestFlow_MultipleCheckouts|TestHandleStripeWebhook' -timeout 300s -v
golangci-lint run
```

## 8. Contract Decisions

- **Transaction helper:** Use `WithTransaction(ctx, s.db, nil, fn)` from `dbhelpers.go:210` rather than raw `s.db.Begin()` / defer-rollback / `Commit()`. Rationale: helper already standardises rollback-on-error and panic safety; `handleCustomerUpdated` predates the helper.
- **Tx propagation:** Pass `*sql.Tx` as the first parameter to `handleSubscriptionCompletion` and `createSubscriptionSchedule`. No `Execer` interface abstraction — the functions are unexported and only ever used from the webhook path.
- **Isolation level:** Default (read committed). Rationale: the short-circuit at line 184 guards against concurrent duplicate processing; serializable would add cost without closing a known gap in this scope.
- **Idempotency on retry:** Rely on existing `ON CONFLICT (session_id) DO UPDATE` on `checkout_sessions` and `ON CONFLICT (subscription_id) DO UPDATE` on `subscriptions`. No new idempotency column in scope.
- **Error surface:** Any tx error is returned from `handleCheckoutCompleted`; the webhook endpoint returns HTTP 500, Stripe retries per its exponential-backoff schedule.

## 9. Testing Plan

| Test | Location | Verifies |
|------|----------|----------|
| `TestHandleCheckoutCompleted_SubscriptionInsertFailure_RollsBackStatus` | `stripe_webhook_service_test.go` (new) | On induced subscription INSERT failure, `checkout_sessions.status` stays `pending` and no `subscriptions` row is created |
| `TestHandleCheckoutCompleted_ScheduleInsertFailure_RollsBackAll` | `stripe_webhook_service_test.go` (new) | On induced schedule INSERT failure, both `checkout_sessions` and `subscriptions` rows are rolled back |
| `TestHandleCheckoutCompleted_SubscriptionBranch_HappyPath_UsesSingleTx` | `stripe_webhook_service_test.go` (new) | Happy path: both rows materialise; verified by post-handler queries |
| `TestHandleCheckoutCompleted_RetryAfterRollback_Succeeds` | `stripe_integration_test.go` (extend) | After a rolled-back failure, a Stripe retry with the same event_id succeeds (both `ON CONFLICT` upserts work under retry) |
| `TestFlow_WebhookRetry_Idempotent` | `stripe_integration_test.go:668+` (existing) | Must still pass; short-circuit at line 184 preserves no-op behavior on a second successful replay |
| `TestFlow_MultipleCheckouts_SameUser` | `stripe_integration_test.go:583–664` (existing) | Must still pass; unchanged branch behavior |

All tests use the existing testcontainers postgres setup via `setupTestDB(t)`.

## 10. Rollout / Validation Checklist

- [ ] `go build ./...` clean
- [ ] `go test ./... -timeout 300s` passes (including all 4 new tests)
- [ ] `golangci-lint run` clean
- [ ] `gofumpt -l` reports no changes needed
- [ ] Manual smoke: restart LPBS API locally, run the existing stripe simulate script (or a webhook replay) and confirm a successful subscription creates both rows
- [ ] `docs/internal/SEAMS.md` updated if any new testability seam was introduced (e.g., a plan-option injection point for failure tests) — review only, likely no change needed

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Wrapping `createSubscriptionSchedule` in tx lengthens the transaction and holds locks longer | Low | Low | Writes are to tables keyed by `subscription_id`; no contention across users |
| Pre-branch `checkout_sessions` UPDATE (lines 191–197) is shared by credit-topup and supporter branches; moving it inside the subscription branch could regress those paths | Medium | Medium | After refactor, each branch owns its own UPDATE with the correct status/subscription_id values; keep a single test per branch to guard |
| Failure tests are flaky due to induced-failure mechanism | Low | Low | Use a deterministic INSERT conflict (e.g., pre-seed `subscriptions` with the target `subscription_id` and a schema-level constraint mismatch on a NOT NULL column) rather than timing-based failure |
| Stripe retry on rollback could drift into a concurrent-handler race with a fresh webhook | Low | Low | The short-circuit at line 184 plus `ON CONFLICT` upserts keeps both paths convergent; the broader concurrent-duplicate race is explicitly out-of-scope and covered by Action 2 / Action 3 |
| `insertOrUpdateSubscription` was referenced by the research as the target name; actual function is `handleSubscriptionCompletion` | N/A | Low | Research naming mismatch noted; implementation targets the real symbol |

## 12. Non-goals / Prohibited Patterns

- Do not add a `stripe_event_id` column to `checkout_sessions` or `subscriptions` — belongs to Action 3.
- Do not introduce a periodic reconciliation job — belongs to Action 2.
- Do not add advisory locks, distributed locks, or external coordinators — the short-circuit + `ON CONFLICT` already handle retry; concurrent-duplicate-webhook detection is explicitly deferred.
- Do not introduce new DB helper abstractions (e.g., a generic `WithWebhookTransaction`). Use the existing `WithTransaction`.
- Do not widen scope to the supporter-contribution or credit-topup branches.
- Do not change Stripe webhook response semantics (keep 500-on-error for retryable failures, 200 for no-ops).

## 13. Definition of Done

- [ ] `handleSubscriptionCompletion` and `createSubscriptionSchedule` accept a `*sql.Tx`
- [ ] `handleCheckoutCompleted`'s subscription branch wraps all three writes in one `WithTransaction(...)` call
- [ ] All four new/extended tests pass
- [ ] All pre-existing webhook tests still pass
- [ ] Build, lint, and format pass cleanly
- [ ] Plan items 11 (Risks) mitigations are each reflected by either a test or a code-level guard
- [ ] Code review confirms no write to `checkout_sessions` or `subscriptions` in the subscription branch bypasses the new `*sql.Tx`
