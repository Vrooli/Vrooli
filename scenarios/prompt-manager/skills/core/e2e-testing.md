## Steer focus: E2E Testing with Browser Automation Studio

Prioritize **end-to-end test coverage** for critical user journeys using BAS workflows.
Do **not** break functionality or regress existing tests; all changes must maintain or improve completeness.

Focus on validating **requirements and user-visible behavior**, not implementation details.

For CLI commands and artifact analysis, see the **browser-automation-studio** skill.

Optional reading:
- `prompt-manager skills read browser-automation-studio`

---

### **1. When to Use E2E Testing**

#### Decision Table

| Question | If YES | If NO |
|----------|--------|-------|
| Does it involve user-visible UI interaction? | E2E candidate | Unit/integration test |
| Does it span multiple components/services? | E2E candidate | Unit test |
| Is it a critical user journey (login, checkout, onboarding)? | E2E required | Consider cost/benefit |
| Can it be fully validated without a browser? | Unit/integration test | E2E candidate |
| Is it testing API contracts only? | Integration test | Continue evaluation |

#### Testing Pyramid Position

```
          ╱╲
         ╱E2E╲           ← BAS workflows (few, critical paths)
        ╱──────╲
       ╱Integration╲      ← API tests, component integration
      ╱──────────────╲
     ╱   Unit Tests    ╲   ← Most tests live here (fast, isolated)
    ╱────────────────────╲
```

**Principle:** E2E tests validate that the system works **for users**. They should be:
- **Few** - Only critical paths
- **Focused** - One journey per test case
- **Supplemental** - Not replacing faster unit/integration tests

---

### **2. Workflow Hierarchy & Organization**

```
scenarios/{{TARGET}}/bas/
├── registry.json          # Auto-generated manifest (DO NOT edit)
├── cases/                 # Test cases WITH assertions
│   ├── 01-foundation/     # Core functionality
│   │   ├── 01-auth/       # Authentication flows
│   │   │   ├── login-success.json
│   │   │   └── login-failure.json
│   │   └── 02-navigation/ # Basic navigation
│   └── 02-features/       # Feature-specific tests
│       └── 01-dashboard/
├── flows/                 # Reusable user journeys (NO assertions)
│   └── complete-checkout.json
└── actions/               # Atomic operations (NO assertions)
    ├── login.json
    └── open-project.json
```

#### Hierarchy Rules

| Level | Contains Assertions | Reusable | Purpose |
|-------|---------------------|----------|---------|
| `actions/` | NO | YES | Atomic steps (login, navigate, fill form) |
| `flows/` | NO | YES | Multi-step journeys (checkout process) |
| `cases/` | YES | NO | Requirement validation with assertions |

#### Ordering Convention

- Use **two-digit prefixes** for folders: `01-foundation/`, `02-features/`
- This controls execution order during test runs
- Foundation tests run first to catch basic failures early

#### Debug Order (Critical)

When test suites fail, debug **bottom-up**:

1. **Actions first** - Are atomic operations working?
2. **Flows second** - Do composed journeys complete?
3. **Cases last** - Are assertions correct for the current behavior?

---

### **3. Selector Registry Integration**

All UI selectors **must** be defined in a central registry to ensure maintainability.

#### Single Source of Truth

```typescript
// ui/src/constants/selectors.ts
const literalSelectors = {
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

#### Component Usage

```tsx
import { selectors } from '@/constants/selectors';

<button data-testid={selectors.dashboard.newProjectButton}>
  New Project
