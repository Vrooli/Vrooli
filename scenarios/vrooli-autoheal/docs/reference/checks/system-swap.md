# Swap Usage Check (system-swap)

Monitors swap usage as an indicator of memory pressure that can cause severe performance degradation.

## Overview

| Property | Value |
|----------|-------|
| Check ID | `system-swap` |
| Category | System |
| Interval | 300 seconds (5 minutes) |
| Platforms | Linux |

## What It Monitors

This check reads `/proc/meminfo` to measure:

- Total swap space
- Used swap space
- Swap usage percentage

## Status Meanings

| Status | Meaning |
|--------|---------|
| **OK** | Swap usage below 50% |
| **Warning** | Swap usage between 50-80% (or no swap configured) |
| **Critical** | Swap usage above 80% |

## Why It Matters

High swap usage indicates the system is under memory pressure:

- **Performance Impact**: Swapping is orders of magnitude slower than RAM access
- **Latency Spikes**: Applications experience unpredictable delays
- **OOM Risk**: Continued memory pressure may trigger the OOM killer
- **Cascading Failures**: Slow responses can cause timeout cascades

For Vrooli deployments, memory pressure often affects:
- Database query performance
- AI model inference latency
- Container startup times

## Common Failure Causes

### 1. Memory-Intensive Applications

```bash
# Check memory usage by process
ps aux --sort=-%mem | head -20

# Check detailed memory info
free -h
cat /proc/meminfo | grep -E "MemTotal|MemFree|MemAvailable|SwapTotal|SwapFree"
```

### 2. Memory Leaks

```bash
# Watch memory over time
watch -n 5 'free -h'

# Check process memory growth
top -o %MEM
```

### 3. Insufficient RAM

```bash
# Check total vs available
free -h

# Check what's using memory
smem -tk
```

### 4. Too Many Containers

```bash
# Check Docker memory usage
docker stats --no-stream

# Check container limits
docker inspect --format '{{.HostConfig.Memory}}' $(docker ps -q)
```

## Troubleshooting Steps

1. **Check current memory state**
   ```bash
   free -h
   swapon --show
   ```

2. **Identify memory consumers**
   ```bash
   ps aux --sort=-%mem | head -10
   ```

3. **Check for memory leaks over time**
   ```bash
   # Install smem if needed
   sudo apt install smem

   # Check proportional set size
   smem -tk | tail -10
   ```

4. **Clear swap (if needed)**
   ```bash
   # Only do this if you have enough free RAM!
   sudo swapoff -a && sudo swapon -a
   ```

5. **Check OOM killer logs**
   ```bash
   dmesg | grep -i "out of memory"
   journalctl -k | grep -i "oom"
   ```

## Configuration

The check accepts the following options:

| Option | Default | Description |
|--------|---------|-------------|
| `warningThreshold` | 50 | Warning at this percentage |
| `criticalThreshold` | 80 | Critical at this percentage |

## Details Returned

```json
{
  "swapTotalKB": 2097148,
  "swapFreeKB": 1856412,
  "swapUsedKB": 240736,
  "swapTotalBytes": 2147479552,
  "swapFreeBytes": 1900966912,
  "swapUsedBytes": 246512640,
  "usedPercent": 11,
  "swapConfigured": true,
  "warningThreshold": 50,
  "criticalThreshold": 80
}
```

## Health Score

The health score is calculated as `100 - usedPercent`.

## No Swap Configured

If no swap is configured, the check returns a **warning** status with the message:
"No swap configured - system may OOM kill processes under memory pressure"

This is a warning because:
- Production systems should have some swap as a safety net
- The OOM killer may terminate important processes without warning

## Related Checks

- **system-disk**: Swap file/partition uses disk space
- **resource-postgres**: Databases are memory-intensive
- **resource-ollama**: AI models require significant memory

## Memory Optimization Tips

1. **Set container memory limits** to prevent runaway processes
2. **Use zram** for compressed swap (faster than disk)
3. **Tune vm.swappiness** for your workload (lower = prefer RAM)
4. **Monitor over time** to right-size your system

---

*Back to [Check Catalog](../check-catalog.md)*
