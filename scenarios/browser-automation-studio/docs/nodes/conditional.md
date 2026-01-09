# Conditional Node

Implements `[REQ:BAS-NODE-CONDITIONAL]` by branching the workflow before the next node executes. Conditions can be evaluated with JavaScript, DOM selectors, or variable comparisons, and branches are wired with the `if_true` / `if_false` handles surfaced in the Workflow Builder.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Condition type** | `expression`, `selector`, or `variable` | No (defaults to `expression`) | Maps to `conditionType` in the compiled instruction. |
| **Expression** | JavaScript that returns truthy/falsey | For `expression` mode | Runs inside the active frame; you can reference globals such as `document`. |
| **Selector** | CSS selector to probe for presence | For `selector` mode | The runtime polls until the element exists or the timeout elapses. |
| **Variable** | Workflow variable name | For `variable` mode | Compared using the configured operator/value. |
| **Operator** | `equals`, `not_equals`, `contains`, `starts_with`, `ends_with`, `greater_than`, `less_than` | Optional | Defaults to `equals` and is only used for variable comparisons. |
| **Comparison value** | Value to compare variables against | Optional | Can be string, number, boolean, or JSON. |
| **Negate result** | Invert the evaluated outcome | Optional | Useful for “if not present” flows without adding a second branch. |
| **Timeout (ms)** | Max wait for the condition | Optional | Clamped between 500 ms and 120 000 ms. Defaults to 10 000 ms. |
| **Poll interval (ms)** | How often selector/variable checks run | Optional | Clamped between 100 ms and 5 000 ms. |

## Behavior

1. The compiler records the node’s branch handles so only the `if_true` or `if_false` edge is followed in `api/automation/executor/plan_builder.go`.
2. The automation compiler/executor preserves the condition payload (mode, poll/timeout, operator defaults, negation flag) in the contract instruction; validation and clamping live in the workflow validator.
3. `api/browserless/cdp/conditional_actions.go` evaluates the condition inside the active frame/session and captures a `ConditionResult` so telemetry and artifacts show which branch fired.
4. The automation executor updates the step metadata with the result, making it visible in Execution Viewer and JSON artifacts for later debugging.

## Examples

### Branch on JavaScript expression

```json
{
  "type": "conditional",
  "data": {
    "conditionType": "expression",
    "expression": "document.querySelector('[data-role=beta-banner]') !== null",
    "timeoutMs": 8000
  }
}
```

Use the `if_true` handle when the beta banner is visible and `if_false` to skip feature-gate steps.

### Compare workflow variables

```json
{
  "type": "conditional",
  "data": {
    "conditionType": "variable",
    "variable": "auth.role",
    "operator": "equals",
    "value": "admin",
    "negate": false,
    "timeoutMs": 2000,
    "pollIntervalMs": 250
  }
}
```

Routes to the true branch for administrators while the false branch handles standard users.

## Limitations & Tips

- Selector conditions only check for existence, not visibility. Pair with `Wait` if you need computed style checks.
- Expression mode runs in the page context; wrap risky logic in `try/catch` to avoid unhandled errors.
- Variable values are read from the execution context snapshot; update variables via Set/Use Variable nodes before branching to avoid stale data.
- Always connect both `if_true` and `if_false` handles—even if one path just leads to an `End` node—to avoid dangling executions.

## Related Nodes

- **Loop** – Use inside loop bodies for per-item branching.
- **Wait** – Gate the conditional with explicit waits when polling dynamic content.
- **Set/Use Variable** – Prepare the variables you later branch on.
