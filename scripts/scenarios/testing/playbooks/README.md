# Generic Workflow Runner

This directory contains the generic workflow runner that allows **any scenario** to define Browser Automation Studio workflow JSON files and have them automatically executed for UI/integration testing.

## Overview

The `workflow-runner.sh` script provides a universal interface for running BAS workflows from any scenario's test suite. It handles:
- Workflow execution via browser-automation-studio API
- Runtime workflow validation (schema, structure, selectors)
- Scenario runtime management (auto-start/stop)
- Result tracking and requirement evidence recording
- Both file-based and database-persisted workflow execution

**Note:** Workflow validation happens at runtime during execution, not as a separate linting step. This ensures validation uses the same code path as actual execution and avoids timing issues with seed data.

## Files

- `workflow-runner.sh` - Main workflow execution engine (generic, works for all scenarios)

## Usage

### From Test Phase Scripts

The recommended way to use the workflow runner is through the phase helper wrapper:

```bash
#!/bin/bash
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

# Run all workflow validations for this scenario
testing::phase::run_workflow_validations \
  --scenario "${TESTING_PHASE_SCENARIO_NAME}" \
  --manage-runtime auto
```

### Direct Usage

You can also call the runner directly for custom scenarios:

```bash
source "${APP_ROOT}/scripts/scenarios/testing/playbooks/workflow-runner.sh"

testing::playbooks::run_workflow \
  --file test/workflows/my-test.json \
  --scenario my-scenario \
  --manage-runtime auto
```

### Options

#### `testing::phase::run_workflow_validations`
- `--scenario NAME` - Target scenario to test (default: browser-automation-studio)
- `--manage-runtime MODE` - Runtime management: `auto`, `start`, `skip` (default: auto)
- `--disallow-missing` - Treat missing workflow files as failures instead of skips

#### `testing::playbooks::run_workflow`
- `--file PATH` - JSON workflow definition (required)
- `--scenario NAME` - Scenario to execute against (default: browser-automation-studio)
- `--manage-runtime MODE` - Auto start/stop scenario: `auto`, `start` (1/true), `skip` (0/false)
- `--timeout SECONDS` - Max wait for scenario readiness (default: 120)
- `--allow-missing` - Return exit code 210 when workflow file missing (for optional tests)

> **Seed Data**: BAS integration tests apply `test/playbooks/__seeds/apply.sh` before workflow execution to generate the `seed-state.json` artifact with dynamic project/workflow IDs. Fixtures using `@seed/` tokens are resolved against this state. Remove `test/artifacts/runtime/seed-state.json` or run `__seeds/cleanup.sh` if you need to regenerate identifiers manually.

## How It Works

### Execution Flow

1. **Workflow Discovery**: Phase helper discovers playbooks from scenario's requirements metadata or playbook registry
2. **Seed Application**: Applies test seed data (`__seeds/apply.sh`) if present, once at phase start
3. **Runtime Management**: Automatically starts target scenario if not running (when `--manage-runtime auto`)
4. **Workflow Resolution**: Inlines fixture references and resolves `@seed/` tokens from seed-state.json
5. **Workflow Execution**: Imports resolved workflow JSON and executes via BAS API (validation happens here)
6. **Result Recording**: Records pass/fail status and requirement evidence
7. **Cleanup**: Optionally deletes test workflows and stops scenario

### Architecture

```
Any Scenario
├── test/playbooks/          # BAS workflow JSON files
│   ├── ui/
│   │   └── page-loads.json
│   └── api/
│       └── endpoints.json
└── test/phases/
    └── test-integration.sh  # Calls run_workflow_validations()
                ↓
scripts/scenarios/testing/playbooks/workflow-runner.sh
                ↓
        browser-automation-studio API
                ↓
        Target Scenario UI/API (any scenario!)
```

## Example: Adding BAS Tests to a New Scenario

### 1. Create Workflow Directory

```bash
mkdir -p scenarios/my-scenario/test/playbooks/ui
```

### 2. Define Workflow JSON

```json
// scenarios/my-scenario/test/playbooks/ui/homepage-loads.json
{
  "metadata": {
    "description": "Verify homepage loads successfully",
    "requirement": "MY-SCENARIO-UI-LOADS",
    "version": 1
  },
  "nodes": [
    {
      "id": "navigate-home",
      "type": "navigate",
      "data": {
        "label": "Navigate to homepage",
        "destinationType": "scenario",
        "scenario": "my-scenario",
        "scenarioPath": "/",
        "waitUntil": "networkidle0"
      }
    },
    {
      "id": "assert-title",
      "type": "assert",
      "data": {
        "label": "Verify page title",
        "selector": "h1",
        "assertMode": "text_contains",
        "expectedValue": "My Scenario"
      }
    }
  ],
  "edges": [
    {"source": "navigate-home", "target": "assert-title"}
  ]
}
```

### 3. Add to Requirements

