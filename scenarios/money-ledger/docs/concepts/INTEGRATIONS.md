# Integrations — Money Ledger

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

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

## Vrooli Resources

This scenario declares no external Vrooli resource for P0. Books, journal, ingest and position are small, local and durable, and SQLite's single-writer model suits an append-only journal. A **secret store is required before P1**: the first adapter that authenticates to a third party needs credentials, and those must never live in scenario config or in the journal database.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None (P0) | not-applicable | Books, journal, ingest and position are small, local and durable. No P0 target is served by a shared resource, and SQLite's single-writer model matches an append-only journal well. | Revisit if a second writer appears, or if the journal outgrows local storage. |
| Secret store | required-before-P1 | The first adapter that authenticates to a third party needs credentials, and those must not live in scenario config or the journal database. | Adding any adapter beyond manual entry and file import. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| Landing Page Business Suite | planned (P1) | Source of subscription and charge events via the commerce adapter. Read-only and one-way — this scenario never writes to it and holds no billing, entitlement or customer state. | The money-event contract, satisfied by an adapter. LPBS is not privileged; it is the first non-trivial adapter, chosen because it is real and messy enough to prove the contract. |
| Offer Desk | inbound only | Offer Desk reads this scenario twice: **actuals** for its earned-versus-intended join, and **financial posture** — runway, goal verdicts with sustain-window progress, and the default-alive gap — because Offer Desk is the `monetization` team's single address and its financial member must be served from the same surface as its catalog members. **This scenario has no knowledge of offers** and must never gain any — the direction is one-way by design, so the ledger stays usable by an operator who sells nothing, which is what the direct-product thesis rests on. | Offer Desk calls this scenario's read API. Nothing here calls Offer Desk. Both reads are bounded by Offer Desk's deadline, not ours; this scenario has no obligation to be fast for them and no awareness that they happened. A partial position must stay legibly partial in the response so the caveat survives to the consumer's surface. |
| Secrets Manager | planned (P1) | Adapter credentials. | Platform secret-store client. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None direct (P0) | not-applicable | Manual entry and file import require no external service, which is why they are the first two adapters. | — |
| Payment processors, banks, marketplaces, brokerages, custodians | planned (P2+) | Each becomes an adapter behind the same contract, added only when an operator actually has money moving through it. | The money-event contract. **Bank access is via aggregator API or file export only — this scenario never stores bank login credentials.** |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| Any adapter's upstream | HTTP error, auth failure, timeout, or malformed payload | An availability record with a reason and the age of the last success. **Never a zero-valued event** — a silent zero is indistinguishable from a real one, and would corrupt every derived figure. Position marks itself partial while any contributing adapter is unavailable. | ingest service tests with faked failing adapters; position tests asserting partial state |
| Secret store | unavailable at adapter start | The adapter does not run and reports unavailable. It never falls back to an unauthenticated call or a cached credential. | ingest tests |
| Landing Page Business Suite | unreachable or slow | Bounded by a short deadline; the commerce adapter reports unavailable and the cursor does not advance, so the next run re-reads rather than skips. | commerce adapter tests |

## The Adapter Contract Is The Integration Strategy

This scenario has an unusual integration posture and it is deliberate: **it does not integrate with named systems — it defines one inbound shape and lets systems satisfy it.**

Every source of money, however different its API, produces the same thing: a dated, signed, attributed event with provenance and a basis. Standardising that shape means a new upstream never requires a change to books, journal or position, and never requires the upstream to conform to anything on its side. It is why no P0 target names a specific external system.

Three rules keep the posture from eroding:

1. **One door.** No caller, internal or external, writes to the journal except through the contract. An architecture test enforces it.
2. **Adapters only emit.** An adapter may not write a balance, alter a goal, or reach another domain. The most breakable part of the system gets exactly one verb.
3. **Unavailable is a value.** An adapter that cannot run reports unavailability with a reason and an age. It never substitutes zero, and never silently omits a window.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
