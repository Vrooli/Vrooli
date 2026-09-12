# Security — Persona

This document is the canonical security and privacy posture for the
scenario: what it holds, who may reach it, what it refuses, and what is
still open.

## Purpose Of This Document

Use this document to answer:

- What sensitive data does the scenario handle?
- How are callers authenticated and authorized?
- Where do secrets live?
- What is the threat model?
- What security work is still outstanding?

## The Governing Idea

**Hold bindings and policy, never the sensitive payload.** Persona is
the scenario an operator is asked to trust with their legal identity, so
its security story cannot rest on access control alone — it rests on
*not having the material in the first place*. Documents live in
`document-manager`, credentials in `secrets-manager`, run identity in
`agent-manager`, and one-time code values nowhere at all. What remains
here is a pointer, a rule, and a record.

That is a deliberate trade: it makes the scenario dependent on three
others, and it makes a compromise of this scenario materially less
valuable than a compromise of a monolithic alternative.

## Data Sensitivity

| Data | Class | Held Here? | Notes |
|---|---|---|---|
| Legal basis (name of a person or entity) | **Personal — high** | Yes | Immutable after creation. The single most identifying field in the scenario. |
| Postal and billing addresses | **Personal — high** | Yes | Released under the same entitlement rule as documents. |
| Identity document content (passport, licence, incorporation papers) | **Personal — critical** | **No** | `document-manager` holds bytes, sensitivity class, and custody journal. |
| Mailbox and provider credentials | **Secret — critical** | **No** | `secrets-manager`. A binding records *which* credential, never the value. |
| One-time code values | **Secret — transient** | **No** | Returned to the caller once and never persisted. Only the fact of the fetch is recorded. |
| Channel bindings (which address, which adapter) | Personal — moderate | Yes | The address itself is identifying; the credential is not here. |
| Handoff checkpoints | Personal — variable | Yes | May embed form data mid-enrolment, which is why the retention window is deliberately short. |
| Account links and recovery paths | **Personal — high** | Yes | Collectively a map of everything a persona has created; valuable to an attacker in aggregate. |
| Action journal | Personal — moderate | Yes | Identifiers and verbs only. Never payloads, code values, or document content. |

**The aggregate risk is higher than any single row.** Account links plus
a controlled mailbox is, in practice, a password-reset path to
everything a persona created. That combination — not any individual
field — is what this scenario's controls are sized against.

## Auth And Authorization

Three distinct mechanisms, deliberately not collapsed into one:

1. **Run identity** — `agent-manager` verifies the caller's signed run
   token through a live call. This scenario never parses, re-signs, or
   locally validates that token, and never holds the signing secret.
2. **Act-as authorization** — the `access` domain decides whether the
   verified human behind that run may act as the requested persona,
   using the persona ACL and, when available, grants from
   `prompt-manager`.
3. **Entitlement** — what a successful resolution actually returns is
   decided in one place, so no call site re-derives it.

### Binding rules

- **Fail closed, everywhere.** An unreachable or unverifiable authority
  refuses act-as. There is no degraded evidence grade, no override flag,
  and no environment variable that relaxes it. The availability cost is
  accepted deliberately.
- **Every degradation is narrower, never wider.** No failure path grants
  access that the healthy path would refuse. `prompt-manager` being down
  removes grant-derived access; it never substitutes a permissive
  default.
- **Attenuation is inherited, not reimplemented.** Persona scopes ride
  `IntersectScopes` and `Attenuate`, so a child run can only ever hold a
  subset of its parent's access.
- **Refusals are journaled as loudly as successes.** A refused act-as is
  exactly the row an investigation needs.

## Secrets

No credential is stored by this scenario. `secrets-manager` is the sole
custodian for mailbox and provider credentials; a channel binding holds
a reference, and the credential is fetched at use time and never cached
to disk.

The scenario's own database is not a secret store and must
never become one. A pull request that adds a credential-shaped column is
a design error, not a convenience.

## Threat Model

