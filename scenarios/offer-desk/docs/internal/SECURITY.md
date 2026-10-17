# Security — Offer Desk

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

## The one-paragraph summary

This scenario holds no money, no customer records, and no credentials. Its data is
commercially sensitive rather than personally sensitive: an unreleased offer graph
discloses strategy. The security-relevant surface is almost entirely **authorization**,
because the scenario's central guarantee is that *an agent may propose a promotion and only
an operator may grant it* (`OT-P0-005`). That makes the promotion path a privilege-escalation
target, and it is the one thing here worth attacking. The second property is integrity of
the append-only audit trail, which is what makes any of it checkable after the fact.

## Data Sensitivity

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Offer graph — nodes, edges, statuses | **medium-high** | `catalog` | Commercially sensitive. Unreleased offers, parked candidates, and channel bets disclose strategy and roadmap ahead of announcement. Not personally identifying. |
| Status history and audit entries | **high** | `catalog` | Records actor, timestamp, prior value, reason. Integrity-critical: it is the only evidence of who promoted what. Not regenerable. |
| Trigger declarations | medium | `gates` | Reveal the conditions under which the operator intends to act — arguably more strategically revealing than the offers themselves. |
| Facts in the registry | medium | `gates` | Operator-supplied values that triggers evaluate against. May include commercial figures such as subscriber counts. |
| Evaluation runs | low | `gates` | Operational history; useful for debugging, low disclosure value. |
| Promotion proposals | medium | `gates` | Show intent before decision. A leaked proposal reads as a commitment that was never made. |
| Board entries | low | `board` | Derived at read time, owns nothing. Sensitivity is inherited from its sources. |
| Money Ledger actuals (read-through) | **high** | not owned | Revenue figures pass through this scenario and are never stored. See Threat Model. |

## Auth And Authorization

This is the substantive section for this scenario.

**The operator-role boundary is a security control, not a workflow nicety.** The source
canon said "agents never self-promote" as an instruction; `OT-P0-005` makes it a permission.
That conversion is the point, and it imposes real obligations:

- **Enforced at the API/service layer only.** The CLI and UI are equally reachable typed
  clients. A promotion refused in the UI but permitted over Connect-RPC is not a control.
- **The actor is recorded, not asserted.** Every transition writes the acting identity from
  the authenticated context, never from a request field. A caller-supplied `actor` is a
  forgery primitive.
- **Agent identity must be distinguishable from operator identity.** If both arrive as an
  unauthenticated local call, the permission is decorative. Until identity is real, treat
  the boundary as unenforced and say so — see Security Gaps.
- **Proposal creation is deliberately unprivileged.** Any agent may propose; the value is
  in the operator's grant. Do not "harden" this by restricting proposals — that breaks the
  intake path without improving the boundary.

No auth provider ships in the scaffold. The single-operator local deployment is the P0
target and needs none.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| None | n/a | no | This scenario authenticates to nothing. |
| Money Ledger client credential, if the ledger ever requires one | Platform secret store | deferred | Today the actuals join is a local typed call. If it crosses a trust boundary, it acquires a credential and this row becomes real. |

The absence of secrets is a deliberate property worth preserving. If a change to this
scenario introduces a credential, that is a boundary change and belongs in `DECISIONS.md`.

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| **Agent self-promotion** | An offer reaches `active` without an operator decision — the exact failure the scenario exists to prevent, and the highest-value attack here. | `OT-P0-005`: the transition to `active` requires an operator role, enforced in the service. Agents may only create proposals. | by-design, **contingent on real identity** |
| Actor spoofing in the audit trail | The trail records a promotion as operator-approved when it was not, making the record actively misleading. | Actor is taken from authenticated context, never from the request body. | required |
| Audit-trail tampering | Destroys the only evidence of who decided what. | Append-only; corrections are new entries, never edits. | by-design |
| Trigger predicate abuse | A crafted trigger causes excessive evaluation cost or reads state it should not. | `GATE-002` admits only declared facts, comparison operators, and boolean composition. No expression evaluation, no I/O in a predicate. | by-design |
| Fact injection | A false fact fires a trigger and manufactures a `trigger-met` state. | Facts are operator-supplied and attributed; a trigger firing produces a *proposal*, never a promotion, so a false fact cannot reach `active` on its own. | by-design |
| Unknown treated as false | Candidates sleep forever, or worse, evaluate as satisfied. | An unknown fact evaluates to **unknown**, not false; the run reports the gap. | by-design |
| Money Ledger read leaks financial data into this scenario's stores | Revenue figures land in logs, board caches, or the database, widening the blast radius of a scenario that is supposed to hold none. | The board owns nothing and computes at read time; ledger values are never persisted and never logged. Short deadline on the call. | by-design |
| Ledger unavailable rendered as zero | An offer appears to have earned nothing when the truth is unknown. Integrity failure, not just a UX one. | `OT-P1-002`: unavailability is stated with a reason and never reported as zero. | by-design |
| Import writes bypassing the state machine | The migration inserts records in states the lifecycle would refuse, permanently corrupting the graph's invariants. | `OT-P0-006`: importer writes go through the same state machine, and run only after lifecycle enforcement is green. | required |
| Source deletion before verified import | Irrecoverable loss of the 22-file source catalog. | Import → verify per-source-file counts → **then** delete. Ordering is absolute. | required |
| Unsafe file upload handling (template `notes`) | Malicious or oversized upload affects storage. | Multipart handler validates metadata; BlobStore seam isolates bytes. | template-reference |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| **Operator-vs-agent identity is not yet real** | **high** | The promotion boundary is the scenario's headline guarantee and cannot be enforced until callers are distinguishable. Resolve before `OT-P0-005` is claimed as met — until then the requirement is designed, not delivered. |
| No auth model | conditional | Required before `OT-P2-003` (multiple offer books) or any networked deployment. |
| Proposal expiry not modelled | low | A stale proposal accumulating indefinitely is noise rather than risk. Revisit if the proposals surface becomes unusable. |
| No rate limit on trigger evaluation | low | Only if `OT-P2-004` (scenario-sourced facts) makes evaluation reach other services, at which point a runaway schedule becomes a denial-of-service against them. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — what may and may not be emitted
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
