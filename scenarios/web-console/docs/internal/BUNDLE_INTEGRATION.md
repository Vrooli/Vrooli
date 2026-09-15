# Bundle integration

Web Console (`app_key: web-console`) is a paid consumer of the `business_suite`
bundle. Its declaration is [`.vrooli/monetization.json`](../../.vrooli/monetization.json).
LPBS owns subscription identity, signed leases, bundle limits, and purchase
rails. Web Console owns the adapter around its voice operation and its user
experience, not the commercial policy.

## Runtime wiring

- `packages/credentialclient-go` provides the shared `vrooli/lpbs-account`
  subscription identity and credential authority integration.
- `packages/entitlementclient-go` verifies the LPBS lease and supplies the
  server-signed feature and limit snapshot.
- `packages/monetization-go` provides shared session, journey, and durable
  usage primitives. Web Console does not maintain a private commercial policy.
- `api/internal/ai/provider.go` uses Ollama first, then the user's credential-
  authority OpenRouter key, then `ai-gateway` for subscription-backed
  inference. Only the last path carries the consumer token to the trusted
  metering rail.
- `api/credentials.go` exposes declaration-checked, metadata-only provisioning
  for the subscription refresh token and OpenRouter BYOK key.

## Declared contract

| Surface | Class | Manifest key | Enforcement/reporting |
|---|---|---|---|
| Routed subscription inference | A | `ai_credits` | `ai-gateway` via `api/internal/ai/provider.go` |
| Hosted voice usage | A | `ai_credits` | `audio-tools` |

The `business_suite` bundle, `web-console` app key, feature requirement, and
limit key are declaration data. The limit value and plan entitlement are
resolved from the verified LPBS lease and runtime catalog, never from Web
Console configuration or environment variables.

BYOK and local Ollama behavior is free because the user supplies the resource.
Subscription-backed inference is trusted only in `ai-gateway`, where the
consumer token is forwarded and LPBS owns reserve/execute/finalize decisions.
Web Console may surface a degraded state, but it never grants wallet headroom.

## Integration rules

1. Keep one `MonetizationAccount`/shared credential identity per machine.
2. Read features and limits from the signed lease; do not add local plan or
   credit tables.
3. Keep `ai_credits` ownership in the service that performs the billable work;
   do not declare a second Web Console meter for the same operation.
4. Keep token verification asymmetric and LPBS-owned. Do not add a shared
   secret or place access/refresh tokens in URLs.
5. Update `.vrooli/monetization.json` enforcement paths whenever the adapter
   moves, then rerun provider and monetization conformance.

## Validation

The canonical static check is the `monetization-conformance` phase. The latest
targeted Web Console conformance run during this change is
`20260902-013751-2f76e759` (pass).
The latest broad Web Console suite remains subject to unrelated pre-existing
quality, unit, workflow, security, and documentation debt; those findings
must remain visible in the full-suite result and must not be relabeled as
monetization success.
