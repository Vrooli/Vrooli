# 🤖 UI Automation with Browser Automation Studio

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

Browser Automation Studio (BAS) enables **declarative UI testing** through JSON workflows that execute via API. This eliminates Playwright/Puppeteer code while maintaining version control and requirement tracking integration.

**What is BAS workflow testing?**
- Write UI tests as **declarative JSON workflows** with nodes and edges
- Execute workflows via **BAS API** (no manual clicking, fully automated)
- Integrate with **requirement tracking** using `automation` validation type
- **Self-testing**: BAS validates itself using its own automation capabilities

**Benefits:**
- ✅ **Declarative** - JSON workflows are version-controlled, not imperative code
- ✅ **No Playwright code** - Node configuration instead of page.click() calls
- ✅ **Requirement tracking** - Auto-sync like unit tests (`type: automation`)
- ✅ **Maintainable** - Update UI → regenerate workflow → commit JSON
- ✅ **AI-friendly** - Agents write JSON directly, no framework learning curve

## Why BAS Workflows

**vs. Playwright/Puppeteer**:
- ❌ Playwright: Requires JavaScript knowledge, imperative code, brittle selectors
- ✅ BAS: Declarative JSON, visual editor option, requirement integration built-in

**vs. Manual Testing**:
- ❌ Manual: Not repeatable, can't integrate with CI, no coverage tracking
- ✅ BAS: Automated, versioned, integrated with requirement tracking
- ℹ️ Manual validations remain a last-resort escape hatch. If you absolutely need one, log it with `vrooli scenario requirements manual-log` so drift detection knows when it expires—but build a BAS workflow as soon as possible to retire the manual checklist entry.

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

**metadata**: Describes purpose and declares reset scope. Requirement linkage now happens in `requirements/*.json` via `validation.ref` entries (the registry builder pulls those IDs automatically), so skip `metadata.requirement` entirely.
```json
{
  "description": "Human-readable purpose",
  "version": 1,             // Increment when workflow changes significantly
  "reset": "none"          // "none" (default) or "full" (reseed all state)
}
```

**settings**: Execution environment configuration
```json
{
  "executionViewport": {
    "width": 1440,      // Browser width
    "height": 900,      // Browser height
    "preset": "desktop" // "mobile" | "tablet" | "desktop"
  }
}
```

**nodes**: Array of steps in the workflow
```json
{
  "id": "unique-node-id",           // Must be unique within workflow
  "type": "navigate",               // Node type (see reference below)
  "position": { "x": 0, "y": 0 },  // Visual editor position (optional for execution)
  "data": {
    "label": "Step description",   // Shown in execution logs
    // ... type-specific properties
  }
}
```

**edges**: Connections defining execution order
```json
{
  "id": "unique-edge-id",
  "source": "source-node-id",  // Node to execute first
  "target": "target-node-id",  // Node to execute after source completes
  "type": "smoothstep"         // Visual style (doesn't affect execution)
}
```

## Workflow Stories & Registry Order

- Prefix every folder under `test/playbooks/` with a two-digit ordinal (for example `01-foundation`, `02-builder`). The runner performs a depth-first traversal through this tree, so the prefixes form the canonical execution story.
- After editing or moving a workflow, regenerate the registry: `node scripts/scenarios/testing/playbooks/build-registry.mjs --scenario <path>`. The generated `test/playbooks/registry.json` is the source of truth for ordering, fixtures, and reset requirements.
- Inspect the registry to ensure the dotted `order` key matches your expectations (e.g., `01.02.03` for the third workflow within `02-builder`). If the order is wrong, rename the folder or file with the desired prefix and regenerate. When in doubt, run `browser-automation-studio playbooks order` to print the story (order, reset, requirement count, description) without opening JSON files.
- `browser-automation-studio playbooks scaffold <folder> <name>` is the fastest way to add a new workflow skeleton. It pre-populates metadata/reset and drops a placeholder `navigate` + `wait` chain under the requested folder so you can immediately wire it into requirements.
- `browser-automation-studio playbooks verify` scans `test/playbooks/` and warns about folders missing the `NN-` prefixes. Run it whenever you add/move directories to keep the traversal predictable.
- `metadata.reset` (`none`, `full`) tells the runner when to reseed. Set it to `none` for read-only flows so multiple playbooks can run back-to-back without tearing down BAS. Use `full` when mutations require completely reseeding state.

