## Steer focus: Browser Automation Studio

Reference for using Browser Automation Studio (BAS) to execute browser workflows, validate UI behavior, and debug automation failures.

BAS is a **browser automation tool** that lets you:
- Run smoke tests to verify pages load correctly
- Validate that UI elements exist and behave as expected
- Execute multi-step user journeys
- Capture screenshots and artifacts for debugging

This skill covers **tool usage**. For e2e testing strategy, workflow organization, selector registry setup, and requirements integration, see the **e2e-testing** skill.

Required reading:
- `prompt-manager skill read knowledge-observatory-tools`

Optional reading:
- `prompt-manager skill read e2e-testing`

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

### **2. Selector References**

BAS workflows use `@selector/` references to target UI elements via `data-testid` attributes. This indirection means workflows survive UI refactors.

```json
{
  "type": "click",
  "data": {
    "selector": "@selector/dashboard.newProjectButton"
  }
}
```

For selector registry setup, naming conventions, and component integration, see the **e2e-testing** skill.

---

### **3. Workflow Location & Structure**

Workflows live in `scenarios/{{TARGET}}/bas/` (where {{TARGET}} is the scenario you're testing) organized into `actions/`, `flows/`, and `cases/` directories. Use `browser-automation-studio schema workflow` to get the current workflow structure.

For organization patterns, hierarchy rules, and requirements integration, see the **e2e-testing** skill.

---

### **4. Core CLI Commands**

#### Single-location capture (prefer for one-page checks)

When all you need is a snapshot of one page — a screenshot, the JS console output, the network requests, or any combination — use `capture` instead of authoring a `workflow execute --step navigate --step screenshot` pipeline. `capture` is one Connect-RPC call against `CaptureService.Capture` and supports a viewport preset / explicit width-height, a wait-for condition, and a CSV of capture types in a single browser session.

```bash
# Desktop screenshot of a running scenario:
browser-automation-studio capture --url scenario=app-monitor,path=/ --capture screenshot --out /tmp/audit

# Mobile-viewport screenshot:
browser-automation-studio capture --url https://example.com --capture screenshot --dimensions mobile --out /tmp/mobile

# Full UI audit (screenshot + console + network) from one page load:
browser-automation-studio capture --url scenario=swarm-manager,path=/backlog --capture screenshot,console-logs,network --out /tmp/audit --json
```

Available `--capture` types: `screenshot`, `console-logs`, `network`, `video`, `dom`, `performance` (CSV, default: `screenshot`). Dimensions presets: `mobile` (390x844), `tablet` (768x1024), `desktop` (1440x900). `--width`/`--height` override the preset. `--wait-for` accepts a CSS selector, the string `networkidle`, or a numeric millisecond timeout. `--dry-run` validates a request without producing artifacts. `--json` emits the proto wire shape; default human output is a Mutation Contract report.

Equivalent prompt-manager actions wrap this command with fixed flags so agents discover them via `prompt-manager discover`:

| Action | Wraps |
|---|---|
| `bas.screenshot` | desktop screenshot |
| `bas.screenshot.mobile` | mobile screenshot |
| `bas.console-logs` | console capture only |
| `bas.network` | network capture only |
| `bas.audit` | screenshot + console + network in one session |
| `bas.status` | BAS health check |

Use `workflow execute` (below) when you need multi-step interaction — login, click, type, multi-page navigation. Capture is only for the single-location case.

#### Execute a Workflow

There are three ways to execute workflows: by name (stored workflow), from a JSON file, or inline with `--step` flags.

**By stored workflow name:**
```bash
# Execute a workflow stored in BAS by name
browser-automation-studio workflow execute my-login-workflow --wait

# List available workflows first
browser-automation-studio workflow list
```

> **Note:** Workflow names must be unique. If multiple workflows share the same name, execution will fail with "multiple workflows match name". Use `--from-file` instead, or rename workflows to be unique.

**From JSON file:**

> **Note:** The `--from-file` flag accepts both absolute and relative paths:
> - **Absolute paths** work from any directory
> - **Relative paths** are resolved against the current working directory first
> - If `--project-root` is provided, relative paths are also resolved against it

```bash
# Run a workflow from a file (use absolute path)
browser-automation-studio workflow execute \
  --from-file /abs/path/to/scenarios/{{TARGET}}/bas/cases/01-foundation/login.json \
  --output /tmp/bas/{{TARGET}}/cases/01-foundation/login \
  --wait

# Or use --project-root with relative path
browser-automation-studio workflow execute \
  --from-file scenarios/{{TARGET}}/bas/cases/01-foundation/login.json \
  --project-root /abs/path/to/scenarios/{{TARGET}}/bas \
  --output /tmp/bas/{{TARGET}}/cases/01-foundation/login \
  --wait

# Run with a starting URL (for workflows without navigate node)
browser-automation-studio workflow execute \
  --from-file scenarios/{{TARGET}}/bas/actions/open-project.json \
  --start-url http://localhost:3000/ \
  --output /tmp/bas/{{TARGET}}/actions/open-project \
  --wait

# Run with parameters (nested in initial_params)
browser-automation-studio workflow execute \
  --from-file scenarios/{{TARGET}}/bas/flows/checkout.json \
  --params '{"initial_params": {"username": "test@example.com"}}' \
  --output /tmp/bas/{{TARGET}}/flows/checkout \
  --wait
```

> **Note:** Custom parameters must be nested in `initial_params`. Example: `--params '{"initial_params": {"username": "test"}}'`

**Inline step execution (for quick tests without JSON files):**
```bash
# Simple smoke test
browser-automation-studio workflow execute \
  --step navigate "http://localhost:3000/dashboard" waitUntil=networkidle \
  --step screenshot fullPage=true \
  --output /tmp/bas/{{TARGET}}/navigate-dashboard \
  --wait

# Navigate to a scenario and assert an element exists
# Note: Use selector= prefix for attribute selectors containing '='
browser-automation-studio workflow execute \
  --step navigate scenario=knowledge-observatory path=/dashboard \
  --step assert selector="[data-testid='dashboard-container']" assertMode=exists \
  --step screenshot \
  --output /tmp/bas/{{TARGET}}/assert-dashboard-container \
  --wait

# Fill a form and submit
browser-automation-studio workflow execute \
  --step navigate "http://localhost:3000/login" \
  --step type "#email" text=test@example.com \
  --step type "#password" text=secret123 \
  --step click "#submit" \
  --step assert selector="[data-testid='dashboard']" assertMode=exists \
  --output /tmp/bas/{{TARGET}}/assert-login-form \
  --wait
```

> **Important:** When using CSS attribute selectors like `[data-testid='dashboard']`, prefix with `selector=` to avoid the `=` being parsed as a key-value delimiter.

> **Tip:** For navigate steps, you can use either a positional URL or `url=` prefix:
> ```bash
> # Both are equivalent:
> --step navigate https://example.com/path
> --step navigate url=https://example.com/path
> ```

**Step format:** `--step <type> [positional] [key=value ...]`

Use `browser-automation-studio schema steps` to see all available step types, their positional arguments, and required/optional key-value parameters. Use `browser-automation-studio schema steps --cli-only` for only CLI-supported steps.

**When to use each approach:**

| Approach | Use When |
|----------|----------|
| `--step` flags | Quick tests, smoke tests, debugging, simple linear flows |
| JSON file | Reusable workflows, complex branching, stored in bas/ |
| Stored name | Workflows already created via UI or API |

#### Workflow Management

```bash
# List all stored workflows
browser-automation-studio workflow list

# Validate workflow JSON syntax
browser-automation-studio workflow lint scenarios/{{TARGET}}/bas/cases/01-foundation/login.json
```

---

### **5. Schema Reference**

Use schema commands to discover step syntax and workflow structure.

```bash
# Get full workflow schema
browser-automation-studio schema workflow

# Get schema for specific node types
browser-automation-studio schema workflow --nodes navigate,click,assert

# List available node types
browser-automation-studio schema node-types

# Get inline step format reference (positional args, key-value pairs)
browser-automation-studio schema steps

# Get CLI-supported steps only
browser-automation-studio schema steps --cli-only

# Get schema in different formats
browser-automation-studio schema steps --format json
browser-automation-studio schema steps --format markdown
```

---

### **6. Example: Smoke Test Cycle**

This example shows the full cycle: run a smoke test and interpret results.

#### Quick Approach: Inline Steps

For quick smoke tests, use inline `--step` flags:

```bash
# Run smoke test with export
browser-automation-studio workflow execute \
  --step navigate scenario={{TARGET}} path=/dashboard waitUntil=networkidle \
  --step screenshot fullPage=true \
  --output /tmp/bas/{{TARGET}}/navigate-dashboard \
  --wait
```

Check results:
```bash
cat /tmp/bas/{{TARGET}}/navigate-dashboard/README.md
```

#### File-Based Approach (for reusable tests)

For workflows you want to save and reuse, create a JSON file in `scenarios/{{TARGET}}/bas/`. Use `browser-automation-studio schema workflow` to get the current workflow structure.

```bash
# Run a saved workflow
browser-automation-studio workflow execute \
  --from-file scenarios/{{TARGET}}/bas/cases/01-foundation/smoke.json \
  --output /tmp/bas/{{TARGET}}/cases/01-foundation/smoke \
  --wait

# Check results
cat /tmp/bas/{{TARGET}}/cases/01-foundation/smoke/README.md

# View screenshots (format: step-NN-<step-id>.png)
ls /tmp/bas/{{TARGET}}/cases/01-foundation/smoke/screenshots/
```

---

### **7. Node Types & Assert Modes**

For complete node schemas and field definitions, use:

```bash
browser-automation-studio schema workflow --nodes navigate,click,assert,wait
browser-automation-studio schema node-types
```

#### Assert Modes Quick Reference

| Mode | Validates |
|------|-----------|
| `exists` | Element is in DOM |
| `not_exists` | Element is NOT in DOM |
| `visible` | Element is visible |
| `hidden` | Element is hidden |
| `text_contains` | Element contains expected text (use `expectedText=`) |
| `text_equals` | Element text matches exactly (use `expectedText=`) |
| `attribute_contains` | Attribute contains value (use `attributeName=` and `expectedValue=`) |
| `attribute_equals` | Attribute matches exactly (use `attributeName=` and `expectedValue=`) |

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
   │    2. View screenshots/ for last screenshot           │
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

### **10. Session Management**

Session profiles store browser state (cookies, localStorage) for reuse across workflow executions. Use them for **manual testing and development workflows** where re-authenticating each time would be tedious.

> **For automated test suites**, use the subflow-based authentication pattern instead. See the **e2e-testing** skill for details.

#### Session Profile Workflow

```
Create profile → Sign in → Save session → Reuse
     │              │            │           │
     ▼              ▼            ▼           ▼
  session       workflow     --save-    --session-
  create       execute      session     profile
```

#### CLI Commands

**Create a session profile:**
```bash
# Create with a specific name
browser-automation-studio session create "Dev Account"

# Create without a name (auto-generates "Session N")
browser-automation-studio session create
```

**Sign in and save the session:**
```bash
# Execute sign-in workflow and save resulting auth state to profile
browser-automation-studio workflow execute \
  --from-file bas/actions/login.json \
  --save-session "Dev Account" \
  --initial-params '{"username":"dev@example.com","password":"..."}' \
  --wait
```

**Reuse session for subsequent testing:**
```bash
# Execute workflow with pre-authenticated browser context (skips sign-in)
browser-automation-studio workflow execute \
  --from-file bas/cases/02-features/admin-dashboard.json \
  --session-profile "Dev Account" \
  --wait
```

**Force fresh session (ignore saved state):**
```bash
browser-automation-studio workflow execute \
  --from-file bas/cases/01-foundation/01-auth/login-flow.json \
  --fresh-session \
  --wait
```

#### Managing Sessions

```bash
# List all session profiles
browser-automation-studio session list

# View profile details (shows browser profile, last used, storage stats)
browser-automation-studio session show "Dev Account"

# Rename a session profile
browser-automation-studio session rename "Dev Account" "Production Account"

# Clear storage state (force re-login on next use)
browser-automation-studio session clear-storage "Dev Account"

# Delete a profile (prompts for confirmation)
browser-automation-studio session delete "Dev Account"

# Delete without confirmation prompt
browser-automation-studio session delete "Dev Account" --force

# All commands support --json for programmatic output
browser-automation-studio session list --json
browser-automation-studio session create "New Profile" --json
browser-automation-studio session show "Dev Account" --json
browser-automation-studio session rename "Old Name" "New Name" --json
browser-automation-studio session delete "Dev Account" --json
browser-automation-studio session clear-storage "Dev Account" --json
```

> **Tip:** All session commands support short ID prefixes (minimum 4 characters). Use the first 8 characters shown in `session list` output:
> ```bash
> browser-automation-studio session show 5677c9e3  # Instead of full UUID
> ```

#### Refreshing Sessions (Load + Save)

Use `--session-profile` and `--save-session` together to load existing state, run a workflow that may update tokens or cookies, and save the refreshed state back:

```bash
# Load existing session, perform actions that refresh tokens, save updated state
browser-automation-studio workflow execute \
  --from-file bas/actions/refresh-auth.json \
  --session-profile "Dev Account" \
  --save-session "Dev Account" \
  --wait
```

This is useful for:
- **Token refresh flows**: Load session with expiring tokens, hit refresh endpoint, save new tokens
- **Session extension**: Perform activity to keep session alive
- **Incremental state building**: Add new cookies/localStorage to existing session

#### Understanding `session show` Output

The `session show` command displays comprehensive profile information:

```
Session Profile
===============
  ID:         9a6cf317-f2ab-4d31-8914-03bc3435a1bf
  Name:       Dev Account
  Created:    2026-01-30 00:35:22
  Updated:    2026-01-30 00:35:53
  Last Used:  2026-01-30 00:35:53

Browser Profile              # Only shown if configured
---------------
  Preset:     stealth        # none, stealth, or custom
  Mouse:      natural        # linear or natural movement style
  Scroll:     stepped        # smooth or stepped scrolling
  Typing:     50-150ms delay # Human-like typing delays
  Pauses:     enabled        # Micro-pauses between actions
  Stealth:    no-automation-flag, webdriver-patch, headless-bypass
  Ad Block:   ads_and_tracking

Storage State
-------------
  Cookies:      3 (1 expired)  # Warning shown for expired cookies
  Origins:      2
  LocalStorage: 5 items

  Cookies:
    session_id (example.com): abc123...
    auth_token (example.com): [HIDDEN]  # Sensitive values masked

  LocalStorage:
    https://example.com:
      user_preferences: {"theme":"dark"...}
```

**Key fields:**
- **Browser Profile**: Anti-detection and behavior settings (stealth mode, typing delays, etc.)
- **Storage State**: Saved cookies and localStorage that will be injected on next use
- **Expired cookies warning**: Alerts you when authentication may fail due to stale cookies

#### When to Use Session Profiles

| Use Case | Session Profile? |
|----------|------------------|
| Manual testing and debugging | ✅ Yes |
| Development workflows | ✅ Yes |
| Long-running admin operations | ✅ Yes |
| Automated test suites | ❌ No (use subflows) |
| CI/CD pipelines | ❌ No (use fresh sessions) |

**Why not for automated tests?**

| Session Profiles | Subflow-Based Auth |
|------------------|-------------------|
| State may be stale (expired cookies) | Always fresh login |
| Test depends on profile existence | Test is self-contained |
| Harder to debug failures | Full login flow in trace |
| Not CI-friendly | Works anywhere |

For automated testing patterns, see the **e2e-testing** skill section on "Authenticated Testing Patterns".

---

### **11. Output Expectations**

### **Execution safety labels**

Set `metadata.execution_mode` on every workflow. `observer` workflows may use
only navigate, screenshot, assert, extract, and wait nodes. A click, input, or
mutating subflow is rejected by workflow-health before execution. Relabel such
a case `mutating` and add `requires_confirmation: "true"` and
`routed_isolation: "true"`; the platform supplies the test-mode header only
after it proves the target's SQL and file isolation lease.

When using BAS for investigation, use `knowledge-observatory-tools` to read the current `problems` doc for `{{TARGET}}`, then document findings under the **E2E Issues** section with execution IDs, output paths, root causes, fixes, and status.

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
