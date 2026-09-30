# Security — Treasury

This document records the security posture of this scenario: what it
holds, who may act on it, what can go wrong, and what is not yet handled.

This scenario's security model differs from an ordinary scenario in one
respect worth stating first. **The dominant threat is not an external
attacker — it is a legitimately authenticated agent that has been
manipulated by content it read.** A treasury agent's job is to visit
counterparty pages and API responses, which means it consumes untrusted
input by design. Every mitigation below is shaped by that.

## Purpose Of This Document

Use this document to answer:

- What sensitive data does this scenario hold, and what does it refuse to hold?
- Who is authorized to do what, and where is that enforced?
- What are the realistic attacks, and what structurally prevents them?
- What is still unhandled?

## Data Sensitivity

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Mandates | high | mandate | A live mandate is spending authority. Reading one reveals what an agent may buy; forging one would be a direct financial compromise. |
| Budgets and scope entries | high | budget | Policy configuration. Modifying a cap is equivalent to raising a spending limit. |
| Authorizations and holds | high | authorization | Reveal spending intent and pattern. |
| Charges and outcomes | high | settlement | Financial record of what moved. |
| Evidence records | high | evidence | The most sensitive artefact here: an evidence record joins who authorized, what was bought, from whom, and for how much. Operator-realm read only. |
| Instrument records | high | instrument | Reference and scope only. Never credential material. |
| Idempotency keys | medium | settlement | Not secret, but predicting one could let a caller probe whether a charge occurred. Treat as unguessable. |
| Books and beneficiary identity | medium | book | Business-identifying. |
| Ledger emissions | medium | ledger | Amount, counterparty and basis; no narrative or reasoning. |
| Relay attempts | low | approval | Delivery telemetry. |

**Deliberately not held, and why that is a security property rather than a
limitation:**

| Not held | Lives in | Security consequence |
|---|---|---|
| Card numbers, CVV, expiry, keys, facilitator secrets | `secrets-manager` | Keeps this scenario outside cardholder-data scope **by construction**, not by policy. A compromise here yields references, not credentials. |
| Identity documents | `document-manager` | A passport breach cannot originate here because a passport never arrives here. |
| Persona, addresses, contact channels | `persona` | Reduces the value of a compromise of this scenario alone. |
| Agent workload identity and signing keys | `agent-manager` | This scenario verifies and cannot mint. It cannot forge the identity it checks. |
| Financial position, runway, tax treatment | `money-ledger` | Limits the blast radius of a read compromise to authorization data rather than complete financial posture. |

## Auth And Authorization

**Two Connect services, two realms.** This is the load-bearing security
decision in the scenario.

| Service | Realm | May do | May not do |
|---|---|---|---|
| `AgentSpend` | agent | Request authorization, execute an automated approved charge, read its own attempt results and budget headroom. | Assert a rail outcome or manual receipt; mutate policy, budgets, gating, mandates, instruments or freeze state — **because no such method is declared on the service.** |
| `TreasuryAdmin` | operator | Policy, budgets, mandates, instruments, approval resolution, freeze, evidence read, and operator-attested manual settlement. | — |

The guarantee is **structural, not a permission check**. `TRS-P0-004`
asserts against the generated proto service descriptor that `AgentSpend`
declares no mutating policy method. An injected prompt can attempt any
call it likes; it cannot invoke an RPC that does not exist. A permission
check, by contrast, is code that could be misconfigured, bypassed by a
future convenience endpoint, or argued past by a sufficiently creative
payload.

`TreasuryAdmin` separately fails closed behind an operator credential. It
accepts `X-Vrooli-Operator-Token` (or an equivalent bearer token) matching
`TREASURY_OPERATOR_TOKEN`, with `API_TOKEN` as a fallback. Missing
configuration disables the admin surface rather than opening it. Any request
that also carries `X-Agent-Identity-Token` is rejected before the operator
credential is considered, so a mixed-realm request cannot borrow ambient
operator authority.

**Authorization is always server-side.** `TRS-P0-002` requires the
evaluator to derive its verdict only from stored state. Caller-supplied
verdict, allowance or override fields are **ignored when present rather
than rejected**, so a probing caller learns nothing from the difference in
response.

**Identity fails closed.** `TRS-P0-005`: transport failure, expired token,
invalid signature and absent token all refuse. Nothing is cached across
requests, because a cache converts a revocation into a delayed refusal —
the wrong failure mode for money. UI and CLI enforce nothing; they are
translation layers over the API.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| Card and issuer credentials | `secrets-manager` | P1 (`TRS-P1-003`) | Resolved by reference at use time. Never persisted here, never returned in a response body, never logged. |
| x402 facilitator keys | `secrets-manager` | P1 (`TRS-P1-001`, `TRS-P1-002`) | Same handling. A self-hosted facilitator's signing material is the highest-value secret in the system. |
| `agent-manager` verification | none held | P0 | Verification is a live call; this scenario holds no signing secret and therefore cannot mint an identity token. |
| Operator API credential | `TREASURY_OPERATOR_TOKEN` or `API_TOKEN` | P0 | Read only at composition time and compared in constant time. Never persisted or returned. Missing configuration disables `TreasuryAdmin`. |
| Mandate signing key | `TREASURY_MANDATE_SIGNING_KEY` | P0 | Read only at composition time and used to HMAC the canonical immutable grant. It is independent from the operator API credential; missing configuration disables issuance rather than creating unsigned authority. |
| Ledger emission credentials | per `money-ledger` contract | P0 | Scoped to emission only. |

