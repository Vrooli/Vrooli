## Steer focus: Browser Automation Studio

Reference for using Browser Automation Studio (BAS) to execute browser workflows, validate UI behavior, and debug automation failures.

BAS is a **browser automation tool** that lets you:
- Run smoke tests to verify pages load correctly
- Validate that UI elements exist and behave as expected
- Execute multi-step user journeys
- Capture screenshots and artifacts for debugging

This skill covers **tool usage**. For e2e testing strategy, workflow organization, selector registry setup, and requirements integration, see the **e2e-testing** skill.

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

#### Execute a Workflow

There are two ways to execute workflows: from a JSON file or inline with `--step` flags.

**From JSON file:**
```bash
# Run a workflow from a file
browser-automation-studio workflow execute \
  --from-file scenarios/{{TARGET}}/bas/cases/01-foundation/login.json \
  --output /tmp/bas/{{TARGET}}/cases/01-foundation/login \
  --wait

# Run with a starting URL (for workflows without navigate node)
browser-automation-studio workflow execute \
  --from-file scenarios/{{TARGET}}/bas/actions/open-project.json \
  --start-url http://localhost:3000/ \
  --output /tmp/bas/{{TARGET}}/actions/open-project \
  --wait

# Run with parameters
browser-automation-studio workflow execute \
  --from-file scenarios/{{TARGET}}/bas/flows/checkout.json \
  --params '{"username": "test@example.com"}' \
  --output /tmp/bas/{{TARGET}}/flows/checkout \
  --wait

# Run with video/trace recording
browser-automation-studio workflow execute \
  --from-file scenarios/{{TARGET}}/bas/cases/01-foundation/login.json \
  --record-video \
  --output /tmp/bas/{{TARGET}}/cases/01-foundation/login \
  --wait
```

**Inline step execution (for quick tests without JSON files):**
```bash
# Simple smoke test
browser-automation-studio workflow execute \
  --step navigate "http://localhost:3000/dashboard" waitUntil=networkidle \
  --step screenshot fullPage=true \
  --output /tmp/bas/{{TARGET}}/navigate-dashboard \
  --wait

# Navigate to a scenario and assert an element exists
browser-automation-studio workflow execute \
  --step navigate scenario=knowledge-observatory path=/dashboard \
  --step assert "[data-testid='dashboard-container']" assertMode=exists \
  --step screenshot \
  --output /tmp/bas/{{TARGET}}/assert-dashboard-container \
  --wait

# Fill a form and submit
browser-automation-studio workflow execute \
  --step navigate "http://localhost:3000/login" \
  --step type "#email" text=test@example.com \
  --step type "#password" text=secret123 \
  --step click "#submit" \
  --step assert "[data-testid='dashboard']" assertMode=exists \
  --output /tmp/bas/{{TARGET}}/assert-login-form \
  --wait
```

**Step format:** `--step <type> [positional] [key=value ...]`

For complete step syntax documentation, use:
```bash
# List all CLI-supported step types
browser-automation-studio schema steps --cli-only

# Get details for specific step types
browser-automation-studio schema steps --types navigate,click,assert

# Get JSON output for programmatic use
browser-automation-studio schema steps --format json
```

**Quick reference for common steps:**

| Node Type | Positional | Key Notes |
|-----------|------------|-----------|
| navigate | url | Or use scenario=, path= |
| click | selector | clickCount=, button= |
| type | selector | text= (required) |
| assert | selector | assertMode= (required) |
| wait | (selector) | durationMs= OR selector= |
| screenshot | (selector) | fullPage= |
| evaluate | expression | - |

**When to use each approach:**

| Approach | Use When |
|----------|----------|
| `--step` flags | Quick tests, smoke tests, debugging, simple linear flows |
| JSON file | Reusable workflows, complex branching, stored in bas/ |


#### Get Schema Reference

```bash
# Get full workflow schema
browser-automation-studio schema workflow

# Get schema for specific node types
browser-automation-studio schema workflow --nodes navigate,click,assert

# List available node types
browser-automation-studio schema node-types
```

---

### **5. Understanding Execution Results**

#### Artifact Location

The `--output` flag determines where artifacts are stored. Always specify an output path:

```
{output-path}/
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

Replace `{output-path}` with your `--output` value:

```bash
# Check if execution passed or failed
cat {output-path}/result.json | jq '.status'

# View error message on failure
cat {output-path}/result.json | jq '.error'

# Find which step failed
cat {output-path}/timeline.json | jq '.steps[] | select(.status == "failed")'

# View the last screenshot (often shows failure state)
ls -t {output-path}/frames/*.jpg | head -1

# Check for JavaScript errors
cat {output-path}/artifacts/console-*.json | jq '.[] | select(.level == "error")'

# Check for failed network requests
cat {output-path}/artifacts/network-*.json | jq '.[] | select(.status >= 400)'
```

---

### **6. Complete Example: Smoke Test Walkthrough**

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

Output:
```
Executing inline workflow with 2 steps
OK: Execution started!
Execution ID: exec-abc123
Waiting for completion...
OK: Execution completed successfully

Results exported to: /tmp/bas/{{TARGET}}/navigate-dashboard
```

Check results:
```bash
cat /tmp/bas/{{TARGET}}/navigate-dashboard/result.json | jq .
```

#### File-Based Approach (for reusable tests)

For workflows you want to save and reuse, create a JSON file in `scenarios/{{TARGET}}/bas/`. Use `browser-automation-studio schema workflow` to get the current workflow structure.

```bash
# Run a saved workflow
browser-automation-studio workflow execute \
  --from-file scenarios/{{TARGET}}/bas/cases/01-foundation/smoke.json \
  --output /tmp/bas/{{TARGET}}/cases/01-foundation/smoke \
  --wait

# Check pass/fail status
cat /tmp/bas/{{TARGET}}/cases/01-foundation/smoke/result.json | jq '.status'

# View screenshot
open /tmp/bas/{{TARGET}}/cases/01-foundation/smoke/frames/screenshot-001.jpg
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
- **Output path:** /tmp/bas/{{TARGET}}/...
- **Screenshot:** {output-path}/frames/screenshot-005.jpg
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
