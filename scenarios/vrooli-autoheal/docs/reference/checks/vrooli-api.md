# Vrooli API Check (vrooli-api)

Monitors the main Vrooli API health endpoint to ensure the central orchestration layer is functioning.

## Overview

| Property | Value |
|----------|-------|
| Check ID | `vrooli-api` |
| Category | Infrastructure |
| Interval | 60 seconds |
| Platforms | All |

## What It Monitors

This check performs an HTTP GET request to the Vrooli API health endpoint:

- Default URL: `http://127.0.0.1:8092/health`
- Timeout: 5 seconds
- Parses JSON response for health status

## Status Meanings

| Status | Meaning |
|--------|---------|
| **OK** | API responds with `ok` or `healthy` status |
| **Warning** | API responds with `degraded` status |
| **Critical** | API unreachable, timeout, or `unhealthy`/`failed` status |

## Why It Matters

The Vrooli API is the **central orchestration layer** for:

- Resource management (start/stop/status)
- Scenario lifecycle management
- Port allocation and coordination
- Health status aggregation
- CLI command execution

If the Vrooli API is down:
- `vrooli` CLI commands will fail
- Resource and scenario management is unavailable
- Other autoheal checks that rely on `vrooli resource status` will fail

## Response Format

The check expects a JSON response like:

```json
{
  "status": "ok",
  "message": "optional message"
}
```

Valid status values:
- `ok` or `healthy` - All good
- `degraded` - Partially functional
- `unhealthy`, `failed`, `error` - Critical issues

## Common Failure Causes

### 1. API Not Running

```bash
# Check if Vrooli API is running
pgrep -f "vrooli" || echo "Vrooli not running"

# Check the API port
ss -tlnp | grep 8092
```

### 2. Port Conflict

```bash
# Check what's using port 8092
lsof -i :8092

# Check Vrooli's port configuration
cat ~/.vrooli/config.json | jq '.api_port'
```

### 3. Startup Issues

```bash
# Check Vrooli logs
cat ~/.vrooli/logs/vrooli.log | tail -50

# Try starting Vrooli manually
vrooli develop
```

### 4. Resource Dependencies

```bash
# The API may depend on certain resources
vrooli resource status postgres
vrooli resource status redis
```

## Troubleshooting Steps

1. **Test the endpoint manually**
   ```bash
   curl -v http://127.0.0.1:8092/health
   ```

2. **Check if API process exists**
   ```bash
   pgrep -af vrooli
   ```

3. **Check port binding**
   ```bash
   ss -tlnp | grep :8092
   ```

4. **Review logs**
   ```bash
   # Main Vrooli logs
   tail -100 ~/.vrooli/logs/vrooli.log

   # Check for recent errors
   grep -i error ~/.vrooli/logs/vrooli.log | tail -20
   ```

5. **Restart Vrooli**
   ```bash
   # If using vrooli develop
   vrooli stop
   vrooli develop
   ```

## Configuration

The check accepts the following options:

| Option | Default | Description |
|--------|---------|-------------|
| `url` | `http://127.0.0.1:8092/health` | Health endpoint URL |
| `timeout` | 5 seconds | Request timeout |

## Details Returned

```json
{
  "url": "http://127.0.0.1:8092/health",
  "timeout": "5s",
  "statusCode": 200,
  "responseTimeMs": 45,
  "apiStatus": "ok"
}
```

On error:
```json
{
  "url": "http://127.0.0.1:8092/health",
  "timeout": "5s",
  "error": "connection refused"
}
```

## Health Score

The health score is based on response time:
- < 1 second: 100
- 1-3 seconds: 75
- > 3 seconds: 50

## Related Checks

- **resource-postgres**: API may depend on database
- **resource-redis**: API may use Redis for caching
- **infra-network**: Network issues affect all HTTP checks

## Recovery Actions

| Action | Description | Risk |
|--------|-------------|------|
| **Restart API** | Stop and restart the Vrooli API server | Medium - brief downtime |
| **Kill Port Process** | Kill any process holding the API port (8092) | High - may affect other services |
| **View Logs** | View recent Vrooli API logs | Safe |
| **Diagnose** | Get diagnostic information including health check, port usage, and process info | Safe |

### Restart API

The restart action:
1. Finds and kills processes on the API port (default 8092)
2. Waits for processes to terminate
3. Attempts to restart using `vrooli develop` or the restart script

### Kill Port Process

Use this when:
- The port is held by a zombie or orphaned process
- The API won't start due to "address already in use"
- A previous API instance didn't terminate cleanly

### Diagnose

The diagnose action gathers:
- Health endpoint response (status code, response body, timing)
- Port usage details (what process is listening on 8092)
- Running Vrooli processes
- Resource status (postgres, redis, etc.)

## Manual Recovery Steps

When this check fails:

1. **First**: Check if the API process crashed and restart it
2. **Second**: Check dependent resources (postgres, redis)
3. **Third**: Review logs for crash reasons
4. **Fourth**: Check system resources (disk, memory)

## Port Configuration

If Vrooli uses a different port:

```bash
# Check current configuration
vrooli config get api_port

# Set a different port
vrooli config set api_port 8093
```

Update the check configuration accordingly.

---

*Back to [Check Catalog](../check-catalog.md)*
