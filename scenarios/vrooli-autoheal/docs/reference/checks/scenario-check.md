# Scenario Checks (scenario-*)

Monitors running Vrooli scenarios to ensure they remain healthy and responsive.

## Overview

| Property | Value |
|----------|-------|
| Check ID | `scenario-{name}` (e.g., `scenario-landing-manager`) |
| Category | Scenario |
| Interval | 60 seconds |
| Platforms | All |

## What It Monitors

Scenario checks run `vrooli scenario status {name}` and interpret the output:

```mermaid
flowchart TD
    A[Run vrooli scenario status] --> B{Exit Code?}
    B -->|Non-zero| C[Warning: Command Failed]
    B -->|Zero| D{Output Contains?}
    D -->|"running"| E[OK: Scenario Healthy]
    D -->|"not running"/"stopped"| F[Warning: Scenario Stopped]
    D -->|Other| G[Warning: Unclear Status]
```

## Status Meanings

| Status | Meaning |
|--------|---------|
| **OK** | Scenario is running and healthy |
| **Warning** | Scenario is stopped or status unclear |
| **Critical** | Scenario health check failed (e.g., health endpoint returned error) |

Unlike resources, scenarios that are simply "stopped" return Warning (not Critical) because it may be intentional.

## Why It Matters

Scenarios are the business logic of Vrooli:
- Each scenario may be a customer-facing service
- Scenario downtime = service downtime
- Early detection enables quick recovery

## Common Failure Causes

### 1. Scenario Process Crashed
```bash
# Check scenario status
vrooli scenario status my-scenario

# View logs
vrooli scenario logs my-scenario

# Restart
vrooli scenario restart my-scenario
```

### 2. Port Already In Use
```bash
# Check ports
vrooli scenario port my-scenario

# Find what's using the port
sudo ss -tlnp | grep 3000

# Kill conflicting process or restart scenario on different port
```

### 3. Resource Dependencies Down
Scenarios depend on resources:
```bash
# Check if required resources are running
vrooli resource status postgres
vrooli resource status redis

# Start missing resources
vrooli resource start postgres
```

### 4. Build Failure
If scenario was recently updated:
```bash
# Check build status
cd scenarios/my-scenario
make build

# Fix errors and rebuild
```

### 5. Configuration Issues
```bash
# Check scenario config
cat scenarios/my-scenario/.vrooli/service.json
```

## Troubleshooting Steps

1. **Check status and logs**
   ```bash
   vrooli scenario status my-scenario
   vrooli scenario logs my-scenario --tail 100
   ```

2. **Check health endpoint**
   ```bash
   # Get port
   PORT=$(vrooli scenario port my-scenario)

   # Test health
   curl http://localhost:$PORT/health
   ```

3. **Check resource dependencies**
   ```bash
   # Look at service.json for dependencies
   cat scenarios/my-scenario/.vrooli/service.json | jq '.dependencies'

   # Verify each is running
   vrooli resource status postgres
   ```

4. **Restart scenario**
   ```bash
   vrooli scenario restart my-scenario
   ```

5. **Check system resources**
   ```bash
   # Memory
   free -h

   # CPU
   top -b -n 1 | head -20

   # Disk
   df -h
   ```

## Configuration

Scenario checks are auto-registered when scenarios are started. Configuration is in `.vrooli/service.json`:

```json
{
  "name": "my-scenario",
  "api": {
    "port": 3000,
    "healthPath": "/health"
  },
  "dependencies": {
    "resources": ["postgres", "redis"],
    "scenarios": ["auth-service"]
  }
}
```

## Related Checks

- **resource-***: Scenarios depend on resources
- **infra-***: Scenarios need infrastructure (network, Docker)
- Other scenario checks (inter-scenario dependencies)

## Recovery Actions

| Action | Description | Risk |
|--------|-------------|------|
| **Start** | Start the scenario | Safe |
| **Stop** | Stop the scenario | Medium - stops service |
| **Restart** | Standard restart of the scenario | Medium - brief downtime |
| **Restart (Clean Stale)** | Stop, cleanup ports, and restart | Medium - more thorough |
| **Cleanup Ports** | Kill processes holding scenario ports | High - may kill unrelated processes |
| **View Logs** | View recent scenario logs | Safe |
| **Diagnose** | Get detailed diagnostic information | Safe |

### Clean Stale Restart

The "Restart (Clean Stale)" action is particularly useful when:
- Scenario won't start due to "port already in use"
- Previous process didn't terminate cleanly
- Orphaned processes are holding ports

This action:
1. Stops the scenario normally
2. Identifies scenario ports via `vrooli scenario port`
3. Kills any processes still holding those ports
4. Starts the scenario fresh

### Port Cleanup Details

The "Cleanup Ports" action:
1. Gets scenario ports from `vrooli scenario port`
2. Uses `lsof` to find processes on each port
3. Sends SIGTERM first (graceful)
4. Falls back to SIGKILL if needed (force)

## Auto-Heal Behavior

When a scenario check fails and auto-recovery is triggered, the system will:
1. Attempt a standard restart
2. If restart fails due to port conflicts, use clean-stale restart
3. Log diagnostic information for manual review

## Scenario Health Endpoints

Scenarios should implement a health endpoint:

```go
// Go example
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    // Check internal health
    if err := checkDatabaseConnection(); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy", "error": err.Error()})
        return
    }
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
})
```

A good health endpoint checks:
- Database connectivity
- Required service dependencies
- Critical internal state

---

*Back to [Check Catalog](../check-catalog.md)*
