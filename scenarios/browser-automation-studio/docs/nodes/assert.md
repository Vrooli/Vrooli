# Assert Node

`[REQ:BAS-NODE-ASSERT-CONDITION]` verifies DOM state, text or attributes in the
current page or selected frame. A logical mismatch fails the step and records
its expected and actual values. Browser errors remain errors even when negated.

## Typed configuration

Use `ActionDefinition.assert` (`AssertParams`) with `ACTION_TYPE_ASSERT`.

| Field | Meaning |
| --- | --- |
| `selector` | Required target selector. |
| `mode` | Required: `ASSERTION_MODE_EXISTS`, `NOT_EXISTS`, `VISIBLE`, `HIDDEN`, `TEXT_EQUALS`, `TEXT_CONTAINS`, `ATTRIBUTE_EQUALS` or `ATTRIBUTE_CONTAINS` (each with the `ASSERTION_MODE_` prefix). |
| `expected` | Typed `JsonValue` for a text or attribute comparison. |
| `attributeName` | Required for attribute comparisons. A missing attribute differs from an empty attribute. |
| `negated` | Expect the opposite predicate; defaults to false. |
| `caseSensitive` | Case sensitivity for text and attribute comparisons; defaults to true. Original values remain in evidence. |
| `failureMessage` | Message for a logical mismatch; does not replace a browser error. |
| `timeoutMs` | Wait for the requested presence/visibility state or comparison target. Defaults to the configured assertion timeout (normally 5000 ms). Zero observes immediately. |

Presence and visibility wait for the requested polarity: negated `EXISTS` waits
for absence, and negated `VISIBLE` waits for hidden or absent. Invalid selectors,
closed pages and other evaluation failures cannot satisfy a negative assertion.
Text and attribute comparisons read the value after the target resolves. Use a
Wait node before the assertion when the value itself changes asynchronously.

Assertions stop execution on failure unless the node's
`executionSettings.continueOnError` is true. For explicit branching, label edges
`success` and `failure`. The executor owns continuation and branch selection.

```json
{
  "id": "check-toast",
  "action": {
    "type": "ACTION_TYPE_ASSERT",
    "assert": {
      "selector": "#toast",
      "mode": "ASSERTION_MODE_TEXT_CONTAINS",
      "expected": { "stringValue": "Project created" },
      "caseSensitive": false,
      "failureMessage": "Project creation was not confirmed",
      "timeoutMs": 8000
    }
  }
}
```

The typed assertion contract has eight DOM modes; it has no JavaScript expression
or regular-expression mode. Use the appropriate workflow action for page script.
Assertion evidence includes the mode, selector, expected/actual values, negation,
case sensitivity, success and mismatch message.
