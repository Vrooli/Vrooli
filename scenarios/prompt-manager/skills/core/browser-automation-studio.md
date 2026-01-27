## Steer focus: Browser Automation Studio

Reference for using Browser Automation Studio (BAS) to execute browser workflows, validate UI behavior, and debug automation failures.

BAS is a **browser automation tool** that lets you:
- Run smoke tests to verify pages load correctly
- Validate that UI elements exist and behave as expected
- Execute multi-step user journeys
- Capture screenshots and artifacts for debugging

This skill covers **tool usage and testability setup**. For e2e testing strategy, workflow organization patterns, and requirements integration, see the **e2e-testing** skill.

Optional reading:
- `prompt-manager skills read e2e-testing`

---

### **1. When to Use BAS**

```
                    What do you need to verify?
                              │
          ┌───────────────────┼───────────────────┐
          │                   │                   │
          ▼                   ▼                   ▼
    Page loads?        Element exists?      User journey?
          │                   │                   │
          ▼                   ▼                   ▼
    Smoke test         Element check        Flow execution
    (navigate +        (navigate +          (multi-step
     screenshot)        assert node)         workflow)
```

| Scenario | BAS Approach |
|----------|--------------|
| Verify a page loads without errors | Smoke test: navigate + screenshot |
| Check a button/form/element exists | Element check: navigate + assert |
| Test login → dashboard flow | Flow execution: multi-step workflow |
| Debug why a UI test failed | Artifact analysis: screenshots + logs |
| Validate a bug fix works | Regression test: targeted workflow |

**When NOT to use BAS:**
- Pure API testing (use integration tests)
- Unit-level logic (use unit tests)
- Performance benchmarking (use dedicated tools)

---

### **2. Setting Up Testability**

For BAS to reliably interact with UI elements, components need stable selectors.

#### Selector Registry (Single Source of Truth)

Every scenario should have a selector registry at `ui/src/constants/selectors.ts`:

```typescript
// ui/src/constants/selectors.ts
export const selectors = {
  dashboard: {
    container: "dashboard-container",
    newProjectButton: "dashboard-new-project-button",
    projectList: "dashboard-project-list",
  },
  auth: {
    loginForm: "auth-login-form",
    emailInput: "auth-email-input",
    passwordInput: "auth-password-input",
    submitButton: "auth-submit-button",
    errorMessage: "auth-error-message",
  },
} as const;
```

#### Adding Selectors to Components

```tsx
import { selectors } from '@/constants/selectors';

// In your component:
<button data-testid={selectors.dashboard.newProjectButton}>
  New Project
</button>
```

#### Selector Naming Conventions

| Pattern | Example | Use When |
|---------|---------|----------|
| `{area}-{element}` | `dashboard-container` | Static elements |
| `{area}-{element}-{variant}` | `auth-submit-button` | Differentiated elements |
| `{area}-{item}-{index}` | `project-card-0` | List items |

**Key principles:**
- Never hardcode `data-testid` strings in workflows - always use the registry
- Add selectors for any element BAS needs to interact with or verify
- Use semantic names that describe purpose, not implementation

#### Referencing Selectors in Workflows

In BAS workflow JSON, reference selectors with the `@selector/` prefix:

```json
{
  "type": "click",
  "data": {
    "selector": "@selector/dashboard.newProjectButton"
  }
}
```

This indirection means workflows survive UI refactors - update the registry once, all workflows follow.

---

### **3. Workflow Location & Structure**

#### Where Workflows Live

```
scenarios/{{TARGET}}/bas/
├── registry.json       # Auto-generated manifest (DO NOT edit manually)
├── actions/            # Atomic reusable steps (NO assertions)
│   ├── login.json
│   └── open-project.json
├── flows/              # User journeys composing actions (NO assertions)
│   └── checkout-flow.json
└── cases/              # Test cases WITH assertions
    ├── 01-foundation/
    │   └── 01-auth/
    │       └── login-success.json
    └── 02-features/
```

#### Hierarchy Quick Reference

