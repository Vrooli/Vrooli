# Gesture Node

`[REQ:BAS-NODE-GESTURE-MOBILE]` simulates high-fidelity touch input (swipes, pinches, taps, and long-presses) so BAS workflows behave like real devices.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Gesture type** | `tap`, `doubleTap`, `longPress`, `swipe`, or `pinch`. | No | Defaults to `tap`. |
| **Element selector** | CSS selector used as the anchor point. | No | If empty, the viewport center (or explicit coordinates) is used. |
| **Direction** | `up`, `down`, `left`, or `right`. | For swipe when no explicit end coordinates | Determines the swipe path. |
| **Distance (px)** | Travel distance for swipe gestures. | For swipe (if no explicit end) | Clamped between 25 and 2,000 px. |
| **Duration (ms)** | How long the gesture lasts. | No | Defaults to 350 ms; min 50 ms. |
| **Steps** | Number of intermediate touch points for swipe/pinch. | No | Defaults to 15; clamped 2-60. |
| **Hold (ms)** | Press duration for long presses. | For `longPress` | Minimum 600 ms enforced. |
| **Scale** | Zoom factor for pinch gestures (e.g., `0.5` pinch in, `2` pinch out). | For `pinch` | Clamped between 0.2 and 4.0. |
| **Start/End coordinates** | Override auto-generated swipe coordinates. | No | Provide `startX`, `startY`, `endX`, `endY`; each is optional. |
| **Timeout (ms)** | How long to wait for the anchor element. | No | Defaults to the gesture timeout (15,000 ms). |

## Runtime Behavior

1. The automation compiler forwards gesture params (type, selector, direction, distance, duration, steps, hold, scale, coordinates) as-is; validation/clamping runs in the workflow validator.
2. `api/browserless/cdp/gestures.go:14-118` resolves the anchor point (selector center or viewport center/coordinates) and dispatches the requested gesture via `Input.dispatchTouchEvent`, automatically constructing swipe paths or pinch arcs.
3. Execution artifacts capture the resolved anchor, generated coordinates, and gesture metadata, making failed gestures easy to diagnose from telemetry alone.

## Example

```json
{
  "type": "gesture",
  "data": {
    "gestureType": "pinch",
    "selector": "#map",
    "scale": 0.5,
    "durationMs": 600,
    "steps": 25
  }
}
```

## Tips

- Provide a selector whenever possible so Browserless can keep gestures aligned with responsive layouts.
- Explicit start/end coordinates are useful for canvas apps; omit them for standard scroll views so BAS auto-computes safe swipe paths.
- Pair with `Rotate` and `Scroll` nodes to cover realistic mobile navigation flows.
- Use `Wait` after long gestures if the app triggers animations (e.g., map zoom) so downstream assertions stay deterministic.

## Related Nodes

- **Rotate** - Validate gesture behavior across orientations.
- **Scroll** - Use when a simple scroll suffices instead of a swipe gesture.
- **Wait** - Buffer UI transitions triggered by the gesture.
