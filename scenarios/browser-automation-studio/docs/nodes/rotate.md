# Rotate Node

`[REQ:BAS-NODE-ROTATE-MOBILE]` reorients the Browserless viewport mid-run so workflows can validate responsive breakpoints, device rotations, and layout persistence for mobile apps.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Orientation** | `portrait` or `landscape`. | No | Defaults to `portrait`. |
| **Angle** | Valid angles depend on orientation: `0` or `180` for portrait, `90` or `270` for landscape. | No | Automatically clamped to the allowed set. |
| **Wait after rotate (ms)** | Optional pause after Chrome reports the new orientation. | No | Helpful when the UI reflows or triggers animations. |

## Runtime Behavior

1. The automation compiler retains the orientation/angle/wait fields in the contract instruction; validation for allowed combinations happens in the workflow validator.
2. BrowserlessEngine maps the instruction to `browserless/cdp/scroll_actions.go` where `Emulation.setDeviceMetricsOverride` applies the new dimensions and orientation, emitting viewport metadata for artifacts.
3. Subsequent Gesture/Scroll/Interaction nodes inherit the updated viewport so coordinates and animations reflect the rotated device.

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
