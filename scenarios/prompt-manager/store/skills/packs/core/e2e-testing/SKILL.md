## Steer focus: E2E Testing with Browser Automation Studio

Prioritize **end-to-end test coverage** for critical user journeys using BAS workflows.
Do **not** break functionality or regress existing tests; all changes must maintain or improve completeness.

Focus on validating **requirements and user-visible behavior**, not implementation details.

For CLI commands and artifact analysis, see the **browser-automation-studio** skill.

Required reading:
- `prompt-manager skill read visited-tracker-tools knowledge-observatory-tools`

Optional reading:
- `prompt-manager skill read browser-automation-studio`

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

#### Acceptance-Criteria Wording (Gherkin)

Write workflow `metadata.description` values and any acceptance criteria in
**Given/When/Then** form: "Given a logged-out user on /login, when they submit
valid credentials, then the dashboard heading is visible." The shape forces
you to name the starting state, the action, and the observable outcome — the
three things a workflow must encode anyway — and it maps one-to-one onto the
workflow's setup steps, action steps, and assertions. Avoid intent-free
descriptions ("test login works").

One anti-pattern to refuse: after a feature is **removed**, do not author a
workflow asserting the old UI is gone (a tombstone test). Delete the old
workflow, update the requirement, and cover the replacement behavior
positively. An ongoing prohibition (e.g. a locked-out user must not reach the
dashboard) is different — write it as a positive assertion on the observable
response (the lockout message is visible).

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

### **6. Authenticated Testing Patterns**

For tests requiring authentication, use **shared sign-in actions** rather than persisted session profiles. This ensures test isolation and reproducibility.

#### Shared Sign-In Action Pattern

Sign-in is an **atomic operation** that lives in `actions/login.json`:

```json
{
  "metadata": {
    "description": "Sign in with provided credentials",
    "version": 1
  },
  "nodes": [
    {
      "id": "navigate-login",
      "type": "navigate",
      "data": { "destinationType": "scenario", "scenarioPath": "/login" }
    },
    {
      "id": "type-email",
      "type": "type",
      "data": { "selector": "@selector/auth.emailInput", "text": "${@params/username}", "clearExisting": true }
    },
    {
      "id": "type-password",
      "type": "type",
      "data": { "selector": "@selector/auth.passwordInput", "text": "${@params/password}", "clearExisting": true }
    },
    {
      "id": "click-submit",
      "type": "click",
      "data": { "selector": "@selector/auth.submitButton" }
    },
    {
      "id": "wait-redirect",
      "type": "wait",
      "data": { "selector": "@selector/dashboard.container", "state": "visible", "timeoutMs": 10000 }
    }
  ],
  "edges": [
    { "id": "e1", "source": "navigate-login", "target": "type-email" },
    { "id": "e2", "source": "type-email", "target": "type-password" },
    { "id": "e3", "source": "type-password", "target": "click-submit" },
    { "id": "e4", "source": "click-submit", "target": "wait-redirect" }
  ]
}
```

#### Test Cases Call Sign-In as Subflow

Cases that need authentication call the shared action:

```json
{
  "nodes": [
    {
      "id": "sign-in",
      "type": "subflow",
      "data": {
        "workflowPath": "actions/login.json",
        "params": {
          "username": "${@params/username}",
          "password": "${@params/password}"
        }
      }
    },
    {
      "id": "assert-dashboard",
      "type": "assert",
      "data": { "selector": "@selector/dashboard.container", "assertMode": "exists" }
    }
  ],
  "edges": [
    { "id": "e1", "source": "sign-in", "target": "assert-dashboard" }
  ]
}
```

#### Test Isolation

- Each test run gets a **fresh browser context** (no session profiles)
- Credentials passed via `--initial-params`
- Session state is **not persisted** between test runs

#### CLI Usage

