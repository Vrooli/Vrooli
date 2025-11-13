# Hover Node

`[REQ:BAS-NODE-HOVER-INTERACTION]` moves the pointer over an element without clicking so hover-only menus, tooltips, or focus states stay active for the next node.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Selector** | CSS selector to hover | Yes | Trimmed automatically before dispatching the instruction.
| **Timeout (ms)** | Wait for the element to appear | No | Default 5 000; minimum 100.
| **Post-hover wait (ms)** | Delay after the cursor settles | No | Gives CSS transitions or network calls time to run.
| **Movement steps** | Number of intermediate pointer positions | No | 1–50 (default 10); higher values produce smoother glides.
| **Duration (ms)** | Time spent moving between positions | No | 50–10 000 (default 350).

## Runtime Behavior

1. `api/browserless/runtime/instructions.go:112-132,1133-1173` validates the selector, clamps timing/step inputs, and writes them into `InstructionParam`.
2. Browserless issues `Input.dispatchMouseEvent` events to simulate moving from the current pointer position to the target element in the requested number of steps/duration.
3. After the hover completes, the runtime optionally waits `waitForMs` before handing off to the next instruction.
4. Telemetry records the selector, timing, and resulting bounding box so Execution Viewer can visualize the pointer path.

## Examples

```json
{
  "type": "hover",
  "data": {
    "selector": "nav .products",
    "steps": 8,
    "durationMs": 500,
    "waitForMs": 300
  }
}
```

## Tips & Limitations

- Pair with **Wait** or **Assert** nodes when hover menus load asynchronous content; the hover itself only guarantees the pointer is in place.
- Use a modest `steps` value (5–12) for natural motion; very high values can slow execution without adding realism.
- When hovering inside frames, switch into the iframe first so selectors resolve correctly.
- If the UI requires scrolling before the hover, add a Scroll node upstream to ensure the target element is on screen.

## Related Nodes

- **Click / Drag & Drop** – Sequence after Hover to interact with menus or drag handles that only appear on hover.
- **Focus** – Drive keyboard focus to inputs for accessibility checks.
