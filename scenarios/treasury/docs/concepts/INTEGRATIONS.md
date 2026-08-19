# Integrations — Treasury

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

One dependency here is unlike the others and is worth stating first.
`agent-manager` is a **hard runtime dependency on the spend path**:
identity verification is a live call, and `TRS-P0-005` refuses spend that
cannot be verified. That is a deliberate availability cost accepted in
exchange for never recording an unattributable payment. Every other
dependency degrades; this one refuses.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, persistence-backed domains | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| `agent-manager` | scenario | yes, on the spend path | `identity` | Live verification call; `X-Agent-Identity-Token` header | **Fail closed** — automated spend is refused and the cause recorded. Console and read paths stay available. |
| `money-ledger` | scenario | yes, eventually | `ledger` | Its existing money-event contract | Emission queues and retries. Settlement is never blocked; money that moved is still recorded locally. |
| `secrets-manager` | scenario | yes, once a rail holds credentials | `instrument`, `rail` | Credential resolution by reference at use time | Automated rails that need a credential refuse. Manual rail and all read paths unaffected. |
| `persona` | scenario | no for machine-native rails | `rail` (card, checkout) | Persona resolution for counterparties expecting a legal person | Rails needing a persona refuse; x402 and manual rails unaffected. |
| `notification-hub` | scenario | no | `approval` | Relay of approval requests | Relay attempt is recorded as failed; the approval outcome is unchanged and the console queue remains fully functional. |
| `browser-automation-studio` | scenario | no (P2) | `rail` (checkout) | Driven checkout under a scoped instrument | Checkout rail unavailable; other rails unaffected. |
| `offer-desk` | scenario | no (P2) | `pricing` (deferred) | Price lookup for inbound metering | Falls back to a locally declared price, with the source recorded on the receipt. |
| x402 facilitator | managed resource (P1) | no for P0 | `rail` (x402) | Verify and settle stablecoin payments | Outbound x402 refuses; inbound metering returns unavailable rather than serving unpaid. |

## Vrooli Resources

**P0 declares no external resources.** SQLite is embedded and sufficient for
the authorization spine. P1 adds one optional managed facilitator resource;
its checked-in configuration deliberately advertises no network or scheme, so
registration and health never imply payment readiness.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| SQLite (embedded) | active | Single-operator custody and low authorization volume. Serialised writers satisfy the row-locking requirement directly. | Inbound x402 metering (`TRS-P1-002`) is the one declared trigger. See [`DATA.md`](DATA.md). |
| x402 facilitator | registered, fail-closed bootstrap (P1) | The digest-pinned Apache-2.0 Second State Rust facilitator is self-hosted on loopback. Its empty checked-in network/scheme configuration cannot move value. | Provision an attended signer, RPC route, explicit network allowlist, funded test wallet, and testnet proof before declaring `TRS-P1-001` / `TRS-P1-002` operational. Selection evidence is in [`../internal/DECISIONS.md`](../internal/DECISIONS.md). |
| Shared database | not-applicable | Not warranted at single-operator volume. | Only the SQLite trigger above. |
| Queue / broker | not-applicable | Ledger emission retry is a small durable table, not a broker workload. | Emission volume that a table cannot drain, which single-operator spend will not produce. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| `agent-manager` | required (P0) | The delegation chain, scope attenuation and signed run claims already exist and are ecosystem-wide. Reimplementing them here would create a second, weaker identity system. | Live verification; `persona_id` read from `Claims.Meta` once that key lands. Consumed only — this scenario adds nothing to the chain. |
| `money-ledger` | required (P0) | Every settlement must reach the journal. | One-way emission through the contract every source satisfies. `money-ledger` gains no knowledge of this scenario, which is what preserves its neutrality. |
| `secrets-manager` | required (P1) | Credential material must not enter this scenario's storage. Keeping it out is what holds `money-ledger`'s no-credential-storage non-goal across the pair. | Resolution by reference at use time; credentials never reach a response body. |
| `persona` | optional (P1) | Counterparties expecting a legal person need a transacting identity this scenario deliberately does not model. | Persona resolution. Not required for machine-native rails. |
| `notification-hub` | optional (P0) | Approval must work when it is down, so it is an enhancement rather than a dependency. | Relay attempt with recorded outcome. |
| `browser-automation-studio` | optional (P2) | Counterparties without an API. | Driven checkout under a pre-scoped instrument. |
| `offer-desk` | optional (P2) | Joins what should earn to what a call costs. | One-way read. `offer-desk` gains no knowledge of this scenario. |
| `document-manager` | not-applicable | Identity documents belong there, but this scenario never touches them — `persona` holds that binding. | Explicitly no contract. Recorded so a later reader sees this was decided, not overlooked. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None named, by design | active decision | No P0 or P1 operational target names a payment vendor. Rails are adapters behind one execution contract, so a vendor is a configuration choice rather than an architectural one. | Adapter interface in `api/internal/rail/`. |
| Stablecoin settlement network | adapter implemented; live configuration pending (P1) | The x402 rail settles on a public network only after operator provisioning. | Inbound settlement is reached through the facilitator. Outbound signing is delegated to an operator-owned wallet RPC and the paid request is sent directly to the priced endpoint. |
| Card issuing provider | planned (P1) | The scoped-card rail needs an issuer. | Behind `api/internal/rail/card/`; the rail interface carries no vendor-specific type, which is what keeps the choice reversible. |

All third-party packages are installed through
`scenario-dependency-analyzer`. No raw package-manager call and no
hand-edit of `.vrooli/dependencies/approved-dependencies.json` is
permitted, per repository dependency governance.

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| `agent-manager` | Transport error, non-200, invalid signature, expired token | **Refuse the spend.** Record the refusal as evidence with its cause. Never accept at a degraded grade. Read paths and the operator console stay available. | `TRS-P0-005` |
| `money-ledger` | Emission rejected or unreachable | Queue and retry. Settlement completes regardless; local evidence is authoritative until the emission is accepted. | `TRS-P0-008` |
| `secrets-manager` | Credential resolution fails | The rail needing it refuses. No cached credential is used, because a cache would convert a revocation into a delayed refusal. | `TRS-P1-003` |
| `notification-hub` | Relay call fails | Record the relay attempt as failed. Approval outcome unchanged; console queue unaffected. | `TRS-P0-006` |
| x402 facilitator | Verify or settle fails | Outbound refuses. Inbound returns unavailable rather than serving an unpaid request, because serving unpaid is the one failure that costs real money. | `TRS-P1-001`, `TRS-P1-002` |
| `persona` | Resolution fails | Rails requiring a persona refuse; machine-native rails are unaffected. | `TRS-P1-003` |
| Rail returns an ambiguous result | Timeout or lost response after the call was made | Enter the `unknown` settlement state. Resolve by querying the rail, never by retrying — retrying an unknown is how double charges happen. | `TRS-P0-011` |

**Degradation summary.** Everything except identity verification degrades
gracefully: the console stays usable, evidence keeps accruing, and the
manual rail keeps working. Identity is the deliberate exception.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DOMAINS.md`](DOMAINS.md) — which domain uses each dependency
- [`DATA.md`](DATA.md) — storage ownership
- [`FLOWS.md`](FLOWS.md) — degradation behaviour inside each flow
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — dependency decisions and their reasons
