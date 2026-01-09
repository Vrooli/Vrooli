# Select Node

`[REQ:BAS-NODE-SELECT-DROPDOWN]` manipulates native `<select>` elements, including multi-select dropdowns, by choosing options via value, label text, or zero-based index.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Selector** | CSS selector for the `<select>` element | Yes | Trimmed automatically.
| **Select by** | `value`, `text`, or `index` | Yes | Defaults to `value`. Multi-select mode disables `index`.
| **Option value** | Value attribute to match | For `value` mode | Accepts interpolation.
| **Displayed text** | Substring of option text | For `text` mode | Case-sensitive match inside the option label.
| **Option index** | Zero-based option index | For `index` mode | Must be ≥ 0.
| **Allow multiple** | Enables multi-select | Optional | When enabled, provide a newline-delimited list of values/text strings.
| **Timeout (ms)** | Wait for the select element | No | Defaults to 5 000.
| **Post-change wait (ms)** | Delay after selecting | No | Useful for forms that re-render on change.

## Runtime Behavior

1. The automation compiler forwards select params (mode, values/text/index, multi) unchanged; validation is handled by the workflow validator/UI.
2. Browserless resolves the selector, adjusts the underlying `<select>` value(s) using the requested mode, and dispatches both `input` and `change` events so frameworks react.
3. When multi-select is enabled, the runtime verifies the target element actually supports multiple selections and raises a descriptive error otherwise.
4. Telemetry stores the selection mode and final list of values for debugging.

## Examples

```json
{
  "type": "select",
  "data": {
    "selector": "select[name=country]",
    "selectBy": "value",
    "value": "US",
    "waitForMs": 250
  }
}
```

```json
{
  "type": "select",
  "data": {
    "selector": "#tags",
    "multiple": true,
    "values": ["primary", "secondary"],
    "timeoutMs": 8000
  }
}
```

## Tips & Limitations

- For React-controlled selects, prefer the value mode—React listens to `change` events with the updated `value`, which the runtime triggers automatically.
- When combining `text` mode with multi-select, ensure each entry is on its own line so the UI stores a clean array.
- Use a preceding Hover/Scroll node if the select is hidden inside an expanded menu.
- If a site replaces native selects with custom components, fall back to Click/Type nodes or emit custom scripts via the Script node.

## Related Nodes

- **Type** – Populate associated inputs once a select determines context.
- **Wait** – Ensure the UI reaction completes after an option change.
- **Hover** – Reveal dropdown menus before selecting hidden elements.