| Directory | Contains Assertions | Reusable | Purpose |
|-----------|---------------------|----------|---------|
| `actions/` | NO | YES | Single operations (login, click, fill) |
| `flows/` | NO | YES | Multi-step journeys |
| `cases/` | YES | NO | Requirement validation |

See **e2e-testing** skill for detailed organization patterns and requirements integration.

#### Minimal Workflow Anatomy

Every workflow has three parts:

```json
{
  "metadata": {
    "description": "What this workflow validates",
    "version": 1
  },
  "nodes": [
    { "id": "step-1", "type": "navigate", "data": { "..." } },
    { "id": "step-2", "type": "screenshot", "data": { "..." } }
  ],
  "edges": [
    { "source": "step-1", "target": "step-2" }
  ]
}
```

---

### **4. Core CLI Commands**

#### Execute a Workflow

```bash
# Run workflow and wait for completion
browser-automation-studio workflow execute \
  --from-file scenarios/{{TARGET}}/bas/cases/01-foundation/login.json \
  --wait

# Run with a starting URL (for workflows without navigate node)
browser-automation-studio workflow execute \
  --from-file scenarios/{{TARGET}}/bas/actions/open-project.json \
  --start-url http://localhost:3000/ \
  --wait

# Run with parameters
browser-automation-studio workflow execute \
  --from-file scenarios/{{TARGET}}/bas/flows/checkout.json \
  --params '{"username": "test@example.com"}' \
  --wait

# Run with video/trace recording
browser-automation-studio workflow execute \
  --from-file scenarios/{{TARGET}}/bas/cases/01-foundation/login.json \
  --record-video \
  --wait
```

#### Check Execution Status

```bash
# List all executions
browser-automation-studio execution list

# List only failed executions
browser-automation-studio execution list --filter failed

# List running executions
browser-automation-studio execution list --filter running
```

#### Export Results

```bash
# Export execution data as JSON
browser-automation-studio execution export <execution-id> --output result.json

# Export to directory (preserves full structure)
browser-automation-studio execution export <execution-id> --output-dir ./exports/my-run

# Generate HTML replay viewer
browser-automation-studio execution render <execution-id> --output ./replay-dir
```

---

### **5. Understanding Execution Results**

#### Artifact Location

All execution artifacts are stored in:

```
scenarios/browser-automation-studio/data/recordings/{executionId}/
├── result.json                # Final outcome (pass/fail, error messages)
├── timeline.json              # Step-by-step execution log
├── frames/                    # Screenshots captured at each step
│   ├── screenshot-001.jpg
│   ├── screenshot-002.jpg
│   └── ...
└── artifacts/
    ├── console-{stepId}.json  # Browser console logs
    ├── network-{stepId}.json  # Network requests/responses
    └── dom-{stepId}.json      # DOM snapshots
```

#### Quick Inspection Commands

```bash
# Check if execution passed or failed
cat scenarios/browser-automation-studio/data/recordings/<id>/result.json | jq '.status'

# View error message on failure
cat scenarios/browser-automation-studio/data/recordings/<id>/result.json | jq '.error'

# Find which step failed
cat scenarios/browser-automation-studio/data/recordings/<id>/timeline.json | jq '.steps[] | select(.status == "failed")'

# View the last screenshot (often shows failure state)
ls -t scenarios/browser-automation-studio/data/recordings/<id>/frames/*.jpg | head -1

# Check for JavaScript errors
cat scenarios/browser-automation-studio/data/recordings/<id>/artifacts/console-*.json | jq '.[] | select(.level == "error")'

# Check for failed network requests
cat scenarios/browser-automation-studio/data/recordings/<id>/artifacts/network-*.json | jq '.[] | select(.status >= 400)'
```

#### Interpreting result.json

```json
{
  "status": "failed",           // "completed" or "failed"
  "error": "SELECTOR_NOT_FOUND: @selector/auth.submitButton",
  "failedNodeId": "click-submit",
  "executionTimeMs": 4523,
  "screenshotCount": 3
}
```

---

### **6. Complete Example: Smoke Test Walkthrough**

