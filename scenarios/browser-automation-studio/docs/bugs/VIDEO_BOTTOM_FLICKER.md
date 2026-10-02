# Video Bottom Flicker Bug — Root Cause Analysis

**Status:** FIXED for the historical geometry repair described below; full-page screenshot/native-video interaction remains reproduced and unrepaired (see the October 2 follow-up).
**Date:** 2026-03-13
**Affected:** ALL Playwright `recordVideo` recordings in browser-automation-studio
**Current repair:** SDK compositor sizing and capture ordering (077 candidate). The original BAS override below is retained as historical investigation.
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


## 2026-09-23 amendment — shared SDK geometry owner

The original headful-only explanation is incomplete. Fresh Chromium 136 probes
reproduce the 87-pixel height loss in regular Chromium with both SDK headless
settings. The headless shell control passes. Setting only the compositor visible
size after window sizing fixes initial, landscape and portrait capture without
changing mobile or DPR emulation. Concurrent SDK screenshots and viewport changes
also reproduce stale DOM geometry and blank image regions. Joining the existing
Screenshotter queue fixes this second failure; a compositor command alone does
not establish capture ordering.

The canonical Rebrowser patch now applies these invariants at SDK Page and
Chromium viewport ownership. SDA installs the unchanged approved 1.52.0 version.
The BAS video-only asynchronous metrics override is removed: it duplicated screen
policy, overwrote mobile emulation, and left its CDP session attached. Existing
video layout stabilization remains separate behavior. Two native recordings at
640×480 and 900×640, DPR2, decode to 55 painted frames with zero gray bottom bands
without that override. Receipt: `/tmp/bas-video-geometry-077/receipt.json`.
Full driver and live-app qualification remain pending at this amendment; refer
to the goal home (`docs/internal/goal/`) for the final candidate's status and limitations.


## 2026-10-02 follow-up — full-page screenshots during native video

This is a separate capture-quality finding, not a reversal of the historical
SDK geometry receipts. Existing BAS shadow health/status verified the route
`http://localhost:15372/api/v1` before bounded adhoc tests using only a
self-contained neutral `data:text/html` ruler page. No product requests,
credentials, external assets, service changes or persistent settings were used.

- Mixed run `15ce1a8f-ff37-4da9-b258-1716a7a6c255`: fixed 1440×900, DPR 2,
  document height 2112; alternated viewport/full-page/viewport/full-page/viewport
  screenshots. Both full-page captures produced shrunken imagery and about 57%
  gray area in native WebM: nine frames at 4.64–4.96s and nine at 10.16–10.48s.
  DOM measurements between captures remained 1440×900. All steps completed.
- Matched viewport-only control `fbd825bc-d4b5-4459-9ede-da7238980ee8`:
  357 frames, zero gray-area events and no large image changes after startup.
  This validates viewport-only capture for this neutral geometry/runtime only.
- Initial taller neutral run `c1ad5a6c-484e-4bf4-8f05-9afa4549dec5` failed:
  raster 2850×12384 exceeded the 201326592-byte screenshot decode budget.
  The failed receipt and video remain evidence; no budget was raised.

The delivered buyer account WebM and H.264 derivative contain the same sizing
defects at the same frames; conversion did not introduce those defects.
Planner Focus's visible flash aligns with a scripted `/focus` reload and is a
separate navigation/loading observation. Do not treat all reported flicker as one
cause, dismiss it as acceptable, or infer product layout correctness from damaged
capture. Internal compositor/CDP operations were not instrumented; that part of
the mechanism remains source-supported inference.

[Native diagnosis and reusable isolation procedure](https://docs.google.com/document/d/1ovXTxVRHgKA4z3-zWPJsR9B56JXNYeacl6iV-cppJOg/edit)
and [separate scope and delivery-route review proposal — execution held pending owner approval](https://docs.google.com/document/d/1P8U8weXPXJTY788NnOYKj2Dx0nthSYHJazo0z3QJ07M/edit)
contain the evidence and approval boundary. Reusable flows, original neutral
videos, timelines, comparisons and receipts are in Library
`libfile_8f96b0d25f7081919ec2eb1dd1e16478` (`capture-quality-evidence.zip`),
SHA-256 `48dd87f0246f6b09df1ce0d328f69cfab86a2bf62c85535d9a12d3c5805fce2f`.
Materialize through the current supported Library route and verify readable
bytes/hashes; paths from another executor are not local evidence.

Product-level workaround validation and destination-player playback remain
pending. Separate-session full-page stills and product capture segmentation are
proposals, not tested remedies here. No repair is approved by this documentation
update. Preserve the existing shared SDK ownership and dependency governance;
do not restore the historical asynchronous BAS CDP override from this report.
Future capture entrypoint: [Screenshot Node capture-quality procedure](../nodes/screenshot.md#capture-quality-during-native-video).
