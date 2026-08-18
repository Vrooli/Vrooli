# Decisions — Treasury

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

Every decision below was made during the workshop that preceded this
scenario's creation, with the operator in the loop. Several of them look
like defaults and are not — the SQLite choice, the single-scenario shape,
and the fail-closed identity rule each had a plausible alternative that was
considered and rejected for a stated reason. Those reasons are the point of
this document.

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
| 2026-08-18 | **The core object is a mandate, not a balance.** | A wallet models *how much money there is*. That is `money-ledger`'s question. The question here is *what a human permitted*. | Rails become adapters satisfying one contract, so no P0 target names a vendor and a new rail arrives without a rewrite. Authority becomes a signed object with a cap and an expiry rather than a credential handed over. | A rail appears whose authorization model genuinely cannot be expressed as a scoped, expiring grant. |
| 2026-08-18 | **Authorization and settlement stay in one scenario with a hard internal seam.** Rejected: splitting them into peer scenarios. | The industry splits them — mandate protocols authorize, settlement protocols move value — and `money-ledger`/`offer-desk` are precedent for splitting siblings. | Avoids a distributed-transaction problem (authorize here, charge there, crash between) inside a single money movement. Also avoids shipping a settlement scenario with no independent value proposition, which would fail the bar that every scenario stand alone. The domain seam is kept clean so the split remains available. | The policy engine is asked to govern non-money actuation, at which point this becomes an authority service and the cut is already drawn. |
| 2026-08-18 | **Spending and earning live together.** Rejected: earning as separate middleware or a control-plane concern. | x402 is symmetric — the same protocol pays for a request and charges for one. | One balance, one facilitator, one policy engine, one evidence trail. Separating them would duplicate all four to serve a distinction no operator experiences. | Earning volume or its consumers diverge far enough from spending that the shared policy engine becomes a constraint rather than a saving. |
| 2026-08-18 | **The agent-facing service declares no policy-mutating method.** Rejected: a permission check on a shared service. | Prompt injection is the dominant threat, and a check is something an injected prompt can try to talk past. | Two Connect services: `AgentSpend` (request, report) and `TreasuryAdmin` (operator realm). The guarantee is the *absence of the RPC*, asserted against the generated proto descriptor, not an authorization rule. An agent holding a valid token cannot disable its own gate. | Never, without an explicit replacement for the structural guarantee. A convenience that adds a policy method to the agent surface silently deletes this decision. |
| 2026-08-18 | **Identity verification fails closed.** Rejected: recording spend at a degraded evidence grade when the authority is unreachable, which is what the ecosystem's attribution model does for ordinary writes. | Attribution for a knowledge write and authorization for a payment have different stakes. | `agent-manager` becomes a hard runtime dependency on the spend path. An outage stops automated spend. The operator console, evidence reads and the manual rail stay available. | A second independent verification path exists, making the single dependency avoidable rather than merely inconvenient. |
| 2026-08-18 | **Consume `agent-manager`'s identity; add nothing to it here.** | `agent-manager/api/internal/identity` already ships signed run claims, a `Subject` naming the verified owner account, explicit attenuated scopes, and `Attenuate()` enforcing one-way narrowing that a child can never widen or outlive. | An earlier draft of this scenario proposed building delegated sub-mandates. That already exists and is better than the sketch. This scenario verifies and reads; it never issues workload identity. The `persona_id` binding lands as one `Meta` key on `agent-manager`'s side, matching the `team_id`/`member_id` workshop already documented in `docs/agent-system/RUNTIME_ATTRIBUTION.md`. | `agent-manager`'s identity model changes shape, or a second identity authority appears. |
| 2026-08-18 | **SQLite, not a shared database.** Rejected: Postgres from day one, on the strength of the credit-wallet precedent. | That precedent needed Postgres because it serves many customers concurrently. This scenario is single-operator with low authorization volume, and SQLite serialises writers, which satisfies the row-locking requirement directly. | Lighter resource footprint. The locking *discipline* carries over unchanged: lock the row, require a caller-supplied idempotency key, retried debit is a no-op after first commit. | **Inbound x402 metering (`TRS-P1-002`) is the single declared trigger.** Concurrent external payers are the only traffic this scenario does not control. `TRS-P1-002` carries a performance validation whose purpose is to observe that boundary. |
| 2026-08-18 | **Operator funds only, enforced at the schema boundary.** Rejected: a documented policy plus a runtime check. | Holding value for a third party is a regulated activity, and a boundary that erodes one feature at a time is how a system arrives there without deciding to. | The persistence schema admits exactly one beneficiary identity, so a third-party balance has nowhere to be stored even if a future handler tried. `TRS-P0-010` tests the schema, not just the handler. | Never casually. Crossing this line is a legal decision, not an engineering one, and needs advice before code. |
| 2026-08-18 | **The manual rail is first-class and ships first.** | The instinct is to treat "the operator paid it" as a fallback for when automation fails. | Phase 0 delivers a complete, useful product that moves no money: every agent purchase becomes proposed, bounded, approved, recorded and reconcilable. It is the right place to discover the mandate contract is wrong, before any custody or vendor commitment. | Never. A degraded manual path would mean an operator's own payment carries weaker evidence than an agent's, which inverts the trust model. |
| 2026-08-18 | **Evidence is append-only with no operator-facing purge.** | A scenario whose job is to prove what was authorized cannot also offer to erase the proof. | Storage grows with authorization volume, which is low by construction for a single-operator instance. Migrations may add columns but never rewrite an evidence row. | Storage growth becomes real, at which point the answer is export-then-archive with its own decision record — not a purge. |
| 2026-08-18 | **Headroom is computed, never stored.** | A stored balance is a number that can silently disagree with its own history. | Headroom derives from authorization and settlement records. Pending authorizations hold headroom before they settle, so two agents planning concurrently cannot both believe the same money is available. | Computation cost becomes measurable, which single-operator volume will not produce. |
| 2026-08-18 | **`money-ledger` gains no knowledge of this scenario.** | It admits every source as an anonymous adapter through one contract; that anonymity is what makes it a neutral record. | Emission is one-way and fire-and-record. Emission never blocks settlement — money that already moved must still be recorded locally, and local evidence is authoritative until the emission is accepted. | Never. A reverse dependency would make the journal an interested party in what it records. |
| 2026-08-18 | **`notification-hub` is a relay, not a dependency.** | An approval surface that stops working when another scenario is down is not a gate. | The scenario owns its approval queue and console. Relay failures are recorded and change no approval outcome. | Never. |
| 2026-08-18 | **Identity documents are out of scope, permanently.** | An earlier draft proposed a document vault here. `document-manager` already owns sensitivity classification as a fail-closed choke point, per-document custody receipts, and an append-only custody journal. | A passport is a document, not a payment object. `persona` holds the binding; this scenario never touches identity documents and stores no reference to one. | Never. |
| 2026-08-18 | **The `unknown` settlement state never auto-transitions to failed.** | After a lost response, whether money moved is genuinely unknown. | `unknown` is a first-class state resolved by querying the rail, not by retrying. This is why settlement targets maturity level 5 — exactly-once under concurrent retry against a partially-observable system is where a checked formal model earns its cost. | Never. Auto-transitioning an unknown to failed is precisely how a system double-charges while believing it retried a failure. |
| 2026-08-18 | **Keep the `vrooli-default` design kit and its token contract intact.** | The kit's calm, dense, operational posture and its long-running-workflow support match this scenario directly. | Scenario-specific work goes into what surfaces communicate, not into redefining the language. Additions above the kit's floor: status never by colour alone, tabular figures on amounts, destructive controls separated from navigation, accessible names on money-acting controls stating amount and counterparty. | The scenario's audience or density needs change materially. |

