# Subscription Entitlements System

This document describes a scenario-specific subsystem spanning `landing-page-business-suite` and `browser-automation-studio`. It is not project-level architecture for Vrooli as a whole.

## Status

- scope: subsystem architecture
- authority: limited to the scenarios named above
- role: analysis and design reference

If the implementation has changed since the last detailed audit, the code for those scenarios is the source of truth.

## What This Document Is For

Use it when you need to understand:

- how subscription state, entitlements, usage, or limits are split across those scenarios
- known design gaps or model mismatches in that subsystem
- where future consolidation or cleanup work may be needed

## What This Document Is Not For

Do not use this file as:

- the canonical architecture guide for Vrooli
- a billing or commercial roadmap
- a claim that every implementation detail here is still current without code verification

For project-level architecture, start with:

- [../../../../docs/concepts/ARCHITECTURE.md](../../../../docs/concepts/ARCHITECTURE.md)
- [../../../../docs/strategy/context.md](../../../../docs/strategy/context.md)
- [../../../../docs/strategy/decisions.md](../../../../docs/strategy/decisions.md)

## Signed entitlement lease

`GET /api/v1/entitlements` is user-authenticated and returns the account
payload plus a compact RS256 `lease`. The lease payload contains the verified
`user_identity`, subscription `status`, `plan_tier`, `plan_rank`, `features`,
authoritative `limits`, and UTC `not_after`. Its header carries `kid` and
`typ: ENTITLEMENT_LEASE`; the public key is available from
`/.well-known/jwks.json`.

Bundled consumers verify the lease locally and may use a cached lease until
`not_after`. A refresh failure never extends that deadline. LPBS remains the
authority for Class A reservations and metered work, while the lease supplies
the offline UX decision and Class B local-capacity limits.

## Maintenance Guidance

If you update the underlying scenarios:

- keep this document tightly scoped to the subsystem
- prefer durable models and current gaps over exhaustive code dumps
- link to the relevant scenario docs or code paths when possible
- remove stale status claims that are no longer verified
