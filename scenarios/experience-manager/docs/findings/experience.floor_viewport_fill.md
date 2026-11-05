# experience.floor_viewport_fill

The inherited viewport-fill floor failed. The captured page surface does not fill the viewport, which can leave app chrome floating above the bottom edge on short pages.

Use a real height chain such as `min-height: 100dvh` on the app shell/root surface, or add a justified `floorOptOuts` entry when a short non-app surface is intentional.
