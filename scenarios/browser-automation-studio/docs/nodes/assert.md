# Assert Node

`[REQ:BAS-NODE-ASSERT-CONDITION]` verifies DOM state, text content, attributes, or custom expressions so workflows fail fast when expectations are not met.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Label** | Friendly name shown in Execution Viewer | No | Stored as `label` in node data.
| **Mode** | `exists`, `not_exists`, `visible`, `text_equals`, `text_contains`, `attribute_equals`, `attribute_contains`, `expression` | Yes | Defaults to `exists`. Expression mode runs JavaScript instead of targeting selectors.
| **Selector** | CSS selector for the target element | Required unless `mode = expression` | Trimmed before execution.
| **Expected value** | String/JSON to compare | For text/attribute modes | Pulled from `expectedValue`, `expected`, or `text` fields.
| **Attribute name** | Attribute to inspect | For attribute modes | Stored as `attribute`.
| **Case sensitive** | Toggle for text comparisons | Optional | Sets `caseSensitive` flag.
| **Continue on failure** | Allows workflow to continue even if assertion fails | Optional | Sets `continueOnFailure`.
| **Negate** | Inverts the evaluation | Optional | `negate: true` flips the outcome.
| **Timeout (ms)** | Maximum wait before failing | Optional | Useful for dynamic content.
| **Custom expression** | JavaScript returning truthy/falsey | For `expression` mode | Runs inside the current frame context.

## Runtime Behavior

1. The automation compiler/executor forwards the assertion payload (mode, selector/value/attribute, negate/continue flags, timeout) as-authored; validation is handled by the workflow validator/UI.
2. Browserless evaluates the condition: selectors run inside the active frame, visibility checks rely on computed bounding boxes, and expression mode executes arbitrary JS with access to `document`.
3. The executor records `ConditionResult` metadata (type, outcome, selector) so Execution Viewer can highlight which branch fired.
4. When `continueOnFailure` is false, the workflow stops immediately and surfaces the provided failure message.

## Examples

```json
{
  "type": "assert",
  "data": {
    "mode": "text_contains",
    "selector": "#toast",
    "expectedValue": "Project created",
    "timeoutMs": 8000,
    "caseSensitive": false
  }
}
```

```json
{
  "type": "assert",
  "data": {
    "mode": "expression",
    "expression": "window.app?.isReady === true",
    "continueOnFailure": true
  }
}
```

## Tips & Limitations

- Combine with Wait nodes for deterministic behavior: Wait for an element, then assert its final text/attribute.
- Attribute assertions work best with stable attributes (e.g., `data-status`). Avoid volatile classes generated per build.
- Expression mode runs in the page context; keep logic small and handle exceptions to avoid masking real failures.
- Use `continueOnFailure` when the assertion influences branching instead of being a hard stop.

## Related Nodes

- **Conditional** – Route the workflow after an assertion passes or fails.
- **Wait** – Provide deterministic timing before asserting.
- **Use Variable** – Store assertion outcomes in variables for later reporting.
