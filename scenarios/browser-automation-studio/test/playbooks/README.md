# Browser Automation Studio Integration Playbooks

This directory contains workflow JSON playbooks for automated integration testing of the Browser Automation Studio UI and functionality.

## Directory Structure

```
test/playbooks/
├── ui/                          # UI user flow tests
│   ├── projects/                # Project management flows
│   │   ├── new-project-*.json   # Project creation workflows
│   │   ├── project-edit.json    # Project editing
│   │   ├── project-list-and-select.json
│   │   └── project-search-filter.json
│   ├── workflows/               # Workflow management flows
│   │   ├── workflow-create-and-load.json
│   │   ├── workflow-select-from-list.json
│   │   └── workflow-version-restore.json
│   └── executions/              # Execution viewing and triggering
│       ├── execution-history-view.json
│       └── execution-from-project-detail.json
├── __subflows/                  # Reusable workflow fragments (setup/preludes)
│   └── ...                      # Require metadata.fixture_id
├── __seeds/                     # Seed data scripts supporting playbooks
│   ├── apply.sh                 # Applies deterministic data fixtures (optional)
│   └── cleanup.sh               # Cleans up data seeded by apply.sh
├── ai/                          # AI-assisted workflow generation
│   └── generated-smoke.json
├── executions/                  # CLI execution and telemetry
│   ├── telemetry-smoke.json
│   └── heartbeat-stall.json
├── projects/                    # Project-level operations
│   └── demo-sanity.json
└── replay/                      # Replay rendering and export
    └── render-check.json
```

## Naming Conventions

### File Naming
- Use **kebab-case** for all filenames
- Prefix with feature area: `project-*`, `workflow-*`, `execution-*`
- Include action verb: `*-create`, `*-edit`, `*-view`, `*-select`, `*-search`
- Examples:
  - ✅ `project-edit.json`
  - ✅ `workflow-select-from-list.json`
  - ✅ `execution-history-view.json`
  - ❌ `ProjectEdit.json`
  - ❌ `edit.json`

### Node IDs
- Use descriptive, lowercase-with-hyphens
- Include node type prefix: `wait-*`, `click-*`, `assert-*`, `navigate-*`
- Examples:
  - `wait-for-dashboard`
  - `click-project-card`
  - `assert-modal-visible`
  - `navigate-to-workflow`

### Fixture IDs
- Store reusable workflow fragments in `__subflows/` and provide a `metadata.fixture_id`
- Use descriptive kebab-case slugs: `open-builder-from-demo`, `open-demo-project`
- Reference fixtures from playbooks with `"workflowId": "@fixture/<slug>"`

## Playbook Structure

Each playbook JSON must include:

### Required Metadata
```json
{
  "metadata": {
    "description": "Clear description of what this workflow validates",
    "requirement": "BAS-REQUIREMENT-ID",  // Links to requirements/
    "version": 1
  }
}
```

### Settings
```json
{
  "settings": {
    "executionViewport": {
      "width": 1920,          // Standard desktop resolution
      "height": 1080,
      "preset": "desktop"
    }
  }
}
```

### Nodes Array
Each node must have:
- `id` - Unique identifier (kebab-case)
- `type` - Node type (navigate, wait, click, type, assert, evaluate, screenshot)
- `position` - {x, y} coordinates for visual layout
- `data` - Configuration object with:
  - `label` - Human-readable description
  - Type-specific fields (selector, text, expression, etc.)

### Edges Array
Defines workflow execution order:
```json
{
  "edges": [
    {
      "id": "e1",
      "source": "node-id-1",
      "target": "node-id-2",
      "type": "smoothstep"
    }
  ]
}
```

## Reusable Subflows (`__subflows/`)

- Keep shared setup/prelude workflows under `test/playbooks/__subflows/`
- Each file must include `metadata.fixture_id` with a unique slug
- Reference a subflow from a playbook by setting `workflowId` to `@fixture/<slug>`
- The test runner automatically inlines the referenced workflow definition before linting and execution
- Structural tests fail when a fixture is missing metadata, duplicated, unused, or referenced by a playbook that doesn't exist

