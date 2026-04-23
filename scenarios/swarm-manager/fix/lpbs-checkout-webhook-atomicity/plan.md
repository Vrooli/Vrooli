# Plan: Wrap checkout_sessions status update + subscription insert in a single transaction

## 1. Purpose

Close the silent partial-failure mode in the LPBS Stripe `checkout.session.completed` webhook handler where `checkout_sessions.status = 'complete'` can commit successfully while the subscription insert that should follow it fails — leaving a user who has paid with no subscription and no detection signal.

This is greenfield work. Do not include compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables.

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

**Canonical pattern already in the codebase:** `handleCustomerUpdated()` at `stripe_webhook_service.go:587–712` wraps its multi-table email migration in `s.db.Begin()` / `tx.Rollback()` / `tx.Commit()`. `addCredits()` at `stripe_credit_service.go:140–214` does the same for credit topups. A generic helper `WithTransaction(ctx, db, opts, fn)` exists at `dbhelpers.go:210–259` — this is the go-forward pattern.

## 4. Scope

**In scope:**
- Wrap the subscription-completion branch of `handleCheckoutCompleted()` in a single DB transaction covering three writes:
  1. `UPDATE checkout_sessions SET status='complete', …` (moved into the subscription branch)
  2. `INSERT … ON CONFLICT DO UPDATE` into `subscriptions` (via `handleSubscriptionCompletion`)
  3. `INSERT` into `subscription_schedules` whenever the plan tier has a scheduled downgrade (via `createSubscriptionSchedule`)
- Pass a `*sql.Tx` through `handleSubscriptionCompletion()` and `createSubscriptionSchedule()` so their SQL runs inside the outer transaction.
- Move the shared `UPDATE checkout_sessions` out of the pre-branch section and into each branch (subscription, credit-topup, supporter) so each branch owns its own completion write. Subscription branch runs it inside the tx; credit-topup and supporter branches run it as bare `db.Exec` alongside their existing writes.
- Add integration tests that (a) inject a failure in the subscription INSERT and assert rollback; (b) inject a failure in the subscription_schedules INSERT and assert full rollback; (c) exercise happy path; (d) confirm that a Stripe retry of the same webhook after a rolled-back failure succeeds and leaves state consistent.

