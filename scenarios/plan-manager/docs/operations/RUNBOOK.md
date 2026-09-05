# Runbook — Plan Manager

This document records operator procedures for running, diagnosing,
recovering, and maintaining plan-manager.

## Purpose Of This Document

Use this document to answer:

- How do I start, stop, and check the scenario?
- What are the likely incidents, and how do I respond?
- How is plan data backed up and restored?
- What routine maintenance is expected, and when do I escalate?

## Start / Stop / Status

- Start: `make start` (preferred) or `vrooli scenario start plan-manager`.
- Stop: `make stop` or `vrooli scenario stop plan-manager`.
- Status: `make status` or the standard scenario status command.
- Logs: `make logs`.
- Never run the binaries directly — the lifecycle wrapper handles process
  naming, ports, and health checks.

## Common Incidents

This scenario is pre-implementation, so the list below is the anticipated
incident set and will be refined with real diagnostics once implemented:

- Scenario will not start: check `make logs` for port conflicts or a
  failed SQLite open at `~/.vrooli`.
- Health endpoint unhealthy: check logs; inspect stored records and rendered
  mirrors with `plan-manager plans ...` once the scenario is healthy again.
- A soft integration is down (code-facts, git-control-tower, test-genie /
  scenario-validation, prompt-manager, meta-optimization-manager,
  agent-manager): expect degraded features (missing code refs, no fresh
  baseline/diff, no validation results, no velocity sink), NOT a crash —
  these integrations degrade gracefully by design.
- Stale plans: staleness results indicate plan→code references may no
  longer match the tree; treat as a data-freshness signal, not an outage.

## Backup / Restore

- Plan data (plan + phase records, plan→code references, validation /
  staleness results, candidate findings, per-plan velocity) lives in the
  shared SQLite store under `~/.vrooli`, not in a scenario-private DB.
- Because the store is scenario-independent, plans persist there even when
  plan-manager is stopped. Plan lifecycle and inspection are owned by the
  `plan-manager` CLI/API/UI.
- Backup/restore uses Vrooli's standard `~/.vrooli` home-store backup
  mechanisms. A scenario-specific backup/restore procedure is deferred
  until the storage schema is implemented.

## Maintenance Tasks

- Keep plan→code references fresh: re-run staleness checks after large
  refactors so plans referencing moved/deleted paths are flagged.
- Review candidate findings: these are unvalidated by design and should be
  promoted or discarded by an operator/agent rather than left to
  accumulate.
- Routine upgrades follow the standard scenario release flow in
  [`DEPLOYMENT.md`](DEPLOYMENT.md).

## Escalation

- First response: an operator triaging plans/handoffs via the UI or
  `plan-manager` CLI.
- Escalate to the plan-manager owner/maintainer when start fails
  persistently, the shared `~/.vrooli` store appears corrupted, or plan
  data integrity is in doubt.
- Defects outside operational scope should be filed as bug reports per the
  standard Vrooli bug-reporting flow rather than hot-patched here.

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — tiers, packaging, rollback
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — signals, logs, and health
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system architecture
