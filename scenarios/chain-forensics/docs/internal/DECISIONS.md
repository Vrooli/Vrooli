# Decisions — Chain Forensics

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
| 2026-11-09 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-11-09 | Keep legal assertions separate from chain observations in the case model. | The 1HQ3 complaint and public Bitcoin transactions support different kinds of statements. | Complaints are stored as sourced claims; only transaction inputs/outputs produce observed edges. | Revisit if the model gains a signed evidentiary-act type that preserves that distinction. |
| 2026-11-09 | Keep graph/evidence logic in `github.com/vrooli/chain-forensics`, without provider clients or scenario transport. | CI2 must be able to exercise the same deterministic logic, while source access and retention remain owner-controlled. | The first package has no external dependencies; scenario API import waits for a governed first-party dependency route. | Revisit after SDA establishes an approved local-module consumption path. |
| 2026-11-09 | Use mempool.space public reads and retain immutable response hashes for the BTC slice. | The 1HQ3 address and transaction facts are publicly observable; rate limits and provider availability still constrain live queries. | Case fixture carries source URL, capture time, raw SHA-256, and explicit gaps; the source does not prove real-world ownership. | Recheck source terms/limits before promoting the adapter beyond a bounded public-case slice. |
| 2026-11-09 | Keep packets private and make serialization explicit. | Public data can become sensitive when analyst notes and conclusions are added. | `MarshalPrivatePacket` refuses non-private cases and only returns bytes; the scenario must still implement local storage and explicit export. | Revisit when the scenario's case store and packet redaction/export workflow are implemented. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
