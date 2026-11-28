---
name: "bundle-integration-steer"
description: "Integrates a scenario into the Vrooli Business Suite bundle so it enforces subscription gating, class-correct metering, and authenticated downloads consistently with every other bundled app, managed through the Landing Page Business Suite (LPBS). Implements the contract in docs/concepts/PAID_FEATURES.md: the client trust boundary, the signed entitlement lease, and the cost-bearing vs local-capacity meter split."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["bundle","lpbs","subscription","metering","entitlements","monetization","steer","architecture"]
  modes: ["steer","architecture"]
  status: "active"
  defaultScope: "work-scope"
  revision: 2
  createdAt: "2026-03-22T12:00:00Z"
  updatedAt: "2026-08-13T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "test-genie", "vrooli"]
    commands: ["prompt-manager skill", "prompt-manager skill read", "test-genie execute", "vrooli"]
  origin:
    kind: "authored"
---
## Steer focus: Bundle Integration

**Scope mode: `architecture`.** Hold to that row of `work-scope` for this session.

Integrate `scenarios/{{TARGET}}/` into the **Vrooli Business Suite bundle** so that it participates in subscription gating, credit-based metering, and authenticated downloads managed by the Landing Page Business Suite (LPBS) scenario.

Your goal is to ensure `{{TARGET}}` enforces entitlements and consumes credits consistently with every other bundled app, so that a single subscription unlocks and meters all apps in the bundle.

Required reading:
- `docs/concepts/PAID_FEATURES.md` — the engineering contract this skill implements. It owns the free/metered/gated decision, the Class A / Class B meter split, the trust boundary, and the current implementation status of each piece. Read it first; this skill is the wiring, not the contract.

Optional reading:
- `prompt-manager skill read api-steer`
- `prompt-manager skill read interoperability-steer`

---

### 0. Why This Skill Exists

The Business Suite bundle is sold under one subscription. Each bundled app must use one identity (LPBS-issued sessions through the shared device credential, not per-app accounts), enforce entitlements, meter correctly for its class (cost-bearing operations charge server-side; local-capacity operations run optimistically and sync), declare its paid surface in `.vrooli/monetization.json` so `monetization-conformance` can validate it, and register in LPBS so the landing page, downloads and entitlements API know it. Without one pattern each app invents its own auth, gating and credit logic: inconsistent enforcement, double charging, bypassed gates.

Earlier revisions told scenarios to validate LPBS tokens with a shared HS256 `JWT_SECRET` and documented a `POST /api/v1/credits/consume` endpoint. Both were wrong: LPBS signs asymmetrically and publishes a JWKS, and that endpoint never existed. A symmetric key that verifies a token can mint one, so shipping it in a desktop bundle is an account-takeover primitive. Treat either pattern in a scenario as a defect (§3).

### 1. Scope Boundaries

In scope: wiring `{{TARGET}}` to the shared LPBS consumer session; resolving and verifying the entitlement lease and gating on it; classifying every metered operation and wiring its Class A or B path; authoring `.vrooli/monetization.json` and passing `monetization-conformance`; registering `{{TARGET}}` as a downloadable app; error handling for auth failures, insufficient credits and lapsed subscriptions; auditing existing code for missing or unsafe integration. Out of scope: LPBS internals (webhooks, Stripe, subscription lifecycle), pricing and tier decisions, paywall UI design, app business logic away from the LPBS seam, landing page, A/B testing and admin portal.

### 2. Architecture

LPBS is the trusted server: identity (magic link, JWT, JWKS), subscriptions and the signed entitlement lease, the credit wallet (reserve and finalize) and the usage ledger (dedupes on `operation_id`). `{{TARGET}}` is an untrusted client in tiers 2 and 3: the shared session (`credentialclient-go`), a gate that verifies the cached lease locally, and Class B meters (optimistic plus outbox). Class A is charged on LPBS, never on the client. `{{TARGET}}` is a consumer of LPBS, never a replica: accounts, subscription state and balances live there. The lease lets the client decide fast and offline; LPBS decides authoritatively for anything that costs money.

