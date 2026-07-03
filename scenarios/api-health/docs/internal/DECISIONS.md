# Decisions — API Health

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
| 2026-07-03 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs keep manifest-backed headings and validation metadata while the content is API Health-specific. | Revisit when scenario adopts a different template or doc contract. |
| 2026-07-03 | Treat scenario-auditor as migration input, not a runtime dependency. | API Health exists to retire the API-rule residue in scenario-auditor. | `.vrooli/service.json` lists scenario-auditor disabled/ignore; implementation must read old rule intent from source and produce a ledger instead of calling scenario-auditor. | Remove once the migration ledger and cutover are complete. |
| 2026-07-03 | Disable requirements auto-sync until implementation tests exist. | The foundation registry contains planned validation refs. | Requirement statuses cannot be accidentally marked complete by refless/planned tests. | Re-enable when real `[REQ:*]` tests exist and validation refs point at concrete files. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| _None yet._ | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
