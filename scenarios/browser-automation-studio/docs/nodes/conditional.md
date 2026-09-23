# Conditional Node

`[REQ:BAS-NODE-CONDITIONAL]` selects a branch using a page expression, element
presence or a workflow variable. Both true and false are successful evaluations.
An evaluation error remains a failed step and cannot select a normal truth branch.

## Typed configuration

Use `ACTION_TYPE_CONDITIONAL` and its `conditional` parameters.

| Field | Meaning |
| --- | --- |
| `conditionType` | Required: `CONDITIONAL_TYPE_EXPRESSION`, `ELEMENT` or `VARIABLE`, each with the `CONDITIONAL_TYPE_` prefix. |
| `expression` | Page JavaScript expression or function body with `return`. Promise results are awaited. Runs in the selected frame. |
| `selector` | CSS selector for element presence. Hidden elements count as present. |
| `variable` | Name in the actual workflow execution store, for example a previous Set Variable or stored Evaluate result. Missing variables are errors. |
| `operator` | Variable comparison: `CONDITIONAL_OPERATOR_EQUALS` (default), `NOT_EQUALS`, `CONTAINS`, `STARTS_WITH`, `ENDS_WITH`, `GT`, `GTE`, `LT` or `LTE`, each with the `CONDITIONAL_OPERATOR_` prefix. |
| `value` | Typed `JsonValue` for the variable comparison. Numeric operators accept finite numbers or numeric strings. Other comparisons use the shared workflow value representation. |
| `negate` | Invert a completed evaluation; never an evaluation error. |
| `timeoutMs` | Element-presence wait, default10000 ms, range0–120000. Zero observes immediately. |
| `pollIntervalMs` | Element polling interval, default250 ms, range1–5000. |

Expressions and variables observe once. Use Wait before checking a value that
changes asynchronously. Element checks poll until present or their timeout;
a genuine timeout means absence. Invalid selectors and closed pages remain errors.
Expressions use the ordinary execution deadline and cancellation policy. A script
that throws after making an effect is not automatically executed again.

## Branches and evidence

Connect edges with labels `true` and `false`. The builder's `IF TRUE` and
`IF FALSE` labels have the same meaning. Only the selected edge executes; an
unwired result ends the path. Evaluator errors can follow an explicitly labeled
`error` or `failure` edge when ordinary `executionSettings.continueOnError`
permits continuation. They never fall through to the first true/false edge.

Condition evidence retains its type, outcome, negation, selector/expression or
variable/operator, and observed/expected values. Browser predicates belong to
the driver; workflow-variable predicates belong to the executor's store.

```json
{
  "id": "check-banner",
  "action": {
    "type": "ACTION_TYPE_CONDITIONAL",
    "conditional": {
      "conditionType": "CONDITIONAL_TYPE_EXPRESSION",
      "expression": "return document.querySelector('#beta-banner') !== null;"
    }
  }
}
```

```json
{
  "id": "check-role",
  "action": {
    "type": "ACTION_TYPE_CONDITIONAL",
    "conditional": {
      "conditionType": "CONDITIONAL_TYPE_VARIABLE",
      "variable": "role",
      "operator": "CONDITIONAL_OPERATOR_EQUALS",
      "value": { "stringValue": "admin" }
    }
  }
}
```
