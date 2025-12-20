# UI Automation with Vrooli Ascension

> **Essential guide for writing UI tests as JSON workflows that integrate with requirement tracking**

## Table of Contents
- [Overview](#overview)
- [Why BAS Workflows](#why-bas-workflows)
- [Workflow Structure](#workflow-structure)
- [Node Types Reference](#node-types-reference)
- [Authoring Workflows](#authoring-workflows)
- [Workflow Lifecycle](#workflow-lifecycle)
- [Requirements Integration](#requirements-integration)
- [Storage Location](#storage-location)
- [Execution Helper](#execution-helper)
- [Troubleshooting](#troubleshooting)
- [See Also](#see-also)

## Overview

Vrooli Ascension (BAS) enables **declarative UI testing** through JSON workflows that execute via API. This eliminates Playwright/Puppeteer code while maintaining version control and requirement tracking integration.

**What is BAS workflow testing?**
- Write UI tests as **declarative JSON workflows** with nodes and edges
- Execute workflows via **BAS API** (no manual clicking, fully automated)
- Integrate with **requirement tracking** using `automation` validation type
- **Self-testing**: BAS validates itself using its own automation capabilities

**Benefits:**
- **Declarative** - JSON workflows are version-controlled, not imperative code
- **No Playwright code** - Node configuration instead of page.click() calls
- **Requirement tracking** - Auto-sync like unit tests (`type: automation`)
- **Maintainable** - Update UI then regenerate workflow then commit JSON
- **AI-friendly** - Agents write JSON directly, no framework learning curve

## Why BAS Workflows

**vs. Playwright/Puppeteer**:
- Playwright: Requires JavaScript knowledge, imperative code, brittle selectors
- BAS: Declarative JSON, visual editor option, requirement integration built-in

**vs. Manual Testing**:
- Manual: Not repeatable, can't integrate with CI, no coverage tracking
- BAS: Automated, versioned, integrated with requirement tracking
- Manual validations remain a last-resort escape hatch. If you absolutely need one, log it with `vrooli scenario requirements manual-log` so drift detection knows when it expires - but build a BAS workflow as soon as possible.

**For AI Agents**:
- BAS workflows are **JSON structures** that agents can write directly
- No need to learn Playwright API or handle async/await patterns
- Clear node types with typed `data` properties
- Execution errors include screenshots and DOM snapshots for debugging

## Workflow Structure

### Minimal Example

```json
{
  "metadata": {
    "description": "Test project creation flow",
    "version": 1
  },
  "settings": {
    "executionViewport": {
      "width": 1440,
      "height": 900,
      "preset": "desktop"
    }
  },
  "nodes": [
    {
      "id": "navigate-home",
      "type": "navigate",
      "position": { "x": 0, "y": 0 },
      "data": {
        "label": "Navigate to homepage",
        "destinationType": "scenario",
        "scenario": "my-scenario",
        "scenarioPath": "/",
        "waitUntil": "networkidle0"
      }
    },
    {
      "id": "click-button",
      "type": "click",
      "position": { "x": 220, "y": 0 },
      "data": {
        "label": "Click create button",
        "selector": "[data-testid='create-btn']",
        "waitForSelector": "[data-testid='create-btn']"
      }
    }
  ],
  "edges": [
    {
      "id": "edge-1",
      "source": "navigate-home",
      "target": "click-button",
      "type": "smoothstep"
    }
  ]
}
```

### Key Components

**metadata**: Describes purpose and declares reset scope.
```json
{
  "description": "Human-readable purpose",
  "version": 1,
  "reset": "none"
}
```

**settings**: Execution environment configuration
```json
{
  "executionViewport": {
    "width": 1440,
    "height": 900,
    "preset": "desktop"
  }
}
```

**nodes**: Array of steps in the workflow
```json
{
  "id": "unique-node-id",
  "type": "navigate",
  "position": { "x": 0, "y": 0 },
  "data": {
    "label": "Step description"
  }
}
```

**edges**: Connections defining execution order
```json
{
  "id": "unique-edge-id",
  "source": "source-node-id",
  "target": "target-node-id",
  "type": "smoothstep"
}
```

## Workflow Stories & Registry Order

- Prefix top-level folders under `bas/cases/` (and `bas/flows/` if used) with a two-digit ordinal (e.g., `01-foundation`, `02-builder`)
- After editing or moving a workflow, regenerate the registry using the playbook build script
- `metadata.reset` (`none`, `full`) tells the runner when to reseed

## Workflow Hierarchy (Critical)

BAS workflows are **hierarchical** and should be authored/tested in this order:
1. **Actions** (`bas/actions/`): atomic, reusable steps; no assertions.
2. **Flows** (`bas/flows/`): compose actions into journeys; still no assertions.
3. **Cases** (`bas/cases/`): assert requirements; call flows or actions.

This ordering is the fastest way to debug failures: stabilize actions first, then flows, then cases.

## Node Types Reference

### Navigate Node

```json
{
  "type": "navigate",
  "data": {
    "label": "Navigate to dashboard",
    "destinationType": "scenario",
    "scenario": "my-scenario",
    "scenarioPath": "/dashboard",
    "waitUntil": "networkidle0",
    "timeoutMs": 30000,
    "waitForMs": 1000
  }
}
```

### Click Node

```json
{
  "type": "click",
  "data": {
    "label": "Click submit button",
    "selector": "[data-testid='submit-btn']",
    "waitForSelector": "[data-testid='submit-btn']",
    "timeoutMs": 10000,
    "waitForMs": 500,
    "clickCount": 1
  }
}
```

### Type Node

```json
{
  "type": "type",
  "data": {
    "label": "Enter project name",
    "selector": "[data-testid='project-name-input']",
    "text": "My Test Project",
    "clearExisting": true,
    "delay": 50
  }
}
```

### Assert Node

```json
{
  "type": "assert",
  "data": {
    "label": "Verify success message",
    "selector": "[data-testid='success-message']",
    "assertMode": "exists",
    "expectedText": "Success!",
    "timeoutMs": 10000,
    "failureMessage": "Success message should appear"
  }
}
```

**Assert modes**: `exists`, `not_exists`, `contains_text`, `exact_text`

### Wait Node

```json
{
  "type": "wait",
  "data": {
    "label": "Wait for animation",
    "durationMs": 1500
  }
}
```

### Screenshot Node

```json
{
  "type": "screenshot",
  "data": {
    "label": "Capture dashboard state",
    "fullPage": true,
    "captureDomSnapshot": true,
    "selector": "[data-testid='chart']",
    "waitForMs": 1000
  }
}
```

## Selector Registry System

BAS uses a **centralized selector registry** to eliminate hardcoded CSS selectors.

### How It Works

**Single Source of Truth**: All selectors defined in `ui/src/constants/selectors.ts`:

```typescript
const literalSelectors = {
  dashboard: {
    newProjectButton: "dashboard-new-project-button",
  },
  workflows: {
    tab: "workflows-tab",
    newButton: "new-workflow-button",
  },
} as const;
```

**In UI Components**:

```typescript
import { selectors } from '@/constants/selectors';

<button data-testid={selectors.dashboard.newProjectButton}>
  New Project
</button>
```

**In Workflows**: Reference with `@selector/` prefix:

```json
{
  "type": "click",
  "data": {
    "selector": "@selector/dashboard.newProjectButton"
  }
}
```

### Dynamic Selectors

```typescript
const dynamicSelectorDefinitions = {
  projects: {
    cardByName: defineDynamicSelector({
      description: "Project card filtered by name",
      selectorPattern: '[data-testid="project-card"][data-project-name="${name}"]',
      params: { name: { type: "string" } },
    }),
  },
} as const;
```

In workflows:

```json
{
  "selector": "@selector/projects.cardByName(name=My Project)"
}
```

## Resilience Settings

Nodes can include a `resilience` object for retries and readiness checks:

```json
{
  "type": "click",
  "data": {
    "selector": "@selector/cta.primary",
    "resilience": {
      "maxAttempts": 3,
      "delayMs": 1500,
      "backoffFactor": 1.5,
      "preconditionSelector": "@selector/app.shell.ready",
      "preconditionTimeoutMs": 10000,
      "successSelector": "@selector/onboarding.stepTwo",
      "successTimeoutMs": 15000
    }
  }
}
```

## Storage Location

Each scenario owns automation assets under `bas/` following the [canonical directory structure](directory-structure.md):

```
bas/
├── registry.json       # Auto-generated manifest
├── cases/              # Assertive validations (mirrors PRD)
│   └── 01-foundation/  # Two-digit prefix for ordering
├── flows/              # Multi-surface user journeys
├── actions/            # Reusable fixtures/subflows
└── seeds/              # Seed entrypoint (seed.go preferred)
```

See [Directory Structure](directory-structure.md) for naming conventions, fixture metadata, and authoring checklist.

## Execution

**Run iteratively while authoring** using the BAS CLI (recommended):
```bash
browser-automation-studio execution create \
  --file bas/cases/01-foundation/01-projects/new-project-create.json \
  --wait
```

**Run in the Playbooks phase** using test-genie:
```bash
test-genie execute my-scenario --preset comprehensive
```

**Via API (test-genie orchestrator):**
```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/test-suite/my-scenario/execute-sync" \
  -H "Content-Type: application/json" \
  -d '{"preset": "comprehensive"}'
```

The business phase automatically discovers and executes workflows defined in `bas/` with `automation` validation types.

## Requirements Integration

Annotate requirement validations with the `automation` type:

```json
{
  "validation": [
    {
      "type": "automation",
      "ref": "bas/cases/02-builder/demo-sanity.json",
      "scenario": "browser-automation-studio",
      "phase": "integration",
      "status": "implemented"
    }
  ]
}
```

## Troubleshooting

### Workflow Fails to Import

**Causes**: JSON file doesn't exist, syntax error, BAS not running

**Solutions**:
```bash
ls -la bas/cases/01-foundation/01-projects/create.json
jq . bas/cases/01-foundation/01-projects/create.json
vrooli scenario status browser-automation-studio
```

### Selector Not Found

For `@selector/` references, the error message will tell you:
1. If manifest was auto-regenerated (selector doesn't exist in selectors.ts)
2. Similar selectors you might have meant

**Solution**: Register selector in `ui/src/constants/selectors.ts` first.

### Workflow Times Out

**Solutions**:
```json
{
  "type": "navigate",
  "data": {
    "waitUntil": "load",
    "waitForMs": 2000
  }
}
```

### Assert Node Fails

```json
{
  "assertMode": "contains_text",
  "expectedText": "Success"
}
```

## Reusable Subflows & Seed Data

- **Subflows**: Store under `bas/actions/`. Reference from case/flow workflows via subflow nodes (`workflow_path: "actions/<slug>.json"`).
- **Seed Data**: Keep in `bas/seeds/` with `seed.go` (or `seed.sh`)
- **Canonical fixtures**: Use `open-demo-project`, `open-builder-from-demo`, `open-demo-workflow`

See [Directory Structure](directory-structure.md) for complete fixture metadata reference, parameter syntax, and token types (`@fixture/`, `@seed/`, `@store/`).

## See Also

- [Playbooks Phase](README.md) - Playbooks phase overview
- [Directory Structure](directory-structure.md) - Canonical layout, fixtures, seeds
- [Writing Testable UIs](../../guides/ui-testability.md) - Design UIs for automation
- [End-to-End Example](../../guides/end-to-end-example.md) - Complete flow from PRD to BAS workflow
- [Requirements Sync](../business/requirements-sync.md) - Understanding `automation` validation type
- [Phases Overview](../README.md) - How BAS integrates with phases
- [Validation Best Practices](../../guides/validation-best-practices.md) - Multi-layer validation strategy

---

**Remember**: BAS workflows are JSON data structures. AI agents can write them directly without learning Playwright APIs or handling async patterns.
