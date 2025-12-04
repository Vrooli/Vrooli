# Recovery Actions

Recovery actions allow you to fix issues detected by health checks, either automatically or manually through the dashboard.

## How Recovery Actions Work

1. **Detection**: A health check identifies a problem
2. **Available Actions**: The check lists what actions can help
3. **Execution**: You (or the autoheal loop) execute an action
4. **Verification**: The next check cycle confirms recovery

## Action Types

### Service Lifecycle
- **Start**: Start a stopped service
- **Stop**: Stop a running service (dangerous)
- **Restart**: Restart to clear state and errors

### Diagnostic
- **View Logs**: Retrieve recent logs for investigation
- **Analyze**: Show detailed diagnostic information
- **List**: Enumerate affected items

### Cleanup
- **Reap Zombies**: Send SIGCHLD to parent processes
- **Flush Cache**: Clear cached data
- **Kill Port**: Terminate processes on specific ports

## Actions by Check

### Infrastructure Checks

| Check | Actions | Notes |
|-------|---------|-------|
| infra-ntp | Enable NTP, Force Sync | Requires sudo |
| infra-resolved | Start, Restart, Flush Cache, Logs | systemd service |
| infra-cloudflared | Start, Restart, Logs | Tunnel recovery |

### System Checks

| Check | Actions | Notes |
|-------|---------|-------|
| system-zombies | Reap, List | Signals parent processes |
| system-ports | Analyze, TIME_WAIT, Kill Port | Diagnostic focused |

### Resource Checks

All resource checks support:
- **Start**: Start if stopped
- **Stop**: Stop if running (dangerous)
- **Restart**: Restart to recover
- **Logs**: View recent logs

## Executing Actions

### Via Dashboard
1. Navigate to the check card
2. Click the actions button
3. Select the desired action
4. Confirm if marked as dangerous

### Via API
```bash
# Get available actions
curl http://localhost:PORT/api/v1/checks/{checkId}/actions

# Execute an action
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
- Success/failure
- Output and errors
- Duration

View history via:
```bash
curl http://localhost:PORT/api/v1/actions/history
```

## Dangerous Actions

Some actions are marked as "dangerous" and require confirmation:
- **Stop**: Stops a service (causes downtime)
- **Restart**: Causes brief downtime
- **Kill Port**: Terminates processes

These actions show a warning before execution in the UI.

## Best Practices

1. **Check logs first**: Use "View Logs" before restart
2. **Understand the impact**: Restart causes brief downtime
3. **Monitor after recovery**: Watch for the issue to recur
4. **Address root cause**: Recovery is a bandaid, not a fix
