# Decisions — Persona

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-08-18 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-08-18 | **Persona is a separate scenario, not an extension of `agent-manager`.** | `agent-manager` already owns workload identity: `Claims.Subject`, attenuated scopes, and `Attenuate()` one-way narrowing. The temptation was to add personas there. | Three arguments decided it: size (639 Go files, 52 internal packages), different consumers (write seams proving attribution versus outbound flows), and different failure semantics (an expired token means respawn; a broken persona means an orphaned account). The cost is one cross-scenario call on the act-as path. | If `agent-manager` is ever decomposed, revisit whether the identity package should absorb this. |
| 2026-08-18 | **Extend the delegation chain; never build a second identity system.** | A persona could have carried its own credentials and its own attenuation. | The chain is `legal person → persona → account subject → run token → child token`. The inner three are agent-manager's and are consumed unchanged. Integration is one `Meta` key (`persona_id`) plus the `persona.act-as:<id>` scope namespace, riding existing `IntersectScopes`/`Attenuate`. Persona permissions inherit child-can-never-widen for free. | If the token contract changes shape upstream. |
| 2026-08-18 | **Act-as fails closed when the verification authority is unreachable.** | The alternative — recording a weaker evidence grade and proceeding, mirroring the existing attribution model — was genuinely considered. | An unverifiable caller cannot act as anyone. No override flag, no environment variable. This makes `agent-manager` a hard runtime dependency on the act-as path; that availability cost was accepted deliberately because an outage is exactly when an attacker would want one. | Only if a same-host verification path removes the availability cost without weakening the guarantee. |
| 2026-08-18 | **Identity documents are bound, never stored.** `document-manager` holds the bytes. | Storing documents here would have removed a dependency and simplified deployment. | `document-manager` already runs sensitivity classification as a fail-closed choke point before tier selection, writes per-document custody receipts, and keeps an append-only custody journal. Duplicating that would create a second, weaker custody story for the most sensitive data in the system. "Who read my passport and when" becomes an existing query rather than new work. | If `document-manager`'s custody guarantees change materially. |
| 2026-08-18 | **No agent-readable read path for document content, under any scope.** | A read scope would have been simpler than release-into-handoff. | Release targets a pre-declared handoff and nothing else. This is the primary mitigation against prompt injection persuading an agent to exfiltrate identity documents. | Never, without a replacement mitigation of equal strength. |
| 2026-08-18 | **Act-as grants live in `prompt-manager`; the persona ACL lives here.** | Both could have lived in one place. | `prompt-manager` already owns teams, members, and contracts, so "which members may act as which persona" is ordinary member configuration rather than a second org chart. The ACL — which humans may act versus only propose — is persona-local because it is a property of the identity, not of the org. Optional at P0 so a single-operator install needs nothing extra. | If team configuration and persona policy start disagreeing in practice. |
| 2026-08-18 | **SQLite, with no external resource, through P2.** | Postgres was considered by analogy with sibling money-adjacent scenarios. | Single-operator, low write volume, no concurrent-writer pressure. Code retrieval is agent-initiated and serialised by the requesting flow. Adding a shared resource would cost deployment simplicity for no measured benefit. | A workload with many concurrent external writers, which this scenario does not have. |
| 2026-08-18 | **One OTP contract, many adapters, no privileged path.** | Email could have been special-cased as "the real one" with others bolted on. | Email, SMS provider, and `device-control` reading a leased phone all satisfy the same retrieval contract. An adapter never falls back to another persona's route — an unavailable adapter is a named failure, because a code fetched as the wrong identity is worse than no code. | If an adapter genuinely cannot satisfy the contract. |
| 2026-08-18 | **A handoff is a typed, resumable state — never an error.** | The obvious implementation is to fail the flow and let a human restart it. | The checkpoint captures everything already completed, so the human does the irreducible step and nothing more. Resumption must not require the originating run to still be alive. Expiry is a first-class terminal state rather than a silent drop. | Never; this is the scenario's central idea. |
| 2026-08-18 | **The scenario never solves, bypasses, or assists in bypassing a human verification control.** | There is obvious grey-area demand for CAPTCHA solving and synthetic identity. | No CAPTCHA solving, no biometric spoofing, no synthetic identity, no persona representing a person who has not authorised it. The wall is a product feature; circumventing it would violate counterparty terms and get every persona banned. Refusing loudly is treated as a positioning asset. | Never. |
| 2026-08-18 | **The journal is append-only, including across migrations.** | A mutable audit table is easier to evolve. | No verb rewrites or deletes a row; correction is a compensating entry. A migration may add columns but never rewrite rows. Refusals are journaled as prominently as successes, since a refused act-as is what an investigation comes for. A decreasing entry count is a critical integrity alarm. | Never. |
| 2026-08-18 | **No multi-tenant hosted deployment (Tier 4).** | It is the obvious commercial path for an identity product. | Hosting other people's persona state contradicts the scenario's entire positioning, which is custody. A hosted deployment needs a different threat model and should be treated as a different product. | Only with a documented threat model authored first. |
| 2026-08-18 | **Documentation authored ahead of implementation; page specs carry `draft` status.** | The alternative was to defer docs until code existed. | The contracts here are the hard part of this scenario, and settling them before code is the point. Experience claims stay aspirational until real routes and selectors exist, and `status: draft` makes reconciliation advisory rather than gating. | Flip pages to `active` as each route lands. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