```json
// scenarios/my-scenario/requirements/ui/rendering.json
{
  "requirements": [
    {
      "id": "MY-SCENARIO-UI-LOADS",
      "category": "ui",
      "title": "Homepage renders successfully",
      "status": "in_progress",
      "prd_ref": "OT-P0-001",
      "validation": [
        {
          "type": "automation",
          "ref": "test/playbooks/ui/homepage-loads.json",
          "phase": "integration",
          "status": "implemented"
        }
      ]
    }
  ]
}
```

### 4. Update Integration Test Phase

```bash
#!/bin/bash
# scenarios/my-scenario/test/phases/test-integration.sh

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

# This single call runs ALL workflow validations defined in requirements
if ! testing::phase::run_workflow_validations \
    --scenario "${TESTING_PHASE_SCENARIO_NAME}" \
    --manage-runtime skip; then
  rc=$?
  if [ "$rc" -ne 0 ] && [ "$rc" -ne 200 ]; then
    testing::phase::add_error "Workflow validations failed"
  fi
fi

testing::phase::end_with_summary "Integration tests completed"
```

### 5. Run Tests

```bash
cd scenarios/my-scenario
./test/phases/test-integration.sh
```

That's it! No scenario-specific boilerplate needed. The workflow runner handles everything.

## Fixture Metadata & Parameterization

Subflows in `test/playbooks/__subflows/` can now expose typed parameters and requirement coverage:

```json
{
  "metadata": {
    "fixture_id": "open-workflow",
    "description": "Load a workflow inside the demo project",
    "parameters": [
      {"name": "project", "type": "string", "required": true},
      {"name": "workflow", "type": "string", "required": true},
      {"name": "mode", "type": "enum", "enumValues": ["builder", "readonly"], "default": "builder"}
    ],
    "requirements": ["BAS-WORKFLOW-DEMO-SEED"]
  },
  "nodes": [
    {"id": "open", "type": "navigate", "data": {"scenario": "browser-automation-studio", "scenarioPath": "/projects"}}
  ]
}
```

- Call fixtures with optional arguments: `"workflowId": "@fixture/open-workflow(project=\"Demo\", workflow=@store/seed.workflowName)"`
- Strings with spaces/punctuation must be wrapped in quotes. Escape quotes with `\"`.
- Use `@store/<key>` to forward runtime values captured earlier in the workflow.
- Use `@seed/<key>` (pulled from `test/artifacts/runtime/seed-state.json`) when you need literal strings during resolution. The resolver replaces those tokens with the latest seed metadata that `test/playbooks/__seeds/apply.sh` wrote.
- Number/boolean parameters reject store references (the resolver enforces types before execution).

When a fixture declares `metadata.requirements`, the resolver automatically propagates those IDs to the parent workflow under `metadata.requirementsFromFixtures`. The phase helper records pass/fail evidence for those requirement IDs whenever the parent workflow runs, so fixture-level guarantees show up in `coverage/phase-results` and requirement reports.

Seeding scripts can also write dynamic context to `test/artifacts/runtime/seed-state.json`. Ship a helper fixture (for example `load-seed-state`) that imports the JSON via `@seed/<key>` tokens and stores the values under canonical keys such as `seed.projectName`, `seed.projectId`, `seed.workflowName`, and `seed.workflowId`. Downstream fixtures then reference `@store/seed.projectName` (or reuse the fixture parameters) without needing to know how each run was seeded.

## Cross-Scenario Testing

The workflow runner is truly generic - browser-automation-studio can test **any scenario**:

```json
// scenarios/scenario-a/test/playbooks/integration/test-scenario-b.json
{
  "metadata": {
    "description": "Scenario A testing Scenario B",
    "requirement": "SCENARIO-A-INTEGRATION-B"
  },
  "nodes": [
    {
      "id": "navigate",
      "type": "navigate",
      "data": {
        "scenario": "scenario-b",  // ← Testing different scenario!
        "scenarioPath": "/api/health"
      }
    }
  ]
}
```

## Backward Compatibility

The old function name `testing::phase::run_bas_automation_validations` is maintained as an alias to `testing::phase::run_workflow_validations` for backward compatibility. Both work identically.

## Advanced: Adhoc Execution

The runner automatically attempts adhoc execution first (no DB pollution):

```bash
testing::playbooks::run_workflow \
  --file my-workflow.json \
  --scenario browser-automation-studio
# ✨ Attempting adhoc workflow execution (no DB pollution)
# Falls back to import+execute if adhoc endpoint unavailable
```

## Troubleshooting

### "workflow runner not available"

Make sure your phase script sources `phase-helpers.sh`:
```bash
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
```

### "Unable to resolve API_PORT"

Ensure the target scenario is running:
```bash
vrooli scenario status my-scenario
vrooli scenario start my-scenario
```

### Workflows not discovered

Check that:
1. Workflow paths in requirements match actual file locations
2. Workflow JSON is valid and not empty
3. Requirements use `type: "automation"` and `ref: "test/playbooks/..."` format

## Related Documentation

- [Phase Testing Architecture](../shell/README.md)
- [Requirements System](../../../requirements/lib/README.md)
- [Browser Automation Studio Documentation](../../../../scenarios/browser-automation-studio/README.md)
