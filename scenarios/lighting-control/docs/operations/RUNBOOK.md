# Runbook — Lighting Control

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
| CLI talks to old API | `lighting-control status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |
| Backend stops during control | Read backend status, then request root owner status | Root owner rebinds only the verified current OpenRGB incarnation; desired state reconciles after binding | Never reuse a prior port/SDK handle; the internal bind endpoint is process-local and authentication-gated. |

## Backup / Restore

Light Desk stores enrollment, zone intent, manual Off, policy configuration and
selection receipts in its scenario SQLite database. OpenRGB permission plans,
resource launch/update journals and root lighting intent have separate owners;
restoring this database does not restore those host records or hardware state.

Use [Data Backup Manager's recovery procedures](../../../data-backup-manager/docs/operations/RUNBOOK.md#disaster-recovery--backup--restore--verify)
for an explicitly requested backup or restore. Normal lighting setup does not
register a target, create a destination or schedule backups. No operator backup
or restore is claimed by this documentation.

### Identify the database

1. Use `vrooli scenario info lighting-control --json` to identify the intended
   live instance and its lifecycle-owned API process. For a shadow or another
   machine, inspect that exact instance; do not reuse a live path.
2. Resolve the database with the API's actual lifecycle environment and account.
   The maintained, read-only library call is
   `storage.SQLitePath(storage.SQLiteConfig{Scenario: "lighting-control"})`
   in [api-core/storage](../../../../packages/api-core/storage/sqlite.go).
   [API startup](../../api/main.go) opens that same scenario through
   `database.Open` with DriverSQLite. Use `SQLitePath` for diagnosis, not
   `SQLiteDSN`, which can create directories.
3. Check the resolved path against the API process's opened database file using
   the host's native process/file observation. Record the namespace and path as
   `DB_PATH` for the commands below. Do not derive a path from this document or
   another machine's home. The resolver accounts for the owning instance,
   `SCENARIO_DATA_DIR`, storage-root overrides and native platform directories.
   A diagnostic in an unrelated shell can resolve a different namespace; stop
   if the running-file observation and resolver disagree. Generic SQLITE_PATH
   and DATABASE_URL do not select this scenario's SQLite file.

### Consistent backup

Review existing target and destination metadata first:

```sh
data-backup-manager targets list --owner lighting-control --json
data-backup-manager destinations list --json
data-backup-manager plans list --json
```

When no matching target exists, explicitly register the resolved file:

```sh
data-backup-manager targets register --owner lighting-control \
  --name state --kind sqlite --locator "$DB_PATH" --json
```

Reuse the returned target ID and a reviewed, healthy encrypted destination.
Create a manual plan only if no suitable plan exists; plan creation retains the
manager's normal coverage checks. An omitted schedule creates no recurring job:

```sh
data-backup-manager plans create --name lighting-control-manual \
  --targets "$TARGET_ID" --destinations "$DESTINATION_ID" --json
data-backup-manager runs preflight --plan "$PLAN_ID" --json
data-backup-manager runs trigger --plan "$PLAN_ID" --json
data-backup-manager runs get "$RUN_ID" --json
```

Use IDs from the preceding owner responses. A queued run or caller timeout is
not completion. Reattach to the returned run and retain its terminal successful
outcome, target binding, snapshot ID, encryption identity and artifact evidence.
The manager's SQLite adapter uses VACUUM INTO for a consistent snapshot,
including when WAL writers are active. Do not copy a running database file or
its sidecars by hand. Keep recovery points and their keys outside the original
host failure domain according to the manager's procedure.

### Verified restore and safe reattachment

1. Select the exact owner/target/destination/snapshot and verify it through the
   manager before changing Light Desk state:

   ```sh
   data-backup-manager restores verify --target "$TARGET_ID" \
     --destination "$DESTINATION_ID" --snapshot "$SNAPSHOT_ID" --json
   data-backup-manager restores get "$VERIFY_ID" --json
   ```

   Wait for a terminal verified result with checksum evidence. This verifies
   the backup artifact; it does not prove Light Desk's logical state or LEDs.
2. Disable lighting with `vrooli runtime lighting disable --json`. Require
   confirmed writer/backend retirement; a pending cleanup error is not a safe
   restore boundary. Then use `make stop` from this scenario directory and
   confirm the lifecycle-owned API is stopped. Keep native lighting intent
   disabled throughout recovery so the supervisor cannot reattach the writer.
3. Retain a verified recovery point of the current state when it is readable.
   Prepare a clean owned target with no open database handles and no stale WAL
   or SHM files. Preserve the previous database and any sidecars together for
   rollback; do not mix an old WAL with the restored database or delete foreign
   files. If the target cannot be prepared without affecting unrelated data,
   stop and resolve ownership before proceeding.
4. Restore through the manager to that exact clean database file:

   ```sh
   data-backup-manager restores restore --target "$TARGET_ID" \
     --destination "$DESTINATION_ID" --snapshot "$SNAPSHOT_ID" \
     --location "$DB_PATH" --json
   data-backup-manager restores get "$RESTORE_ID" --json
   ```

   Require the terminal restored receipt, expected target path and artifact
   evidence. A transport acknowledgement alone is insufficient.
5. With lighting still disabled, use `make start` and the lifecycle wait once.
   API startup applies owned schemas and identity-source checks. Inspect health,
   enrolled stable controller identities, zone intent, policy and manual-Off
   state through the normal read API/CLI/UI. Check them against the intended
   recovery point. Treat stale policy selections or legacy/ambiguous controller
   identities as pending review; do not force enrollment or rewrite schema rows.
6. While the runtime remains disabled, use an authenticated write-capable
   principal to save durable Off without an expiry, then read it back:

   ```sh
   lighting-control control off --all-enrolled true --json
   lighting-control control desired --json
   ```

   Require the persisted all-enrolled Off override to be active before enabling
   a backend. If authentication or this checkpoint fails, keep lighting disabled.
   Re-enable only after the operator reviews the restored intent and current
   source/device permission qualification, using the existing Settings/runtime
   owner. Then qualify any intended static/effect output deliberately. Database
   restoration does not establish current device presence,
   firmware persistence or visible output. On schema/identity/health failure,
   keep lighting disabled and restore the retained known-good point through the
   same manager route.

Record target generation, run/recovery-point/restore IDs and logical checks in
the normal work record. This procedure describes the supported owners; it does
not claim that an operator recovery drill has been performed on this machine.

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
