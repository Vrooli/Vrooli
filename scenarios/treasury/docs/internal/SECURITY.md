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

<!-- EXAMPLE-DOMAIN:notes START -->
The shipped worked-example `notes` domain carries placeholder data only
(removed by `template-manager detemplate`):

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Template notes data | low | notes reference | Local development data only; replace with real scenario data classification. |
| Attachment bytes | unknown | notes reference | Treat as potentially sensitive if retained in product scope. |
<!-- EXAMPLE-DOMAIN:notes END -->

## Auth And Authorization

**Two Connect services, two realms.** This is the load-bearing security
decision in the scenario.

| Service | Realm | May do | May not do |
|---|---|---|---|
| `AgentSpend` | agent | Request authorization, report an outcome, read its own attempt results and budget headroom. | Anything that mutates policy, budgets, gating, mandates, instruments or freeze state — **because no such method is declared on the service.** |
| `TreasuryAdmin` | operator | Everything: policy, budgets, mandates, instruments, approval resolution, freeze, evidence read. | — |

The guarantee is **structural, not a permission check**. `TRS-P0-004`
asserts against the generated proto service descriptor that `AgentSpend`
declares no mutating policy method. An injected prompt can attempt any
call it likes; it cannot invoke an RPC that does not exist. A permission
check, by contrast, is code that could be misconfigured, bypassed by a
future convenience endpoint, or argued past by a sufficiently creative
payload.

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
| Ledger emission credentials | per `money-ledger` contract | P0 | Scoped to emission only. |

No secret is held in `.vrooli/service.json`, environment defaults, or
scenario storage.

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| **Prompt injection raising a charge** — a counterparty page persuades the agent to authorize more, or to a different recipient. | Direct financial loss. | The mandate is signed *before* the agent reads untrusted content. Evaluation is server-side against stored state. Caller-supplied override fields are ignored. The amount and counterparty in the approval prompt come from the mandate, not from the agent's request. | designed |
| **Prompt injection disabling the gate** — the agent is persuaded to turn off approval. | Loss of the human checkpoint. | The method does not exist on the agent-facing service. `TRS-P0-004`. | designed |
| **Compromised agent token** — a valid token is stolen or a run is hijacked. | Bounded. The token's scopes are attenuated by `agent-manager` and can never widen; the mandate caps amount, counterparty and expiry; approval still gates. | Structural: the blast radius of a stolen token is exactly one mandate's cap. | designed |
| **Replay of a charge request** — the same authorization is submitted repeatedly. | Double spending. | Caller-supplied idempotency key is required, not defaulted. A repeated key returns the first outcome. `TRS-P0-011`. | designed |
| **Retrying an ambiguous settlement** — the rail was called, the response lost. | Double charge. | `unknown` is a first-class state resolved by querying the rail, never by retrying. `TRS-P0-012`, and the reason settlement targets formal-model maturity. | designed |
| **Identity authority unavailable, treated as permissive** | Unattributable spend. | Fail closed. `TRS-P0-005`. Explicitly rejects the degraded-grade pattern used for ordinary attribution writes. | designed |
| **Credential exfiltration through a response or log** | Card compromise. | Credentials never enter this scenario; only references. `TRS-P1-003` asserts non-leakage. The rails page carries an `element-absent` experience claim for credential material. | designed |
| **Third-party custody drift** — a feature quietly starts holding value for someone else. | Regulated activity entered without deciding to. | The schema admits exactly one beneficiary identity. `TRS-P0-010` tests the schema, not just the handler. | designed |
| **Evidence tampering to hide a spend** | Loss of the audit property the scenario exists for. | Append-only store; update and delete refused by construction. No operator-facing purge exists. | designed |
| **Approval fatigue** — an operator approves without reading. | The human gate becomes ceremonial. | Partly a UX problem, addressed by making amount and counterparty the first thing read and by keeping the queue short through well-scoped budgets. **Not fully mitigated** — see gaps. | partial |
| **Malicious or compromised rail adapter** | Funds diverted to an attacker-controlled counterparty. | Adapters are governed dependencies installed through `scenario-dependency-analyzer`; the mandate's counterparty scope bounds where money can go regardless of adapter behaviour. | designed |
| Unsafe file upload handling | Malicious or oversized upload could affect storage. | Multipart handler validates metadata and BlobStore seam isolates bytes. | template-reference |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| Nothing is implemented yet. Every mitigation above is `designed`, not `verified`. | high | Implementation. No claim here should be repeated as fact until its requirement's validation is earned by a passing test rather than asserted. |
| Approval fatigue is only partly addressed. | medium | Observe real queue depth once an automated rail is live. If operators approve in under a second, the gate is ceremonial and needs a design answer, not more copy. |
| No rate limiting on the agent-facing authorization surface. | medium | Before the first automated rail. A compromised agent cannot exceed its mandate, but it can generate unbounded refusals and evidence rows. |
| Evidence read access is realm-scoped but not further partitioned. | low | Multi-operator use, which the operator-funds-only decision currently makes out of scope. |
| Instrument revocation is local; propagating it to each rail is adapter-specific. | medium | Second automated rail, when the shared shape becomes visible. |
| `agent-manager` is a single point of failure on the spend path. | accepted | Deliberate, per the fail-closed decision. Revisit only if a second independent verification path exists. |
| No formal model exists yet for settlement. | medium | Settlement implementation. Its maturity target is level 5 precisely because hand-written tests are not convincing for exactly-once under concurrent retry. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`../concepts/FLOWS.md`](../concepts/FLOWS.md) — the illegal transitions that carry security weight
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`DECISIONS.md`](DECISIONS.md) — why the gate is structural rather than a check
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
