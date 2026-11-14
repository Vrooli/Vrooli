# Bundle integration

Browser Automation Studio (`app_key: browser-automation-studio`) is a paid
consumer of the `business_suite` bundle. The scenario declares that contract
in [`.vrooli/monetization.json`](../.vrooli/monetization.json). LPBS owns
subscription identity, signed entitlement leases, bundle limits, and the
Stripe/Apple/Google purchase rails; BAS owns only the operation-specific
adapters and presentation.

## Runtime wiring

- `packages/credentialclient-go` resolves the shared `vrooli/lpbs-account`
  credential. BAS does not create a second subscription identity or token
  store.
- `packages/entitlementclient-go` verifies the LPBS lease and exposes the
  server-signed feature and limit snapshot.
- `packages/monetization-go` supplies the shared Class B usage outbox and
  idempotent operation reporting. BAS persists it through its scenario-owned
  storage adapter.
- `api/middleware/entitlement.go` is the boundary for protected BAS routes.
  Class A operations require a valid server lease. Class B operations may use
  the signed local lease while offline and enqueue usage for reconciliation.
- The subscription settings surface adapts the shared
  `MonetizationAccount` component; it does not define a second plan catalog or
  entitlement ladder.

## Declared surfaces

The manifest currently declares these features:

| Key | Class | BAS enforcement |
|---|---|---|
| `ai` | A | entitlement middleware and AI service path |
| `recording` | B | entitlement middleware |
| `watermark_free` | B | entitlement middleware and entitlement service |

It declares these meters:

| Limit key | Class | Reporting |
|---|---|---|
| `ai_credits` | A | AI/credit service; server-authoritative |
| `workflow_executions` | B | shared monetization outbox |

Limit values and plan membership are never read from BAS configuration or an
environment variable. They come from the verified LPBS lease and the LPBS
runtime catalog.

## Desktop and session integration

The Electron shell uses the scenario-to-desktop template's loopback
authorization-code flow with PKCE. The refresh credential is stored through
the platform credential authority under `vrooli/lpbs-account`; access tokens
remain process memory. A custom-scheme callback is not an authentication
credential channel and token-bearing callbacks are rejected.

## Validation

The canonical checks are the `monetization-conformance` phase and the
manifest-driven `scenario-to-desktop` trust-boundary journey. Focused shared
kit, BAS entitlement/credits, provider-conformance, and desktop auth tests
must pass before a release. The latest recorded conformance run is
`20260817-152344-8d69a0bb` (pass). The latest broad BAS run,
`20260817-131928-c21b61c2`, still reports unrelated documentation and UI
coverage debt; that debt is not represented as monetization conformance
success.

When changing a declared feature or meter, update the manifest and its
enforcement paths together, then run the conformance phase and the BAS
targeted tests. Do not add a BAS-local tier table, credit-cost table, private
outbox, or shared-secret token verifier.