## Node Types Reference

### Navigate Node

**Purpose**: Navigate to a URL or scenario

**Type-specific data**:
```json
{
  "type": "navigate",
  "data": {
    "label": "Navigate to dashboard",
    "destinationType": "scenario",  // "scenario" | "url"
    "scenario": "my-scenario",      // Scenario name (when destinationType="scenario")
    "scenarioPath": "/dashboard",   // Path within scenario
    "url": "https://example.com",   // Direct URL (when destinationType="url")
    "waitUntil": "networkidle0",    // "load" | "domcontentloaded" | "networkidle0" | "networkidle2"
    "timeoutMs": 30000,             // Max wait time
    "waitForMs": 1000               // Additional wait after navigation
  }
}
```

**Use scenario navigation** when testing Vrooli scenarios (auto-resolves ports):
```json
{
  "destinationType": "scenario",
  "scenario": "browser-automation-studio",
  "scenarioPath": "/projects"
  // Resolves to http://localhost:{UI_PORT}/projects
}
```

### Click Node

**Purpose**: Click an element

**Type-specific data**:
```json
{
  "type": "click",
  "data": {
    "label": "Click submit button",
    "selector": "[data-testid='submit-btn']",  // CSS selector
    "waitForSelector": "[data-testid='submit-btn']",  // Wait for element before clicking
    "timeoutMs": 10000,
    "waitForMs": 500,  // Wait after click
    "clickCount": 1    // Number of clicks (default: 1)
  }
}
```

**Best practices**:
- Always use `@selector/` references instead of raw CSS selectors
- Set `waitForSelector` to ensure element exists
- Add `waitForMs` if subsequent nodes need time to react

## Selector Registry System

BAS uses a **centralized selector registry** to eliminate hardcoded CSS selectors and prevent drift between UI code and tests.

### How It Works

**Single Source of Truth**: All selectors are defined in `ui/src/consts/selectors.ts` (or `ui/src/constants/selectors.ts` for scenarios that follow the BAS alias):

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

**In UI Components**: Import and use the selector:

