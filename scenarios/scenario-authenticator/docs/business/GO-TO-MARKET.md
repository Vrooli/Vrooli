# Go To Market — Scenario Authenticator

This document records the adoption path, positioning, audiences, and
sequencing for the fleet's Identity Provider (IdP). Everything below is a
pre-implementation **hypothesis** derived from [`../../PRD.md`](../../PRD.md),
not a committed plan — the scenario is pre-launch and nothing is built.

## Purpose Of This Document

Use this document to answer:

- Who adopts this scenario, and in what order?
- What is the positioning for each audience?
- How does adoption sequencing track the PRD launch plan?
- What evidence advances the plan?

## Positioning

scenario-authenticator is **the highest-leverage compound-value seam in
the fleet**: every future product that needs users adopts it as a scenario
dependency instead of rolling its own auth. The "sale" is mostly internal
and structural — adoption, not a price tag. The pitch is the same to every
audience:

> One permanent, secure identity capability the whole fleet builds on.
> Resolve it by slug, verify its tokens locally against JWKS (no
> per-request callback), and stop reinventing accounts, sessions, MFA,
> token signing, and federation. The **realm** primitive lets the same
> code serve a single household server and a multi-tenant SaaS.

Against the status quo (each scenario hand-rolling auth, or sharing one
Postgres-backed auth DB): this removes the shared-DB blast radius, gives a
stateless local-verify contract so consumers scale without calling back,
and is free/BYOK for self-hosters.

## Audiences (in adoption order)

| Audience | What they get | Why they adopt | When |
|---|---|---|---|
| **Internal Vrooli scenarios (first)** | A typed Connect client + JWKS local-verify contract; resolve by slug via `api-core/discovery`. | Stop reimplementing auth; inherit RS256/JWKS/rotation/sessions/MFA for free. device-sync-hub is the first live consumer. | P0 |
| **Self-hosters / operators** | The complete capability, free, with their own signing keys and SQLite storage; single default realm "feels single-tenant." | Run private identity for their own stack with no metering and no lock-in. | P1 |
| **Hosted SaaS products built on it** | A realm per customer (B2B) or product (B2C); managed-DB backing + multi-instance HA via the storage seam. | Ship a product with users without building an IdP; monetize *their* tiers on the identity issued here. | P2 |

## Adoption Sequencing (mirrors the PRD launch plan)

1. **P0 — unblock the first consumer.** Land the auth core (accounts +
   Argon2id + RS256/JWKS/keypair persistence + refresh rotation with reuse
   detection + Redis-backed sessions + rate-limit/lockout + audit log +
   single default realm with `aud`-scoped tokens + Connect surface + CLI
   parity + SQLite via the storage seam). **Then migrate device-sync-hub's
   forwarder from REST to the typed Connect client and prove the live
   first-run flow end-to-end (OT-P0-012).** No P1 work begins until that is
   green. This is the proof that the local-verify contract holds against a
   real consumer.
2. **P1 — enable self-host + the first SaaS.** Realms as a true tenant
   boundary, TOTP MFA, OAuth social federation, API keys/client-credentials,
   per-realm scopes, passkeys, admin console + self-service UI, account
   recovery. This is the surface a self-hoster runs and the first SaaS
   product builds on.
3. **P2 — enterprise.** SAML/enterprise SSO, OIDC-provider mode ("Login
   with Vrooli"), groups/orgs hierarchy, delegated policy engine, per-realm
   key isolation + automated rotation, managed-DB/HA backing, SCIM
   provisioning.

The compounding effect: each adopter that reuses this instead of
hand-rolling auth makes the next product cheaper to build, which is the
recursive-value argument for prioritizing it as foundational infra.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| Internal scenario reuse | New scenarios adopt scenario-authenticator as a dependency instead of rolling their own auth | Stable Connect client + JWKS local-verify contract, the RP integration guide ([`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md)) | # of consuming RPs and their auth volume |
| Reference integration (device-sync-hub) | A live, proven first consumer is the strongest adoption asset | OT-P0-012 migration green, documented forwarder pattern | First RP verifying tokens unchanged on the Connect client |
| Self-host distribution | Operators run private identity on the local Vrooli stack | Install/runbook docs, key backup/rotation guidance | Self-host installs that stand up the default realm |
| Hosted SaaS enablement | Products provision a realm per customer/product | Managed-DB seam (OT-P2-006), per-realm provisioning, branding | First SaaS product running a per-customer realm |

## Validation Experiments

| Experiment | Audience | Threshold | Decision |
|---|---|---|---|
| Live consumer migration | Internal scenarios | device-sync-hub verifies tokens unchanged on the Connect client (OT-P0-012) | If met, the local-verify contract is proven; open P1. If not, fix the contract before adding consumers. |
| Second internal adopter | Internal scenarios | A second RP adopts via slug + JWKS local-verify with no auth code of its own | Validates reusability; informs which P1 surfaces to prioritize. |
| Self-host stand-up | Operators | An operator stands up the default realm + first admin from the runbook unaided | Validates self-host ergonomics + docs. |
| First SaaS realm | Hosted SaaS | One product runs a per-customer realm and gates its tiers on issued identity | Validates the indirect monetization model (LPBS entitlements). |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — indirect model + pricing routed to canon
- [`../../PRD.md`](../../PRD.md) — launch sequencing, Appendix A (IdP/RP), D (ecosystem-fit)
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — the RP integration contract
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — adoption telemetry
