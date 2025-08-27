## Auto Orchestration (auto/)

> **⚠️ DEPRECATION NOTICE**: This folder will be replaced by Vrooli scenarios once they're production-ready. The following scenarios will take over:
> - **scenario-generator-v1**: Replaces scenario-improvement loop with autonomous scenario generation
> - **resource-experimenter**: Replaces resource-improvement loop with systematic resource testing
> - **agent-metareasoning-manager**: Provides higher-level orchestration and decision-making
> - **ai-model-orchestra-controller**: Handles resource optimization and intelligent routing
> - **system-monitor**: Monitors health and triggers improvement cycles
>
> These scenarios embody Vrooli's vision where scenarios themselves become the infrastructure for continuous enhancement.

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
${APP_ROOT}/auto/task-manager.sh --task <task> start
${APP_ROOT}/auto/task-manager.sh --task <task> stop

# Force stop (KILL)
${APP_ROOT}/auto/task-manager.sh --task <task> force-stop

# Status, logs, rotation, and JSON summaries
${APP_ROOT}/auto/task-manager.sh --task <task> status
${APP_ROOT}/auto/task-manager.sh --task <task> logs -f
${APP_ROOT}/auto/task-manager.sh --task <task> rotate
${APP_ROOT}/auto/task-manager.sh --task <task> json summary

# Foreground loop (Ctrl-C to exit); optional max iterations
${APP_ROOT}/auto/task-manager.sh --task <task> run-loop
${APP_ROOT}/auto/task-manager.sh --task <task> run-loop --max 3

# One-shot iteration and health check
${APP_ROOT}/auto/task-manager.sh --task <task> once
${APP_ROOT}/auto/task-manager.sh --task <task> health
```

### Using the shims (optional)

Convenience shims exist for the common loops (delegate to `task-manager.sh`):

```bash
# Scenario improvement loop
${APP_ROOT}/auto/manage-scenario-loop.sh start
${APP_ROOT}/auto/manage-scenario-loop.sh stop

# Resource improvement loop
${APP_ROOT}/auto/manage-resource-loop.sh start
${APP_ROOT}/auto/manage-resource-loop.sh stop

# Foreground run via central manager (formerly simple wrappers)
${APP_ROOT}/auto/task-manager.sh --task scenario-improvement run-loop
${APP_ROOT}/auto/task-manager.sh --task resource-improvement run-loop
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
${APP_ROOT}/auto/task-manager.sh --task <task> logs -f

# Rotate logs immediately (keeps recent N = ROTATE_KEEP)
${APP_ROOT}/auto/task-manager.sh --task <task> rotate
# Rotate events ledger and prune
${APP_ROOT}/auto/task-manager.sh --task <task> rotate --events 5

# Quick JSON summaries
${APP_ROOT}/auto/task-manager.sh --task <task> json summary
${APP_ROOT}/auto/task-manager.sh --task <task> json recent 10
${APP_ROOT}/auto/task-manager.sh --task <task> json durations
```


## Useful Environment Variables

- `INTERVAL_SECONDS` (default 60): Delay between iterations
- `MAX_TURNS` (default 30): Max turns for the code worker
- `TIMEOUT` (default 1800): Per-iteration timeout (seconds)
- `MAX_CONCURRENT_WORKERS` (default task-defined): Concurrency limit
- `MAX_TCP_CONNECTIONS` (default 100): TCP gating threshold
- `LOOP_TCP_FILTER` (default includes `claude|anthropic|resource-claude-code`): Process/TCP filter. Empty disables gating
- `PROMPT_PATH`: Absolute path to a custom prompt
- `OLLAMA_SUMMARY_MODEL` (default `llama3.2:3b`): Used to generate `summary.txt` when available

Task-specific: None currently configured


## Selection Helpers (resource-improvement)

Located in `auto/tools/selection/`:

- `resource-candidates.sh`: Prioritizes resource candidates; merges live runtime status when available

Example:

```bash
${APP_ROOT}/auto/tools/selection/resource-candidates.sh
```


## Health and Troubleshooting

```bash
# Check binaries, data dir, and prompt discovery
${APP_ROOT}/auto/task-manager.sh --task <task> health

# Dry-run to inspect composed prompt and config
${APP_ROOT}/auto/task-manager.sh --task <task> dry-run
```

Notes:
- Logs and events auto-rotate; keep count controlled by `ROTATE_KEEP`.
- In plan mode the loop avoids network gating and side effects, focusing on diagnostics and command plans. 