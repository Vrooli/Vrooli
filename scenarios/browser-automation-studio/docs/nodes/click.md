# Click Node

`[REQ:BAS-NODE-CLICK]` simulates mouse clicks against DOM elements, combining selector management, screenshot pickers, and AI-powered element suggestions so workflows stay resilient.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Selector** | CSS selector to target | Yes | Stored in `clickConfig.selector`. Use the screenshot picker, AI suggestions, or manual entry.
| **Button** | `left`, `middle`, `right` | No | Defaults to `left`; mapped to CDP `Input.dispatchMouseEvent`.
| **Click count** | `1` or `2` | No | Double-click when set to `2`.
| **Timeout (ms)** | Element wait timeout | No | Defaults to 30 000 ms; clamped in runtime validation.
| **Wait after (ms)** | Additional wait after the click | No | Gives UI animations time to finish.
| **Wait for selector** | Optional selector to await post-click | No | Adds implicit synchronization without separate Wait nodes.
| **Target URL override** | Optional static URL for the picker preview | No | Useful when upstream nodes don’t yet provide a screenshot URL.

### Selector & Preview Tooling
- **Screenshot overlay:** Captures the upstream node’s screenshot, lets you point-and-click to sample coordinates, and records offsets for future visualization.
- **DOM hierarchy panel:** Shows potential selectors (`div#submit`, `button.primary`, etc.) extracted from the screenshot metadata.
- **AI suggestions:** Describe the target (“submit new project”), and the node queries the element inspector endpoint to propose selectors and DOM paths.
- **Manual URL field:** Provide a staging URL for screenshots when the workflow hasn’t yet navigated upstream.

## Runtime Behavior

1. `runtime/instructions.go:793-820` validates the selector/button/count combo, applies timeout defaults, and copies wait settings into the `Instruction`.
2. The Browserless session waits for the selector, scrolls it into view, and dispatches the configured button events. When `waitForSelector` is set, it waits for that selector after the click to confirm downstream UI changes.
3. Telemetry captures the selector, click strategy, and DOM path inside execution artifacts, making debugging easier via Execution Viewer.

## Examples

### Primary button click with safety wait
```json
{
  "type": "click",
  "data": {
    "selector": "button[type=submit]",
    "waitForSelector": "[data-testid=success-toast]",
    "timeoutMs": 20000,
    "waitForMs": 500
  }
}
```

### Double-click a grid row
```json
{
  "type": "click",
  "data": {
    "selector": "[data-row-id=invoice-2031]",
    "button": "left",
    "clickCount": 2
  }
}
```

## Tips & Limitations

- Pair with the **Hover** node when menus require hover-before-click behavior.
- Keep selectors specific but resilient—prefer `data-testid` or IDs over brittle nth-child selectors; AI suggestions often include both so you can choose.
- For canvas/webgl apps where DOM selectors aren’t available, fall back to `Script` nodes or consider pointer automation via `Gesture`.
- The screenshot picker uses a cached upstream frame; refresh it after significant UI changes to avoid selecting stale coordinates.

## Related Nodes

- **Type**: Fill inputs before submitting forms via Click.
- **Wait**: Guard asynchronous transitions triggered by clicks.
- **Drag & Drop**: Use when simple clicks are insufficient for moving cards/files.
