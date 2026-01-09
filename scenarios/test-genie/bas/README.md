# Scenario Playbooks

Store BAS workflows here. Keep it short.

## Separation Of Concerns (Important)

- **`bas/cases/**`**: the Playbooks phase test cases that test-genie executes.
- **Everything else under `bas/`** (`actions/`, `flows/`, etc.): reusable workflows/subflows that cases can reference, and that you can run outside test-genie for automations.

test-genie does **not** rewrite workflows. BAS resolves scenario navigation, tokens, and subflow paths at runtime. test-genie’s job is orchestration: provision isolated DB/Redis, run seeds once, restart the scenario against those resources, then execute the `bas/cases/**` workflows.

## Workflow Contract

Workflows must use **V2 proto-JSON format**. Legacy V1 format (with `node.type` + `node.data`) and steps format (with `steps[]` array) are rejected by preflight validation.

### Required Format (V2 Proto-JSON)

```json
{
  "metadata": {
    "name": "workflow-name",
    "description": "What the workflow validates",
    "labels": {
      "reset": "none",
      "requirements_json": "[\"REQ-001\"]"
    }
  },
  "settings": {
    "viewport_width": 1440,
    "viewport_height": 900
  },
  "nodes": [
    {
      "id": "step-1",
      "action": {
        "type": "ACTION_TYPE_NAVIGATE",
        "navigate": {
          "destination_type": "NAVIGATE_DESTINATION_TYPE_SCENARIO",
          "scenario": "my-scenario",
          "scenario_path": "/",
          "wait_until": "NAVIGATE_WAIT_EVENT_NETWORKIDLE",
          "timeout_ms": 30000
        },
        "metadata": { "label": "Navigate to home" }
      }
    }
  ],
  "edges": [
    { "id": "e1", "source": "step-1", "target": "step-2", "type": "WORKFLOW_EDGE_TYPE_SMOOTHSTEP" }
  ]
}
```

### Action Types

| Action Type | Description | Key Fields |
|------------|-------------|------------|
| `ACTION_TYPE_NAVIGATE` | Navigate to URL or scenario | `navigate.destination_type`, `navigate.scenario`, `navigate.url` |
| `ACTION_TYPE_CLICK` | Click an element | `click.selector` |
| `ACTION_TYPE_INPUT` | Type text into an element | `input.selector`, `input.text` |
| `ACTION_TYPE_WAIT` | Wait for an element state | `wait.selector`, `wait.state`, `wait.timeout_ms` |
| `ACTION_TYPE_ASSERT` | Assert element condition | `assert.selector`, `assert.mode`, `assert.expected_value` |
| `ACTION_TYPE_SUBFLOW` | Call another workflow | `subflow.workflow_path`, `subflow.params` |
| `ACTION_TYPE_HTTP_REQUEST` | Make HTTP request | `http_request.method`, `http_request.url` |
| `ACTION_TYPE_SHELL` | Execute shell command | `shell.command` |

### Common Enum Values

- **Navigate destination**: `NAVIGATE_DESTINATION_TYPE_URL`, `NAVIGATE_DESTINATION_TYPE_SCENARIO`
- **Wait events**: `NAVIGATE_WAIT_EVENT_NETWORKIDLE`, `NAVIGATE_WAIT_EVENT_LOAD`, `NAVIGATE_WAIT_EVENT_DOMCONTENTLOADED`
- **Wait states**: `WAIT_STATE_VISIBLE`, `WAIT_STATE_HIDDEN`, `WAIT_STATE_ATTACHED`
- **Assertion modes**: `ASSERTION_MODE_VISIBLE`, `ASSERTION_MODE_EXISTS`, `ASSERTION_MODE_TEXT_CONTAINS`, `ASSERTION_MODE_TEXT_EQUALS`
- **Edge types**: `WORKFLOW_EDGE_TYPE_SMOOTHSTEP`

### Runtime Context

The Playbooks phase provides BAS:

- `project_root`: absolute path to the scenario's `bas/` directory (for resolving `workflowPath` relative to `bas/`)
- `initial_params`: the contents of `coverage/runtime/seed-state.json` (seeded data)

After adding or moving a case under `bas/cases/`, regenerate the registry from the scenario directory:

```bash
test-genie registry build
```

This regenerates `bas/registry.json` (tracked), which test-genie uses to determine which `bas/cases/**` files to execute.

## Playbooks isolation quickstart

- Playbooks automatically start the scenario against temporary Postgres/Redis for this phase. Seeds run once and write `coverage/runtime/seed-state.json`.
- Retain for debugging: `TEST_GENIE_PLAYBOOKS_RETAIN=1 test-genie execute my-scenario --preset comprehensive`
  - Observations will include ready-to-run `psql`/`redis-cli` commands to inspect the retained DB/Redis.
- Normal runs drop the temp resources and restart the scenario on its usual resources after Playbooks finishes.

See [Directory Structure](../../docs/phases/playbooks/directory-structure.md) for complete documentation on playbooks layout, fixtures, and naming conventions.
