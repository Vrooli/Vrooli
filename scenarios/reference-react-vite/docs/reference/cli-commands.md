# CLI Command Reference

This document provides a complete reference for the reference-react-vite CLI commands.

## Installation

```bash
cd scenarios/reference-react-vite/cli && ./install.sh
```

This installs the CLI to `~/.vrooli/bin/reference-react-vite`.

## Global Options

All commands support these global options:

| Option | Description |
|--------|-------------|
| `--api-base <url>` | Override API base URL (default: auto-detected) |
| `--auto-start` | Auto-start the scenario if not running |
| `--dry-run` | Validate without executing mutations |
| `--no-color` | Disable ANSI color output (or set NO_COLOR env) |
| `--color` | Force-enable ANSI color output |

## Health Commands

### status

Check API health and readiness.

```bash
reference-react-vite status
```

**Output:**
```
Status: healthy
Ready: true
Service: reference-react-vite
Version: 1.0.0
Dependencies:
  postgres: connected
```

## Task Commands

[CODE: cli/app.go#cmdTaskList]
[CODE: cli/app.go#cmdTaskCreate]

### task list

List tasks with optional filters.

```bash
reference-react-vite task list [options]
```

**Options:**
| Option | Description |
|--------|-------------|
| `--project <id>` | Filter by project ID |
| `--status <status>` | Filter by status (pending, in_progress, completed, archived) |
| `--priority <1-3>` | Filter by priority (1=low, 2=medium, 3=high) |
| `--limit <n>` | Maximum tasks to return (default: 20) |
| `--offset <n>` | Number of tasks to skip (default: 0) |
| `--json` | Output raw JSON |

**Example:**
```bash
# List all tasks
reference-react-vite task list

# List pending tasks for a project
reference-react-vite task list --project abc123 --status pending

# JSON output for scripting
reference-react-vite task list --json
```

### task get

Get a task by ID.

```bash
reference-react-vite task get <id> [--json]
```

**Example:**
```bash
reference-react-vite task get abc12345-6789-0123-4567-890123456789
```

### task create

Create a new task.

```bash
reference-react-vite task create --title <title> [options]
```

**Options:**
| Option | Description |
|--------|-------------|
| `--title <title>` | Task title (required) |
| `--description <text>` | Task description |
| `--project <id>` | Project ID to assign task to |
| `--priority <1-3>` | Priority (1=low, 2=medium, 3=high) |
| `--json` | Output raw JSON |

**Example:**
```bash
# Create a simple task
reference-react-vite task create --title "Review PR #123"

# Create task with all options
reference-react-vite task create \
  --title "Implement feature X" \
  --description "Add support for feature X" \
  --project proj-123 \
  --priority 3
```

### task update

Update an existing task.

```bash
reference-react-vite task update <id> [options]
```

**Options:**
| Option | Description |
|--------|-------------|
| `--title <title>` | New title |
| `--description <text>` | New description |
| `--status <status>` | New status (pending, in_progress, completed, archived) |
| `--priority <1-3>` | New priority |
| `--json` | Output raw JSON |

**Example:**
```bash
# Mark task as completed
reference-react-vite task update abc123 --status completed

# Update multiple fields
reference-react-vite task update abc123 --title "Updated title" --priority 1
```

### task delete

Delete a task.

```bash
reference-react-vite task delete <id>
```

**Example:**
```bash
reference-react-vite task delete abc123
```

## Project Commands

[CODE: cli/app.go#cmdProjectList]
[CODE: cli/app.go#cmdProjectCreate]

### project list

List projects with optional filters.

```bash
reference-react-vite project list [options]
```

**Options:**
| Option | Description |
|--------|-------------|
| `--status <status>` | Filter by status (active, paused, complete, archived) |
| `--limit <n>` | Maximum projects to return (default: 20) |
| `--offset <n>` | Number of projects to skip (default: 0) |
| `--json` | Output raw JSON |

**Example:**
```bash
# List all projects
reference-react-vite project list

# List active projects only
reference-react-vite project list --status active
```

### project get

Get a project by ID.

```bash
reference-react-vite project get <id> [--json]
```

### project create

Create a new project.

```bash
reference-react-vite project create --name <name> [options]
```

**Options:**
| Option | Description |
|--------|-------------|
| `--name <name>` | Project name (required) |
| `--description <text>` | Project description |
| `--color <hex>` | Project color (hex code, e.g., #FF5733) |
| `--json` | Output raw JSON |

**Example:**
```bash
# Create a simple project
reference-react-vite project create --name "Q1 Goals"

# Create project with color
reference-react-vite project create \
  --name "Marketing Campaign" \
  --description "2026 marketing initiatives" \
  --color "#FF5733"
```

### project update

Update an existing project.

```bash
reference-react-vite project update <id> [options]
```

**Options:**
| Option | Description |
|--------|-------------|
| `--name <name>` | New name |
| `--description <text>` | New description |
| `--status <status>` | New status (active, paused, complete, archived) |
| `--color <hex>` | New color (hex code) |
| `--json` | Output raw JSON |

**Example:**
```bash
# Archive a project
reference-react-vite project update proj-123 --status archived
```

### project delete

Delete a project.

```bash
reference-react-vite project delete <id>
```

## Note Commands

[CODE: cli/app.go#cmdNoteList]
[CODE: cli/app.go#cmdNoteCreate]

Notes are attached to tasks and provide annotations or comments.

### note list

List notes for a task.

```bash
reference-react-vite note list --task <task_id> [options]
```

**Options:**
| Option | Description |
|--------|-------------|
| `--task <id>` | Task ID (required) |
| `--limit <n>` | Maximum notes to return (default: 20) |
| `--offset <n>` | Number of notes to skip (default: 0) |
| `--json` | Output raw JSON |

**Example:**
```bash
reference-react-vite note list --task abc123
```

### note get

Get a note by ID.

```bash
reference-react-vite note get <id> [--json]
```

### note create

Create a new note on a task.

```bash
reference-react-vite note create --task <task_id> --content <content> [options]
```

**Options:**
| Option | Description |
|--------|-------------|
| `--task <id>` | Task ID to attach note to (required) |
| `--content <text>` | Note content (required) |
| `--author <name>` | Note author name |
| `--json` | Output raw JSON |

**Example:**
```bash
# Add a note to a task
reference-react-vite note create \
  --task abc123 \
  --content "Added unit tests for this feature" \
  --author "dev-team"
```

### note update

Update an existing note.

```bash
reference-react-vite note update <id> --content <content> [--json]
```

**Example:**
```bash
reference-react-vite note update note-123 --content "Updated note content"
```

### note delete

Delete a note.

```bash
reference-react-vite note delete <id>
```

## Configuration Commands

### configure

View or update CLI settings.

```bash
# View current settings
reference-react-vite configure

# Set API base URL
reference-react-vite configure api_base http://localhost:15001

# Set authentication token
reference-react-vite configure token my-api-token
```

Configuration is stored in `~/.vrooli/config/reference-react-vite/config.json`.

## Environment Variables

The CLI respects these environment variables (in precedence order):

| Variable | Description |
|----------|-------------|
| `REFERENCE_REACT_VITE_API_BASE` | API base URL |
| `REFERENCE_REACT_VITE_API_PORT` | API port (combined with localhost) |
| `REFERENCE_REACT_VITE_API_TOKEN` | Authentication token |
| `API_BASE_URL` | Fallback API base URL |
| `VITE_API_BASE_URL` | Fallback (shared with UI) |

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | General error (invalid arguments, API error, etc.) |

## Related Documentation

- [DOC: docs/reference/api-endpoints.md] - REST API reference
- [DOC: docs/reference/configuration.md] - Configuration options
- [DOC: docs/QUICKSTART.md] - Getting started guide
