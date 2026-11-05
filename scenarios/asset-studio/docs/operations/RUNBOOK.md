# Runbook — Asset Studio

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
| CLI talks to old API | `asset-studio status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |
| Renders stuck submitted | `vrooli scenario status ai-gateway`, render job list | Wait — jobs queue rather than fail when the gateway is unreachable. **Do not reach for a vendor API directly**; there is no such path by design (D-008). | If the gateway is healthy and jobs still hang, the dispatch seam is at fault, not the gateway. |
| Unexpected generation spend | `render cost` by day and by spec | Identify the spec; a multi-frame or video spec is the usual cause. | No budget cap exists until `ASSET-P1-006`. Until then this is detected after the fact, which is a known gap. |
| Import reports zero new items | Whether the source actually changed; import key table | Expected behaviour — re-import is a no-op for unchanged content (D-010). | If a source visibly changed and still imports nothing, the hash normalisation is wrong. That is a defect. |
| Import aborts on an item | Import report naming the item and the failing field | Fix the source in the marketing catalogue by decision. | **Do not loosen the schema to make import quiet** — that defeats `ASSET-P0-003`. The catalogue has never been validated, so early failures are expected. |
| Release refused | The stated cause on the release control | Resolve it: judge the outstanding frame, empty `credential_claims`, add alt text, or set disclosure. | The gate failing closed is correct behaviour. If the cause is unclear, that is a UI defect worth filing. |
| Asset bytes missing but the record exists | Blob directory, disk space, `assets` table | Should be impossible — a blob write failure fails the job rather than recording a metadata row. | If it happens, it is a real defect in the render/store ordering. File it. |

## Backup / Restore

The generated template uses local SQLite state. Product scenarios must
define backup and restore procedures before production deployment.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| SQLite database | deferred | deferred | Define before deployment. Holds identities, provenance, and verdicts — the parts that cannot be regenerated. |
| Blob files (artifacts) | deferred | deferred | **Lower priority than the database.** A released artifact is re-renderable from its provenance (`ASSET-P1-010`) at the cost of a generation call; an identity version or a conformance verdict is not recoverable at any price. Back up the database first. |

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff | `make test` |
| Inspect logs | as needed | `make logs` |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
