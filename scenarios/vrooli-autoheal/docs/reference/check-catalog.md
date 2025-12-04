# Check Catalog

Complete reference of all available health checks.

## Infrastructure Checks

### infra-network

**Purpose:** Verify basic network connectivity.

| Property | Value |
|----------|-------|
| ID | `infra-network` |
| Category | Infrastructure |
| Interval | 60 seconds |
| Platforms | All |
| Importance | Critical |

**What it checks:**
- TCP connection to 8.8.8.8:53 (Google DNS)
- Connection timeout: 5 seconds

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | Connection successful |
| Critical | Connection failed or timeout |

**Troubleshooting:**
- Check firewall rules for outbound TCP on port 53
- Verify network interface is up
- Test manually: `nc -zv 8.8.8.8 53`

---

### infra-dns

**Purpose:** Verify DNS resolution is working.

| Property | Value |
|----------|-------|
| ID | `infra-dns` |
| Category | Infrastructure |
| Interval | 60 seconds |
| Platforms | All |
| Importance | Critical |

**What it checks:**
- DNS lookup for google.com
- Resolution timeout: 5 seconds

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | Resolution successful |
| Critical | Resolution failed |

**Troubleshooting:**
- Check /etc/resolv.conf (Linux)
- Verify DNS server is reachable
- Test manually: `nslookup google.com`

---

### infra-docker

**Purpose:** Verify Docker daemon is healthy.

| Property | Value |
|----------|-------|
| ID | `infra-docker` |
| Category | Infrastructure |
| Interval | 120 seconds |
| Platforms | All (if Docker available) |
| Importance | High |

**What it checks:**
- Runs `docker info` command
- Checks command exit code

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | Docker info succeeds |
| Warning | Docker not installed |
| Critical | Docker installed but unresponsive |

**Troubleshooting:**
- Check Docker service: `systemctl status docker`
- Check Docker socket permissions
- Test manually: `docker info`

---

### infra-cloudflared

**Purpose:** Verify Cloudflare tunnel service is running.

| Property | Value |
|----------|-------|
| ID | `infra-cloudflared` |
| Category | Infrastructure |
| Interval | 120 seconds |
| Platforms | All (if cloudflared available) |
| Importance | High |

**What it checks:**
- Cloudflared process is running
- Service status via systemctl (Linux)

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | Cloudflared running |
| Warning | Cloudflared not installed |
| Critical | Cloudflared installed but not running |

**Troubleshooting:**
- Check service: `systemctl status cloudflared`
- Check tunnel config in ~/.cloudflared/
- View logs: `journalctl -u cloudflared`

---

### infra-rdp

**Purpose:** Verify remote desktop access is available.

| Property | Value |
|----------|-------|
| ID | `infra-rdp` |
| Category | Infrastructure |
| Interval | 300 seconds |
| Platforms | Linux (xrdp), Windows (TermService) |
| Importance | Medium |

**What it checks:**
- Linux: xrdp service status
- Windows: TermService status

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | RDP service running |
| Warning | RDP not installed/configured |
| Critical | RDP installed but not running |

**Troubleshooting:**
- Linux: `systemctl status xrdp`
- Windows: `Get-Service TermService`
- Check port 3389 is open

---

## Vrooli Resource Checks

### resource-postgres

**Purpose:** Verify PostgreSQL resource is healthy.

| Property | Value |
|----------|-------|
| ID | `resource-postgres` |
| Category | Vrooli Resources |
| Interval | 60 seconds |
| Platforms | All |
| Importance | Critical |

**What it checks:**
- Runs `vrooli resource status postgres`
- Parses status output

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | Resource running |
| Warning | Resource starting/stopping |
| Critical | Resource stopped/failed |

**Troubleshooting:**
- Start resource: `vrooli resource start postgres`
- Check logs: `vrooli resource logs postgres`
- Verify container: `docker ps | grep postgres`

---

### resource-redis

**Purpose:** Verify Redis resource is healthy.

| Property | Value |
|----------|-------|
| ID | `resource-redis` |
| Category | Vrooli Resources |
| Interval | 60 seconds |
| Platforms | All |
| Importance | High |

**What it checks:**
- Runs `vrooli resource status redis`
- Parses status output

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | Resource running |
| Warning | Resource starting/stopping |
| Critical | Resource stopped/failed |

---

## System Checks

### system-disk

**Purpose:** Monitor disk space usage.

| Property | Value |
|----------|-------|
| ID | `system-disk` |
| Category | System |
| Interval | 300 seconds |
| Platforms | All |
| Importance | High |

**What it checks:**
- Disk usage percentage for root filesystem
- Inode usage percentage

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | Usage < 80% |
| Warning | Usage 80-90% |
| Critical | Usage > 90% |

**Troubleshooting:**
- Check usage: `df -h`
- Find large files: `du -sh /* | sort -rh | head`
- Clean Docker: `docker system prune`

---

### system-swap

**Purpose:** Monitor swap usage.

| Property | Value |
|----------|-------|
| ID | `system-swap` |
| Category | System |
| Interval | 300 seconds |
| Platforms | Linux, macOS |
| Importance | Medium |

**What it checks:**
- Swap usage percentage
- Swap availability

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | Swap usage < 50% |
| Warning | Swap usage 50-80% |
| Critical | Swap usage > 80% |

---

### system-zombies

**Purpose:** Detect zombie processes.

| Property | Value |
|----------|-------|
| ID | `system-zombies` |
| Category | System |
| Interval | 300 seconds |
| Platforms | Linux, macOS |
| Importance | Low |

**What it checks:**
- Count of zombie (defunct) processes

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | No zombies |
| Warning | 1-10 zombies |
| Critical | > 10 zombies |

**Troubleshooting:**
- Find zombies: `ps aux | grep defunct`
- Kill parent process to clean up

---

## Watchdog Check

### watchdog-status

**Purpose:** Verify OS watchdog is protecting autoheal.

| Property | Value |
|----------|-------|
| ID | `watchdog-status` |
| Category | Meta |
| Interval | 300 seconds |
| Platforms | All |
| Importance | Medium |

**What it checks:**
- Watchdog service installed
- Watchdog service running

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | Watchdog active |
| Warning | Watchdog not installed |
| Critical | Watchdog installed but not running |

---

## Check Intervals Summary

| Interval | Checks | Use Case |
|----------|--------|----------|
| 60s | Network, DNS, Resources | Critical services, fast detection |
| 120s | Docker, Cloudflared | Services that recover slowly |
| 300s | Disk, Swap, Zombies, RDP, Watchdog | Slow-changing metrics |

## Adding Custom Checks

See the [Adding Health Checks](../guides/adding-checks.md) guide for instructions on creating your own checks.
