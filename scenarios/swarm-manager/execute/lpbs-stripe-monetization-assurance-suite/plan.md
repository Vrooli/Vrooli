# Deepen LPBS Stripe Monetization Assurance — Implementation Plan

## 1. Purpose

Raise the automated assurance level of LPBS's Stripe-backed monetization to launch quality. The system already has solid Stripe integration with ~19 test files covering checkout, webhooks, entitlements, and downloads. This plan targets the identified gaps — particularly around cross-concern integration, race conditions, coupon abuse, and security-sensitive flows — to ensure customers reliably receive what they pay for and cannot access paid functionality without valid payment.

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Key source files:
- `scenarios/landing-page-business-suite/api/stripe_checkout_service.go` — checkout session creation
- `scenarios/landing-page-business-suite/api/stripe_webhook_service.go` — webhook processing (714 lines)
- `scenarios/landing-page-business-suite/api/account_service.go` — entitlement fulfillment
- `scenarios/landing-page-business-suite/api/download_service.go` — download gating + DownloadAuthorizer
- `scenarios/landing-page-business-suite/api/stripe_coupon_service.go` — coupon eligibility & intro pricing

## 3. Problem Statement

While LPBS has comprehensive unit-level Stripe test coverage, several integration-level gaps exist that could lead to:
- **Revenue loss**: customers accessing paid downloads without valid subscriptions
- **Customer harm**: customers paying but failing to receive download access
- **Abuse vectors**: coupon replay, eligibility bypass, or free-access exploitation
- **Race conditions**: concurrent operations (email migration + payment, subscription status change + download) producing inconsistent state

These gaps are identified through codebase analysis, not production incidents. The goal is preventive assurance before launch.

## 4. Scope

### In Scope
- 3 new dedicated integration test files (decision: round 1 d1=A, round 2 d1=A)
- E2E tests: checkout → webhook → entitlement → download (happy path and failure modes)
- Shared `monetizationTestHarness` struct bundling StripeService + StripeWebhookService + AccountService + DownloadAuthorizer + mock Stripe server + test DB (decision: round 2 d2=A)
- Webhook resilience: out-of-order delivery for 2-3 critical event reorderings (decision: round 1 d5=A)
- Webhook signatures use real HMAC-SHA256 signing with test secret via reused/extracted `signWebhookPayload` helper (decision: round 2 d3=A)
- Intro coupon lifecycle E2E + 2-3 key abuse cases (decision: round 1 d4=A)
- Download gating integration: full 8-status subscription matrix × download authorization (decision: round 2 d4=B)
- Concurrency tests for 2 critical paths only (decision: round 1 d3=A): email migration + payment webhook, subscription status change + download authorization
- DownloadAuthorizer error propagation tests for entitlement lookup failures (decision: round 1 d6=A)

### Out of Scope
- Payment anomaly detection / reconciliation guardrails (separate backlog item: `research/lpbs-payment-guardrails-and-reconciliation-plan`)
- Stripe Dashboard or admin UI testing
- Subscription schedule phase transitions (complex Stripe-side logic, not gating for launch)
- Rate limiting implementation (operational concern, not test coverage)
- Audit logging implementation (separate concern)
- Code changes to existing Stripe services — this is a test-deepening effort only

## 5. Current Technical Context

### Key Components
| Component | File | Lines | Purpose |
|-----------|------|-------|---------|
| Checkout | `stripe_checkout_service.go` | ~400 | Session creation, price resolution, coupon application |
| Webhooks | `stripe_webhook_service.go` | ~714 | Signature verification, 8 event types, idempotency |
| Entitlements | `account_service.go` | ~200 | Subscription lookup, caching, status mapping |
| Download Gating | `download_service.go` | ~680 | DownloadAuthorizer: 4-step authorization flow |
| Coupons | `stripe_coupon_service.go` | ~150 | Eligibility, plan-specific mapping, intro pricing |

### Existing Test Coverage (Strong)
- 75 total test files, 19 Stripe-specific
- `stripe_integration_test.go` — 9 full-cycle tests (intro pricing, credits, cancel, email change, coupon abuse, multiple checkouts, webhook retry, invoice-paid)
- `stripe_idempotency_test.go` — webhook event ID deduplication
- `stripe_concurrent_test.go` — race condition basics
- `download_authorizer_test.go` — 9 unit tests covering gating logic, error propagation, input validation
- `stripe_coupon_security_test.go` — coupon eligibility edge cases
- `account_service_entitlements_test.go` — 11 tests covering entitlement status mapping

### Identified Gaps (to be filled by this plan)
1. No E2E test: checkout with intro coupon → webhook → markIntroUsed → reject re-use
2. No test: subscription expires → download access denied → re-subscribe → access restored
3. No test: out-of-order webhook delivery (invoice.paid before checkout.completed)
4. No test: concurrent email migration + payment webhook
5. No test: download gating when entitlement service errors (integration-level, not unit-level)
6. No E2E test: checkout → webhook → entitlement active → download authorized
7. No exhaustive test: all 8 Stripe subscription statuses × download authorization outcome

