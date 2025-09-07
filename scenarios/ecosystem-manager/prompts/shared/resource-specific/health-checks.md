# Health Checks

## Purpose
Health checks are the heartbeat of resources. They ensure services are alive, responsive, and ready to serve. Without reliable health checks, the entire system becomes unpredictable.

## Health Check Requirements

### Mandatory for All Resources
Every resource MUST implement:
1. **Basic health endpoint** - Returns simple alive status
2. **Readiness check** - Confirms service is ready to accept requests
3. **Liveness check** - Confirms service is still running properly
4. **Startup probe** - Allows time for initialization

## Implementation Patterns

### Basic Health Check
```bash
# Minimal health check implementation
check_health() {
    curl -sf http://localhost:${PORT}/health >/dev/null 2>&1
}
```

### Robust Health Check
```bash
# Production-ready health check
check_health() {
    local timeout="${1:-5}"
    local retries="${2:-3}"
    local wait="${3:-1}"
    local endpoint="${4:-health}"
    
    echo "Checking health of ${RESOURCE_NAME}..."
    
    for i in $(seq 1 $retries); do
        if timeout $timeout curl -sf http://localhost:${PORT}/${endpoint} >/dev/null 2>&1; then
            echo "✅ ${RESOURCE_NAME} is healthy"
            return 0
        fi
        
        if [ $i -lt $retries ]; then
            echo "⏳ Attempt $i failed, waiting ${wait}s before retry..."
            sleep $wait
        fi
    done
    
    echo "❌ ${RESOURCE_NAME} health check failed after $retries attempts"
    return 1
}
```

### Advanced Health Check
```bash
# Comprehensive health check with details
check_health_detailed() {
    local response
    local http_code
    
    # Get response and HTTP code
    response=$(curl -sf -w '\n%{http_code}' http://localhost:${PORT}/health 2>/dev/null)
    http_code=$(echo "$response" | tail -1)
    body=$(echo "$response" | head -n -1)
    
    # Check HTTP code
    if [ "$http_code" != "200" ]; then
        echo "❌ Health check returned HTTP $http_code"
        return 1
    fi
    
    # Parse JSON response if available
    if command -v jq >/dev/null 2>&1 && echo "$body" | jq . >/dev/null 2>&1; then
        local status=$(echo "$body" | jq -r '.status // .healthy // "unknown"')
        local message=$(echo "$body" | jq -r '.message // ""')
        
        if [ "$status" = "healthy" ] || [ "$status" = "true" ]; then
            echo "✅ ${RESOURCE_NAME} is healthy: $message"
            return 0
        else
            echo "❌ ${RESOURCE_NAME} is unhealthy: $message"
            return 1
        fi
    fi
    
    # Simple response
    echo "✅ ${RESOURCE_NAME} responded to health check"
    return 0
}
```

## API Implementation

### Go Health Endpoint
```go
// Basic health endpoint
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "healthy": true,
        "timestamp": time.Now().Unix(),
    })
})

// Advanced health endpoint
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    health := checkSystemHealth()
    
    if !health.Healthy {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    
    json.NewEncoder(w).Encode(health)
})

type HealthStatus struct {
    Healthy     bool                   `json:"healthy"`
    Version     string                 `json:"version"`
    Timestamp   int64                  `json:"timestamp"`
    Checks      map[string]CheckResult `json:"checks"`
}

type CheckResult struct {
    Healthy bool   `json:"healthy"`
    Message string `json:"message,omitempty"`
    Latency int64  `json:"latency_ms,omitempty"`
}

func checkSystemHealth() HealthStatus {
    status := HealthStatus{
        Healthy:   true,
        Version:   VERSION,
        Timestamp: time.Now().Unix(),
        Checks:    make(map[string]CheckResult),
    }
    
    // Check database
    start := time.Now()
    if err := checkDatabase(); err != nil {
        status.Healthy = false
        status.Checks["database"] = CheckResult{
            Healthy: false,
            Message: err.Error(),
            Latency: time.Since(start).Milliseconds(),
        }
    } else {
        status.Checks["database"] = CheckResult{
            Healthy: true,
            Latency: time.Since(start).Milliseconds(),
        }
    }
    
    // Check dependencies
    for name, url := range dependencies {
        start := time.Now()
        if err := checkDependency(url); err != nil {
            status.Healthy = false
            status.Checks[name] = CheckResult{
                Healthy: false,
                Message: err.Error(),
                Latency: time.Since(start).Milliseconds(),
            }
        } else {
            status.Checks[name] = CheckResult{
                Healthy: true,
                Latency: time.Since(start).Milliseconds(),
            }
        }
    }
    
    return status
}
```

### Node.js Health Endpoint
```javascript
// Basic health endpoint
app.get('/health', (req, res) => {
    res.json({ healthy: true });
});

// Advanced health endpoint
app.get('/health', async (req, res) => {
    const health = await checkHealth();
    
    res.status(health.healthy ? 200 : 503).json(health);
});

async function checkHealth() {
    const checks = {};
    let healthy = true;
    
    // Check database
    try {
        await db.ping();
        checks.database = { healthy: true };
    } catch (error) {
        checks.database = { 
            healthy: false, 
            message: error.message 
        };
        healthy = false;
    }
    
    // Check Redis
    try {
        await redis.ping();
        checks.redis = { healthy: true };
    } catch (error) {
        checks.redis = { 
            healthy: false, 
            message: error.message 
        };
        healthy = false;
    }
    
    return {
        healthy,
        timestamp: Date.now(),
        uptime: process.uptime(),
        checks
    };
}
```

