# Security — Token Economy

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

## Data Sensitivity

**This scenario stores behavioral data about children.** That is the sentence
that should govern every decision below, and it is stated first deliberately.

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Holder records | **medium — pertains to a minor** | holders | Display name and a `scenario-authenticator` subject. Deliberately no date of birth, address, contact details or payment instrument — the scenario has no use for any of them. |
| Journal events | **medium — behavioral record of a minor** | journal | What a child did to earn and what they chose to redeem, over time. Permanent and uncompacted, which makes minimization at write time the only control that matters. |
| Actor provenance | low | journal | Acting identity plus verification status. Internal attribution, not personal data beyond the holder link. |
| Grants and rules | low | grants | Amounts, scopes and expiry. Reveals what a household rewards. |
| Catalog entries | low | catalog | Household-authored reward names. Can incidentally reveal family circumstances (a trip, a medical reward); treat as private, not public. |
| Redemptions and reservations | **medium** | redemption | What a child wanted and whether an adult refused it, with the recorded reason. |
| Token types | low | mints | Economy identity. Carries no monetary field by contract. |
| Earning submissions | low | earning | Adapter dedup keys and summarized payloads. Payloads are summarized, never stored whole. |

**What makes the posture defensible:** nothing leaves the machine — no
processor, no analytics vendor, no cloud sync — and no regulated identifier is
collected. There is also no monetary value (`TKE-P0-014`), so no financial
record about a minor exists to protect.

<!-- EXAMPLE-DOMAIN:notes START -->
The shipped worked-example `notes` domain carries placeholder data only
(removed by `template-manager detemplate`):

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Template notes data | low | notes reference | Local development data only; replace with real scenario data classification. |
| Attachment bytes | unknown | notes reference | Treat as potentially sensitive if retained in product scope. |
<!-- EXAMPLE-DOMAIN:notes END -->

## Auth And Authorization

Identity is owned by **`scenario-authenticator`** and is a hard, fail-closed
dependency. Tokens are verified locally against the published JWKS; this
scenario never stores a credential and never performs a per-request
authorization callback.

Three controls carry the security model, and all three are structural rather
than procedural:

1. **Two Connect services (`TKE-P0-005`).** A minter service owns mint, grant,
   catalog and rule mutation. A holder service owns view, redeem and request.
   A holder presenting a perfectly valid token cannot mint, because the RPC is
   not on the service they can reach. This is a codegen-visible boundary, not a
   runtime role check that a bug could bypass.
2. **Repository-layer isolation (`TKE-P0-006`).** Cross-holder scoping is
   applied in the repository, not only the handler, so a future handler written
   without the check still cannot leak. Refusals do not disclose whether the
   other holder exists.
3. **Server-side rule evaluation (`TKE-P0-003`).** A client-supplied assertion
   that a rule was already satisfied is ignored. The surface that read untrusted
   input never holds the decision — the same posture `treasury` takes against
   prompt injection.

UI and CLI are translation layers and enforce nothing. Any authorization they
appear to apply is presentation only.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| None owned by this scenario | n/a | no | Deliberate. No third-party service, no payment processor, no chain key, no push provider. Identity material belongs to `scenario-authenticator`; this scenario holds only verified claims. |
| Adapter credentials (future) | `secrets-manager` | conditional | If earning adapters authenticate per-adapter, credentials belong in `secrets-manager`, never in scenario storage. Decide before `TKE-P1-009`. |
| Chain key material (P2 only) | undecided | no | Would be introduced only by `TKE-P2-001`, and only after a recorded custody decision. Holding a key is a commitment the scenario deliberately has not made. |

## Threat Model

Ordered by what would actually hurt.

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| **One holder reads or spends another's balance** | Ends trust in a household product permanently. A sibling seeing another's rewards is a real-world argument, not an abstraction. | Repository-layer scoping plus handler checks plus a two-session BAS case (`TKE-P0-006`). Refusal does not disclose existence. | planned — highest priority |
| **A holder escalates to minter authority** | A child grants themselves tokens; the economy is meaningless. | Separate Connect services; the mutation RPCs do not exist on the holder service (`TKE-P0-005`). | planned |
| **Double-spend under retry or partial failure** | Balance and journal diverge in an append-only store that has *no repair verb by design*. | Debit and event in one transaction under a row lock, keyed by a caller-supplied idempotency key; tested against **induced** failure, not assumed (`TKE-P0-009`). | planned |
| **An earning adapter inflates a balance by replay or flood** | Economy debased; the reward loses meaning. | Dedup keys make replay a no-op (`TKE-P0-007`). Rate limiting is deliberately deferred until an adapter exists to abuse it; recorded in `PROBLEMS.md` rather than pre-built. | partial |
| **A compromised or buggy agent issues grants** | Tokens appear with no accountable cause. | Every event carries actor provenance and verification status (`TKE-P0-011`); an unverified caller is recorded as unverified, never promoted. Grants are visible in the minter's journal. | planned |
| **Rule programs used as a code-execution surface** | Remote execution on the household machine. | Conditions come from a closed vocabulary; caller-supplied scripts are never accepted (`TKE-P1-002`). | planned |
| **History rewritten to hide a mistake** | The audit property the product is built on is void. | The repository exposes no update or delete for journal rows, asserted structurally; corrections are compensating events (`TKE-P0-010`). | planned |
| **`scenario-authenticator` unavailable and the scenario degrades open** | Isolation becomes cosmetic exactly when nobody is watching. | Fail closed. Authenticated surfaces refuse; there is no unauthenticated fallback view. | planned |
| **Behavioral data about a minor leaving the machine** | The privacy claim that is half the product's differentiation is false. | No third-party service exists in the dependency inventory; introducing one requires a recorded decision. | structural |
| Unsafe file upload handling | Malicious or oversized upload could affect storage. | Not applicable to this scenario — no binary upload path exists. Inherited from the removable `notes` example. | template-reference |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| **Nothing is implemented.** Every mitigation above is `planned`; none is proven. | high | Gate 6 — the first real vertical slice. Until then this table states intent, not posture. |
| No erasure path for a departing holder; removal tombstones so events stay attributable. | medium while self-hosted, **blocking for any hosted deployment** | Any deployment where the operator is not the sole data holder. Must be designed before launch, never retrofitted. |
| Earning-adapter rate limiting deferred. | low | The first adapter that is not operator entry (`TKE-P1-009`). |
| Earning-submission dedup window undefined — too short double-grants, too long grows unbounded. | medium | Must be fixed before `TKE-P0-007` ships. |
| Adapter credential custody undecided. | low | Before `TKE-P1-009`, if adapters authenticate individually. |
| Journal export is holder-scoped by design but the delivery mechanism is unspecified; a shared link would defeat the isolation boundary. | medium | Before `TKE-P1-010`. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
