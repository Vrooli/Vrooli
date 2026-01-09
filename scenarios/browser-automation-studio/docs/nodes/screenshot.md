# Screenshot Node

`[REQ:BAS-NODE-SCREENSHOT-CAPTURE]` captures viewport or element screenshots as execution artifacts, supporting highlights, masks, and full-page renders.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Name** | Identifier for the screenshot | No | Defaults to step index; useful for Execution Viewer labels.
| **Viewport width/height** | Override dimensions for capture | No | Leave blank to reuse the workflow viewport.
| **Full page** | Capture entire page height | No | When enabled, ignores viewport height.
| **Wait after (ms)** | Delay before capture | No | Lets animations finish.
| **Focus selector** | Element to focus first | No | Helpful for inputs or menus that require focus.
| **Highlight selectors** | Array of selectors to highlight | No | Styled via color/padding/border radius inputs.
| **Mask selectors** | Selectors to mask | No | Obscures sensitive data.
| **Background** | Background color for transparent pages | No |
| **Zoom factor** | Scales the capture | No | e.g., `1.5` for higher-resolution renders.
| **Capture DOM snapshot** | Include DOM HTML in artifacts | No | Useful for debugging.

## Runtime Behavior

1. The automation compiler keeps screenshot params intact (name, viewport overrides, focus/highlight/mask/zoom, DOM snapshot flag); validation is handled by the workflow validator and UI.
2. BrowserlessEngine applies the payload through `browserless/cdp/actions.go`, driving focus/highlights/masks before capturing via `Page.captureScreenshot`. Overlay metadata is preserved for replay.
3. Execution artifacts include the PNG plus highlight/mask metadata and optional DOM snapshot so the Execution Viewer and exports render the overlays correctly.

## Example

```json
{
  "type": "screenshot",
  "data": {
    "name": "checkout",
    "fullPage": true,
    "waitForMs": 500,
    "highlightSelectors": ["#total", ".cta"],
    "maskSelectors": ["[data-sensitive=true]"]
  }
}
```

## Tips

- When capturing modals, use Focus or Wait nodes beforehand to ensure the modal is fully visible.
- Mask selectors are great for hiding PII before shareable exports.
- Capture DOM snapshots sparingly—they increase artifact size but help diagnose flaky UI.
- Consider a Rotate node before screenshots to validate responsive layouts.

## Related Nodes

- **Assert** – Validate UI before capturing proof.
- **Wait** – Ensure asynchronous content is rendered first.
- **Rotate** – Capture portrait vs. landscape states.
