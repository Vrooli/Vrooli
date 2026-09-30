# Monetization Conformance Phase

The phase validates the declaration and trust boundary for scenarios that gate
features or report usage to LPBS.

## North Star

A monetized scenario declares its paid surface, carries no shared service
secret, derives identity from a verified consumer token, verifies leases with
asymmetric public keys, and sends Class A work through the trusted authority.

## The rungs and their gates

L0 means the inventory cannot be inspected. L1 means the source is readable and
trust-boundary findings are visible. L2 means the declaration and enforcement
paths are present. L3 means meter classes have the required server or durable
outbox posture. L4 is a clean monetization contract.

## What each finding means

- `money.undeclared_monetization`: applicable monetization code has no sidecar declaration.
- `money.service_token_in_client_bundle`: a shared service credential remains in client source or deployment configuration.
- `money.symmetric_token_verification`: client code verifies a token with a symmetric signing secret.
- `money.feature_not_enforced`: a declared feature or meter has no real enforcement path.
- `money.meter_missing_tier_limits`: a meter has no lease-backed limit declaration.
- `money.identity_from_request_body`: usage identity is accepted from request data.
- `money.cost_bearing_meter_client_executed`: Class A work is executable in an untrusted client.
- `money.limits_from_local_config`: a lease limit is duplicated in local configuration.
- `money.gate_blocks_offline`: a transient outage denies work without consulting a valid cached lease.
- `money.no_outbox_for_local_meter`: Class B usage lacks a durable outbox.
- `money.no_account_surface`: an entitled scenario does not declare a reachable account, plan, subscription, and credit surface.
- `money.no_journey_probe`: an entitled scenario does not declare a provider-neutral monetization journey probe.

These findings are gating errors because each can undermine account isolation.

The provider contract is version `1.2.0`. An entitled scenario must declare
`account_surface_paths` and `journey_probe_paths`; scenarios outside the paid
release surface may remain not applicable.

## The canonical fix

Add `.vrooli/monetization.json`, use the shared consumer session and LPBS JWKS,
derive identity from verified claims, and route Class A operations through LPBS.
Persist Class B reports in a durable outbox before attempting delivery.

## How to verify

```bash
test-genie phases inspect monetization-conformance --json
test-genie execute browser-automation-studio --phases monetization-conformance
```
