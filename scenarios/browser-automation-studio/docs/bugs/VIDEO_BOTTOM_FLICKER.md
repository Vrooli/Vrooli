# Video Bottom Flicker Bug — Root Cause Analysis

**Status:** FIXED
**Date:** 2026-03-13
**Affected:** ALL Playwright `recordVideo` recordings in browser-automation-studio
**Fix:** CDP `Emulation.setDeviceMetricsOverride` in `context-builder.ts`
**Related:** [Playwright #36032](https://github.com/microsoft/playwright/issues/36032) (fixed in Playwright v1.55.0, BAS uses rebrowser-playwright 1.52.0)

## Symptom

Every video recorded by Playwright's `recordVideo` feature has a uniform gray bar (RGB ~128,128,128) in the bottom portion that intermittently appears and disappears, creating visible flickering during playback.

## Root Cause

**The `headless: false` + `--headless=new` browser launch configuration causes Playwright to miscalculate window dimensions for video recording.**

When Playwright launches with `headless: false`:
1. It calculates window size as `viewport + browser_chrome_height` (assuming a headed browser with toolbars, address bar, etc.)
2. But `--headless=new` means there's no actual browser chrome
3. Chrome's video encoder captures the full window area at the requested `recordVideo.size`
4. Only viewport-height pixels have rendered page content
5. The gap fills with gray — the VP8 framebuffer default (YUV neutral = RGB 128,128,128)

This is a known Playwright bug ([#36032](https://github.com/microsoft/playwright/issues/36032)) fixed in v1.55.0 via Chromium update. Since BAS uses `rebrowser-playwright@1.52.0` (which only goes up to 1.52.0), we applied a targeted CDP workaround.

## Experimental Validation

### Reproduction (automated script)
```
Config: headless:false + --headless=new + DPI=2 + viewport 1440x900
Result: 73/73 frames have gray bar (100%)
```

### Hypothesis testing results

| Hypothesis | Test | Result |
|-----------|------|--------|
| Chromium 136 bug, fixed in 140 | Used chromium-1187 (v140) | **REJECTED** — gray persists |
| `headless: false` window sizing | Used `headless: true` | **CONFIRMED** — 0/73 gray |
| `--disable-infobars` fixes it | Added flag | **REJECTED** — gray persists |
| `--force-device-scale-factor=1` | Added flag | **REJECTED** — gray persists |
| `--window-size` matching viewport | `--window-size=1440,900` | **CONFIRMED** — 0/48 gray |
| Large window size (3840x2160) | `--window-size=3840,2160` | **REJECTED** — must match exactly |
| CDP window resize per-page | `Browser.setWindowBounds` | **REJECTED** — gray persists |
| CDP `setDeviceMetricsOverride` | Per-page with screenW/H | **CONFIRMED** — 0/48 gray |

### Key experiment: CDP Emulation.setDeviceMetricsOverride
```typescript
session.send('Emulation.setDeviceMetricsOverride', {
  width: viewportWidth,
  height: viewportHeight,
  deviceScaleFactor,
  mobile: false,
  screenWidth: viewportWidth,   // ← This is the critical parameter
  screenHeight: viewportHeight, // ← Tells compositor the true screen size
});
```
This overrides Chrome's compositor screen dimensions, which the video encoder uses for frame sizing. When `screenWidth`/`screenHeight` match the viewport, the encoder captures exactly the rendered area with no gray fill.

### Multi-viewport validation
```
1440x900@2x: 60 frames — ✓ CLEAN
1280x720@2x: 60 frames — ✓ CLEAN
```

## Fix Implementation

**File:** `playwright-driver/src/session/context-builder.ts`

When video recording is enabled, register a `context.on('page')` listener that applies `Emulation.setDeviceMetricsOverride` via CDP to each new page. The listener fires before any navigation, ensuring the metrics are set before the first video frame is captured.

The fix is non-fatal: if the CDP call fails, a warning is logged but the session continues normally (video may have the gray bar but all other functionality is unaffected).

## Detection Methodology

### Quantified findings (primary video: execution 02aafadc)

| Metric | Value |
|--------|-------|
| Total frames | 64 |
| Frames with gray bottom | 20 / 64 (31%) |
| Gray region height | Always exactly 87px |
| Gray region starts at | Always y=813 (in 900px frame) |
| Boundary sharpness | Avg pixel jump of ~110 at y=812→813 |
| Gray pixel values | (128,128,128) ± 2 — VP8 framebuffer default |
| Content shift | NONE — content above y=812 identical in both states |

### Cross-video validation (31 videos analyzed)
- **1280x720:** 0/14 had gray in existing recordings (likely recorded with different config)
- **1440x900:** 12/17 had gray (intermittent per-frame)
- Controlled reproduction: 100% of frames had gray at ALL viewport sizes with `headless:false` + `--headless=new`

## Previous fix attempts (all ineffective)

1. **CSS viewport stabilization** — Prevents scrollbar reflow but doesn't affect video encoder dimensions
2. **FFmpeg filter chain improvements** — Only affects export/render path, not Playwright's `recordVideo`
3. **Pixel-level test coverage** — Tests the FFmpeg assembly path, not the Playwright recording path
