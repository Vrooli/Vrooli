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
2. The Playwright driver applies the payload, driving focus/highlights/masks before capturing via `Page.captureScreenshot`. Overlay metadata is preserved for replay.
3. Execution artifacts include the PNG plus highlight/mask metadata and optional DOM snapshot so the Execution Viewer and exports render the overlays correctly.

The driver reports dimensions decoded from the saved PNG/JPEG bytes. When a
full-page PNG exceeds the image budget, it retries JPEG at the same full-page
extent; if that still exceeds the budget, capture fails with the raster size.
It never labels a viewport crop as a successful full-page result. During native
video recording, full-page still capture is rejected with guidance to use a
viewport screenshot or a separate capture session. PNG and JPEG handlers use
the same configured `fullPage` default unless the action explicitly overrides
it.

Successful step outcomes retain `screenshot_requested_extent`,
`screenshot_actual_extent`, `screenshot_media_type`, and
`screenshot_degraded` in their existing notes map. Raster width and height
describe the saved image bytes. A rejected capture records
`actual_extent=not_captured` in its failure receipt; it does not produce a
success-shaped image record.

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

## Capture quality during native video

Full-page screenshots can corrupt concurrent native `recordVideo` imagery even
when the encoded canvas and DOM viewport measurements stay fixed. The driver
now rejects this combination. Use explicit viewport-only screenshots
(`full_page: false` in V2 action payloads) or capture the full page in a separate
session; do not interpret a shrunken/gray recorded frame as product
responsive-layout evidence. Keep requested and actual viewport, DPR, screenshot
extent, execution identity, timestamps and original hashes together.

The [verified neutral comparison and approval boundary](../bugs/VIDEO_BOTTOM_FLICKER.md#2026-10-02-follow-up--full-page-screenshots-during-native-video)
records two full-page capture bursts with 18 damaged frames, versus zero in the
matched viewport-only control. Product-level workaround validation and
consuming-player playback remain pending. Obtaining full-page stills in a
separate session is proposed, not validated by this comparison; never silently
substitute a cropped artifact for a requested full-page image.

Before accepting a recording, inspect decoded frames around screenshot and
navigation events as well as the native timestamps. Unexplained gray padding,
blank frames or geometry excursions fail capture-quality acceptance. Label
intentional navigation/loading and retain complete originals and failed attempts;
any edited presentation clip needs a separate derivative identity and edit map.
Local hash verification, successful decode, frame quality and destination playback
are separate gates. Recheck the permitted target/fixture route before product
capture; a BAS shadow instance does not prove target-app isolation.

## Related Nodes

- **Assert** – Validate UI before capturing proof.
- **Wait** – Ensure asynchronous content is rendered first.
- **Rotate** – Capture portrait vs. landscape states.
