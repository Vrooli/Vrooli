# System Monitor Operations Runbook

## Metrics Lifecycle: Retention & Compaction

The `metrics` table dominates `system-monitor.db` size. Left unbounded, it grows
~16–17 MB of JSON payload per full collection day. Retention prunes stale rows;
compaction reclaims the freed file space.

System-monitor cleanup is limited to its own metrics lifecycle. Broad host
disk-pressure remediation belongs to storage-manager: system-monitor observes
pressure and attribution, then operators preview and apply reclaim candidates
through storage-manager policy and audit.

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

## Disk Pressure: Detection and Escalation

> Rewritten after the 2026-07-31 incident, in which the host filled to 100
> percent while three healthy safeguards did nothing.

### One command to read the current state

```bash
curl -s http://localhost:16914/api/v1/disk-pressure | jq
```

It reports current usage, the active band, the evaluation interval, the most
recent threshold violation, the observation that caused the last band
transition, and the most recent remediation result — everything needed to
answer "is there pressure, and did anything act on it".

### The threshold loop

`ThresholdScheduler` evaluates disk usage every `threshold_check_interval`
seconds (default 20) whenever the monitor is active. It reads its settings
live on every tick, so changes take effect without a restart.

Usage is measured the way `df` measures it — `used / (used + available)`, not
`used / total`. The difference is the superuser reserve, which was 93 GB on the
incident host: reading free blocks reported 87 percent while `df` reported 93.

### Bands

Every boundary is settings-driven; none is hardcoded.

| Band | Default | Action |
| --- | --- | --- |
| normal | below `disk_threshold` | Record the sample. No alert. |
| warning | `disk_threshold` (80) | Persist a `ThresholdViolation`. No remediation. |
| high | `disk_high_percent` (90) | Ask storage-manager for a conservative preview. Nothing is deleted. |
| critical | `disk_critical_percent` (95) | storage-manager applies safe-tier providers with no operator present. |

`disk_threshold` *is* the warning boundary — there is deliberately not a
separate setting meaning the same thing.

### Hysteresis

Three mechanisms keep escalation deliberate rather than noisy:

- **Cooldown** (`disk_escalation_cooldown_seconds`, default 1800): at most one
  record per band per window. During the incident the disk sat above its
  threshold for days; a level-only rule would have produced thousands of
  identical records.
- **Debounce** (`disk_escalation_debounce_ticks`, default 2): a new band must
  be observed on consecutive ticks before it takes effect, so one noisy sample
  cannot escalate.
- **Fast-fill bypass** (`disk_fast_fill_jump_percent`, default 5): a rise of at
  least this many points in a single tick escalates immediately. The incident's
  own growth was 3-5 GB per day, but a runaway process can fill 100 GB in
  minutes, and waiting for a confirming tick is the wrong response to that.

De-escalation is **not** debounced and resets the cooldown: dropping below a
boundary is good news, and holding a stale high band would keep remediation
armed after the pressure is gone.

### Two independent paths to remediation

system-monitor and vrooli-autoheal each report pressure to storage-manager on
their own. This is deliberate — two safeguards routed through a single mediator
share a failure mode, which is what the incident exposed. storage-manager
deduplicates reports on partition and band, so duplicate concurrent reports
produce one execution and reclaimed bytes are never double-counted.

Only `safe`-tier providers run unattended. `safe_with_owner`, `conditional`,
and `forbidden` are withheld and reported in the response.

Host pressure is a third remediation path. `vrooli-watchdog --report-only`
reads CPU pressure, stranded memory, process/fork growth, and workload
ownership from the shared setpoint and emits evidence without deleting or
restarting anything. Reclaim is available only through the governed lifecycle
action, with the saturation and serving-request brakes applied; abandoned
workloads go to the conditional `undeclared-workload` provider for operator
approval.

### Host floor

Underneath both sits the emergency watchdog at
`~/.vrooli/libexec/vrooli-watchdog`, installed by the `emergency_watchdog`
safeguard (`vrooli setup`) and run every five minutes by the native user
scheduler. It is a self-contained binary and does not depend on the running
scenario or a Go toolchain at runtime. It watches available (not free) space, requests
`high`-band cleanup below its floor and `critical` below half the floor, bounds
its own log, and tolerates a failed write — during the 2026-07-31 incident it
died with `printf: write error: No space left on device`.

Since 2026-08-19 it also holds its unit restarts while the host is saturated
(CPU PSI `some.avg10` at or above its threshold): restarting into a machine that
cannot schedule adds load to a load problem. Thresholds are safeguard config,
not edits to the script, which is managed and overwritten by setup.

Host-level steps (tmpfiles override, filesystem reserve, journal bounds) are in
[docs/reference/environment-management.md](../../../../docs/reference/environment-management.md#host-disk-floor-operator-steps).

### Manual handoff

When investigating by hand, identify pressure and likely owners first:

```bash
system-monitor metrics processes --json
storage-manager cleanup plan --json
```

Apply only an approved storage-manager plan with an idempotency key. Do not
add broad deletion, Docker prune, journal vacuum, package-cache cleanup, or
scenario-private cleanup paths to system-monitor; those are storage-manager
provider responsibilities, with private data delegated to owner scenarios.

## Collection headroom

Each built-in collector has a declared duration and subprocess budget in
`api/internal/services/collector_cost.go`. The policy reserves at least half of
the configured metrics interval for persistence, attribution, and the host.
The health payload exposes `metrics.self.headroom_ok` and
`metrics.self.headroom_reason`; a false value means the monitor is consuming
more than its declared collection share and should be investigated before
shortening the interval. Raspberry Pi qualification must record this field
over a representative run; build success alone is not hardware evidence.

## Lifecycle stop

Use the control plane to stop the scenario so it can signal the recorded API
and UI PIDs safely:

```bash
vrooli scenario stop system-monitor
vrooli scenario status system-monitor
```

Do not use `pkill`, `killall`, or a broad command-line pattern. Process
ownership belongs to the control plane and the structure-health validator flags
unscoped stop commands.
