# Runbook — Storage Manager

This document records operator procedures for running, diagnosing,
recovering, and maintaining the scenario.

## Purpose Of This Document

Use this document to answer:

- How do I start, stop, and inspect the scenario?
- What checks should I run during an incident?
- How do I back up or restore state?
- Where should operational issues be recorded?

## Start / Stop / Status

Use lifecycle-managed commands from the scenario directory:

```bash
make setup
make start
make status
make logs
make stop
make test
```

Do not start API/UI binaries directly. The lifecycle owns process
naming, ports, health checks, and logs.

## Common Incidents

| Symptom | Checks | Fix | Escalation |
|---|---|---|---|
| Scenario does not start | `make status`, `make logs` | `make restart`, then inspect lifecycle logs | Record recurring failures in `../internal/PROBLEMS.md`. |
| API unhealthy | `/health`, SQLite path, API logs | Run `make setup`, verify writable data dir | Check `INTEGRATIONS.md` for dependency expectations. |
| UI blank or stale | UI port, browser console, `ui/dist` freshness | `make setup` then `make restart` | Add troubleshooting entry if recurring. |
| CLI talks to old API | `storage-manager status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |

## Backup / Restore

The generated template uses local SQLite state. Product scenarios must
define backup and restore procedures before production deployment.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| SQLite database | deferred | deferred | Define before deployment. |
| Blob files | deferred | deferred | Define if binary/blob domains remain. |

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff | `make test` |
| Inspect logs | as needed | `make logs` |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |

## Storage accounting workflow

The operator console reads the same API-backed ledger as the CLI. Start with
`storage-manager inventory` and `storage-manager census`; inspect
`storage-manager history` and `storage-manager infra-health` before making a
policy decision. A census is read-only and persists an immutable snapshot. A
snapshot is only `closed` when measured bytes equal attributed bytes plus the
explicitly unattributed remainder and no unreadable paths remain.

Long-lived API processes can record scheduled observations by setting
`STORAGE_CENSUS_INTERVAL` to a duration of at least one minute (the default is
6 hours). The first scheduled observation waits one full interval so
readiness never triggers a surprise host scan. Scheduled observations never
apply cleanup or placement migrations.

The scheduler waits from the end of a pass and additionally caps census work
at `STORAGE_CENSUS_DUTY_CYCLE` of wall time (default `0.10`): a pass that takes
`d` is followed by at least `max(interval, 9d)` of idle time, and each overrun
is logged once with the idle gap chosen. A snapshot older than the interval
reports `staleness_verdict: stale`.

The walk is scoped by `config/storage-census-policy.json`. Roots default to
the attribution universe (the repository, `~/.vrooli`, and the XDG class roots
under `~/.local/share`, `~/.config`, `~/.cache`, `~/.local/state`); an empty
`roots` list means those defaults, never the whole device. The device is still
measured by statfs, and bytes outside the roots are reported as one remainder
at the device root, so the accounting identity holds. Exclusions take a `path`
or a `name` (a directory base name pruned wherever it appears; `node_modules`
by default) and always a `reason`. `scan_coverage.scanned_roots` and
`scan_coverage.pruned_directories` in the report say exactly what was walked.
An operator who wants a whole-device walk declares `$DEVICE_ROOT` as a root.

The owner inventory (the input to census, retention, and the operator read
models) is served from a five-minute cache (`STORAGE_INVENTORY_CACHE_TTL`);
`GET /api/v1/storage/inventory` with `Cache-Control: no-cache` forces a reload
after a manifest edit.

For the conversation-search Qdrant surface, run
`storage-manager storage qdrant --limit 100`. Treat active, leased, protected,
quarantined, and unreadable generations as non-reclaimable in storage-manager.
If a generation is eligible, use the authenticated agent-manager owner preview
and apply endpoints; never remove Qdrant files directly. A missing control
token or unavailable alias is an escalation condition, not a zero-byte result.

### Owner-specific retention checks

Use the owner operation named by the inventory before approving reclaim:

| Surface | Owner policy | Safe verification |
|---|---|---|
| Agent-manager Qdrant generations | Active alias and rollback generations stay protected; failed/expired generations require owner preview and lease re-check. | `storage-manager storage qdrant --limit 100`, then the authenticated agent-manager preview/apply flow. |
| Image-tools job outputs | Only successful terminal `out/` blobs are candidates; models, adapters, inputs, active/referenced, and pinned outputs stay protected. | `storage-manager cleanup providers --json`, then image-tools estimate/preview with recovery authorization. |
| Browser recordings/captures | Owner-managed 7-day age and 20 GiB/5 GiB byte budgets; active work and newest keep-count entries stay protected. | Run the BAS owner retention preview/fixture; storage-manager provider remains disabled by default. |
| System-monitor metrics | Retention is owner-local; prune precedes optional compaction and reports row/payload/reclaimed bytes. | `system-monitor maintenance retention preview --days 30`, then compact preview/apply with explicit confirmation. |
| Control-plane timings/logs | `~/.vrooli/metrics` and `~/.vrooli/logs` are contract-visible; active markers are protected. Timings rotate at 64 MiB/30 days with three backups. | Inspect storage-manager inventory/growth and the recorder or lifecycle log rotation receipt. |

Never approve a generic parent-directory sweep when a child is marked
`safe_with_owner`, `protected`, `unreadable`, or `aggregate_drift`.

Legacy `report_json` migration is not part of startup. If the narrow entry
sample model must be rebuilt, set `STORAGE_CENSUS_BACKFILL=1` for a maintenance
start; the migration waits 30 seconds after readiness and has a two-minute
context limit. Remove the setting after the maintenance run.

The static Test Genie `storage` phase remains a fast isolation/persistence
gate. It does not run this host census; use the storage-manager comprehensive
run for product acceptance and live API truthfulness.

## Disk is filling

1. Run `storage-manager storage growth --window 24h`.
2. Inspect the fastest positive-slope owner and its ceiling status.
3. Run `storage-manager cleanup plan --json`.
4. Apply only the approved safe tier, or use the owner approval token named by
   the plan for an owner-delegated provider.
5. Run `storage-manager cleanup audit --json` and record the reclaimed bytes.

For an owner cleanup, retain the preview identity, approval mode, provider,
planned bytes, skipped items, and post-apply receipt. A zero-byte result with a
blocked reason is an escalation signal; it is not proof that the target is
empty.

## A provider is blocked

1. Run `storage-manager cleanup plan --json`.
2. Read the provider's blocked reason.
3. If the reason is `owner scenario client unavailable`, verify local scenario
   discovery and restart the owner through its lifecycle.
4. If the reason is `owner scenario unreachable`, inspect the owner health endpoint.
5. If the reason is `owner scenario does not implement cleanup`, file or route the owner
   capability gap; do not delete its files by hand.

## A ceiling is not binding

1. Run `storage-manager storage validate <owner>`.
2. Read `CEILING_NOT_BINDING` and the measured bytes.
3. Replace a point-in-time ceiling with a workload-derived `max_bytes` or
   `max_age` value.
4. Keep `regenerable` explicit and run the owner validation again.

## Validation evidence

For focused owner work, run the affected package tests first, then the
server-owned scenario phases:

```bash
vrooli scenario test system-monitor --phases structure,contracts,unit --wait --json
vrooli scenario test image-tools --phases structure,contracts,unit --wait --json
```

Use the returned `test-genie runs wait --json <scenario> <run-id>` command only
once when a run was started without `--wait`. Save the terminal receipt and
owner fixture output under the plan's evidence directory. A failed broad phase
must remain visible with its classification and limitation; do not rewrite it
as a passing cleanup result.

## Retention troubleshooting

| Symptom | Check | Action |
|---|---|---|
| Preview times out | `storage-manager storage status`, census staleness, and provider latency | Use the bounded view, cached inventory, or owner-specific preview; report an unreadable/timeout disposition. |
| Provider is blocked | `storage-manager cleanup plan --json` and `cleanup providers --json` | Read the exact owner, lease, token, or approval reason. Restart the owner through lifecycle if unreachable; never bypass it with filesystem deletion. |
| Preview is stale | Plan revision, snapshot identity, active alias, leases, and current byte count | Re-run preview. Apply must reject changed identity or skip changed items. |
| Reclaim is partial | Receipt skipped IDs, warnings, provider errors, and post-apply `storage growth` | Keep the receipt, inspect protected/live items, and perform another bounded preview only after the cause is resolved. |
| Compaction is interrupted | System-monitor maintenance status and SQLite stats | Leave current rows intact, restart the owner, preview again, and compact only after prune and free-space checks. |
| Journald cap is absent | `vrooli host safeguard list --json` and host-hardening inspection | Apply the control-plane `host_hardening` safeguard in an approved maintenance window; scenarios do not edit `/etc/systemd`. |

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Investigation artifacts

The 2026-09-09 remediation evidence is host-local and must not be treated as
portable production state:

- [`/home/matthalloran8/.vrooli/plan-artifacts/storage-accountability-and-retention-remediation-qdrant/source-inventory.md`](/home/matthalloran8/.vrooli/plan-artifacts/storage-accountability-and-retention-remediation-qdrant/source-inventory.md)
- [`/home/matthalloran8/.vrooli/plan-artifacts/storage-accountability-and-retention-remediation-qdrant/disk-investigation-evidence.md`](/home/matthalloran8/.vrooli/plan-artifacts/storage-accountability-and-retention-remediation-qdrant/disk-investigation-evidence.md)
- [`/home/matthalloran8/.vrooli/plan-artifacts/storage-accountability-and-retention-remediation-qdrant/evidence-2026-09-09/`](/home/matthalloran8/.vrooli/plan-artifacts/storage-accountability-and-retention-remediation-qdrant/evidence-2026-09-09/)

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
