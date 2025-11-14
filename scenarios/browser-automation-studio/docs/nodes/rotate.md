# Rotate Node

`[REQ:BAS-NODE-ROTATE-MOBILE]` reorients the Browserless viewport mid-run so workflows can validate responsive breakpoints, device rotations, and layout persistence for mobile apps.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Orientation** | `portrait` or `landscape`. | No | Defaults to `portrait`. |
| **Angle** | Valid angles depend on orientation: `0` or `180` for portrait, `90` or `270` for landscape. | No | Automatically clamped to the allowed set. |
| **Wait after rotate (ms)** | Optional pause after Chrome reports the new orientation. | No | Helpful when the UI reflows or triggers animations. |

## Runtime Behavior

1. `api/browserless/runtime/instructions.go:1257-1289` normalizes the orientation/angle combination, preventing invalid values (e.g., 90 degrees in portrait mode) and adds an optional post-rotate wait.
2. `api/browserless/cdp/scroll_actions.go:91-141` calls `Emulation.setDeviceMetricsOverride` with the rotated width/height, updates the session's cached viewport, and records the new dimensions plus `orientationType` in execution artifacts.
3. Any subsequent Gesture/Scroll/Interaction nodes inherit the updated viewport size so coordinates continue to line up with mobile expectations.

## Example

```json
{
  "type": "rotate",
  "data": {
    "orientation": "landscape",
    "angle": 270,
    "waitForMs": 500
  }
}
```

## Tips

- Insert a Rotate node before capturing screenshots to exercise both portrait and landscape marketing shots.
- Combine with `Gesture` nodes to mimic users rotating a device and immediately swiping.
- Use the wait field when your CSS triggers a transition or map re-render after rotation; it keeps downstream assertions deterministic.

## Related Nodes

- **Gesture** - Touch interactions behave differently across orientations; verify them back-to-back.
- **Scroll** - Ensure scroll locking behaves after rotation changes.
- **Screenshot** - Capture both orientations for regression artifacts.
