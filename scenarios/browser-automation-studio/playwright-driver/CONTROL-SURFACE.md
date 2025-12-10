# Playwright Driver Control Surface

This document describes the tunable levers available in the Playwright Driver. These configuration options allow operators and developers to adjust behavior for different environments, workloads, and use cases.

## Configuration Overview

All configuration is centralized in `src/config.ts`. Values are loaded from environment variables with sensible defaults.

### Configuration Groups

| Group | Purpose | Primary Audience |
|-------|---------|------------------|
| `server` | HTTP server settings | Operator |
| `browser` | Browser launch options | Operator |
| `session` | Session lifecycle management | Operator |
| `execution` | Instruction execution timeouts | Operator/Agent |
| `recording` | Record mode behavior | Developer/Agent |
| `telemetry` | Observability data collection | Operator |
| `logging` | Log output configuration | Operator |
| `metrics` | Prometheus metrics endpoint | Operator |

## Execution Timeouts

The most impactful levers for reliability vs speed tradeoffs.

| Environment Variable | Default | Range | Description |
|---------------------|---------|-------|-------------|
| `EXECUTION_DEFAULT_TIMEOUT_MS` | 30000 | 1000-300000 | General operations (click, type, etc.) |
| `EXECUTION_NAVIGATION_TIMEOUT_MS` | 45000 | 5000-300000 | Page navigation (goto, reload) |
| `EXECUTION_WAIT_TIMEOUT_MS` | 30000 | 1000-300000 | Wait operations (waitForSelector) |
| `EXECUTION_ASSERTION_TIMEOUT_MS` | 15000 | 1000-120000 | Assertion checks |
| `EXECUTION_REPLAY_TIMEOUT_MS` | 10000 | 1000-120000 | Replay preview actions |

### Tuning Guidance

**Fast local testing:**
```bash
EXECUTION_DEFAULT_TIMEOUT_MS=15000
EXECUTION_NAVIGATION_TIMEOUT_MS=20000
EXECUTION_ASSERTION_TIMEOUT_MS=5000
```

**Flaky/slow sites:**
```bash
EXECUTION_DEFAULT_TIMEOUT_MS=60000
EXECUTION_NAVIGATION_TIMEOUT_MS=90000
EXECUTION_WAIT_TIMEOUT_MS=60000
```

**CI environments:**
Use defaults or slightly higher to account for resource contention.

## Recording Configuration

Controls Record Mode behavior.

### Core Settings

| Environment Variable | Default | Range | Description |
|---------------------|---------|-------|-------------|
| `RECORDING_MAX_BUFFER_SIZE` | 10000 | 100-100000 | Max actions buffered per session |
| `RECORDING_MIN_SELECTOR_CONFIDENCE` | 0.3 | 0-1 | Minimum confidence for selector candidates |
| `RECORDING_DEFAULT_SWIPE_DISTANCE` | 300 | 50-1000 | Default swipe gesture distance (px) |

### Debounce Settings

Controls responsiveness vs. batching tradeoff for event capture.

| Environment Variable | Default | Range | Description |
|---------------------|---------|-------|-------------|
| `RECORDING_INPUT_DEBOUNCE_MS` | 500 | 50-2000 | Keystroke batching delay |
| `RECORDING_SCROLL_DEBOUNCE_MS` | 150 | 50-1000 | Scroll event debouncing |

### Selector Generation Settings

Controls how element selectors are generated during recording.

| Environment Variable | Default | Range | Description |
|---------------------|---------|-------|-------------|
| `RECORDING_MAX_CSS_DEPTH` | 5 | 2-10 | Max CSS path traversal depth |
| `RECORDING_INCLUDE_XPATH` | true | boolean | Include XPath as fallback strategy |

### Buffer Size Tradeoffs

- **Lower values** (100-1000): Lower memory usage, may lose actions in long sessions
- **Default** (10000): Good balance for most use cases
- **Higher values** (50000+): Suitable for long recording sessions, higher memory

### Selector Confidence Tradeoffs

- **Lower values** (0.1-0.3): More selector candidates, higher recall
- **Default** (0.3): Balanced - filters out low-quality selectors
- **Higher values** (0.5-0.8): Fewer but more stable selectors

### Debounce Tradeoffs

**More responsive recording (lower debounce):**
```bash
RECORDING_INPUT_DEBOUNCE_MS=100
RECORDING_SCROLL_DEBOUNCE_MS=50
```
Captures more individual events, higher granularity, but more noise.

**More batching (higher debounce):**
```bash
RECORDING_INPUT_DEBOUNCE_MS=1000
RECORDING_SCROLL_DEBOUNCE_MS=300
```
Fewer, more coalesced events. Better for slow connections or high-latency scenarios.

### Selector Generation Tradeoffs

**Shorter selectors (lower depth):**
```bash
RECORDING_MAX_CSS_DEPTH=3
```
Produces shorter CSS paths, but may be less unique on complex pages.