| Threat | Vector | Mitigation | Residual |
|---|---|---|---|
| **Prompt injection into a release** | A merchant page persuades an agent to request a document release it does not need | Release targets a **pre-declared** handoff and is evaluated server-side; there is no agent-readable read path for document content under any scope | An agent could still open a handoff for a legitimate-looking but unnecessary step; the operator sees the request before acting, which is why handoff copy must state *why* a document is needed |
| **Confused deputy across personas** | A flow acquires a code or address belonging to a different persona | Adapters never fall back to another persona's route; a resolution names exactly one persona and no default exists | Operator error in selecting a persona remains possible; mitigated by making kind and basis prominent |
| **Privilege escalation by a child run** | A sub-agent requests broader persona access than its parent | `Attenuate()` rejects widening at the token layer before this scenario is reached | Depends on agent-manager's correctness, which is the right place for it to depend |
| **Mailbox compromise** | The controlled address is the recovery path for every linked account | Credential held in `secrets-manager`, never surfaced to an agent; staleness checks detect auth failure | Compromise of `secrets-manager` itself is out of scope here and is that scenario's threat model |
| **Journal tampering to hide an action** | An attacker edits or deletes attribution rows | Append-only by construction; update and delete are unreachable from any handler; correction is a compensating row | Direct database access by a host-level attacker; mitigated only by export-and-archive |
| **Verification-authority spoofing** | An attacker forges a run token | The signing secret never leaves `agent-manager`; verification is a live call, not local validation | A compromise of `agent-manager` compromises this scenario; accepted, and the reason act-as is the only privileged verb |
| **Silent identity misuse** | An agent acts as a persona nobody authorised | ACL evaluated server-side; every act-as journaled with the authorising human | An authorised human acting maliciously is out of scope — the journal makes it attributable, not impossible |
| **Data exfiltration through a handoff** | A handoff is crafted to deliver sensitive material to the wrong person | Handoffs deliver to the operator's configured channels only; relay targets are operator-configured, never caller-supplied | Operator misconfiguration of the relay |

### Explicit non-defenses

The scenario does **not** defend against, and must never claim to:

- A malicious operator acting as their own persona. Attribution, not
  prevention, is the goal.
- Host-level compromise. If an attacker has the disk, the bindings and
  the journal are readable.
- Compromise of `agent-manager`, `secrets-manager`, or
  `document-manager`, each of which owns its own threat model.

### An anti-goal worth stating

This scenario **never solves, bypasses, or assists in bypassing a human
verification control.** No CAPTCHA solving, no biometric spoofing, no
synthetic identity, and no persona representing a person who has not
authorised it. The wall is a product feature. Circumventing it would be
both a terms-of-service violation for the counterparty and the fastest
route to every persona being banned. A handoff routes a human to satisfy
a control legitimately; it never coaches them around it.

## Security Gaps

| Gap | Risk | Planned Resolution |
|---|---|---|
| No rate limiting on code retrieval | A loop could hammer a mailbox and trip provider abuse controls | Bound retrieval attempts per persona per window; sized during `channels` implementation |
| Journal has no external anchoring | A host-level attacker could truncate the tail | Export-and-archive at `PSN-P0-011`; periodic off-host anchoring deferred until a real audit demand exists |
| Handoff checkpoints are unencrypted at rest | Mid-enrolment form data sits in SQLite | Short retention (1 year post-terminal) at P0; field-level encryption considered if checkpoints prove to carry regulated data |
| Attestation signing key management undefined | `PSN-P1-007` emits signed tokens with no rotation story | Resolve during P1; reuse `scenario-authenticator` rather than minting a private scheme |
| No ACL change-approval flow | A single compromised operator session can widen an ACL | Journaled today, gated never; revisit if multi-operator installs become real |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — what is stored and for how long
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency failure behavior
- [`../concepts/FLOWS.md`](../concepts/FLOWS.md) — the act-as and release flows
- [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md) — incident response
- [`DECISIONS.md`](DECISIONS.md) — durable choices behind this posture
