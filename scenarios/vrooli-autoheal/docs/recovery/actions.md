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
| Status | No | Check current resource status |

Resources monitored: postgres, redis, ollama, qdrant, searxng

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
vrooli-autoheal tick

# Force all checks regardless of interval
vrooli-autoheal tick --force
```

## Action Logging

Coding-agent escalation uses the control-plane recovery ledger. If that ledger
cannot be opened, autoheal keeps its API, monitoring and mechanical recovery
available; an escalation request returns an explicit unavailable error and no
admission ID. Repair the ledger through its owner before expecting agent repair.
The health response exposes this initialization verdict under
`dependencies.coding_agent_recovery`; it is explicitly optional for API
readiness and is not a current native-execution observation.

Inspect investigations with `vrooli agent recover list --json`.
`vrooli agent recover reconcile --limit 32 --json` marks historical or expired
execution claims without native identity as `unconfirmed`, retaining diagnostics
and duplicate suppression. This is an uncertainty receipt, not proof that an
agent exited or that the scenario recovered. It stops and starts no process.

The native escalation implementation uses a retained governed root executable
and a separate versioned execution sidecar keyed by the admission ID. Before
dispatch, it commits a once-only native intent. The fixed worker must prove its
actual image, boot/process incarnation and native invocation, then claim that
intent. Its editor lease is bound to the actual operation cgroup before the
runner starts; a sibling agent scope cannot escape this job's deadline. The
native limits are ten minutes, 2 GiB and 1024 tasks, with at most two unresolved
native jobs admitted through the shared ledger. The hourly per-scenario window
and suppression of unknown historical executions remain in force.

Autoheal calls the control-plane reconciler every thirty seconds and before
another escalation request. Reconciliation never replays a consumed dispatch.
An admitted request whose durable dispatch bit is still unset can make its first
submission after complete native absence, retained-image and original-specification
checks. A crash after that bit is committed remains an uncertain start until
the admitted deadline; it is never submitted again. A failed attempt retains the
hourly budget. The model context retains the admission deadline; the independent
native ten-minute runtime ceiling starts at native activation, with the native
stop grace applying to descendant exclusion.

Autoheal distinguishes a retained admission receipt from native execution
ownership. An exact admitted native execution without exit proof defers
mechanical actions for that target. This hold survives temporary healthy
observations and Autoheal API replacement because the control-plane snapshot
reads the durable admission and execution together. Autoheal refreshes that
snapshot before a mechanical effect. Other targets and monitoring continue.
Terminal or historical unconfirmed receipts without a native sidecar preserve
the coding budget; they do not claim a physical writer or disable mechanical
recovery. A lost dispatch acknowledgement retains the known admission instead
of racing it with a restart. Snapshot failure preserves an already observed
native owner; an unavailable optional broker does not invent an owner.

It revokes before exclusion, requires exact native tree quiescence before
closing an unresolved receipt, and retires its retained executable and bounded
reserved atomic-write temporaries. A lost
main with remaining descendants is a failed worker, not a live investigation.
Unavailable native evidence preserves suppression and inputs. Existing runtime
registry expiry owns lease retirement after worker death. Runner completion
still does not prove that the scenario is healthy; subsequent health checks do.

Scenario check details retain the control plane's `healthError`, and failed
recovery verification includes that cause with the status. An initializing
readiness response must remain distinguishable from a persistent failure or a
generic degraded service. The outage investigation observed an old initializing
snapshot causing a second restart after the API had recovered; a healthy
lifecycle verdict and current API readiness must agree before that path is
qualified. Reducing the supervision policy's sensitivity is not the remedy.

Preparation commits its descriptor before copying the worker. An empty admission
ledger is durable before the first preparation. A missing ledger with retained
native inputs is explicit state loss and cannot reset the budget. Under the
ledger lock, reconciliation revokes unadmitted preparations and proves native
absence or exact exclusion before retiring their images. An unidentified
published worker, unexpected file or nonregular input is preserved for owner
inspection. Complete unit/cgroup absence also requires the captured root's
incarnation to be gone when a root identity was retained.

Rollout status is recorded in the active outage workstream's
`native-coding-agent-execution-milestone.json`. Native execution is deployed;
two configured-model file repairs passed, including one across Autoheal API
replacement. The crash-edge milestone records subsequent preparation/dispatch
qualification separately. An actual automatically triggered scenario/lifecycle
repair remains a separate obligation. The installed root must advertise worker
protocol 1 before the new API enables escalation. A missing protocol degrades
escalation explicitly while monitoring and mechanical recovery stay available.

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
