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

# View details of a specific profile or template
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
| Improvement with specific steer mode | Single steer mode (improver tasks only) | `--steer-mode "test"` |

**Rules:**
- Prefer a built-in template when one fits.
- `--steer-mode` / `--steer-queue` only work on **improver** tasks (not generators). They accept a single valid mode name, not a comma-separated list.
- For generator tasks (`task add`), use `--steer-profile` exclusively.

### 3. Create the Task

The type prefix (`resource` or `scenario`) is required. **Flags must appear before positional arguments** (Go's flag parser silently ignores flags after positional args).

```bash
# New scenario with a steering profile
ecosystem-manager task add --steer-profile <profile-id> scenario <name>

# New resource with a steering profile
ecosystem-manager task add --steer-profile <profile-id> resource <name>

# Improve existing scenario with a steering profile
ecosystem-manager task improve --steer-profile <profile-id> scenario <name>

# Improve with a specific steer mode (improver tasks only)
ecosystem-manager task improve --steer-mode "<mode>" scenario <name>
```

> **Note:** `task improve` requires the target scenario or resource to already exist in the system. The API validates this even with `--dry-run`, since dry-run runs the full validation path before returning a preview response.

### 4. Verify Task Creation

```bash
# Confirm the task was created with correct steering
ecosystem-manager task show <task-id>
```

### 5. Ensure Queue is Running

```bash
# Check if the queue processor is active
ecosystem-manager queue status

# Start the processor if paused
ecosystem-manager queue start
```

The queue processor starts paused by default for safety. Always check and start it after creating tasks.

## Verification

After completing the workflow:

1. `ecosystem-manager task show <id>` -- confirms task exists with type, operation, priority, and steer mode
2. `ecosystem-manager task show <id> --json` -- use JSON output to verify the full steering configuration including auto-steer profile ID
3. `ecosystem-manager queue status` -- confirms processor is running and will pick up the task

> **Note:** `--dry-run` IDs are ephemeral and cannot be used with `task show`. Dry-run validates the full request (including target existence for improver tasks) but only confirms the request *would* succeed.

## Guardrails

- **Never bypass the lifecycle system.** Use `vrooli scenario start ecosystem-manager` to start the ecosystem-manager scenario itself.
- **Queue processor starts paused by default.** This is intentional for safety -- always explicitly start it after verifying task configuration.
- **Prefer existing profiles over custom queues.** Profiles are tested combinations; custom modes should be the exception.
- **One task per scenario at a time.** Do not create duplicate tasks for the same scenario.
- **Use `--dry-run` for validation.** Pass `--dry-run` to validate task creation without persisting. The CLI shows `[DRY RUN]` in output.

## Command Reference

### Task Management

| Command | Description |
|---------|-------------|
| `ecosystem-manager task add [resource\|scenario] <name>` | Create a new generation task |
| `ecosystem-manager task improve [resource\|scenario] <name>` | Create an improvement task (target must exist) |
| `ecosystem-manager task list` | List all tasks (pending, in-progress, completed) |
| `ecosystem-manager task show <id>` | Show full details of a specific task |
| `ecosystem-manager task status <id> --status <S>` | Update a task's status |
| `ecosystem-manager task status <id> --phase <P>` | Update a task's current phase |
| `ecosystem-manager task delete <id>` | Delete a task |

Aliases: `task ls` = `task list`, `task get` = `task show`, `task rm` = `task delete`, `tasks` = `task`

#### Task List Filters

| Flag | Description |
|------|-------------|
| `--status <S>` | Filter by status (pending, in-progress, completed) |
| `--type <T>` | Filter by type (resource, scenario) |
| `--operation <O>` | Filter by operation (generator, improver) |
| `--json` | Output as JSON |

### Steering Discovery

| Command | Description |
|---------|-------------|
| `ecosystem-manager steer profiles` | List all available steering profiles |
| `ecosystem-manager steer templates` | List all available steering templates |
| `ecosystem-manager steer show <id>` | View details of a profile or template |

Aliases: `steer ls` = `steer profiles`, `steer get` = `steer show`

### Queue Control

| Command | Description |
|---------|-------------|
| `ecosystem-manager queue status` | Check if queue processor is running |
| `ecosystem-manager queue start` | Start the queue processor |
| `ecosystem-manager queue stop` | Stop the queue processor |

### Logs

| Command | Description |
|---------|-------------|
| `ecosystem-manager logs` | View API logs (default: last 50 lines) |
| `ecosystem-manager logs -f` | Follow log output in real time |
| `ecosystem-manager logs -n <count>` | Show specific number of lines |

### Configuration

| Command | Description |
|---------|-------------|
| `ecosystem-manager configure` | View or update CLI settings (api_base, token) |

### Steering Flags

Use these flags with `task add` or `task improve`. Flags must appear **before** positional arguments:

| Flag | Description | Example |
|------|-------------|---------|
| `--steer-profile <id>` | Use a named steering profile (works on all tasks) | `--steer-profile balanced` |
| `--steer-queue "<mode>"` | Single steer mode (improver tasks only) | `--steer-queue "test"` |
| `--steer-mode "<mode>"` | Alias for `--steer-queue` (improver tasks only) | `--steer-mode "progress"` |
| `--priority <P>` | Set task priority (low, medium, high, critical) | `--priority high` |
| `--category <C>` | Set task category (used with `task add`) | `--category general` |
| `--json` | Output as JSON | `--json` |

### Global Flags

These are provided by cli-core and work with all commands:

| Flag | Description |
|------|-------------|
| `--dry-run` | Validate without executing mutations |
| `--api-base <url>` | Override API endpoint |
| `--auto-start` | Start scenario if API unavailable |
| `--no-color` | Disable ANSI color output |
| `--color` | Force-enable ANSI color output |

## Troubleshooting

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| Queue processor not picking up tasks | Processor starts paused by default | Run `ecosystem-manager queue start` |
| `Error: api error (404)` on `status` | API health endpoint not at expected path | Check `ecosystem-manager queue status` for connectivity |
| Flags ignored (e.g. `--json`, `--steer-profile`) | Flag placed after positional args | Move flags before positional args: `task add --steer-profile balanced scenario <name>` |
| `Manual steering is only supported for scenario improver tasks` | Used `--steer-queue` or `--steer-mode` on a generator task | Use `--steer-profile` instead, or switch to `task improve` |
| `Improver tasks require at least one target` | Target scenario/resource doesn't exist | Verify the scenario exists in `scenarios/` directory before running `task improve` |
