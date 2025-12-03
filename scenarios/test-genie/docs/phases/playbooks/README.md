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
        BROWSERLESS[Connect Browserless<br/>Headless Chrome]
        EXECUTE[Execute Workflows<br/>Navigate, Click, Assert]
        ARTIFACTS[Collect Artifacts<br/>Screenshots, DOM snapshots]
    end

    START[Start] --> RUNTIME{Scenario<br/>Running?}
    RUNTIME -->|Yes| DISCOVER
    RUNTIME -->|No| SKIP[Skip Phase]

    DISCOVER --> REGISTRY
    REGISTRY --> BROWSERLESS
    BROWSERLESS --> EXECUTE
    EXECUTE --> ARTIFACTS
    ARTIFACTS --> DONE[Complete]

    EXECUTE -.->|assertion fails| FAIL[Fail]
    BROWSERLESS -.->|unavailable| SKIP

    style DISCOVER fill:#e8f5e9
    style REGISTRY fill:#fff3e0
    style BROWSERLESS fill:#e3f2fd
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

## Workflow Registry

Workflows are registered in `test/playbooks/registry.json`:

```json
{
  "workflows": [
    {
      "id": "create-project",
      "path": "ui/projects/create.json",
      "requirements": ["MY-PROJECT-CREATE"]
    }
  ]
}
```

## Browserless Integration

The phase uses Browserless for headless browser automation:

```bash
# Ensure Browserless is running
vrooli resource status browserless

# Start if needed
vrooli resource start browserless
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All workflows pass |
| 1 | Workflow assertions failed |
| 2 | Skipped (runtime/browserless unavailable) |

## Configuration

```json
{
  "phases": {
    "playbooks": {
      "timeout": 180,
      "browserless": {
        "endpoint": "http://localhost:3000"
      },
      "artifacts": {
        "screenshots": true,
        "domSnapshots": true
      }
    }
  }
}
```

## Related Documentation

- [UI Automation with BAS](ui-automation-with-bas.md) - Writing BAS workflows

## See Also

- [Phases Overview](../README.md) - All phases
- [Integration Phase](../integration/README.md) - Previous phase
- [Business Phase](../business/README.md) - Next phase
