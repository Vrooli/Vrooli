# Workflow Runner Reference

> **⚠️ DEPRECATED**: This document describes the legacy bash-based workflow runner. BAS workflows are now executed through the Go-native test-genie orchestrator during the business phase. See [UI Automation with BAS](../phases/playbooks/ui-automation-with-bas.md) for the current approach.

**Status**: Deprecated (Legacy Reference)
**Last Updated**: 2025-12-02
**Superseded By**: Go-native orchestrator in `api/orchestrator/phases/business.go`

---

This document describes the legacy workflow runner infrastructure. For new development, use the test-genie API or CLI to execute BAS workflows.

## Overview (Legacy)

The deprecated `workflow-runner.sh` script provided a universal interface for running BAS workflows. This functionality is now handled by:

- Workflow execution via browser-automation-studio API
- Runtime workflow validation (schema, structure, selectors)
- Scenario runtime management (auto-start/stop)
- Result tracking and requirement evidence recording
- Both file-based and database-persisted workflow execution

**Note:** Workflow validation happens at runtime during execution, not as a separate linting step.

## Location

```
scripts/scenarios/testing/playbooks/workflow-runner.sh
```

## Usage

### From Test Phase Scripts (Recommended)

Use the phase helper wrapper for automatic discovery and execution:

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

For custom scenarios or manual execution:

```bash
source "${APP_ROOT}/scripts/scenarios/testing/playbooks/workflow-runner.sh"

testing::playbooks::run_workflow \
  --file test/playbooks/my-test.json \
  --scenario my-scenario \
  --manage-runtime auto
```

## Command Options

### `testing::phase::run_workflow_validations`

| Option | Default | Description |
|--------|---------|-------------|
| `--scenario NAME` | browser-automation-studio | Target scenario to test |
| `--manage-runtime MODE` | auto | Runtime management: `auto`, `start`, `skip` |
| `--disallow-missing` | - | Treat missing workflow files as failures instead of skips |

### `testing::playbooks::run_workflow`

| Option | Default | Description |
|--------|---------|-------------|
| `--file PATH` | (required) | JSON workflow definition |
| `--scenario NAME` | browser-automation-studio | Scenario to execute against |
| `--manage-runtime MODE` | auto | Auto start/stop: `auto`, `start` (1/true), `skip` (0/false) |
| `--timeout SECONDS` | 120 | Max wait for scenario readiness |
| `--allow-missing` | - | Return exit code 210 when workflow file missing (for optional tests) |

## Execution Flow

```
1. Workflow Discovery
   Phase helper discovers playbooks from scenario's requirements
   metadata or playbook registry

2. Seed Application
   Applies test seed data (__seeds/seed.go or seed.sh) if present, once at phase start

3. Runtime Management
   Automatically starts target scenario if not running (when --manage-runtime auto)

4. Workflow Resolution
   Inlines fixture references and resolves @seed/ tokens from seed-state.json

5. Workflow Execution
   Imports resolved workflow JSON and executes via BAS API (validation happens here)

6. Result Recording
   Records pass/fail status and requirement evidence

7. Cleanup
   Optionally deletes test workflows and stops scenario
```

## Architecture

```
Any Scenario
  test/playbooks/          # BAS workflow JSON files
    ui/
      page-loads.json
    api/
      endpoints.json
  test/phases/
    test-integration.sh   # Calls run_workflow_validations()
            |
            v
scripts/scenarios/testing/playbooks/workflow-runner.sh
            |
            v
    browser-automation-studio API
            |
            v
    Target Scenario UI/API (any scenario!)
```

## Seed Data Management

BAS integration tests apply seed data before workflow execution:

```
test/playbooks/
  __seeds/
    seed.go         # Generates seed-state.json with dynamic IDs (or seed.sh)
test/artifacts/
  runtime/
    seed-state.json # Contains project/workflow IDs for @seed/ tokens
```

**Usage in workflows:**
- Use `@seed/<key>` tokens (e.g., `@seed/projectId`) for literal string resolution
- These are resolved from `test/artifacts/runtime/seed-state.json`
- Remove `seed-state.json` or run `cleanup.sh` to regenerate identifiers

## Fixture Parameters

Subflows in `test/playbooks/__subflows/` can expose typed parameters:

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
  "nodes": [...]
}
```

**Calling fixtures with arguments:**
```json
{
  "workflowId": "@fixture/open-workflow(project=\"Demo\", workflow=@store/seed.workflowName)"
}
```

**Parameter rules:**
- Strings with spaces/punctuation must be wrapped in quotes
- Escape quotes with `\"`
- Use `@store/<key>` to forward runtime values captured earlier
- Number/boolean parameters reject store references (types enforced)

**Requirement propagation:**
When a fixture declares `metadata.requirements`, the resolver automatically propagates those IDs to the parent workflow under `metadata.requirementsFromFixtures`. The phase helper records pass/fail evidence for those requirement IDs whenever the parent workflow runs.

## Cross-Scenario Testing

The workflow runner is truly generic - any scenario can test any other scenario:

```json
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
        "scenario": "scenario-b",
        "scenarioPath": "/api/health"
      }
    }
  ]
}
```

## Adhoc Execution

The runner automatically attempts adhoc execution first (no DB pollution):

```bash
testing::playbooks::run_workflow \
  --file my-workflow.json \
  --scenario browser-automation-studio
# Attempting adhoc workflow execution (no DB pollution)
# Falls back to import+execute if adhoc endpoint unavailable
```

## Backward Compatibility

The old function name `testing::phase::run_bas_automation_validations` is maintained as an alias to `testing::phase::run_workflow_validations`. Both work identically.

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

## See Also

- [UI Automation with BAS](../phases/playbooks/ui-automation-with-bas.md) - Authoring workflows
- [UI Testability](../guides/ui-testability.md) - Design UIs for automation
- [Phases Overview](../phases/README.md) - Phase specifications
