# Configuration Reference

This document describes the tunable levers (configuration options) for the Lifestyle Dashboard API. These settings allow operators to adjust behavior without code changes.

## Overview

The Lifestyle Dashboard uses a layered configuration approach:

1. **Defaults**: Sensible values baked into the `config` package
2. **Environment Variables**: Override defaults at runtime
3. **Code Changes**: For advanced customization (modify `config/config.go`)

All configuration is centralized in `api/config/config.go`.

---

## Database Configuration

Settings for SQLite database connection and retry behavior.

| Setting | Default | Environment Variable | Valid Range | Description |
|---------|---------|---------------------|-------------|-------------|
| `MaxOpenConns` | 1 | - | 1 (SQLite) | Maximum open connections. Must be 1 for SQLite to enforce single-writer constraint. |
| `MaxIdleConns` | 1 | - | 0-100 | Maximum idle connections in pool. |
| `BusyTimeout` | 5s | - | 1s-60s | How long SQLite waits for locks before returning BUSY. |
| `RetryMaxAttempts` | 3 | - | 1-10 | Connection attempts before failing. |
| `RetryBaseDelay` | 100ms | - | 10ms-1s | Initial delay between retry attempts. |
| `RetryMaxDelay` | 500ms | - | 100ms-30s | Maximum delay between retries. |

### Why MaxOpenConns = 1?

SQLite uses file-based locking and doesn't support concurrent writes. Setting `MaxOpenConns=1` ensures:
- No "database is locked" errors
- Predictable write ordering
- Optimal performance for SQLite's single-writer architecture

For PostgreSQL migration, this would increase to 25-100.

---

## Query Configuration

Settings that control API query behavior and limits.

| Setting | Default | Environment Variable | Valid Range | Description |
|---------|---------|---------------------|-------------|-------------|
| `DefaultEventLimit` | 100 | `LD_DEFAULT_EVENT_LIMIT` | 10-1000 | Events returned when no `?limit` specified. |
| `MaxEventLimit` | 1000 | - | 100-10000 | Maximum events a client can request. Protects against expensive queries. |
| `DefaultTimelineDays` | 7 | `LD_DEFAULT_TIMELINE_DAYS` | 1-365 | Days shown in timeline when no `?days` specified. |
| `MaxTimelineDays` | 365 | - | 30-3650 | Maximum timeline range to prevent expensive aggregations. |

### Tuning Tips

- **High-traffic deployments**: Lower `DefaultEventLimit` to 50 to reduce response sizes
- **Dashboard displays**: Increase `DefaultTimelineDays` to 30 for monthly views
- **Memory-constrained systems**: Lower `MaxEventLimit` to 500

### Example: Environment Override

```bash
# Reduce default event limit for bandwidth-constrained environments
export LD_DEFAULT_EVENT_LIMIT=50

# Show 14-day timeline by default
export LD_DEFAULT_TIMELINE_DAYS=14
```

---

## Health Check Configuration

Settings for domain health checks (checking if registered domains are healthy).

| Setting | Default | Environment Variable | Valid Range | Description |
|---------|---------|---------------------|-------------|-------------|
| `Timeout` | 5s | `LD_HEALTH_CHECK_TIMEOUT_SECS` | 1-30 | Time to wait for domain health endpoint response. |
| `UnhealthyThreshold` | 300 | - | 300-600 | HTTP status codes >= this value are considered unhealthy. |

### Behavior

When checking a domain's health (`GET /api/v1/domains/{name}/health`):

1. If domain has no `health_url`, returns current cached status
2. If domain has `health_url`, makes HTTP request with configured timeout
3. Response `< 300` → healthy, `>= 300` → unhealthy

### Tuning Tips

- **Slow services**: Increase timeout to 10s+ if domains have slow health endpoints
- **Fast networks**: Reduce timeout to 2s for quicker failure detection

```bash
# Allow 10 seconds for domain health checks
export LD_HEALTH_CHECK_TIMEOUT_SECS=10
```

---

## CORS Configuration

Cross-Origin Resource Sharing settings for browser access.

| Setting | Default | Description |
|---------|---------|-------------|
| `AllowedOrigins` | `[]` (any) | Origins allowed to make cross-origin requests. Empty = allow any. |
| `AllowCredentials` | `true` | Whether credentials can be included in requests. |

### Production Configuration

In production, restrict origins to your specific UI domains:

```go
// In config/config.go
func DefaultCORSConfig() CORSConfig {
    return CORSConfig{
        AllowedOrigins:   []string{"https://your-domain.com"},
        AllowCredentials: true,
    }
}
```

---

## What's NOT Configurable (By Design)

These values are intentionally not exposed as configuration:

| Setting | Why Not Configurable |
|---------|---------------------|
| Database schema | Schema changes require migrations, not runtime config |
| API version prefix (`/api/v1`) | Breaking change for clients; version via new prefix |
| UUID generation | Standard format; no benefit to alternatives |
| Timestamp format (RFC3339) | Standard format; clients depend on it |
| Log format | Simple text logging; switch to structured via code change |

---

## Environment Variable Summary

```bash
# Query defaults
LD_DEFAULT_EVENT_LIMIT=100       # Events returned by default
LD_DEFAULT_TIMELINE_DAYS=7       # Timeline days by default

# Health checks
LD_HEALTH_CHECK_TIMEOUT_SECS=5   # Domain health check timeout

# Standard scenario variables (managed by Vrooli)
API_PORT=15881                   # API server port
SQLITE_PATH=/data/lifestyle.db   # Database file path
```

---

## Configuration Validation

The config package validates values at load time:
- Invalid environment values fall back to defaults (with warning)
- Negative values are ignored
- Non-numeric strings for numeric settings are ignored

There are no runtime errors from misconfiguration - only warnings in logs and fallback behavior.

---

## Adding New Configuration

When adding new tunable levers:

1. Add to appropriate section in `config/config.go`
2. Provide a sensible default
3. Document valid range and impact
4. Add tests in `config/config_test.go`
5. Update this document
