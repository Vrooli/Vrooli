# Health Check System Overview

The Vrooli Autoheal system monitors infrastructure, system resources, and application health to detect and recover from failures automatically.

## Check Categories

### Infrastructure Checks
Core connectivity and services required for the system to function:
- **Network** - Internet connectivity via TCP
- **DNS** - Domain name resolution
- **NTP** - Time synchronization
- **Docker** - Container runtime health
- **Cloudflared** - Tunnel connectivity to Cloudflare
- **Certificate** - TLS certificate expiry monitoring
- **Display** - Display manager health (if applicable)
- **RDP** - Remote desktop serviceability: can a client connect and authenticate (if enabled)
- **systemd-resolved** - DNS resolver service

### System Checks
Operating system resource monitoring:
- **Disk Space** - Storage availability
- **Inodes** - File system metadata limits
- **Swap** - Memory pressure indicators
- **Memory** - RAM usage monitoring
- **CPU Load** - System load averages
- **GPU** - GPU utilization (if available)
- **Zombies** - Defunct process detection
- **Ports** - Ephemeral port exhaustion
- **Claude Cache** - Claude Code cache disk usage

### Resource Checks
Vrooli-managed services:
- **PostgreSQL** - Database health
- **Redis** - Cache health
- **Ollama** - AI inference service
- **Qdrant** - Vector database
- **SearXNG** - Metasearch engine

### Vrooli Checks
Internal Vrooli platform health:
- **Vrooli API** - Core API responsiveness
- **Scenarios** - Per-scenario health from `vrooli scenario status --json`
- **Watchdog** - OS-level watchdog service status
- **Orphan Processes** - Core-maintained orphan state from `vrooli orphans --json`
- **Stale Locks** - Core-maintained lock state from `vrooli locks --json`

Autoheal does not maintain fallback orphan or stale-lock heuristics. Cleanup delegates to `vrooli cleanup orphans` and `vrooli cleanup locks`.

## Status Levels

| Status | Color | Meaning |
|--------|-------|---------|
| OK | Green | Healthy - no action needed |
| Warning | Amber | Degraded - attention recommended |
| Critical | Red | Failed - immediate action required |

## Check Intervals

Each check runs on its own schedule:
- **Fast** (30s): Network, DNS
- **Standard** (60s): Most service checks
- **Slow** (300s): System resource checks

The `--force` flag on tick commands overrides intervals and runs all checks immediately.

## Recovery Actions

Many checks support automatic or manual recovery actions:
- **Start/Stop/Restart** - Service lifecycle management
- **Reap Zombies** - Clean up defunct processes
- **Flush Cache** - Clear DNS resolver cache
- **View Logs** - Diagnose failures

See the Recovery Actions documentation for details.

## Durable action and outage evidence

Every attempted automatic or manual recovery is written to `action_logs` with
the check, action, success, timeout, duration, output, error, and timestamp.
`heal_trackers` remains the durable retry counter. The unused
`autoheal_actions` table was removed so there is only one action ledger.

Canonical scenario and resource checks also write interval evidence to
`outage_records`. The first explicit unavailable observation opens one row;
repeated observations leave that row open; the next explicit available
observation closes it. A degraded member that is still serving is available,
and an unreadable source does not fabricate either side of an interval.

The typed measures API and CLI expose both interval aggregates for any member:

```text
vrooli-autoheal measure outages --member-id resource-qdrant --window-hours 24
```

The response reports total unavailable seconds and the distinct outage count.
Open outages are measured through the end of the requested window without
being closed by the query.

## Bounded recovery and suspension

Automatic recovery stops after `global.maxRestartAttempts` consecutive real
failures (three by default). At the limit, autoheal writes an escalated
disposition to `heal_trackers`, opens a durable incident, and suspends further
actions for that check. Health observations continue, and `/api/v1/status`
plus `vrooli-autoheal check suspended` show the reason, lifetime counters, and
whether the target has never succeeded or previously succeeded.

After an operator fixes the cause, resume only that check:

```text
vrooli-autoheal check resume scenario-example
```

Resume clears the active failure streak and suspension. It preserves lifetime
attempt counters and the escalation disposition as historical evidence. A
subsequent successful recovery records the `healed` disposition.

`try_start` is not automatically demoted to `ignore`. A repeatedly failing
`try_start` member remains a warning in the supervision set while its recovery
actions are suspended. This stops futile retries without silently reducing
coverage; changing intent still requires changing the owning declaration.
