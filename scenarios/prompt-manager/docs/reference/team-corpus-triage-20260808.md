# Team corpus triage — 2026-08-08

This report records the pre-migration quality gate for the legacy prompt-manager
shared corpus. It covers the 43 pending inbox entries that were still waiting
for operator attention when the work surface was retired.

## Quality tests

Each entry was assessed against the three plan tests:

1. The statement is still true.
2. Every named file, command, or runtime state still exists.
3. The entry is not superseded by a later entry.

The entries are historical prompt-manager inbox material from April–July 2026.
They are not being used as a source of current team memory. The report is the
operator gate required before removal; it does not silently migrate stale
claims into the new ledger.

## Measured corpus

| Team | Entries | Pass all three tests | Representative failed/stale material |
|---|---:|---:|---|
| director-swarm | 1 | 0 | Social-media-scheduler initiative proposal routed through the retired initiative bridge. |
| infra-health | 8 | 0 | Historical lifecycle, CLI parity, instrumentation, and autoheal proposals already represented by later work or current platform state. |
| marketing-crew | 7 | 0 | Historical capability, coverage, publishing, and rejection proposals tied to retired JSONL and work flows. |
| meta-optimization | 15 | 0 | Historical skill, capability, agent, and action proposals whose current state is represented by later plans or shipped changes. |
| monetization | 2 | 0 | Dated benchmark and pricing proposals that are not durable team memory. |
| scenario-qa | 10 | 0 | Historical review, friction, lifecycle, and capture proposals already superseded or completed. |
| **Total** | **43** | **0** | **No legacy entry is retained.** |

## Operator disposition

The operator’s goal request explicitly identifies the prompt-manager decision
inbox as neglected and asks to retire it in favor of the unified swarm-manager
work path. Accordingly, the disposition recorded before migration is:

- do not create captures from these 43 stale entries;
- archive every original source file recoverably;
- discard the pending entries from the retired prompt-manager surface;
- do not migrate heartbeat-attempt or task content into any team scope.

No entry is treated as current source-ledger memory. If a historical idea is
still needed, it must be re-filed deliberately as a new swarm-manager item by
an agent or the operator after this cutover.
