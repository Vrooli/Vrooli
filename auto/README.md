## Auto Orchestration (auto/)

This folder contains the lightweight automation framework used to run continuous improvement loops for Vrooli resources and scenarios. It provides:

- Generic loop core in `auto/lib/loop.sh`
- Task modules in `auto/tasks/<task>/`
- Entry shims and a central manager in `auto/*.sh`
- Persisted runtime artifacts in `auto/data/<task>/`

### Key Concepts

- **Task**: A loopable unit of work (e.g., `scenario-improvement`, `resource-improvement`).
- **Loop Core**: Reusable engine that schedules iterations, manages logs, event ledger, and summaries.
- **Task Manager**: Router that loads a task module and delegates to the loop core with subcommands.


## Common Commands

You can control loops either via the per-task shims or the central manager. Replace `<task>` with `scenario-improvement` or `resource-improvement`.

### Using the central manager

```bash
# Start/stop a loop in the background
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> start
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> stop

# Force stop (KILL)
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> force-stop

# Status, logs, rotation, and JSON summaries
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> status
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> logs -f
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> rotate
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> json summary

# Foreground loop (Ctrl-C to exit); optional max iterations
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> run-loop
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> run-loop --max 3

# One-shot iteration and health check
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> once
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> health
```

### Using the shims (optional)

Convenience wrappers exist for the common loops:

```bash
# Scenario improvement loop
/home/matthalloran8/Vrooli/auto/manage-scenario-loop.sh start
/home/matthalloran8/Vrooli/auto/manage-scenario-loop.sh stop

# Resource improvement loop
/home/matthalloran8/Vrooli/auto/manage-resource-loop.sh start
/home/matthalloran8/Vrooli/auto/manage-resource-loop.sh stop

# Simple wrappers that run the loop in the foreground
/home/matthalloran8/Vrooli/auto/scenario-improvement-loop.sh
/home/matthalloran8/Vrooli/auto/resource-improvement-loop.sh
```


## Prompts and Task Modules

- Default prompts live under `auto/tasks/<task>/prompts/`.
- You can override the prompt with `--prompt /abs/path/to/prompt.md` or the `PROMPT_PATH` env var.
- Task modules (e.g., `auto/tasks/resource-improvement/task.sh`) expose hooks and helper context for the loop core.


## Data, Logs, and Summaries

Each task writes to `auto/data/<task>/`:

- `loop.log`: Rolling loop log
- `events.ndjson`: Event ledger (start/finish, duration, exit code)
- `summary.json`: Computed metrics; `summary.txt`: optional NL summary (if Ollama available)
- `iterations/iter-<N>.log`: Redacted worker output per iteration
- `loop.pid`: Manager PID, `workers.pids`: active worker wrappers, `loop.lock`: lock file

Utilities:

```bash
# Follow logs
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> logs -f

# Rotate logs immediately (keeps recent N = ROTATE_KEEP)
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> rotate
# Rotate events ledger and prune
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> rotate --events 5

# Quick JSON summaries
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> json summary
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> json recent 10
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> json durations
```


## Useful Environment Variables

- `INTERVAL_SECONDS` (default 300): Delay between iterations
- `MAX_TURNS` (default 25): Max turns for the code worker
- `TIMEOUT` (default 1800): Per-iteration timeout (seconds)
- `MAX_CONCURRENT_WORKERS` (default task-defined): Concurrency limit
- `MAX_TCP_CONNECTIONS` (default 15): TCP gating threshold
- `LOOP_TCP_FILTER` (default includes `claude|anthropic|resource-claude-code`): Process/TCP filter. Empty disables gating
- `PROMPT_PATH`: Absolute path to a custom prompt
- `OLLAMA_SUMMARY_MODEL` (default `llama3.2:3b`): Used to generate `summary.txt` when available

Task-specific:

- Resource loop respects `RESOURCE_IMPROVEMENT_MODE`:
  - `plan` (default): plan-only; TCP gating disabled automatically; no file edits
  - `apply-safe`: allow non-destructive changes (e.g., start/stop, health checks)
  - `apply`: allow broader changes when safe


## Selection Helpers (resource-improvement)

Located in `auto/tools/selection/`:

- `resource-list.sh`: Lists resources in JSON from CLI or `.vrooli/service.json`
- `resource-candidates.sh`: Prioritizes resource candidates; merges live runtime status when available

Example:

```bash
/home/matthalloran8/Vrooli/auto/tools/selection/resource-candidates.sh
```


## Health and Troubleshooting

```bash
# Check binaries, data dir, and prompt discovery
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> health

# Dry-run to inspect composed prompt and config
/home/matthalloran8/Vrooli/auto/task-manager.sh --task <task> dry-run
```

Notes:
- Logs and events auto-rotate; keep count controlled by `ROTATE_KEEP`.
- In plan mode the loop avoids network gating and side effects, focusing on diagnostics and command plans. 