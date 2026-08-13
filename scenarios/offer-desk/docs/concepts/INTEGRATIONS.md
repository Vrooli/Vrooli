# Integrations — Offer Desk

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

This scenario declares no external Vrooli resource. Nodes, triggers and proposals are small, local and durable, and no operational target is served by a shared resource — SQLite's single-writer model is a fit, not a compromise, for an append-only status history.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None | not-applicable | Nodes, triggers and proposals are small, local and durable. No operational target is served by a shared resource. | Revisit only if a second writer appears. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| Money Ledger | planned (P1) | Supplies actuals for the earned-versus-intended join. Optional at runtime and degrades to a stated unavailability. | Typed read client with a short deadline. One-way: Money Ledger has no knowledge of offers and must never gain any. |
| Landing Page Business Suite | **deliberately none** | Commerce data reaches this scenario indirectly, through Money Ledger. Coupling directly to a billing system would put pricing and entitlement concepts into a scenario whose whole boundary is that it holds neither. | — |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None | not-applicable | This scenario reaches no external service. Every fact it evaluates is supplied by an operator or, at P2, by another Vrooli scenario. | Revisit only with OT-P2-004. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| Money Ledger | unreachable, slow, or erroring | The board marks the actuals source unavailable with a reason. **Never zero earnings** — indistinguishable from an offer that genuinely earned nothing. Every other board source continues to rank. | board tests with a faked unavailable ledger |
| Fact sources (P2) | a named fact has no current value | The trigger evaluates to **unknown**, the node stays in `candidate`, and the run reports the missing fact. Never false. | gates evaluation tests |

## Why This Scenario Has Almost No Integrations

The short dependency list is a design outcome, not an early-stage accident.

This scenario's entire boundary is that it holds *intent* — what should be sold and what state that intent is in. The moment it reads a payment processor directly, it acquires opinions about money; the moment it writes to one, it acquires opinions about customers. Both belong elsewhere, and both are already owned: commerce upstream, and actuals in Money Ledger.

So the only runtime dependency is a read of Money Ledger for one optional join, and even that degrades to an honest unavailability rather than blocking anything. The scenario is fully useful with no dependency reachable at all.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
