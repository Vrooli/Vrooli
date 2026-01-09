# Configuration Tiers

This document explains the configuration options for the Playwright Driver, organized by importance and target audience.

## Configuration Philosophy

The Playwright Driver has ~45 configuration options. This is intentional - browser automation has many tunable aspects. However, **most users should only need to adjust 5-7 options**. The defaults are carefully chosen for common usage.

## Tier 1: Operational Essentials

**Audience**: Operators, DevOps, anyone deploying the driver
**When to change**: Initial deployment, resource constraints, reliability tuning

| Option | Env Variable | Default | Purpose |
|--------|-------------|---------|---------|
| `server.port` | `PLAYWRIGHT_DRIVER_PORT` | 39400 | HTTP server port |
| `session.maxConcurrent` | `MAX_SESSIONS` | 10 | Maximum parallel browser sessions |
| `execution.defaultTimeoutMs` | `EXECUTION_DEFAULT_TIMEOUT_MS` | 30000 | Main reliability dial - increase for slow sites |
| `logging.level` | `LOG_LEVEL` | info | Verbosity (debug/info/warn/error) |
| `metrics.enabled` | `METRICS_ENABLED` | true | Prometheus metrics endpoint |

### Quick Start for Common Scenarios

**Development/Testing (fast feedback)**:
```bash
export EXECUTION_DEFAULT_TIMEOUT_MS=10000
export LOG_LEVEL=debug
```

**Production (reliability focus)**:
```bash
export MAX_SESSIONS=5
export EXECUTION_DEFAULT_TIMEOUT_MS=45000
```

**CI/CD (resource constrained)**:
```bash
export MAX_SESSIONS=2
export EXECUTION_DEFAULT_TIMEOUT_MS=60000
```

---

## Tier 2: Advanced Tuning

**Audience**: Developers, performance engineers
**When to change**: Specific performance requirements, debugging issues

### Execution Timeouts
Fine-grained timeout control when `defaultTimeoutMs` isn't enough:

| Option | Env Variable | Default | When to Adjust |
|--------|-------------|---------|----------------|
| `navigationTimeoutMs` | `EXECUTION_NAVIGATION_TIMEOUT_MS` | 45000 | Slow initial page loads |
| `waitTimeoutMs` | `EXECUTION_WAIT_TIMEOUT_MS` | 30000 | Dynamic content loading |
| `assertionTimeoutMs` | `EXECUTION_ASSERTION_TIMEOUT_MS` | 15000 | Fast assertion feedback |

### Telemetry Controls
Control what observability data is collected:

| Option | Env Variable | Default | Impact |
|--------|-------------|---------|--------|
| `telemetry.screenshot.enabled` | `SCREENSHOT_ENABLED` | true | Disable for bandwidth savings |
| `telemetry.dom.enabled` | `DOM_ENABLED` | true | Disable for large pages |
| `telemetry.screenshot.quality` | `SCREENSHOT_QUALITY` | 80 | Lower for smaller payloads |

### Session Pool
For high-throughput scenarios:

| Option | Env Variable | Default | Purpose |
|--------|-------------|---------|---------|
| `session.poolSize` | `SESSION_POOL_SIZE` | 5 | Pre-warmed sessions |
| `session.idleTimeoutMs` | `SESSION_IDLE_TIMEOUT_MS` | 300000 | Session TTL |

---

## Tier 3: Internal/Rarely Changed

**Audience**: Driver developers, advanced troubleshooting
**When to change**: Almost never - these should just work

These options exist for completeness but have well-tuned defaults:

| Group | Options | Why You Shouldn't Change |
|-------|---------|-------------------------|
| Recording | `debounce.*`, `selector.*` | Optimized for human typing speed |
| Frame Streaming | `useScreencast`, `fallbackToPolling` | Auto-detects best method |
| Performance | `bufferSize`, `logSummaryInterval` | Diagnostic-only |
| Browser | `args`, `ignoreHTTPSErrors` | Playwright defaults are good |

### If You Think You Need to Change Tier 3 Options

1. **File an issue first** - there may be a better solution
2. **Check the source** - `src/config.ts` has detailed comments
3. **Measure impact** - some options have subtle interactions

---

## Environment Variable Reference

All configuration is done via environment variables. The full list:

```bash
# === Tier 1: Essential ===
PLAYWRIGHT_DRIVER_PORT=39400
MAX_SESSIONS=10
EXECUTION_DEFAULT_TIMEOUT_MS=30000
LOG_LEVEL=info
METRICS_ENABLED=true

# === Tier 2: Advanced ===
# Timeouts
EXECUTION_NAVIGATION_TIMEOUT_MS=45000
EXECUTION_WAIT_TIMEOUT_MS=30000
EXECUTION_ASSERTION_TIMEOUT_MS=15000
EXECUTION_REPLAY_TIMEOUT_MS=10000

# Telemetry
SCREENSHOT_ENABLED=true
SCREENSHOT_QUALITY=80
DOM_ENABLED=true
CONSOLE_ENABLED=true
NETWORK_ENABLED=true

# Sessions
SESSION_POOL_SIZE=5
SESSION_IDLE_TIMEOUT_MS=300000

# === Tier 3: Internal ===
# Server
PLAYWRIGHT_DRIVER_HOST=127.0.0.1
REQUEST_TIMEOUT_MS=300000
MAX_REQUEST_SIZE=5242880

# Browser
HEADLESS=true
BROWSER_EXECUTABLE_PATH=
BROWSER_ARGS=
IGNORE_HTTPS_ERRORS=false

# Recording
RECORDING_MAX_BUFFER_SIZE=10000
RECORDING_MIN_SELECTOR_CONFIDENCE=0.3
RECORDING_INPUT_DEBOUNCE_MS=500
RECORDING_SCROLL_DEBOUNCE_MS=150
RECORDING_MAX_CSS_DEPTH=5
RECORDING_INCLUDE_XPATH=true

# Frame Streaming
FRAME_STREAMING_USE_SCREENCAST=true
FRAME_STREAMING_FALLBACK=true

# Performance Debug
PLAYWRIGHT_DRIVER_PERF_ENABLED=false
PLAYWRIGHT_DRIVER_PERF_INCLUDE_HEADERS=true
PLAYWRIGHT_DRIVER_PERF_LOG_INTERVAL=60
PLAYWRIGHT_DRIVER_PERF_BUFFER_SIZE=100

# Metrics
METRICS_PORT=9090

# Logging
LOG_FORMAT=json
```

---

## Decision Guide

```
Need to deploy?
├─ Yes → Set Tier 1 options, done
└─ No

Having timeout issues?
├─ Everything → Increase EXECUTION_DEFAULT_TIMEOUT_MS
└─ Just navigation → Increase EXECUTION_NAVIGATION_TIMEOUT_MS

High memory usage?
├─ Many sessions → Reduce MAX_SESSIONS
└─ Large pages → Reduce SCREENSHOT_QUALITY, disable DOM_ENABLED

Debugging?
├─ Set LOG_LEVEL=debug
└─ Enable PLAYWRIGHT_DRIVER_PERF_ENABLED=true for frame timing
```

---

## See Also

- `src/config.ts` - Configuration schema with detailed comments
- `ARCHITECTURE.md` - System architecture and change axes