---

### 3. The Trust Boundary: read this before writing any code

A scenario on a user's machine is untrusted: in tier-2 desktop and tier-3 mobile its API binary ships in the bundle, its code is readable and its database writable. Anything it enforces locally is advisory and anything it holds is a secret the user has. `monetization-conformance` fails the build on three rules:

| Rule | Finding code |
|---|---|
| Never ship a shared service secret to a client (a static write token is a forgery primitive) | `money.service_token_in_client_bundle` |
| Take identity from the verified token, never the request body | `money.identity_from_request_body` |
| Never verify an LPBS token with a shared symmetric secret | `money.symmetric_token_verification` |

Local entitlement checks still drive button states, prompts and fast decisions, as a user-experience affordance and not a security boundary; LPBS re-checks on every cost-bearing call.

---

### 4. Identity and the Entitlement Lease

**4.1 No account system.** Users sign in to LPBS by magic link; `{{TARGET}}` never stores passwords, sends auth email or keeps a user table. Wire only the shared device session: `credentialclient.ConsumerSessionResolver{Credentials, LPBSBaseURL}.Resolve(ctx)` (`packages/credentialclient-go`) returns a short-lived access token held in memory. The durable refresh token lives in the credential authority as `vrooli/lpbs-account` field `refresh-token`, which is why signing in to one Vrooli app signs in to all of them; do not invent a second credential.

**4.2 The lease.** `{{TARGET}}` does not verify LPBS access tokens itself. `GET <LPBS_BASE_URL>/api/v1/entitlements` with `Authorization: Bearer <user access token>` returns the entitlement payload (`status`, `plan_tier`, `plan_rank`, `price_id`, `features`, `limits[]` of `{limit_key, limit_value, reset_period}`, `billing_cycle_start`, `credits`, `subscription`, `not_after`) signed with the key published at `/.well-known/jwks.json`. Cache it and verify the signature locally on each gate, so a gate is a local check and a paid desktop app keeps working on a plane. Read `plan_rank`, `limits[]` and `not_after` from the verified lease and never duplicate them in scenario config.

**4.3 Gate rules.** Verify against JWKS, never a shared secret. Read limits from the lease, not local config such as `TIER_LIMITS_JSON` (`money.limits_from_local_config`). Degrade to the cached lease while valid and never hard-fail on a transient network error (`money.gate_blocks_offline`). After `not_after` with no refresh, fall back to the free tier and say why. Every gate has one chokepoint whose source paths are declared as the feature's `enforcement_paths` in `.vrooli/monetization.json`; a declared feature with no gate call there fails conformance.

**4.4 Status.** `active` and `trialing` allow; `past_due` allows with a warning (grace period, not cancellation); `canceled` denies with "subscription ended"; `inactive` and anything else deny with the upgrade prompt.

---

### 5. Metering

Read `docs/concepts/PAID_FEATURES.md` "Decision 2: which meter class?" first. The dividing line is who pays, not where code runs: an export uploading to Vrooli-owned storage is Class A though it runs locally.

- **Class A (cost-bearing, Vrooli pays per use).** Executed by LPBS, never bypassable, no offline path, enforced reserve, execute, finalize server-side. For LLM inference call **ai-gateway**, which already routes metered inference to LPBS; `{{TARGET}}` writes no metering code. Implement Class A directly only when the cost is neither inference nor audio, and then the operation must run on LPBS per `PAID_FEATURES.md`. A cost-bearing meter the client executes fails as `money.cost_bearing_meter_client_executed`. BYOK stays valid: the user pays their own way at zero credits.
- **Class B (local-capacity).** Check the lease limit locally, run the operation, append `{operation_id, limit_key, amount, occurred_at}` to a durable outbox, drain on a ticker, on reconnect and at startup. Never block on the network. The outbox is durable and its path declared in `.vrooli/monetization.json` (a goroutine with three retries loses paid usage silently; `money.no_outbox_for_local_meter`). Reuse one `operation_id` across the ledger row and every retry (LPBS dedupes). Record backlog over the limit as overage, never claw back after the fact. Show pending state honestly.
- **Sending usage.** `POST <LPBS_BASE_URL>/api/v1/usage/report` with the user's access token; LPBS derives identity from verified claims and ignores any identity in the body. No service token, and there is no `/api/v1/credits/consume`.
- **Displaying credits.** Credits are internal units: convert with `display_credits_multiplier` and `display_credits_label` from the entitlement payload; never show raw units.

