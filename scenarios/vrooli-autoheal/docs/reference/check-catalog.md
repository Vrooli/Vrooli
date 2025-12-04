# Check Catalog

Complete reference of all available health checks.

## Infrastructure Checks

### infra-network

**Purpose:** Verify basic network connectivity.

| Property | Value |
|----------|-------|
| ID | `infra-network` |
| Category | Infrastructure |
| Interval | 30 seconds |
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

[Detailed documentation](checks/infra-network.md)

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

[Detailed documentation](checks/infra-dns.md)

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

[Detailed documentation](checks/infra-docker.md)

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

[Detailed documentation](checks/infra-cloudflared.md)

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

[Detailed documentation](checks/infra-rdp.md)

---

### vrooli-api

**Purpose:** Verify the main Vrooli API is healthy.

| Property | Value |
|----------|-------|
| ID | `vrooli-api` |
| Category | Infrastructure |
| Interval | 60 seconds |
| Platforms | All |
| Importance | Critical |

**What it checks:**
- HTTP GET to `http://127.0.0.1:8092/health`
- Parses JSON response for health status
- Measures response time

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | API responds with `ok` or `healthy` |
| Warning | API responds with `degraded` |
| Critical | API unreachable or returns error status |

**Troubleshooting:**
- Check if Vrooli is running: `pgrep -f vrooli`
- Test endpoint: `curl http://127.0.0.1:8092/health`
- Start Vrooli: `vrooli develop`

[Detailed documentation](checks/vrooli-api.md)

---

## Vrooli Resource Checks

### resource-postgres

**Purpose:** Verify PostgreSQL resource is healthy.

| Property | Value |
|----------|-------|
| ID | `resource-postgres` |
| Category | Resource |
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

[Detailed documentation](checks/resource-check.md)

---

### resource-redis

**Purpose:** Verify Redis resource is healthy.

| Property | Value |
|----------|-------|
| ID | `resource-redis` |
| Category | Resource |
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

### resource-ollama

**Purpose:** Verify Ollama AI resource is healthy.

| Property | Value |
|----------|-------|
| ID | `resource-ollama` |
| Category | Resource |
| Interval | 60 seconds |
| Platforms | All |
| Importance | High |

**What it checks:**
- Runs `vrooli resource status ollama`
- Parses status output

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | Resource running |
| Warning | Resource starting/stopping |
| Critical | Resource stopped/failed |

---

### resource-qdrant

**Purpose:** Verify Qdrant vector database resource is healthy.

| Property | Value |
|----------|-------|
| ID | `resource-qdrant` |
| Category | Resource |
| Interval | 60 seconds |
| Platforms | All |
| Importance | High |

**What it checks:**
- Runs `vrooli resource status qdrant`
- Parses status output

---

### resource-searxng

**Purpose:** Verify SearXNG metasearch engine resource is healthy.

| Property | Value |
|----------|-------|
| ID | `resource-searxng` |
| Category | Resource |
| Interval | 60 seconds |
| Platforms | All |
| Importance | Medium |

**What it checks:**
- Runs `vrooli resource status searxng`
- Parses status output

---

### resource-browserless

**Purpose:** Verify Browserless headless Chrome resource is healthy.

| Property | Value |
|----------|-------|
| ID | `resource-browserless` |
| Category | Resource |
| Interval | 60 seconds |
| Platforms | All |
| Importance | Medium |

**What it checks:**
- Runs `vrooli resource status browserless`
- Parses status output

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
- Disk usage percentage for configured partitions
- Default: `/` partition

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

[Detailed documentation](checks/system-disk.md)

---

### system-inode

**Purpose:** Monitor inode usage to prevent file creation failures.

| Property | Value |
|----------|-------|
| ID | `system-inode` |
| Category | System |
| Interval | 300 seconds |
| Platforms | Linux |
| Importance | Medium |

**What it checks:**
- Inode usage percentage for configured partitions
- Default: `/` partition

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | Inode usage < 80% |
| Warning | Inode usage 80-90% |
| Critical | Inode usage > 90% |

**Troubleshooting:**
- Check inode usage: `df -i`
- Find directories with many files: `find / -xdev -type d -exec sh -c 'echo "$(find "$0" -maxdepth 1 | wc -l) $0"' {} \; | sort -rn | head`

[Detailed documentation](checks/system-inode.md)

---

### system-swap

**Purpose:** Monitor swap usage as memory pressure indicator.

| Property | Value |
|----------|-------|
| ID | `system-swap` |
| Category | System |
| Interval | 300 seconds |
| Platforms | Linux |
| Importance | Medium |

**What it checks:**
- Swap usage percentage
- Swap availability

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | Swap usage < 50% |
| Warning | Swap usage 50-80% (or no swap configured) |
| Critical | Swap usage > 80% |

**Troubleshooting:**
- Check swap: `free -h`
- Find memory consumers: `ps aux --sort=-%mem | head`