</button>
```

#### Workflow Reference

In BAS workflows, reference selectors with `@selector/` prefix:

```json
{
  "type": "click",
  "data": {
    "selector": "@selector/dashboard.newProjectButton"
  }
}
```

#### Dynamic Selectors

For elements that need parameters:

```typescript
// In selectors.ts
const dynamicSelectorDefinitions = {
  projects: {
    cardByName: defineDynamicSelector({
      description: "Project card by name",
      selectorPattern: '[data-testid="project-card"][data-project-name="${name}"]',
      params: { name: { type: "string" } },
    }),
  },
} as const;
```

In workflows:

```json
{
  "type": "click",
  "data": {
    "selector": "@selector/projects.cardByName(name=My Project)"
  }
}
```

**Benefits:**
- Workflows survive UI refactors
- Single place to update when selectors change
- Type-safe references catch typos early

---

### **4. Requirements Integration**

Link E2E tests to requirements using the `automation` validation type.

#### In Requirements JSON

```json
{
  "id": "REQ-AUTH-001",
  "description": "User can log in with valid credentials",
  "validation": [
    {
      "type": "automation",
      "ref": "bas/cases/01-foundation/01-auth/login-success.json",
      "scenario": "{{TARGET}}",
      "phase": "integration",
      "status": "implemented"
    }
  ]
}
```

#### In Workflow Metadata

```json
{
  "metadata": {
    "description": "Verify successful login flow",
    "labels": {
      "requirements_json": "[\"REQ-AUTH-001\"]"
    }
  }
}
```

#### Validation Types

| Type | Tool | When to Use |
|------|------|-------------|
| `automation` | BAS workflow | UI interactions, visual validation |
| `test` | Unit/integration test | Code-level behavior |
| `manual` | Human verification | Last resort only |

**If using manual validation**, log it with:
```bash
vrooli scenario requirements manual-log {{TARGET}} REQ-XXX "Reason for manual check"
```
Then build a BAS workflow as soon as possible.

---

### **5. Workflow Authoring Patterns**

#### Minimal Workflow Structure

```json
{
  "metadata": {
    "description": "Verify login success redirects to dashboard",
    "version": 1,
    "reset": "none"
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
      "id": "navigate-login",
      "type": "navigate",
      "position": { "x": 0, "y": 0 },
      "data": {
        "label": "Go to login page",
        "destinationType": "scenario",
        "scenario": "{{TARGET}}",
        "scenarioPath": "/login",
        "waitUntil": "networkidle0"
      }
    },
    {
      "id": "type-email",
      "type": "type",
      "position": { "x": 220, "y": 0 },
      "data": {
        "label": "Enter email",
        "selector": "@selector/auth.emailInput",
        "text": "test@example.com",
        "clearExisting": true
      }
    },
    {
      "id": "type-password",
      "type": "type",
      "position": { "x": 440, "y": 0 },
      "data": {
        "label": "Enter password",
        "selector": "@selector/auth.passwordInput",
        "text": "password123",
        "clearExisting": true
      }
    },
    {
      "id": "click-submit",
      "type": "click",
      "position": { "x": 660, "y": 0 },
      "data": {
        "label": "Click submit",
        "selector": "@selector/auth.submitButton"
      }
    },
    {
      "id": "assert-dashboard",
      "type": "assert",
      "position": { "x": 880, "y": 0 },
      "data": {
        "label": "Verify dashboard visible",
        "selector": "@selector/dashboard.container",
        "assertMode": "exists",
        "timeoutMs": 10000,
        "failureMessage": "Dashboard should be visible after login"
      }
    }
  ],
  "edges": [
    { "id": "e1", "source": "navigate-login", "target": "type-email", "type": "smoothstep" },
    { "id": "e2", "source": "type-email", "target": "type-password", "type": "smoothstep" },
    { "id": "e3", "source": "type-password", "target": "click-submit", "type": "smoothstep" },
    { "id": "e4", "source": "click-submit", "target": "assert-dashboard", "type": "smoothstep" }
  ]
}
```

#### Node Types Quick Reference

| Type | Purpose | Key Data Fields |
|------|---------|-----------------|
| `navigate` | Go to URL/scenario | `scenario`, `scenarioPath`, `waitUntil` |
| `click` | Click element | `selector`, `clickCount` |
| `type` | Enter text | `selector`, `text`, `clearExisting` |
| `assert` | Verify condition | `selector`, `assertMode`, `expectedText` |
| `wait` | Pause execution | `durationMs` or `selector` + `state` |
| `screenshot` | Capture state | `fullPage`, `captureDomSnapshot` |
| `evaluate` | Run JavaScript | `expression`, `store_result` |

#### Assert Modes

| Mode | Validates |
|------|-----------|
| `exists` | Element is present in DOM |
| `not_exists` | Element is NOT in DOM |
| `visible` | Element is visible (not hidden) |
| `contains_text` | Element contains expected text |
| `exact_text` | Element text matches exactly |

#### Resilience Settings

For flaky elements or slow pages:

```json
{
  "type": "click",
  "data": {
    "selector": "@selector/cta.primary",
    "resilience": {
      "maxAttempts": 3,
      "delayMs": 1500,
      "backoffFactor": 1.5,
      "preconditionSelector": "@selector/app.ready",
      "preconditionTimeoutMs": 10000,
      "successSelector": "@selector/nextStep.visible",
      "successTimeoutMs": 15000
    }
  }
}
```

---

### **6. Memory Management with Visited Tracker**

Track E2E test coverage systematically across sessions:

**At session start:**
```bash
# Get least-visited workflow files
visited-tracker least-visited \
  --location scenarios/{{TARGET}}/bas \
  --pattern "**/*.json" \
  --tag e2e \
  --name "{{TARGET}} - E2E Coverage" \
  --limit 5
