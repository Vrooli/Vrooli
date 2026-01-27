## Steer focus: Browser Automation Studio

Reference for using Browser Automation Studio (BAS) to execute workflows, analyze artifacts, and debug UI automation failures.

This skill covers **tool usage**. For e2e testing strategy and workflow organization, see the **e2e-testing** skill.

Required reading:
- `prompt-manager skills read e2e-testing`

---

### **1. Core Commands Reference**

#### Workflow Execution

```bash
# Execute workflow from file (most common)
browser-automation-studio workflow execute \
  --from-file scenarios/{{TARGET}}/bas/cases/01-foundation/login.json \
  --wait

# Execute subflow/action without navigate node (provide start URL)
browser-automation-studio workflow execute \
  --from-file scenarios/{{TARGET}}/bas/actions/open-project.json \
  --start-url http://localhost:8080/ \
  --wait

# Execute with parameters
browser-automation-studio workflow execute \
  --from-file scenarios/{{TARGET}}/bas/flows/checkout.json \
  --params '{"username": "test@example.com", "product_id": "123"}' \
  --wait

# Execute with seed data application
browser-automation-studio workflow execute \
  --from-file scenarios/{{TARGET}}/bas/cases/02-features/create-project.json \
  --seed needs-applying \
  --wait

# Execute and capture video/trace
browser-automation-studio workflow execute \
  --from-file scenarios/{{TARGET}}/bas/cases/01-foundation/login.json \
  --record-video \
  --record-trace \
  --wait
```

#### Execution Monitoring

```bash
# Watch execution in real-time (WebSocket telemetry)
browser-automation-studio execution watch <execution-id>

# List executions with status filter
browser-automation-studio execution list
browser-automation-studio execution list --filter running
browser-automation-studio execution list --filter failed
browser-automation-studio execution list --filter completed

# Stop a running execution
browser-automation-studio execution stop <execution-id>
```

#### Artifact Export

```bash
# Export execution data as JSON (replay schema)
browser-automation-studio execution export <execution-id> --output result.json

# Export to directory (preserves structure)
browser-automation-studio execution export <execution-id> --output-dir ./exports/my-run

# Generate HTML replay (step-through viewer)
browser-automation-studio execution render <execution-id> --output ./replay-dir

# Generate video of execution (MP4/WEBM)
browser-automation-studio execution render-video <execution-id>
```

---

### **2. Artifact Location Map**

All execution artifacts are stored in a consistent structure:

```
scenarios/browser-automation-studio/data/recordings/{executionId}/
├── result.json                # Final outcome summary
├── timeline.json              # Step-by-step execution log
├── frames/                    # Screenshots per step
│   ├── screenshot-001.jpg
│   ├── screenshot-002.jpg
│   └── ...
└── artifacts/
    ├── console-{stepId}.json  # Console log events
    ├── network-{stepId}.json  # Network request/response
    └── dom-{stepId}.json      # DOM snapshots
```

#### Quick Artifact Inspection

```bash
# View result summary
cat scenarios/browser-automation-studio/data/recordings/<id>/result.json | jq .

# Check execution status
cat scenarios/browser-automation-studio/data/recordings/<id>/result.json | jq '.status'

# List failed steps
cat scenarios/browser-automation-studio/data/recordings/<id>/timeline.json | jq '.steps[] | select(.status == "failed")'

# Check console errors
cat scenarios/browser-automation-studio/data/recordings/<id>/artifacts/console-*.json | jq '.[] | select(.level == "error")'

# Check for slow network requests (>2s)
cat scenarios/browser-automation-studio/data/recordings/<id>/artifacts/network-*.json | jq '.[] | select(.duration_ms > 2000)'

# View last screenshot before failure
ls -t scenarios/browser-automation-studio/data/recordings/<id>/frames/*.jpg | head -1
```

---

### **3. AI Navigation (Prompt-Based Testing)**

AI navigation uses vision models to drive the browser based on natural language goals. The vision agent captures screenshots, annotates elements, and decides actions iteratively.

