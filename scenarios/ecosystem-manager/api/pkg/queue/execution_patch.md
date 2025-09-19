# Integration Patch for execution.go

## Changes needed in execution.go to use the new ProcessManager:

### 1. In the Processor struct, add:
```go
type Processor struct {
    // ... existing fields ...
    processManager *ProcessManager  // Add this
}
```

### 2. In NewProcessor function:
```go
func NewProcessor(queueDir string) *Processor {
    return &Processor{
        // ... existing initialization ...
        processManager: NewProcessManager(),  // Add this
    }
}
```

### 3. Replace the process management in executeClaudeCode:

**BEFORE (lines 273-283):**
```go
// Start the command
if err := cmd.Start(); err != nil {
    return &tasks.ClaudeCodeResponse{
        Success: false,
        Error:   fmt.Sprintf("Failed to start Claude Code: %v", err),
    }, nil
}

// Register the running process for tracking
qp.registerRunningProcess(task.ID, cmd, ctx, cancel)
defer qp.unregisterRunningProcess(task.ID) // Always cleanup on exit
```

**AFTER:**
```go
// Start the managed process
if err := qp.processManager.StartProcess(task.ID, cmd, ctx, cancel); err != nil {
    return &tasks.ClaudeCodeResponse{
        Success: false,
        Error:   fmt.Sprintf("Failed to start Claude Code: %v", err),
    }, nil
}
```

### 4. Replace cmd.Wait() handling (line 319):

**BEFORE:**
```go
// Wait for completion
waitErr := cmd.Wait()
```

**AFTER:**
```go
// The process manager handles Wait() in a goroutine
// We just need to wait for completion or timeout
select {
case <-ctx.Done():
    // Context cancelled (timeout or manual termination)
    waitErr := ctx.Err()
case <-time.After(timeoutDuration):
    // Shouldn't happen as context handles timeout
    waitErr := context.DeadlineExceeded
}
```

### 5. Replace TerminateRunningProcess method:

**BEFORE:**
```go
func (qp *Processor) TerminateRunningProcess(taskID string) error {
    // ... existing complex termination logic ...
}
```

**AFTER:**
```go
func (qp *Processor) TerminateRunningProcess(taskID string) error {
    return qp.processManager.TerminateProcess(taskID, 5*time.Second)
}
```

### 6. Add graceful shutdown in main.go or server initialization:

```go
// On server shutdown
defer func() {
    log.Println("Terminating all running processes...")
    processor.processManager.TerminateAll(10 * time.Second)
}()
```

## Benefits of This Approach:

1. **No Zombie Processes**: The process reaper automatically cleans up any zombies
2. **Guaranteed Wait()**: Every process has Wait() called in a goroutine
3. **Process Groups**: Child processes are grouped for clean termination
4. **Graceful Escalation**: SIGTERM first, then SIGKILL if needed
5. **Concurrent Safety**: Proper locking prevents race conditions
6. **Clean Shutdown**: All processes terminated on server shutdown

## Testing the Changes:

```bash
# Test zombie prevention
vrooli scenario run ecosystem-manager

# In another terminal, check for zombies
ps aux | grep defunct

# The process reaper should log when it cleans up zombies:
# "Reaped zombie process: PID 12345, exit status: 0"
```

## Additional Recommendations:

1. **Add metrics** for process lifecycle events
2. **Add health checks** to detect stuck processes
3. **Consider using** `exec.CommandContext` consistently
4. **Add process resource limits** using SysProcAttr.Rlimit
5. **Log process trees** for debugging complex scenarios