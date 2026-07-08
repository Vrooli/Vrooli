# System Monitor Operations Runbook

## Metrics Lifecycle: Retention & Compaction

The `metrics` table dominates `system-monitor.db` size. Left unbounded, it grows
~16–17 MB of JSON payload per full collection day. Retention prunes stale rows;
compaction reclaims the freed file space.

System-monitor cleanup is limited to its own metrics lifecycle. Broad host
disk-pressure remediation belongs to cleanup-manager: system-monitor observes
pressure and attribution, then operators preview and apply reclaim candidates
through cleanup-manager policy and audit.

### How it runs automatically

- A settings-driven scheduler prunes metrics older than `metrics_retention_days`.
- It runs once at startup when `retention_run_on_startup` is `true`, then every
  `retention_check_interval_seconds`.
- When `compact_after_retention` is `true`, each scheduled prune is followed by a
  `VACUUM`.
- Settings changes take effect on the next cycle — no restart required.
- See [reference/configuration.md](../reference/configuration.md#metrics-retention--compaction).

### Safe manual maintenance workflow

Run these from any host with the `system-monitor` CLI pointed at the API:

1. **Preview retention** — read-only; confirms how many rows/bytes and what time
   range would be removed.
   ```bash
   system-monitor maintenance retention preview --days 30
   ```
2. **Apply retention** — destructive; requires `--confirm`.
   ```bash
   system-monitor maintenance retention apply --days 30 --confirm
   ```
3. **Preview compaction** — read-only; shows reclaimable bytes (freelist pages).
   ```bash
   system-monitor maintenance compact preview
   ```
4. **Pause collection if desired.** Compaction serializes against metric writes,
   so it is safe to run live, but pausing reduces lock contention on a busy DB:
   ```bash
   system-monitor settings maintenance --state active   # suppress alerting noise
   system-monitor settings update --active false        # stop collection
   ```
5. **Apply compaction** — destructive (rewrites the file); requires `--confirm`.
   ```bash
   system-monitor maintenance compact apply --confirm
   ```
6. **Restart / resume and verify health.**
   ```bash
   system-monitor settings update --active true
   curl -s http://localhost:<API_PORT>/health
   ```

### Increasing retention before pruning

Retention defaults to 30 days. If you need to keep more forensic history, raise
the window *before* applying retention:

```bash
system-monitor settings update --metric-interval 10   # example settings edit
# raise metrics_retention_days via the Settings API / settings file, then:
system-monitor maintenance retention preview --days 90
```

## Backup Posture

Treat the metrics history as tiered when deciding what to back up:

- **Recent metrics (within the retention window):** useful and worth backing up.
  They power `/api/v1/metrics/current` and `/api/v1/metrics/timeline` and recent
  investigations.
- **Stale, high-frequency metrics (older than the retention window):** low value
  and effectively reconstructable noise. Prune and compact *before* backup so
  `system-monitor.db` backup payload reflects meaningful data, not abandoned
  history.

Recommended pre-backup sequence: preview retention → apply retention → compact →
back up. This keeps Data Backup Manager from treating gigabytes of stale metrics
as meaningful payload.

## Disk Pressure Handoff

When disk usage crosses investigation or alert thresholds, use system-monitor
to identify pressure and likely owners, then hand off remediation:

```bash
system-monitor metrics process-timeline --window 5m --top 20 --json
cleanup-manager cleanup plan --json
```

Apply only an approved cleanup-manager plan with an idempotency key. Do not
add broad deletion, Docker prune, journal vacuum, package-cache cleanup, or
scenario-private cleanup paths to system-monitor; those are cleanup-manager
provider responsibilities, with private data delegated to owner scenarios.