## Open Decisions

These are recorded as unresolved rather than silently defaulted.

| Question | Why it is open | What would settle it | Blocks |
|---|---|---|---|
| Which x402 facilitator implementation to self-host. | The most complete facilitator ecosystem is not Go. Adding a non-Go runtime to this stack is a real cost that should be chosen deliberately rather than absorbed. | A comparison of the available self-hostable facilitators against the cost of a second runtime in this scenario's resource footprint, plus whether a Go implementation has become viable. | `TRS-P1-001`, `TRS-P1-002`. Nothing in P0. |
| Whether the policy engine generalises beyond money. | `device-control`'s lease model — discover, describe capability, acquire lease, execute bounded action, retain evidence — is the same shape as a mandate. Two instances is not a pattern, and generalising now would be speculative. | A third bounded-actuation surface appearing, or `device-control` independently needing what this engine grew. | Nothing. The internal seam is kept clean so this stays possible. |
| The idempotency-key retention window. | 180 days is a judgement about plausible client retry behaviour, not a measurement. | Observing real client retry behaviour once an automated rail is live. | Nothing; the window is safe to lengthen. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| 2026-08-18 | This scenario would build delegated sub-mandates so an agent could issue a narrower grant to a sub-agent. | Consume `agent-manager`'s `Attenuate()`. | Reading `agent-manager/api/internal/identity` showed one-way scope narrowing with expiry bounds already shipped and ecosystem-wide. Building a second one would have created a weaker parallel authority. |
| 2026-08-18 | This scenario would hold identity documents under `secrets-manager`. | `document-manager` holds them; `persona` holds the binding; this scenario holds nothing. | `document-manager` already runs a sensitivity and custody spine built for exactly this, including a per-document custody receipt that answers "who read my passport and when" for free. |
| 2026-08-18 | Postgres from day one for balance mutation. | SQLite, with one declared migration trigger. | The Postgres precedent was a multi-customer billing surface. Single-operator custody does not carry that requirement, and SQLite's serialised writers satisfy the locking discipline directly. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — bounded contexts these decisions shaped
- [`../concepts/DATA.md`](../concepts/DATA.md) — storage and retention consequences
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency contracts
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