This example shows the full cycle: create a workflow, run it, and interpret results.

#### Step 1: Create the Workflow

Save this to `/tmp/smoke-test.json`:

```json
{
  "metadata": {
    "description": "Smoke test - verify dashboard loads",
    "version": 1
  },
  "nodes": [
    {
      "id": "navigate",
      "type": "navigate",
      "position": { "x": 0, "y": 0 },
      "data": {
        "label": "Go to dashboard",
        "destinationType": "scenario",
        "scenario": "{{TARGET}}",
        "scenarioPath": "/dashboard",
        "waitUntil": "networkidle0"
      }
    },
    {
      "id": "screenshot",
      "type": "screenshot",
      "position": { "x": 220, "y": 0 },
      "data": {
        "label": "Capture page state",
        "fullPage": true
      }
    }
  ],
  "edges": [
    { "id": "e1", "source": "navigate", "target": "screenshot" }
  ]
}
```

#### Step 2: Run the Workflow

```bash
browser-automation-studio workflow execute \
  --from-file /tmp/smoke-test.json \
  --wait
```

Output will include the execution ID:

```
Execution started: exec-abc123
Waiting for completion...
✓ Execution completed: exec-abc123
Status: completed
```

#### Step 3: View the Screenshot

```bash
# Find the screenshot
ls scenarios/browser-automation-studio/data/recordings/exec-abc123/frames/

# Open it (or use your preferred image viewer)
open scenarios/browser-automation-studio/data/recordings/exec-abc123/frames/screenshot-001.jpg
```

#### Step 4: Check the Result

```bash
cat scenarios/browser-automation-studio/data/recordings/exec-abc123/result.json | jq .
```

If successful:
```json
{
  "status": "completed",
  "executionTimeMs": 2341,
  "screenshotCount": 1
}
```

---

### **7. Common Patterns**

#### Verify Element Exists

```json
{
  "metadata": {
    "description": "Verify submit button is present",
    "version": 1
  },
  "nodes": [
    {
      "id": "navigate",
      "type": "navigate",
      "data": {
        "destinationType": "scenario",
        "scenario": "{{TARGET}}",
        "scenarioPath": "/login"
      }
    },
    {
      "id": "assert-button",
      "type": "assert",
      "data": {
        "selector": "@selector/auth.submitButton",
        "assertMode": "exists",
        "timeoutMs": 5000,
        "failureMessage": "Submit button should be present on login page"
      }
    }
  ],
  "edges": [
    { "source": "navigate", "target": "assert-button" }
  ]
}
```

#### Verify Element Contains Text

```json
{
  "id": "assert-heading",
  "type": "assert",
  "data": {
    "selector": "@selector/dashboard.heading",
    "assertMode": "contains_text",
    "expectedText": "Welcome",
    "timeoutMs": 5000
  }
}
```

#### Wait for Element Before Acting

```json
{
  "id": "wait-ready",
  "type": "wait",
  "data": {
    "selector": "@selector/app.loadingSpinner",
    "state": "hidden",
    "timeoutMs": 10000
  }
}
```

#### Node Types Reference

| Type | Purpose | Key Fields |
|------|---------|------------|
| `navigate` | Go to URL | `scenario`, `scenarioPath`, `waitUntil` |
| `click` | Click element | `selector` |
| `type` | Enter text | `selector`, `text`, `clearExisting` |
| `assert` | Verify condition | `selector`, `assertMode`, `expectedText` |
| `wait` | Wait for state | `selector`, `state`, `timeoutMs` |
| `screenshot` | Capture image | `fullPage` |

#### Assert Modes

| Mode | Validates |
|------|-----------|
| `exists` | Element is in DOM |
| `not_exists` | Element is NOT in DOM |
| `visible` | Element is visible |
| `contains_text` | Element contains expected text |
| `exact_text` | Element text matches exactly |

---

### **8. Debugging Failures**

