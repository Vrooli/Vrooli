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
- Swap paging traffic (pages per second)

## Level and rate matrix

Swap level is a stock; paging traffic is the flow used for severity. A high
level with near-zero traffic is healthy because cold pages are parked. High
traffic is critical at either level; lower active traffic is a warning. If the
rate source is unavailable the check warns with the reason instead of
escalating from level alone. The named thresholds are 1 page/sec for
near-zero and 128 pages/sec for high traffic, based on the incident host's
observed steady-state paging rates.

## Status Meanings

| Status | Meaning |
|--------|---------|
| **OK** | Swap level is below 50% with quiet traffic, or a high level has near-zero traffic |
| **Warning** | Paging traffic is at least 1 and below 128 pages/sec, or its rate source is unavailable/invalid |
| **Critical** | Paging traffic is at least 128 pages/sec |

## Why It Matters

Swap occupancy records pages parked on disk, including cold pages left from earlier pressure. Current paging traffic, available RAM and memory PSI distinguish present contention from historical occupancy:

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

3. **Inspect maintained host findings and owner-safe recovery**
   ```bash
   vrooli-autoheal check get system-host-pressure
   vrooli-autoheal check actions system-host-pressure
   vrooli-autoheal check get system-containment-occupancy
   ```

   Inspect exact workload ownership before disposing of an abandoned session.
   Resource recycling remains subject to the control plane's declared ownership,
   serving-state and pressure brakes. A swap/RSS ratio alone does not establish
   that an active process is disposable. Do not clear swap, drop caches or restart
   shared services merely to reduce this gauge.

5. **Check OOM killer logs**
   ```bash
   dmesg | grep -i "out of memory"
   journalctl -k | grep -i "oom"
   ```

## Configuration

The check accepts the following options:

| Option | Default | Description |
|--------|---------|-------------|
| `warningThreshold` | 50 | Legacy occupancy diagnostic/ProcReader fixture threshold; production severity uses paging |
| `criticalThreshold` | 80 | Legacy occupancy diagnostic/ProcReader fixture threshold; production severity uses paging |

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
  ,"swapTrafficPagesPerSecond": 0
  ,"swapTrafficRateAvailable": true
  ,"nearZeroRateThreshold": 1
  ,"highRateThreshold": 128
}
```

## Health Score

The production health score is `100 - 100 * min(pagingRate / 128, 1)`, rounded down in the subtraction. Quiet paging therefore scores 100 even when cold swap pages remain occupied. Occupancy remains in the returned details. Legacy ProcReader-only fixtures retain their occupancy assertions; missing production rate evidence warns.

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