## 6. Target End State

A test suite that provides high confidence in these properties:
1. **Pay → Get**: Every successful payment results in correct entitlement and download access
2. **No Pay → No Get**: Expired, canceled, or absent subscriptions block gated downloads
3. **Webhook Resilience**: Out-of-order, duplicate, and delayed webhooks produce correct final state
4. **Coupon Integrity**: Intro coupons are single-use, eligibility checks are accurate, and bypass attempts fail
5. **Concurrency Safety**: 2 critical concurrent paths (email migration + payment, status change + download) resolve correctly
6. **Security**: Email normalization prevents account fragmentation; gating cannot be bypassed via parameter manipulation
7. **Exhaustive Status Coverage**: All 8 Stripe subscription statuses produce correct download authorization outcomes

## 7. Implementation Strategy

Phases execute in order 1→2→3→4 (decision: round 1 d2=C).

### Shared Infrastructure: `monetizationTestHarness`

Before implementing test phases, create a shared test harness struct used across all new test files (decision: round 2 d2=A):

```go
type monetizationTestHarness struct {
    db                *sql.DB
    stripeService     *StripeService
    webhookService    *StripeWebhookService
    accountService    *AccountService
    downloadAuthorizer *DownloadAuthorizer
    stripeServer      *httptest.Server  // mock Stripe API
    webhookSecret     string
}
```

- Place in a shared test helper file or define in `stripe_monetization_e2e_test.go` and import from the other test files (all in `package main` so they share visibility).
- `setup()` method creates all services with shared DB (via `setupTestDB(t)`) and mock Stripe `httptest.Server`.
- `teardown()` cleans up DB state and closes the mock server.
- Include a `signWebhookPayload(payload []byte) string` method that signs with HMAC-SHA256 using the test webhook secret (decision: round 2 d3=A), reusing the existing pattern from `stripe_integration_test.go`.

### Phase 1: Checkout + Webhook Integration Tests
**New file: `stripe_monetization_e2e_test.go`**

Test cases:
- `TestE2E_Checkout_Webhook_Entitlement_Download_HappyPath` — Full flow: create checkout session → simulate checkout.session.completed webhook → verify subscription created → verify DownloadAuthorizer grants access
- `TestE2E_Checkout_StripeAPIError_ReturnsError` — Checkout with simulated Stripe API failure
- `TestE2E_Webhook_OutOfOrder_InvoicePaid_BeforeCheckoutCompleted` — Fire invoice.paid webhook first, then checkout.session.completed; verify final state is correct
- `TestE2E_Webhook_OutOfOrder_SubscriptionUpdated_BeforeCheckoutCompleted` — Fire customer.subscription.updated before checkout.session.completed
- `TestE2E_Webhook_CustomerUpdated_InterleavedWithPayment` — Email change webhook interleaved with payment events
- `TestE2E_Webhook_UserDeletedBetweenEvents` — Process webhook for a user that was deleted after checkout but before invoice

### Phase 2: Entitlement + Download Gating Integration
**New file: `download_entitlement_integration_test.go`**

#### Full 8-Status Matrix (decision: round 2 d4=B)

Test all Stripe subscription statuses against download authorization:

| Status | Expected Download Access | Test Case |
|--------|------------------------|-----------|
| `active` | Allowed | `TestIntegration_Status_Active_Download_Allowed` |
| `past_due` | Denied (degraded) | `TestIntegration_Status_PastDue_Download_Denied` |
| `canceled` | Denied | `TestIntegration_Status_Canceled_Download_Denied` |
| `unpaid` | Denied | `TestIntegration_Status_Unpaid_Download_Denied` |
| `incomplete` | Denied | `TestIntegration_Status_Incomplete_Download_Denied` |
| `incomplete_expired` | Denied | `TestIntegration_Status_IncompleteExpired_Download_Denied` |
| `trialing` | Allowed | `TestIntegration_Status_Trialing_Download_Allowed` |
| `paused` | Denied | `TestIntegration_Status_Paused_Download_Denied` |

#### Transition Tests

- `TestIntegration_Subscription_Active_To_Canceled_Download_Denied` — Active subscription → cancel → download authorization denied
- `TestIntegration_Subscription_Canceled_To_Resubscribed_Download_Restored` — Canceled → new checkout → active → download restored
- `TestIntegration_Subscription_Active_To_PastDue_To_Active` — Past_due degrades access, recovery restores it
- `TestIntegration_Subscription_Trial_Expired_No_Payment_Download_Denied` — Trial ends without payment → access denied

#### Error Propagation

- `TestIntegration_Entitlement_LookupError_Download_Denied` — Entitlement service returns error → download denied (fail-closed behavior, decision: round 1 d6=A)

### Phase 3: Coupon & Intro Pricing Assurance
**New file: `coupon_lifecycle_e2e_test.go`**

Test cases:
- `TestE2E_IntroCoupon_Apply_Use_RejectReuse` — First checkout applies intro coupon → webhook marks used → second checkout rejects coupon
- `TestE2E_IntroCoupon_EmailVariant_Bypass_Blocked` — Attempt to re-use coupon with email case variation (e.g., User@Example.com vs user@example.com); blocked by normalization
- `TestE2E_Coupon_Expired_RejectedAtCheckout` — Expired coupon applied to checkout → rejected

