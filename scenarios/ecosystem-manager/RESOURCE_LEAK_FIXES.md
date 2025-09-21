# Resource Leak Controls (2025-02 refresh)

The ecosystem-manager queue now relies on a single execution registry that mirrors the
filesystem: if a task file lives in `queue/in-progress`, the registry must contain an active
reservation for the same task ID. This section documents the safeguards that prevent orphaned
Claude Code agents and mismatched task state after the February 2025 refactor.

## Previous issues
- API restarts left behind unmanaged `resource-claude-code` processes.
- Multiple tracking maps (`runningProcesses` + process manager) drifted out of sync.
- Tasks could be moved back to `pending` while the agent was still assembling prompts,
  causing duplicate launches or infinite "In Progress" loops.

## Current safeguards

### 1. Single execution registry
- `reserveExecution(...)` is called immediately after the task file is moved into
  `queue/in-progress`. This ensures the reconciler and UI see the task as busy even while the
  executor is still assembling prompts.
- `registerExecution(...)` upgrades the reservation with the real `*exec.Cmd` once Claude starts.
- `unregisterExecution(...)` runs exactly once via the coordinated cleanup path. Early failures
  (prompt assembly errors, agent launch issues) are caught by a guard in `executeTask` that clears
  the reservation before returning.

### 2. Process manager integration
- All agents run inside a managed process group via `ProcessManager`. `TerminateRunningProcess`
  first asks the manager to cancel the context and send `SIGTERM`, falling back to direct
  `resource-claude-code agents stop` only when the manager had nothing to do.
- The process reaper in `queue/reaper.go` continues to reap zombies automatically.

### 3. Startup hygiene
- On boot the processor still calls `cleanupOrphanedProcesses()`, which scans `pgrep -f`
  output for `resource-claude-code run` entries and kills any PID that is not represented in
  the new execution registry. This catches leftovers from previous crashes or manual kills.

### 4. Shutdown semantics
- `Processor.Stop()` halts the queue loop and asks the process manager to terminate any
  remaining executions with a 10-second grace window. As soon as a task is marked inactive
  via the Settings toggle, the queue stops scheduling new work but existing agents are allowed
  to finish naturally.

### 5. Log lifecycle
- Every execution initializes a bounded log buffer; logs are flushed to
  `queue/logs/task-runs/<task-id>.log` when the agent exits. Forced terminations call
  `ResetTaskLogs` so retried tasks start with a clean slate.

## Removed components (and why)
- The old persistence file (`/tmp/ecosystem-manager-processes.json`) and 30-second health monitor
  were removed. With a single authoritative registry, we no longer need to continuously reconcile
  auxiliary state, and avoiding disk I/O eliminates a source of stale data during rapid restarts.

## Operational tips
- To inspect active agents, use the API endpoint `GET /api/queue/processes` or
  run `resource-claude-code agents list` and match tags of the form `ecosystem-<task-id>`.
- If a task appears stuck, run `resource-claude-code agents stop ecosystem-<task-id>` and then
  call the admin endpoint `POST /api/queue/processes/terminate` with the task ID to trigger the
  cleanup path.
- For audit trails, review the persisted task log files and the queue broadcast events emitted from
  `appendTaskLog`.

## Testing checklist
- Scenario restart with active agents → orphan cleanup removes leftover PIDs.
- Rate-limit recovery → task moves back to `pending`, registry entry disappears, and logs are kept.
- Manual termination via API → process manager cancels the agent, registry entry is cleared, file
  moves back to `pending`.
- Prompt assembly failure (before agent launch) → reservation is cleared by the early-exit guard.

These measures keep the filesystem, in-memory registry, and external agent state aligned, which is
critical for preventing the "forever in progress" loops we experienced previously.