```bash
browser-automation-studio workflow execute \
  --from-file bas/cases/01-foundation/01-auth/login-success.json \
  --initial-params '{"username":"test@example.com","password":"secret"}' \
  --wait
```

#### Example Directory Structure

```
bas/
  actions/
    login.json                    # Atomic sign-in action (params: username, password)
    logout.json
    open-project.json
  flows/
    complete-checkout.json        # Multi-step journey using actions
  cases/
    01-foundation/01-auth/
      login-success.json          # Calls actions/login.json, then asserts dashboard
      login-failure.json          # Tests invalid credentials
    02-features/
      admin-dashboard.json        # Calls login with admin creds, tests admin features
```

#### Why Not Session Profiles for Tests?

| Session Profiles | Subflow-Based Auth |
|------------------|-------------------|
| State may be stale (expired cookies) | Always fresh login |
| Test depends on profile existence | Test is self-contained |
| Harder to debug failures | Full login flow in trace |
| Not CI-friendly | Works anywhere |

**Rule:** Session profiles are for **manual testing and debugging**, not automated test suites.

---

### **7. Memory Management with Visited Tracker**

Use the `visited-tracker-tools` skill for tracking visited files, with LOCATION set to `scenarios/{{TARGET}}/bas` and TAG set to `e2e`.

---

### **8. Documentation**

Use `knowledge-observatory-tools` to read the current `problems` doc for `{{TARGET}}`, then update the **E2E Issues** section with your findings (critical user journeys lacking coverage, flaky tests and root causes, selector registry gaps, requirement validation status).

---

### **9. Database Isolation (Automatic)**

When test-genie runs your playbooks, the scenario's database is automatically isolated — every test run hits a per-run test database, not the real one. **You write your workflows exactly the same way** regardless; nothing about the BAS JSON changes.

How it works (two paths, test-genie picks one):

| Path | What happens | When |
|---|---|---|
| **Routed** | A per-run test pool is installed on the running scenario via RPC. test-genie injects `X-Vrooli-Test-Mode: 1` as a browser-context header so every UI→API request from your playbook hits the test pool. No restart. | Scenario uses `*database.RoutedDB` + mounts `apihttp.TestModeMiddleware` (the `react-vite` template ships in this shape — new scenarios get it for free). |
| **Fallback** | Scenario is stopped, restarted with env pointing at the test DB, playbooks run, then restarted normally. | Scenario still holds raw `*sql.DB` handles or otherwise can't be routed. |

What this means for test authors:

- **Seed data**: any `bas/seeds/seed.go` runs against the test DB, not prod. Seed freely — it won't leak.
- **Mutations are safe**: workflows can create, update, delete — the changes go to the test DB and are torn down after the run.
- **Don't try to set the test-mode header yourself** in workflow JSON. It's already attached at the browser context for every request; doing it again per-step is redundant and confusing.
- **If your test only passes against prod data**, it's not really an E2E test — it's an observation. Fix it: seed what you need.

Full details (mode flag, opt-in for new scenarios, lease/concurrency model) live in `path:scenarios/storage-health/docs/concepts/test-isolation-contract.md`. Test authors usually don't need to read it.

---

### **10. Output Expectations**

You may update:
* Workflow JSON files in `path:bas/cases/`, `path:bas/flows/`, `path:bas/actions/`
* Selector registry in `ui/src/constants/selectors.ts`
* Component `data-testid` attributes
* Requirements validation references
* Test documentation via `knowledge-observatory-tools`

You **must**:
* Keep workflows focused on user-visible behavior
* Link workflows to requirements where applicable
* Use selector registry (never hardcode selectors in workflows)
* Document coverage gaps
* Avoid testing implementation details
* Maintain debug order: actions → flows → cases

Focus this loop on delivering **practical, high-impact e2e coverage** that validates critical user journeys and catches regressions before they reach users.

**Avoid superficial workflows that increase coverage numbers without validating real user behavior. Only add tests that would catch meaningful regressions.**