## Startup Grace Period

### Allow Time for Initialization
```bash
# Wait for service to be ready
wait_for_ready() {
    local max_wait="${1:-60}"  # Maximum seconds to wait
    local check_interval="${2:-2}"  # Seconds between checks
    
    echo "Waiting for ${RESOURCE_NAME} to be ready..."
    
    local elapsed=0
    while [ $elapsed -lt $max_wait ]; do
        if check_health 2 1 >/dev/null 2>&1; then
            echo "✅ ${RESOURCE_NAME} is ready after ${elapsed}s"
            return 0
        fi
        
        echo "⏳ Not ready yet, waiting... (${elapsed}/${max_wait}s)"
        sleep $check_interval
        elapsed=$((elapsed + check_interval))
    done
    
    echo "❌ ${RESOURCE_NAME} failed to become ready within ${max_wait}s"
    return 1
}
```

## Readiness vs Liveness

### Readiness Check
```bash
# Check if service is ready to accept traffic
check_readiness() {
    # Service is ready if:
    # 1. Health endpoint responds
    # 2. Dependencies are connected
    # 3. Initialization is complete
    
    local response=$(curl -sf http://localhost:${PORT}/ready 2>/dev/null)
    local ready=$(echo "$response" | jq -r '.ready // false')
    
    if [ "$ready" = "true" ]; then
        echo "✅ Service is ready"
        return 0
    else
        echo "⏳ Service not ready: $(echo "$response" | jq -r '.message // "initializing"')"
        return 1
    fi
}
```

### Liveness Check
```bash
# Check if service should be restarted
check_liveness() {
    # Service is alive if:
    # 1. Process is running
    # 2. Not deadlocked
    # 3. Can respond to requests
    
    # First check process
    if ! pgrep -f "${RESOURCE_NAME}" >/dev/null; then
        echo "❌ Process not running"
        return 1
    fi
    
    # Then check responsiveness
    if ! timeout 10 curl -sf http://localhost:${PORT}/live >/dev/null 2>&1; then
        echo "❌ Service not responding (possible deadlock)"
        return 1
    fi
    
    echo "✅ Service is alive"
    return 0
}
```

## Integration with Lifecycle

### In Setup Phase
```bash
setup() {
    # ... setup logic ...
    
    # Verify health after setup
    if ! check_health; then
        echo "ERROR: Setup completed but health check failed"
        return 1
    fi
}
```

### In Start Phase
```bash
start() {
    # Start the service
    start_service
    
    # Wait for it to be ready
    if ! wait_for_ready; then
        echo "ERROR: Service failed to become ready"
        stop_service
        return 1
    fi
    
    echo "Service started successfully"
}
```

### In Stop Phase
```bash
stop() {
    # Graceful shutdown
    echo "Stopping service..."
    
    # Stop accepting new requests
    curl -X POST http://localhost:${PORT}/shutdown/prepare
    
    # Wait for ongoing requests to complete
    sleep 2
    
    # Stop the service
    stop_service
    
    # Verify stopped
    if check_health 1 1 >/dev/null 2>&1; then
        echo "WARNING: Service still responding after stop"
        force_stop_service
    fi
}
```

## Monitoring Integration

### Expose Metrics
```go
// Prometheus metrics for health
var (
    healthCheckTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "health_check_total",
            Help: "Total number of health checks",
        },
        []string{"status"},
    )
    
    healthCheckDuration = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name: "health_check_duration_seconds",
            Help: "Duration of health checks",
        },
    )
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    defer func() {
        healthCheckDuration.Observe(time.Since(start).Seconds())
    }()
    
    health := checkHealth()
    
    if health.Healthy {
        healthCheckTotal.WithLabelValues("success").Inc()
        w.WriteHeader(http.StatusOK)
    } else {
        healthCheckTotal.WithLabelValues("failure").Inc()
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    
    json.NewEncoder(w).Encode(health)
}
```

## Common Health Check Failures

### Pattern 1: Timeout Too Short
```bash
# Problem: Health check times out during high load
# Solution: Increase timeout or add circuit breaker
check_health 10 3  # 10 second timeout, 3 retries
```

### Pattern 2: Missing Dependencies
```bash
# Problem: Health returns OK but dependencies are down
# Solution: Check all critical dependencies
check_health_with_deps() {
    check_health || return 1
    check_dependency postgres || return 1
    check_dependency redis || return 1
}
```

### Pattern 3: Startup Failures
```bash
# Problem: Health check fails during startup
# Solution: Add startup grace period
start_with_grace() {
    start_service
    sleep 5  # Grace period
    wait_for_ready
}
```

## Best Practices

### DO's
✅ **Keep health checks fast** (<1 second ideally)
✅ **Check actual functionality** not just process existence
✅ **Include dependency checks** for critical services
✅ **Add appropriate timeouts** to prevent hanging
✅ **Return meaningful status** in response body
✅ **Implement graceful degradation** for non-critical checks

### DON'Ts
❌ **Don't check everything** - Focus on critical paths
❌ **Don't block on slow checks** - Use async where possible
❌ **Don't ignore failures** - Log and alert appropriately
❌ **Don't return OK when degraded** - Be honest about state
❌ **Don't forget startup time** - Allow services to initialize

## Remember

**Health checks are critical** - They determine service availability

**Fast and accurate** - Quick response, truthful status

**Fail fast, recover gracefully** - Detect issues quickly, handle smoothly

**Monitor the monitors** - Track health check metrics

**Documentation saves debugging** - Clear health check specs prevent confusion

Good health checks are the foundation of reliable systems. Invest time in making them robust.