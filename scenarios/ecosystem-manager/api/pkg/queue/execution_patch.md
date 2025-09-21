# Execution Pipeline Notes

This document captures the post-refactor task execution lifecycle inside the queue processor.

## High-level flow

1. `ProcessQueue` moves the task file into `queue/in-progress/` and immediately calls
   `reserveExecution(taskID, agentTag, startedAt)` so the in-memory registry mirrors the
   filesystem before any work starts.
2. `executeTask` assembles prompts and, once the Claude Code process starts, upgrades the
   reservation via `registerExecution(...)`. The registry now holds the PID, agent tag, and
   start time used by the UI, metrics, and reconciler.
3. `callClaudeCode` streams logs, waits on the managed process, and defers a cleanup routine
   that always calls `unregisterExecution`. If prompt assembly or agent launch fails, the
   early-exit guard inside `executeTask` also clears the reservation to prevent phantom
   in-progress tasks.
4. `finalizeTaskStatus` moves the file to its terminal column *after* the agent exits, keeping
   the filesystem as the single source of truth.

## Termination semantics

- `TerminateRunningProcess` first asks `ProcessManager` to cancel the process group. Only when
  the process manager had nothing to do do we fall back to `stopClaudeAgent` directly.
- Forced terminations reset the rolling log buffer and drop the execution reservation so the
  reconciler can safely re-queue the task on the next pass.

## Why this matters

- The queue no longer relies on parallel maps (`runningProcesses` vs. process manager). A single
  execution registry drives UI telemetry, reconciliation, and termination logic.
- Reserving the slot before prompt assembly prevents the reconciler from returning the task to
  `pending` during long pre-flight work, eliminating duplicate launches and infinite loops.
- Cleanup paths are symmetric: the registry entry disappears exactly once, regardless of success,
  failure, rate limiting, or manual termination, which keeps the UI, API, and filesystem aligned.

These invariants are critical for keeping task execution reliable now that task files are the
canonical state indicator.
