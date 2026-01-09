# Disk Space Check (system-disk)

Monitors disk space usage across configured partitions to prevent service failures from disk exhaustion.

## Overview

| Property | Value |
|----------|-------|
| Check ID | `system-disk` |
| Category | System |
| Interval | 300 seconds (5 minutes) |
| Platforms | All |

## What It Monitors

This check uses the `statfs` system call to measure:

- Total disk space per partition
- Used disk space per partition
- Usage percentage

Default partitions monitored: `/`

## Status Meanings

| Status | Meaning |
|--------|---------|
| **OK** | All partitions below 80% usage |
| **Warning** | At least one partition between 80-90% usage |
| **Critical** | At least one partition above 90% usage |

## Why It Matters

Disk exhaustion causes:
- Database write failures and corruption
- Log file loss (unable to write)
- Container creation failures
- Application crashes
- System instability

Docker images, logs, and database files are common space consumers in Vrooli deployments.

## Common Failure Causes

### 1. Docker Images and Containers

```bash
# Check Docker disk usage
docker system df

# Clean unused images, containers, volumes
docker system prune -a --volumes

# Remove dangling images only
docker image prune
```

### 2. Log Files

```bash
# Find large log files
find /var/log -type f -size +100M -exec ls -lh {} \;

# Check journal disk usage
journalctl --disk-usage

# Vacuum old journal entries
sudo journalctl --vacuum-time=7d
```

### 3. Database Files

```bash
# Check PostgreSQL data directory
du -sh /var/lib/postgresql/

# For Vrooli's PostgreSQL resource
vrooli resource logs postgres | tail -20
```

### 4. Build Artifacts

```bash
# Find large directories
du -sh /* 2>/dev/null | sort -rh | head -10

# Check node_modules and build caches
find ~ -name "node_modules" -type d -exec du -sh {} \; 2>/dev/null
```

## Troubleshooting Steps

1. **Identify the full partition**
   ```bash
   df -h
   ```

2. **Find largest directories**
   ```bash
   du -sh /* 2>/dev/null | sort -rh | head -10
   ```

3. **Check for deleted but open files**
   ```bash
   lsof +L1 | grep deleted
   ```

4. **Clean package manager caches**
   ```bash
   # APT
   sudo apt clean

   # npm
   npm cache clean --force

   # pnpm
   pnpm store prune
   ```

5. **Review old log files**
   ```bash
   sudo find /var/log -name "*.gz" -mtime +30 -delete
   ```

## Configuration

The check accepts the following options:

| Option | Default | Description |
|--------|---------|-------------|
| `partitions` | `["/"]` | List of mount points to monitor |
| `warningThreshold` | 80 | Warning at this percentage |
| `criticalThreshold` | 90 | Critical at this percentage |

## Details Returned

```json
{
  "partitions": [
    {
      "partition": "/",
      "usedPercent": 45,
      "usedBytes": 48318382080,
      "totalBytes": 107374182400,
      "freeBytes": 59055800320,
      "status": "ok"
    }
  ],
  "warningThreshold": 80,
  "criticalThreshold": 90
}
```

## Health Score

The health score is calculated as `100 - worstPartitionUsagePercent`. For example:
- 45% disk usage = 55 health score
- 85% disk usage = 15 health score

## Related Checks

- **system-inode**: Inode exhaustion can occur even with free disk space
- **infra-docker**: Docker issues can be caused by disk pressure

## Auto-Heal Actions

When this check fails, consider:
1. Running `docker system prune` automatically
2. Rotating old log files
3. Alerting administrators (disk cleanup often requires human judgment)

---

*Back to [Check Catalog](../check-catalog.md)*