```

**After analyzing/updating a workflow:**
```bash
visited-tracker visit <workflow-path> \
  --location scenarios/{{TARGET}}/bas \
  --tag e2e \
  --note "<summary: assertions added, coverage gaps, linked requirements>"
```

**When workflow is complete:**
```bash
visited-tracker exclude <workflow-path> \
  --location scenarios/{{TARGET}}/bas \
  --tag e2e \
  --reason "Comprehensive coverage - all assertions validated against requirements"
```

**Before ending session:**
```bash
visited-tracker campaigns note \
  --location scenarios/{{TARGET}}/bas \
  --tag e2e \
  --name "{{TARGET}} - E2E Coverage" \
  --note "<progress summary, remaining gaps, priority areas>"
```

---

### **7. Maintain Scenario Constraints**

* Do **not** change core business logic to make tests pass
* Do **not** weaken assertions to avoid failures
* Do **not** skip flaky tests without documenting root cause
* Prefer **fixing the issue** over working around it in tests
* E2E tests must validate **desired behavior**, not current (possibly buggy) behavior
* All selectors must go through the registry - no hardcoded `[data-testid="..."]` in workflows

---

### **8. Documentation**

Update the **E2E Issues** section of `docs/internal/PROBLEMS.md`:

* The code is the source of truth. Verify existing claims before extending.
* Create `docs/internal/` directory if needed.

Include:
* Critical user journeys lacking coverage
* Flaky tests and their root causes
* Selector registry gaps
* Requirement validation status
* Execution IDs for reproducibility

---

### **9. Output Expectations**

You may update:
* Workflow JSON files in `bas/cases/`, `bas/flows/`, `bas/actions/`
* Selector registry in `ui/src/constants/selectors.ts`
* Component `data-testid` attributes
* Requirements validation references
* Test documentation in `docs/internal/PROBLEMS.md`

You **must**:
* Keep workflows focused on user-visible behavior
* Link workflows to requirements where applicable
* Use selector registry (never hardcode selectors in workflows)
* Document coverage gaps
* Avoid testing implementation details
* Maintain debug order: actions → flows → cases

Focus this loop on delivering **practical, high-impact e2e coverage** that validates critical user journeys and catches regressions before they reach users.

**Avoid superficial workflows that increase coverage numbers without validating real user behavior. Only add tests that would catch meaningful regressions.**