---

### 6. LPBS-Side Registration

`{{TARGET}}` appears in downloads and entitlements only when registered in `download_apps` (per app: `bundle_key` `business_suite` matching `plans.json`, `app_key` `{{TARGET}}`, name, tagline, description, icon, install overview and steps, storefronts, display order) and `download_assets` (per platform and `variant_key`, normally `default`: `artifact_url`, `artifact_source` `direct` or `managed`, `release_version`, `release_notes`, `checksum`, `requires_entitlement`). Do not write registration SQL or hand-type release metadata: register from the artifact `scenario-to-desktop` produced, which supplies platform, version and SHA-256, with `bundle_key`, `app_key` and `requires_entitlement` read from the scenario declaration, through the LPBS catalog service in `scenarios/landing-page-business-suite/api/internal/delivery/catalog.go` (idempotent on `bundle_key`, `app_key`, `platform`, `variant_key`).

`requires_entitlement` is `true` on every asset of a paid or premium app (an active subscription is needed to download). A free, demo or open-source app sets it `false`; if it has credit-consuming premium features, gate those at runtime, and if it has none, ask whether it belongs in the bundle at all.

Verify the loop: `GET /api/v1/downloads?app={{TARGET}}&platform=linux` with a subscriber token returns the artifact URL, the same request with an expired or missing token returns 401 or 403, and `GET /api/v1/entitlements` works for an active subscriber.

---

### 7. LPBS Client Configuration

Connection details come from environment variables: `LPBS_BASE_URL`, `LPBS_JWKS_URL` (`<LPBS_BASE_URL>/.well-known/jwks.json`) and `JWT_ISSUER` (`landing-page-business-suite`). There is deliberately no shared signing secret (a `JWT_SECRET` is a defect, §3), and the refresh credential is not an environment variable (§4.1). Use a plain HTTP client with a 10-second timeout.

**Offline is a normal state for a tier-2 desktop app**, and "deny everywhere" breaks a paying customer on a plane. Gated features follow the lease: valid means allow with a stale-data notice, expired means free tier with the reason in the UI. Class B runs and queues, never blocking. Class A fails with "this needs a connection", not "access denied", because it is a network call by definition. Sign-in and token refresh fail gracefully and keep the existing session. Revenue integrity lives in Class A, where the wallet check happens on LPBS; refusing gated features during an outage protects nothing and costs goodwill.

### 8. Error Responses

Return LPBS-compatible shapes, `{"error": "<UI-safe message>", "error_type": "unauthorized|forbidden|validation|rate_limited|server_error", "retryable": false}`, and never leak raw LPBS errors: missing or invalid JWT is 401 `unauthorized` ("Please sign in to continue"); no subscription is 403 `forbidden` ("An active subscription is required"); insufficient credits is 402 `forbidden` ("Insufficient credits, top up to continue", with balance and top-up guidance); LPBS unreachable with no cache is 503 `server_error`; rate limited is 429 `rate_limited`.

---

### 9. Integration Checklist

