# Drag & Drop Node

`[REQ:BAS-NODE-DRAG-DROP]` simulates pointer drags between two selectors, supporting both HTML5 drag/drop APIs and pointer-based fallbacks for apps that listen to mouse events.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Drag from selector** | CSS selector for the draggable element | Yes | Trimmed automatically.
| **Drop target selector** | CSS selector for the destination | Yes | Used for pointer movement and HTML5 drop events.
| **Hold before drag (ms)** | Delay between mousedown and movement | No | Default 150; ensures press-and-hold listeners fire.
| **Pointer steps** | Number of intermediate positions | No |
| **Drag duration (ms)** | Total time spent moving | No | 50–20 000 (default 600).
| **Offset X / Y** | Additional offset applied to the drag path | No | Useful when you need to grab handles rather than element centers.
| **Timeout (ms)** | Wait for selectors | No | Default 30 000; minimum 500.
| **Wait after (ms)** | Delay after dropping | No | Helps when the board re-renders.

## Runtime Behavior

1. `api/browserless/runtime/instructions.go:1368-1450` normalizes selectors, clamps steps/duration/offsets, and validates required fields.
2. Browserless first scrolls both elements into view, issues pointer events to simulate pressing, moving, and releasing, and synthesizes HTML5 `dragstart/dragenter/dragover/drop/dragleave` events so frameworks like React DnD respond.
3. When offsets are provided, the runtime adjusts the drag path accordingly and records the values in execution artifacts for replay.
4. Telemetry captures the movement profile (steps, duration, offsets) so Execution Viewer explains how cards moved.

## Example

```json
{
  "type": "dragDrop",
  "data": {
    "sourceSelector": ".kanban-column:nth-child(1) .card:nth-child(1)",
    "targetSelector": "#done-column .dropzone",
    "holdMs": 200,
    "steps": 20,
    "durationMs": 800,
    "waitForMs": 500
  }
}
```

## Tips

- Keep selectors precise; dragging from a list container may move the wrong item.
- Increase `steps`/`duration` for apps that rely on human-like movement to display drop previews.
- Some libraries require hovering the target for a moment—set `waitForMs` to give them time to respond.
- For canvas-based drag interactions, consider Script nodes to dispatch custom events.

## Related Nodes

- **Hover** – Reveal drag handles before starting.
- **Click** – Select items before dragging in multi-select UIs.
- **Wait/Assert** – Confirm the item moved to the desired column.
