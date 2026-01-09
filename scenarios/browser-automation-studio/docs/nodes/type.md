# Type Node

`[REQ:BAS-NODE-TYPE-INPUT]` handles keyboard input for text fields, password boxes, and content-editable regions. It can inherit the upstream page URL, or you can override the target URL when capturing selectors during authoring.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Page URL override** | Optional absolute URL for the element picker | No | Defaults to the upstream Navigate node; stored as `data.url` if provided.
| **Selector** | CSS selector for the input/element | Yes | Populated via the element picker or manual entry.
| **Text** | String to type (supports newlines) | Yes | Stored as-is; interpolation works via Use Variable nodes.
| **Delay (ms)** | Per-character delay | No | Optional `delayMs` property; when omitted the runtime uses Browserless defaults.
| **Clear field first** | Clears existing value before typing | No | Set `clear: true` via JSON editor / future UI toggle.
| **Submit (Enter)** | Press Enter after typing | No | Set `submit: true` to trigger form submissions automatically.
| **Timeout (ms)** | Wait for selector | No | Defaults to 30 000 in the runtime.

_The current UI exposes the primary URL/selector/text fields; advanced options (delay, clear, submit) remain available via JSON editing or future node settings._

## Runtime Behavior

1. The compiler records the node as `StepTypeInput` and passes the raw data into the runtime.
2. The automation executor forwards selector/text/timeout/delay/clear/submit params to the engine; validation is handled by the workflow validator.
3. Browserless waits for the selector, optionally clears the field, types the text using the requested cadence, and presses Enter when `submit` is set.
4. Execution artifacts capture the typed value (masked for password fields) along with timing metadata for replay/debugging.

## Examples

```json
{
  "type": "type",
  "data": {
    "selector": "input[name=email]",
    "text": "{{creds.email}}",
    "delayMs": 20,
    "clear": true
  }
}
```

## Tips & Limitations

- Use `Set Variable` + interpolation to avoid hard-coding secrets inside the node.
- When interacting with content-editable editors, ensure the selector references the editable container, not the surrounding div.
- Combine with **Focus** to ensure inputs gain focus before typing, especially on forms with custom focus management.
- After typing sensitive values, consider a Screenshot node configured to mask the field to prevent secrets from appearing in artifacts.

## Related Nodes

- **Click** – Focus the input before typing.
- **Shortcut / Keyboard** – Send Enter, Tab, or custom keystrokes after typing.
- **Wait** – Let validation tooltips settle before assertions.