Everything true before integration is complete:
- **Trust boundary:** no shared service token or symmetric secret in any tier-2 or tier-3 bundle (including no `deployment.tiers.*.secrets` entry with `classification: service` in `.vrooli/service.json`); every write to LPBS carries the user's access token; no client-executed cost-bearing meter.
- **Identity:** sign-in through `credentialclient-go` and `vrooli/lpbs-account`; JWKS verification; 401 with the standard shape; no own registration or login.
- **Entitlements:** gates resolve through the lease; the status table (§4.4) is honored; a valid cached lease survives outages; limits come from the lease; one chokepoint per feature in `enforcement_paths`.
- **Metering:** every meter declares a `class`; Class A runs on LPBS; Class B never blocks and uses a durable outbox with stable `operation_id`; BYOK bypasses charges; 402 handled; credits shown through the display multiplier.
- **Declaration:** `.vrooli/monetization.json` validates; `bundle_key` names a real bundle and `app_key` matches the `download_apps` row; every declared meter has `subscription_tier_limits` rows for every tier; `test-genie execute {{TARGET}} --phases monetization-conformance` passes.
- **Registration:** `download_apps` and `download_assets` rows exist per supported platform with correct `requires_entitlement`, version and checksum; a subscriber can download and a non-subscriber gets 403.
- **Errors:** standard shape throughout, no raw LPBS messages, 10-second timeouts.

---

### 10. Bundle Integration Audit (Brownfield)

Audit existing integration first with `rg` over `scenarios/{{TARGET}}`: JWT and bearer handling in the API; trust-boundary hits (`JWT_SECRET`, `SERVICE_SECRET`, `HS256`, `SigningMethodHMAC`, which are financial-security defects); `user_identity` read from payloads; `.deployment.tiers` tier-2 and tier-3 `secrets` in `service.json` (`jq`); existing LPBS, entitlement, credit, wallet, billing or stripe code; `type User struct`, `type Account struct`, `bcrypt`; and `LPBS_BASE_URL`, `JWT_SECRET`, `JWT_ISSUER` usage.

Red flags, worst first (the first four are financial-security defects: stop and fix them before continuing):
1. A shared service token or symmetric signing secret anywhere the client can reach.
2. A write to LPBS carrying `user_identity` in the body rather than deriving it from the token.
3. A cost-bearing meter charged by client code.
4. Symmetric HMAC token verification.
5. Own registration or login, or own Stripe integration.
6. Gated features with no entitlement check, or one not represented in `enforcement_paths`.
7. Tier limits read from scenario config; a local meter with in-memory retries and no durable outbox; a gate that hard-fails offline despite a valid lease.
8. No `.vrooli/monetization.json`, or one that does not match what the code gates.
9. Raw LPBS errors passed through; no LPBS registration; `requires_entitlement` false on assets that should be gated.

Record findings in `scenarios/{{TARGET}}/docs/internal/BUNDLE_INTEGRATION.md` (read it at session start; the code and LPBS database are the truth, so verify claims, correct inaccuracies and add findings): status per area (auth, entitlements, metering, registration, errors), a gated-features inventory (feature, gate type, credit cost, idempotent), issues with `file:line`, and priority actions. Track audited files with `visited-tracker` (location `scenarios/{{TARGET}}`, tag `bundle-integration`).

---

### 11. Output Expectations

You may add LPBS auth middleware, entitlement checks on gated endpoints, LPBS client configuration and environment variables, register `{{TARGET}}` in LPBS through the catalog service, and write `docs/internal/BUNDLE_INTEGRATION.md`.

You must use LPBS as the single source of truth for auth, subscriptions and credits; verify leases against LPBS's published JWKS and never ship a shared signing secret; serve a valid cached lease through a transient outage and expire to the free tier at `not_after`; use the standard error shape; put `scenario` in credit metadata; and set `requires_entitlement` correctly.

You must not implement registration, login or account management in `{{TARGET}}`, store subscription state or balances in its database, integrate Stripe directly, block Class B on an outage (append to the outbox and drain later), expose internal credit amounts, bypass entitlement checks, or make superficial changes.
