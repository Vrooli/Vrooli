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

## Identity and account ownership

The entitlement lease is not the project-wide identity contract. LPBS owns
business accounts, subscriptions, commercial entitlements, usage, and lease
issuance. It is currently also a compatibility issuer for the consumer
magic-link/JWT flow. The target architecture moves person identity and
authentication primitives to `scenario-authenticator`; LPBS remains the
business and commercial authority.

This distinction matters for bundled desktop scenarios:

- a private local bundle does not require an LPBS sign-in merely to use local
  capability;
- a user who wants paid features may explicitly link the local installation to
  a selected LPBS business account through a short-lived, scoped browser/device
  flow; multiple-account users must choose the account before issuance;
- email equality, copied browser tokens, and a local app's supervisor token
  are not account-linking mechanisms;
- a valid identity token does not grant a commercial entitlement, and an
  entitlement lease does not grant local API or filesystem authority.

### Account-scoped commerce

Account-bound checkout requests may include `business_account_id`. LPBS
requires an authenticated membership check, replaces any request-supplied
email with the selected account's billing identity, and carries the account
identifier through provider metadata and the durable checkout/subscription
projection. Desktop lease issuance reads only the selected account's
subscription and account-scoped credit wallet; it never falls back to a
same-email subscription or wallet from another business account. Legacy public
checkout without an account identifier remains email-compatible during the
migration period and is not an account-scoped purchase.

The project-level contract is [Identity and Authentication](../../../../docs/concepts/IDENTITY-AND-AUTHENTICATION.md).

## Deployment topology

The entitlement system does not choose the identity-provider topology. LPBS
must declare it with the deployment:

| Deployment | Identity source | Entitlement behavior |
|---|---|---|
| Public hosted LPBS | Hosted `scenario-authenticator` boundary or configured external provider | LPBS maps the verified principal to its business account and issues leases |
| Self-hosted LPBS mini-Vrooli | Installation-local `scenario-authenticator` dependency | LPBS maps the local principal to its business account; the realm is not global |
| Personal desktop bundle | No human identity provider; private installation boundary | No sign-in or entitlement is required for local capability; paid features trigger explicit linking |
| Multi-user desktop bundle | Local or remote authenticator selected by operator | Identity and coarse capability are checked locally/remotely; LPBS lease remains commercial authority |

For a self-hosted deployment, the first LPBS administrator may be linked to
the first local realm administrator during bootstrap. The mapping stores
provider, realm, principal, LPBS account, installation, timestamps, and
revocation state. It must not use email equality as proof and must not copy an
LPBS website session into a local app.

The lease subject identifies the verified principal and the LPBS business
account context needed for commercial decisions. A lease does not grant
filesystem, supervisor, scenario, or object-level authority. Conversely, a
valid authenticator token does not grant a plan feature or download right. A
desktop consumer must also bind the signed lease to its business account,
installation, resource, audience, link ID, and exact scope set before using
it. Offline use ends at the signed lease expiry; server-side unlink is
observed on the next status refresh, and local unlink must delete the stored
lease immediately.

The desktop client uses `auth.connectDesktop` to complete the explicit flow:
LPBS browser consent issues the one-use scoped code, the client redeems it with
a verified local-provider proof, and only the resulting signed lease crosses
the durable desktop credential boundary. Website access and refresh tokens are
not restored from desktop storage.

## Administration boundary

LPBS product administrators manage accounts, subscriptions, entitlements,
usage, and leases. Identity administrators manage credentials, MFA, sessions,
locks, and coarse capabilities in `scenario-authenticator`. When an LPBS
screen needs an identity operation, the LPBS API makes an authenticated
API-to-API call through `api-core/discovery`, preserves actor/target context,
and records both product and identity audit events. It never writes
authenticator tables directly. Unimplemented management methods remain
unavailable until the authenticator contract ships them.

## Maintenance Guidance

If you update the underlying scenarios:

- keep this document tightly scoped to the subsystem
- prefer durable models and current gaps over exhaustive code dumps
- link to the relevant scenario docs or code paths when possible
- remove stale status claims that are no longer verified