```typescript
import { selectors } from '@/consts/selectors'; // or '@constants/selectors'

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

### Benefits

- ✅ **No drift** - UI and tests use the same selector source
- ✅ **Type-safe** - UI code gets TypeScript autocomplete
- ✅ **Maintainable** - Rename once in selectors.ts, updates everywhere
- ✅ **Auto-regeneration** - Manifest rebuilds automatically when needed

### Auto-Regeneration

The selector manifest (`ui/src/consts/selectors.manifest.json` or `ui/src/constants/selectors.manifest.json`) is automatically regenerated when tests detect it's stale. **You never need to manually regenerate it.**

**Workflow:**

1. Add selector to `selectors.ts`
2. Import and use it in your UI component
3. Reference it in test workflow with `@selector/`
4. Run tests → **manifest auto-regenerates** on first use

**If you get a "selector not found" error:**

The error message will tell you if:
- The manifest was auto-regenerated but selector still not found → **Register it in selectors.ts first**
- The selector exists but has a typo → **Suggestions are shown**

### Dynamic Selectors

For selectors with parameters, use `defineDynamicSelector`:

```typescript
const dynamicSelectorDefinitions = {
  projects: {
    cardByName: defineDynamicSelector({
      description: "Dashboard project card filtered by name",
      selectorPattern: '[data-testid="project-card"][data-project-name="${name}"]',
      params: { name: { type: "string" } },
    }),
  },
} as const;
```

**In workflows:**

```json
{
  "selector": "@selector/projects.cardByName(name=My Project)"
}
```

## Resilience Settings

Nodes that interact with the DOM can include a `resilience` object to centralize retries and selector readiness checks. The builder exposes these fields via a dedicated panel, so authors no longer need to duplicate `wait` + `click` chains.

```json
{
  "id": "click-primary-cta",
  "type": "click",
  "data": {
    "label": "Start onboarding",
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

- `maxAttempts`, `delayMs`, and `backoffFactor` wire up the executor's retry engine (attempts = retries + 1).
- `preconditionSelector` waits for additional DOM state before the node executes, with optional `preconditionTimeoutMs` and `preconditionWaitMs` for fine-tuning.
- `successSelector` confirms the UI reached the expected state before the workflow advances. Pair it with `successTimeoutMs`/`successWaitMs` when transitions involve animations.

`workflow lint --strict` validates these fields, and integration logs surface misconfigurations alongside selector warnings.

> **Default behavior:** When a node omits `resilience`, BAS automatically waits for the node's selector (or `waitForSelector` when present), retries the step up to three total attempts with a ~750 ms delay and light backoff, and reuses the node timeout when gating DOM readiness. Setting `maxAttempts` to `1` (or providing explicit values) disables those defaults per node.

### Type Node

**Purpose**: Enter text into an input field

**Type-specific data**:
```json
{
  "type": "type",
  "data": {
    "label": "Enter project name",
    "selector": "[data-testid='project-name-input']",
    "text": "My Test Project",
    "clearExisting": true,  // Clear field before typing
    "delay": 50             // Milliseconds between keystrokes (simulates human typing)
  }
}
```

### Assert Node

**Purpose**: Verify element existence, text content, or attributes

**Type-specific data**:
```json
{
  "type": "assert",
  "data": {
    "label": "Verify success message",
    "selector": "[data-testid='success-message']",
    "assertMode": "exists",         // "exists" | "not_exists" | "contains_text" | "exact_text"
    "expectedText": "Success!",     // Required for text assertions
    "timeoutMs": 10000,
    "failureMessage": "Success message should appear after project creation"
  }
}
```

**Assert modes**:
- `exists`: Element present in DOM
- `not_exists`: Element not in DOM (useful for testing removal)
- `contains_text`: Element text contains substring (case-sensitive)
- `exact_text`: Element text exactly matches (case-sensitive)

### Wait Node

**Purpose**: Pause execution for a fixed duration

**Type-specific data**:
```json
{
  "type": "wait",
  "data": {
    "label": "Wait for animation",
    "durationMs": 1500  // Milliseconds to wait
  }
}
```

**When to use**:
- After navigation to let page settle
- After modal open/close animations
- When waiting for visual transitions

**Prefer assertions over fixed waits** when possible - they're more reliable.

### Screenshot Node

**Purpose**: Capture page or element screenshot

**Type-specific data**:
```json
{
  "type": "screenshot",
  "data": {
    "label": "Capture dashboard state",
    "fullPage": true,              // Capture entire page (scrolls if needed)
    "captureDomSnapshot": true,    // Also capture DOM for replay
    "selector": "[data-testid='chart']",  // Capture specific element (optional)
    "waitForMs": 1000              // Wait before capturing
  }
}
```

**Artifacts**:
- Screenshots saved to execution artifacts
- Accessible via `/api/v1/executions/{id}/artifacts`
- Viewable in BAS UI replay panel

### Extract Node

**Purpose**: Extract data from page (text, attributes, HTML)

**Type-specific data**:
```json
{
  "type": "extract",
  "data": {
    "label": "Extract username",
    "selector": "[data-testid='user-name']",
    "extractMode": "text",        // "text" | "attribute" | "html"
    "attribute": "data-value",    // Required when extractMode="attribute"
    "variable": "extractedUsername"  // Store in execution context (future use)
  }
}
```

## Authoring Workflows

### For AI Agents (Direct JSON)

When writing workflows programmatically, follow this pattern:

1. **Start with navigation**
   ```json
   {
     "id": "nav-start",
     "type": "navigate",
     "data": {
       "destinationType": "scenario",
       "scenario": "target-scenario",
       "scenarioPath": "/"
     }
   }
   ```

2. **Use data-testid selectors** (from Writing Testable UIs guide)
   ```json
   {
     "selector": "[data-testid='element-id']"  // ✅ Stable
   }
   // NOT:
   { "selector": "div > button.btn-primary" }  // ❌ Brittle
   ```

3. **Add waits strategically**
   - After navigation: `waitForMs: 1000-2000`
   - After clicks that trigger navigation: `waitForMs: 500`
   - Before assertions: Use assert node's built-in timeout

4. **Include screenshots** at key points
   - After navigation (capture initial state)
   - Before form submission (capture input values)
   - After expected state changes

5. **Use descriptive labels**
   ```json
   { "label": "Click submit after filling form" }  // ✅ Clear
   { "label": "Click button" }                     // ❌ Vague
   ```

6. **Create linear flows first**, branch later
   - Start with happy path (success scenario)
   - Add error paths once happy path works

### Example: Complete Form Workflow

```json
{
  "metadata": {
    "description": "Test complete project creation with validation",
    "version": 1
  },
  "settings": {
    "executionViewport": { "width": 1440, "height": 900, "preset": "desktop" }
  },
  "nodes": [
    {
      "id": "nav",
      "type": "navigate",
      "position": { "x": 0, "y": 0 },
      "data": {
        "label": "Navigate to homepage",
        "destinationType": "scenario",
        "scenario": "my-scenario",
        "scenarioPath": "/",
        "waitUntil": "networkidle0",
        "waitForMs": 1500
      }
    },
    {
      "id": "screenshot-initial",
      "type": "screenshot",
      "position": { "x": 220, "y": -100 },
      "data": {
        "label": "Capture initial state",
        "fullPage": true,
        "captureDomSnapshot": true
      }
    },
    {
      "id": "click-create",
      "type": "click",
      "position": { "x": 220, "y": 100 },
      "data": {
        "label": "Click new project button",
        "selector": "[data-testid='new-project-btn']",
        "waitForSelector": "[data-testid='new-project-btn']",
        "waitForMs": 500
      }
    },
    {
      "id": "assert-modal",
      "type": "assert",
      "position": { "x": 440, "y": 100 },
      "data": {
        "label": "Verify modal opened",
        "selector": "[data-testid='project-modal']",
        "assertMode": "exists",
        "timeoutMs": 5000,
        "failureMessage": "Project creation modal should appear"
      }
    },
    {
      "id": "fill-name",
      "type": "type",
      "position": { "x": 660, "y": 100 },
      "data": {
        "label": "Enter project name",
        "selector": "[data-testid='project-name-input']",
        "text": "E2E Test Project",
        "clearExisting": true
      }
    },
    {
      "id": "fill-description",
      "type": "type",
      "position": { "x": 880, "y": 100 },
      "data": {
        "label": "Enter description",
        "selector": "[data-testid='project-description-input']",
        "text": "Created by automated workflow test"
      }
    },
    {
      "id": "screenshot-form",
      "type": "screenshot",
      "position": { "x": 1100, "y": 0 },
      "data": {
        "label": "Capture filled form",
        "fullPage": false,
        "captureDomSnapshot": true
      }
    },
    {
      "id": "submit",
      "type": "click",
      "position": { "x": 1100, "y": 200 },
      "data": {
        "label": "Submit form",
        "selector": "[data-testid='project-modal-submit']",
        "waitForMs": 2000
      }
    },
    {
      "id": "assert-success",
      "type": "assert",
      "position": { "x": 1320, "y": 200 },
      "data": {
        "label": "Verify success message",
        "selector": "[data-testid='success-message']",
        "assertMode": "contains_text",
        "expectedText": "Project created successfully",
        "timeoutMs": 10000
      }
    }
  ],
  "edges": [
    { "id": "e1", "source": "nav", "target": "screenshot-initial" },
    { "id": "e2", "source": "nav", "target": "click-create" },
    { "id": "e3", "source": "click-create", "target": "assert-modal" },
    { "id": "e4", "source": "assert-modal", "target": "fill-name" },
    { "id": "e5", "source": "fill-name", "target": "fill-description" },
    { "id": "e6", "source": "fill-description", "target": "screenshot-form" },
    { "id": "e7", "source": "fill-description", "target": "submit" },
    { "id": "e8", "source": "submit", "target": "assert-success" }
  ]
}
```

**Note**: The `position` values are for visual editor layout and don't affect execution order. Execution follows the `edges` graph.

## Workflow Lifecycle

```mermaid
graph TB
    Author[Author workflow<br/>in BAS UI] --> Export[Export to JSON<br/>via BAS API]
    Export --> Commit[Commit to repo<br/>test/playbooks/*.json]
    Commit --> Phase[Phase script<br/>executes workflow]
    Phase --> Import[Import to BAS<br/>Testing Harness project]
    Import --> Execute[Execute via<br/>/workflows/{id}/execute]
    Execute --> Track[Update requirement<br/>tracking]
    Track --> Cleanup[Cleanup temp<br/>workflows]

    style Author fill:#e1f5ff
    style Export fill:#fff3e0
    style Commit fill:#f3e5f5
    style Execute fill:#e8f5e9
    style Track fill:#c8e6c9
```

**Key insight:** Workflows are created visually, stored as code, executed programmatically.

## Storage Location

- Each scenario owns its automation assets under `scenarios/<name>/test/playbooks/`.
- Use the canonical layout: `capabilities/<operational-target>/<surface>/` for feature coverage
  (01-foundation, 02-builder, 03-execution, 04-ai, 05-replay, 06-nonfunctional), `journeys/<flow>/`
  for cross-surface flows, plus `__subflows/` and `__seeds/` for reusable fragments.
- Keep `test/playbooks/README.md` under 100 lines describing how folders tie back to requirements
  and selectors.
- Workflow definitions are exported JSON payloads from the BAS UI/API. Do not hand-edit the graph
  structure—regenerate from BAS when behaviour changes.
- Optionally attach metadata in `workflow.metadata` to describe intent, version, or requirement IDs.
- Avoid compatibility shims—move files into the final layout immediately rather than keeping legacy
  duplicates.

## Execution Helper

Use the shared helper in `scripts/scenarios/testing/playbooks/browser-automation-studio.sh`:

```bash
source "${APP_ROOT}/scripts/scenarios/testing/playbooks/browser-automation-studio.sh"

if testing::playbooks::bas::run_workflow \
    --file "test/playbooks/capabilities/03-execution/01-automation/telemetry-smoke.json" \
    --scenario "browser-automation-studio"; then
  testing::phase::add_requirement --id BAS-EXEC-TELEMETRY-AUTOMATION --status passed \
    --evidence "Telemetry workflow executed"
else
  # Non-zero exit codes indicate execution failure; exit 210 marks missing workflow exports.
  ...
fi
```

Or allow the phase helper to execute every validated workflow automatically:

```bash
# Runs all `type: automation` validations assigned to this phase
testing::phase::run_bas_automation_validations --scenario "$SCENARIO_NAME" --manage-runtime skip
```

Key behaviours (API-native pipeline):

- Ensures `jq` and `curl` are available, resolves the scenario’s `API_PORT` via `vrooli`, and talks to
  `http://localhost:<API_PORT>/api/v1` directly—no CLI dependency.
- Auto-creates (or reuses) a “Testing Harness” BAS project rooted at
  `<scenario>/data/projects/testing-harness` so imported workflows live outside the demo data.
- Imports JSON definitions with `POST /workflows/create`, executes them with
  `POST /workflows/{id}/execute`, then polls `/executions/{id}` until BAS reports the final status.
- Deletes temporary workflows through the bulk-delete endpoint when `--keep-workflow` is not set,
  keeping the database clean between test runs.
- Automatically starts/stops the scenario when `--manage-runtime auto` (default) and tolerates being
  sourced outside the phased shell helpers (it only calls `testing::core::*` helpers when present).
- Returns exit code **210** when `--allow-missing` is passed and the workflow JSON is absent, letting
  callers treat unexported flows as skips during migrations.
- Exposes execution metadata so phases can attach precise evidence:
  - `TESTING_PLAYBOOKS_BAS_LAST_WORKFLOW_ID`
  - `TESTING_PLAYBOOKS_BAS_LAST_EXECUTION_ID`
  - `TESTING_PLAYBOOKS_BAS_LAST_SCENARIO`
- Surfaces the raw execution JSON when BAS returns `failed/errored`, which makes it easy to include
  the Browserless error string in requirement evidence.

## Requirements Integration

Annotate requirement validations with the `automation` type and reference the workflow JSON path:

```yaml
validation:
  - type: automation
    ref: test/playbooks/capabilities/02-builder/01-projects/demo-sanity.json
    scenario: browser-automation-studio   # optional override
    status: planned
    notes: Replays the seeded demo workflow end-to-end.

  # Example pointing at an existing workflow stored inside BAS
  - type: automation
    workflow_id: WORKFLOW-123456
    phase: integration
    scenario: browser-automation-studio
    status: planned
    notes: Executes a shared workflow without committing JSON to the repo.
```

During `testing::phase::init`, these validations are exposed via
`testing::phase::expected_validations_for <requirement_id>` so phase scripts can iterate over
expected workflows without hardcoding filenames.

## Authoring Checklist

1. Build/record the workflow inside BAS.
2. Export the workflow JSON (via API or CLI) and store under `test/playbooks/`.
3. Link the workflow to requirement validations and phase scripts.
4. Commit supporting fixtures (mock servers, test data) alongside the workflow when necessary.
5. Regenerate exports whenever BAS node semantics change to keep tests honest.

Maintaining this contract keeps UI smoke coverage declarative: requirements declare which workflows prove them, and phases simply execute those workflows via the shared helper.

## Workflow Execution Flow

```mermaid
sequenceDiagram
    participant Phase as Test Phase
    participant Helper as BAS Helper
    participant API as BAS API
    participant Browser as Browserless

    Phase->>Helper: testing::phase::run_bas_automation_validations()
    Helper->>API: POST /workflows/create (JSON)
    API-->>Helper: workflow_id
    Helper->>API: POST /workflows/{id}/execute
    API->>Browser: Launch browser session
    Browser->>API: Execute workflow steps
    API->>Helper: Poll /executions/{id}
    Helper->>Helper: Check status (running → complete)
    Helper->>API: DELETE /workflows/{id}
    Helper-->>Phase: Status (passed/failed/skipped)
    Phase->>Phase: testing::phase::add_requirement()
```

## Troubleshooting

### Workflow Fails to Import

**Symptom**: Phase script errors with "Failed to import BAS workflow"

**Causes**:
1. JSON file doesn't exist at specified path
2. JSON syntax error (invalid JSON)
3. BAS scenario not running
4. Missing required fields in workflow JSON

**Solutions**:
```bash
# 1. Verify file exists
ls -la test/playbooks/ui/projects/create.json

# 2. Validate JSON syntax
jq . test/playbooks/ui/projects/create.json

# 3. Check BAS is running
vrooli scenario status browser-automation-studio

# 4. Check required fields
# Workflow must have: metadata, nodes, edges
```

### Selector Not Found

**Symptom**: Workflow fails with "selector not found" error during resolution or execution

**For @selector/ references:**

The error message will automatically tell you:
1. If manifest was auto-regenerated (selector truly doesn't exist in selectors.ts)
2. Similar selectors you might have meant
3. Exact steps to register a new selector

**Solution**: Register the selector in `ui/src/consts/selectors.ts` (or `ui/src/constants/selectors.ts`) first:

```typescript
const literalSelectors = {
  myFeature: {
    submitButton: "my-feature-submit-btn",  // testId value
  },
};
```

Then import and use it in your UI:

```typescript
import { selectors } from '@/consts/selectors'; // or '@constants/selectors' for BAS

<button data-testid={selectors.myFeature.submitButton}>Submit</button>
```

The manifest will auto-regenerate when you run tests again.

**For raw CSS selectors (legacy workflows):**

**Causes**:
1. `data-testid` doesn't exist in UI
2. Timing issue (element not loaded yet)
3. Modal/dialog not opened
4. Typo in selector

**Solutions**:
```json
// 1. Verify data-testid exists in UI code
// Check: <button data-testid="submit-btn">

// 2. Add wait before selector-dependent nodes
{
  "id": "wait-for-load",
  "type": "wait",
  "data": { "durationMs": 1500 }
}

// 3. Use waitForSelector in click/type nodes
{
  "type": "click",
  "data": {
    "selector": "[data-testid='submit-btn']",
    "waitForSelector": "[data-testid='submit-btn']",  // Waits up to timeout
    "timeoutMs": 10000
  }
}

// 4. Double-check selector syntax
"[data-testid='name']"  // ✅ Correct
"[data-testid=name]"    // ❌ Missing quotes
"data-testid='name'"    // ❌ Missing brackets
```

**Recommendation**: Migrate to `@selector/` references to prevent drift and get better error messages.

### Workflow Times Out

**Symptom**: Workflow execution exceeds timeout, returns incomplete

**Causes**:
1. Page never reaches `networkidle0`
2. Continuous polling/requests preventing idle state
3. Timeout too short for slow operations
4. Infinite loop in workflow

**Solutions**:
```json
// 1. Use different waitUntil strategy
{
  "type": "navigate",
  "data": {
    "waitUntil": "load"  // Instead of "networkidle0"
  }
}

// 2. Add explicit waits after navigation
{
  "type": "navigate",
  "data": {
    "waitUntil": "domcontentloaded",
    "waitForMs": 2000  // Additional wait
  }
}

// 3. Increase timeouts
{
  "data": {
    "timeoutMs": 60000  // 60 seconds for slow operations
  }
}

// 4. Check edges for cycles
// Edges should form DAG (directed acyclic graph), not loops
```

### Assert Node Fails

**Symptom**: Assert node fails despite element appearing correct visually

**Causes**:
1. Text assertion is case-sensitive
2. Element contains extra whitespace
3. Element not yet updated with expected content
4. Wrong assert mode

**Solutions**:
```json
// 1. Use contains_text instead of exact_text
{
  "assertMode": "contains_text",
  "expectedText": "Success"  // Matches "Success!" and "Success: Project created"
}

// 2. Trim expected text
{
  "expectedText": "Project Name"  // Matches "  Project Name  " with whitespace
}

// 3. Add wait before assertion
{
  "id": "wait-update",
  "type": "wait",
  "data": { "durationMs": 1000 }
}

// 4. Choose correct mode
"assertMode": "exists"         // Element in DOM
"assertMode": "contains_text"  // Text includes substring
"assertMode": "exact_text"     // Text exactly matches
```

### BAS Scenario Not Running

**Symptom**: Execution helper reports "Unable to resolve API_PORT"

**Causes**:
1. BAS scenario not started
2. Scenario crashed/stopped
3. Port allocation conflict

**Solutions**:
```bash
# 1. Start BAS
vrooli scenario start browser-automation-studio

# 2. Check status
vrooli scenario status browser-automation-studio

# 3. Check logs for errors
vrooli scenario logs browser-automation-studio

# 4. Restart if needed
vrooli scenario stop browser-automation-studio
vrooli scenario start browser-automation-studio --clean-stale
```

### Workflow Execution Shows Success but Requirement Fails

**Symptom**: Workflow executes successfully but requirement shows "failing"

**Causes**:
1. Requirement module is missing a `type: "automation"` validation pointing at the workflow
2. The registry is stale (workflow moved but `registry.json` was not regenerated)
3. Phase script skipped `testing::phase::add_requirement`

**Solutions**:
```json
// 1. Add automation validation to the requirement JSON
{
  "id": "BAS-PROJECT-CREATE",
  "validation": [
    {
      "type": "automation",
      "ref": "test/playbooks/capabilities/01-foundation/01-projects/new-project-create.json",
      "phase": "integration"
    }
  ]
}

// 2. Regenerate + inspect registry order/reset metadata
node scripts/scenarios/testing/playbooks/build-registry.mjs --scenario scenarios/browser-automation-studio
cat scenarios/browser-automation-studio/test/playbooks/registry.json | jq '.playbooks[] | select(.file | test("new-project-create"))'

// 3. Verify the integration phase uses the helper
testing::phase::run_bas_automation_validations --scenario "$SCENARIO_NAME"
```

If the requirement still fails, inspect `coverage/automation/test/playbooks/.../new-project-create.timeline.json` to see the exact step output and review the paired screenshot captured during failure.

## Logging & Failure Artifacts

- Successful workflows now emit a single summary line. Set `TESTING_PLAYBOOKS_VERBOSE=1` to stream every heartbeat/step in real time while authoring a new flow.
- When a workflow fails, the runner prints the buffered heartbeat log and writes two artifacts: `coverage/automation/<workflow>.timeline.json` (full timeline) and `coverage/automation/<workflow>.png` (latest screenshot if available). Inspect these files before rerunning slow suites.
- The registry lists each playbook's `reset` level. Use `"reset": "none"` for read-only flows; use `"reset": "full"` when mutations require completely reseeding Browser Automation Studio state.

## Reusable Subflows & Seed Data

- **Subflows:** Store shared setup/prelude workflows under `test/playbooks/__subflows/`. Each file must declare `metadata.fixture_id` (kebab-case slug). Reference the subflow from any `workflowCall` node by setting `workflowId` to `@fixture/<slug>(key=value, ...)`. The resolver validates parameter types (string/number/boolean/enum), substitutes `${fixture.param}` tokens throughout the nested flow, and throws if required parameters are missing. Strings containing spaces or punctuation must be quoted, while runtime store references can be passed as `@store/<key>`.
- **Fixture requirements:** When a fixture guarantees requirement coverage (for example, "Demo workflow is seeded"), list those IDs in `metadata.requirements`. The resolver bubbles them up to the calling workflow's `metadata.requirementsFromFixtures`, and the phase helper records pass/fail evidence for every propagated requirement automatically.
- **Validation:** Structural tests fail when subflows are missing `fixture_id`, define duplicate slugs, are never referenced, or when a playbook references a slug that does not exist.
- **Seed Data:** Keep deterministic seed scripts in `test/playbooks/__seeds/`. Provide an `apply.sh` to populate data (projects, workflows, etc.) and a matching `cleanup.sh` to remove it. The integration runner automatically executes `apply.sh` after Browser Automation Studio is healthy and always calls `cleanup.sh` at phase completion—even when tests fail—so the scenario returns to a clean state.
- **Portability:** All Vrooli scenarios that lean on BAS e2e workflows share the same conventions (`__subflows` + `__seeds`), which keeps automation projects portable and prevents stray workflows from polluting the main BAS authoring experience.
- **Canonical fixtures:** The Browser Automation Studio scenario ships `open-demo-project` (dashboard → Demo project detail), `open-builder-from-demo` (loads a fresh builder from that project), and `open-demo-workflow` (opens the seeded Demo workflow inside the builder). Reuse them instead of rebuilding those sequences in every workflow.

## See Also

### Related Guides
- **[Writing Testable UIs](writing-testable-uis.md)** - Use `data-testid`, design for automation
- **[End-to-End Example](end-to-end-example.md)** - Complete flow from PRD to BAS workflow
- **[Requirement Tracking](requirement-tracking.md)** - Understanding `automation` validation type
- **[Scenario Testing](scenario-testing.md)** - Complete testing workflow
- **[Phased Testing Architecture](../architecture/PHASED_TESTING.md)** - How BAS integrates with phases

### Reference Implementation
- **[browser-automation-studio scenario](../../../scenarios/browser-automation-studio/)** - Self-testing example
- **[Example workflows](../../../scenarios/browser-automation-studio/test/playbooks/)** - Real workflow JSON files
- **[demo-sanity.json](../../../scenarios/browser-automation-studio/test/playbooks/projects/demo-sanity.json)** - Simple workflow example

### Tools
- **[browser-automation-studio.sh](../../../scripts/scenarios/testing/playbooks/browser-automation-studio.sh)** - Execution helper source code

---

**Remember**: BAS workflows are JSON data structures. AI agents can write them directly without learning Playwright APIs or handling async patterns.
