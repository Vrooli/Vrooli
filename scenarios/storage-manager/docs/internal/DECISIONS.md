# Decisions — Storage Manager

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
| 2026-07-08 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-09-14 | Measure a scenario's class-data entries at both the class root and the lifecycle data directory (`api-core/storage.ResolveOwnerStorageLocations`); prune only the primary path. | Live scenarios opened through `SQLitePath` write `scenarios/<name>/data`, but owner resolution measured only `~/.vrooli/data/vrooli/<name>`. Agent Manager's 274 GiB database and experience-manager's 17 GB database were unmeasured. Across 292 declared entries the primary path changed for none, 138 gained a location, and 92 of those exist. | Budgets alarm on real bytes. Pruning targets are unchanged, so no regenerable WAL/SHM entry can start deleting a live database's file. | Revisit when every scenario resolves data through one root, or a pruner needs the lifecycle location. |
| 2026-09-14 | Measure non-regenerable budgets before the protected runtime-home refusal. | `~/.vrooli/data` and `~/.vrooli/state` are contract-protected, so every class-data and class-state budget was refused before it was measured and no non-regenerable alarm ever fired. | The refusal still blocks pruning; measurement, alarms, and owner reclaim now run. | Revisit if measurement itself becomes expensive enough to need gating. |
| 2026-09-14 | Backstop non-regenerable breaches by asking the owner through a declared `reclaim.operation`, never by deleting. | Storage-manager cannot know which rows or files an owner may drop. | Owner receipts and escalations appear in `/api/v1/retention/budget-status`; a failed or insufficient reclaim escalates. | Revisit when an owner needs authenticated or approval-gated reclaim. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
