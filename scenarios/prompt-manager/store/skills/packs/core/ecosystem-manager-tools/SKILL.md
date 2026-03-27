# Ecosystem Manager Tools

## Intent

Use the ecosystem-manager CLI to create steered tasks and manage agent loop execution for scenario generation and improvement.

## Scope

**In scope:**
- Task creation with steering configuration
- Steering profile and template discovery
- Queue control (status, start, stop)
- Task monitoring and verification
- Task deletion
- Log viewing and CLI configuration

**Out of scope:**
- Authoring new steering profiles or templates (profile CRUD is currently API/UI-only)
- Auto Steer execution internals
- Direct scenario code implementation (that is handled by ecosystem-manager's agent loops)

## Core Workflow

### 1. Discover Available Steering

Before creating a task, discover what steering options exist:

```bash
# List all steering profiles (curated combinations of modes)
ecosystem-manager steer profiles

# List all steering templates (reusable queue patterns)
ecosystem-manager steer templates

# View details of a specific profile or template (resolved by ID or slug)
ecosystem-manager steer show <id>
```

### 2. Select a Steering Strategy

**Profiles vs Templates:** The system has two kinds of steering configurations:
- **Templates** are built-in presets with slug IDs (e.g., `balanced`, `rapid-mvp`). List with `steer templates`.
- **Profiles** are user-created multi-phase configurations with UUID IDs. List with `steer profiles`.

The `--steer-profile` flag accepts both template slugs and profile UUIDs.

Use this decision table to choose the right approach:

| Situation | Strategy | Example |
|-----------|----------|---------|
| New scenario, standard scope | Built-in template (`balanced` or `rapid-mvp`) | `--steer-profile balanced` |
| Scenario improvement, quality focus | Built-in template (`production-ready` or `refactor-test-focus`) | `--steer-profile production-ready` |
| UX-heavy scenario | Built-in template (`ux-excellence`) | `--steer-profile ux-excellence` |
| Custom multi-phase workflow | User-created profile (UUID from `steer profiles`) | `--steer-profile <uuid>` |
| Improvement with specific steer mode | Single steer mode (scenario improver tasks only) | `--steer-mode "test"` |
| Improvement with ordered mode list | Steering queue (scenario improver tasks only) | `--steer-queue "progress,test,refactor"` |
| No steering needed (manual management) | No steering flag | `ecosystem-manager task add scenario <name>` |

**Rules:**
- Prefer a built-in template when one fits.
- `--steer-mode` only works on **scenario improver** tasks (`task improve scenario <name>`). It accepts a single valid mode name (e.g. `"test"`).
- `--steer-queue` only works on **scenario improver** tasks. It accepts a comma-separated list of valid mode names (e.g. `"progress,test,refactor"`). The modes execute in order.
- For generator tasks (`task add`), use `--steer-profile` exclusively.

### 3. Create the Task

The type prefix (`resource` or `scenario`) is required.

```bash
# New scenario with a steering profile
ecosystem-manager task add --steer-profile <profile-id> scenario <name>

# New resource with a steering profile
ecosystem-manager task add --steer-profile <profile-id> resource <name>

# Improve existing scenario with a steering profile
ecosystem-manager task improve --steer-profile <profile-id> scenario <name>

# Improve with a specific steer mode (improver tasks only)
ecosystem-manager task improve --steer-mode "<mode>" scenario <name>

# Improve with an ordered list of steer modes (improver tasks only)
ecosystem-manager task improve --steer-queue "progress,test,refactor" scenario <name>
```

> **Note:** `task improve` requires the target scenario or resource to already exist on disk. `task add` (generators) does not — the target will be created.

**Optional flags for task creation:**
- `--priority <level>`: Set task priority (`low`, `medium`, `high`, `critical`). Default: `medium`.
- `--category <name>`: Category for generator tasks (`task add` only). Default: `general`.
- `--notes <text>` / `--notes-file <path>`: Persist task notes that become prompt context on every loop.
- `--origin-source <name>`: Record the upstream system that created the task.
- `--origin-backlog-item <kind/name>`: Record the upstream backlog item reference.
- `--origin-item-folder <abs-path>`: Record the absolute path to the upstream item folder.
- `--handoff-dir <abs-path>`: Record and validate an upstream handoff package. The CLI derives `brief.md`, `manifest.json`, and `source-index.json` from this directory and auto-loads `brief.md` into task notes when notes were not provided explicitly.

For swarm-manager idea execution, the standard task-creation pattern is:

```bash
ecosystem-manager task add --steer-profile <profile-id> \
  --handoff-dir "<item-folder>/handoff" \
  --origin-source swarm-manager \
  --origin-backlog-item "idea/<item-name>" \
  --origin-item-folder "<item-folder>" \
  scenario <name>
```

> **Tip:** Most commands support `--json` for machine-readable output. Use it when you need to parse or pipe results.

### 4. List and Filter Tasks

```bash
# List all tasks
ecosystem-manager task list

# Filter by status, type, or operation
ecosystem-manager task list --status pending
ecosystem-manager task list --type scenario --operation improver
```

Use filtering to check for existing tasks before creating new ones (see "One task per scenario" guardrail).

### 5. Verify Task Creation

```bash
# Confirm the task was created with correct steering
ecosystem-manager task show <task-id>
```

### 6. Update Task Status (optional)

```bash
# Manually transition a task's status (e.g., pause or reset a stuck task)
ecosystem-manager task status <task-id> --status <new-status>
```

### 7. Ensure Queue is Running

```bash
# Check if the queue processor is active
ecosystem-manager queue status

# Start the processor if paused
ecosystem-manager queue start
```

The queue processor starts paused by default for safety. Always check and start it after creating tasks.

### 8. Monitor Logs (optional)

```bash
# View recent API logs
ecosystem-manager logs

# Follow logs in real-time
ecosystem-manager logs -f
```

## Verification

After completing the workflow:

1. `ecosystem-manager task show <id>` -- confirms task exists with type, operation, priority, and steer mode
2. `ecosystem-manager task show <id> --json` -- use JSON output to verify the full steering configuration including auto-steer profile ID
3. `ecosystem-manager task show <id> --json` -- when using upstream handoff context, also verify `notes` plus the `origin.*` fields
4. `ecosystem-manager queue status` -- confirms processor is running and will pick up the task

## Guardrails

- **Never bypass the lifecycle system.** Use `vrooli scenario start ecosystem-manager` to start the ecosystem-manager scenario itself.
- **Queue processor starts paused by default.** This is intentional for safety -- always explicitly start it after verifying task configuration.
- **Prefer existing profiles over custom modes.** Profiles are tested combinations; custom modes should be the exception.
- **One task per scenario at a time.** Use `ecosystem-manager task list --type scenario` to check for existing tasks before creating new ones.
- **Use `--dry-run` for validation.** Pass `--dry-run` to validate task creation without persisting. The CLI shows `[DRY RUN]` in output. The output includes a generated task ID and `View:` command, but since nothing is persisted, running that command returns a 404 — this is expected.

## Command Reference

Run `ecosystem-manager help` for the full command list. Each group supports `help`:

```bash
ecosystem-manager task help
ecosystem-manager steer help
ecosystem-manager queue help
```

Global flags (`--dry-run`, `--api-base`, `--auto-start`, `--no-color`, `--color`) work with all commands.

## Troubleshooting

Error messages now include inline recovery hints with suggested commands. The table below provides additional context:

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| Queue processor not picking up tasks | Processor starts paused by default | Run `ecosystem-manager queue start` (shown in CLI next-steps) |
| `Error: api error (404)` on `status` | API health endpoint not at expected path | Check `ecosystem-manager queue status` for connectivity |
| `Error: ... Manual steering ... only supported for improver tasks` | `--steer-mode` used with generator task or resource improver | Use `--steer-profile` instead, or use `task improve scenario <name>` |
| `Error: ... Steering queue ... only supported for scenario improver tasks` | `--steer-queue` used with generator task or non-scenario target | Use `--steer-profile` instead, or use `task improve scenario <name>` |
| `Error: Invalid mode in steering_queue` | One or more modes in the queue are not valid steer modes | Check valid modes with `ecosystem-manager steer templates` and fix the queue list |
| Task already exists for this scenario (code: `duplicate_task`) | Duplicate task for same target | CLI recovery hint shows `task show <existing-id>` command |
| Target not found (code: `target_not_found`) | Improver target doesn't exist on disk | Use `task add` for new targets, or verify the target path |
| Profile not found (code: `profile_not_found`) | Invalid `--steer-profile` value | CLI recovery hint shows `steer profiles` and `steer templates` commands |
