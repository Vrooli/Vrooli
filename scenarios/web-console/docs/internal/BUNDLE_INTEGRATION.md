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
- `packages/monetization-go` provides the durable, idempotent Class B usage
  outbox. Web Console supplies its scenario-owned storage adapter and does not
  maintain a private outbox protocol.
- `api/monetization.go` is the single Web Console enforcement and reporting
  adapter. Its declared `voice_synthesis` feature is Class B and its
  `voice_minutes` meter is reported through the shared outbox.

## Declared contract

| Surface | Class | Manifest key | Enforcement/reporting |
|---|---|---|---|
| Voice synthesis | B | `voice_synthesis` | `api/monetization.go` |
| Voice usage | B | `voice_minutes` | `api/monetization.go` + shared outbox |

The `business_suite` bundle, `web-console` app key, feature requirement, and
limit key are declaration data. The limit value and plan entitlement are
resolved from the verified LPBS lease and runtime catalog, never from Web
Console configuration or environment variables.

Class B behavior is intentionally useful while LPBS is temporarily
unreachable: the locally verified lease controls the operation and usage is
queued for reconciliation. Reconciliation is idempotent by `operation_id`.
Class A behavior, where introduced, must require a live server decision and
must fail closed when that authority cannot be reached.

## Integration rules

1. Keep one `MonetizationAccount`/shared credential identity per machine.
2. Read features and limits from the signed lease; do not add local plan or
   credit tables.
3. Report Class B usage through `packages/monetization-go` and preserve the
   operation id across retries.
4. Keep token verification asymmetric and LPBS-owned. Do not add a shared
   secret or place access/refresh tokens in URLs.
5. Update `.vrooli/monetization.json` enforcement paths whenever the adapter
   moves, then rerun provider and monetization conformance.

## Validation

The canonical static check is the `monetization-conformance` phase. The latest
recorded Web Console conformance run is `20260817-131830-4f0ef77a` (pass).
The latest broad Web Console suite remains subject to unrelated pre-existing
quality, unit, workflow, security, and documentation debt; those findings
must remain visible in the full-suite result and must not be relabeled as
monetization success.