No secret is held in `.vrooli/service.json`, environment defaults, or
scenario storage.

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| **Prompt injection raising a charge** — a counterparty page persuades the agent to authorize more, or to a different recipient. | Direct financial loss. | The mandate is signed *before* the agent reads untrusted content. Evaluation is server-side against stored state. Caller-supplied outcome and rail fields are rejected. Approval, instrument and settlement scopes project from stored authority, with live identity rebound at execution. | verified for the P0 authorization and settlement path |
| **Prompt injection disabling the gate** — the agent is persuaded to turn off approval. | Loss of the human checkpoint. | The method does not exist on the agent-facing service. The generated descriptor is pinned to an exact reviewed method allowlist, and every admin method rejects the agent realm. `TRS-P0-004`. | verified |
| **Compromised agent token** — a valid token is stolen or a run is hijacked. | Bounded. The token's scopes are attenuated by `agent-manager` and can never widen; the mandate caps amount, counterparty and expiry; approval still gates. | Structural: the blast radius of a stolen token is exactly one mandate's cap. | designed |
| **Replay of a charge request** — the same authorization is submitted repeatedly. | Double spending. | Caller-supplied idempotency key is required, not defaulted. A unique durable claim commits before the rail call and a repeated key returns the first record. `TRS-P0-011`. | verified by concurrent SQLite and real-transport tests |
| **Retrying an ambiguous settlement** — the rail was called, the response lost. | Double charge. | `unknown` is a first-class state resolved by querying the rail, never by retrying. `TRS-P0-011`; the generated Quint model checks this transition boundary and production tests replay it. | verified |
| **Identity authority unavailable, treated as permissive** | Unattributable spend. | Fail closed. `TRS-P0-005`. Explicitly rejects the degraded-grade pattern used for ordinary attribution writes. | verified |
| **Credential exfiltration through a response or log** | Card compromise. | Instrument storage contains a logical reference only; API responses redact even that reference; use-time resolution follows a revalidated live mandate. Schema and transport tests enforce these boundaries. | partial — storage and API boundaries verified; automated rail logging remains to be exercised |
| **Third-party custody drift** — a feature quietly starts holding value for someone else. | Regulated activity entered without deciding to. | The schema admits exactly one beneficiary identity; subordinate tables have no beneficiary field; raw-SQL tests reject a second beneficiary and cross-book authority. `TRS-P0-010` tests the schema, not just the handler. | verified |
| **Evidence tampering to hide a spend** | Loss of the audit property the scenario exists for. | Append-only SQLite triggers refuse update/delete; colliding identifiers with different content are rejected; no API, CLI, or UI purge/rewrite operation exists. | verified |
| **Money Ledger unavailable after value moved** | Downstream financial position temporarily omits a real expense. | Terminal settlement, immutable evidence, and a durable idempotent outbox commit together. A background dispatcher retries; destination deduplication is `(adapter_id, external_id)`. | verified by automated tests and attended stop/settle/restart drill |
| **Approval fatigue** — an operator approves without reading. | The human gate becomes ceremonial. | Partly a UX problem, addressed by making amount and counterparty the first thing read and by keeping the queue short through well-scoped budgets. **Not fully mitigated** — see gaps. | partial |
| **Malicious or compromised rail adapter** | Funds diverted to an attacker-controlled counterparty. | Adapters are governed dependencies installed through `scenario-dependency-analyzer`; the mandate's counterparty scope bounds where money can go regardless of adapter behaviour. | designed |
| Unsafe file upload handling | Malicious or oversized upload could affect storage. | Multipart handler validates metadata and BlobStore seam isolates bytes. | template-reference |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| The first automated provider remains later work; current production movement is operator-attested manual settlement. | high | Complete the first automated rail and repeat the attended transaction proof against its query semantics before claiming unattended production readiness. |
| Approval fatigue is only partly addressed. | medium | Observe real queue depth once an automated rail is live. If operators approve in under a second, the gate is ceremonial and needs a design answer, not more copy. |
| No rate limiting on the agent-facing authorization surface. | medium | Before the first automated rail. A compromised agent cannot exceed its mandate, but it can generate unbounded refusals and evidence rows. |
| The operator realm uses one static bearer credential rather than short-lived per-operator sessions. | medium | Before remote or multi-operator deployment. Replace the `operatorauth.Authorizer` implementation without weakening the two-service descriptor boundary. |
| Evidence read access is realm-scoped but not further partitioned. | low | Multi-operator use, which the operator-funds-only decision currently makes out of scope. |
| Instrument revocation is local; propagating it to each rail is adapter-specific. | medium | Second automated rail, when the shared shape becomes visible. |
| `agent-manager` is a single point of failure on the spend path. | accepted | Deliberate, per the fail-closed decision. Revisit only if a second independent verification path exists. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`../concepts/FLOWS.md`](../concepts/FLOWS.md) — the illegal transitions that carry security weight
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`DECISIONS.md`](DECISIONS.md) — why the gate is structural rather than a check
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
