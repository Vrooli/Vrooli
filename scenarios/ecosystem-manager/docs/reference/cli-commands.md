# CLI Commands

## Core
- `ecosystem-manager --help`
- `ecosystem-manager version`

## Task and Queue Operations
- `ecosystem-manager tasks ...`
- `ecosystem-manager queue ...`
- `ecosystem-manager steer ...`

## Behavior Notes
- Unknown arguments are rejected with non-zero exit status.
- Wrapper delegates to compiled CLI binary when available.

[CODE: cli/main.go]
[CODE: cli/tasks/commands.go]
[CODE: cli/queue/commands.go]
[CODE: cli/steer/commands.go]
[CODE: cli/install.sh]