**Out of scope:**
- Adding a `stripe_event_id` idempotency column on `checkout_sessions` or `subscriptions` (Action 3 / `execute/lpbs-payment-anomaly-log-and-alerts` will add idempotency + anomaly log infrastructure broadly).
- A periodic Stripe reconciliation ticker (Action 2 / `execute/lpbs-stripe-reconciliation-ticker`).
- Wrapping the credit-topup branch's subscription-relevant writes in a transaction — `addCredits()` is already atomic for its own writes; this fix only moves the per-branch `UPDATE checkout_sessions` into that branch.
- Wrapping the supporter-contribution branch in a transaction (separate code path, not named in the research finding; will be revisited if Action 2's reconciliation finds gaps).
- Any schema migration to add FK between `checkout_sessions` and `subscriptions`.
- Introducing a test-only seam on `StripeService` (e.g., a `subscriptionInsertHook`). The failure-injection strategy below uses real DB state only.

## 5. Current Technical Context

Key files and symbols:

| File | Symbol | Lines | Role |
|------|--------|-------|------|
| `scenarios/landing-page-business-suite/api/stripe_webhook_service.go` | `handleCheckoutCompleted` | 154–228 | Top-level handler; does bare `db.Exec` UPDATE on checkout_sessions and branches |
| `scenarios/landing-page-business-suite/api/stripe_webhook_service.go` | `handleSubscriptionCompletion` | 230–285 | Bare `db.Exec` INSERT into subscriptions |
| `scenarios/landing-page-business-suite/api/stripe_webhook_service.go` | `createSubscriptionSchedule` | ~290–320 | Bare `db.Exec` INSERT into subscription_schedules |
| `scenarios/landing-page-business-suite/api/stripe_webhook_service.go` | `handleCustomerUpdated` | 587–712 | Reference pattern: multi-table tx with `s.db.Begin()` / `Commit()` (predates the canonical helper) |
| `scenarios/landing-page-business-suite/api/dbhelpers.go` | `WithTransaction` | 210–259 | Canonical transaction helper `(ctx, *sql.DB, *sql.TxOptions, func(*sql.Tx) error) error` |
| `scenarios/landing-page-business-suite/api/dbhelpers.go` | `WithSerializableTransaction`, `WithReadCommittedTransaction` | 261–291 | Isolation-level variants (not used here) |
| `scenarios/landing-page-business-suite/api/stripe_credit_service.go` | `addCredits` | 140–214 | Reference pattern: idempotent insert in tx |
| `scenarios/landing-page-business-suite/api/stripe_handlers_test.go` | `TestHandleCheckoutCreateAndWebhookEndToEnd`, `TestHandleStripeWebhookCreditTopup` | 71–221 | Existing end-to-end webhook tests (testcontainers) |
| `scenarios/landing-page-business-suite/api/stripe_integration_test.go` | `TestFlow_MultipleCheckouts_SameUser`, `TestFlow_WebhookRetry_Idempotent` | 583–780+ | Existing integration tests — extend for atomicity |

Schema (from test setup in `stripe_integration_test.go`):
- `checkout_sessions(session_id UNIQUE, status, subscription_id, customer_id, customer_email, …)` — no `stripe_event_id`
- `subscriptions(subscription_id UNIQUE, status, plan_tier, price_id, bundle_key, …)` — no `stripe_event_id`, no FK to checkout_sessions; several NOT NULL columns (e.g., `plan_tier`, `price_id`) which the failure-injection strategy below exploits
- `subscription_schedules(schedule_id UNIQUE, subscription_id, …)` — written by `createSubscriptionSchedule()`

## 6. Target End State

- `handleCheckoutCompleted()`'s subscription branch opens a single `*sql.Tx` (via `WithTransaction`) that encloses all three writes: the `UPDATE checkout_sessions`, the `INSERT … ON CONFLICT DO UPDATE` into `subscriptions`, and the `INSERT` into `subscription_schedules` when the plan has a scheduled downgrade.
- Any error inside the transaction rolls all three writes back. The handler returns the error so Stripe retries.
- `handleSubscriptionCompletion()` and `createSubscriptionSchedule()` accept a `*sql.Tx` as their first parameter and use it instead of `s.db`.
- The shared pre-branch `UPDATE checkout_sessions` no longer exists; each branch owns its own completion UPDATE. Credit-topup and supporter branches invoke `s.db.Exec()` with their own status/subscription_id values; the subscription branch runs the UPDATE inside the tx.
- `checkout_sessions.status='complete'` is never observed without a matching `subscriptions` row (absent direct DB manipulation).
- A rolled-back handler invocation leaves state unchanged; a subsequent Stripe retry re-enters the branch and succeeds (both table writes have `ON CONFLICT` upserts, so retry is safe even if the first attempt partially wrote before rollback).

## 7. Implementation Strategy (phased)

**Phase 1 — Thread tx through the subscription helpers**
- Change `handleSubscriptionCompletion` signature to accept `*sql.Tx` as its first parameter.
- Change `createSubscriptionSchedule` the same way.
- Replace `s.db.Exec(...)` inside these functions with `tx.Exec(...)`.
- No public API change; both are unexported methods on `*StripeService`.

**Phase 2 — Move the `UPDATE checkout_sessions` into each branch**
- Remove the shared `UPDATE checkout_sessions` at `stripe_webhook_service.go:191–197`.
- In the subscription branch: inside `WithTransaction(ctx, s.db, nil, func(tx *sql.Tx) error { … })`, run the UPDATE as `tx.Exec(...)`, then call `handleSubscriptionCompletion(tx, …)`, then conditionally `createSubscriptionSchedule(tx, …)`.
- In the credit-topup branch: run the UPDATE as `s.db.Exec(...)` inline with the existing credit-topup work (credit writes remain atomic inside `addCredits`; this UPDATE is a straightforward follow-on).
- In the supporter branch: run the UPDATE as `s.db.Exec(...)` inline with existing supporter-path work.
- Keep the SET clause identical to today's to avoid behavior drift; the only semantic change is that the subscription branch's UPDATE now rolls back when the branch errors.

**Phase 3 — Preserve pre-branch duplicate detection**
- The existing `if session.Status == "complete" { return nil }` short-circuit at lines 184–189 must stay. It reads the session status before opening the transaction. The read uses the default read-committed isolation, which is sufficient for a short-circuit.

**Phase 4 — Failure-injection tests**

The atomicity tests force the subscription INSERT to fail deterministically by pre-seeding a conflicting `subscriptions` row whose existing values violate a NOT NULL constraint when combined with the `ON CONFLICT DO UPDATE` clause (e.g., a pre-existing row with the target `subscription_id` whose required columns force the UPSERT into an invalid state). This is the authoritative failure-injection mechanism — do not add a production-code seam (hook) for this purpose, and do not use sqlmock.

- `TestHandleCheckoutCompleted_SubscriptionInsertFailure_RollsBackStatus` (new): pre-seed `subscriptions` so the UPSERT fails; fire the webhook; assert `checkout_sessions.status` remains `pending` and the pre-seeded `subscriptions` row is unchanged.
- `TestHandleCheckoutCompleted_ScheduleInsertFailure_RollsBackAll` (new): pre-seed `subscription_schedules` to induce failure there (same mechanism); fire a webhook for a plan tier that triggers `createSubscriptionSchedule`; assert `checkout_sessions.status` remains `pending` and no `subscriptions` row materialises.
- `TestHandleCheckoutCompleted_SubscriptionBranch_HappyPath_UsesSingleTx` (new): fire a clean subscription webhook; assert both rows (and schedule row when applicable) materialise.
- `TestHandleCheckoutCompleted_RetryAfterRollback_Succeeds` (extend `stripe_integration_test.go`): induce a rollback, remove the failure-injection row, replay the same webhook; assert final state is consistent (both `ON CONFLICT` upserts succeed).

**Phase 5 — Verification**
```bash
cd scenarios/landing-page-business-suite/api
gofumpt -w stripe_webhook_service.go stripe_webhook_service_test.go
go build ./...
go test ./... -run 'TestHandleCheckoutCompleted|TestFlow_WebhookRetry|TestFlow_MultipleCheckouts|TestHandleStripeWebhook' -timeout 300s -v
go test ./... -timeout 600s
golangci-lint run
```
Fix all lint, type, and unit test issues in modified files — including pre-existing ones. The user will restart the scenario manually; do not run `vrooli scenario restart` from the executing agent.

## 8. Contract Decisions

- **Transaction helper:** Use `WithTransaction(ctx, s.db, nil, fn)` from `dbhelpers.go:210`. Rationale: helper centralises rollback-on-error and panic safety; `handleCustomerUpdated` predates the helper, so its raw-Begin pattern is not a target to emulate.
- **Transaction scope:** The tx wraps all three writes — `UPDATE checkout_sessions` + `INSERT subscriptions` + `INSERT subscription_schedules` (when applicable). A failed schedule insert rolls back the subscription as well, preventing an active subscription without its scheduled downgrade.
- **Tx propagation:** Pass `*sql.Tx` as the first parameter to `handleSubscriptionCompletion` and `createSubscriptionSchedule`. No `Execer` interface abstraction — the functions are unexported and only ever used from the webhook path.
- **Per-branch UPDATE:** Each branch (subscription, credit-topup, supporter) owns its own `UPDATE checkout_sessions`. No shared helper and no pre-branch UPDATE. Rationale: cleanest ownership; subscription branch's UPDATE belongs inside its tx; credit-topup and supporter branches stay as plain `s.db.Exec`.
- **Isolation level:** Default (read committed) via `WithTransaction(..., nil, ...)`. Rationale: the short-circuit at line 184 guards against concurrent duplicate processing; serializable would add cost without closing a gap in this scope.
- **Failure injection in tests:** Pre-seed a conflicting `subscriptions` row (and `subscription_schedules` row for the schedule test) that causes the UPSERT to fail against a schema-level constraint. No production-code seam, no sqlmock.
- **Idempotency on retry:** Rely on existing `ON CONFLICT (session_id) DO UPDATE` on `checkout_sessions` and `ON CONFLICT (subscription_id) DO UPDATE` on `subscriptions`. No new idempotency column in scope.
- **Error surface:** Any tx error is returned from `handleCheckoutCompleted`; the webhook endpoint returns HTTP 500, Stripe retries per its exponential-backoff schedule.

## 9. Testing Plan

| Test | Location | Verifies |
|------|----------|----------|
| `TestHandleCheckoutCompleted_SubscriptionInsertFailure_RollsBackStatus` | `stripe_webhook_service_test.go` (new) | On induced subscription UPSERT failure, `checkout_sessions.status` stays `pending` and no new `subscriptions` row is created |
| `TestHandleCheckoutCompleted_ScheduleInsertFailure_RollsBackAll` | `stripe_webhook_service_test.go` (new) | On induced `subscription_schedules` failure, `checkout_sessions` and `subscriptions` writes are both rolled back |
| `TestHandleCheckoutCompleted_SubscriptionBranch_HappyPath_UsesSingleTx` | `stripe_webhook_service_test.go` (new) | Happy path: all three rows materialise for a scheduled-downgrade plan; subscription row materialises for a non-scheduled plan |
| `TestHandleCheckoutCompleted_RetryAfterRollback_Succeeds` | `stripe_integration_test.go` (extend) | After a rolled-back failure, clearing the injected conflict and replaying the same webhook succeeds via the `ON CONFLICT` upserts |
| `TestFlow_WebhookRetry_Idempotent` | `stripe_integration_test.go:668+` (existing) | Must still pass; short-circuit at line 184 preserves no-op behavior on a second successful replay |
| `TestFlow_MultipleCheckouts_SameUser` | `stripe_integration_test.go:583–664` (existing) | Must still pass; unchanged branch behavior |
| Credit-topup and supporter webhook tests | `stripe_handlers_test.go`, `stripe_integration_test.go` (existing) | Must still pass; per-branch UPDATE refactor does not regress non-subscription paths |

All tests use the existing testcontainers postgres setup via `setupTestDB(t)`.

## 10. Rollout / Validation Checklist

- [ ] `go build ./...` clean
- [ ] `go test ./... -timeout 600s` passes (including all new tests and all existing webhook tests)
- [ ] `golangci-lint run` clean for the whole API package (fix pre-existing warnings in touched files too)
- [ ] `gofumpt -l` reports no changes needed
- [ ] User manually runs `vrooli scenario restart landing-page-business-suite` after reviewing the diff
- [ ] `docs/internal/SEAMS.md` reviewed — no new seam was introduced, so no change expected; confirm nothing needs updating

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Wrapping `createSubscriptionSchedule` in tx lengthens the transaction and holds locks longer | Low | Low | Writes are to tables keyed by `subscription_id`; no contention across users |
| Per-branch `UPDATE checkout_sessions` refactor regresses credit-topup or supporter branches | Medium | Medium | Keep each branch's UPDATE semantically identical to today's shared UPDATE; rely on existing credit-topup and supporter tests (and add explicit assertions in happy-path tests) to guard |
| Failure-injection via pre-seeded conflicting row becomes flaky if `ON CONFLICT DO UPDATE` clause is loosened later | Low | Medium | The test must encode the exact constraint violation it exploits (comment pointing at the NOT NULL column); schema migrations that relax that column must update the test |
| Stripe retry on rollback could drift into a concurrent-handler race with a fresh webhook | Low | Low | The short-circuit at line 184 plus `ON CONFLICT` upserts keeps both paths convergent; the broader concurrent-duplicate race is explicitly out-of-scope and covered by Action 2 / Action 3 |
| Naming mismatch: research refers to `insertOrUpdateSubscription`; the actual function is `handleSubscriptionCompletion` | N/A | Low | Implementation targets the real symbol; no rename |

## 12. Non-goals / Prohibited Patterns

- Do not add a `stripe_event_id` column to `checkout_sessions` or `subscriptions` — belongs to Action 3.
- Do not introduce a periodic reconciliation job — belongs to Action 2.
- Do not add advisory locks, distributed locks, or external coordinators — the short-circuit + `ON CONFLICT` already handle retry; concurrent-duplicate-webhook detection is explicitly deferred.
- Do not introduce new DB helper abstractions (e.g., a generic `WithWebhookTransaction`). Use the existing `WithTransaction`.
- Do not add a production-code seam (`subscriptionInsertHook` etc.) purely for test use.
- Do not use `sqlmock` for these tests — the LPBS convention is real-DB integration tests via testcontainers.
- Do not widen scope to the supporter-contribution or credit-topup branches' internal atomicity.
- Do not change Stripe webhook response semantics (keep 500-on-error for retryable failures, 200 for no-ops).

## 13. Definition of Done

- [ ] `handleSubscriptionCompletion` and `createSubscriptionSchedule` accept a `*sql.Tx` as their first parameter and use it for all writes
- [ ] `handleCheckoutCompleted`'s subscription branch wraps the `UPDATE checkout_sessions` + `INSERT subscriptions` + conditional `INSERT subscription_schedules` in one `WithTransaction(...)` call
- [ ] The shared pre-branch `UPDATE checkout_sessions` at lines 191–197 is removed; each branch owns its own UPDATE
- [ ] All four new/extended tests pass deterministically across 5 consecutive local runs
- [ ] All pre-existing webhook and integration tests still pass
- [ ] `go build`, `go test`, `golangci-lint run`, and `gofumpt` all clean (including pre-existing issues in modified files)
- [ ] Every risk in Section 11 is reflected by either a test or a code-level guard
- [ ] Code review confirms no write to `checkout_sessions` or `subscriptions` in the subscription branch bypasses the new `*sql.Tx`