#### When to Use AI Navigation

| Use Case | Example |
|----------|---------|
| Exploratory testing | "Navigate around the app and find any broken links or error states" |
| Complex multi-step flows | "Complete the checkout process for a standard order" |
| Smoke testing | "Log in and verify the dashboard loads with data" |
| Edge case discovery | "Try to break the form validation by entering unusual inputs" |
| Visual regression detection | "Compare the current layout to expected design" |

#### API Usage

```bash
# Start AI navigation session (requires active BAS session)
curl -X POST "http://localhost:${PLAYWRIGHT_DRIVER_PORT}/session/${SESSION_ID}/ai-navigate" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Log in with test@example.com / password123 and navigate to the dashboard",
    "model": "qwen3-vl-30b",
    "api_key": "${OPENROUTER_API_KEY}",
    "max_steps": 20,
    "callback_url": "http://localhost:${CALLBACK_PORT}/ai-step"
  }'

# Response includes navigation_id for tracking
# {"navigation_id": "nav-abc123", "status": "started"}

# Check navigation status
curl "http://localhost:${API_PORT}/api/v1/ai-navigate/${NAVIGATION_ID}/status"

# Abort in-progress navigation
curl -X POST "http://localhost:${API_PORT}/api/v1/ai-navigate/${NAVIGATION_ID}/abort"

# Resume paused navigation
curl -X POST "http://localhost:${API_PORT}/api/v1/ai-navigate/${NAVIGATION_ID}/resume"
```

#### Available Vision Models

| Model | Provider | Best For |
|-------|----------|----------|
| `qwen3-vl-30b` | OpenRouter | Cost-effective, general purpose |
| `gpt-4o` | OpenRouter | Complex reasoning, highest accuracy |
| `claude-sonnet-4` | Anthropic | Balanced quality/cost |
| Custom Ollama | Local | Privacy-sensitive, offline testing |

#### Callback Events

The vision agent POSTs step events to your callback URL:

```json
{
  "navigation_id": "nav-abc123",
  "step": 5,
  "action": {
    "type": "click",
    "target": {"x": 450, "y": 320},
    "element": "[data-testid='submit-button']"
  },
  "screenshot_url": "/artifacts/step-5.jpg",
  "dom_snapshot": "...",
  "reasoning": "Found submit button, clicking to proceed",
  "status": "completed"
}
```

---

### **4. Debugging Decision Tree**

```
                    Workflow failed?
                          │
          ┌───────────────┴───────────────┐
         YES                              NO
          │                                │
          ▼                                ▼
   Check result.json                 Success - document
   for error message                 in test report
          │
          ▼
   ┌──────────────────────────────────────────────────────┐
   │                    Error Type?                        │
   ├──────────────────────────────────────────────────────┤
   │                                                       │
   │  "SELECTOR_NOT_FOUND"                                 │
   │    1. Check selectors.ts registry exists              │
   │    2. Verify data-testid in component                 │
   │    3. View last screenshot - is element visible?      │
   │    4. Check if element is in iframe/shadow DOM        │
   │                                                       │
   ├──────────────────────────────────────────────────────┤
   │                                                       │
   │  "TIMEOUT"                                            │
   │    1. Check network-*.json for slow/failed APIs       │
   │    2. View frames/ for last screenshot before timeout │
   │    3. Check console-*.json for JS errors              │
   │    4. Increase timeoutMs in workflow node             │
   │    5. Add explicit wait node before action            │
   │                                                       │
   ├──────────────────────────────────────────────────────┤
   │                                                       │
   │  "ASSERTION_FAILED"                                   │
   │    1. Check expectedText vs actual in result.json     │
   │    2. View DOM snapshot at failing step               │
   │    3. Verify selector targets correct element         │
   │    4. Check if content is dynamic/async               │
   │                                                       │
   ├──────────────────────────────────────────────────────┤
   │                                                       │
   │  "NAVIGATION_ERROR"                                   │
   │    1. Check network-*.json for 4xx/5xx responses      │
   │    2. Verify scenario is running (vrooli scenario     │
   │       status {{TARGET}})                              │
   │    3. Check console-*.json for JS errors on load      │
   │    4. Verify URL/route exists in application          │
   │                                                       │
   └──────────────────────────────────────────────────────┘
```

