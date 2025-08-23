# Agent-S2 TCP Connection Leak - Fix Recommendations

## Executive Summary
The TCP connection leak in Agent-S2 is caused by mitmproxy running in transparent mode without proper connection management. The proxy intercepts ALL system traffic and doesn't properly close connections, leading to catastrophic resource exhaustion.

## Root Cause Analysis

### The Problem Chain
1. **Transparent Proxy Mode**: mitmproxy intercepts ALL TCP traffic on ports 80/443
2. **No Connection Limits**: mitmproxy has no max connection settings
3. **No Connection Pooling**: Each request creates new connections without reuse
4. **Memory Constraint Mismatch**: Container limited to 2GB but defaults specify 4GB
5. **No Connection Cleanup**: Connections remain in ESTABLISHED state indefinitely

### Evidence
- **Command**: `mitmdump -s proxy_addon.py --mode transparent --listen-port 8085`
- **Issue**: No `--set connection_strategy=lazy` or connection limits
- **Result**: 64,819+ simultaneous connections accumulating

## Immediate Fixes (Stop the Bleeding)

### 1. Emergency Mitigation (Do Now)
```bash
# Stop the container
docker stop agent-s2

# Or increase memory limit to match defaults
docker update agent-s2 --memory 4g --memory-swap 8g
```

### 2. Quick Configuration Fix
Edit `/scripts/resources/agents/agent-s2/docker/config/supervisor.conf`:

```ini
[program:security-proxy]
command=/home/agents2/.local/bin/mitmdump \
  -s /opt/agent-s2/agent_s2/server/services/proxy_addon.py \
  --mode transparent \
  --listen-port 8085 \
  --set confdir=/home/agents2/.mitmproxy \
  --set block_global=false \
  --set upstream_cert=false \
  --set connection_strategy=lazy \
  --set stream_large_bodies=50m \
  --set body_size_limit=50m \
  --set keep_host_header=true \
  --set tcp_hosts='^(.*)$'
```

### 3. Add Connection Management Script
Create `/scripts/resources/agents/agent-s2/docker/scripts/connection-monitor.sh`:

```bash
#!/bin/bash
# Connection monitoring and cleanup for Agent-S2

MAX_CONNECTIONS=500
CHECK_INTERVAL=60

while true; do
    # Count current connections
    CONN_COUNT=$(ss -tn state established | wc -l)
    
    if [ "$CONN_COUNT" -gt "$MAX_CONNECTIONS" ]; then
        echo "WARNING: Connection count ($CONN_COUNT) exceeds limit ($MAX_CONNECTIONS)"
        
        # Kill old TIME_WAIT connections
        ss -K state time-wait
        
        # If still too high, restart mitmdump
        if [ "$CONN_COUNT" -gt $((MAX_CONNECTIONS * 2)) ]; then
            echo "CRITICAL: Restarting mitmdump due to connection leak"
            supervisorctl restart security-proxy
        fi
    fi
    
    sleep "$CHECK_INTERVAL"
done
```

## Short-term Solutions (This Week)

### 1. Switch from Transparent to Regular Proxy Mode
Transparent mode is overkill for Agent-S2's use case. Switch to regular proxy:

```python
# In proxy_service.py
opts = options.Options(
    listen_port=self.port,
    mode=["regular"],  # Change from transparent
    ssl_insecure=True,
    confdir=os.path.expanduser("~/.mitmproxy"),
    # Add connection limits
    connection_strategy="lazy",
    stream_large_bodies="50m",
    body_size_limit="50m",
)
```

### 2. Implement Connection Pooling in proxy_addon.py
Add connection management to the proxy addon:

```python
import urllib3
from functools import lru_cache

class SecurityProxyAddon:
    def __init__(self):
        # ... existing init code ...
        
        # Add connection pooling
        self.pool_manager = urllib3.PoolManager(
            num_pools=10,
            maxsize=50,
            timeout=30.0,
            retries=urllib3.Retry(total=3, backoff_factor=0.1)
        )
        
    @lru_cache(maxsize=100)
    def get_connection_pool(self, host: str):
        """Get or create connection pool for host"""
        return self.pool_manager.connection_from_host(host)
        
    def done(self):
        """Cleanup when addon is unloaded"""
        self.pool_manager.clear()
```

### 3. Fix Memory Configuration
Update docker-compose or run command:

```yaml
# docker-compose.yml
services:
  agent-s2:
    mem_limit: 4g  # Match the defaults.sh setting
    memswap_limit: 8g
    mem_reservation: 1g
```

### 4. Add Systemd/Supervisor Monitoring
Add to supervisor.conf:

```ini
[program:connection-monitor]
command=/opt/agent-s2/connection-monitor.sh
user=agents2
autorestart=true
priority=376
stdout_logfile=/tmp/conn-monitor.log
stderr_logfile=/tmp/conn-monitor.err
```

