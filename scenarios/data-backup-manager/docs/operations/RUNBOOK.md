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

### Coverage — register default protection

Discovery only *suggests* durable state worth protecting; a plan can only back
up *registered* targets. On a fresh install that gap lets an operator build a
plan that looks protective while it covers a single self-registered target.
The coverage surface closes it: it reports what is registered vs
recommended-but-unregistered vs sensitive, and bulk-registers the recommended
non-sensitive defaults in one action.

```bash
data-backup-manager coverage report                       # registered/recommended/sensitive/planned/verified
data-backup-manager coverage accept-defaults --dry-run    # preview — registers nothing
data-backup-manager coverage accept-defaults              # register non-sensitive discovered durable targets
```

Sensitive suggestions (credential/token files such as `codex/auth`,
`claude-code/credentials`) are **never** registered by default. Register them
deliberately only after review:

```bash
data-backup-manager coverage accept-defaults --include-sensitive
```

Coverage reads no file contents and registers locators only. Because plan
creation is guarded (see below), running `coverage accept-defaults` before
`plans create` is the normal first-backup order.

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

Note: `prepare-plan --json` wraps the plan as `{"plan": {…}}`, but
`prepare-execute --plan-json` expects the **inner** plan object. Extract it and
pass the exact `confirmation_phrase`:

```bash
PLAN=$(data-backup-manager destinations prepare-plan \
  --location <mount> --action create-subdir --subdir vrooli-backups \
  --json | jq -c '.plan')
data-backup-manager destinations prepare-execute \
  --plan-json "$PLAN" \
  --confirm "$(jq -r '.confirmation_phrase' <<<"$PLAN")" \
  --dry-run false
```

After review, create the destination at the bundle root (a slug-safe name):

```bash
data-backup-manager destinations create \
  --name elements-local --backend filesystem \
  --location <mount>/vrooli-backups
```

This materializes a self-describing bundle at `<mount>/vrooli-backups`:
`README.txt`, `RECOVERY.txt`, `vrooli-backup-destination.json`, and the vanilla
kopia repository under `repositories/elements-local.kopia`. Open `README.txt` on
the drive to confirm it explains itself; the `repository_location` reported by
`destinations get` is the path to hand to plain kopia for standalone recovery.

#### First real backup (data-safe order)

1. Run the disposable proof first (see "Gated backup proof" in
   [cli-commands](../reference/cli-commands.md)) under a `/tmp` root to confirm
   create → run → verify → restore → `diff` is clean and the bundle metadata
   contains no secret value.
2. Register default coverage so the plan protects all known non-regenerable
   durable state, not just one self-registered target:
   `coverage report` → `coverage accept-defaults --dry-run` →
   `coverage accept-defaults`. Plan creation is blocked while non-sensitive
   recommendations remain unregistered (override with
   `plans create --allow-incomplete-coverage` only on purpose).
3. Only then create the real external-drive destination and run a plan. Verify
   (`restores verify`) before trusting it; verify is non-destructive.

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

For a filesystem destination, the drive itself explains how: read
`RECOVERY.txt` at the bundle root, and connect plain kopia to the **nested**
repository path (`repository_location` /
`<bundle-root>/repositories/<slug>.kopia` — recorded in
`vrooli-backup-destination.json`), **not** the bundle root. The manifest also
records the vault `secret_ref` (a reference path, never the passphrase value).

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