**CSS-only selectors:**
```bash
RECORDING_INCLUDE_XPATH=false
```
Disables XPath fallback. Use when CSS selectors are preferred or XPath causes issues.

## Session Management

| Environment Variable | Default | Range | Description |
|---------------------|---------|-------|-------------|
| `MAX_SESSIONS` | 10 | 1-100 | Maximum concurrent sessions |
| `SESSION_IDLE_TIMEOUT_MS` | 300000 | 10000-3600000 | Session idle timeout (5 min default) |
| `SESSION_POOL_SIZE` | 5 | 1-50 | Pre-warmed session pool size |
| `CLEANUP_INTERVAL_MS` | 60000 | 5000-600000 | Idle session cleanup interval |

## Telemetry Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `SCREENSHOT_ENABLED` | true | Enable screenshot capture |
| `SCREENSHOT_FULL_PAGE` | true | Capture full page vs viewport |
| `SCREENSHOT_QUALITY` | 80 | JPEG quality (1-100) |
| `SCREENSHOT_MAX_SIZE` | 512000 | Max screenshot size in bytes |
| `DOM_ENABLED` | true | Enable DOM snapshot capture |
| `DOM_MAX_SIZE` | 524288 | Max DOM snapshot size |
| `CONSOLE_ENABLED` | true | Capture console logs |
| `CONSOLE_MAX_ENTRIES` | 100 | Max console entries per instruction |
| `NETWORK_ENABLED` | true | Capture network events |
| `NETWORK_MAX_EVENTS` | 200 | Max network events per instruction |
| `HAR_ENABLED` | false | Enable HAR file recording |
| `VIDEO_ENABLED` | false | Enable video recording |
| `TRACING_ENABLED` | false | Enable Playwright tracing |

## Browser Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `HEADLESS` | true | Run browser headless |
| `BROWSER_EXECUTABLE_PATH` | (auto) | Custom browser executable |
| `BROWSER_ARGS` | (none) | Comma-separated browser args |
| `IGNORE_HTTPS_ERRORS` | false | Ignore HTTPS certificate errors |

## Server Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `PLAYWRIGHT_DRIVER_PORT` | 39400 | HTTP server port |
| `PLAYWRIGHT_DRIVER_HOST` | 127.0.0.1 | HTTP server bind host |
| `REQUEST_TIMEOUT_MS` | 300000 | Max request duration |
| `MAX_REQUEST_SIZE` | 5242880 | Max request body size |

## Logging & Metrics

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `LOG_LEVEL` | info | Log level (debug/info/warn/error) |
| `LOG_FORMAT` | json | Log format (json/text) |
| `METRICS_ENABLED` | true | Enable Prometheus metrics |
| `METRICS_PORT` | 9090 | Metrics endpoint port |

## Internal Constants (Not Configurable)

Some values are intentionally kept as internal constants:

| Constant | Value | Reason |
|----------|-------|--------|
| `MIN_TIMEOUT_MS` | 100 | Safety floor - timeouts below this are unreliable |
| `MAX_TIMEOUT_MS` | 600000 | Safety ceiling - 10 minute max |
| `ALLOWED_PROTOCOLS` | http, https, about, data | Security boundary |
| `MAX_CLOCK_DRIFT_MS` | 60000 | Idempotency edge case |
| `IDEMPOTENCY_KEY_MAX_AGE_MS` | 3600000 | Replay window limit |
| `URL_MAX_LENGTH` | 8192 | Security - prevents URL-based attacks |
| `MAX_PENDING_REQUESTS` | 500 | Network collector memory bound |
| `MAX_REQUEST_AGE_MS` | 30000 | Network collector stale request cleanup |
| `DRAG_STEPS` | 10 | Drag animation smoothness |
| `SWIPE_STEPS` | 10 | Swipe animation smoothness |

These are kept internal because:
1. They affect security boundaries
2. They're edge case handling unlikely to need tuning
3. They're implementation details that could break if changed

## Programmatic Configuration

Configuration is also available programmatically:

```typescript
import { loadConfig, type Config } from './config';

const config: Config = loadConfig();

// Access execution timeouts
const timeout = config.execution.defaultTimeoutMs;

// Access recording settings
const bufferSize = config.recording.maxBufferSize;

// Access recording debounce settings
const inputDebounce = config.recording.debounce.inputMs;
const scrollDebounce = config.recording.debounce.scrollMs;

// Access selector generation settings
const maxCssDepth = config.recording.selector.maxCssDepth;
const includeXPath = config.recording.selector.includeXPath;
```

## Adding New Levers

When adding a new configurable lever:

1. Add to the appropriate group in `ConfigSchema` in `src/config.ts`
2. Add environment variable parsing in `loadConfig()`
3. Document the tradeoff in code comments
4. Add entry to this document
5. Set a sensible default that works for common usage
6. Consider validation bounds (min/max values)