[Detailed documentation](checks/system-swap.md)

---

### system-zombies

**Purpose:** Detect zombie processes indicating resource leaks.

| Property | Value |
|----------|-------|
| ID | `system-zombies` |
| Category | System |
| Interval | 300 seconds |
| Platforms | Linux |
| Importance | Low |

**What it checks:**
- Count of zombie (defunct) processes

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | Less than 5 zombies |
| Warning | 5-20 zombies |
| Critical | More than 20 zombies |

**Troubleshooting:**
- Find zombies: `ps aux | awk '$8=="Z"'`
- Kill parent process to clean up

[Detailed documentation](checks/system-zombies.md)

---

### system-ports

**Purpose:** Monitor ephemeral port usage to prevent connection failures.

| Property | Value |
|----------|-------|
| ID | `system-ports` |
| Category | System |
| Interval | 300 seconds |
| Platforms | Linux |
| Importance | Medium |

**What it checks:**
- Ephemeral port range from `/proc/sys/net/ipv4/ip_local_port_range`
- Active connections using ports in that range

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | Port usage < 70% |
| Warning | Port usage 70-85% |
| Critical | Port usage > 85% |

**Troubleshooting:**
- Check port range: `cat /proc/sys/net/ipv4/ip_local_port_range`
- Count connections: `ss -tan | wc -l`
- Check TIME_WAIT: `ss -tan | grep TIME-WAIT | wc -l`

[Detailed documentation](checks/system-ports.md)

---

### system-claude-cache

**Purpose:** Monitor Claude Code cache to prevent file watcher exhaustion.

| Property | Value |
|----------|-------|
| ID | `system-claude-cache` |
| Category | System |
| Interval | 3600 seconds (1 hour) |
| Platforms | All |
| Importance | Medium |

**What it checks:**
- File count in ~/.claude/ directory
- Stale files eligible for cleanup

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | File count < 5,000 |
| Warning | File count 5,000-10,000 |
| Critical | File count > 10,000 |

**Recovery actions:**
- Cleanup Stale Files (safe)
- Full Cleanup (medium risk)
- Analyze Cache

**Troubleshooting:**
- Count files: `find ~/.claude -type f | wc -l`
- Check inotify limit: `cat /proc/sys/fs/inotify/max_user_watches`

[Detailed documentation](checks/system-claude-cache.md)

---

### infra-certificate

**Purpose:** Monitor SSL/TLS certificate expiration.

| Property | Value |
|----------|-------|
| ID | `infra-certificate` |
| Category | Infrastructure |
| Interval | 3600 seconds (1 hour) |
| Platforms | All |
| Importance | High |

**What it checks:**
- Certificate expiration dates in ~/.cloudflared/
- System certificate paths

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | All certs > 7 days until expiry |
| Warning | At least one cert expires within 7 days |
| Critical | At least one cert expires within 3 days |

**Troubleshooting:**
- Check cert: `openssl x509 -in ~/.cloudflared/cert.pem -noout -dates`
- Renew cloudflared: `cloudflared login`

[Detailed documentation](checks/infra-certificate.md)

---

### infra-display

**Purpose:** Monitor display manager and X11/Wayland on Linux.

| Property | Value |
|----------|-------|
| ID | `infra-display` |
| Category | Infrastructure |
| Interval | 300 seconds |
| Platforms | Linux |
| Importance | Medium |

**What it checks:**
- Display manager service (GDM, LightDM, SDDM)
- X11/Wayland responsiveness

**Status mapping:**
| Status | Condition |
|--------|-----------|
| OK | Display manager running, X11 responsive |
| Warning | Display manager running but X11 unresponsive |
| Critical | Display manager not running |

**Recovery actions:**
- Restart Display Manager (dangerous - disconnects sessions)
- Check Status
- View Logs

**Troubleshooting:**
- Check DM: `systemctl status gdm`
- Check X11: `xdpyinfo`

[Detailed documentation](checks/infra-display.md)

---

## Check Intervals Summary

| Interval | Checks | Use Case |
|----------|--------|----------|
| 30s | Network | Critical connectivity, fast detection |
| 60s | DNS, Vrooli API, Resources | Core services |
| 120s | Docker, Cloudflared | Services that recover slowly |
| 300s | Disk, Inode, Swap, Zombies, Ports, RDP | Slow-changing metrics |

## Check Categories Summary

| Category | Count | Description |
|----------|-------|-------------|
| Infrastructure | 9 | Network, DNS, Docker, Cloudflared, RDP, NTP, Resolved, Certificate, Display |
| Resource | 6 | PostgreSQL, Redis, Ollama, Qdrant, SearXNG, Browserless |
| System | 6 | Disk, Inode, Swap, Zombies, Ports, Claude Cache |
| **Total** | **21** | |

## Adding Custom Checks

See the [Adding Health Checks](../guides/adding-checks.md) guide for instructions on creating your own checks.
