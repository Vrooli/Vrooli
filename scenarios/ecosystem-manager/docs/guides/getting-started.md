# Getting Started Guide

## Prerequisites

1. Vrooli workspace initialized.
2. Scenario dependencies healthy (`agent-manager`, `postgres`).

## First Workflow

1. Start the scenario: `make start`.
2. Create a low-risk test task.
3. Observe task transitions in the board and logs.
4. Verify output artifacts in queue storage.

## Validation Commands

```bash
vrooli scenario test ecosystem-manager
ecosystem-manager --help
ecosystem-manager version
```

[DOC: docs/reference/api-endpoints.md]
[DOC: docs/reference/cli-commands.md]
