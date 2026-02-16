# Ecosystem Manager Tools

## Intent

Use the ecosystem-manager CLI to create steered tasks and manage agent loop execution for scenario generation and improvement.

## Scope

**In scope:**
- Task creation with steering configuration
- Steering profile and template discovery
- Queue control (status, start, stop)
- Task monitoring and verification

**Out of scope:**
- Authoring new steering profiles or templates
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

Use this decision table to choose the right approach:

| Situation | Strategy | Example |
|-----------|----------|---------|
| New scenario, standard scope | Existing profile (`balanced` or `rapid-mvp`) | `--steer-profile balanced` |
| Scenario improvement, quality focus | Existing profile (`production-ready` or `refactor-test-focus`) | `--steer-profile production-ready` |
| UX-heavy scenario | Existing profile (`ux-excellence`) | `--steer-profile ux-excellence` |
| Custom needs, no profile fits | Steer queue with specific mode order | `--steer-queue "progress,ux,test,polish"` |

**Rule:** Prefer an existing profile when one fits. Only use `--steer-queue` when no profile matches the idea's needs.

### 3. Create the Task

```bash
# New scenario with a steering profile
ecosystem-manager task add scenario <name> --steer-profile <profile-id>

# Improve existing scenario with a steering profile
ecosystem-manager task improve scenario <name> --steer-profile <profile-id>

# Custom steering queue (when no profile fits)
ecosystem-manager task add scenario <name> --steer-queue "progress,test,refactor"
```

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

1. `ecosystem-manager task show <id>` -- confirms task exists with correct steering configuration
2. `ecosystem-manager queue status` -- confirms processor is running and will pick up the task

## Guardrails

- **Never bypass the lifecycle system.** Use `vrooli scenario start ecosystem-manager` to start the ecosystem-manager scenario itself.
- **Queue processor starts paused by default.** This is intentional for safety -- always explicitly start it after verifying task configuration.
- **Prefer existing profiles over custom queues.** Profiles are tested combinations; custom queues should be the exception.
- **One task per scenario at a time.** Do not create duplicate tasks for the same scenario.

## Command Reference

### Task Management

| Command | Description |
|---------|-------------|
| `ecosystem-manager task add scenario <name>` | Create a new scenario generation task |
| `ecosystem-manager task improve scenario <name>` | Create a scenario improvement task |
| `ecosystem-manager task list` | List all tasks (pending, in-progress, completed) |
| `ecosystem-manager task show <id>` | Show full details of a specific task |
| `ecosystem-manager task status` | Summary of task counts by status |

### Steering Discovery

| Command | Description |
|---------|-------------|
| `ecosystem-manager steer profiles` | List all available steering profiles |
| `ecosystem-manager steer templates` | List all available steering templates |
| `ecosystem-manager steer show <id>` | View details of a profile or template |

### Queue Control

| Command | Description |
|---------|-------------|
| `ecosystem-manager queue status` | Check if queue processor is running |
| `ecosystem-manager queue start` | Start the queue processor |
| `ecosystem-manager queue stop` | Stop the queue processor |

### Steering Flags

Use these flags with `task add` or `task improve`:

| Flag | Description | Example |
|------|-------------|---------|
| `--steer-profile <id>` | Use a named steering profile | `--steer-profile balanced` |
| `--steer-queue "<modes>"` | Custom comma-separated mode order | `--steer-queue "progress,test,refactor"` |
