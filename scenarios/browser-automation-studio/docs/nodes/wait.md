# Wait Node

`[REQ:BAS-NODE-WAIT-CONDITION]` pauses workflow execution until a time delay elapses, a selector appears, or navigation completes. Use it to stabilize asynchronous UI before continuing.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Wait type** | `time`, `element`, or `navigation` | No (defaults to `time`) | Stored as `waitType` in the instruction payload.
| **Duration (ms)** | Milliseconds to sleep | For `time` | Defaults to 1 000 when omitted.
| **Selector** | CSS selector to await | For `element` | Required; trimmed before dispatch. Waits until the element exists in the DOM.
| **Timeout (ms)** | Maximum wait for element/navigation | Optional | Applied to both `element` and `navigation` waits; clamped inside runtime.

_Note:_ The UI currently exposes the time/element options. Navigation waits are configured via the JSON editor using `{ "waitType": "navigation", "timeoutMs": 10000 }`.

## Runtime Behavior

1. `api/browserless/runtime/instructions.go:723-750` decodes the node, normalizes the wait type, and enforces required fields (e.g., selector for element mode).
2. For `time`, it sets `DurationMs` (default 1 000). For `element`, it sets `Selector` and `TimeoutMs`. For `navigation`, it sets `WaitType="navigation"` plus the optional timeout.
3. Browserless either sleeps locally (`time.Sleep`), polls the DOM for the selector, or waits for the next navigation event coming from Chrome DevTools Protocol, depending on the mode.
4. Step telemetry records the final wait type/duration for Execution Viewer.

## Examples

```json
{
  "type": "wait",
  "data": {
    "waitType": "element",
    "selector": "[data-testid=toast-success]",
    "timeoutMs": 15000
  }
}
```

```json
{
  "type": "wait",
  "data": {
    "waitType": "time",
    "duration": 3000
  }
}
```

## Tips & Limitations

- Prefer `element` waits over arbitrary delays so workflows stay resilient. Combine with Assert nodes to verify the element’s content after it appears.
- Navigation waits listen for the next top-level navigation; use them after clicks that trigger page loads but before interacting with the new page.
- For SPAs that update via network calls without navigation, pair element waits with `Wait` nodes that watch specific loading indicators.
- Time waits under 200 ms risk being clamped by the runtime defaults—use realistic buffers to avoid flake.

## Related Nodes

- **Assert** – Validate the condition that motivated the wait.
- **Scroll / Hover / Click** – Sequence after Wait once the UI is ready.
- **Conditional** – Branch if an awaited element never appears (using `continueOnFailure`).