### Provided Fixtures

| Fixture ID | Description |
| --- | --- |
| `open-demo-project` | Navigates to the dashboard, selects **Demo Browser Automations**, and verifies the project detail page is ready for assertions. |
| `open-builder-from-demo` | Calls `open-demo-project`, clicks **New Workflow**, bypasses the AI modal, and waits for the builder canvas to render. |
| `open-demo-workflow` | Reuses `open-demo-project`, opens the seeded "Demo: Capture Example.com Hero" workflow, and waits until its nodes render inside the builder. |

Use these fixtures whenever a workflow needs to begin from a known dashboard or builder state instead of repeating the navigation boilerplate.

## Selector Registry

- `ui/src/consts/selectors.ts` is the canonical list of every `data-testid` in the BAS UI. Components import from that file and the workflow tooling resolves selectors against it.
- Reference selectors inside playbooks with the `@selector/<key>` token (for example `@selector/projects.grid`) instead of hardcoding `[data-testid="..."]`. The resolver replaces the token with the attribute selector before linting/execution.
- Dynamic selectors are exposed with parameterized tokens such as `@selector/executions.filters.filter(filter=failed)` or `@selector/ai.modal.promptExample(index=0)`. Definitions live in `ui/src/consts/selectors.ts`, so both the UI and automation flows stay aligned even for selectors that embed runtime data (indexes, filter names, etc.).
- Convenience helpers like `@selector/projects.cardByName(name=Demo Browser Automations)` and `@selector/workflows.cardByName(name=Demo: Capture Example.com Hero)` expand into attribute selectors that match list items by their data attributes, so the workflows stay expressive without repeating brittle CSS.
- Example:

```json
{
  "id": "assert-projects-visible",
  "type": "assert",
  "data": {
    "label": "Projects grid is visible",
    "selector": "@selector/projects.grid",
    "assertMode": "exists"
  }
}
```

- If a selector key is missing, add it to `ui/src/consts/selectors.ts` (and run Prettier). The workflow resolver and validator will raise descriptive errors pointing back to that file when an unknown selector is referenced.

Example fixture:

```json
{
  "metadata": {
    "description": "Loads the builder and selects the Demo project",
    "fixture_id": "open-builder-from-demo"
  },
  "nodes": [ ... ],
  "edges": [ ... ]
}
```

Example usage in a playbook node:

```json
{
  "id": "call-builder-setup",
  "type": "workflowCall",
  "data": {
    "label": "Reuse builder prelude",
    "workflowId": "@fixture/open-builder-from-demo"
  }
}
```

## Seed Data (`__seeds/`)

- Optional `test/playbooks/__seeds/apply.sh` can prepare deterministic data (projects, workflows, etc.) before integration tests run
- Provide a matching `cleanup.sh` to remove any state created by the apply script; the integration harness runs cleanup even on failure
- Scripts execute from the scenario root, so you can call the BAS CLI, Go utilities, or direct HTTP requests
- The bundled `apply.sh` wipes all projects through the API, recreates the **Demo Browser Automations** project under `data/projects/demo`, and seeds a simple workflow so list/select flows always have at least one entry. `cleanup.sh` removes every project at phase end so the next run starts fresh.
- Example `apply.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
browser-automation-studio workflow create "Seed Login" \
  --folder /__seeds \
  --from-file test/playbooks/__seeds/workflows/login.json
```

The apply/cleanup scripts ensure Browser Automation Studio returns to its original state after every phase, so fixtures never leak into normal usage.

## Node Types Reference

### Navigate
```json
{
  "type": "navigate",
  "data": {
    "label": "Open BAS dashboard",
    "destinationType": "scenario",
    "scenario": "browser-automation-studio",
    "scenarioPath": "/",
    "waitUntil": "networkidle0",
    "timeoutMs": 45000,
    "waitForMs": 2000
  }
}
```

