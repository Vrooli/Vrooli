# Recovery Actions

Recovery actions allow you to fix issues detected by health checks, either automatically or manually through the dashboard.

## How Recovery Actions Work

1. **Detection**: A health check identifies a problem
2. **Available Actions**: The check lists what actions can help
3. **Execution**: You (or the autoheal loop) execute an action
4. **Verification**: The action verifies success by re-running the check
5. **Logging**: Result is persisted to database with success/failure status

## Automatic Verification

Recovery actions for critical operations (start, restart) automatically verify success:

- After executing the action, waits a short period for the service to initialize
- Re-runs the health check to confirm the issue is resolved
- Reports verified success only if the check now passes
- Reports failure if the check still fails after the action

This ensures you get accurate feedback about whether recovery actually worked.

## Action Types

### Service Lifecycle
- **Start**: Start a stopped service (verified)
- **Stop**: Stop a running service (dangerous, not verified)
- **Restart**: Restart to clear state and errors (verified)

### Diagnostic
- **View Logs**: Retrieve recent logs for investigation
- **Diagnose**: Show detailed diagnostic information
- **List**: Enumerate affected items
- **Info**: Get service/resource information

### Cleanup
- **Reap Zombies**: Send SIGCHLD to parent processes (verified)
- **Flush Cache**: Clear cached data
- **Cleanup Ports**: Terminate processes on specific ports
- **Prune**: Remove unused Docker resources

## Actions by Check

### Infrastructure Checks

| Check | Actions | Verified | Notes |
|-------|---------|----------|-------|
| infra-docker | Start, Restart, Prune, Logs, Info, Open Desktop | Yes (start/restart) | systemctl for Linux, Docker Desktop for macOS |
| infra-cloudflared | Start, Restart, Test Tunnel, Logs, Diagnose | Yes (start/restart) | Cloudflare tunnel service |
| infra-dns | Restart Resolved, Flush Cache, Test External | Yes (restart) | DNS resolution checks |
| infra-resolved | Start, Restart, Flush Cache, Logs | Yes (start/restart) | systemd-resolved service |
| infra-ntp | Enable NTP, Force Sync | Yes | Time synchronization |

### System Checks

| Check | Actions | Verified | Notes |
|-------|---------|----------|-------|
| system-zombies | Reap, List | Yes (reap) | Signals parent processes, confirms cleanup |
| system-ports | Analyze, TIME_WAIT, Kill Port | No | Diagnostic focused |

### Resource Checks

All resource checks support these actions:

| Action | Verified | Notes |
|--------|----------|-------|
| Start | Yes | Start if stopped, verify running |
| Stop | No | Stop if running (dangerous) |
| Restart | Yes | Restart to recover, verify running |
| Logs | No | View recent logs |

Resources monitored: postgres, redis, ollama, qdrant, searxng, browserless

### Scenario Checks

All scenario checks support these actions:

| Action | Verified | Notes |
|--------|----------|-------|
| Start | Yes | Start if stopped, verify running |
| Stop | No | Stop if running (dangerous) |
| Restart | Yes | Restart to recover, verify running |
| Restart Clean | Yes | Stop, cleanup ports, start fresh |
| Cleanup Ports | No | Kill processes on scenario ports |
| Logs | No | View recent logs |
| Diagnose | No | Gather status, ports, and logs |

## Executing Actions

### Via Dashboard
1. Navigate to the check card
2. Click the actions button
3. Select the desired action
4. Confirm if marked as dangerous
5. View verification result in the response

### Via API
```bash
# Get available actions
curl http://localhost:PORT/api/v1/checks/{checkId}/actions

# Execute an action (returns verification status)
curl -X POST http://localhost:PORT/api/v1/checks/{checkId}/actions/{actionId}
```

### Via CLI
```bash
# Trigger a full health cycle (runs all checks)
vrooli autoheal tick

# Force all checks regardless of interval
vrooli autoheal tick --force
```

## Action Logging

All executed actions are logged to the database:
- Timestamp
- Check ID
- Action ID
- Success/failure (includes verification result)
- Output and errors
- Duration (includes verification time)

View history via:
```bash
curl http://localhost:PORT/api/v1/actions/history

# Filter by check
curl http://localhost:PORT/api/v1/actions/history?checkId=resource-postgres
```

## Dangerous Actions

Some actions are marked as "dangerous" and require confirmation:
- **Stop**: Stops a service (causes downtime)
- **Restart**: Causes brief downtime
- **Kill Port**: Terminates processes
- **Prune**: Removes Docker data

These actions show a warning before execution in the UI.

## Verification Timing

Different checks have different verification delays to allow services to initialize:

| Check Type | Delay | Reason |
|------------|-------|--------|
| DNS | 2s | DNS resolver is fast to start |
| NTP | 3s | NTP sync may take a moment |
| Docker | 5s | Container daemon initialization |
| Cloudflared | 5s | Tunnel establishment |
| Resources | 3s | CLI-managed services |
| Scenarios | 5s | Multiple components to start |
| Zombies | 2s | SIGCHLD processing |

## Best Practices

1. **Check logs first**: Use "View Logs" before restart
2. **Understand the impact**: Restart causes brief downtime
3. **Trust verification**: Actions report verified success
4. **Monitor after recovery**: Watch for the issue to recur
5. **Address root cause**: Recovery is a bandaid, not a fix
6. **Use clean restart**: For scenarios with port conflicts, use "Restart Clean"
