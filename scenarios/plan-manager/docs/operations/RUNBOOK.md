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

### Automatic investigation triggers

Plan Manager owns trigger policy, execution/phase identity, eligibility, and
incident linkage. Agent Manager owns investigation admission, evidence,
diagnosis, and the durable result. The automatic path composes exactly one
`plan-manager.investigate` Program Runtime operation for an eligible incident;
it never reads an agent transcript, applies a recommendation, or treats prose
as evidence.

Inspect or operate the policy with:

```bash
plan-manager investigate policy
plan-manager investigate preview --file observation.json
plan-manager investigate record --file observation.json
plan-manager investigate occurrences --execution-id <execution-id> --json
plan-manager investigate incidents --execution-id <execution-id> --limit 50 --json
plan-manager investigate incident --fingerprint <fingerprint> --json
plan-manager investigate link --fingerprint <fingerprint> --investigation-id <id>
```

`occurrences` includes suppressed evaluations such as known owner waits;
`incidents` is a bounded newest-first history view; `incident` returns the
typed record and its eligible occurrence history for one fingerprint. The
incident state is durable and idempotent by occurrence/fingerprint. The
normal states are `eligible`, `dispatched`, `completed`, and
`dispatch_failed`; a pending nested Agent Manager diagnosis remains
`dispatched`, not completed. Repeating the same occurrence reuses the existing
incident, while a new execution/phase revision produces a new identity. The
automatic path does not auto-apply changes.

For recovery, inspect the linked Agent Manager investigation by ID and perform
one bounded wait:

```bash
agent-manager investigation get <investigation-id> --json
agent-manager investigation wait <investigation-id> --timeout-seconds 30 --json
```

If the trigger was recorded but dispatch was unavailable, retry the same
observation after the dependency is healthy. If an investigation was admitted
outside the automatic dispatcher, link it explicitly with
`plan-manager investigate link`; do not create a second investigation for the same durable
occurrence. Cancellation and failed nested work are terminal diagnostics, not
successful completion.

Dispatch claims are leased for ten minutes. If Plan Manager stops after it
claims an eligible incident but before it records the Program Runtime
acknowledgement, the next identical observation can recover the expired claim
and retry dispatch. A fresh claim is never recovered while its lease is still
active, and duplicate or reordered observations remain attached to the same
fingerprint/family instead of creating another investigation.

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