### Wait
```json
{
  "type": "wait",
  "data": {
    "label": "Wait for projects grid",
    "selector": "@selector/projects.grid",
    "waitType": "element",      // "element" or "duration"
    "timeoutMs": 10000,
    "waitForMs": 1000
  }
}
```

**Important**: Always set `waitType` to either `"element"` or `"duration"`. Never leave it empty.

### Click
```json
{
  "type": "click",
  "data": {
    "label": "Click Demo project",
    "selector": "@selector/projects.cardByName(name=Demo Browser Automations)",
    "timeoutMs": 5000,
    "waitForMs": 2000
  }
}
```

### Type
```json
{
  "type": "type",
  "data": {
    "label": "Enter project name",
    "selector": "@selector/dialogs.project.nameInput",
    "text": "Test Project",
    "clearFirst": true,
    "timeoutMs": 5000,
    "waitForMs": 300
  }
}
```

### Assert
```json
{
  "type": "assert",
  "data": {
    "label": "Modal should be visible",
    "selector": "@selector/dialogs.project.root",
    "assertMode": "exists",     // "exists", "not_exists", "text_contains"
    "timeoutMs": 5000,
    "failureMessage": "Project modal should appear"
  }
}
```

### Evaluate
```json
{
  "type": "evaluate",
  "data": {
    "label": "Store project name",
    "expression": "document.querySelector('h1')?.textContent?.trim() || 'Unknown'",
    "storeAs": "projectName"   // Variable name for later use
  }
}
```

### Screenshot
```json
{
  "type": "screenshot",
  "data": {
    "label": "Capture final state",
    "fullPage": true,
    "waitForMs": 500
  }
}
```

## Best Practices

### Selector Strategy
1. **Prefer selector tokens** (most reliable)
   ```json
   "selector": "@selector/projects.card"
   ```

2. **Use semantic HTML when possible**
   ```json
   "selector": "h1"
   "selector": "button[aria-label=\"Submit\"]"
   ```

3. **Avoid CSS classes** (prone to styling changes)
   ```json
   "selector": ".btn-primary"  // ❌ Fragile
   ```

4. **Use dynamic selectors for parameterized cases**
   ```json
   "selector": "@selector/workflows.cardByName(name=Demo: Capture Example.com Hero)"
   ```

### Wait Strategy
- Always wait after navigation: `waitForMs: 2000`
- Wait for dynamic content to load before assertions
- Use `waitType: "element"` for UI elements
- Use `waitType: "duration"` for animations/transitions

### Variable Usage
Store dynamic values for later steps:
```json
{
  "type": "evaluate",
  "data": {
    "expression": "`Project ${Date.now()}`",
    "storeAs": "uniqueName"
  }
},
{
  "type": "type",
  "data": {
    "text": "{{uniqueName}}",  // Reference stored variable
    "selector": "input"
  }
}
```

### Assertions
- Add assertions at key checkpoints
- Use descriptive failure messages
- Verify both positive and negative cases
- Screenshot after critical assertions

## Requirement Linking

Each playbook must be referenced in a requirement file under `requirements/`:

```json
{
  "id": "BAS-PROJECT-EDIT",
  "validation": [
    {
      "type": "automation",              // ✅ Required for integration phase
      "ref": "test/playbooks/ui/projects/project-edit.json",
      "phase": "integration",
      "status": "implemented",
      "notes": "Validates project editing flow"
    }
  ]
}
```

**Important**: Use `"type": "automation"` (not `"test"`) for playbooks to be executed in integration phase.

## Running Playbooks

### Via Integration Tests
```bash
cd scenarios/browser-automation-studio
bash test/phases/test-integration.sh
```