```
                    Workflow failed?
                          │
          ┌───────────────┴───────────────┐
         YES                              NO
          │                                │
          ▼                                ▼
   Check result.json                 Success - done
   for error type
          │
          ▼
   ┌──────────────────────────────────────────────────────┐
   │                    Error Type?                        │
   ├──────────────────────────────────────────────────────┤
   │                                                       │
   │  "SELECTOR_NOT_FOUND"                                 │
   │    1. Check selector exists in selectors.ts           │
   │    2. Verify data-testid is on the component          │
   │    3. View last screenshot - is element visible?      │
   │    4. Check if element is in iframe/shadow DOM        │
   │                                                       │
   ├──────────────────────────────────────────────────────┤
   │                                                       │
   │  "TIMEOUT"                                            │
   │    1. Check network-*.json for slow/failed requests   │
   │    2. View frames/ for last screenshot                │
   │    3. Check console-*.json for JS errors              │
   │    4. Increase timeoutMs in the failing node          │
   │    5. Add wait node before the action                 │
   │                                                       │
   ├──────────────────────────────────────────────────────┤
   │                                                       │
   │  "ASSERTION_FAILED"                                   │
   │    1. Check expectedText vs actual in result.json     │
   │    2. View DOM snapshot at failing step               │
   │    3. Verify selector targets correct element         │
   │    4. Check if content is dynamic/async loaded        │
   │                                                       │
   ├──────────────────────────────────────────────────────┤
   │                                                       │
   │  "NAVIGATION_ERROR"                                   │
   │    1. Check network-*.json for 4xx/5xx responses      │
   │    2. Verify scenario is running:                     │
   │       vrooli scenario status {{TARGET}}               │
   │    3. Check console-*.json for JS errors on load      │
   │    4. Verify the URL/route exists in the application  │
   │                                                       │
   └──────────────────────────────────────────────────────┘
```

#### Debug Order (Critical)

When tests fail, debug **bottom-up through the hierarchy**:

1. **Actions first** - Verify atomic steps work in isolation
2. **Flows second** - Verify composed journeys complete
3. **Cases last** - Verify assertions match expected behavior

This isolates failures to the smallest reproducible unit.

---

### **9. When to Create or Update Workflows**

| Situation | Action |
|-----------|--------|
| New feature added | Create smoke test (navigate + screenshot) |
| New user journey | Create flow in `flows/` |
| New requirement to validate | Create case in `cases/` with assertions |
| Bug fix for UI issue | Add regression test targeting the fix |
| UI refactor | Update selector registry, verify existing workflows pass |
| Selector changed | Update `selectors.ts`, workflows auto-update via `@selector/` |
| Flaky test | Investigate root cause, add wait nodes or increase timeouts |

**Workflow creation priority:**
1. Critical user journeys (login, checkout, core features)
2. Areas with recent bugs
3. Complex interactions prone to regression
4. New features as they're built

---

### **10. Maintain Scenario Constraints**

* This skill is for **using** BAS to investigate and validate, not modifying BAS itself
* Do **not** edit `bas/registry.json` manually - it's auto-generated
* Do **not** hardcode selectors in workflows - always use `@selector/` references
* Do **not** modify scenario business logic to make tests pass
* Use artifacts to **diagnose** issues, then fix in source code appropriately

---

### **11. Output Expectations**

When using BAS for investigation, document findings in `docs/internal/PROBLEMS.md` under the E2E Issues section:

**Template:**
```markdown
### [Issue Title]
- **Execution ID:** exec-abc123
- **Screenshot:** data/recordings/exec-abc123/frames/screenshot-005.jpg
- **Root cause:** [What's actually wrong]
- **Fix:** [What needs to change in source code]
- **Status:** Pending/Fixed
```

You may:
* Execute workflows to validate UI behavior
* Analyze artifacts to diagnose failures
* Create temporary workflows in `/tmp/` for debugging
* Update selector registry when adding testability
* Add `data-testid` attributes to components

You **must**:
* Document findings with execution IDs for reproducibility
* Link artifacts (screenshots, logs) in documentation
* Identify root causes, not just symptoms
* Provide actionable remediation steps
* Use selector registry for all new selectors
