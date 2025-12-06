# Playbooks Phase

**ID**: `playbooks` (also known as E2E)
**Timeout**: 120 seconds
**Optional**: Yes (when runtime not available)
**Requires Runtime**: Yes

The playbooks phase executes Browser Automation Studio (BAS) workflows for end-to-end UI testing. Workflows are declarative JSON files that automate browser interactions.

## What Gets Tested

```mermaid
graph TB
    subgraph "Playbooks Phase"
        DISCOVER[Discover Workflows<br/>test/playbooks/**/*.json]
        REGISTRY[Load Registry<br/>test/playbooks/registry.json]
        BAS[Connect BAS<br/>Browser Automation Studio]
        EXECUTE[Execute Workflows<br/>Navigate, Click, Assert]
        ARTIFACTS[Collect Artifacts<br/>Screenshots, DOM snapshots]
    end

    START[Start] --> RUNTIME{Scenario<br/>Running?}
    RUNTIME -->|Yes| DISCOVER
    RUNTIME -->|No| SKIP[Skip Phase]

    DISCOVER --> REGISTRY
    REGISTRY --> BAS
    BAS --> EXECUTE
    EXECUTE --> ARTIFACTS
    ARTIFACTS --> DONE[Complete]

    EXECUTE -.->|assertion fails| FAIL[Fail]
    BAS -.->|unavailable| SKIP

    style DISCOVER fill:#e8f5e9
    style REGISTRY fill:#fff3e0
    style BAS fill:#e3f2fd
    style EXECUTE fill:#f3e5f5
    style ARTIFACTS fill:#fff9c4
```

## Workflow Structure

Workflows are JSON files defining browser automation steps:

```json
{
  "metadata": {
    "description": "Test project creation flow",
    "version": 1
  },
  "nodes": [
    {
      "id": "navigate-home",
      "type": "navigate",
      "data": {
        "destinationType": "scenario",
        "scenario": "my-scenario",
        "scenarioPath": "/"
      }
    },
    {
      "id": "click-create",
      "type": "click",
      "data": {
        "selector": "[data-testid='create-btn']"
      }
    }
  ],
  "edges": [
    {
      "source": "navigate-home",
      "target": "click-create"
    }
  ]
}
```

## Directory Structure

Playbooks follow a canonical directory layout:

```
test/playbooks/
├── registry.json       # Auto-generated manifest
├── capabilities/       # Feature tests (mirrors PRD)
│   ├── 01-foundation/  # Two-digit prefix for ordering
│   └── 02-builder/
├── journeys/           # Multi-surface user flows
├── __subflows/         # Reusable fixtures
└── __seeds/            # Setup/cleanup scripts
```

Key conventions:
- **Two-digit prefixes** (`01-`, `02-`) ensure deterministic execution order
- **`__subflows/`** contains fixtures referenced via `@fixture/<slug>`
- **`__seeds/`** contains `apply.sh` and `cleanup.sh` for test data

See [Directory Structure](directory-structure.md) for complete documentation including fixture metadata, token types, and authoring checklist.

## Workflow Registry

The registry (`test/playbooks/registry.json`) is **auto-generated** and tracks all playbooks:

```json
{
  "_note": "AUTO-GENERATED — run 'test-genie registry build' to refresh",
  "scenario": "my-scenario",
  "generated_at": "2025-12-05T10:00:00Z",
  "playbooks": [
    {
      "file": "test/playbooks/capabilities/01-foundation/create-project.json",
      "description": "Creates a new project",
      "order": "01.01",
      "requirements": ["MY-PROJECT-CREATE"],
      "fixtures": [],
      "reset": "full"
    }
  ]
}
```

Regenerate after adding or moving playbooks:

```bash
test-genie registry build
```

## Browser Automation Studio Integration

The phase uses Browser Automation Studio (BAS) for workflow execution. BAS is a separate Vrooli scenario that provides a Playwright-based browser automation engine.

```bash
# Ensure BAS is running
vrooli scenario status browser-automation-studio

# Start if needed
vrooli scenario start browser-automation-studio
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All workflows pass |
| 1 | Workflow assertions failed |
| 2 | Skipped (runtime/BAS unavailable) |

## Configuration

Configure the playbooks phase in `.vrooli/testing.json`:

```json
{
  "playbooks": {
    "enabled": true,
    "bas": {
      "endpoint": "http://localhost:8080/api/v1",
      "timeout_ms": 30000,
      "launch_timeout_ms": 60000
    },
    "execution": {
      "stop_on_first_failure": false,
      "default_step_timeout_ms": 30000,
      "ignore_validation_errors": false,
      "dry_run": false
    },
    "artifacts": {
      "screenshots": true,
      "dom_snapshots": true
    }
  },
  "phases": {
    "playbooks": {
      "timeout": "3m"
    }
  }
}
```

### Execution Options

| Option | Default | Description |
|--------|---------|-------------|
| `ignore_validation_errors` | `false` | When BAS validation request fails (e.g., BAS unavailable), continue with execution instead of failing. **Not recommended for production.** |
| `dry_run` | `false` | Validate all workflows without executing them. Useful for CI pre-checks or debugging workflow syntax. Skips seed scripts and scenario startup. |

### Dry-Run Mode

Dry-run mode validates workflows through BAS without executing browser automation:

```bash
# Enable via testing.json or environment
# In testing.json:
{
  "playbooks": {
    "execution": {
      "dry_run": true
    }
  }
}
```

In dry-run mode:
- Workflows are loaded, resolved, and validated via BAS
- Seed scripts (`__seeds/apply.sh`) are **not** run
- Required scenarios (referenced via `destinationType: scenario`) are **not** started
- Execution is skipped after successful validation
- Each workflow returns with `(dry-run: validated only)` in its stats

## Related Documentation

- [Directory Structure](directory-structure.md) - Canonical layout, fixtures, seeds, naming conventions
- [UI Automation with BAS](ui-automation-with-bas.md) - Writing workflow JSON, node types, selectors

## See Also

- [Phases Overview](../README.md) - All phases
- [Integration Phase](../integration/README.md) - Previous phase
- [Business Phase](../business/README.md) - Next phase
