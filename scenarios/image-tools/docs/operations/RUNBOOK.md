# Runbook — Image Tools

This document records operator procedures for running, diagnosing,
recovering, and maintaining the local-first image toolbox.

## Purpose Of This Document

Use this document to answer:

- How do I start, stop, and inspect the scenario?
- What checks should I run during an incident?
- How do I back up or restore state?
- Where should operational issues be recorded?

## Start / Stop / Status

Use lifecycle-managed commands from the scenario directory:

```bash
make setup     # build API/CLI/UI, install scenario CLI (run once / on dep changes)
make start     # start API + UI; opt-in resources stay down until a job needs them
make status    # running surfaces and their ports
make logs      # tail API + UI logs
make stop      # clean shutdown
make restart   # stop then start
make test      # run the scenario test lifecycle
```

Equivalently `vrooli scenario start image-tools` (and `stop`/`status`/
`logs`/`test`). Do **not** start API/UI binaries directly or exec the
standalone backends yourself — the lifecycle owns process naming,
ports, health checks, logs, and `/ready` gating for on-demand resources.

## Common Incidents

| Symptom | Check | Fix |
|---|---|---|
| Scenario does not start | `make status`, `make logs` | `make restart`; if stale build, `make setup` then start. |
| API unhealthy | `/health`, `/ready`, data dir writable, API logs | `make setup`, verify writable data dir, then `make restart`. |
| UI blank or stale | UI port, browser console, `ui/dist` freshness | `make setup` then `make restart`. |
| Model download fails / disk full | `image-tools model list`, free disk on the storage volume, API logs | Free disk or change storage location; re-run `image-tools model install <id>` (downloads are checksummed and resumable/opt-in). |
| GPU not detected | `vrooli host inventory --json` output (the `capabilities` seam / `internal/hostinventory` collector), API logs for fallback messages | Expected to fall back to the CPU-capable default with a visible warning; verify NVIDIA drivers if GPU is expected. Op still completes on CPU (slower). |
| Job queue backed up | `image-tools jobs list`, queue-depth metric, GPU contention | By design heavy GPU jobs are serialized; cheap CPU ops run concurrently. Let the queue drain or abort low-priority jobs; do not parallelize GPU jobs. |
| BYOK provider error / over-budget | API logs, BYOK provider config, pre-op cost estimate | Verify the API key/quota; the mandatory pre-op cost estimate should have surfaced cost — re-run with a local tier or top up the provider. |
| ComfyUI configured but down | `/ready` for the ComfyUI resource, API logs | No action required for correctness — ops automatically use the standalone backend; restart/disable ComfyUI if its `/ready` flaps. |
| Oversized / decompression-bomb upload rejected | API logs, the request's reported limits | Expected guard at the ingestion boundary. Resize/down-scale the source or raise the configured limit deliberately if legitimate. |

Record recurring failures in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md).

## Backup / Restore

Two data classes back this scenario; both are covered by the standard
Vrooli scenario backup substrate (data-backup-manager).

| Data | Backup Procedure | Restore Procedure |
|---|---|---|
| SQLite metadata (jobs, recipes, model registry state, measures) | Snapshot the scenario database via data-backup-manager scenario backup. | Restore the snapshot, then `make restart`. |
| Image binaries (api-core storage / blobstore) | Backed up as the scenario's storage namespace via data-backup-manager. | Restore the storage namespace, then `make restart`. |
| Opt-in model weights | Not backed up (reproducible via the model registry). | Re-install on demand with `image-tools model install <id>`. |

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff | `make test` |
| Inspect logs | as needed | `make logs` |
| Prune old job outputs | per retention policy | Remove expired job outputs from blobstore per configured retention; verify with `image-tools jobs list`. |
| Update / garbage-collect model weights | as models mature / when disk-pressured | `image-tools model list`, then `install`/`remove`/`enable`/`disable` to prune unused or stale weights. |
| Refresh BYOK provider rates | when provider pricing changes | Update the configured per-provider rate table so pre-op cost estimates stay accurate. |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |

## Escalation

If you spot a defect outside your current scope, file it via the
`report-bug` workflow to scenario-qa
(`prompt-manager skill read report-bug` → knowledge-add). Record known
operational issues in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md)
and append meaningful completed work to
[`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
