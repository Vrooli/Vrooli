# Replay Layout Model

This note documents how the replay renderer computes sizing and positioning so the browser frame, cursor, watermark, and overlays stay aligned across settings preview, replay composer, and export capture.

## Coordinate spaces
- Canvas: The presentation output size (the styled replay background).
- Viewport: The captured screenshot size (browser content area).
- Display: The rendered size after fitting the canvas into available bounds.
- Frame: The browser chrome + viewport rect positioned inside the canvas.

## Layout rules
- The canvas is the authoritative output size. It is optionally scaled to fit a container (contain fit).
- The browser frame width is derived from the canvas width and `browserScale`.
- The viewport aspect ratio is preserved using the captured screenshot dimensions.
- Chrome header height is reserved inside the canvas so the viewport rect always fits below it.
- The viewport is centered within the remaining canvas area after reserving chrome header height.
- The frame rect is derived from the viewport rect plus chrome header height.
- Overlays (cursor, highlight/mask, focused element, watermark) are positioned in viewport space.

## Key files
- ui/src/domains/replay-layout/compute.ts
- ui/src/domains/exports/replay/ReplayPlayer.tsx
- ui/src/domains/replay-style/renderer/ReplayStyleFrame.tsx

## Notes for future changes
- If adding new overlays, pick a coordinate space first (viewport vs canvas) and use the layout model to convert.
- If presentation sizing changes, update layout tests in `ui/src/domains/replay-layout/__tests__/compute.test.ts`.
- When chrome header sizing changes, update the header height used by the layout input.
