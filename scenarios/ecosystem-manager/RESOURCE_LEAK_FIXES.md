# Resource Leak Prevention Fixes

## Problem Description

The ecosystem-manager had several resource leak vulnerabilities:

1. **API restart scenario**: When the API restarts, running Claude Code processes become orphaned
2. **Stale process tracking**: Dead processes remained registered as "running"
3. **Incomplete error handling**: Failed operations could leave processes running
4. **No health monitoring**: No mechanism to detect when processes die unexpectedly

## Solution Implemented

### 1. Process Persistence (`/tmp/ecosystem-manager-processes.json`)

**What it does**: Saves running process information to disk so it survives API restarts.

**Structure**:
```json
[
  {
    "task_id": "resource-generator-example-20250110",
    "process_id": 12345,
    "start_time": "2025-01-10T12:34:56Z"
  }
]
```

**Lifecycle**:
- **On startup**: Load persisted processes, kill any that are still alive (orphans)
- **Process start**: Add to persistence file
- **Process end**: Remove from persistence file  
- **API shutdown**: Clear persistence file

### 2. Health Monitoring (Every 30 seconds)

**What it does**: Periodically checks if registered processes are still alive using `kill -0 PID`.

**Behavior**:
- Scans all registered processes
- Removes dead processes from registry
- Updates persistence file
- Logs cleanup activity

### 3. Orphan Process Cleanup

**What it does**: On startup, scans for claude-code processes that aren't in our registry.

**Process**:
```bash
pgrep -f "resource-claude-code"  # Find all claude-code processes
```
- Compare found PIDs with registered PIDs
- Kill any unregistered processes (orphans)
- Log cleanup activity

### 4. Improved Error Handling

**What it does**: Ensures processes are terminated even when operations fail.

**Enhanced scenarios**:
- If `io.ReadAll()` fails → Cancel context + kill process
- Always use context cancellation before process kill
- Graceful termination (SIGTERM) → Force kill (SIGKILL) → Verify death

## New Configuration

### Processor Settings
```go
processFile: "/tmp/ecosystem-manager-processes.json"
healthCheckInterval: 30 * time.Second
```

### Process Termination Strategy
1. **SIGTERM** (graceful shutdown)
2. **Wait 100ms**  
3. **SIGKILL** (force kill)
4. **Verify process is dead**

## API Changes

### New Methods Added
- `loadPersistedProcesses()` - Load processes on startup
- `persistProcesses()` - Save processes to disk
- `processHealthMonitor()` - Background health checking
- `cleanupOrphanedProcesses()` - Kill orphaned processes
- `isProcessAlive(pid)` - Check if PID exists
- `killProcess(pid)` - Graceful then forceful termination

### Modified Methods
- `registerRunningProcess()` - Now persists after registration
- `unregisterRunningProcess()` - Now persists after removal  
- `Stop()` - Now stops health monitoring and clears persistence

## Startup Sequence

1. **Create processor** → Set up persistence file path
2. **Load persisted processes** → Read previous process list  
3. **Clean up orphaned processes** → Kill any alive processes from previous run
4. **Start health monitor** → Begin background health checking
5. **Start queue processing** → Begin normal operation

## Shutdown Sequence

1. **Stop health monitoring** → End background monitoring
2. **Stop queue processing** → End task execution
3. **Clear persistence file** → Remove process tracking file

## Testing

### Verification Commands
```bash
# Check for orphaned processes
pgrep -f "resource-claude-code"

# Check persistence file
cat /tmp/ecosystem-manager-processes.json

# Monitor health checking (check logs)
tail -f /var/log/ecosystem-manager.log | grep "Process.*alive"
```

### Test Scenarios
1. **Normal restart**: API stops/starts cleanly → No orphans
2. **Crash restart**: API crashes → Orphans cleaned up on restart  
3. **Process death**: Claude process dies → Detected and cleaned up within 30s
4. **Failed operations**: I/O errors → Process properly terminated

## Security Considerations

- **Process file permissions**: 0644 (readable by owner and group)
- **PID validation**: Only positive PIDs accepted
- **Signal permissions**: Uses standard Unix signals (TERM/KILL)
- **Cleanup bounds**: Only kills claude-code processes, not arbitrary PIDs

## Monitoring

### Log Messages to Watch For
```
INFO: Loaded N persisted processes from /tmp/ecosystem-manager-processes.json
INFO: Cleaned up N orphaned processes from previous run  
INFO: Process 12345 for task task-id is no longer alive - removing from registry
INFO: Cleaned up N dead processes from registry
```

### Metrics Available
- `executing_count`: Current registered processes
- Health check frequency: Every 30 seconds
- Persistence updates: On process start/stop

## Benefits

✅ **No orphaned processes** after API restarts  
✅ **Accurate process tracking** with health monitoring  
✅ **Robust error recovery** with proper process termination  
✅ **Observable system** with detailed logging  
✅ **Minimal overhead** with efficient process checking  

## Potential Issues & Mitigations

| Issue | Mitigation |
|-------|------------|
| Persistence file corruption | JSON parsing errors are logged, system continues |
| High PID reuse | Health checks verify actual process, not just PID existence |
| Signal permission denied | Errors are logged, system continues with other processes |
| Disk space for persistence file | File is small (<1KB for typical use) and cleaned on shutdown |

This implementation provides comprehensive protection against resource leaks while maintaining system stability and observability.