### Individual Playbook (for debugging)
```bash
# Via workflow runner helper
testing::playbooks::run_workflow \
  --file test/playbooks/ui/projects/project-edit.json \
  --scenario browser-automation-studio \
  --manage-runtime auto
```

### Via CLI (if scenario is already running)
```bash
browser-automation-studio execution create \
  --file test/playbooks/ui/projects/project-edit.json \
  --wait
```

## Troubleshooting

### Playbook Not Executing
1. **Check requirement link**: Verify playbook is referenced in requirements/*.json
2. **Check validation type**: Must be `"type": "automation"` for integration tests
3. **Check file path**: Path in requirement must match actual file location
4. **Check imports**: Requirement file must be imported in requirements/index.json

### Execution Failures
1. **Selector not found**:
   - Add `data-testid` attribute to UI component
   - Check if element is rendered conditionally
   - Increase timeout or add wait node before click/assert

2. **Timing issues**:
   - Add `waitForMs` after navigation/clicks
   - Use wait nodes for dynamic content
   - Increase `timeoutMs` for slow operations

3. **Assert failures**:
   - Check selector specificity
   - Verify element visibility (not hidden/display:none)
   - Add screenshot node before assertion for debugging

### Common Errors
- `wait node has unsupported type ""` → Set `waitType` to `"element"` or `"duration"`
- `Selector not found` → Add data-testid to component or increase timeout
- `Workflow not discovered` → Check requirement file is imported in index.json

## Examples

### Complete Playbook Example
See `test/playbooks/ui/projects/new-project-create.json` for a comprehensive example with:
- Navigation
- Form filling
- Validation
- Assertions
- Screenshots

### Minimal Playbook Example
```json
{
  "metadata": {
    "description": "Verify dashboard loads",
    "requirement": "BAS-DASHBOARD-LOAD",
    "version": 1
  },
  "settings": {
    "executionViewport": {
      "width": 1920,
      "height": 1080,
      "preset": "desktop"
    }
  },
  "nodes": [
    {
      "id": "navigate",
      "type": "navigate",
      "position": {"x": 0, "y": 0},
      "data": {
        "label": "Navigate to dashboard",
        "destinationType": "scenario",
        "scenario": "browser-automation-studio",
        "scenarioPath": "/",
        "waitUntil": "networkidle0",
        "timeoutMs": 30000
      }
    },
    {
      "id": "wait",
      "type": "wait",
      "position": {"x": 200, "y": 0},
      "data": {
        "label": "Wait for projects grid",
        "selector": "@selector/projects.grid",
        "waitType": "element",
        "timeoutMs": 10000
      }
    },
    {
      "id": "assert",
      "type": "assert",
      "position": {"x": 400, "y": 0},
      "data": {
        "label": "Projects grid exists",
        "selector": "@selector/projects.grid",
        "assertMode": "exists",
        "failureMessage": "Dashboard should show projects grid"
      }
    }
  ],
  "edges": [
    {"id": "e1", "source": "navigate", "target": "wait", "type": "smoothstep"},
    {"id": "e2", "source": "wait", "target": "assert", "type": "smoothstep"}
  ]
}
```

## Contributing

### Adding New Playbooks
1. Create JSON file in appropriate subdirectory
2. Follow naming conventions
3. Include complete metadata with requirement ID
4. Add comprehensive assertions
5. Add reference to corresponding requirement file
6. Test locally before committing
7. Update this README if adding new patterns

### Modifying Existing Playbooks
1. Test changes locally first
2. Update `version` number in metadata
3. Update requirement notes if behavior changes
4. Re-run integration tests to verify

## Related Documentation

- [Requirements Tracking](../../requirements/README.md) - How playbooks link to requirements
- [Integration Test Guide](../../docs/testing/integration.md) - Running integration tests
- [API Documentation](../../docs/api/README.md) - Execute-adhoc endpoint reference
- [INTEGRATION_TEST_IMPROVEMENTS.md](../../INTEGRATION_TEST_IMPROVEMENTS.md) - Recent improvements
