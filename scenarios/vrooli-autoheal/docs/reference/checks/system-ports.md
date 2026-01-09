# Ephemeral Port Usage Check (system-ports)

Monitors ephemeral (dynamic) port usage to prevent connection failures from port exhaustion.

## Overview

| Property | Value |
|----------|-------|
| Check ID | `system-ports` |
| Category | System |
| Interval | 300 seconds (5 minutes) |
| Platforms | Linux |

## What It Monitors

This check measures:

- The ephemeral port range from `/proc/sys/net/ipv4/ip_local_port_range`
- Active connections using ports in that range from `/proc/net/tcp` and `/proc/net/tcp6`
- Usage percentage and top consumers

## Status Meanings

| Status | Meaning |
|--------|---------|
| **OK** | Port usage below 70% |
| **Warning** | Port usage between 70-85% |
| **Critical** | Port usage above 85% |

## Why It Matters

Ephemeral port exhaustion causes:

- `cannot assign requested address` errors
- Failed outbound connections
- API call failures
- Database connection failures
- Service-to-service communication breakdowns

This is particularly problematic for:
- High-traffic services making many outbound connections
- Services with connection pooling issues
- Load balancers and proxies

## Understanding Ephemeral Ports

```
Default Linux Range: 32768-60999 (28,232 ports)

When you make an outbound connection:
1. Kernel assigns an ephemeral port from this range
2. Connection is identified by (src_ip:src_port, dst_ip:dst_port)
3. Port is held until connection closes + TIME_WAIT expires
4. TIME_WAIT typically lasts 60 seconds
```

## Common Failure Causes

### 1. Connection Churn

Many short-lived connections exhaust ports due to TIME_WAIT:

```bash
# Check TIME_WAIT connections
ss -tan | grep TIME-WAIT | wc -l

# Check total connections per state
ss -tan | awk '{print $1}' | sort | uniq -c | sort -rn
```

### 2. Connection Pool Misconfiguration

```bash
# Check connections to common services
ss -tan | grep :5432 | wc -l  # PostgreSQL
ss -tan | grep :6379 | wc -l  # Redis
ss -tan | grep :443 | wc -l   # HTTPS
```

### 3. Connection Leaks

```bash
# Monitor connection count over time
watch -n 5 'ss -tan | wc -l'

# Check by process
ss -tanp | awk '{print $NF}' | sort | uniq -c | sort -rn | head -20
```

## Troubleshooting Steps

1. **Check current port usage**
   ```bash
   # Count connections in ephemeral range
   ss -tan | awk -F: '{print $2}' | awk '{print $1}' | sort -n | \
     awk -v min=32768 -v max=60999 '$1>=min && $1<=max' | wc -l
   ```

2. **Check the ephemeral port range**
   ```bash
   cat /proc/sys/net/ipv4/ip_local_port_range
   ```

3. **Find top port consumers**
   ```bash
   ss -tanp | grep -E ":(3276[8-9]|327[7-9][0-9]|32[8-9][0-9]{2}|3[3-5][0-9]{3}|5[0-9]{4}|60[0-9]{3})" | \
     awk '{print $NF}' | sort | uniq -c | sort -rn | head -10
   ```

4. **Check TIME_WAIT settings**
   ```bash
   cat /proc/sys/net/ipv4/tcp_tw_reuse
   cat /proc/sys/net/ipv4/tcp_fin_timeout
   ```

5. **Tune if needed (temporary)**
   ```bash
   # Expand port range
   sudo sysctl -w net.ipv4.ip_local_port_range="1024 65535"

   # Enable TIME_WAIT reuse
   sudo sysctl -w net.ipv4.tcp_tw_reuse=1

   # Reduce FIN timeout
   sudo sysctl -w net.ipv4.tcp_fin_timeout=30
   ```

## Configuration

The check accepts the following options:

| Option | Default | Description |
|--------|---------|-------------|
| `warningThreshold` | 70 | Warning at this percentage |
| `criticalThreshold` | 85 | Critical at this percentage |

## Details Returned

```json
{
  "portMin": 32768,
  "portMax": 60999,
  "totalPorts": 28232,
  "usedPorts": 5432,
  "usedPercent": 19,
  "warningThreshold": 70,
  "criticalThreshold": 85,
  "topConsumers": [
    {"localAddr": "00000000", "count": 2341},
    {"localAddr": "0100007F", "count": 1523}
  ]
}
```

Note: Local addresses are in hex format (network byte order).

## Health Score

The health score is calculated as `100 - usedPercent`.

## Related Checks

- **infra-network**: Network issues may correlate with port exhaustion
- **resource-postgres**: Database connections use ephemeral ports
- **resource-redis**: Cache connections use ephemeral ports

## Prevention Tips

1. **Use connection pooling** for database and HTTP clients
2. **Set appropriate pool sizes** based on expected load
3. **Enable keep-alive** for long-lived connections
4. **Monitor trends** to catch gradual increases
5. **Tune kernel parameters** for high-traffic servers

## Kernel Tuning for High Traffic

Add to `/etc/sysctl.conf`:

```bash
# Expand ephemeral port range
net.ipv4.ip_local_port_range = 1024 65535

# Allow TIME_WAIT socket reuse
net.ipv4.tcp_tw_reuse = 1

# Reduce FIN timeout
net.ipv4.tcp_fin_timeout = 30

# Increase connection tracking
net.netfilter.nf_conntrack_max = 1048576
```

---

*Back to [Check Catalog](../check-catalog.md)*