### Phase 4: Concurrency & Security Tests
**Added to existing: `stripe_concurrent_test.go`** (2 new tests only, decision: round 1 d3=A)

Test cases:
- `TestConcurrent_EmailMigration_DuringPaymentWebhook` — Concurrent email migration (customer.updated) and payment webhook (invoice.paid) for same customer; verify no data corruption
- `TestConcurrent_SubscriptionStatusChange_DuringDownloadAuth` — Subscription transitions to canceled while DownloadAuthorizer is mid-authorization; verify deterministic outcome

## 8. Contract Decisions

- Tests use the existing testcontainers pattern (`setupTestDB(t)` in `test_helpers_test.go`)
- Tests mock Stripe API calls at HTTP level via `httptest.NewServer` (no live Stripe calls)
- Tests verify database state directly for webhook idempotency and entitlement assertions
- DownloadAuthorizer tests use real DB with seeded subscription/entitlement data
- Webhook signatures use real HMAC-SHA256 signing with test secret via reused `signWebhookPayload` helper (decision: round 2 d3=A)
- New files: `stripe_monetization_e2e_test.go`, `download_entitlement_integration_test.go`, `coupon_lifecycle_e2e_test.go` (decision: round 2 d1=A)
- Shared `monetizationTestHarness` struct bundles all 4 services for cross-concern tests (decision: round 2 d2=A)

## 9. Testing Plan

All new tests are the deliverable. Test organization:
- 3 new dedicated test files (decision: round 2 d1=A):
  - `stripe_monetization_e2e_test.go` — checkout → webhook → entitlement → download flows + out-of-order webhooks
  - `download_entitlement_integration_test.go` — full 8-status matrix + lifecycle transitions + error propagation
  - `coupon_lifecycle_e2e_test.go` — intro coupon lifecycle + abuse cases
- 2 new tests added to existing `stripe_concurrent_test.go`
- Integration tests use `setupTestDB(t)` for real Postgres via testcontainers
- Stripe API interactions mocked at HTTP level (existing pattern in `stripe_integration_test.go`)
- Webhook events signed with HMAC-SHA256 using test webhook secret via `signWebhookPayload` helper
- Shared `monetizationTestHarness` provides pre-wired services for cross-concern tests
- Each test is self-contained: creates its own tables, seeds data, cleans up

## 10. Rollout / Validation Checklist

- [ ] All new tests pass: `cd scenarios/landing-page-business-suite/api && go test ./... -timeout 300s`
- [ ] Existing tests still pass (no regressions)
- [ ] `go vet ./...` clean
- [ ] `gofumpt` formatted

## 11. Risks + Mitigations

| Risk | Mitigation |
|------|-----------|
| Test complexity from mocking Stripe webhooks | Reuse existing httptest.NewServer + HMAC signing pattern from stripe_integration_test.go; shared `monetizationTestHarness` reduces boilerplate |
| Flaky concurrent tests | Use deterministic synchronization (channels, mutexes), not timing-based; only 2 concurrency tests to minimize surface |
| Over-mocking hides real integration bugs | Prefer testcontainers for DB; only mock external Stripe API |
| Test maintenance burden | Focus on behavioral contracts, not implementation details; shared harness centralizes wiring |
| Full 8-status matrix maintenance | Some statuses (paused, incomplete_expired) may not be reachable in practice; document expected behavior inline so future maintainers understand intent |
| Dependency on lpbs-desktop-release-contract-hardening | That item may change DownloadAuthorizer or download_assets schema; tests should use the seam interfaces, not internal details |

## 12. Non-goals / Prohibited Patterns

- Do not implement rate limiting, audit logging, or anomaly detection (separate items)
- Do not refactor existing Stripe code — this is a test-deepening effort
- Do not add compatibility shims or feature flags
- Do not test Stripe Dashboard behavior or admin UI flows
- Do not make live Stripe API calls in tests
- Do not add more than 2 concurrency test cases (round 1 d3=A)

## 13. Definition of Done

- [ ] E2E test: checkout → webhook → entitlement → download (happy path)
- [ ] E2E test: checkout → webhook → entitlement → download denied (subscription canceled)
- [ ] Full 8-status subscription matrix: all Stripe statuses produce correct download authorization outcome
- [ ] Subscription lifecycle transition tests: active→canceled→resubscribed, active→past_due→active, trialing→expired
- [ ] Intro coupon lifecycle E2E: apply → use → reject re-use
- [ ] 2-3 coupon abuse cases: email variant bypass, expired coupon rejection
- [ ] Out-of-order webhook delivery tests (2-3 critical reorderings)
- [ ] Entitlement error propagation test (fail-closed verification)
- [ ] 2 concurrent operation tests (email migration + payment, status change + download)
- [ ] Shared `monetizationTestHarness` provides pre-wired services
- [ ] All tests pass, no regressions, code formatted with gofumpt