---

### **5. Scenario Workflow Management**

#### Locating Existing Workflows

```bash
# List all workflow files in a scenario
find scenarios/{{TARGET}}/bas -name "*.json" -type f

# Check auto-generated registry manifest
cat scenarios/{{TARGET}}/bas/registry.json | jq '.playbooks[].file'

# Search for workflows by requirement ID
rg "REQ-AUTH-001" scenarios/{{TARGET}}/bas --type json

# Find workflows that test a specific selector
rg "@selector/dashboard" scenarios/{{TARGET}}/bas --type json
```

#### Workflow Hierarchy

```
bas/
├── registry.json       # Auto-generated manifest (DO NOT edit manually)
├── actions/            # Atomic reusable steps (NO assertions)
│   ├── login.json
│   └── open-project.json
├── flows/              # User journeys composing actions (NO assertions)
│   └── checkout-flow.json
└── cases/              # Test cases WITH assertions (mirrors PRD)
    ├── 01-foundation/
    │   ├── 01-auth/
    │   │   └── login-success.json
    │   └── 02-navigation/
    └── 02-features/
```

#### Debug Order (Critical)

When debugging failures, work **bottom-up through the hierarchy**:

1. **Actions first** - Verify atomic steps work in isolation
2. **Flows second** - Verify composed journeys complete
3. **Cases last** - Verify assertions against requirements

This approach isolates failures to the smallest reproducible unit.

#### Creating Temporary Workflows

For ad-hoc testing, create a minimal workflow:

```json
{
  "metadata": {
    "description": "Quick test - delete after debugging",
    "version": 1
  },
  "nodes": [
    {
      "id": "nav",
      "type": "navigate",
      "data": {
        "destinationType": "scenario",
        "scenario": "{{TARGET}}",
        "scenarioPath": "/dashboard"
      }
    },
    {
      "id": "screenshot",
      "type": "screenshot",
      "data": {
        "fullPage": true
      }
    }
  ],
  "edges": [
    { "source": "nav", "target": "screenshot" }
  ]
}
```

Save to `/tmp/quick-test.json` and run:

```bash
browser-automation-studio workflow execute --from-file /tmp/quick-test.json --wait
```

---

### **6. Maintain Scenario Constraints**

* This skill is for **using** BAS to investigate and validate, not for modifying scenario code
* Do **not** change workflow files unless specifically debugging them or asked to improve coverage
* Do **not** modify `ui/src/constants/selectors.ts` without understanding e2e implications
* Use artifacts to **diagnose** issues, then apply fixes appropriately in source code
* Never edit `bas/registry.json` manually - it's auto-generated by test-genie

---

### **7. Output Expectations**

When using BAS for investigation, document your findings:

**In `docs/internal/PROBLEMS.md` under E2E Issues section:**
* Execution IDs for reproducibility
* Links to specific artifacts (screenshots, logs)
* Root cause analysis
* Actionable next steps

**Example entry:**
```markdown
### Login flow fails on slow networks
- **Execution ID:** exec-abc123
- **Screenshot:** data/recordings/exec-abc123/frames/screenshot-005.jpg
- **Root cause:** API timeout before spinner appears
- **Fix:** Add waitForSelector before clicking submit
- **Status:** Pending fix in auth.tsx
```

You may:
* Execute workflows to validate UI behavior
* Analyze artifacts to diagnose failures
* Create temporary workflows for debugging
* Use AI navigation for exploratory testing

You **must**:
* Document findings with execution IDs
* Link artifacts for reproducibility
* Identify root causes, not just symptoms
* Provide actionable remediation steps
