# Agent-S2 TCP Connection Leak Issue

## Critical Issue Summary
**Status**: 🔴 CRITICAL - Causes system-wide instability  
**Discovered**: 2025-08-07  
**Impact**: RDP disconnections, system resource exhaustion, container OOM kills  
**Affected Component**: `/agent_s2/server/services/proxy_addon.py` (mitmdump)

## Problem Description

Agent-S2's mitmdump proxy process is leaking TCP connections at a massive scale, causing:
- **64,819+ simultaneous TCP connections** (normal: <1000)
- **2GB+ memory consumption** leading to OOM kills every ~3 minutes
- **System-wide network instability** affecting RDP and other services
- **2.67 million TCP resets** indicating connection handling failures

## Symptoms

### Primary Indicators
1. **Mass TCP Connection Accumulation**
   ```bash
   # Check current connections
   ss -s | grep estab
   # Shows: TCP: 69084 (estab 64819, ...)
   ```

2. **Repeated OOM Kills**
   ```
   Memory cgroup out of memory: Killed process 2476489 (mitmdump) 
   total-vm:3940536kB, anon-rss:1770880kB
   ```

3. **RDP Disconnection Errors**
   ```
   [ERROR] BIO_read returned system error 110: Connection timed out
   [ERROR] BIO_read returned system error 104: Connection reset by peer
   [ERROR] BIO_should_retry returned system error 32: Broken pipe
   ```

### Connection Distribution
- **34,652 connections** to 192.168.1.173 (internal)
- **23,799 connections** to 34.36.57.103 (external)
- Thousands more distributed across various IPs

## Root Cause Analysis

The proxy addon appears to be:
1. **Not properly closing connections** after request completion
2. **Accumulating connection state** in memory
3. **Possibly duplicating connections** for each intercepted request
4. **Lacking connection pooling or limits**

### Failure Cascade
```
1. Proxy intercepts request → Creates connection
2. Request completes → Connection not closed
3. Connections accumulate → Memory grows
4. Hits 2GB limit → OOM kill
5. Container restarts → Brief recovery
6. Cycle repeats every ~3 minutes
```

## Immediate Mitigation

### Stop the Bleeding
```bash
# Option 1: Stop agent-s2 temporarily
docker stop agent-s2

# Option 2: Increase memory limits
docker update agent-s2 --memory 8g --memory-swap 16g

# Option 3: Restart periodically (band-aid)
while true; do
  sleep 300
  docker restart agent-s2
done
```

### Clear Stuck Connections
```bash
# Reduce TCP timeout values
sudo sysctl -w net.ipv4.tcp_fin_timeout=30
sudo sysctl -w net.ipv4.tcp_tw_reuse=1
sudo sysctl -w net.ipv4.tcp_tw_recycle=1  # Use with caution

# Force connection cleanup
sudo ss -K state time-wait
```

## Monitoring Commands

```bash
# Real-time connection monitoring
watch -n 5 'ss -s | grep estab'

# Track connections by destination
watch -n 10 'ss -tnp | grep ESTAB | awk "{print \$5}" | cut -d: -f1 | sort | uniq -c | sort -rn | head -10'

# Monitor OOM events
dmesg -w | grep -E "oom-kill|mitmdump"

# Check container memory usage
docker stats agent-s2 --no-stream
```

## Permanent Fix Requirements

### Code Investigation Areas
1. **proxy_addon.py**
   - Connection lifecycle management
   - Request/response completion handlers
   - Connection pooling implementation
   - Resource cleanup on errors

2. **proxy_service.py**
   - Service initialization parameters
   - Connection limits configuration
   - Memory management settings

### Configuration Changes Needed
1. **Add connection limits**
   ```python
   # In proxy configuration
   max_connections = 100
   connection_timeout = 30
   idle_connection_timeout = 60
   ```

2. **Implement connection pooling**
   ```python
   from urllib3 import PoolManager
   pool = PoolManager(maxsize=50, block=True)
   ```

3. **Add resource cleanup**
   ```python
   def response(self, flow):
       # Process response
       ...
       # Ensure connection cleanup
       if hasattr(flow, 'server_conn'):
           flow.server_conn.close()
   ```

## Testing Protocol

After implementing fixes:

1. **Baseline Test**
   ```bash
   # Record initial state
   ss -s > /tmp/connections_before.txt
   docker stats agent-s2 --no-stream > /tmp/memory_before.txt
   ```

2. **Load Test**
   ```bash
   # Run typical workload for 30 minutes
   # Monitor connections every minute
   for i in {1..30}; do
     sleep 60
     echo "Minute $i: $(ss -s | grep estab)"
   done
   ```

3. **Success Criteria**
   - Established connections stay below 500
   - Memory usage remains under 1GB
   - No OOM kills in 24 hours
   - RDP remains stable

## Related Issues
- Container memory limits may be too restrictive
- Network buffer tuning needed for high-connection scenarios
- Consider implementing circuit breaker pattern

## References
- [mitmproxy Connection Handling](https://docs.mitmproxy.org/stable/concepts-connections/)
- [Python TCP Connection Management](https://docs.python.org/3/library/socket.html#socket.socket.close)
- [Docker Resource Constraints](https://docs.docker.com/config/containers/resource_constraints/)

## Update Log
- 2025-08-07: Initial discovery and documentation
- TODO: Add proxy_addon.py analysis results
- TODO: Test connection pooling implementation
- TODO: Validate fix effectiveness