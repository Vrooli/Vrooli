# Runbook — Data Backup Manager

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
| Backup overdue / failing | `/health` (flags overdue/failed), last-success per target, run history | Inspect the failing run's logs; re-run the plan on demand; verify the destination is reachable | If a target has no recent success, treat as a recovery risk — escalate. |
| Destination unreachable / not writable | Destination dry-run, backend credentials in `vault`, network/path | Restore vault secret access; re-validate destination; pause dependent plans | Escalate before the next scheduled run if it cannot be fixed in time. |
| Storage cap reached (alert+block) | Per-destination usage vs cap | Add capacity, tighten retention policy, or add a destination — **never** delete backups manually to free space | Escalate; blocked writes mean new backups are not landing. |
| Verified-restore failing | Verify-mode run output, checksums, scratch restore logs | Treat the affected target as un-recoverable until verify passes; do **not** remove its committed copy from git | Escalate immediately — this is the gate protecting recoverability. |
| UI blank or stale | UI port, `ui/dist` freshness, server logs | `make setup` then `make restart` | Add troubleshooting entry if recurring. |
| CLI talks to old API | `data-backup-manager status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |

## Disaster Recovery — Backup / Restore / Verify

This is the scenario's reason to exist. The operating model has three
loops: **back up** registered targets on schedule or on demand,
**verify** that backups can restore, and **restore** when needed.

### Catalog orientation

The manager's domain is TARGET (a registered source) × DESTINATION (an
encrypted kopia repository) × PLAN (a many-to-many binding with schedule
and retention) → RUN → RESTORE. Scenarios self-register their targets;
the manager's SQLite catalog is a cache and run history, rebuilt from
re-registration on boot — it is not the source of truth.

```bash
data-backup-manager targets list        # registered backup targets
data-backup-manager destinations list   # configured kopia repositories
data-backup-manager plans list          # target-to-destination bindings
data-backup-manager runs list           # run history
data-backup-manager restores list       # restore and verify history
```

### Destination onboarding

Discovery and readiness analysis are read-only. A removable drive is not
used until an operator reviews it and creates a destination explicitly.
Prefer a dedicated repository subdirectory such as
`<mount>/vrooli-backups` instead of the mount root:

```bash
data-backup-manager discovery destinations
data-backup-manager destinations readiness --location <mount>
data-backup-manager destinations prepare-plan \
  --location <mount> \
  --action create-subdir \
  --subdir vrooli-backups
```

`prepare-execute` defaults to dry-run. Real execution currently supports
only the non-destructive `create_subdir` action, after confirmation and
device-identity revalidation. Formatting, relabeling, and clearing files
remain unsupported operationally; handle those outside this scenario until
the Linux drive-preparation adapter is implemented and validated.

### Back up (scheduled + on-demand)

Plans run on cadence via the in-process scheduler. Operators (and
scenarios) can trigger a run manually:

```bash
data-backup-manager runs trigger --plan <plan-id>
data-backup-manager runs list
```

Source capture is per kind: filesystem (direct), SQLite (`VACUUM INTO`),
Postgres (`pg_dump`), Redis (prefix `SCAN`+`DUMP`, best-effort —
non-transactional), Qdrant (snapshot API), object-storage (S3/MinIO
mirror). All artifacts land in an encrypted destination; none are
written under the scenario source tree.

### Verify (the recoverability gate)

Verified restore is first-class. Verify mode test-restores a target to a
scratch location and checksums the result, proving the backup is
actually recoverable:

```bash
data-backup-manager restores verify \
  --target <target-id> \
  --destination <destination-id> \
  --snapshot <snapshot-id>
```

**Rule:** committed runtime data is removed from git only after its
target has a passing verified restore. A target that cannot verify is
treated as un-recoverable until it does.

### Restore (privileged)

Restore rehydrates source data to an operator-chosen location. It is a
privileged operation (see `../internal/SECURITY.md`):

```bash
data-backup-manager restores restore \
  --target <target-id> \
  --destination <destination-id> \
  --snapshot <snapshot-id> \
  --location <path>
```

### Standalone restore (Vrooli down)

Because destinations are plain kopia repositories, recovery does not
depend on this scenario being up. With the repo passphrase from `vault`,
an operator can restore directly with the kopia CLI even if the manager,
API, or the whole Vrooli stack is unavailable. This is a deliberate
disaster-recovery property — keep the passphrase recoverable
independently of the data it protects.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| Registered source targets | Scheduled/on-demand plan run into an encrypted kopia destination | `restores restore` (whole target in v1); `restores verify` to prove recoverability | implemented; must be validated with a real destination/plan/run per installation |
| Manager catalog (SQLite) | Not separately backed up — reconstructable from scenario re-registration on boot | Rebuilt automatically as scenarios re-register | by design |

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