## Long-term Solutions (Next Month)

### 1. Replace mitmproxy with Application-Level Filtering
Instead of system-wide transparent proxy, implement request filtering at the application level:

```python
# In browser automation code
from selenium.webdriver.common.proxy import Proxy, ProxyType

proxy = Proxy()
proxy.proxy_type = ProxyType.MANUAL
proxy.http_proxy = "localhost:8085"
proxy.ssl_proxy = "localhost:8085"

capabilities = webdriver.DesiredCapabilities.FIREFOX
proxy.add_to_capabilities(capabilities)
```

### 2. Implement Circuit Breaker Pattern
Add automatic fallback when proxy is overloaded:

```python
class ProxyCircuitBreaker:
    def __init__(self, failure_threshold=5, timeout=60):
        self.failure_count = 0
        self.failure_threshold = failure_threshold
        self.timeout = timeout
        self.last_failure = None
        self.state = "CLOSED"  # CLOSED, OPEN, HALF_OPEN
        
    def call(self, func, *args, **kwargs):
        if self.state == "OPEN":
            if time.time() - self.last_failure > self.timeout:
                self.state = "HALF_OPEN"
            else:
                raise ProxyUnavailable("Proxy circuit breaker is OPEN")
                
        try:
            result = func(*args, **kwargs)
            if self.state == "HALF_OPEN":
                self.state = "CLOSED"
                self.failure_count = 0
            return result
        except Exception as e:
            self.failure_count += 1
            self.last_failure = time.time()
            if self.failure_count >= self.failure_threshold:
                self.state = "OPEN"
            raise
```

### 3. Move to Container-Specific Proxy
Instead of system-wide proxy, use per-container proxy configuration:

```bash
# Only proxy Firefox traffic, not all system traffic
docker run --rm \
  -e HTTP_PROXY=http://proxy:8085 \
  -e HTTPS_PROXY=http://proxy:8085 \
  -e NO_PROXY=localhost,127.0.0.1 \
  firefox-container
```

## Testing Protocol

### 1. Connection Leak Test
```bash
# Before fix
docker exec agent-s2 ss -s | grep estab

# Run workload
for i in {1..100}; do
  curl -x localhost:8085 https://example.com &
done

# After 5 minutes
docker exec agent-s2 ss -s | grep estab
# Should be < 100 connections, not thousands
```

### 2. Memory Usage Test
```bash
# Monitor memory usage
docker stats agent-s2 --no-stream

# Should stay under 2GB with proper connection management
```

### 3. RDP Stability Test
```bash
# Monitor RDP disconnections
journalctl -f | grep "gnome-remote" &

# Run agent-s2 workload
# Should see no RDP disconnections
```

## Configuration Changes Summary

### 1. supervisor.conf
- Add `connection_strategy=lazy`
- Add connection monitoring program

### 2. defaults.sh
- Ensure AGENTS2_MEMORY_LIMIT matches docker config

### 3. proxy_service.py
- Switch from transparent to regular mode
- Add connection pooling

### 4. Docker Configuration
- Increase memory limit to 4GB
- Add memory swap limit of 8GB

## Monitoring Dashboard
Create monitoring script at `/scripts/resources/agents/agent-s2/monitor-health.sh`:

```bash
#!/bin/bash
echo "=== Agent-S2 Health Monitor ==="
echo "Time: $(date)"
echo ""
echo "Container Status:"
docker ps | grep agent-s2
echo ""
echo "Memory Usage:"
docker stats agent-s2 --no-stream --format "table {{.MemUsage}}\t{{.MemPerc}}"
echo ""
echo "Connection Count:"
docker exec agent-s2 ss -s | grep estab
echo ""
echo "Recent OOM Events:"
dmesg | grep -i "oom.*mitmdump" | tail -5
echo ""
echo "Proxy Process:"
docker exec agent-s2 ps aux | grep mitmdump | head -1
```

## Expected Outcomes
After implementing these fixes:
- ✅ TCP connections stay below 500
- ✅ Memory usage stays under 2GB
- ✅ No OOM kills
- ✅ RDP connections remain stable
- ✅ System load returns to normal

## Priority Actions
1. **NOW**: Stop agent-s2 or increase memory limit
2. **TODAY**: Add connection_strategy=lazy to supervisor.conf
3. **THIS WEEK**: Implement connection monitoring
4. **NEXT SPRINT**: Switch from transparent to regular proxy mode
5. **FUTURE**: Move to application-level filtering

## References
- [mitmproxy Connection Strategy](https://docs.mitmproxy.org/stable/concepts-options/#connection_strategy)
- [mitmproxy Memory Management](https://github.com/mitmproxy/mitmproxy/issues/2410)
- [TCP Connection Limits in Linux](https://www.kernel.org/doc/Documentation/networking/ip-sysctl.txt)
- [Docker Memory Management](https://docs.docker.com/config/containers/resource_constraints/)