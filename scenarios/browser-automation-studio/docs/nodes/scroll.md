# Scroll Node

`[REQ:BAS-NODE-SCROLL-NAVIGATION]` automates viewport movement so workflows can reveal lazy-loaded content, bring elements into view, or reposition the window to precise coordinates.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Scroll type** | `page`, `element`, `position`, or `untilVisible` | Yes | Determines which other fields apply. Defaults to `page`.
| **Direction** | `up`, `down`, `left`, `right` | For `page`/`untilVisible` | Combined with `amount` to control step size.
| **Amount (px)** | Pixels per move | For `page`/`untilVisible` | Clamped to 10–5 000. Default 400.
| **Selector** | CSS selector to scroll | For `element` | Uses `element.scrollIntoView` with optional behavior.
| **Target selector** | Selector to look for | For `untilVisible` | Workflow scrolls the page until this selector is visible or `maxScrolls` is reached.
| **Behavior** | `auto`/`smooth` | For `page` | Mirrors DOM `scrollTo` behavior.
| **X / Y** | Coordinates to scroll to | For `position` | Clamped to ±500 000.
| **Max scrolls** | Attempts for `page`/`untilVisible` | No | 1–200 (default 12) safety guard against infinite loops.
| **Timeout (ms)** | Element wait timeout | No | Defaults to 5 000 (100–120 000 clamp).
| **Post-wait (ms)** | Delay after scroll | No | Useful when sticky headers animate after motion.

## Runtime Behavior

1. The automation compiler/executor forwards scroll params (type/direction/amount/selectors/behavior/timeouts) unchanged; validation is handled by the workflow validator/UI.
2. The Browserless session dispatches either window-level `Page.scroll` commands or executes frame-aware `scrollIntoView`/`scrollTo` scripts, depending on the mode.
3. For `untilVisible`, the runtime loops up to `maxScrolls`, checking `IntersectionObserver` state each time; failure raises a descriptive error that includes the selector and attempts.
4. Telemetry captures the final coordinates/direction so replay tooling can show how far the viewport moved.

## Examples

### Scroll page down gradually
```json
{
  "type": "scroll",
  "data": {
    "scrollType": "page",
    "direction": "down",
    "amount": 600,
    "behavior": "smooth",
    "maxScrolls": 8,
    "waitForMs": 250
  }
}
```

### Scroll until a card becomes visible
```json
{
  "type": "scroll",
  "data": {
    "scrollType": "untilVisible",
    "targetSelector": "[data-testid=load-more]",
    "direction": "down",
    "amount": 500,
    "timeoutMs": 15000
  }
}
```

## Tips & Limitations

- When targeting nested scroll containers, use the **Element** mode and point `selector` at the container instead of the window.
- `untilVisible` stops as soon as `getBoundingClientRect()` indicates the target is within the viewport; pair with a `Wait` node if you also need network-idle guarantees.
- Large `amount` values speed up pagination but can overshoot sticky headers. Prefer smaller steps with higher `maxScrolls` for delicate pages.
- Combine with `Frame Switch` before scrolling inside iframes.

## Related Nodes

- **Wait** – Verify the content revealed by scrolling before proceeding.
- **Hover / Click** – Interact with newly visible UI elements.
- **Gesture** – Use for mobile swipe equivalents.
