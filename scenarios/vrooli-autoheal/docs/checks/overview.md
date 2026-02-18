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
- **RDP** - Remote desktop access (if enabled)
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
- **Browserless** - Headless Chrome

### Vrooli Checks
Internal Vrooli platform health:
- **Vrooli API** - Core API responsiveness
- **Scenarios** - Per-scenario health (dynamic, based on config)
- **Watchdog** - OS-level watchdog service status
- **Orphan Processes** - Leaked processes from stopped scenarios
- **Stale Locks** - Stale lock file detection

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
